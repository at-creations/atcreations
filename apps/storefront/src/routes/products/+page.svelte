<script lang="ts">
	import { sized } from '$lib/images';
	import { formatTotal } from '$lib/format';
	import { m } from '$lib/paraglide/messages';
	import { t } from '$lib/i18n';
	import { productsHref, withQuery } from '$lib/urls';
	import Seo from '$lib/components/Seo.svelte';
	import SectionTitle from '$lib/components/SectionTitle.svelte';
	import CategoryPills from '$lib/components/CategoryPills.svelte';
	import ProductCard from '$lib/components/ProductCard.svelte';
	import Pagination from '$lib/components/Pagination.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const locale = $derived(data.locale);
	const selected = $derived(data.selectedCategory);
	const selectedName = $derived(selected ? t(selected.name, locale) : '');
	const selectedDescription = $derived(selected ? t(selected.description, locale) : '');

	const heading = $derived(selected ? selectedName : m.products_title());
	const baseTitle = $derived(
		selected ? m.products_category_meta_title({ category: selectedName }) : m.products_title()
	);
	const title = $derived(
		data.page > 1 ? `${baseTitle} – ${m.pagination_page({ page: data.page })}` : baseTitle
	);
	const description = $derived(
		selectedDescription ||
			(selected
				? m.products_category_meta_description({ category: selectedName })
				: m.products_meta_description())
	);

	// Canonical keeps the category and page (each is a distinct, indexable listing).
	const canonicalQuery = $derived(
		withQuery('', { category: selected?.slug, page: data.page > 1 ? data.page : undefined })
	);
</script>

<Seo
	{title}
	{description}
	image={selected?.image?.url}
	imageAlt={selected ? selectedName : undefined}
	imageWidth={selected?.image?.width}
	imageHeight={selected?.image?.height}
	query={canonicalQuery}
	noindex={data.products.length === 0}
/>

<div class="container-page pt-28 pb-20">
	<SectionTitle level={1} title={heading} />

	<CategoryPills categories={data.categories} selected={selected?.slug ?? null} {locale} />

	{#if selected && (selected.image || selectedDescription)}
		<div class="mt-10 flex max-w-3xl flex-col gap-5 sm:flex-row sm:items-center">
			{#if selected.image}
				<img
					src={sized(selected.image, 256)}
					alt=""
					width={selected.image.width}
					height={selected.image.height}
					class="size-24 shrink-0 rounded-lg object-cover"
				/>
			{/if}
			{#if selectedDescription}
				<p class="whitespace-pre-line text-ink-soft">{selectedDescription}</p>
			{/if}
		</div>
	{/if}

	<p class="mt-10 mb-8 text-sm text-ink-soft" aria-live="polite">
		{m.products_showing({ shown: data.products.length, total: formatTotal(data.meta, locale) })}
	</p>

	{#if data.products.length === 0}
		<div class="panel p-12">
			<p class="text-lg font-medium text-ink">{m.products_no_products()}</p>
			<p class="mt-2 text-ink-soft">{m.products_try_other()}</p>
		</div>
	{:else}
		<ul class="grid grid-cols-2 gap-x-6 gap-y-12 lg:grid-cols-3 xl:grid-cols-4">
			{#each data.products as product, i (product.id)}
				<li><ProductCard {product} {locale} eager={i < 4} /></li>
			{/each}
		</ul>
	{/if}

	<Pagination
		current={data.meta.page}
		totalPages={data.meta.totalPages}
		href={(p) => productsHref(locale, { category: selected?.slug, page: p })}
	/>
</div>
