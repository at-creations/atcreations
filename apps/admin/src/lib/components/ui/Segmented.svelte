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

<div
	role="radiogroup"
	aria-label={label}
	class={[
		'inline-flex rounded-lg bg-zinc-100 p-0.5',
		full && 'flex w-full',
		disabled && 'pointer-events-none opacity-60'
	]}
>
	{#each options as option (option.value)}
		{@const active = option.value === value}
		<button
			type="button"
			role="radio"
			aria-checked={active}
			aria-label={iconOnly ? option.label : undefined}
			title={iconOnly ? option.label : undefined}
			tabindex={active ? 0 : -1}
			{disabled}
			class={[
				'flex h-8 items-center justify-center gap-1.5 rounded-md text-sm font-medium transition-all',
				iconOnly ? 'w-8' : 'px-3',
				full && 'flex-1',
				active ? 'bg-white text-zinc-900 shadow-sm' : 'text-zinc-500 hover:text-zinc-800'
			]}
			onclick={() => {
				value = option.value;
				onchange?.(option.value);
			}}
			onkeydown={(e) => {
				if (e.key !== 'ArrowRight' && e.key !== 'ArrowLeft') return;
				e.preventDefault();
				const i = options.findIndex((o) => o.value === value);
				const step = e.key === 'ArrowRight' ? 1 : -1;
				const next = options[(i + step + options.length) % options.length];
				if (!next) return;
				value = next.value;
				onchange?.(next.value);
				(e.currentTarget.parentElement?.children[options.indexOf(next)] as HTMLElement)?.focus();
			}}
		>
			{#if option.dot}<span class={['size-2 rounded-full', option.dot]} aria-hidden="true"
				></span>{/if}
			{#if option.icon}<Icon name={option.icon} size={16} />{/if}
			{#if !iconOnly}{option.label}{/if}
		</button>
	{/each}
</div>
