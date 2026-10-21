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

/**
 * Multipart upload with progress events (fetch has no upload progress). Uses the same bearer
 * token, never cookies. Resolves with the parsed `data` member of the response envelope.
 */
export async function uploadWithProgress<T>(
	method: 'POST' | 'PUT',
	path: string,
	form: FormData,
	onProgress?: (fraction: number) => void
): Promise<T> {
	const token = await getToken();
	return new Promise<T>((resolve, reject) => {
		const xhr = new XMLHttpRequest();
		xhr.open(method, `${apiBaseUrl()}${path}`);
		xhr.withCredentials = false;
		xhr.responseType = 'text';
		if (token) xhr.setRequestHeader('Authorization', `Bearer ${token}`);
		xhr.setRequestHeader('Accept', 'application/json');
		xhr.upload.onprogress = (event) => {
			if (event.lengthComputable && onProgress) onProgress(event.loaded / event.total);
		};
		xhr.onerror = () => reject(new ApiError({ status: 0, title: 'Network error', errors: [] }));
		xhr.onload = () => {
			let body: unknown;
			try {
				body = xhr.responseText ? JSON.parse(xhr.responseText) : undefined;
			} catch {
				body = xhr.responseText;
			}
			if (xhr.status === 401) onUnauthorized();
			if (xhr.status >= 200 && xhr.status < 300) {
				onProgress?.(1);
				resolve((body as { data: T } | undefined)?.data as T);
			} else {
				reject(parseProblem(xhr.status, body));
			}
		};
		xhr.send(form);
	});
}
