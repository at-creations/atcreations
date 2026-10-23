<script lang="ts">
	import type { CategoryRef } from '@atcreations/api-types';
	import { Popover } from 'bits-ui';
	import { localized } from '$lib/i18n';
	import { m } from '$lib/paraglide/messages.js';

	let { categories }: { categories: CategoryRef[] } = $props();
	const first = $derived(categories[0]);
	const rest = $derived(categories.slice(1));
</script>

{#if !first}
	<span class="badge bg-zinc-100 text-zinc-500 italic">{m.products_uncategorized()}</span>
{:else}
	<span class="inline-flex flex-wrap items-center gap-1">
		<span class="badge max-w-40 truncate bg-brand-100 text-brand-800">{localized(first.name)}</span>
		{#if rest.length > 0}
			<Popover.Root>
				<Popover.Trigger
					class="badge cursor-pointer bg-zinc-100 text-zinc-700 hover:bg-zinc-200 focus-visible:ring-2 focus-visible:ring-brand-300 focus-visible:outline-none"
					aria-label={m.products_all_categories()}
				>
					{m.products_more_categories({ count: rest.length })}
				</Popover.Trigger>
				<Popover.Portal>
					<Popover.Content
						sideOffset={6}
						class="z-[60] max-w-xs animate-pop-in rounded-xl border border-zinc-200 bg-white p-3 shadow-popover"
					>
						<p class="mb-2 text-xs font-medium text-zinc-500">{m.products_all_categories()}</p>
						<ul class="flex flex-wrap gap-1">
							{#each categories as category (category.id)}
								<li class="badge bg-brand-100 text-brand-800">{localized(category.name)}</li>
							{/each}
						</ul>
					</Popover.Content>
				</Popover.Portal>
			</Popover.Root>
		{/if}
	</span>
{/if}
