<script lang="ts">
	import type { CategoryRef } from '@atcreations/api-types';
	import { localized } from '$lib/i18n';
	import { m } from '$lib/paraglide/messages.js';

	let { categories }: { categories: CategoryRef[] } = $props();
	const uid = $props.id();
	const first = $derived(categories[0]);
	const rest = $derived(categories.slice(1));
</script>

{#if !first}
	<span class="badge bg-slate-100 text-slate-500 italic">{m.products_uncategorized()}</span>
{:else}
	<span class="inline-flex flex-wrap items-center gap-1">
		<span class="badge max-w-40 truncate bg-brand-50 text-brand-800">{localized(first.name)}</span>
		{#if rest.length > 0}
			<button
				type="button"
				class="badge cursor-pointer bg-slate-100 text-slate-700 hover:bg-slate-200"
				popovertarget="{uid}-more"
				aria-label={m.products_all_categories()}
			>
				{m.products_more_categories({ count: rest.length })}
			</button>
			<div
				id="{uid}-more"
				popover="auto"
				class="m-auto max-w-xs rounded-lg border border-slate-200 bg-white p-3 shadow-xl"
			>
				<p class="mb-2 text-xs font-semibold text-slate-500 uppercase">
					{m.products_all_categories()}
				</p>
				<ul class="flex flex-wrap gap-1">
					{#each categories as category (category.id)}
						<li class="badge bg-brand-50 text-brand-800">{localized(category.name)}</li>
					{/each}
				</ul>
			</div>
		{/if}
	</span>
{/if}
