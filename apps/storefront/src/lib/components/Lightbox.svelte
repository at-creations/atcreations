<script lang="ts">
	import ChevronLeft from '@lucide/svelte/icons/chevron-left';
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
	import X from '@lucide/svelte/icons/x';
	import { m } from '$lib/paraglide/messages';
	import type { GalleryImage } from './gallery';

	interface Props {
		images: GalleryImage[];
		index?: number;
		open?: boolean;
	}

	let { images, index = $bindable(0), open = $bindable(false) }: Props = $props();

	const current = $derived(images[index]);

	// A modal <dialog> provides the focus trap, inert background, Escape-to-close and
	// focus restoration natively. The attachment re-runs whenever `open` changes.
	function modal(node: HTMLDialogElement) {
		if (open && !node.open) node.showModal();
		else if (!open && node.open) node.close();
	}

	function go(delta: number) {
		if (images.length === 0) return;
		index = (index + delta + images.length) % images.length;
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

	function onBackdropClick(event: MouseEvent) {
		// Clicks on the dialog element itself (not its content) are clicks on the backdrop.
		if (event.target === event.currentTarget) open = false;
	}
</script>

<dialog
	{@attach modal}
	aria-label={m.gallery_dialog_label()}
	class="m-0 h-dvh max-h-none w-screen max-w-none bg-black/95 p-0 text-white backdrop:bg-black/80"
	onclose={() => (open = false)}
	onkeydown={onKeydown}
	onclick={onBackdropClick}
>
	{#if open && current}
		<div class="flex h-full flex-col">
			<div class="flex items-center justify-between p-4">
				<p class="text-sm" aria-live="polite">
					{m.gallery_counter({ current: index + 1, total: images.length })}
				</p>
				<button
					type="button"
					class="rounded-full p-2 hover:bg-white/10 focus-visible:outline-white"
					onclick={() => (open = false)}
					aria-label={m.gallery_close()}
				>
					<X class="size-7" />
				</button>
			</div>

			<div class="relative flex min-h-0 flex-1 items-center justify-center px-4 pb-6 sm:px-16">
				{#key index}
					<img
						src={current.url}
						alt={current.alt}
						width={current.width}
						height={current.height}
						class="max-h-full max-w-full animate-fade-in object-contain"
					/>
				{/key}

				{#if images.length > 1}
					<button
						type="button"
						class="absolute top-1/2 left-2 -translate-y-1/2 rounded-full bg-white/10 p-3 hover:bg-white/20 focus-visible:outline-white"
						onclick={() => go(-1)}
						aria-label={m.gallery_prev()}
					>
						<ChevronLeft class="size-6" />
					</button>
					<button
						type="button"
						class="absolute top-1/2 right-2 -translate-y-1/2 rounded-full bg-white/10 p-3 hover:bg-white/20 focus-visible:outline-white"
						onclick={() => go(1)}
						aria-label={m.gallery_next()}
					>
						<ChevronRight class="size-6" />
					</button>
				{/if}
			</div>
		</div>
	{/if}
</dialog>
