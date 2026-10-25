<script lang="ts">
	import type { PublicProduct } from '@atcreations/api-types';
	import Star from '@lucide/svelte/icons/star';
	import { m } from '$lib/paraglide/messages';
	import { isDiscounted, isNewProduct } from '$lib/product';

	interface Props {
		product: Pick<
			PublicProduct,
			'featured' | 'createdAt' | 'inStock' | 'priceCents' | 'finalPriceCents' | 'discountPercent'
		>;
		class?: string;
		/** Off in product grids, where most items shown are featured anyway. */
		showFeatured?: boolean;
	}

	let { product, class: className = '', showFeatured = true }: Props = $props();

	const isNew = $derived(isNewProduct(product));
	const onSale = $derived(isDiscounted(product));
</script>

{#if (showFeatured && product.featured) || isNew || onSale || !product.inStock}
	<ul class={['flex flex-wrap gap-1.5', className]}>
		{#if onSale}
			<li class="rounded-md bg-marigold px-2 py-1 text-xs font-bold text-ink">
				{m.product_sale({ percent: product.discountPercent })}
			</li>
		{/if}
		{#if showFeatured && product.featured}
			<li
				class="flex items-center gap-1 rounded-md bg-white px-2 py-1 text-xs font-semibold text-ink"
			>
				<Star class="size-3 text-marigold" fill="currentColor" />
				{m.product_featured()}
			</li>
		{/if}
		{#if isNew}
			<li class="rounded-md bg-ink px-2 py-1 text-xs font-semibold text-white">
				{m.product_new()}
			</li>
		{/if}
		{#if !product.inStock}
			<li class="rounded-md bg-white/90 px-2 py-1 text-xs font-semibold text-ink-soft">
				{m.product_out_of_stock()}
			</li>
		{/if}
	</ul>
{/if}
