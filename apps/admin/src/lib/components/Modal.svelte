<script lang="ts">
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
		/** While busy, Escape and the close button are ignored. */
		busy?: boolean;
		onclose?: () => void;
		children: Snippet;
		footer?: Snippet;
	} = $props();

	let dialog = $state<HTMLDialogElement>();
	let returnFocus: HTMLElement | null = null;
	const uid = $props.id();

	$effect(() => {
		const el = dialog;
		if (!el) return;
		if (open && !el.open) {
			returnFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
			el.showModal();
		} else if (!open && el.open) {
			el.close();
		}
	});

	function requestClose() {
		if (busy) return;
		open = false;
	}

	function handleClose() {
		open = false;
		onclose?.();
		const target = returnFocus;
		returnFocus = null;
		if (target?.isConnected) target.focus();
	}

	const widths = { sm: 'max-w-md', md: 'max-w-lg', lg: 'max-w-2xl' } as const;
</script>

<dialog
	bind:this={dialog}
	aria-labelledby="{uid}-title"
	aria-describedby={description ? `${uid}-desc` : undefined}
	class={[
		'm-auto w-[calc(100%-2rem)] rounded-xl bg-white p-0 text-zinc-900 shadow-2xl',
		widths[size]
	]}
	oncancel={(event) => {
		event.preventDefault();
		requestClose();
	}}
	onclose={handleClose}
>
	{#if open}
		<div class="flex max-h-[85vh] flex-col">
			<header class="flex items-start justify-between gap-4 border-b border-zinc-200 px-5 py-4">
				<div>
					<h2 id="{uid}-title" class="text-lg font-semibold">{title}</h2>
					{#if description}
						<p id="{uid}-desc" class="mt-1 text-sm text-zinc-600">{description}</p>
					{/if}
				</div>
				<button
					type="button"
					class="btn-ghost -mr-2 btn-sm"
					aria-label={m.common_close()}
					disabled={busy}
					onclick={requestClose}
				>
					<Icon name="x" />
				</button>
			</header>
			<div class="overflow-y-auto px-5 py-4">
				{@render children()}
			</div>
			{#if footer}
				<footer
					class="flex flex-wrap justify-end gap-2 border-t border-zinc-200 bg-zinc-50 px-5 py-3"
				>
					{@render footer()}
				</footer>
			{/if}
		</div>
	{/if}
</dialog>
