<script lang="ts">
	import { page } from '$app/state';
	import { m } from '$lib/paraglide/messages';
	import { LOCALES, localizedPath, ogLocale, stripLocale, type Locale } from '$lib/i18n';
	import { truncate } from '$lib/seo';
	import { assetUrl } from '$lib/urls';

	interface Props {
		/** Page title; " | AT Creations" is appended unless `rawTitle` is set. */
		title: string;
		description: string;
		rawTitle?: boolean;
		/** Absolute image URL; defaults to the site OG image. */
		image?: string;
		imageAlt?: string;
		/** Pixel size of `image`, so crawlers can render previews without fetching it first. */
		imageWidth?: number;
		imageHeight?: number;
		/** Product price/availability for `product:*` tags (with `type="product"`). */
		product?: { priceCents: number | null; currency: string; inStock: boolean };
		type?: 'website' | 'product' | 'article';
		/** Query string kept in canonical/alternate URLs (e.g. `?category=pottery`). */
		query?: string;
		noindex?: boolean;
	}

	let {
		title,
		description,
		rawTitle = false,
		image,
		imageAlt,
		imageWidth,
		imageHeight,
		product,
		type = 'website',
		query = '',
		noindex = false
	}: Props = $props();

	const locale = $derived(page.data.locale);
	const siteUrl = $derived(page.data.config.siteUrl);
	const path = $derived(stripLocale(page.url.pathname));
	const urlFor = (l: Locale) => `${siteUrl}${localizedPath(l, path)}${query}`;
	const canonical = $derived(urlFor(locale));
	const fullTitle = $derived(rawTitle ? title : `${title} | ${m.site_name()}`);
	const desc = $derived(truncate(description));
	const ogImage = $derived(image ?? assetUrl(page.data.config.assetBaseUrl, 'og-image.png'));
	// The site-wide share image (og-image.png) is 1337×700.
	const ogWidth = $derived(image ? imageWidth : 1337);
	const ogHeight = $derived(image ? imageHeight : 700);
	const ogImageType = $derived(imageType(ogImage));
	const ogAlt = $derived(imageAlt ?? m.og_image_alt());

	function imageType(url: string): string | undefined {
		const ext = new URL(url).pathname.split('.').pop()?.toLowerCase();
		if (ext === 'png') return 'image/png';
		if (ext === 'jpg' || ext === 'jpeg') return 'image/jpeg';
		if (ext === 'webp') return 'image/webp';
		return undefined;
	}
</script>

<svelte:head>
	<title>{fullTitle}</title>
	<meta name="description" content={desc} />
	{#if noindex}<meta name="robots" content="noindex, follow" />{/if}
	<link rel="canonical" href={canonical} />
	{#each LOCALES as l (l)}
		<link rel="alternate" hreflang={l} href={urlFor(l)} />
	{/each}
	<link rel="alternate" hreflang="x-default" href={urlFor('en')} />

	<meta property="og:site_name" content={m.site_name()} />
	<meta property="og:type" content={type} />
	<meta property="og:title" content={fullTitle} />
	<meta property="og:description" content={desc} />
	<meta property="og:url" content={canonical} />
	<meta property="og:image" content={ogImage} />
	{#if ogImage.startsWith('https://')}
		<meta property="og:image:secure_url" content={ogImage} />
	{/if}
	{#if ogImageType}<meta property="og:image:type" content={ogImageType} />{/if}
	{#if ogWidth && ogHeight}
		<meta property="og:image:width" content={String(ogWidth)} />
		<meta property="og:image:height" content={String(ogHeight)} />
	{/if}
	<meta property="og:image:alt" content={ogAlt} />
	<meta property="og:locale" content={ogLocale(locale)} />
	{#each LOCALES.filter((l) => l !== locale) as l (l)}
		<meta property="og:locale:alternate" content={ogLocale(l)} />
	{/each}

	<meta name="twitter:card" content="summary_large_image" />
	<meta name="twitter:title" content={fullTitle} />
	<meta name="twitter:description" content={desc} />
	<meta name="twitter:image" content={ogImage} />
	<meta name="twitter:image:alt" content={ogAlt} />

	{#if product}
		{#if product.priceCents !== null}
			<meta property="product:price:amount" content={(product.priceCents / 100).toFixed(2)} />
			<meta property="product:price:currency" content={product.currency} />
		{/if}
		<meta property="product:availability" content={product.inStock ? 'in stock' : 'out of stock'} />
	{/if}
</svelte:head>
