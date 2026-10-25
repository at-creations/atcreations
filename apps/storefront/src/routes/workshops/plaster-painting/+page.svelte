<script lang="ts">
	import Brush from '@lucide/svelte/icons/brush';
	import Clock from '@lucide/svelte/icons/clock';
	import Gift from '@lucide/svelte/icons/gift';
	import Heart from '@lucide/svelte/icons/heart';
	import Users from '@lucide/svelte/icons/users';
	import { m } from '$lib/paraglide/messages';
	import { localizedPath } from '$lib/i18n';
	import { assetUrl } from '$lib/urls';
	import Seo from '$lib/components/Seo.svelte';
	import SectionTitle from '$lib/components/SectionTitle.svelte';
	import Breadcrumbs from '$lib/components/Breadcrumbs.svelte';
	import Gallery from '$lib/components/Gallery.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const asset = (file: string) => assetUrl(data.config.assetBaseUrl, file);

	const images = $derived([
		// Our own photos only (add more as they are taken).
		{ url: asset('plaster-models.jpg'), alt: m.plaster_img_models() }
	]);
	const offerings = $derived([
		m.plaster_offering_figurines(),
		m.plaster_offering_paints(),
		m.plaster_offering_tools(),
		m.plaster_offering_space(),
		m.plaster_offering_guidance()
	]);
	const benefits = $derived([
		{ icon: Brush, title: m.plaster_benefit_relax_title(), text: m.plaster_benefit_relax_text() },
		{
			icon: Users,
			title: m.plaster_benefit_connect_title(),
			text: m.plaster_benefit_connect_text()
		},
		{ icon: Gift, title: m.plaster_benefit_create_title(), text: m.plaster_benefit_create_text() },
		{
			icon: Heart,
			title: m.plaster_benefit_discover_title(),
			text: m.plaster_benefit_discover_text()
		}
	]);
</script>

<Seo
	title={m.plaster_title()}
	description={m.plaster_summary()}
	image={asset('plaster-models.jpg')}
	type="article"
/>

<div class="container-page pt-24 pb-20 md:pt-28">
	<Breadcrumbs
		items={[
			{ label: m.nav_home(), href: localizedPath(data.locale) },
			{ label: m.nav_workshops(), href: localizedPath(data.locale, '/workshops') },
			{ label: m.plaster_title() }
		]}
	/>

	<SectionTitle level={1} title={m.plaster_title()} />

	<div class="grid gap-10 lg:grid-cols-12 lg:gap-14">
		<div class="lg:col-span-7">
			<Gallery {images} label={m.plaster_title()} />
		</div>
		<div class="lg:col-span-5">
			<h2 class="heading-section">{m.plaster_welcome_heading()}</h2>
			<p class="mt-4 text-lg leading-relaxed">{m.plaster_welcome_message()}</p>
			<ul class="mt-6 flex flex-wrap gap-x-8 gap-y-3 font-medium text-ink">
				<li class="flex items-center gap-2">
					<Clock class="size-5 text-teak" />{m.workshops_duration()}
				</li>
				<li class="flex items-center gap-2">
					<Users class="size-5 text-teak" />{m.workshops_capacity()}
				</li>
			</ul>
			<a href={localizedPath(data.locale, '/contact')} class="mt-8 btn-primary"
				>{m.plaster_book_cta()}</a
			>
		</div>
	</div>

	<section class="mt-20 grid gap-10 md:grid-cols-12" aria-labelledby="provide-title">
		<h2 id="provide-title" class="heading-section md:col-span-4">{m.plaster_provide_heading()}</h2>
		<ul class="space-y-3 text-lg md:col-span-8">
			{#each offerings as offering (offering)}
				<li class="flex gap-3">
					<Brush class="mt-1 size-5 shrink-0 text-teak" />
					<span>{offering}</span>
				</li>
			{/each}
		</ul>
	</section>
</div>

<section class="bg-wall" aria-labelledby="benefits-title">
	<div class="container-page py-16 md:py-20">
		<h2 id="benefits-title" class="heading-section">{m.plaster_benefits_heading()}</h2>
		<ul class="mt-10 grid grid-cols-1 gap-x-10 gap-y-10 sm:grid-cols-2 lg:grid-cols-4">
			{#each benefits as benefit (benefit.title)}
				<li>
					<benefit.icon class="size-6 text-teak" />
					<h3 class="mt-3 text-lg font-semibold">{benefit.title}</h3>
					<p class="mt-2 leading-relaxed">{benefit.text}</p>
				</li>
			{/each}
		</ul>
		<p class="mt-12 max-w-2xl text-lg font-medium text-ink">{m.plaster_closing()}</p>
	</div>
</section>
