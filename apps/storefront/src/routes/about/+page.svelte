<script lang="ts">
	import { m } from '$lib/paraglide/messages';
	import Seo from '$lib/components/Seo.svelte';
	import AboutIntro from '$lib/components/AboutIntro.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

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
