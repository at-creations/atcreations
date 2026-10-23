import type { AdminCategory, Category, CategoryInput } from '@atcreations/api-types';
import {
	createMutation,
	queryOptions,
	useQueryClient,
	type QueryClient
} from '@tanstack/svelte-query';
import { api, unwrap } from '$lib/api';
import { keys } from './keys';

export function categoriesQuery() {
	return queryOptions({
		queryKey: keys.categories.all,
		queryFn: async ({ signal }) =>
			(await unwrap(api().GET('/v1/admin/categories', { signal }))).data
	});
}

/** Insert or replace a category in the cached list (keeping its product count), then mark dependants stale. */
export function putCategory(queryClient: QueryClient, category: Category) {
	queryClient.setQueryData<AdminCategory[]>(keys.categories.all, (old) => {
		if (!old) return old;
		return old.some((c) => c.id === category.id)
			? old.map((c) => (c.id === category.id ? { ...category, productCount: c.productCount } : c))
			: [...old, { ...category, productCount: 0 }];
	});
	void queryClient.invalidateQueries({ queryKey: keys.categories.all });
	// Products embed category names in their badges.
	void queryClient.invalidateQueries({ queryKey: keys.products.lists() });
}

export function createSaveCategoryMutation() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: async ({ id, body }: { id?: string; body: CategoryInput }) =>
			(id
				? await unwrap(api().PUT('/v1/admin/categories/{id}', { params: { path: { id } }, body }))
				: await unwrap(api().POST('/v1/admin/categories', { body }))
			).data,
		onSuccess: (category) => putCategory(queryClient, category)
	}));
}

export function createDeleteCategoryMutation() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: (id: string) =>
			unwrap(api().DELETE('/v1/admin/categories/{id}', { params: { path: { id } } })),
		onSuccess: (_data, id) => {
			queryClient.setQueryData<AdminCategory[]>(keys.categories.all, (old) =>
				old?.filter((c) => c.id !== id)
			);
			void queryClient.invalidateQueries({ queryKey: keys.categories.all });
			// Products may have lost this category.
			return queryClient.invalidateQueries({ queryKey: keys.products.all });
		}
	}));
}

export function createRemoveCategoryImageMutation() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: async (id: string) =>
			(await unwrap(api().DELETE('/v1/admin/categories/{id}/image', { params: { path: { id } } })))
				.data,
		onSuccess: (category) => putCategory(queryClient, category)
	}));
}

/** Save the display order. Optimistic: the list reorders at once and rolls back on failure. */
export function createReorderCategoriesMutation() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: async (ordered: AdminCategory[]) =>
			(
				await unwrap(
					api().PUT('/v1/admin/categories/order', {
						body: { categoryIds: ordered.map((c) => c.id) }
					})
				)
			).data,
		onMutate: async (ordered) => {
			await queryClient.cancelQueries({ queryKey: keys.categories.all });
			const previous = queryClient.getQueryData<AdminCategory[]>(keys.categories.all);
			queryClient.setQueryData(keys.categories.all, ordered);
			return { previous };
		},
		onError: (_e, _v, context) => {
			if (context?.previous) queryClient.setQueryData(keys.categories.all, context.previous);
		},
		onSuccess: (list) => {
			queryClient.setQueryData(keys.categories.all, list);
			// Product badges list categories in display order.
			void queryClient.invalidateQueries({ queryKey: keys.products.lists() });
		}
	}));
}
