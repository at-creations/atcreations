<script lang="ts">
	import Icon from '$lib/components/Icon.svelte';
	import { m } from '$lib/paraglide/messages.js';
	import type { UploadItem } from './items';

	let {
		item,
		oncancel,
		onretry,
		canRetry = () => true,
		class: className = ''
	}: {
		item: UploadItem;
		oncancel?: (item: UploadItem) => void;
		onretry?: (item: UploadItem) => void;
		/** Whether this item can actually be retried (files rejected up front cannot). */
		canRetry?: (item: UploadItem) => boolean;
		class?: string;
	} = $props();

	const retryable = $derived(item.canRetry && !!onretry && canRetry(item));

	const failed = $derived(item.phase === 'failed');
	const indeterminate = $derived(
		item.phase === 'queued' || item.phase === 'waiting' || item.phase === 'retrying'
	);
	// Overall bar: sending is the first half, processing the second.
	const bar = $derived(
		item.phase === 'sending'
			? item.percent / 2
			: item.phase === 'processing'
				? 50 + item.percent / 2
				: 0
	);
</script>

<div
	class={[
		'relative flex aspect-square flex-col overflow-hidden rounded-xl ring-1',
		failed ? 'bg-rose-50 ring-rose-200' : 'bg-zinc-50 ring-zinc-200',
		className
	]}
	role="group"
	aria-label="{item.name}: {item.label}"
>
	{#if item.preview && !failed}
		<img
			src={item.preview}
			alt=""
			class="absolute inset-0 size-full object-cover opacity-40 blur-[1px]"
		/>
	{/if}
	<div class="relative flex flex-1 flex-col items-center justify-center gap-1.5 p-2 text-center">
		{#if failed}
			<Icon name="alert" size={20} class="text-rose-500" />
		{:else if item.phase === 'sending'}
			<Icon name="upload" size={20} class="text-brand-600" />
		{:else}
			<span
				class="size-5 animate-spin rounded-full border-2 border-brand-200 border-t-brand-600"
				aria-hidden="true"
			></span>
		{/if}
		<span
			class={['text-[11px] leading-tight font-medium', failed ? 'text-rose-700' : 'text-zinc-700']}
		>
			{item.label}
		</span>
		{#if item.error}
			<span class="line-clamp-2 text-[10px] leading-tight text-rose-600" title={item.error}
				>{item.error}</span
			>
		{/if}
		<span class="w-full truncate text-[10px] text-zinc-400" title={item.name}>{item.name}</span>
	</div>

	{#if !failed}
		<div class="relative h-1 bg-zinc-200/70">
			{#if indeterminate}
				<div class="absolute inset-y-0 w-1/3 animate-indeterminate bg-brand-300"></div>
			{:else}
				<div class="h-full bg-brand-400 transition-[width] duration-300" style:width="{bar}%"></div>
			{/if}
		</div>
	{/if}

	{#if retryable || (item.canCancel && oncancel)}
		<div class="absolute top-1.5 right-1.5 flex gap-1">
			{#if retryable && onretry}
				<button
					type="button"
					class="btn-icon btn-ghost size-6 bg-white/90 shadow-sm"
					aria-label="{m.up_retry()}: {item.name}"
					onclick={() => onretry(item)}
				>
					<Icon name="history" size={12} />
				</button>
			{/if}
			{#if item.canCancel && oncancel}
				<button
					type="button"
					class="btn-icon btn-ghost size-6 bg-white/90 shadow-sm hover:text-rose-600"
					aria-label="{failed ? m.up_dismiss() : m.up_cancel()}: {item.name}"
					onclick={() => oncancel(item)}
				>
					<Icon name="x" size={12} />
				</button>
			{/if}
		</div>
	{/if}
</div>
