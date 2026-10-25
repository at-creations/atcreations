<script lang="ts">
	import Check from '@lucide/svelte/icons/check';
	import XCircle from '@lucide/svelte/icons/circle-x';
	import Mail from '@lucide/svelte/icons/mail';
	import MapPin from '@lucide/svelte/icons/map-pin';
	import { untrack } from 'svelte';
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
	const inStock = $derived(product.inStock);
	const address = $derived(data.storeInfo?.address?.trim() || '');
	const images = $derived(
		product.images.map((image) => ({
			url: image.url,
			alt: t(image.alt, locale) || name,
			width: image.width,
			height: image.height,
			variants: image.variants
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
	// Depends on the id only, so new data for the same product (e.g. a locale switch) doesn't count again.
	const productId = $derived(product.id);
	$effect(() => {
		const endpoint = `${untrack(() => data.config.publicApiUrl)}/v1/products/${encodeURIComponent(productId)}/views`;
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
		inStock: product.inStock
	}}
/>
<JsonLd data={jsonLd} />

<div class="container-page pt-24 pb-20 md:pt-28">
	<Breadcrumbs
		items={[
			{ label: m.nav_home(), href: localizedPath(locale) },
			{ label: m.nav_products(), href: productsHref(locale) },
			{ label: name }
		]}
	/>

	<article class="grid grid-cols-1 gap-10 lg:grid-cols-12 lg:gap-16">
		<div class="lg:col-span-7">
			{#key product.id}
				<Gallery {images} label={name} />
			{/key}
		</div>

		<div class="lg:col-span-5 lg:pt-2">
			{#if product.categories.length > 0}
				<ul class="flex flex-wrap gap-x-4 gap-y-1">
					{#each product.categories as category (category.id)}
						<li>
							<a
								href={productsHref(locale, { category: category.slug })}
								class="text-ink-soft hover:text-ink hover:underline"
							>
								{t(category.name, locale)}
							</a>
						</li>
					{/each}
				</ul>
			{/if}
			<h1 class="mt-2 heading-page">{name}</h1>
			<ProductBadges {product} class="mt-4" />

			<div class="mt-6">
				<Price {product} {locale} size="lg" showSavings />
			</div>

			<p class="mt-4 flex items-center gap-2 font-medium text-ink">
				{#if inStock}
					<Check class="size-5 text-teak" />{m.product_in_stock()}
				{:else}
					<XCircle class="size-5 text-ink-soft" />{m.product_out_of_stock()}
				{/if}
			</p>

			{#if description}
				<section class="mt-8 border-t border-line pt-6" aria-labelledby="description-title">
					<h2 id="description-title" class="sr-only">{m.product_description()}</h2>
					<p class="max-w-prose leading-relaxed whitespace-pre-line">{description}</p>
				</section>
			{/if}

			<div class="mt-8 rounded-lg bg-wall p-5">
				{#if address}
					<p class="flex items-start gap-2.5 text-ink">
						<MapPin class="mt-0.5 size-5 shrink-0 text-teak" />
						{m.product_visit_note({ address })}
					</p>
				{/if}
				<a href={localizedPath(locale, '/contact')} class={['btn-primary', address && 'mt-4']}>
					<Mail class="size-5" />
					{m.product_contact_cta()}
				</a>
			</div>
		</div>
	</article>
</div>
