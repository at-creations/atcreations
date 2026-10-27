<script lang="ts">
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
	import { page } from '$app/state';
	import { m } from '$lib/paraglide/messages';
	import { breadcrumbJsonLd, type Crumb } from '$lib/structured-data';
	import JsonLd from './JsonLd.svelte';

	interface Props {
		items: Crumb[];
	}

	let { items }: Props = $props();

	const jsonLd = $derived(
		breadcrumbJsonLd(
			page.data.config.siteUrl,
			items,
			`${page.data.config.siteUrl}${page.url.pathname}`
		)
	);
</script>

<JsonLd data={jsonLd} />

<nav aria-label={m.breadcrumb_label()} class="mb-6 text-sm">
	<ol class="flex flex-wrap items-center gap-1 text-ink-soft">
		{#each items as item, i (i)}
			<li class="flex items-center gap-1">
				{#if i > 0}<ChevronRight class="size-4 text-ink-soft/60" />{/if}
				{#if item.href && i < items.length - 1}
					<a href={item.href} class="hover:text-ink hover:underline">{item.label}</a>
				{:else}
					<span aria-current="page" class="font-medium text-ink">{item.label}</span>
				{/if}
			</li>
		{/each}
	</ol>
</nav>
