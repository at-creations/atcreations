<script lang="ts">
	import type { StoreInfo } from '@atcreations/api-types';
	import { m } from '$lib/paraglide/messages';
	import { localizedPath, type Locale } from '$lib/i18n';
	import { assetUrl } from '$lib/urls';
	import BusinessHours from './BusinessHours.svelte';

	interface Props {
		locale: Locale;
		assetBaseUrl: string;
		storeInfo: StoreInfo | null;
	}

	let { locale, assetBaseUrl, storeInfo }: Props = $props();

	const year = new Date().getFullYear();
	const link = 'text-white/80 transition-colors hover:text-white hover:underline';

	const quickLinks = $derived([
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
	const hours = $derived(storeInfo?.businessHours ?? []);
</script>

<footer class="bg-ink text-white/80">
	<div class="container-page grid grid-cols-1 gap-12 py-14 md:grid-cols-12 md:py-16">
		<div class="md:col-span-4">
			<img
				src={assetUrl(assetBaseUrl, 'logo-square.png')}
				alt={m.footer_logo_alt()}
				width="48"
				height="48"
				loading="lazy"
				class="size-12 rounded-lg bg-white p-1"
			/>
			<p class="mt-5 max-w-xs font-display text-xl leading-snug text-white">{m.footer_tagline()}</p>
		</div>

		<nav aria-labelledby="footer-quick-links" class="md:col-span-2">
			<h2 id="footer-quick-links" class="mb-4 font-sans text-sm font-semibold text-white">
				{m.footer_quick_links()}
			</h2>
			<ul class="space-y-2.5">
				{#each quickLinks as item (item.path)}
					<li><a href={localizedPath(locale, item.path)} class={link}>{item.label}</a></li>
				{/each}
			</ul>
		</nav>

		<div class="md:col-span-6">
			<h2 class="mb-4 font-sans text-sm font-semibold text-white">{m.footer_contact()}</h2>
			<div class="grid gap-8 sm:grid-cols-2">
				<address class="max-w-60 space-y-2.5 not-italic">
					<p>{address || m.contact_to_be_updated()}</p>
					{#if phone}
						<p><a href={`tel:${phone.replace(/[^\d+]/g, '')}`} class={link}>{phone}</a></p>
					{/if}
					{#if email}
						<p><a href={`mailto:${email}`} class={[link, 'break-all']}>{email}</a></p>
					{/if}
				</address>
				{#if hours.length > 0}
					<div class="[&_dt]:text-white">
						<BusinessHours {hours} {locale} columns={1} />
					</div>
				{/if}
			</div>
		</div>
	</div>

	<div class="border-t border-white/10">
		<div
			class="container-page flex flex-col gap-3 py-6 text-sm text-white/60 sm:flex-row sm:items-center sm:justify-between"
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
