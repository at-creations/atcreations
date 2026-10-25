<script lang="ts">
	import { sized, srcset } from '$lib/images';
	import ChevronLeft from '@lucide/svelte/icons/chevron-left';
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
	import Maximize from '@lucide/svelte/icons/maximize';
	import ImageOff from '@lucide/svelte/icons/image-off';
	import { m } from '$lib/paraglide/messages';
	import Lightbox from './Lightbox.svelte';
	import type { GalleryImage } from './gallery';

	interface Props {
		images: GalleryImage[];
		/** Accessible name of the carousel region. */
		label?: string;
		aspect?: 'square' | 'video';
	}

	let { images, label, aspect = 'square' }: Props = $props();

	let index = $state(0);
	let lightboxOpen = $state(false);
	const current = $derived(images[index]);
	const total = $derived(images.length);

	function go(delta: number) {
		if (total === 0) return;
		index = (index + delta + total) % total;
	}

	function onKeydown(event: KeyboardEvent) {
		if (event.key === 'ArrowLeft') {
			event.preventDefault();
			go(-1);
		} else if (event.key === 'ArrowRight') {
			event.preventDefault();
			go(1);
		}
	}
</script>

<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<section
	aria-roledescription="carousel"
	aria-label={label ?? m.gallery_label()}
	class="space-y-3"
	onkeydown={onKeydown}
>
	<div
		class={[
			'relative overflow-hidden rounded-t-lg bg-wall',
			aspect === 'square' ? 'aspect-square' : 'aspect-video'
		]}
	>
		{#if current}
			<button
				type="button"
				class="group block size-full cursor-zoom-in"
				onclick={() => (lightboxOpen = true)}
				aria-label={m.gallery_open({ current: index + 1, total })}
			>
				{#key index}
					<img
						src={current.url}
						srcset={srcset(current)}
						sizes="(min-width: 1280px) 600px, (min-width: 1024px) 50vw, 100vw"
						alt={current.alt}
						width={current.width}
						height={current.height}
						class="size-full animate-fade-in object-cover"
						fetchpriority={index === 0 ? 'high' : undefined}
					/>
				{/key}
				<span
					class="absolute right-3 bottom-3 rounded-full bg-ink/70 p-2 text-white opacity-80 group-hover:opacity-100"
					aria-hidden="true"
				>
					<Maximize class="size-5" />
				</span>
			</button>

			{#if total > 1}
				<button
					type="button"
					class="absolute top-1/2 left-3 -translate-y-1/2 rounded-full bg-white/90 p-2 text-ink shadow-md hover:bg-white"
					onclick={() => go(-1)}
					aria-label={m.gallery_prev()}
				>
					<ChevronLeft class="size-5" />
				</button>
				<button
					type="button"
					class="absolute top-1/2 right-3 -translate-y-1/2 rounded-full bg-white/90 p-2 text-ink shadow-md hover:bg-white"
					onclick={() => go(1)}
					aria-label={m.gallery_next()}
				>
					<ChevronRight class="size-5" />
				</button>
				<p
					class="absolute top-3 right-3 rounded-full bg-ink/70 px-2.5 py-1 text-xs font-medium text-white"
					aria-live="polite"
				>
					{m.gallery_counter({ current: index + 1, total })}
				</p>
			{/if}
		{:else}
			<div class="flex size-full flex-col items-center justify-center gap-2 text-ink-soft/80">
				<ImageOff class="size-12" />
				<p>{m.gallery_no_image()}</p>
			</div>
		{/if}
	</div>
	<div class="shelf [--shelf-gap:0px]" aria-hidden="true"></div>
	{#if current?.caption}
		<p class="text-sm text-ink-soft">{current.caption}</p>
	{/if}

	{#if total > 1}
		<ul class="flex gap-2 overflow-x-auto pt-2 pb-1">
			{#each images as image, i (i)}
				<li class="shrink-0">
					<button
						type="button"
						class={[
							'block size-16 overflow-hidden rounded-lg border-2 transition sm:size-20',
							i === index ? 'border-ink' : 'border-transparent opacity-70 hover:opacity-100'
						]}
						aria-label={m.gallery_thumb({ current: i + 1, total })}
						aria-current={i === index ? 'true' : undefined}
						onclick={() => (index = i)}
					>
						<img src={sized(image, 160)} alt="" loading="lazy" class="size-full object-cover" />
					</button>
				</li>
			{/each}
		</ul>
	{/if}
</section>

<Lightbox {images} bind:index bind:open={lightboxOpen} />
