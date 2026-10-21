<script lang="ts">
	import Brush from '@lucide/svelte/icons/brush';
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
		{ url: asset('workshop-space.jpg'), alt: m.plaster_img_space() },
		{ url: asset('plaster-models.jpg'), alt: m.plaster_img_models() },
		{ url: asset('finished-works.jpg'), alt: m.plaster_img_finished() },
		{ url: asset('painting-session.jpg'), alt: m.plaster_img_session() }
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
	image={asset('workshop-space.jpg')}
	type="article"
/>

<div class="container-page max-w-6xl animate-fade-in pt-24 pb-20">
	<Breadcrumbs
		items={[
			{ label: m.nav_home(), href: localizedPath(data.locale) },
			{ label: m.nav_workshops(), href: localizedPath(data.locale, '/workshops') },
			{ label: m.plaster_title() }
		]}
	/>

	<SectionTitle level={1} title={m.plaster_title()} />

	<div class="mx-auto max-w-4xl">
		<Gallery {images} label={m.plaster_title()} aspect="video" />
	</div>

	<section class="mt-12 text-center">
		<h2 class="text-2xl font-semibold text-gray-900">{m.plaster_welcome_heading()}</h2>
		<p class="mx-auto mt-4 max-w-3xl text-lg leading-relaxed">{m.plaster_welcome_message()}</p>
	</section>

	<section class="mt-16 card p-6">
		<h2 class="mb-6 flex items-center text-xl font-semibold text-gray-900">
			<span class="mr-3 rounded-md bg-blue-100 p-2 text-blue-700"><Brush class="size-5" /></span>
			{m.plaster_provide_heading()}
		</h2>
		<ul class="space-y-4 pl-4">
			{#each offerings as offering (offering)}
				<li class="flex items-start gap-2">
					<span class="text-xl leading-none font-bold text-blue-600" aria-hidden="true">•</span>
					<span>{offering}</span>
				</li>
			{/each}
		</ul>
	</section>

	<section class="mt-16">
		<h2 class="mb-8 text-center text-xl font-semibold text-gray-900">
			{m.plaster_benefits_heading()}
		</h2>
		<ul class="grid grid-cols-1 gap-6 md:grid-cols-2">
			{#each benefits as benefit (benefit.title)}
				<li class="card p-6">
					<div class="mb-3 flex items-center gap-4">
						<div class="rounded-full bg-blue-50 p-3 text-blue-600">
							<benefit.icon class="size-6" />
						</div>
						<h3 class="text-lg font-medium text-gray-900">{benefit.title}</h3>
					</div>
					<p class="sm:pl-16">{benefit.text}</p>
				</li>
			{/each}
		</ul>
	</section>

	<p
		class="mt-16 rounded-xl bg-linear-to-r from-blue-50 to-indigo-50 p-8 text-center text-lg font-medium text-gray-800"
	>
		{m.plaster_closing()}
	</p>
</div>
