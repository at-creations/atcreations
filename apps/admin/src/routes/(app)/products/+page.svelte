<script lang="ts">
	import type { Product, ProductStatus } from '@atcreations/api-types';
	import { createQuery, keepPreviousData } from '@tanstack/svelte-query';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import CategoryBadges from '$lib/components/CategoryBadges.svelte';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import FilterTabs from '$lib/components/FilterTabs.svelte';
	import Icon from '$lib/components/Icon.svelte';
	import LoadState from '$lib/components/LoadState.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Pagination from '$lib/components/Pagination.svelte';
	import StatusBadge from '$lib/components/StatusBadge.svelte';
	import { formatDate, localized, priceErrorMessage, productStatusLabel } from '$lib/i18n';
	import { safeImageUrl, storefrontProductUrl } from '$lib/links';
	import { toastError } from '$lib/notify';
	import { m } from '$lib/paraglide/messages.js';
	import {
		categoriesQuery,
		createDeleteProductMutation,
		productListQuery,
		type ProductListParams
	} from '$lib/queries';
	import { confirmAction } from '$lib/state/confirm.svelte';
	import { getAdminSession } from '$lib/state/session.svelte';
	import { toast } from '$lib/state/toast.svelte';
	import { formatCAD, parseDollars } from '$lib/utils/money';
	import { readEnum, readInt, readString, withParams } from '$lib/utils/query';

	const session = getAdminSession();

	const STATUSES: readonly ProductStatus[] = ['published', 'draft', 'archived'];
	const SORTS = [
		'newest',
		'oldest',
		'price_asc',
		'price_desc',
		'name_asc',
		'name_desc',
		'stock_asc',
		'popular'
	] as const;
	type Sort = (typeof SORTS)[number];
	const sortLabels: Record<Sort, () => string> = {
		newest: m.sort_newest,
		oldest: m.sort_oldest,
		price_asc: m.sort_price_asc,
		price_desc: m.sort_price_desc,
		name_asc: m.sort_name_asc,
		name_desc: m.sort_name_desc,
		stock_asc: m.sort_stock_asc,
		popular: m.sort_popular
	};
	const PAGE_SIZES = [10, 25, 50] as const;
	const UNCATEGORIZED = 'uncategorized';
	const LOW_STOCK = 5;

	// --- URL query = source of truth ---
	const filters = $derived.by(() => {
		const p = page.url.searchParams;
		const min = parseDollars(readString(p, 'minPrice', 20));
		const max = parseDollars(readString(p, 'maxPrice', 20));
		const category = readString(p, 'category', 40);
		return {
			page: readInt(p, 'page', 1),
			pageSize: readInt(p, 'pageSize', 25, PAGE_SIZES),
			q: readString(p, 'q'),
			status: readEnum(p, 'status', STATUSES),
			category: category === UNCATEGORIZED || /^[a-f0-9]{24}$/.test(category) ? category : '',
			featured: readEnum(p, 'featured', ['true', 'false'] as const),
			minPrice: min.ok ? min.cents : null,
			maxPrice: max.ok ? max.cents : null,
			sort: readEnum(p, 'sort', SORTS, 'newest') ?? 'newest'
		};
	});

	/** Everything except status/page: shared by the list and the per-status counters. */
	const baseParams = $derived<ProductListParams>({
		q: filters.q || undefined,
		category: filters.category && filters.category !== UNCATEGORIZED ? filters.category : undefined,
		uncategorized: filters.category === UNCATEGORIZED ? true : undefined,
		featured: filters.featured === undefined ? undefined : filters.featured === 'true',
		minPrice: filters.minPrice ?? undefined,
		maxPrice: filters.maxPrice ?? undefined
	});

	const listParams = $derived<ProductListParams>({
		...baseParams,
		page: filters.page,
		pageSize: filters.pageSize,
		status: filters.status,
		sort: filters.sort
	});

	// keepPreviousData: paging/filtering keeps the current rows (dimmed) instead of flashing.
	const productsQuery = createQuery(() => ({
		...productListQuery(listParams),
		placeholderData: keepPreviousData
	}));
	const categoriesList = createQuery(() => categoriesQuery());
	const deleteProduct = createDeleteProductMutation();

	// Counters for the status tabs, for the current search/filters.
	const countAll = createQuery(() => ({
		...productListQuery({ ...baseParams, pageSize: 1 }),
		select: (res) => res.meta.total
	}));
	const countPublished = createQuery(() => ({
		...productListQuery({ ...baseParams, pageSize: 1, status: 'published' }),
		select: (res) => res.meta.total
	}));
	const countDraft = createQuery(() => ({
		...productListQuery({ ...baseParams, pageSize: 1, status: 'draft' }),
		select: (res) => res.meta.total
	}));
	const countArchived = createQuery(() => ({
		...productListQuery({ ...baseParams, pageSize: 1, status: 'archived' }),
		select: (res) => res.meta.total
	}));
	const statusTabs = $derived([
		{ value: '' as const, label: m.common_all(), count: countAll.data ?? null },
		{
			value: 'published' as const,
			label: productStatusLabel('published'),
			count: countPublished.data ?? null
		},
		{ value: 'draft' as const, label: productStatusLabel('draft'), count: countDraft.data ?? null },
		{
			value: 'archived' as const,
			label: productStatusLabel('archived'),
			count: countArchived.data ?? null
		}
	]);

	const products = $derived(productsQuery.data?.data ?? []);
	const meta = $derived(productsQuery.data?.meta ?? null);
	const categories = $derived(categoriesList.data ?? []);
	const showError = $derived(productsQuery.isError && !productsQuery.data);

	function navigate(changes: Record<string, string | number | null | undefined>) {
		// Same page, only the query string changes (built from page.url).
		// eslint-disable-next-line svelte/no-navigation-without-resolve
		goto(withParams(page.url, changes), { keepFocus: true, noScroll: true });
	}

	// --- Search (debounced 300 ms) ---
	let searchInput = $derived(filters.q);
	let searchTimer: ReturnType<typeof setTimeout> | undefined;
	function onSearchInput(value: string) {
		searchInput = value;
		clearTimeout(searchTimer);
		searchTimer = setTimeout(() => navigate({ q: value.trim() }), 300);
	}
	$effect(() => () => clearTimeout(searchTimer));

	// --- More filters panel: featured + price range (dollars in the UI and URL, cents for the API) ---
	let panelOpen = $state(false);
	let minInput = $derived(page.url.searchParams.get('minPrice') ?? '');
	let maxInput = $derived(page.url.searchParams.get('maxPrice') ?? '');
	let minError = $state<string>();
	let maxError = $state<string>();

	function applyPrice(event: SubmitEvent) {
		event.preventDefault();
		const min = parseDollars(minInput);
		const max = parseDollars(maxInput);
		minError = priceErrorMessage(min);
		maxError = priceErrorMessage(max);
		if (!min.ok || !max.ok) return;
		if (min.cents !== null && max.cents !== null && min.cents > max.cents) {
			maxError = m.validation_min_max();
			return;
		}
		navigate({
			minPrice: min.cents === null ? null : (min.cents / 100).toFixed(2),
			maxPrice: max.cents === null ? null : (max.cents / 100).toFixed(2)
		});
		panelOpen = false;
	}

	const categoryName = $derived(
		filters.category === UNCATEGORIZED
			? m.products_uncategorized()
			: localized(categories.find((c) => c.id === filters.category)?.name ?? { en: '…' })
	);

	/** Removable chips for every active filter except status (shown by the tabs). */
	const chips = $derived.by(() => {
		const out: { key: string; label: string; clear: Record<string, null> }[] = [];
		if (filters.q) out.push({ key: 'q', label: `“${filters.q}”`, clear: { q: null } });
		if (filters.category)
			out.push({
				key: 'category',
				label: `${m.products_filter_category()}: ${categoryName}`,
				clear: { category: null }
			});
		if (filters.featured)
			out.push({
				key: 'featured',
				label: filters.featured === 'true' ? m.products_featured_only() : m.products_not_featured(),
				clear: { featured: null }
			});
		if (filters.minPrice !== null || filters.maxPrice !== null) {
			const lo = filters.minPrice === null ? '' : formatCAD(filters.minPrice);
			const hi = filters.maxPrice === null ? '' : formatCAD(filters.maxPrice);
			out.push({
				key: 'price',
				label: `${m.products_price_range()}: ${lo || '0'} – ${hi || '∞'}`,
				clear: { minPrice: null, maxPrice: null }
			});
		}
		return out;
	});
	const panelFilterCount = $derived(
		(filters.category ? 1 : 0) +
			(filters.featured ? 1 : 0) +
			(filters.minPrice !== null || filters.maxPrice !== null ? 1 : 0)
	);
	const hasFilters = $derived(chips.length > 0 || !!filters.status);

	function clearFilters() {
		minError = maxError = undefined;
		searchInput = '';
		goto(resolve('/products'), { keepFocus: true, noScroll: true });
	}

	async function remove(product: Product) {
		const name = localized(product.name);
		const ok = await confirmAction({
			title: m.products_delete_title(),
			message: m.products_delete_message({ name }),
			confirmLabel: m.common_delete(),
			danger: true
		});
		if (!ok) return;
		try {
			await deleteProduct.mutateAsync(product.id);
			toast.success(m.products_deleted());
			if (products.length === 1 && filters.page > 1) navigate({ page: filters.page - 1 });
		} catch (error) {
			toastError(error, m.products_delete_failed());
		}
	}

	/** Row click opens the product, unless the click was on a control or text was selected. */
	function openRow(event: MouseEvent, product: Product) {
		const target = event.target as HTMLElement;
		if (target.closest('a, button, input, label') || window.getSelection()?.toString()) return;
		goto(resolve(`/products/${product.id}`));
	}

	const uid = $props.id();
</script>

{#snippet price(product: Product)}
	{#if product.finalPriceCents === null}
		<span class="text-sm text-zinc-400">{m.products_contact_price()}</span>
	{:else}
		<span class="font-medium text-zinc-900 tabular-nums">{formatCAD(product.finalPriceCents)}</span>
		{#if product.discountPercent > 0 && product.priceCents !== null}
			<span class="ml-1 text-xs text-zinc-400 tabular-nums line-through"
				>{formatCAD(product.priceCents)}</span
			>
			<span class="ml-1 badge bg-red-50 text-red-700"
				>{m.products_discount({ percent: product.discountPercent })}</span
			>
		{/if}
	{/if}
{/snippet}

{#snippet stockLabel(product: Product)}
	{#if product.stock === 0}
		<span class="inline-flex items-center gap-1.5 text-sm font-medium text-red-600">
			<span class="size-1.5 rounded-full bg-red-500" aria-hidden="true"
			></span>{m.dash_out_of_stock()}
		</span>
	{:else}
		<span
			class={[
				'text-sm tabular-nums',
				product.stock < LOW_STOCK ? 'font-medium text-amber-700' : 'text-zinc-600'
			]}>{m.dash_in_stock({ count: product.stock })}</span
		>
	{/if}
{/snippet}

{#snippet thumb(product: Product, size: string)}
	{@const cover = safeImageUrl(product.images[0]?.url)}
	{#if cover}
		<img
			src={cover}
			alt=""
			class={[size, 'shrink-0 rounded-lg object-cover ring-1 ring-zinc-200']}
			loading="lazy"
		/>
	{:else}
		<span
			class={[
				size,
				'flex shrink-0 items-center justify-center rounded-lg bg-zinc-100 text-zinc-400'
			]}
		>
			<Icon name="image" size={16} label={m.products_no_image()} />
		</span>
	{/if}
{/snippet}

{#snippet rowActions(product: Product)}
	{@const siteUrl = product.status === 'published' ? storefrontProductUrl(product.slug) : null}
	<div class="flex items-center justify-end gap-0.5">
		{#if siteUrl}
			<!-- eslint-disable svelte/no-navigation-without-resolve -- resolved or external URL -->
			<a
				class="btn-ghost size-8 btn-sm px-0"
				href={siteUrl}
				target="_blank"
				rel="noopener noreferrer"
				title={m.products_view_on_site()}
			>
				<Icon name="external" size={16} />
				<span class="sr-only">{m.products_view_on_site()}</span>
			</a>
			<!-- eslint-enable svelte/no-navigation-without-resolve -->
		{/if}
		<a
			class="btn-ghost size-8 btn-sm px-0"
			href={resolve(`/products/${product.id}`)}
			title={m.products_edit_tooltip()}
		>
			<Icon name="edit" size={16} />
			<span class="sr-only">{m.common_edit()}</span>
		</a>
		{#if session.can('products:delete')}
			<button
				type="button"
				class="btn-ghost size-8 btn-sm px-0 hover:bg-red-50 hover:text-red-600"
				title={m.products_delete_tooltip()}
				onclick={() => remove(product)}
			>
				<Icon name="trash" size={16} />
				<span class="sr-only">{m.common_delete()}</span>
			</button>
		{/if}
	</div>
{/snippet}

<PageHeader title={m.nav_products()}>
	{#snippet actions()}
		{#if session.can('products:write')}
			<a class="btn-primary" href={resolve('/products/new')}
				><Icon name="plus" size={16} />{m.products_add()}</a
			>
		{/if}
	{/snippet}
</PageHeader>

<section class="card" aria-busy={productsQuery.isFetching}>
	<div class="border-b border-zinc-200 px-3">
		<FilterTabs
			tabs={statusTabs}
			value={filters.status ?? ''}
			label={m.products_filter_status()}
			onselect={(status) => navigate({ status })}
		/>
	</div>

	<!-- Toolbar -->
	<div class="flex flex-col gap-2 p-3 md:flex-row md:items-center" aria-label={m.common_filters()}>
		<div class="relative flex-1">
			<label for="{uid}-q" class="sr-only">{m.common_search()}</label>
			<Icon
				name="search"
				size={16}
				class="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-zinc-400"
			/>
			<input
				id="{uid}-q"
				type="search"
				class="input pl-9"
				placeholder={m.products_search_placeholder()}
				maxlength="100"
				value={searchInput}
				oninput={(e) => onSearchInput(e.currentTarget.value)}
			/>
		</div>
		<div class="flex flex-wrap items-center gap-2">
			<label for="{uid}-sort" class="sr-only">{m.common_sort_by()}</label>
			<select
				id="{uid}-sort"
				class="input w-auto flex-1 md:flex-none"
				value={filters.sort}
				onchange={(e) =>
					navigate({ sort: e.currentTarget.value === 'newest' ? null : e.currentTarget.value })}
			>
				{#each SORTS as sort (sort)}
					<option value={sort}>{sortLabels[sort]()}</option>
				{/each}
			</select>
			<button
				type="button"
				class={['btn-secondary', panelOpen && 'border-brand-300 bg-brand-50 text-brand-800']}
				aria-expanded={panelOpen}
				aria-controls="{uid}-panel"
				onclick={() => (panelOpen = !panelOpen)}
			>
				<Icon name="filter" size={16} />{m.common_filters()}
				{#if panelFilterCount > 0}
					<span class="rounded-full bg-brand-200 px-1.5 text-[11px] font-semibold text-brand-900"
						>{panelFilterCount}</span
					>
				{/if}
			</button>
		</div>
	</div>

	{#if panelOpen}
		<div
			id="{uid}-panel"
			class="grid animate-slide-down gap-4 border-t border-zinc-100 bg-zinc-50/60 p-4 md:grid-cols-[auto_auto_1fr]"
		>
			<div>
				<label for="{uid}-category" class="label">{m.products_filter_category()}</label>
				<select
					id="{uid}-category"
					class="input w-auto min-w-44"
					value={filters.category}
					onchange={(e) => navigate({ category: e.currentTarget.value })}
				>
					<option value="">{m.products_all_categories()}</option>
					<option value={UNCATEGORIZED}>{m.products_uncategorized()}</option>
					{#each categories as category (category.id)}
						<option value={category.id}>{localized(category.name)}</option>
					{/each}
				</select>
			</div>
			<fieldset>
				<legend class="label">{m.products_filter_featured()}</legend>
				<div class="inline-flex rounded-lg border border-zinc-300 bg-white p-0.5 shadow-xs">
					{#each [['', m.common_all()], ['true', m.products_featured_only()], ['false', m.products_not_featured()]] as [value, label] (value)}
						<button
							type="button"
							aria-pressed={(filters.featured ?? '') === value}
							class={[
								'rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
								(filters.featured ?? '') === value
									? 'bg-zinc-900 text-white'
									: 'text-zinc-600 hover:text-zinc-900'
							]}
							onclick={() => navigate({ featured: value })}>{label}</button
						>
					{/each}
				</div>
			</fieldset>
			<form class="flex flex-wrap items-end gap-2" onsubmit={applyPrice} novalidate>
				<fieldset class="flex flex-wrap items-start gap-2">
					<legend class="label">{m.products_price_range()} (CAD)</legend>
					<div>
						<label for="{uid}-min" class="sr-only">{m.products_min()}</label>
						<input
							id="{uid}-min"
							class="input w-28"
							inputmode="decimal"
							placeholder={m.products_min()}
							bind:value={minInput}
							aria-invalid={minError ? 'true' : undefined}
							aria-describedby={minError ? `${uid}-min-err` : undefined}
						/>
						{#if minError}<p id="{uid}-min-err" class="mt-1 text-xs text-red-600">
								{minError}
							</p>{/if}
					</div>
					<span class="mt-2 text-zinc-400" aria-hidden="true">–</span>
					<div>
						<label for="{uid}-max" class="sr-only">{m.products_max()}</label>
						<input
							id="{uid}-max"
							class="input w-28"
							inputmode="decimal"
							placeholder={m.products_max()}
							bind:value={maxInput}
							aria-invalid={maxError ? 'true' : undefined}
							aria-describedby={maxError ? `${uid}-max-err` : undefined}
						/>
						{#if maxError}<p id="{uid}-max-err" class="mt-1 text-xs text-red-600">
								{maxError}
							</p>{/if}
					</div>
					<button type="submit" class="btn-secondary">{m.common_apply()}</button>
				</fieldset>
			</form>
		</div>
	{/if}

	{#if chips.length > 0}
		<div class="flex flex-wrap items-center gap-2 border-t border-zinc-100 px-3 py-2.5">
			{#each chips as chip (chip.key)}
				<span
					class="inline-flex items-center gap-1 rounded-full border border-zinc-200 bg-white py-0.5 pr-1 pl-2.5 text-xs font-medium text-zinc-700"
				>
					{chip.label}
					<button
						type="button"
						class="rounded-full p-0.5 text-zinc-400 hover:bg-zinc-100 hover:text-zinc-700"
						aria-label="{m.common_clear()}: {chip.label}"
						onclick={() => {
							if (chip.key === 'q') searchInput = '';
							navigate(chip.clear);
						}}
					>
						<Icon name="x" size={12} />
					</button>
				</span>
			{/each}
			<button
				type="button"
				class="text-xs font-medium text-brand-600 hover:text-brand-700"
				onclick={clearFilters}>{m.products_clear_filters()}</button
			>
		</div>
	{/if}

	<div class="border-t border-zinc-200">
		{#if showError}
			<LoadState
				error={productsQuery.error}
				fallback={m.products_load_failed()}
				onretry={() => productsQuery.refetch()}
			/>
		{:else if productsQuery.isPending}
			<div class="divide-y divide-zinc-100" role="status" aria-label={m.common_loading()}>
				{#each [0, 1, 2, 3, 4] as i (i)}
					<div class="flex items-center gap-3 px-4 py-3">
						<div class="size-11 animate-pulse rounded-lg bg-zinc-100"></div>
						<div class="h-3.5 w-1/3 animate-pulse rounded bg-zinc-100"></div>
					</div>
				{/each}
			</div>
		{:else if products.length === 0}
			{#if hasFilters}
				<EmptyState icon="search" title={m.products_no_results()}>
					<button type="button" class="btn-secondary" onclick={clearFilters}
						>{m.products_clear_filters()}</button
					>
				</EmptyState>
			{:else}
				<EmptyState icon="box" title={m.products_empty_title()}>
					{#if session.can('products:write')}
						<a class="btn-primary" href={resolve('/products/new')}
							><Icon name="plus" size={16} />{m.products_add()}</a
						>
					{/if}
				</EmptyState>
			{/if}
		{:else}
			<!-- Table (≥ lg) -->
			<div class="hidden overflow-x-auto lg:block">
				<table
					class={[
						'w-full text-left text-sm',
						productsQuery.isPlaceholderData && 'opacity-60 transition-opacity'
					]}
				>
					<thead class="border-b border-zinc-200 bg-zinc-50/80">
						<tr>
							<th scope="col" class="th pl-5">{m.products_col_product()}</th>
							<th scope="col" class="th">{m.products_col_status()}</th>
							<th scope="col" class="th">{m.products_col_stock()}</th>
							<th scope="col" class="th">{m.products_col_price()}</th>
							<th scope="col" class="th">{m.products_col_categories()}</th>
							<th scope="col" class="th">{m.products_col_created()}</th>
							<th scope="col" class="th pr-5 text-right"
								><span class="sr-only">{m.common_actions()}</span></th
							>
						</tr>
					</thead>
					<tbody class="divide-y divide-zinc-100">
						{#each products as product (product.id)}
							<tr
								class="group cursor-pointer transition-colors hover:bg-zinc-50"
								onclick={(e) => openRow(e, product)}
							>
								<td class="td pl-5">
									<div class="flex items-center gap-3">
										{@render thumb(product, 'size-11')}
										<div class="min-w-0">
											<div class="flex items-center gap-1.5">
												<a
													class="block max-w-72 truncate font-medium text-zinc-900 group-hover:text-brand-700"
													href={resolve(`/products/${product.id}`)}>{localized(product.name)}</a
												>
												{#if product.featured}
													<Icon
														name="star"
														size={14}
														filled
														class="text-amber-400"
														label={m.products_featured()}
													/>
												{/if}
											</div>
											<span class="block truncate font-mono text-xs text-zinc-400"
												>{product.slug}</span
											>
										</div>
									</div>
								</td>
								<td class="td"><StatusBadge kind="product" value={product.status} /></td>
								<td class="td">{@render stockLabel(product)}</td>
								<td class="td whitespace-nowrap">{@render price(product)}</td>
								<td class="td"><CategoryBadges categories={product.categories} /></td>
								<td class="td whitespace-nowrap text-zinc-500">{formatDate(product.createdAt)}</td>
								<td
									class="td pr-5 opacity-60 transition-opacity group-focus-within:opacity-100 group-hover:opacity-100"
									>{@render rowActions(product)}</td
								>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>

			<!-- List (< lg) -->
			<ul
				class={[
					'divide-y divide-zinc-100 lg:hidden',
					productsQuery.isPlaceholderData && 'opacity-60'
				]}
			>
				{#each products as product (product.id)}
					<li class="relative flex gap-3 p-4">
						{@render thumb(product, 'size-14')}
						<div class="min-w-0 flex-1">
							<div class="flex items-start gap-2">
								<a
									class="min-w-0 flex-1 truncate font-medium text-zinc-900 after:absolute after:inset-0"
									href={resolve(`/products/${product.id}`)}>{localized(product.name)}</a
								>
								<StatusBadge kind="product" value={product.status} />
							</div>
							<div class="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-sm">
								<span>{@render price(product)}</span>
								{@render stockLabel(product)}
							</div>
						</div>
					</li>
				{/each}
			</ul>
		{/if}
	</div>
	{#if meta && !showError && products.length > 0}
		<Pagination
			{meta}
			pageSizes={[...PAGE_SIZES]}
			onpage={(p) => navigate({ page: p })}
			onpagesize={(size) => navigate({ pageSize: size })}
		/>
	{/if}
</section>
