<script lang="ts">
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
	import { m } from '$lib/paraglide/messages';

	interface Crumb {
		label: string;
		href?: string;
	}

	interface Props {
		items: Crumb[];
	}

	let { items }: Props = $props();
</script>

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
