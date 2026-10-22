import type { StoreInfo, StoreInfoInput } from '@atcreations/api-types';
import { createMutation, queryOptions, useQueryClient } from '@tanstack/svelte-query';
import { api, unwrap } from '$lib/api';
import { keys } from './keys';

export function storeInfoQuery() {
	return queryOptions({
		queryKey: keys.storeInfo,
		queryFn: async ({ signal }) => (await unwrap(api().GET('/v1/store-info', { signal }))).data
	});
}

export function createSaveStoreInfoMutation() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: async (body: StoreInfoInput) =>
			(await unwrap(api().PUT('/v1/admin/store-info', { body }))).data,
		onSuccess: async (info) => {
			await queryClient.cancelQueries({ queryKey: keys.storeInfo });
			queryClient.setQueryData<StoreInfo>(keys.storeInfo, info);
		}
	}));
}
