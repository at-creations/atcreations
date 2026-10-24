import createClient from 'openapi-fetch';
import { error } from '@sveltejs/kit';
import type { Category, PageMeta, paths, PublicProduct, StoreInfo } from '@atcreations/api-types';
import { config } from './env';
import { requestContext } from './request-context';

const TIMEOUT_MS = 5_000;

type ProductQuery = NonNullable<paths['/v1/products']['get']['parameters']['query']>;

let client: ReturnType<typeof createClient<paths>> | undefined;

function api() {
	client ??= createClient<paths>({
		baseUrl: config().apiInternalUrl,
		// Every SSR request to the API is bounded by a 5s timeout.
		fetch: (request: Request) => {
			// Forward the visitor's address so the API rate-limits per visitor rather than
			// treating every SSR request as coming from this container. The API only trusts
			// this header from addresses listed in its TRUSTED_PROXIES.
			const clientAddress = requestContext()?.clientAddress;
			if (clientAddress) request.headers.set('x-forwarded-for', clientAddress);
			return fetch(request, { signal: AbortSignal.timeout(TIMEOUT_MS) });
		},
		headers: { accept: 'application/json' }
	});
	return client;
}

interface ApiResult<T> {
	data?: T;
	error?: unknown;
	response: Response;
}

/**
 * Unwrap an openapi-fetch result: API 404 → SvelteKit 404, any other failure
 * (network error, timeout, 4xx/5xx) → 503 with a generic message. Upstream details are
 * logged server-side only and never sent to the client.
 */
async function unwrap<T>(label: string, request: () => Promise<ApiResult<T>>): Promise<T> {
	let result: ApiResult<T>;
	try {
		result = await request();
	} catch (err) {
		const reason = err instanceof Error ? err.name : 'unknown';
		console.error(`[api] ${label} failed: ${reason}`);
		error(503, 'Service unavailable');
	}
	if (result.response.status === 404) error(404, 'Not found');
	if (!result.response.ok || result.data === undefined) {
		console.error(`[api] ${label} returned HTTP ${result.response.status}`);
		error(503, 'Service unavailable');
	}
	return result.data;
}

export async function listProducts(
	query: ProductQuery
): Promise<{ data: PublicProduct[]; meta: PageMeta }> {
	return unwrap('GET /v1/products', () => api().GET('/v1/products', { params: { query } }));
}

export async function getProduct(slug: string): Promise<PublicProduct> {
	const body = await unwrap(`GET /v1/products/{slug}`, () =>
		api().GET('/v1/products/{slug}', { params: { path: { slug } } })
	);
	return body.data;
}

/** Categories change rarely and are the same for everyone: kept in memory for a minute. */
const CATEGORIES_TTL_MS = 60_000;
let categoriesCache: { at: number; value: Promise<Category[]> } | null = null;

export function listCategories(): Promise<Category[]> {
	const now = Date.now();
	if (!categoriesCache || now - categoriesCache.at > CATEGORIES_TTL_MS) {
		const value = unwrap('GET /v1/categories', () => api().GET('/v1/categories')).then(
			(body) => body.data
		);
		categoriesCache = { at: now, value };
		// A failure is not cached: the next request tries again.
		value.catch(() => {
			if (categoriesCache?.value === value) categoriesCache = null;
		});
	}
	return categoriesCache.value;
}

export async function getStoreInfo(): Promise<StoreInfo> {
	const body = await unwrap('GET /v1/store-info', () => api().GET('/v1/store-info'));
	return body.data;
}

/** Like `fn()`, but resolves to `fallback` instead of throwing (for non-critical data). */
export async function orFallback<T>(fn: () => Promise<T>, fallback: T): Promise<T> {
	try {
		return await fn();
	} catch {
		return fallback;
	}
}

// Private: every HTML response carries its own CSP nonce, so a shared cache (CDN) must never
// serve one visitor's page to another. The API's cache keeps rendering cheap instead.
export const CATALOGUE_CACHE_CONTROL = 'private, max-age=60, stale-while-revalidate=600';
