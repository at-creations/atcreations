<script lang="ts">
	import { imageUrl } from '$lib/utils/images';
	import type { Product } from '@atcreations/api-types';
	import { resolve } from '$app/paths';
	import { localized } from '$lib/i18n';
	import { m } from '$lib/paraglide/messages.js';
	import Icon from '../Icon.svelte';
	import StatusBadge from '../StatusBadge.svelte';
	import Thumb from '../ui/Thumb.svelte';
	import PriceLabel from './PriceLabel.svelte';
	import StockLabel from './StockLabel.svelte';

	let { product }: { product: Product } = $props();

	const category = $derived(product.categories[0]);
</script>

<a
	href={resolve(`/products/${product.id}`)}
	class="group flex flex-col overflow-hidden rounded-2xl border border-zinc-200/70 bg-white transition hover:-translate-y-0.5 hover:shadow-lg hover:shadow-zinc-200/60 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-400"
>
	<div class="relative aspect-[4/3] overflow-hidden bg-zinc-100">
		<Thumb
			src={imageUrl(product.images[0], 600)}
			class="size-full transition-transform duration-300 group-hover:scale-[1.03]"
			rounded=""
			iconSize={28}
		/>
		<div class="absolute inset-x-2 top-2 flex items-start justify-between">
			<span class="rounded-md bg-white/90 shadow-sm backdrop-blur">
				<StatusBadge kind="product" value={product.status} />
			</span>
			{#if product.featured}
				<span
					class="flex size-7 items-center justify-center rounded-full bg-white/90 text-amber-400 shadow-sm backdrop-blur"
				>
					<Icon name="star" size={14} filled label={m.products_featured()} />
				</span>
			{/if}
		</div>
		{#if product.images.length > 1}
			<span
				class="absolute right-2 bottom-2 flex items-center gap-1 rounded-md bg-zinc-900/60 px-1.5 py-0.5 text-[11px] font-medium text-white backdrop-blur"
			>
				<Icon name="image" size={11} />{product.images.length}
			</span>
		{/if}
	</div>
	<div class="flex flex-1 flex-col gap-1 p-3.5">
		<p class="truncate text-xs text-zinc-400">
			{category ? localized(category.name) : m.products_uncategorized()}
		</p>
		<p
			class="line-clamp-2 text-sm leading-snug font-medium text-zinc-900 group-hover:text-brand-700"
		>
			{localized(product.name)}
		</p>
		<div class="mt-auto flex items-end justify-between gap-2 pt-2">
			<PriceLabel
				finalCents={product.finalPriceCents}
				baseCents={product.priceCents}
				discountPercent={product.discountPercent}
			/>
			<StockLabel stock={product.stock} />
		</div>
	</div>
</a>
