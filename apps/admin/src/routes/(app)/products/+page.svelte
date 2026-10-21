<script lang="ts">
	import type { Category, PageMeta, Product, ProductStatus } from '@atcreations/api-types';
	import { untrack } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { api, unwrap } from '$lib/api';
	import CategoryBadges from '$lib/components/CategoryBadges.svelte';
	import Icon from '$lib/components/Icon.svelte';
	import LoadState from '$lib/components/LoadState.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Pagination from '$lib/components/Pagination.svelte';
	import StatusBadge from '$lib/components/StatusBadge.svelte';
	import { formatDate, localized, priceErrorMessage, productStatusLabel } from '$lib/i18n';
	import { safeImageUrl, storefrontProductUrl } from '$lib/links';
	import { toastError } from '$lib/notify';
	import { m } from '$lib/paraglide/messages.js';
	import { confirmAction } from '$lib/state/confirm.svelte';
	import { getAdminSession } from '$lib/state/session.svelte';
	import { toast } from '$lib/state/toast.svelte';
	import { formatCAD, parseDollars } from '$lib/utils/money';
	import { readEnum, readInt, readString, withParams } from '$lib/utils/query';

	const session = getAdminSession();

	const STATUSES: readonly ProductStatus[] = ['draft', 'published', 'archived'];
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

	// --- URL query = source of truth ---
	const filters = $derived.by(() => {
		const p = page.url.searchParams;
		const min = parseDollars(readString(p, 'minPrice', 20));
		const max = parseDollars(readString(p, 'maxPrice', 20));
		const category = readString(p, 'category', 40);
		return {
			page: readInt(p, 'page', 1),
			pageSize: readInt(p, 'pageSize', 10, PAGE_SIZES),
			q: readString(p, 'q'),
			status: readEnum(p, 'status', STATUSES),
			category: category === UNCATEGORIZED || /^[a-f0-9]{24}$/.test(category) ? category : '',
			featured: readEnum(p, 'featured', ['true', 'false'] as const),
			minPrice: min.ok ? min.cents : null,
			maxPrice: max.ok ? max.cents : null,
			sort: readEnum(p, 'sort', SORTS, 'newest') ?? 'newest'
		};
	});

	let products = $state<Product[]>([]);
	let meta = $state<PageMeta | null>(null);
	let loading = $state(true);
	let loadError = $state<unknown>(null);
	let categories = $state<Category[]>([]);
	let seq = 0;

	async function load(f: typeof filters) {
		const current = ++seq;
		loading = true;
		loadError = null;
		try {
			const res = await unwrap(
				api().GET('/v1/admin/products', {
					params: {
						query: {
							page: f.page,
							pageSize: f.pageSize,
							q: f.q || undefined,
							status: f.status,
							category: f.category && f.category !== UNCATEGORIZED ? f.category : undefined,
							uncategorized: f.category === UNCATEGORIZED ? true : undefined,
							featured: f.featured === undefined ? undefined : f.featured === 'true',
							minPrice: f.minPrice ?? undefined,
							maxPrice: f.maxPrice ?? undefined,
							sort: f.sort
						}
					}
				})
			);
			if (current !== seq) return;
			products = res.data;
			meta = res.meta;
		} catch (error) {
			if (current === seq) loadError = error;
		} finally {
			if (current === seq) loading = false;
		}
	}

	$effect(() => {
		const f = filters;
		untrack(() => load(f));
	});

	$effect(() => {
		unwrap(api().GET('/v1/admin/categories')).then(
			(res) => (categories = res.data),
			() => (categories = [])
		);
	});

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

	// --- Price range (dollars in the UI and URL, cents for the API) ---
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

	const hasFilters = $derived(
		!!(
			filters.q ||
			filters.status ||
			filters.category ||
			filters.featured ||
			filters.minPrice !== null ||
			filters.maxPrice !== null
		)
	);

	function clearFilters() {
		minError = maxError = undefined;
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
			await unwrap(
				api().DELETE('/v1/admin/products/{id}', { params: { path: { id: product.id } } })
			);
			toast.success(m.products_deleted());
			if (products.length === 1 && filters.page > 1) navigate({ page: filters.page - 1 });
			else load(filters);
		} catch (error) {
			toastError(error, m.products_delete_failed());
		}
	}

	const uid = $props.id();
</script>

{#snippet price(product: Product)}
	{#if product.finalPriceCents === null}
		<span class="text-sm text-slate-500 italic">{m.products_contact_price()}</span>
	{:else}
		<span class="font-medium">{formatCAD(product.finalPriceCents)}</span>
		{#if product.discountPercent > 0 && product.priceCents !== null}
			<span class="ml-1 text-xs text-slate-400 line-through">{formatCAD(product.priceCents)}</span>
			<span class="ml-1 badge bg-red-100 text-red-700"
				>{m.products_discount({ percent: product.discountPercent })}</span
			>
		{/if}
	{/if}
{/snippet}

{#snippet thumb(product: Product, size: string)}
	{@const cover = safeImageUrl(product.images[0]?.url)}
	{#if cover}
		<img src={cover} alt="" class={[size, 'rounded-lg object-cover']} loading="lazy" />
	{:else}
		<span
			class={[
				size,
				'flex items-center justify-center rounded-lg bg-slate-100 text-[10px] text-slate-400'
			]}
		>
			<Icon name="image" size={16} label={m.products_no_image()} />
		</span>
	{/if}
{/snippet}

{#snippet rowActions(product: Product)}
	{@const siteUrl = product.status === 'published' ? storefrontProductUrl(product.slug) : null}
	<div class="flex items-center justify-end gap-1">
		<a
			class="btn-ghost btn-sm"
			href={resolve(`/products/${product.id}`)}
			title={m.products_edit_tooltip()}
		>
			<Icon name="edit" size={16} />
			<span class="sr-only sm:not-sr-only">{m.common_edit()}</span>
		</a>
		{#if siteUrl}
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- external storefront URL -->
			<a class="btn-ghost btn-sm" href={siteUrl} target="_blank" rel="noopener noreferrer">
				<Icon name="external" size={16} />
				<span class="sr-only">{m.products_view_on_site()}</span>
			</a>
		{/if}
		{#if session.can('products:delete')}
			<button
				type="button"
				class="btn-ghost btn-sm text-red-600 hover:bg-red-50 hover:text-red-700"
				title={m.products_delete_tooltip()}
				onclick={() => remove(product)}
			>
				<Icon name="trash" size={16} />
				<span class="sr-only">{m.common_delete()}</span>
			</button>
		{/if}
	</div>
{/snippet}

<PageHeader title={m.products_title()}>
	{#snippet actions()}
		{#if session.can('products:write')}
			<a class="btn-primary" href={resolve('/products/new')}
				><Icon name="plus" />{m.products_add()}</a
			>
		{/if}
	{/snippet}
</PageHeader>

<section class="mb-4 card p-4" aria-label={m.common_filters()}>
	<div class="grid gap-3 md:grid-cols-2 xl:grid-cols-5">
		<div class="relative md:col-span-2">
			<label for="{uid}-q" class="sr-only">{m.common_search()}</label>
			<Icon
				name="search"
				size={16}
				class="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-slate-400"
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
		<div>
			<label for="{uid}-status" class="sr-only">{m.products_filter_status()}</label>
			<select
				id="{uid}-status"
				class="input"
				value={filters.status ?? ''}
				onchange={(e) => navigate({ status: e.currentTarget.value })}
			>
				<option value="">{m.products_filter_status()}: {m.common_all()}</option>
				{#each STATUSES as status (status)}
					<option value={status}>{productStatusLabel(status)}</option>
				{/each}
			</select>
		</div>
		<div>
			<label for="{uid}-category" class="sr-only">{m.products_filter_category()}</label>
			<select
				id="{uid}-category"
				class="input"
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
		<div>
			<label for="{uid}-featured" class="sr-only">{m.products_filter_featured()}</label>
			<select
				id="{uid}-featured"
				class="input"
				value={filters.featured ?? ''}
				onchange={(e) => navigate({ featured: e.currentTarget.value })}
			>
				<option value="">{m.products_filter_featured()}: {m.common_all()}</option>
				<option value="true">{m.products_featured_only()}</option>
				<option value="false">{m.products_not_featured()}</option>
			</select>
		</div>
	</div>
	<div class="mt-3 flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
		<form class="flex flex-wrap items-start gap-2" onsubmit={applyPrice} novalidate>
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
					{#if minError}<p id="{uid}-min-err" class="mt-1 text-xs text-red-600">{minError}</p>{/if}
				</div>
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
					{#if maxError}<p id="{uid}-max-err" class="mt-1 text-xs text-red-600">{maxError}</p>{/if}
				</div>
				<button type="submit" class="btn-secondary">{m.common_apply()}</button>
			</fieldset>
		</form>
		<div class="flex flex-wrap items-center gap-2">
			<label for="{uid}-sort" class="text-sm text-slate-600">{m.common_sort_by()}</label>
			<select
				id="{uid}-sort"
				class="input w-auto"
				value={filters.sort}
				onchange={(e) =>
					navigate({ sort: e.currentTarget.value === 'newest' ? null : e.currentTarget.value })}
			>
				{#each SORTS as sort (sort)}
					<option value={sort}>{sortLabels[sort]()}</option>
				{/each}
			</select>
			{#if hasFilters}
				<button type="button" class="btn-ghost" onclick={clearFilters}
					>{m.products_clear_filters()}</button
				>
			{/if}
		</div>
	</div>
</section>

<section class="overflow-hidden card" aria-busy={loading}>
	{#if loadError}
		<LoadState
			error={loadError}
			fallback={m.products_load_failed()}
			onretry={() => load(filters)}
		/>
	{:else if loading && products.length === 0}
		<LoadState loading />
	{:else if products.length === 0}
		<p class="p-10 text-center text-slate-500">{m.products_no_results()}</p>
	{:else}
		<!-- Table (≥ xl) -->
		<table class={['hidden w-full text-left text-sm xl:table', loading && 'opacity-60']}>
			<thead class="bg-slate-50 text-xs font-semibold tracking-wide text-slate-500 uppercase">
				<tr>
					<th scope="col" class="px-4 py-3">{m.products_col_product()}</th>
					<th scope="col" class="px-4 py-3">{m.products_col_status()}</th>
					<th scope="col" class="px-4 py-3">{m.products_col_categories()}</th>
					<th scope="col" class="px-4 py-3">{m.products_col_price()}</th>
					<th scope="col" class="px-4 py-3 text-right">{m.products_col_stock()}</th>
					<th scope="col" class="px-4 py-3">{m.products_col_created()}</th>
					<th scope="col" class="px-4 py-3 text-right">{m.common_actions()}</th>
				</tr>
			</thead>
			<tbody class="divide-y divide-slate-100">
				{#each products as product (product.id)}
					<tr class="hover:bg-slate-50">
						<td class="px-4 py-3">
							<div class="flex items-center gap-3">
								{@render thumb(product, 'size-12')}
								<div class="min-w-0">
									<a
										class="block max-w-64 truncate font-medium text-slate-900 hover:underline"
										href={resolve(`/products/${product.id}`)}>{localized(product.name)}</a
									>
									<span class="block font-mono text-xs text-slate-400">{product.slug}</span>
								</div>
								{#if product.featured}
									<Icon
										name="star"
										size={16}
										filled
										class="text-amber-400"
										label={m.products_featured()}
									/>
								{/if}
							</div>
						</td>
						<td class="px-4 py-3"><StatusBadge kind="product" value={product.status} /></td>
						<td class="px-4 py-3"><CategoryBadges categories={product.categories} /></td>
						<td class="px-4 py-3">{@render price(product)}</td>
						<td class={['px-4 py-3 text-right font-medium', product.stock === 0 && 'text-red-600']}
							>{product.stock}</td
						>
						<td class="px-4 py-3 text-slate-600">{formatDate(product.createdAt)}</td>
						<td class="px-4 py-3">{@render rowActions(product)}</td>
					</tr>
				{/each}
			</tbody>
		</table>

		<!-- Cards (< xl) -->
		<ul class={['divide-y divide-slate-100 xl:hidden', loading && 'opacity-60']}>
			{#each products as product (product.id)}
				<li class="flex gap-3 p-4">
					{@render thumb(product, 'size-16')}
					<div class="min-w-0 flex-1 space-y-1">
						<div class="flex items-start gap-2">
							<a
								class="min-w-0 flex-1 truncate font-medium hover:underline"
								href={resolve(`/products/${product.id}`)}>{localized(product.name)}</a
							>
							{#if product.featured}
								<Icon
									name="star"
									size={16}
									filled
									class="text-amber-400"
									label={m.products_featured()}
								/>
							{/if}
						</div>
						<div class="flex flex-wrap items-center gap-2">
							<StatusBadge kind="product" value={product.status} />
							<CategoryBadges categories={product.categories} />
						</div>
						<div class="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm">
							<span>{@render price(product)}</span>
							<span class={['text-slate-600', product.stock === 0 && 'text-red-600']}
								>{m.products_col_stock()}: {product.stock}</span
							>
							<span class="text-xs text-slate-500">{formatDate(product.createdAt)}</span>
						</div>
						{@render rowActions(product)}
					</div>
				</li>
			{/each}
		</ul>
	{/if}
	{#if meta && !loadError}
		<Pagination
			{meta}
			pageSizes={[...PAGE_SIZES]}
			onpage={(p) => navigate({ page: p })}
			onpagesize={(size) => navigate({ pageSize: size })}
		/>
	{/if}
</section>
