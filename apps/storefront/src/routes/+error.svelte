<script lang="ts">
	import { page } from '$app/state';
	import { m } from '$lib/paraglide/messages';
	import { currentLocale } from '$lib/locale';
	import { localizedPath } from '$lib/i18n';
	import { assetUrl } from '$lib/urls';

	const locale = currentLocale();
	const notFound = $derived(page.status === 404);
	const assetBase = $derived(page.data.config?.assetBaseUrl);
</script>

<svelte:head>
	<title>{notFound ? m.error_404_title() : m.error_title()} | {m.site_name()}</title>
	<meta name="robots" content="noindex" />
</svelte:head>

<div
	class="container-page flex min-h-[70vh] flex-col items-center justify-center pt-28 pb-20 text-center"
>
	{#if assetBase}
		<img
			src={assetUrl(assetBase, 'logo-square.png')}
			alt={m.footer_logo_alt()}
			width="96"
			height="96"
			class="mb-8 size-24 rounded-2xl shadow-card"
		/>
	{/if}
	<h1 class="text-3xl font-bold text-gray-900 md:text-4xl">
		{notFound ? m.error_404_title() : m.error_title()}
	</h1>
	{#if notFound}
		<p class="mt-4 text-lg">{m.error_404_message1()}</p>
		<p class="mt-2 text-gray-600">{m.error_404_message2()}</p>
	{:else}
		<p class="mt-4 max-w-xl text-lg">{m.error_message()}</p>
	{/if}
	<div class="mt-10 flex flex-wrap justify-center gap-4">
		<a href={localizedPath(locale)} class="btn-primary">{m.error_home()}</a>
		<a href={localizedPath(locale, '/products')} class="btn-secondary">{m.error_products()}</a>
		{#if !notFound}
			<a href={page.url.pathname + page.url.search} class="btn-secondary" data-sveltekit-reload>
				{m.error_try_again()}
			</a>
		{/if}
	</div>
</div>
