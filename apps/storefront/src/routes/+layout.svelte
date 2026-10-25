<script lang="ts">
	import '../app.css';
	import { m } from '$lib/paraglide/messages';
	import Header from '$lib/components/Header.svelte';
	import Footer from '$lib/components/Footer.svelte';
	import { afterNavigate } from '$app/navigation';
	import { trackPageView } from '$lib/analytics';
	import type { LayoutProps } from './$types';

	let { data, children }: LayoutProps = $props();

	// One page view per navigation. The external referrer only applies to the first page.
	afterNavigate(({ from, to }) => {
		if (!to?.url || from?.url.pathname === to.url.pathname) return;
		trackPageView(data.config.publicApiUrl, to.url.pathname, from ? '' : document.referrer);
	});
</script>

<a
	href="#main"
	class="sr-only z-50 rounded-md bg-marigold px-4 py-2 font-medium text-white focus:not-sr-only focus:fixed focus:top-2 focus:left-2"
>
	{m.skip_to_content()}
</a>

<Header locale={data.locale} assetBaseUrl={data.config.assetBaseUrl} />

<main id="main" tabindex="-1" class="min-h-[60vh] focus:outline-none">
	{@render children()}
</main>

<Footer locale={data.locale} assetBaseUrl={data.config.assetBaseUrl} storeInfo={data.storeInfo} />
