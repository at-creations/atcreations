<script lang="ts">
	import { Label, Switch } from 'bits-ui';
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

	const uid = $props.id();
</script>

<div class={['flex items-center justify-between gap-4', disabled && 'opacity-60']}>
	<span class="min-w-0">
		<Label.Root
			for="{uid}-switch"
			class={[
				'flex items-center gap-1.5 text-sm font-medium text-zinc-900',
				disabled ? 'cursor-not-allowed' : 'cursor-pointer'
			]}
		>
			{@render icon?.()}{label}
		</Label.Root>
		{#if description}
			<span id="{uid}-desc" class="mt-0.5 block text-xs text-zinc-500">{description}</span>
		{/if}
	</span>
	<Switch.Root
		id="{uid}-switch"
		bind:checked
		{disabled}
		aria-describedby={description ? `${uid}-desc` : undefined}
		class="peer inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full bg-zinc-200 px-0.5 transition-colors focus-visible:ring-2 focus-visible:ring-brand-300 focus-visible:outline-none disabled:cursor-not-allowed data-[state=checked]:bg-brand-400"
	>
		<Switch.Thumb
			class="pointer-events-none block size-4 rounded-full bg-white shadow-sm transition-transform data-[state=checked]:translate-x-4"
		/>
	</Switch.Root>
</div>
