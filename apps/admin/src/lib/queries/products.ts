import type { Image, Product, ProductInput } from '@atcreations/api-types';
import {
	createMutation,
	queryOptions,
	useQueryClient,
	type QueryClient
} from '@tanstack/svelte-query';
import { api, unwrap } from '$lib/api';
import { keys, type ProductListParams, type ProductStatsParams } from './keys';

export function productListQuery(params: ProductListParams) {
	return queryOptions({
		queryKey: keys.products.list(params),
		queryFn: ({ signal }) =>
			unwrap(api().GET('/v1/admin/products', { params: { query: params }, signal }))
	});
}

/** Counts per status and stock alerts for a set of filters, in one request. */
export function productStatsQuery(params: ProductStatsParams = {}) {
	return queryOptions({
		queryKey: keys.products.stats(params),
		queryFn: async ({ signal }) =>
			(await unwrap(api().GET('/v1/admin/products/stats', { params: { query: params }, signal })))
				.data
	});
}

export function productDetailQuery(id: string) {
	return queryOptions({
		queryKey: keys.products.detail(id),
		queryFn: async ({ signal }) =>
			(await unwrap(api().GET('/v1/admin/products/{id}', { params: { path: { id } }, signal })))
				.data
	});
}

/**
 * Store the server's canonical product (with its new `version`) in the cache. In-flight fetches of
 * the same product are cancelled first so an older response cannot overwrite it afterwards.
 */
export async function putProduct(queryClient: QueryClient, product: Product) {
	const queryKey = keys.products.detail(product.id);
	await queryClient.cancelQueries({ queryKey, exact: true });
	queryClient.setQueryData<Product>(queryKey, product);
	void queryClient.invalidateQueries({ queryKey: keys.products.lists() });
}

/** Refetch a product from the server (e.g. after a 409) and return the fresh copy. */
export function reloadProduct(queryClient: QueryClient, id: string): Promise<Product> {
	return queryClient.fetchQuery({ ...productDetailQuery(id), staleTime: 0 });
}

export function createSaveProductMutation() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: async ({ id, body }: { id?: string; body: ProductInput }) =>
			(id
				? await unwrap(api().PUT('/v1/admin/products/{id}', { params: { path: { id } }, body }))
				: await unwrap(api().POST('/v1/admin/products', { body }))
			).data,
		onSuccess: (product) => putProduct(queryClient, product)
	}));
}

export function createDeleteProductMutation() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: (id: string) =>
			unwrap(api().DELETE('/v1/admin/products/{id}', { params: { path: { id } } })),
		onSuccess: (_data, id) => {
			queryClient.removeQueries({ queryKey: keys.products.detail(id), exact: true });
			return queryClient.invalidateQueries({ queryKey: keys.products.lists() });
		}
	}));
}

export function createReorderImagesMutation() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: async ({ id, images }: { id: string; images: Image[] }) =>
			(
				await unwrap(
					api().PUT('/v1/admin/products/{id}/images/order', {
						params: { path: { id } },
						body: { imageIds: images.map((i) => i.id) }
					})
				)
			).data,
		// Optimistic: show the new order immediately, roll back on failure.
		onMutate: async ({ id, images }) => {
			const queryKey = keys.products.detail(id);
			await queryClient.cancelQueries({ queryKey, exact: true });
			const previous = queryClient.getQueryData<Product>(queryKey);
			if (previous) queryClient.setQueryData<Product>(queryKey, { ...previous, images });
			return { previous };
		},
		onError: (_error, { id }, context) => {
			if (context?.previous) queryClient.setQueryData(keys.products.detail(id), context.previous);
		},
		onSuccess: (product) => putProduct(queryClient, product)
	}));
}

export function createRemoveImageMutation() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: async ({ id, imageId }: { id: string; imageId: string }) =>
			(
				await unwrap(
					api().DELETE('/v1/admin/products/{id}/images/{imageId}', {
						params: { path: { id, imageId } }
					})
				)
			).data,
		onSuccess: (product) => putProduct(queryClient, product)
	}));
}
