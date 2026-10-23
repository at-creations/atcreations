<script lang="ts">
	import type { FieldControlProps } from '../Field.svelte';
	import Icon from '../Icon.svelte';

	let {
		value = $bindable(null),
		min = 0,
		max = Number.MAX_SAFE_INTEGER,
		step = 1,
		control,
		decreaseLabel,
		increaseLabel
	}: {
		value?: number | null;
		min?: number;
		max?: number;
		step?: number;
		/** Accessibility attributes from <Field>. */
		control?: FieldControlProps;
		decreaseLabel: string;
		increaseLabel: string;
	} = $props();

	function bump(delta: number) {
		const current = Number.isFinite(value) ? (value as number) : min;
		value = Math.min(max, Math.max(min, Math.round(current + delta)));
	}
</script>

<div
	class="flex h-9 items-stretch overflow-hidden rounded-lg border border-zinc-300 bg-white shadow-xs focus-within:border-brand-400 focus-within:ring-3 focus-within:ring-brand-200/70"
>
	<button
		type="button"
		class="flex w-9 items-center justify-center text-zinc-500 hover:bg-zinc-50 hover:text-zinc-900 disabled:opacity-40"
		aria-label={decreaseLabel}
		disabled={(value ?? min) <= min}
		onclick={() => bump(-step)}
	>
		<Icon name="minus" size={14} />
	</button>
	<input
		{...control}
		type="number"
		inputmode="numeric"
		{min}
		{max}
		{step}
		class="w-full min-w-0 [appearance:textfield] border-x border-zinc-200 text-center text-sm tabular-nums focus:outline-none [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
		bind:value
	/>
	<button
		type="button"
		class="flex w-9 items-center justify-center text-zinc-500 hover:bg-zinc-50 hover:text-zinc-900 disabled:opacity-40"
		aria-label={increaseLabel}
		disabled={(value ?? min) >= max}
		onclick={() => bump(step)}
	>
		<Icon name="plus" size={14} />
	</button>
</div>
