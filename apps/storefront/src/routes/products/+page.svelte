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

	// Canonical keeps the category and page (each is a distinct, indexable listing).
	const canonicalQuery = $derived(
		withQuery('', { category: selected?.slug, page: data.page > 1 ? data.page : undefined })
	);
</script>

<Seo
	title={selected ? m.products_category_meta_title({ category: selectedName }) : m.products_title()}
	description={selectedDescription || m.products_meta_description()}
	image={selected?.image?.url}
	query={canonicalQuery}
/>

<div class="container-page pt-28 pb-20">
	<SectionTitle level={1} title={m.products_title()} />

	<CategoryPills categories={data.categories} selected={selected?.slug ?? null} {locale} />

	{#if selected}
		<section
			class="mt-10 flex max-w-3xl flex-col gap-5 sm:flex-row sm:items-center"
			aria-labelledby="category-title"
		>
			{#if selected.image}
				<img
					src={sized(selected.image, 256)}
					alt=""
					width={selected.image.width}
					height={selected.image.height}
					class="size-24 shrink-0 rounded-lg object-cover"
				/>
			{/if}
			<div>
				<h2 id="category-title" class="heading-section">{selectedName}</h2>
				{#if selectedDescription}
					<p class="mt-2 whitespace-pre-line text-ink-soft">{selectedDescription}</p>
				{/if}
			</div>
		</section>
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
