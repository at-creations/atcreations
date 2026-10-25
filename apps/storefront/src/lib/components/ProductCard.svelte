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
	const category = $derived(product.categories[0]);
</script>

<!-- The image stands on the shared shelf edge (see .shelf); the text sits below it. -->
<article class="group relative flex h-full flex-col">
	<div class="relative aspect-square overflow-hidden rounded-t-md bg-wall">
		{#if cover}
			<img
				src={cover.url}
				srcset={srcset(cover)}
				sizes={GRID_SIZES}
				alt=""
				width={cover.width}
				height={cover.height}
				loading={eager ? 'eager' : 'lazy'}
				decoding="async"
				class="size-full object-cover transition-transform duration-500 group-hover:scale-[1.03] motion-reduce:group-hover:scale-100"
			/>
		{:else}
			<div class="flex size-full items-center justify-center text-ink-soft/60" aria-hidden="true">
				<ImageOff class="size-10" />
			</div>
		{/if}
		<ProductBadges {product} showFeatured={false} class="absolute top-2.5 left-2.5" />
	</div>
	<div class="shelf" aria-hidden="true"></div>

	<div class="flex flex-1 flex-col pt-4">
		{#if category}
			<p class="text-sm text-ink-soft">{t(category.name, locale)}</p>
		{/if}
		<h3 class="mt-0.5 line-clamp-2 font-sans text-base leading-snug font-semibold">
			<a
				href={productHref(locale, product.slug)}
				class="group-hover:underline group-hover:decoration-marigold group-hover:decoration-2 group-hover:underline-offset-4 after:absolute after:inset-0"
			>
				{name}
			</a>
		</h3>
		<div class="mt-2">
			<Price {product} {locale} />
		</div>
	</div>
</article>
