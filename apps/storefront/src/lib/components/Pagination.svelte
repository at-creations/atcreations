<script lang="ts">
	import ChevronLeft from '@lucide/svelte/icons/chevron-left';
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
	import { m } from '$lib/paraglide/messages';
	import { getPageItems } from '$lib/pagination';

	interface Props {
		current: number;
		totalPages: number;
		/** Builds the link for a given page number. */
		href: (page: number) => string;
	}

	let { current, totalPages, href }: Props = $props();

	const items = $derived(getPageItems(current, totalPages));
	const base =
		'flex size-10 items-center justify-center rounded-full text-sm font-medium transition-colors';
	const idle = 'border border-line bg-white text-ink-soft hover:bg-wall hover:text-ink';
</script>

{#if items.length > 0}
	<nav aria-label={m.pagination_label()} class="mt-12 flex justify-center">
		<ul class="flex flex-wrap items-center gap-1.5">
			<li>
				{#if current > 1}
					<a href={href(current - 1)} class={[base, idle]} aria-label={m.pagination_prev()}>
						<ChevronLeft class="size-5" />
					</a>
				{:else}
					<span class={[base, 'text-ink-soft/40']} aria-hidden="true"
						><ChevronLeft class="size-5" /></span
					>
				{/if}
			</li>
			{#each items as item (item)}
				<li>
					{#if typeof item === 'number'}
						<a
							href={href(item)}
							class={[
								base,
								item === current ? 'bg-marigold text-white shadow-md hover:bg-wall-deep' : idle
							]}
							aria-current={item === current ? 'page' : undefined}
							aria-label={m.pagination_page({ page: item })}
						>
							{item}
						</a>
					{:else}
						<span class="flex h-10 items-center px-1 text-ink-soft/80" aria-hidden="true">…</span>
					{/if}
				</li>
			{/each}
			<li>
				{#if current < totalPages}
					<a href={href(current + 1)} class={[base, idle]} aria-label={m.pagination_next()}>
						<ChevronRight class="size-5" />
					</a>
				{:else}
					<span class={[base, 'text-ink-soft/40']} aria-hidden="true"
						><ChevronRight class="size-5" /></span
					>
				{/if}
			</li>
		</ul>
	</nav>
{/if}
