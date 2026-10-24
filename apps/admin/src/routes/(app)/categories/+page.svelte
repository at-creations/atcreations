<script lang="ts">
	import { imageUrl } from '$lib/utils/images';
	import type { AdminCategory } from '@atcreations/api-types';
	import { createQuery } from '@tanstack/svelte-query';
	import { resolve } from '$app/paths';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import Icon from '$lib/components/Icon.svelte';
	import LoadState from '$lib/components/LoadState.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import SearchInput from '$lib/components/ui/SearchInput.svelte';
	import Thumb from '$lib/components/ui/Thumb.svelte';
	import Tip from '$lib/components/ui/Tip.svelte';
	import { localized } from '$lib/i18n';
	import { toastError } from '$lib/notify';
	import { m } from '$lib/paraglide/messages.js';
	import { categoriesQuery, createReorderCategoriesMutation } from '$lib/queries';
	import { getAdminSession } from '$lib/state/session.svelte';
	import { moveItem } from '$lib/utils/files';

	const session = getAdminSession();
	const canWrite = $derived(session.can('categories:write'));

	const list = createQuery(() => categoriesQuery());
	const reorder = createReorderCategoriesMutation();
	const categories = $derived(list.data ?? []);

	let search = $state('');
	const query = $derived(search.trim().toLocaleLowerCase());
	const visible = $derived(
		query
			? categories.filter((c) =>
					[c.name.en, c.name.vi ?? '', c.slug].some((s) => s.toLocaleLowerCase().includes(query))
				)
			: categories
	);
	// Reordering only makes sense on the full list.
	const canReorder = $derived(canWrite && !query && categories.length > 1);

	let dragIndex = $state<number | null>(null);
	let dropIndex = $state<number | null>(null);
	let announcement = $state('');

	async function move(from: number, to: number, focus = false) {
		if (to < 0 || to >= categories.length || from === to || reorder.isPending) return;
		const next = moveItem(categories, from, to);
		try {
			await reorder.mutateAsync(next);
			announcement = m.cat_order_saved();
			if (focus) document.getElementById(`cat-move-${next[to]?.id}`)?.focus();
		} catch (error) {
			toastError(error, m.cat_order_failed());
		}
	}

	const productsHref = (c: AdminCategory) => `${resolve('/products')}?category=${c.id}`;
</script>

<PageHeader title={m.nav_categories()}>
	{#snippet meta()}
		{#if list.data}
			<span
				class="rounded-full bg-zinc-100 px-2 py-0.5 text-xs font-medium text-zinc-600 tabular-nums"
				>{categories.length}</span
			>
		{/if}
	{/snippet}
	{#snippet actions()}
		{#if canWrite}
			<a class="btn-primary" href={resolve('/categories/new')}>
				<Icon name="plus" size={16} />{m.cat_add()}
			</a>
		{/if}
	{/snippet}
</PageHeader>

{#if list.isPending}
	<div class="card"><LoadState loading /></div>
{:else if list.isError && !list.data}
	<div class="card">
		<LoadState error={list.error} fallback={m.cat_load_failed()} onretry={() => list.refetch()} />
	</div>
{:else if categories.length === 0}
	<div class="card">
		<EmptyState icon="tag" title={m.cat_none()}>
			{#if canWrite}
				<a class="btn-primary" href={resolve('/categories/new')}
					><Icon name="plus" size={16} />{m.cat_add()}</a
				>
			{/if}
		</EmptyState>
	</div>
{:else}
	<div class="mb-4 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
		<SearchInput bind:value={search} class="sm:max-w-xs" placeholder={m.cat_search()} delay={0} />
		{#if canReorder}
			<p class="flex items-center gap-1.5 text-xs text-zinc-500">
				<Icon name="grip" size={14} />{m.cat_drag_hint()}
			</p>
		{/if}
	</div>
	<p class="sr-only" aria-live="polite">{announcement}</p>

	<div class={['overflow-hidden card', reorder.isPending && 'opacity-80']}>
		{#if visible.length === 0}
			<EmptyState icon="search" title={m.cat_no_results()} />
		{:else}
			<ol class="divide-y divide-zinc-100" aria-busy={reorder.isPending}>
				{#each visible as category, index (category.id)}
					<li
						draggable={canReorder}
						class={[
							'group flex items-center gap-3 px-3 py-3 transition-colors sm:px-4',
							dropIndex === index &&
								dragIndex !== index &&
								'bg-brand-50 ring-1 ring-brand-300 ring-inset',
							dragIndex === index && 'opacity-40',
							'hover:bg-zinc-50'
						]}
						ondragstart={(e) => {
							dragIndex = index;
							e.dataTransfer?.setData('text/plain', category.id);
							if (e.dataTransfer) e.dataTransfer.effectAllowed = 'move';
						}}
						ondragover={(e) => {
							if (dragIndex === null) return;
							e.preventDefault();
							dropIndex = index;
						}}
						ondragleave={() => {
							if (dropIndex === index) dropIndex = null;
						}}
						ondrop={(e) => {
							e.preventDefault();
							if (dragIndex !== null && dragIndex !== index) void move(dragIndex, index);
							dragIndex = dropIndex = null;
						}}
						ondragend={() => (dragIndex = dropIndex = null)}
					>
						{#if canReorder}
							<span
								class="hidden cursor-grab text-zinc-300 group-hover:text-zinc-500 active:cursor-grabbing sm:block"
								aria-hidden="true"
							>
								<Icon name="grip" size={16} />
							</span>
						{/if}
						<Thumb
							src={imageUrl(category.image, 96)}
							class="size-12 sm:size-14"
							rounded="rounded-xl"
							alt=""
						/>
						<div class="min-w-0 flex-1">
							<a
								href={resolve(`/categories/${category.id}`)}
								class="block truncate font-medium text-zinc-900 hover:text-brand-700"
							>
								{category.name.en}
							</a>
							<p class="truncate text-sm text-zinc-500">
								{#if category.name.vi}<span lang="vi">{category.name.vi}</span>&nbsp;·&ensp;
								{/if}<span class="font-mono text-xs">{category.slug}</span>
							</p>
						</div>
						<!-- eslint-disable svelte/no-navigation-without-resolve -- resolved path + query -->
						<a
							href={productsHref(category)}
							class="hidden shrink-0 rounded-full bg-zinc-100 px-2.5 py-1 text-xs font-medium text-zinc-600 tabular-nums hover:bg-brand-100 hover:text-brand-800 sm:inline-block"
						>
							{m.cat_product_count({ count: category.productCount })}
						</a>
						<!-- eslint-enable svelte/no-navigation-without-resolve -->
						{#if canReorder}
							<div class="flex shrink-0 flex-col">
								<button
									id="cat-move-{category.id}"
									type="button"
									class="btn-icon btn-ghost size-6"
									aria-label="{m.common_move_up()}: {localized(category.name)}"
									disabled={index === 0 || reorder.isPending}
									onclick={() => move(index, index - 1, true)}
								>
									<Icon name="up" size={14} />
								</button>
								<button
									type="button"
									class="btn-icon btn-ghost size-6"
									aria-label="{m.common_move_down()}: {localized(category.name)}"
									disabled={index === visible.length - 1 || reorder.isPending}
									onclick={() => move(index, index + 1, true)}
								>
									<Icon name="down" size={14} />
								</button>
							</div>
						{/if}
						<Tip text={m.common_edit()}>
							{#snippet children({ props })}
								<a
									{...props}
									href={resolve(`/categories/${category.id}`)}
									class="btn-icon btn-ghost size-8 shrink-0"
									aria-label="{m.common_edit()}: {localized(category.name)}"
								>
									<Icon name="right" size={16} />
								</a>
							{/snippet}
						</Tip>
					</li>
				{/each}
			</ol>
		{/if}
	</div>
{/if}
