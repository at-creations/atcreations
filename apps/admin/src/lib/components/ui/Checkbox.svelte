<script lang="ts">
	import { Checkbox, Label } from 'bits-ui';
	import Icon from '../Icon.svelte';

	let {
		checked = $bindable(false),
		label,
		description,
		disabled = false
	}: {
		checked?: boolean;
		label: string;
		description?: string;
		disabled?: boolean;
	} = $props();

	const uid = $props.id();
</script>

<div class={['flex items-start gap-2.5', disabled && 'opacity-60']}>
	<Checkbox.Root
		id="{uid}-cb"
		bind:checked
		{disabled}
		aria-describedby={description ? `${uid}-desc` : undefined}
		class="mt-0.5 flex size-4 shrink-0 items-center justify-center rounded border border-zinc-300 bg-white text-brand-900 transition-colors focus-visible:ring-2 focus-visible:ring-brand-300 focus-visible:outline-none disabled:cursor-not-allowed data-[state=checked]:border-brand-400 data-[state=checked]:bg-brand-300"
	>
		{#snippet children({ checked: isChecked })}
			{#if isChecked}<Icon name="check" size={12} />{/if}
		{/snippet}
	</Checkbox.Root>
	<span>
		<Label.Root for="{uid}-cb" class="cursor-pointer text-sm text-zinc-800">{label}</Label.Root>
		{#if description}<span id="{uid}-desc" class="block text-xs text-zinc-500">{description}</span
			>{/if}
	</span>
</div>
