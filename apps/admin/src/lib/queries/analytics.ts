import { queryOptions } from '@tanstack/svelte-query';
import { api, unwrap } from '$lib/api';
import { keys, type AnalyticsDays } from './keys';

/** The report changes with every order and visit; refresh it while the page is open. */
export const ANALYTICS_POLL_MS = 60_000;

export function analyticsQuery(days: AnalyticsDays) {
	return queryOptions({
		queryKey: keys.analytics.report(days),
		queryFn: async ({ signal }) =>
			(await unwrap(api().GET('/v1/admin/analytics', { params: { query: { days } }, signal })))
				.data,
		refetchInterval: ANALYTICS_POLL_MS
	});
}
