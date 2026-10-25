<script lang="ts">
	import type { StoreInfo } from '@atcreations/api-types';
	import Clock from '@lucide/svelte/icons/clock';
	import MapPin from '@lucide/svelte/icons/map-pin';
	import Phone from '@lucide/svelte/icons/phone';
	import { m } from '$lib/paraglide/messages';
	import { localizedPath, type Locale } from '$lib/i18n';
	import { formatTime } from '$lib/format';
	import { todaysHours } from '$lib/hours';
	import { directionsUrl } from '$lib/maps';
	import { assetUrl } from '$lib/urls';

	interface Props {
		locale: Locale;
		assetBaseUrl: string;
		storeInfo: StoreInfo | null;
	}

	let { locale, assetBaseUrl, storeInfo }: Props = $props();

	const address = $derived(storeInfo?.address?.trim() || '');
	const phone = $derived(storeInfo?.phone?.trim() || '');
	const today = $derived(todaysHours(storeInfo?.businessHours ?? []));
	const directions = $derived(directionsUrl(address));
</script>

<section class="bg-wall pt-16" aria-labelledby="hero-title">
	<div class="container-page grid items-end gap-10 pt-12 pb-10 md:pt-16 lg:grid-cols-12 lg:gap-14">
		<div class="lg:col-span-7 lg:pb-12">
			<h1
				id="hero-title"
				class="font-display text-5xl leading-[0.95] font-bold tracking-tight text-ink sm:text-6xl xl:text-7xl"
			>
				{m.home_hero_title()}
			</h1>
			<p class="mt-6 max-w-xl text-lg leading-relaxed text-ink-soft md:text-xl">
				{m.home_hero_text()}
			</p>
			<div class="mt-9 flex flex-wrap items-center gap-x-7 gap-y-4">
				<a href={localizedPath(locale, '/products')} class="btn-primary px-7 text-lg">
					{m.home_browse()}
				</a>
				<a href={localizedPath(locale, '/workshops/plaster-painting')} class="text-lg link">
					{m.home_workshop_link()}
				</a>
			</div>
		</div>

		<!-- The shop itself, standing on a shelf like everything it sells. -->
		<div class="mx-auto w-full max-w-md lg:col-span-5 lg:max-w-none">
			<img
				src={assetUrl(assetBaseUrl, 'about_col.jpg')}
				alt={m.about_image_alt()}
				width="765"
				height="956"
				fetchpriority="high"
				class="aspect-[4/5] w-full rounded-t-lg object-cover"
			/>
			<div class="shelf [--shelf-gap:2.5rem]" aria-hidden="true"></div>
		</div>
	</div>

	{#if address || today || phone}
		<div class="border-t border-wall-deep">
			<ul
				class="container-page flex flex-col gap-x-10 gap-y-3 py-5 text-ink sm:flex-row sm:flex-wrap sm:items-center"
			>
				{#if address}
					<li class="flex items-center gap-2.5">
						<MapPin class="size-5 shrink-0 text-teak" />
						{#if directions}
							<a
								href={directions}
								target="_blank"
								rel="noopener noreferrer"
								class="link font-medium"
							>
								{address}<span class="sr-only"> ({m.contact_opens_new_tab()})</span>
							</a>
						{:else}
							<span>{address}</span>
						{/if}
					</li>
				{/if}
				{#if today}
					<li class="flex items-center gap-2.5">
						<Clock class="size-5 shrink-0 text-teak" />
						{#if today === 'closed'}
							{m.visit_closed_today()}
						{:else}
							{m.visit_open_today({
								hours: `${formatTime(today.open, locale)} – ${formatTime(today.close, locale)}`
							})}
						{/if}
					</li>
				{/if}
				{#if phone}
					<li class="flex items-center gap-2.5">
						<Phone class="size-5 shrink-0 text-teak" />
						<a href={`tel:${phone.replace(/[^\d+]/g, '')}`} class="hover:underline">{phone}</a>
					</li>
				{/if}
			</ul>
		</div>
	{/if}
</section>
