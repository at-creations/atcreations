import { MutationCache, QueryClient } from '@tanstack/svelte-query';
import { ApiError } from '$lib/api-error';
import { keys } from './keys';

export const STALE_TIME = 30_000;
export const GC_TIME = 5 * 60_000;
export const MAX_RETRIES = 2;

/**
 * Retry predicate for queries. `failureCount` is the number of failures so far (0 on the first).
 * Network failures (status 0) and 5xx are retried up to {@link MAX_RETRIES} times; every 4xx
 * (401/403/404/409/422…) is final, and so is anything that is not an ApiError (a bug).
 */
export function shouldRetry(failureCount: number, error: unknown): boolean {
	if (failureCount >= MAX_RETRIES) return false;
	if (!(error instanceof ApiError)) return false;
	return error.status === 0 || error.status >= 500;
}

/**
 * One client per browser session (created by the authenticated layout; SSR is off).
 * Any successful mutation marks the audit log stale, since every write is audited.
 */
export function createAdminQueryClient(): QueryClient {
	const client: QueryClient = new QueryClient({
		defaultOptions: {
			queries: {
				staleTime: STALE_TIME,
				gcTime: GC_TIME,
				refetchOnWindowFocus: true,
				retry: shouldRetry
			},
			mutations: { retry: false }
		},
		mutationCache: new MutationCache({
			onSuccess: () => client.invalidateQueries({ queryKey: keys.audit.all })
		})
	});
	return client;
}
