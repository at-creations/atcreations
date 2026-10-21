<script lang="ts">
	import type { Product } from '@atcreations/api-types';
	import { m } from '$lib/paraglide/messages';
	import { formatMoney } from '$lib/format';
	import { isDiscounted, savingsCents } from '$lib/product';
	import type { Locale } from '$lib/i18n';

	interface Props {
		product: Pick<Product, 'priceCents' | 'finalPriceCents' | 'discountPercent'>;
		locale: Locale;
		size?: 'sm' | 'lg';
		showSavings?: boolean;
	}

	let { product, locale, size = 'sm', showSavings = false }: Props = $props();

	const discounted = $derived(isDiscounted(product));
	const finalCents = $derived(product.finalPriceCents ?? product.priceCents);
</script>

{#if finalCents === null}
	<p class={['text-gray-600', size === 'lg' ? 'text-lg font-medium' : 'text-sm']}>
		{m.product_contact_price()}
	</p>
{:else}
	<div class={['flex flex-wrap items-baseline', size === 'lg' ? 'gap-x-3 gap-y-1' : 'gap-x-2']}>
		<p class={['font-bold text-blue-700', size === 'lg' ? 'text-3xl' : 'text-base']}>
			{#if discounted}<span class="sr-only">{m.product_current_price()}</span>{/if}
			{formatMoney(finalCents, locale)}
		</p>
		{#if discounted && product.priceCents !== null}
			<p class={['text-gray-500 line-through', size === 'lg' ? 'text-lg' : 'text-xs']}>
				<span class="sr-only">{m.product_original_price()}</span>
				{formatMoney(product.priceCents, locale)}
			</p>
		{/if}
	</div>
	{#if showSavings && discounted}
		<p class="mt-1 text-sm font-medium text-red-700">
			{m.product_you_save({
				amount: formatMoney(savingsCents(product), locale),
				percent: product.discountPercent
			})}
		</p>
	{/if}
{/if}
