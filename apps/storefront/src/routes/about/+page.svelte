<script lang="ts">
	import { m } from '$lib/paraglide/messages';
	import Seo from '$lib/components/Seo.svelte';
	import AboutIntro from '$lib/components/AboutIntro.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

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
			text: m.about_journey_text()
		},
		{
			title: m.about_craftsmen_title(),
			text: m.about_craftsmen_text()
		}
	]);
</script>

<Seo title={m.about_title()} description={m.about_meta_description()} type="article" />

<div class="container-page pt-28 pb-20 md:pt-32">
	<AboutIntro assetBaseUrl={data.config.assetBaseUrl}>
		{#snippet heading()}
			<h1 class="mb-8 heading-page">{m.about_title()}</h1>
		{/snippet}
	</AboutIntro>

	<div class="mt-24 grid grid-cols-1 gap-16 lg:grid-cols-2">
		{#each stories as story (story.title)}
			<section>
				<h2 class="heading-section">{story.title}</h2>
				<p class="mt-3 max-w-xl text-lg leading-relaxed">{story.text}</p>
			</section>
		{/each}
	</div>
</div>

<section class="bg-wall" aria-labelledby="values-title">
	<div class="container-page py-16 md:py-20">
		<h2 id="values-title" class="heading-section">{m.about_values_title()}</h2>
		<dl class="mt-10 grid grid-cols-1 gap-x-10 gap-y-8 sm:grid-cols-2 lg:grid-cols-4">
			{#each values as value (value.title)}
				<div>
					<dt class="font-display text-lg font-semibold text-ink">{value.title}</dt>
					<dd class="mt-2 leading-relaxed">{value.text}</dd>
				</div>
			{/each}
		</dl>
	</div>
</section>

<section class="container-page py-16 md:py-20" aria-labelledby="visit-title">
	<div class="max-w-3xl">
		<h2 id="visit-title" class="heading-section">{m.about_visit_title()}</h2>
		<ul class="mt-6 space-y-3 text-lg leading-relaxed">
			{#each visitPoints as point (point)}
				<li class="flex gap-3">
					<span class="mt-3 size-1.5 shrink-0 rounded-full bg-teak" aria-hidden="true"></span>
					<span>{point}</span>
				</li>
			{/each}
		</ul>
		<p class="mt-8 text-lg">{m.about_closing()}</p>
	</div>
</section>
