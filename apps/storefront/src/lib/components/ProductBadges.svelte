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
	}

	let { product, class: className = '' }: Props = $props();

	const isNew = $derived(isNewProduct(product));
	const onSale = $derived(isDiscounted(product));
</script>

{#if product.featured || isNew || onSale || !product.inStock}
	<ul class={['flex flex-wrap gap-1.5', className]}>
		{#if onSale}
			<li class="rounded-md bg-red-600 px-2 py-1 text-xs font-bold text-white shadow-sm">
				{m.product_sale({ percent: product.discountPercent })}
			</li>
		{/if}
		{#if product.featured}
			<li
				class="flex items-center gap-1 rounded-md bg-blue-600 px-2 py-1 text-xs font-bold text-white shadow-sm"
			>
				<Star class="size-3" fill="currentColor" />
				{m.product_featured()}
			</li>
		{/if}
		{#if isNew}
			<li class="rounded-md bg-emerald-700 px-2 py-1 text-xs font-bold text-white shadow-sm">
				{m.product_new()}
			</li>
		{/if}
		{#if !product.inStock}
			<li class="rounded-md bg-gray-700 px-2 py-1 text-xs font-bold text-white shadow-sm">
				{m.product_out_of_stock()}
			</li>
		{/if}
	</ul>
{/if}
