<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		checked = $bindable(false),
		label,
		description,
		disabled = false,
		icon
	}: {
		checked?: boolean;
		label: string;
		description?: string;
		disabled?: boolean;
		/** Optional leading icon next to the label. */
		icon?: Snippet;
	} = $props();
</script>

<label
	class={[
		'flex items-center justify-between gap-4',
		disabled ? 'cursor-not-allowed opacity-60' : 'cursor-pointer'
	]}
>
	<span class="min-w-0">
		<span class="flex items-center gap-1.5 text-sm font-medium text-zinc-900">
			{@render icon?.()}{label}
		</span>
		{#if description}<span class="mt-0.5 block text-xs text-zinc-500">{description}</span>{/if}
	</span>
	<input type="checkbox" role="switch" class="peer sr-only" {disabled} bind:checked />
	<span
		class="relative h-5 w-9 shrink-0 rounded-full bg-zinc-200 transition-colors peer-checked:bg-brand-400 peer-focus-visible:ring-2 peer-focus-visible:ring-brand-300 after:absolute after:top-0.5 after:left-0.5 after:size-4 after:rounded-full after:bg-white after:shadow-sm after:transition-transform peer-checked:after:translate-x-4"
		aria-hidden="true"
	></span>
</label>
