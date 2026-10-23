<script lang="ts">
	import { Dialog } from 'bits-ui';
	import type { Snippet } from 'svelte';
	import { m } from '$lib/paraglide/messages.js';
	import Icon from './Icon.svelte';

	let {
		open = $bindable(false),
		title,
		description,
		size = 'md',
		busy = false,
		onclose,
		children,
		footer
	}: {
		open?: boolean;
		title: string;
		description?: string;
		size?: 'sm' | 'md' | 'lg';
		/** While busy, Escape, outside clicks and the close button are ignored. */
		busy?: boolean;
		/** Called when the user dismisses the dialog (Escape, outside click, close button). */
		onclose?: () => void;
		children: Snippet;
		footer?: Snippet;
	} = $props();

	const widths = { sm: 'max-w-md', md: 'max-w-lg', lg: 'max-w-2xl' } as const;
</script>

<Dialog.Root
	bind:open
	onOpenChange={(next) => {
		if (!next) onclose?.();
	}}
>
	<Dialog.Portal>
		<Dialog.Overlay class="fixed inset-0 z-50 animate-fade-in bg-zinc-900/40 backdrop-blur-[2px]" />
		<Dialog.Content
			escapeKeydownBehavior={busy ? 'ignore' : 'close'}
			interactOutsideBehavior={busy ? 'ignore' : 'close'}
			class={[
				'fixed top-1/2 left-1/2 z-50 flex max-h-[85vh] w-[calc(100%-2rem)] -translate-x-1/2 -translate-y-1/2 animate-pop-in flex-col rounded-2xl bg-white text-zinc-900 shadow-popover focus:outline-none',
				widths[size]
			]}
		>
			<header class="flex items-start justify-between gap-4 border-b border-zinc-100 px-5 py-4">
				<div>
					<Dialog.Title class="text-base font-semibold">{title}</Dialog.Title>
					{#if description}
						<Dialog.Description class="mt-1 text-sm text-zinc-500">{description}</Dialog.Description
						>
					{/if}
				</div>
				<Dialog.Close
					class="btn-icon btn-ghost -mr-2 size-8"
					aria-label={m.common_close()}
					disabled={busy}
				>
					<Icon name="x" size={16} />
				</Dialog.Close>
			</header>
			<div class="overflow-y-auto px-5 py-4">
				{@render children()}
			</div>
			{#if footer}
				<footer
					class="flex flex-wrap justify-end gap-2 rounded-b-2xl border-t border-zinc-100 bg-zinc-50/70 px-5 py-3"
				>
					{@render footer()}
				</footer>
			{/if}
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>
