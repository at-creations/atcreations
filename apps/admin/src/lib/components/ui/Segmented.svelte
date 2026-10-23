<script lang="ts" module>
	import type { IconName } from '../Icon.svelte';

	export interface SegmentedOption<V extends string> {
		value: V;
		label: string;
		icon?: IconName;
		/** Small coloured dot before the label (e.g. a status colour). */
		dot?: string;
	}
</script>

<script lang="ts" generics="T extends string">
	import { ToggleGroup } from 'bits-ui';
	import Icon from '../Icon.svelte';

	let {
		options,
		value = $bindable(),
		label,
		/** Show icons only (labels become tooltips and accessible names). */
		iconOnly = false,
		full = false,
		disabled = false,
		onchange
	}: {
		options: SegmentedOption<T>[];
		value: T;
		/** Accessible name of the group. */
		label: string;
		iconOnly?: boolean;
		full?: boolean;
		disabled?: boolean;
		onchange?: (value: T) => void;
	} = $props();
</script>

<ToggleGroup.Root
	type="single"
	bind:value={
		() => value,
		(next) => {
			// A segmented control always has a selection: clicking the active item changes nothing.
			if (!next || next === value) return;
			value = next as T;
			onchange?.(next as T);
		}
	}
	{disabled}
	aria-label={label}
	class={[
		'inline-flex rounded-lg bg-zinc-100 p-0.5 data-[disabled]:pointer-events-none data-[disabled]:opacity-60',
		full && 'flex w-full'
	]}
>
	{#each options as option (option.value)}
		<ToggleGroup.Item
			value={option.value}
			aria-label={iconOnly ? option.label : undefined}
			title={iconOnly ? option.label : undefined}
			class={[
				'flex h-8 items-center justify-center gap-1.5 rounded-md text-sm font-medium text-zinc-500 transition-all hover:text-zinc-800 focus-visible:ring-2 focus-visible:ring-brand-300 focus-visible:outline-none data-[state=on]:bg-white data-[state=on]:text-zinc-900 data-[state=on]:shadow-sm',
				iconOnly ? 'w-8' : 'px-3',
				full && 'flex-1'
			]}
		>
			{#if option.dot}<span class={['size-2 rounded-full', option.dot]} aria-hidden="true"
				></span>{/if}
			{#if option.icon}<Icon name={option.icon} size={16} />{/if}
			{#if !iconOnly}{option.label}{/if}
		</ToggleGroup.Item>
	{/each}
</ToggleGroup.Root>
