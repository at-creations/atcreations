<script lang="ts">
	import Check from '@lucide/svelte/icons/check';
	import XCircle from '@lucide/svelte/icons/circle-x';
	import Mail from '@lucide/svelte/icons/mail';
	import { m } from '$lib/paraglide/messages';
	import { localizedPath, t } from '$lib/i18n';
	import { productsHref } from '$lib/urls';
	import { truncate } from '$lib/seo';
	import Seo from '$lib/components/Seo.svelte';
	import JsonLd from '$lib/components/JsonLd.svelte';
	import Breadcrumbs from '$lib/components/Breadcrumbs.svelte';
	import Gallery from '$lib/components/Gallery.svelte';
	import Price from '$lib/components/Price.svelte';
	import ProductBadges from '$lib/components/ProductBadges.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const locale = $derived(data.locale);
	const product = $derived(data.product);
	const name = $derived(t(product.name, locale));
	const description = $derived(t(product.description, locale));
	const inStock = $derived(product.stock > 0);
	const images = $derived(
		product.images.map((image) => ({
			url: image.url,
			alt: t(image.alt, locale) || name,
			width: image.width,
			height: image.height
		}))
	);

	const jsonLd = $derived.by(() => {
		const url = `${data.config.siteUrl}${localizedPath(locale, `/products/${product.slug}`)}`;
		const ld: Record<string, unknown> = {
			'@context': 'https://schema.org',
			'@type': 'Product',
			name,
			description: truncate(description, 5000),
			sku: product.id,
			url,
			image: product.images.map((i) => i.url),
			brand: { '@type': 'Brand', name: 'AT Creations' }
		};
		if (product.categories[0]) ld.category = t(product.categories[0].name, locale);
		if (product.finalPriceCents !== null) {
			ld.offers = {
				'@type': 'Offer',
				url,
				priceCurrency: product.currency,
				price: (product.finalPriceCents / 100).toFixed(2),
				availability: inStock ? 'https://schema.org/InStock' : 'https://schema.org/OutOfStock',
				itemCondition: 'https://schema.org/NewCondition'
			};
		}
		return ld;
	});

	// Fire-and-forget view counter, once per product shown (also on client-side navigation
	// between products). Errors (network, 404, 429) are deliberately ignored.
	$effect(() => {
		const endpoint = `${data.config.publicApiUrl}/v1/products/${encodeURIComponent(product.id)}/views`;
		fetch(endpoint, { method: 'POST', credentials: 'omit', keepalive: true, mode: 'cors' }).catch(
			() => {}
		);
	});
</script>

<Seo
	title={name}
	description={description || m.products_meta_description()}
	image={product.images[0]?.url}
	imageAlt={name}
	imageWidth={product.images[0]?.width}
	imageHeight={product.images[0]?.height}
	type="product"
	product={{
		priceCents: product.finalPriceCents,
		currency: product.currency,
		inStock: product.stock > 0
	}}
/>
<JsonLd data={jsonLd} />

<div class="container-page pt-24 pb-20">
	<Breadcrumbs
		items={[
			{ label: m.nav_home(), href: localizedPath(locale) },
			{ label: m.nav_products(), href: productsHref(locale) },
			{ label: name }
		]}
	/>

	<article class="grid grid-cols-1 gap-10 lg:grid-cols-2 lg:gap-14">
		{#key product.id}
			<Gallery {images} label={name} />
		{/key}

		<div class="animate-fade-in">
			<ProductBadges {product} class="mb-4" />
			<h1 class="text-3xl font-bold tracking-tight text-gray-900 md:text-4xl">{name}</h1>

			<div class="mt-6">
				<Price {product} {locale} size="lg" showSavings />
			</div>

			<p class="mt-6 flex items-center gap-2 font-medium">
				<span class="text-gray-600">{m.product_availability()}:</span>
				{#if inStock}
					<span class="flex items-center gap-1 text-emerald-700">
						<Check class="size-5" />{m.product_in_stock()}
					</span>
				{:else}
					<span class="flex items-center gap-1 text-gray-700">
						<XCircle class="size-5" />{m.product_out_of_stock()}
					</span>
				{/if}
			</p>

			{#if product.categories.length > 0}
				<div class="mt-6">
					<h2 class="mb-2 text-sm font-semibold tracking-wide text-gray-600 uppercase">
						{m.product_categories()}
					</h2>
					<ul class="flex flex-wrap gap-2">
						{#each product.categories as category (category.id)}
							<li>
								<a
									href={productsHref(locale, { category: category.slug })}
									class="inline-block rounded-full bg-blue-50 px-3 py-1 text-sm font-medium text-blue-700 hover:bg-blue-100"
								>
									{t(category.name, locale)}
								</a>
							</li>
						{/each}
					</ul>
				</div>
			{/if}

			{#if description}
				<section class="mt-8 border-t border-gray-200 pt-6" aria-labelledby="description-title">
					<h2 id="description-title" class="mb-3 text-xl font-semibold text-gray-900">
						{m.product_description()}
					</h2>
					<p class="leading-relaxed whitespace-pre-line text-gray-700">{description}</p>
				</section>
			{/if}

			<a href={localizedPath(locale, '/contact')} class="mt-8 btn-primary">
				<Mail class="size-5" />
				{m.product_contact_cta()}
			</a>
		</div>
	</article>
</div>
