<script lang="ts">
	import type { StoreInfo } from '@atcreations/api-types';
	import Clock from '@lucide/svelte/icons/clock';
	import ExternalLink from '@lucide/svelte/icons/external-link';
	import Mail from '@lucide/svelte/icons/mail';
	import MapPin from '@lucide/svelte/icons/map-pin';
	import Phone from '@lucide/svelte/icons/phone';
	import { m } from '$lib/paraglide/messages';
	import { directionsUrl, mapEmbedUrl } from '$lib/maps';
	import type { Locale } from '$lib/i18n';
	import BusinessHours from './BusinessHours.svelte';

	interface Props {
		storeInfo: StoreInfo | null;
		locale: Locale;
		/** Heading level for the card titles. */
		headingLevel?: 2 | 3;
	}

	let { storeInfo, locale, headingLevel = 3 }: Props = $props();

	const email = $derived(storeInfo?.email?.trim() || '');
	const phone = $derived(storeInfo?.phone?.trim() || '');
	const address = $derived(storeInfo?.address?.trim() || '');
	const mapSrc = $derived(mapEmbedUrl(storeInfo?.mapUrl, address, locale));
	const directions = $derived(directionsUrl(address));
	const heading = $derived(headingLevel === 2 ? 'h2' : 'h3');
	const cardClass = 'card card-hover flex items-start gap-5 p-6';
	const iconWrap = 'shrink-0 rounded-full bg-blue-50 p-3 text-blue-700';
	const titleClass = 'text-lg font-semibold text-gray-900';
</script>

<div class="mx-auto grid max-w-3xl grid-cols-1 gap-6 md:grid-cols-2 lg:gap-8">
	<div class={cardClass}>
		<div class={iconWrap}><Mail class="size-6" /></div>
		<div class="min-w-0">
			<svelte:element this={heading} class={titleClass}>{m.contact_email()}</svelte:element>
			{#if email}
				<a
					href={`mailto:${email}`}
					class="mt-1 block font-medium break-all text-blue-700 hover:underline"
				>
					{email}
				</a>
			{:else}
				<p class="mt-1">{m.contact_to_be_updated()}</p>
			{/if}
		</div>
	</div>

	<div class={cardClass}>
		<div class={iconWrap}><Phone class="size-6" /></div>
		<div>
			<svelte:element this={heading} class={titleClass}>{m.contact_phone()}</svelte:element>
			{#if phone}
				<a
					href={`tel:${phone.replace(/[^\d+]/g, '')}`}
					class="mt-1 block hover:text-blue-700 hover:underline"
				>
					{phone}
				</a>
			{:else}
				<p class="mt-1">{m.contact_to_be_updated()}</p>
			{/if}
		</div>
	</div>

	<div class={[cardClass, 'md:col-span-2']}>
		<div class={iconWrap}><MapPin class="size-6" /></div>
		<div>
			<svelte:element this={heading} class={titleClass}>{m.contact_address()}</svelte:element>
			<address class="mt-1 not-italic">{address || m.contact_to_be_updated()}</address>
		</div>
	</div>

	<div class={[cardClass, 'md:col-span-2']}>
		<div class={iconWrap}><Clock class="size-6" /></div>
		<div class="w-full">
			<svelte:element this={heading} class={[titleClass, 'mb-2']}
				>{m.contact_hours()}</svelte:element
			>
			<BusinessHours hours={storeInfo?.businessHours ?? []} {locale} />
		</div>
	</div>
</div>

{#if mapSrc}
	<div class="mx-auto mt-12 max-w-3xl card p-6">
		<svelte:element
			this={heading}
			class="mb-4 flex items-center gap-2 text-xl font-semibold text-gray-900"
		>
			<MapPin class="size-5 text-blue-700" />
			{m.contact_find_us()}
		</svelte:element>
		<div class="aspect-video w-full overflow-hidden rounded-lg bg-gray-100">
			<iframe
				src={mapSrc}
				title={m.contact_map_title()}
				class="size-full border-0"
				loading="lazy"
				referrerpolicy="no-referrer-when-downgrade"
				allowfullscreen
			></iframe>
		</div>
		{#if directions}
			<p class="mt-4 text-center">
				<a
					href={directions}
					target="_blank"
					rel="noopener noreferrer"
					class="inline-flex items-center gap-1 font-medium text-blue-700 hover:underline"
				>
					{m.contact_get_directions()}
					<ExternalLink class="size-4" />
					<span class="sr-only">{m.contact_opens_new_tab()}</span>
				</a>
			</p>
		{/if}
	</div>
{/if}
