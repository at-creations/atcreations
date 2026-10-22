import { queryOptions } from '@tanstack/svelte-query';
import { api, unwrap } from '$lib/api';
import { keys, type AuditListParams } from './keys';

export function auditListQuery(params: AuditListParams) {
	return queryOptions({
		queryKey: keys.audit.list(params),
		queryFn: ({ signal }) =>
			unwrap(api().GET('/v1/admin/audit', { params: { query: params }, signal }))
	});
}
