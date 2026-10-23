<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import CategoryForm from '$lib/components/categories/CategoryForm.svelte';
	import CategoryImage from '$lib/components/categories/CategoryImage.svelte';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import Icon from '$lib/components/Icon.svelte';
	import LoadState from '$lib/components/LoadState.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Thumb from '$lib/components/ui/Thumb.svelte';
	import { localized } from '$lib/i18n';
	import { toastError } from '$lib/notify';
	import { m } from '$lib/paraglide/messages.js';
	import { categoriesQuery, createDeleteCategoryMutation } from '$lib/queries';
	import { confirmAction } from '$lib/state/confirm.svelte';
	import { getAdminSession } from '$lib/state/session.svelte';
	import { toast } from '$lib/state/toast.svelte';

	const session = getAdminSession();
	const list = createQuery(() => categoriesQuery());
	const deleteCategory = createDeleteCategoryMutation();
	const id = $derived(page.params.id ?? '');
	const category = $derived(list.data?.find((c) => c.id === id));
	// Set while deleting, so the page does not flash "not found" before navigating away.
	let deleted = $state(false);

	async function remove() {
		if (!category) return;
		const name = localized(category.name);
		const ok = await confirmAction({
			title: m.cat_delete_title(),
			message: m.cat_delete_message({ name }),
			confirmLabel: m.common_delete(),
			danger: true
		});
		if (!ok) return;
		deleted = true;
		try {
			await deleteCategory.mutateAsync(category.id);
			toast.success(m.cat_deleted({ name }));
			await goto(resolve('/categories'));
		} catch (error) {
			deleted = false;
			toastError(error, m.cat_delete_failed());
		}
	}
</script>

{#if list.isPending || (deleted && !category)}
	<div class="space-y-6" role="status" aria-label={m.common_loading()}>
		<div class="h-8 w-64 animate-pulse rounded-md bg-zinc-200"></div>
		<div class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_20rem]">
			<div class="h-80 animate-pulse card"></div>
			<div class="h-64 animate-pulse card"></div>
		</div>
	</div>
{:else if list.isError && !list.data}
	<div class="card">
		<LoadState error={list.error} fallback={m.cat_load_failed()} onretry={() => list.refetch()} />
	</div>
{:else if !category}
	<div class="card">
		<EmptyState icon="tag" title={m.cat_not_found()}>
			<a class="btn-secondary" href={resolve('/categories')}
				><Icon name="left" size={16} />{m.nav_categories()}</a
			>
		</EmptyState>
	</div>
{:else}
	<PageHeader
		title={category.name.en}
		back={{ href: resolve('/categories'), label: m.nav_categories() }}
	>
		{#snippet leading()}
			<Thumb src={category.image?.url} class="hidden size-12 sm:flex" rounded="rounded-xl" />
		{/snippet}
		{#snippet actions()}
			{#if session.can('categories:write')}
				<button
					type="button"
					class="btn-ghost text-zinc-500 hover:bg-rose-50 hover:text-rose-600"
					disabled={deleteCategory.isPending}
					onclick={remove}
				>
					<Icon name="trash" size={16} />{m.common_delete()}
				</button>
			{/if}
		{/snippet}
	</PageHeader>
	{#key category.id}
		<CategoryForm {category}>
			{#snippet aside()}
				<CategoryImage {category} />
				<Card title={m.nav_products()}>
					<div class="flex items-center justify-between gap-3">
						<span class="text-2xl font-semibold text-zinc-900 tabular-nums"
							>{category.productCount}</span
						>
						<!-- eslint-disable svelte/no-navigation-without-resolve -- resolved path + query -->
						<a
							class="btn-secondary btn-sm"
							href={`${resolve('/products')}?category=${category.id}`}
						>
							{m.cat_view_products()}<Icon name="right" size={14} />
						</a>
						<!-- eslint-enable svelte/no-navigation-without-resolve -->
					</div>
				</Card>
			{/snippet}
		</CategoryForm>
	{/key}
{/if}
