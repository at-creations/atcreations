import createClient, { type Client, type Middleware } from 'openapi-fetch';
import type { paths } from '@atcreations/api-types';
import { env } from '$env/dynamic/public';
import { ApiError, parseProblem } from './api-error';

export { ApiError, parseProblem, fieldErrorMap } from './api-error';

type TokenGetter = () => Promise<string | null | undefined>;

let getToken: TokenGetter = async () => null;
let onUnauthorized: () => void = () => {};

/**
 * Wire the API client to the Clerk session. Called once by the authenticated layout.
 * The token is fetched per request (Clerk caches & refreshes it) and is never persisted.
 */
export function configureApi(options: { getToken: TokenGetter; onUnauthorized: () => void }) {
	getToken = options.getToken;
	onUnauthorized = options.onUnauthorized;
}

/** A fresh session token for requests made outside openapi-fetch (file uploads). */
export function sessionToken(): Promise<string | null | undefined> {
	return getToken();
}

/** Report a 401 seen outside openapi-fetch (ends the session like any other 401). */
export function reportUnauthorized(): void {
	onUnauthorized();
}

export function apiBaseUrl(): string {
	return (env.PUBLIC_API_URL || 'http://localhost:8080').replace(/\/+$/, '');
}

const authMiddleware: Middleware = {
	async onRequest({ request }) {
		const token = await getToken();
		if (token) request.headers.set('Authorization', `Bearer ${token}`);
		return request;
	},
	async onResponse({ response }) {
		if (response.status === 401) onUnauthorized();
		return response;
	}
};

let client: Client<paths> | undefined;

/** Typed openapi-fetch client. Created lazily so the runtime PUBLIC_API_URL is used. */
export function api(): Client<paths> {
	if (!client) {
		client = createClient<paths>({ baseUrl: apiBaseUrl(), credentials: 'omit' });
		client.use(authMiddleware);
	}
	return client;
}

type FetchResult<T> = { data?: T; error?: unknown; response: Response };

/** Resolve an openapi-fetch call to its data, throwing a typed ApiError on failure. */
export async function unwrap<T>(call: Promise<FetchResult<T>>): Promise<T> {
	let result: FetchResult<T>;
	try {
		result = await call;
	} catch (cause) {
		throw new ApiError({ status: 0, title: 'Network error', detail: String(cause), errors: [] });
	}
	if (!result.response.ok || result.error !== undefined) {
		throw parseProblem(result.response.status, result.error);
	}
	return result.data as T;
}
