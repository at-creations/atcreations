<script lang="ts">
	import { imageUrl } from '$lib/utils/images';
	import type { Product, ProductStatus } from '@atcreations/api-types';
	import { createQuery, keepPreviousData } from '@tanstack/svelte-query';
	import { Popover } from 'bits-ui';
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
	import PriceLabel from '$lib/components/products/PriceLabel.svelte';
	import ProductCard from '$lib/components/products/ProductCard.svelte';
	import StockLabel from '$lib/components/products/StockLabel.svelte';
	import StatusBadge from '$lib/components/StatusBadge.svelte';
	import Chip from '$lib/components/ui/Chip.svelte';
	import SearchInput from '$lib/components/ui/SearchInput.svelte';
	import Segmented from '$lib/components/ui/Segmented.svelte';
	import Thumb from '$lib/components/ui/Thumb.svelte';
	import Tip from '$lib/components/ui/Tip.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import {
		formatDate,
		formatTotal,
		localized,
		MAX_COUNT,
		priceErrorMessage,
		productStatusLabel
	} from '$lib/i18n';
	import { storefrontProductUrl } from '$lib/links';
	import { toastError } from '$lib/notify';
	import { m } from '$lib/paraglide/messages.js';
	import {
		categoriesQuery,
		createDeleteProductMutation,
		productListQuery,
		productStatsQuery,
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
	const PAGE_SIZES = [12, 24, 48] as const;
	const UNCATEGORIZED = 'uncategorized';

	// --- View (grid of cards or a dense table), remembered per browser ---
	type View = 'grid' | 'table';
	const VIEW_KEY = 'atc-admin:products-view';
	let view = $state<View>(readView());
	function readView(): View {
		try {
			return localStorage.getItem(VIEW_KEY) === 'table' ? 'table' : 'grid';
		} catch {
			return 'grid';
		}
	}
	function setView(next: View) {
		view = next;
		try {
			localStorage.setItem(VIEW_KEY, next);
		} catch {
			// Preference only.
		}
	}

	// --- URL query = source of truth ---
	const filters = $derived.by(() => {
		const p = page.url.searchParams;
		const min = parseDollars(readString(p, 'minPrice', 20));
		const max = parseDollars(readString(p, 'maxPrice', 20));
		const category = readString(p, 'category', 40);
		return {
			page: readInt(p, 'page', 1),
			pageSize: readInt(p, 'pageSize', 24, PAGE_SIZES),
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

	// keepPreviousData: paging/filtering keeps the current rows (dimmed) instead of flashing.
	const productsQuery = createQuery(() => ({
		...productListQuery({
			...baseParams,
			page: filters.page,
			pageSize: filters.pageSize,
			status: filters.status,
			sort: filters.sort
		}),
		placeholderData: keepPreviousData
	}));
	const categoriesList = createQuery(() => categoriesQuery());
	const deleteProduct = createDeleteProductMutation();

	// One request for every tab's counter (same filters, any status).
	const stats = createQuery(() => ({
		...productStatsQuery(baseParams),
		placeholderData: keepPreviousData
	}));
	// Counts stop at 10,000 (the API caps them): shown as "10,000+".
	const tabCount = (n: number | undefined) =>
		n === undefined ? null : formatTotal({ total: n, totalCapped: n >= MAX_COUNT });
	const statusTabs = $derived([
		{ value: '' as const, label: m.common_all(), count: tabCount(stats.data?.all) },
		...STATUSES.map((s) => ({
			value: s,
			label: productStatusLabel(s),
			count: tabCount(stats.data?.[s])
		}))
	]);

	const products = $derived(productsQuery.data?.data ?? []);
	const meta = $derived(productsQuery.data?.meta ?? null);
	const categories = $derived(categoriesList.data ?? []);
	const showError = $derived(productsQuery.isError && !productsQuery.data);
	const dimmed = $derived(productsQuery.isPlaceholderData);

	function navigate(changes: Record<string, string | number | null | undefined>) {
		// Same page, only the query string changes (built from page.url).
		// eslint-disable-next-line svelte/no-navigation-without-resolve
		goto(withParams(page.url, changes), { keepFocus: true, noScroll: true });
	}

	let search = $derived(filters.q);

	// --- Filters panel: category, featured, price (dollars in the UI and URL, cents for the API) ---
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
	}

	const categoryItems = $derived([
		{ value: '', label: m.products_all_categories() },
		{ value: UNCATEGORIZED, label: m.products_uncategorized() },
		...categories.map((c) => ({ value: c.id, label: localized(c.name) }))
	]);

	const categoryName = $derived(
		filters.category === UNCATEGORIZED
			? m.products_uncategorized()
			: localized(categories.find((c) => c.id === filters.category)?.name ?? { en: '…' })
	);

	/** Removable chips for every active filter except status (tabs) and search (its own box). */
	const chips = $derived.by(() => {
		const out: { key: string; label: string; clear: Record<string, null> }[] = [];
		if (filters.category)
			out.push({ key: 'category', label: categoryName, clear: { category: null } });
		if (filters.featured)
			out.push({
				key: 'featured',
				label: filters.featured === 'true' ? m.products_featured_only() : m.products_not_featured(),
				clear: { featured: null }
			});
		if (filters.minPrice !== null || filters.maxPrice !== null) {
			const lo = filters.minPrice === null ? '0' : formatCAD(filters.minPrice);
			const hi = filters.maxPrice === null ? '∞' : formatCAD(filters.maxPrice);
			out.push({ key: 'price', label: `${lo} – ${hi}`, clear: { minPrice: null, maxPrice: null } });
		}
		return out;
	});
	const hasFilters = $derived(chips.length > 0 || !!filters.status || !!filters.q);

	function clearFilters() {
		minError = maxError = undefined;
		search = '';
		goto(resolve('/products'), { keepFocus: true, noScroll: true });
	}

	async function remove(product: Product) {
		const ok = await confirmAction({
			title: m.products_delete_title(),
			message: m.products_delete_message({ name: localized(product.name) }),
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

<PageHeader title={m.nav_products()}>
	{#snippet meta()}
		{#if stats.data !== undefined && !hasFilters}
			<span
				class="rounded-full bg-zinc-100 px-2 py-0.5 text-xs font-medium text-zinc-600 tabular-nums"
				>{tabCount(stats.data.all)}</span
			>
		{/if}
	{/snippet}
	{#snippet actions()}
		<Segmented
			label={m.products_view()}
			iconOnly
			value={view}
			onchange={setView}
			options={[
				{ value: 'grid', label: m.products_view_grid(), icon: 'grid' },
				{ value: 'table', label: m.products_view_table(), icon: 'list' }
			]}
		/>
		{#if session.can('products:write')}
			<a class="btn-primary" href={resolve('/products/new')}
				><Icon name="plus" size={16} />{m.products_add()}</a
			>
		{/if}
	{/snippet}
</PageHeader>

<!-- Filters -->
<div class="mb-5 space-y-3">
	<div class="border-b border-zinc-200">
		<FilterTabs
			tabs={statusTabs}
			value={filters.status ?? ''}
			label={m.products_filter_status()}
			onselect={(status) => navigate({ status })}
		/>
	</div>

	<div class="flex flex-col gap-2 sm:flex-row sm:items-center">
		<SearchInput
			bind:value={search}
			class="flex-1 sm:max-w-sm"
			placeholder={m.products_search_placeholder()}
			onsearch={(q) => navigate({ q })}
		/>
		<div class="flex items-center gap-2 sm:ml-auto">
			<Popover.Root bind:open={panelOpen}>
				<Popover.Trigger
					class={['btn-secondary', panelOpen && 'border-brand-300 bg-brand-50 text-brand-800']}
				>
					<Icon name="filter" size={16} />{m.common_filters()}
					{#if chips.length > 0}
						<span class="rounded-full bg-brand-200 px-1.5 text-[11px] font-semibold text-brand-900"
							>{chips.length}</span
						>
					{/if}
				</Popover.Trigger>
				<Popover.Portal>
					<Popover.Content
						align="end"
						sideOffset={6}
						class="z-[60] grid w-[min(92vw,22rem)] animate-pop-in gap-4 rounded-2xl border border-zinc-200 bg-white p-4 shadow-popover"
					>
						<div>
							<span class="label">{m.products_filter_category()}</span>
							<Select
								label={m.products_filter_category()}
								class="min-w-48"
								value={filters.category}
								items={categoryItems}
								onchange={(category) => navigate({ category })}
							/>
						</div>
						<div>
							<span class="label">{m.products_filter_featured()}</span>
							<Segmented
								label={m.products_filter_featured()}
								value={filters.featured ?? ''}
								onchange={(featured) => navigate({ featured })}
								options={[
									{ value: '', label: m.common_all() },
									{ value: 'true', label: m.products_featured_only() },
									{ value: 'false', label: m.products_not_featured() }
								]}
							/>
						</div>
						<form onsubmit={applyPrice} novalidate>
							<fieldset>
								<legend class="label">{m.products_price_range()}</legend>
								<div class="flex items-start gap-2">
									<div>
										<label for="{uid}-min" class="sr-only">{m.products_min()}</label>
										<input
											id="{uid}-min"
											class="input w-24"
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
									<span class="mt-2 text-zinc-300" aria-hidden="true">–</span>
									<div>
										<label for="{uid}-max" class="sr-only">{m.products_max()}</label>
										<input
											id="{uid}-max"
											class="input w-24"
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
								</div>
							</fieldset>
						</form>
					</Popover.Content>
				</Popover.Portal>
			</Popover.Root>
			<Select
				label={m.common_sort_by()}
				class="w-auto min-w-40"
				value={filters.sort}
				items={SORTS.map((s) => ({ value: s, label: sortLabels[s]() }))}
				onchange={(v) => navigate({ sort: v === 'newest' ? null : v })}
			/>
		</div>
	</div>

	{#if chips.length > 0}
		<div class="flex flex-wrap items-center gap-2">
			{#each chips as chip (chip.key)}
				<Chip label={chip.label} onremove={() => navigate(chip.clear)} />
			{/each}
			<button
				type="button"
				class="text-xs font-medium text-zinc-500 hover:text-zinc-900"
				onclick={clearFilters}>{m.products_clear_filters()}</button
			>
		</div>
	{/if}
</div>

<!-- Results -->
<div aria-busy={productsQuery.isFetching}>
	{#if showError}
		<div class="card">
			<LoadState
				error={productsQuery.error}
				fallback={m.products_load_failed()}
				onretry={() => productsQuery.refetch()}
			/>
		</div>
	{:else if productsQuery.isPending}
		<div
			class="grid grid-cols-2 gap-4 md:grid-cols-3 xl:grid-cols-4"
			role="status"
			aria-label={m.common_loading()}
		>
			{#each Array.from({ length: 8 }, (_, i) => i) as i (i)}
				<div class="overflow-hidden card">
					<div class="aspect-[4/3] animate-pulse bg-zinc-100"></div>
					<div class="space-y-2 p-3.5">
						<div class="h-3 w-1/3 rounded bg-zinc-100"></div>
						<div class="h-3.5 w-3/4 rounded bg-zinc-100"></div>
					</div>
				</div>
			{/each}
		</div>
	{:else if products.length === 0}
		<div class="card">
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
		</div>
	{:else if view === 'grid'}
		<ul
			class={[
				'grid grid-cols-2 gap-4 transition-opacity md:grid-cols-3 xl:grid-cols-4',
				dimmed && 'opacity-60'
			]}
		>
			{#each products as product (product.id)}
				<li class="flex"><ProductCard {product} /></li>
			{/each}
		</ul>
	{:else}
		<div class={['overflow-hidden card transition-opacity', dimmed && 'opacity-60']}>
			<div class="relative overflow-x-auto">
				<table class="w-full min-w-[760px] text-left text-sm">
					<thead class="border-b border-zinc-200 bg-zinc-50/80">
						<tr>
							<th scope="col" class="th pl-5">{m.products_col_product()}</th>
							<th scope="col" class="th">{m.products_col_status()}</th>
							<th scope="col" class="th">{m.products_col_stock()}</th>
							<th scope="col" class="th">{m.products_col_price()}</th>
							<th scope="col" class="th">{m.products_col_categories()}</th>
							<th scope="col" class="th">{m.products_col_created()}</th>
							<th scope="col" class="th pr-5"><span class="sr-only">{m.common_actions()}</span></th>
						</tr>
					</thead>
					<tbody class="divide-y divide-zinc-100">
						{#each products as product (product.id)}
							{@const siteUrl =
								product.status === 'published' ? storefrontProductUrl(product.slug) : null}
							<tr
								class="group cursor-pointer transition-colors hover:bg-zinc-50"
								onclick={(e) => openRow(e, product)}
							>
								<td class="td pl-5">
									<div class="flex items-center gap-3">
										<Thumb src={imageUrl(product.images[0], 96)} class="size-11" />
										<div class="min-w-0">
											<a
												class="flex max-w-72 items-center gap-1.5 font-medium text-zinc-900 group-hover:text-brand-700"
												href={resolve(`/products/${product.id}`)}
											>
												<span class="truncate">{localized(product.name)}</span>
												{#if product.featured}
													<Icon
														name="star"
														size={13}
														filled
														class="shrink-0 text-amber-400"
														label={m.products_featured()}
													/>
												{/if}
											</a>
											<span class="block truncate font-mono text-xs text-zinc-400"
												>{product.slug}</span
											>
										</div>
									</div>
								</td>
								<td class="td"><StatusBadge kind="product" value={product.status} /></td>
								<td class="td"><StockLabel stock={product.stock} /></td>
								<td class="td whitespace-nowrap">
									<PriceLabel
										finalCents={product.finalPriceCents}
										baseCents={product.priceCents}
										discountPercent={product.discountPercent}
									/>
								</td>
								<td class="td"><CategoryBadges categories={product.categories} /></td>
								<td class="td whitespace-nowrap text-zinc-500">{formatDate(product.createdAt)}</td>
								<td class="td pr-5">
									<div
										class="flex justify-end gap-0.5 opacity-0 transition-opacity group-focus-within:opacity-100 group-hover:opacity-100"
									>
										{#if siteUrl}
											<Tip text={m.products_view_on_site()}>
												{#snippet children({ props })}
													<!-- eslint-disable svelte/no-navigation-without-resolve -- external storefront URL -->
													<a
														{...props}
														class="btn-icon btn-ghost size-8"
														href={siteUrl}
														target="_blank"
														rel="noopener noreferrer"
														aria-label={m.products_view_on_site()}
													>
														<Icon name="external" size={16} />
													</a>
													<!-- eslint-enable svelte/no-navigation-without-resolve -->
												{/snippet}
											</Tip>
										{/if}
										{#if session.can('products:delete')}
											<Tip text={m.products_delete_tooltip()}>
												{#snippet children({ props })}
													<button
														{...props}
														type="button"
														class="btn-icon btn-ghost size-8 hover:bg-rose-50 hover:text-rose-600"
														aria-label={m.products_delete_tooltip()}
														onclick={() => remove(product)}
													>
														<Icon name="trash" size={16} />
													</button>
												{/snippet}
											</Tip>
										{/if}
									</div>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>
	{/if}

	{#if meta && !showError && products.length > 0}
		<div class="mt-5 card [&>nav]:border-t-0">
			<Pagination
				{meta}
				pageSizes={[...PAGE_SIZES]}
				onpage={(p) => navigate({ page: p })}
				onpagesize={(size) => navigate({ pageSize: size })}
			/>
		</div>
	{/if}
</div>
