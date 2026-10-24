<script lang="ts">
	import { GRID_SIZES, srcset } from '$lib/images';
	import type { PublicProduct } from '@atcreations/api-types';
	import ImageOff from '@lucide/svelte/icons/image-off';
	import { t, type Locale } from '$lib/i18n';
	import { productHref } from '$lib/urls';
	import Price from './Price.svelte';
	import ProductBadges from './ProductBadges.svelte';

	interface Props {
		product: PublicProduct;
		locale: Locale;
		/** Load the image eagerly (for above-the-fold cards). */
		eager?: boolean;
	}

	let { product, locale, eager = false }: Props = $props();

	const name = $derived(t(product.name, locale));
	const cover = $derived(product.images[0]);
	const visibleCategories = $derived(product.categories.slice(0, 2));
	const extraCategories = $derived(product.categories.length - visibleCategories.length);
</script>

<article class="group relative flex h-full card-hover flex-col overflow-hidden card">
	<div class="relative aspect-[4/3] w-full overflow-hidden bg-gray-100">
		{#if cover}
			<img
				src={cover.url}
				srcset={srcset(cover)}
				sizes={GRID_SIZES}
				alt={name}
				width={cover.width}
				height={cover.height}
				loading={eager ? 'eager' : 'lazy'}
				decoding="async"
				class="size-full object-cover transition-transform duration-500 group-hover:scale-105 motion-reduce:group-hover:scale-100"
			/>
		{:else}
			<div class="flex size-full items-center justify-center text-gray-400" aria-hidden="true">
				<ImageOff class="size-10" />
			</div>
		{/if}
		<ProductBadges {product} class="absolute top-3 right-3 left-3" />
	</div>

	<div class="flex flex-1 flex-col p-5">
		{#if visibleCategories.length > 0}
			<ul class="mb-2 flex flex-wrap gap-1.5">
				{#each visibleCategories as category (category.id)}
					<li class="rounded-md bg-blue-50 px-2 py-0.5 text-xs font-medium text-blue-700">
						{t(category.name, locale)}
					</li>
				{/each}
				{#if extraCategories > 0}
					<li class="text-xs text-gray-500">+{extraCategories}</li>
				{/if}
			</ul>
		{/if}

		<h3 class="line-clamp-2 text-lg font-semibold text-gray-900 group-hover:text-blue-700">
			<a href={productHref(locale, product.slug)} class="after:absolute after:inset-0">
				{name}
			</a>
		</h3>

		<p class="mt-2 line-clamp-2 text-sm text-gray-600">{t(product.description, locale)}</p>

		<div class="mt-auto border-t border-gray-100 pt-3">
			<Price {product} {locale} />
		</div>
	</div>
</article>
