import { MutationCache, QueryClient } from '@tanstack/svelte-query';
import { ApiError } from '$lib/api-error';
import { keys } from './keys';

/**
 * Freshness policy: cached data is shown instantly (stale-while-revalidate) and re-checked in the
 * background whenever a screen mounts, the tab regains focus or the network comes back. The short
 * stale time only collapses duplicate requests from components mounting together. Volatile data
 * (orders) additionally polls while the tab is visible — see ORDERS_POLL_MS.
 */
export const STALE_TIME = 5_000;
export const GC_TIME = 10 * 60_000;
export const MAX_RETRIES = 2;
/** Cross-tab freshness: a write in one admin tab invalidates the caches of the others. */
export const SYNC_CHANNEL = 'atc-admin:query-sync';
const SYNC_MESSAGE = 'invalidate';

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
 * Any successful mutation marks the audit log stale, since every write is audited, and is
 * broadcast to the admin's other tabs.
 */
export function createAdminQueryClient(): QueryClient {
	const client: QueryClient = new QueryClient({
		defaultOptions: {
			queries: {
				staleTime: STALE_TIME,
				gcTime: GC_TIME,
				refetchOnMount: true,
				refetchOnWindowFocus: true,
				refetchOnReconnect: true,
				// Polling pauses in background tabs; focus triggers a refetch on return.
				refetchIntervalInBackground: false,
				retry: shouldRetry
			},
			mutations: { retry: false }
		},
		mutationCache: new MutationCache({
			onSuccess: () => {
				void client.invalidateQueries({ queryKey: keys.audit.all });
				// Tell the admin's other open tabs that data changed.
				sync?.postMessage(SYNC_MESSAGE);
			}
		})
	});
	const sync = typeof BroadcastChannel === 'undefined' ? null : new BroadcastChannel(SYNC_CHANNEL);
	if (sync) {
		sync.onmessage = (event) => {
			// Active queries refetch now; inactive ones on their next use.
			if (event.data === SYNC_MESSAGE) void client.invalidateQueries();
		};
	}
	if (typeof window !== 'undefined') {
		// A tab restored from the back/forward cache may hold old data.
		window.addEventListener('pageshow', (event) => {
			if (event.persisted) void client.invalidateQueries();
		});
	}
	return client;
}
