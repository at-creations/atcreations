<script lang="ts">
	import { m } from '$lib/paraglide/messages.js';
	import { formatCAD } from '$lib/utils/money';

	let {
		finalCents,
		baseCents,
		discountPercent = 0,
		size = 'sm'
	}: {
		/** Price after discount; null = "contact for price". */
		finalCents: number | null;
		baseCents: number | null;
		discountPercent?: number;
		size?: 'sm' | 'lg';
	} = $props();
</script>

{#if finalCents === null}
	<span class={['text-zinc-400', size === 'lg' ? 'text-sm' : 'text-xs']}
		>{m.products_contact_price()}</span
	>
{:else}
	<span class="inline-flex flex-wrap items-baseline gap-x-1.5">
		<span
			class={['font-semibold text-zinc-900 tabular-nums', size === 'lg' ? 'text-lg' : 'text-sm']}
			>{formatCAD(finalCents)}</span
		>
		{#if discountPercent > 0 && baseCents !== null}
			<span class="text-xs text-zinc-400 tabular-nums line-through">{formatCAD(baseCents)}</span>
			<span class="badge bg-rose-100 text-rose-700"
				>{m.products_discount({ percent: discountPercent })}</span
			>
		{/if}
	</span>
{/if}
