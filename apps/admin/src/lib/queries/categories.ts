import type { Category, CategoryInput } from '@atcreations/api-types';
import {
	createMutation,
	queryOptions,
	useQueryClient,
	type QueryClient
} from '@tanstack/svelte-query';
import { api, unwrap, uploadWithProgress } from '$lib/api';
import { keys } from './keys';

export function categoriesQuery() {
	return queryOptions({
		queryKey: keys.categories.all,
		queryFn: async ({ signal }) =>
			(await unwrap(api().GET('/v1/admin/categories', { signal }))).data
	});
}

/** Insert or replace a category in the cached list, then mark dependants stale. */
export function putCategory(queryClient: QueryClient, category: Category) {
	queryClient.setQueryData<Category[]>(keys.categories.all, (old) => {
		if (!old) return old;
		return old.some((c) => c.id === category.id)
			? old.map((c) => (c.id === category.id ? category : c))
			: [...old, category];
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
			queryClient.setQueryData<Category[]>(keys.categories.all, (old) =>
				old?.filter((c) => c.id !== id)
			);
			void queryClient.invalidateQueries({ queryKey: keys.categories.all });
			// Products may have lost this category.
			return queryClient.invalidateQueries({ queryKey: keys.products.all });
		}
	}));
}

export function createCategoryImageMutation() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: async ({ id, file }: { id: string; file?: File }) => {
			if (!file) {
				return (
					await unwrap(
						api().DELETE('/v1/admin/categories/{id}/image', { params: { path: { id } } })
					)
				).data;
			}
			const form = new FormData();
			form.append('file', file, file.name);
			return uploadWithProgress<Category>(
				'PUT',
				`/v1/admin/categories/${encodeURIComponent(id)}/image`,
				form
			);
		},
		onSuccess: (category) => {
			if (category) putCategory(queryClient, category);
			else void queryClient.invalidateQueries({ queryKey: keys.categories.all });
		}
	}));
}
