<script lang="ts">
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
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
			href: localizedPath(data.locale, '/workshops/plaster-painting'),
			duration: m.workshops_duration(),
			capacity: m.workshops_capacity()
		}
	]);
</script>

<Seo title={m.workshops_meta_title()} description={m.workshops_meta_description()} />

<div class="container-page max-w-6xl pt-28 pb-20">
	<SectionTitle level={1} title={m.workshops_title()} />
	<p class="mx-auto mb-12 max-w-3xl text-center text-lg">{m.workshops_intro()}</p>

	<ul class="space-y-6">
		{#each workshops as workshop (workshop.id)}
			<li
				class="relative flex animate-fade-in flex-col overflow-hidden card hover:shadow-card-hover md:flex-row"
			>
				<div class="relative aspect-video shrink-0 md:aspect-auto md:w-2/5 xl:w-1/3">
					<img
						src={workshop.image}
						alt=""
						loading="lazy"
						class="absolute inset-0 size-full object-cover"
					/>
				</div>
				<div class="flex flex-1 flex-col justify-between p-6 md:p-8">
					<div>
						<h2 class="mb-3 text-xl font-semibold text-gray-900 md:text-2xl">{workshop.title}</h2>
						<p class="mb-6 max-w-prose text-gray-600">{workshop.summary}</p>
						<ul class="mb-6 grid grid-cols-1 gap-4 text-sm text-gray-600 sm:grid-cols-2">
							<li class="flex items-center gap-2">
								<Clock class="size-4 text-blue-600" />{workshop.duration}
							</li>
							<li class="flex items-center gap-2">
								<Users class="size-4 text-blue-600" />{workshop.capacity}
							</li>
						</ul>
					</div>
					<a
						href={workshop.href}
						class="inline-flex items-center font-medium text-blue-700 after:absolute after:inset-0 hover:underline"
					>
						{m.workshops_learn_more()}<span class="sr-only">: {workshop.title}</span>
						<ChevronRight class="ml-1 size-4" />
					</a>
				</div>
			</li>
		{/each}
	</ul>
</div>
