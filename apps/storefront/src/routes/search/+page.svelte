<script lang="ts">
	import { formatTotal } from '$lib/format';
	import { untrack } from 'svelte';
	import { goto } from '$app/navigation';
	import { navigating, page } from '$app/state';
	import SearchIcon from '@lucide/svelte/icons/search';
	import { m } from '$lib/paraglide/messages';
	import { localizedPath, t } from '$lib/i18n';
	import {
		PRICE_RANGE_KEYS,
		SORTS,
		parsePriceRange,
		parseQuery,
		parseSort,
		searchStateToParams,
		type PriceRange,
		type SearchState,
		type Sort
	} from '$lib/search';
	import Seo from '$lib/components/Seo.svelte';
	import SectionTitle from '$lib/components/SectionTitle.svelte';
	import ProductCard from '$lib/components/ProductCard.svelte';
	import ProductCardSkeleton from '$lib/components/ProductCardSkeleton.svelte';
	import Pagination from '$lib/components/Pagination.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const locale = $derived(data.locale);
	const action = $derived(localizedPath(locale, '/search'));

	const SORT_LABELS: Record<Sort, () => string> = {
		relevance: m.sort_relevance,
		newest: m.sort_newest,
		price_asc: m.sort_price_asc,
		price_desc: m.sort_price_desc,
		name_asc: m.sort_name_asc,
		name_desc: m.sort_name_desc
	};
	const PRICE_LABELS: Record<PriceRange, () => string> = {
		all: m.price_all,
		'0-50': m.price_0_50,
		'50-100': m.price_50_100,
		'100-250': m.price_100_250,
		'250-up': m.price_250_up
	};

	// Local text so typing isn't disturbed by URL round-trips; re-synced when the URL
	// changes from elsewhere (back/forward, links).
	let q = $state(untrack(() => data.state.q));
	$effect(() => {
		const incoming = data.state.q;
		if (parseQuery(untrack(() => q)) !== incoming) q = incoming;
	});

	let timer: ReturnType<typeof setTimeout> | undefined;
	$effect(() => () => clearTimeout(timer));

	const isLoading = $derived(navigating.to?.url.pathname === page.url.pathname);

	function navigate(patch: Partial<SearchState>) {
		clearTimeout(timer);
		const next: SearchState = { ...data.state, page: 1, ...patch };
		const qs = searchStateToParams(next).toString();
		goto(qs ? `${action}?${qs}` : action, { replaceState: true, keepFocus: true, noScroll: true });
	}

	function onInput(event: Event & { currentTarget: HTMLInputElement }) {
		q = event.currentTarget.value;
		clearTimeout(timer);
		timer = setTimeout(() => {
			const term = parseQuery(q);
			if (term !== data.state.q) navigate({ q: term });
		}, 300);
	}

	function onSubmit(event: SubmitEvent) {
		event.preventDefault();
		navigate({ q: parseQuery(q) });
	}

	function pageHref(p: number): string {
		const qs = searchStateToParams({ ...data.state, page: p }).toString();
		return qs ? `${action}?${qs}` : action;
	}

	const hasFilters = $derived(
		data.state.q !== '' ||
			data.state.category !== '' ||
			data.state.sort !== 'relevance' ||
			data.state.price !== 'all'
	);
</script>

<Seo title={m.search_title()} description={m.search_meta_description()} noindex={hasFilters} />

<div class="container-page pt-28 pb-20">
	<SectionTitle level={1} title={m.search_title()} />

	<form
		method="GET"
		{action}
		role="search"
		class="mx-auto max-w-5xl card p-5 md:p-6"
		onsubmit={onSubmit}
	>
		<label for="search-q" class="mb-1.5 block text-sm font-medium text-gray-800">
			{m.search_label()}
		</label>
		<div class="flex gap-2">
			<div class="relative flex-1">
				<SearchIcon
					class="pointer-events-none absolute top-1/2 left-3 size-5 -translate-y-1/2 text-gray-500"
				/>
				<input
					id="search-q"
					name="q"
					type="search"
					value={q}
					oninput={onInput}
					maxlength="100"
					autocomplete="off"
					placeholder={m.search_placeholder()}
					class="field pl-10"
				/>
			</div>
			<button type="submit" class="btn-primary">{m.search_button()}</button>
		</div>

		<fieldset class="mt-5 grid grid-cols-1 gap-4 sm:grid-cols-3">
			<legend class="sr-only">{m.search_filters()}</legend>
			<div>
				<label for="search-category" class="mb-1.5 block text-sm font-medium text-gray-800">
					{m.search_category()}
				</label>
				<select
					id="search-category"
					name="category"
					class="field"
					value={data.state.category}
					onchange={(e) => navigate({ category: e.currentTarget.value })}
				>
					<option value="">{m.search_all_categories()}</option>
					{#each data.categories as category (category.id)}
						<option value={category.slug}>{t(category.name, locale)}</option>
					{/each}
				</select>
			</div>
			<div>
				<label for="search-sort" class="mb-1.5 block text-sm font-medium text-gray-800">
					{m.search_sort()}
				</label>
				<select
					id="search-sort"
					name="sort"
					class="field"
					value={data.state.sort}
					onchange={(e) => navigate({ sort: parseSort(e.currentTarget.value) })}
				>
					{#each SORTS as sort (sort)}
						<option value={sort}>{SORT_LABELS[sort]()}</option>
					{/each}
				</select>
			</div>
			<div>
				<label for="search-price" class="mb-1.5 block text-sm font-medium text-gray-800">
					{m.search_price()}
				</label>
				<select
					id="search-price"
					name="price"
					class="field"
					value={data.state.price}
					onchange={(e) => navigate({ price: parsePriceRange(e.currentTarget.value) })}
				>
					{#each PRICE_RANGE_KEYS as range (range)}
						<option value={range}>{PRICE_LABELS[range]()}</option>
					{/each}
				</select>
			</div>
		</fieldset>

		{#if hasFilters}
			<p class="mt-4 text-right">
				<a href={action} class="text-sm font-medium text-blue-700 hover:underline">
					{m.search_clear()}
				</a>
			</p>
		{/if}
	</form>

	<p class="mt-10 mb-6 text-sm text-gray-600" aria-live="polite">
		{m.search_results({ shown: data.products.length, total: formatTotal(data.meta, locale) })}
	</p>

	{#if isLoading}
		<ul
			class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4"
			aria-busy="true"
		>
			{#each { length: 8 }, i (i)}
				<li><ProductCardSkeleton /></li>
			{/each}
		</ul>
	{:else if data.products.length === 0}
		<div class="card p-12 text-center">
			<p class="text-lg font-medium text-gray-900">{m.search_no_results()}</p>
			<p class="mt-2 text-gray-600">{m.search_try_adjusting()}</p>
		</div>
	{:else}
		<ul class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
			{#each data.products as product (product.id)}
				<li><ProductCard {product} {locale} /></li>
			{/each}
		</ul>
	{/if}

	<Pagination current={data.meta.page} totalPages={data.meta.totalPages} href={pageHref} />
</div>
