<script lang="ts" module>
	export interface SelectItem {
		value: string;
		label: string;
		disabled?: boolean;
	}
</script>

<script lang="ts">
	import { Select } from 'bits-ui';
	import type { FieldControlProps } from '../Field.svelte';
	import Icon from '../Icon.svelte';

	let {
		items,
		value = $bindable(''),
		onchange,
		label,
		placeholder,
		control,
		disabled = false,
		size = 'md',
		class: className = '',
		/** Label shown in the trigger instead of the item label (e.g. "Sort: Newest"). */
		display
	}: {
		items: SelectItem[];
		value?: string;
		onchange?: (value: string) => void;
		/** Accessible name when there is no visible <label>. */
		label?: string;
		placeholder?: string;
		/** Accessibility attributes from <Field> (id, aria-describedby…). */
		control?: FieldControlProps;
		disabled?: boolean;
		size?: 'sm' | 'md';
		class?: string;
		display?: (item: SelectItem | undefined) => string;
	} = $props();

	const selected = $derived(items.find((i) => i.value === value));
	const text = $derived(display ? display(selected) : (selected?.label ?? placeholder ?? ''));
</script>

<Select.Root type="single" bind:value {items} {disabled} onValueChange={(v) => onchange?.(v)}>
	<Select.Trigger
		{...control}
		aria-label={label}
		class={[
			'flex w-full items-center justify-between gap-2 rounded-lg border border-zinc-300 bg-white text-left shadow-xs transition-colors hover:border-zinc-400 focus:border-brand-400 focus:ring-3 focus:ring-brand-200/70 focus:outline-none disabled:cursor-not-allowed disabled:bg-zinc-50 disabled:text-zinc-500 aria-invalid:border-red-500 data-[state=open]:border-brand-400 data-[state=open]:ring-3 data-[state=open]:ring-brand-200/70',
			size === 'sm' ? 'h-8 px-2.5 text-xs' : 'h-9 px-3 text-sm',
			className
		]}
	>
		<span class={['truncate', !selected && !display && 'text-zinc-400']}>{text}</span>
		<Icon name="down" size={14} class="text-zinc-400" />
	</Select.Trigger>
	<Select.Portal>
		<Select.Content
			sideOffset={4}
			class="z-[60] max-h-72 min-w-(--bits-select-anchor-width) animate-pop-in overflow-hidden rounded-xl border border-zinc-200 bg-white p-1 shadow-popover focus:outline-none"
		>
			<Select.Viewport>
				{#each items as item (item.value)}
					<Select.Item
						value={item.value}
						label={item.label}
						disabled={item.disabled}
						class="flex h-8 cursor-pointer items-center justify-between gap-3 rounded-lg px-2.5 text-sm text-zinc-700 outline-none select-none data-[disabled]:cursor-not-allowed data-[disabled]:opacity-50 data-[highlighted]:bg-brand-50 data-[highlighted]:text-brand-900"
					>
						{#snippet children({ selected: isSelected })}
							<span class="truncate">{item.label}</span>
							{#if isSelected}<Icon name="check" size={14} class="text-brand-600" />{/if}
						{/snippet}
					</Select.Item>
				{/each}
			</Select.Viewport>
		</Select.Content>
	</Select.Portal>
</Select.Root>
