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
	<meta property="og:image:alt" content={imageAlt ?? m.og_image_alt()} />
	<meta property="og:locale" content={ogLocale(locale)} />
	{#each LOCALES.filter((l) => l !== locale) as l (l)}
		<meta property="og:locale:alternate" content={ogLocale(l)} />
	{/each}

	<meta name="twitter:card" content="summary_large_image" />
	<meta name="twitter:title" content={fullTitle} />
	<meta name="twitter:description" content={desc} />
	<meta name="twitter:image" content={ogImage} />
</svelte:head>
