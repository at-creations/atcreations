<script lang="ts">
	import Clock from '@lucide/svelte/icons/clock';
	import Users from '@lucide/svelte/icons/users';
	import { m } from '$lib/paraglide/messages';
	import { localizedPath } from '$lib/i18n';
	import { assetUrl } from '$lib/urls';
	import Seo from '$lib/components/Seo.svelte';
	import SectionTitle from '$lib/components/SectionTitle.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const workshops = $derived([
		{
			id: 'plaster-painting',
			title: m.plaster_title(),
			summary: m.plaster_summary(),
			image: assetUrl(data.config.assetBaseUrl, 'workshop-space.jpg'),
			caption: m.workshop_vietnam_caption(),
			href: localizedPath(data.locale, '/workshops/plaster-painting'),
			duration: m.workshops_duration(),
			capacity: m.workshops_capacity()
		}
	]);
</script>

<Seo title={m.workshops_meta_title()} description={m.workshops_meta_description()} />

<div class="container-page pt-28 pb-20 md:pt-32">
	<SectionTitle level={1} title={m.workshops_title()} />
	<p class="mb-14 max-w-2xl text-lg leading-relaxed">{m.workshops_intro()}</p>

	<ul class="space-y-16">
		{#each workshops as workshop (workshop.id)}
			<li class="group relative grid items-center gap-8 md:grid-cols-12 md:gap-12">
				<div class="md:col-span-6">
					<img
						src={workshop.image}
						alt={workshop.caption}
						loading="lazy"
						class="aspect-[4/3] w-full rounded-t-lg object-cover"
					/>
					<div class="shelf [--shelf-gap:2rem]" aria-hidden="true"></div>
					<p class="mt-2 text-sm text-ink-soft">{workshop.caption}</p>
				</div>
				<div class="md:col-span-6">
					<h2 class="heading-section md:text-4xl">{workshop.title}</h2>
					<p class="mt-4 max-w-prose text-lg leading-relaxed">{workshop.summary}</p>
					<ul class="mt-6 flex flex-wrap gap-x-8 gap-y-3 font-medium text-ink">
						<li class="flex items-center gap-2">
							<Clock class="size-5 text-teak" />{workshop.duration}
						</li>
						<li class="flex items-center gap-2">
							<Users class="size-5 text-teak" />{workshop.capacity}
						</li>
					</ul>
					<a href={workshop.href} class="mt-8 btn-primary after:absolute after:inset-0">
						{m.workshops_learn_more()}<span class="sr-only">: {workshop.title}</span>
					</a>
				</div>
			</li>
		{/each}
	</ul>
</div>
