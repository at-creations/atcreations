<script lang="ts">
	import { m } from '$lib/paraglide/messages';
	import { assetUrl } from '$lib/urls';
	import Seo from '$lib/components/Seo.svelte';
	import AboutIntro from '$lib/components/AboutIntro.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const asset = (file: string) => assetUrl(data.config.assetBaseUrl, file);

	const values = $derived([
		{ title: m.about_uniqueness_title(), text: m.about_uniqueness_text() },
		{ title: m.about_quality_title(), text: m.about_quality_text() },
		{ title: m.about_connection_title(), text: m.about_connection_text() },
		{ title: m.about_creativity_title(), text: m.about_creativity_text() }
	]);
	const visitPoints = $derived([
		m.about_visit_point1(),
		m.about_visit_point2(),
		m.about_visit_point3(),
		m.about_visit_point4()
	]);
	const stories = $derived([
		{
			title: m.about_journey_title(),
			text: m.about_journey_text(),
			image: asset('pottery.jpg'),
			alt: m.about_crafts_image_alt()
		},
		{
			title: m.about_craftsmen_title(),
			text: m.about_craftsmen_text(),
			image: asset('wood-craft.jpg'),
			alt: m.about_craftsman_image_alt()
		}
	]);
</script>

<Seo title={m.about_title()} description={m.about_meta_description()} type="article" />

<div class="container-page max-w-6xl animate-fade-in pt-28 pb-20">
	<AboutIntro assetBaseUrl={data.config.assetBaseUrl}>
		{#snippet heading()}
			<h1 class="mb-6 text-3xl font-semibold text-gray-900 lg:text-4xl">{m.about_title()}</h1>
		{/snippet}
	</AboutIntro>

	<div class="mt-20 grid grid-cols-1 gap-10 lg:grid-cols-2">
		{#each stories as story (story.title)}
			<section class="flex flex-col card p-8 hover:shadow-card-hover">
				<h2 class="mb-4 flex items-center text-xl font-semibold text-gray-900">
					<span class="mr-3 h-8 w-2 rounded-full bg-blue-500" aria-hidden="true"></span>
					{story.title}
				</h2>
				<p class="mb-6 leading-relaxed">{story.text}</p>
				<div class="mt-auto overflow-hidden rounded-lg">
					<img
						src={story.image}
						alt={story.alt}
						width="600"
						height="400"
						loading="lazy"
						class="h-auto w-full object-cover transition-transform duration-500 hover:scale-105"
					/>
				</div>
			</section>
		{/each}
	</div>

	<section class="mt-16 rounded-2xl bg-linear-to-r from-blue-50 to-indigo-50 p-8 shadow-card">
		<h2 class="mb-6 text-center text-xl font-semibold text-gray-900">{m.about_values_title()}</h2>
		<ul class="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-4">
			{#each values as value (value.title)}
				<li class="rounded-xl bg-white p-5 shadow-sm transition hover:shadow-md">
					<h3 class="mb-2 font-semibold text-blue-700">{value.title}</h3>
					<p class="text-sm">{value.text}</p>
				</li>
			{/each}
		</ul>
	</section>

	<section class="mt-16">
		<h2 class="mb-6 text-center text-xl font-semibold text-gray-900">{m.about_visit_title()}</h2>
		<ul class="mx-auto max-w-4xl space-y-4">
			{#each visitPoints as point (point)}
				<li class="flex items-start gap-2">
					<span class="text-blue-600" aria-hidden="true">•</span>
					<span>{point}</span>
				</li>
			{/each}
		</ul>
		<p class="mx-auto mt-8 max-w-4xl italic">{m.about_closing()}</p>
	</section>
</div>
