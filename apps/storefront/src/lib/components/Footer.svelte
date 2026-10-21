<script lang="ts">
	import type { StoreInfo } from '@atcreations/api-types';
	import Mail from '@lucide/svelte/icons/mail';
	import MapPin from '@lucide/svelte/icons/map-pin';
	import Phone from '@lucide/svelte/icons/phone';
	import { m } from '$lib/paraglide/messages';
	import { localizedPath, type Locale } from '$lib/i18n';
	import { assetUrl } from '$lib/urls';

	interface Props {
		locale: Locale;
		assetBaseUrl: string;
		storeInfo: StoreInfo | null;
	}

	let { locale, assetBaseUrl, storeInfo }: Props = $props();

	const year = new Date().getFullYear();
	const link = 'text-gray-600 transition-colors hover:text-blue-700';

	const quickLinks = $derived([
		{ path: '/', label: m.nav_home() },
		{ path: '/products', label: m.nav_products() },
		{ path: '/workshops', label: m.nav_workshops() },
		{ path: '/about', label: m.nav_about() },
		{ path: '/contact', label: m.nav_contact() }
	]);
	const legalLinks = $derived([
		{ path: '/privacy', label: m.privacy_link() },
		{ path: '/terms', label: m.terms_link() }
	]);

	const email = $derived(storeInfo?.email?.trim() || '');
	const phone = $derived(storeInfo?.phone?.trim() || '');
	const address = $derived(storeInfo?.address?.trim() || '');
</script>

<footer class="border-t border-gray-200 bg-white">
	<div class="container-page grid grid-cols-1 gap-10 py-12 md:grid-cols-2 md:py-16 lg:grid-cols-4">
		<div class="space-y-4">
			<div class="flex items-center gap-2">
				<img
					src={assetUrl(assetBaseUrl, 'logo-square.png')}
					alt={m.footer_logo_alt()}
					width="40"
					height="40"
					loading="lazy"
					class="size-10 rounded-md"
				/>
				<span class="text-xl font-semibold text-gray-900">{m.site_name()}</span>
			</div>
			<p class="text-sm leading-relaxed text-gray-600 md:max-w-72">{m.footer_description()}</p>
		</div>

		<nav aria-labelledby="footer-quick-links">
			<h2 id="footer-quick-links" class="mb-4 text-lg font-semibold text-gray-900">
				{m.footer_quick_links()}
			</h2>
			<ul class="space-y-3">
				{#each quickLinks as item (item.path)}
					<li><a href={localizedPath(locale, item.path)} class={link}>{item.label}</a></li>
				{/each}
			</ul>
		</nav>

		<nav aria-labelledby="footer-legal">
			<h2 id="footer-legal" class="mb-4 text-lg font-semibold text-gray-900">{m.footer_legal()}</h2>
			<ul class="space-y-3">
				{#each legalLinks as item (item.path)}
					<li><a href={localizedPath(locale, item.path)} class={link}>{item.label}</a></li>
				{/each}
			</ul>
		</nav>

		<div>
			<h2 class="mb-4 text-lg font-semibold text-gray-900">{m.footer_contact()}</h2>
			<address class="space-y-3 not-italic">
				<p class="flex items-start gap-2">
					<MapPin class="mt-0.5 size-[18px] shrink-0 text-blue-700" />
					<span class="text-gray-600">{address || m.contact_to_be_updated()}</span>
				</p>
				<p class="flex items-center gap-2">
					<Phone class="size-[18px] shrink-0 text-blue-700" />
					{#if phone}
						<a href={`tel:${phone.replace(/[^\d+]/g, '')}`} class={link}>{phone}</a>
					{:else}
						<span class="text-gray-600">{m.contact_to_be_updated()}</span>
					{/if}
				</p>
				<p class="flex items-center gap-2">
					<Mail class="size-[18px] shrink-0 text-blue-700" />
					{#if email}
						<a href={`mailto:${email}`} class={link}>{email}</a>
					{:else}
						<span class="text-gray-600">{m.contact_to_be_updated()}</span>
					{/if}
				</p>
			</address>
		</div>
	</div>

	<div class="border-t border-gray-200">
		<div
			class="container-page flex flex-col items-center justify-between gap-4 py-6 text-sm text-gray-600 sm:flex-row"
		>
			<p>&copy; {year} AT Creations. {m.footer_rights()}</p>
			<ul class="flex gap-6">
				{#each legalLinks as item (item.path)}
					<li><a href={localizedPath(locale, item.path)} class={link}>{item.label}</a></li>
				{/each}
			</ul>
		</div>
	</div>
</footer>
