<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		title,
		description,
		actions,
		footer,
		padded = true,
		class: className = '',
		children
	}: {
		title?: string;
		description?: string;
		/** Right side of the header (links, small buttons). */
		actions?: Snippet;
		/** Muted strip at the bottom (totals, summaries). */
		footer?: Snippet;
		/** Pad the body; turn off for lists and tables that run edge to edge. */
		padded?: boolean;
		class?: string;
		children: Snippet;
	} = $props();

	const uid = $props.id();
</script>

<section class={['card', className]} aria-labelledby={title ? `${uid}-title` : undefined}>
	{#if title || actions}
		<header class="card-header">
			<div class="min-w-0">
				{#if title}<h2 id="{uid}-title" class="card-title">{title}</h2>{/if}
				{#if description}<p class="mt-0.5 text-xs text-zinc-500">{description}</p>{/if}
			</div>
			{#if actions}<div class="flex shrink-0 items-center gap-2">{@render actions()}</div>{/if}
		</header>
	{/if}
	<div class={[padded && 'p-5']}>
		{@render children()}
	</div>
	{#if footer}
		<footer class="rounded-b-2xl border-t border-zinc-100 bg-zinc-50/70 px-5 py-3 text-sm">
			{@render footer()}
		</footer>
	{/if}
</section>
