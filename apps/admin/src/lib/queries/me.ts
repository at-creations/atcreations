import { queryOptions } from '@tanstack/svelte-query';
import { api, unwrap } from '$lib/api';
import { keys } from './keys';

/** The signed-in staff member (role + permissions). */
export function meQuery() {
	return queryOptions({
		queryKey: keys.me,
		queryFn: async ({ signal }) => (await unwrap(api().GET('/v1/admin/me', { signal }))).data
	});
}
