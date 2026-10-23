import type { UploadJob } from '@atcreations/api-types';
import { createMutation, queryOptions, useQueryClient } from '@tanstack/svelte-query';
import { api, unwrap } from '$lib/api';
import { keys } from './keys';

type Kind = 'product' | 'category';

export const isActiveJob = (j: UploadJob) => j.status === 'queued' || j.status === 'processing';

/** Poll fast while something is processing; stop when everything has settled. */
export const UPLOAD_POLL_MS = 1000;

export function uploadJobsQuery(kind: Kind, id: string) {
	return queryOptions({
		queryKey: keys.uploads.list(kind, id),
		queryFn: async ({ signal }) => {
			const res =
				kind === 'product'
					? await unwrap(
							api().GET('/v1/admin/products/{id}/uploads', { params: { path: { id } }, signal })
						)
					: await unwrap(
							api().GET('/v1/admin/categories/{id}/uploads', { params: { path: { id } }, signal })
						);
			return res.data;
		},
		refetchInterval: (query) => (query.state.data?.some(isActiveJob) ? UPLOAD_POLL_MS : false),
		// Progress must be live even in a background tab while the user waits for it.
		refetchIntervalInBackground: true
	});
}

/** Cancel a queued upload or dismiss a failed one. */
export function createCancelUploadMutation(kind: Kind, id: () => string) {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: (jobId: string) =>
			kind === 'product'
				? unwrap(
						api().DELETE('/v1/admin/products/{id}/uploads/{jobId}', {
							params: { path: { id: id(), jobId } }
						})
					)
				: unwrap(
						api().DELETE('/v1/admin/categories/{id}/uploads/{jobId}', {
							params: { path: { id: id(), jobId } }
						})
					),
		onSuccess: (_d, jobId) =>
			queryClient.setQueryData<UploadJob[]>(keys.uploads.list(kind, id()), (jobs) =>
				jobs?.filter((j) => j.id !== jobId)
			)
	}));
}
