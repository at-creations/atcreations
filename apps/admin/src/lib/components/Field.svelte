<script lang="ts">
	import type { Snippet } from 'svelte';
	import { m } from '$lib/paraglide/messages.js';

	export interface FieldControlProps {
		id: string;
		'aria-describedby'?: string;
		'aria-invalid'?: 'true';
		'aria-required'?: 'true';
	}

	let {
		label,
		error,
		hint,
		required = false,
		optional = false,
		class: className = '',
		children
	}: {
		label: string;
		error?: string | null;
		hint?: string;
		required?: boolean;
		optional?: boolean;
		class?: string;
		/** Receives the attributes the control must spread to stay accessible. */
		children: Snippet<[FieldControlProps]>;
	} = $props();

	const id = $props.id();
	const describedBy = $derived(
		[error ? `${id}-error` : '', hint ? `${id}-hint` : ''].filter(Boolean).join(' ') || undefined
	);
	const control = $derived<FieldControlProps>({
		id,
		'aria-describedby': describedBy,
		'aria-invalid': error ? 'true' : undefined,
		'aria-required': required ? 'true' : undefined
	});
</script>

<div class={className}>
	<label class="label" for={id}>
		{label}
		{#if required}<span class="text-red-600" aria-hidden="true">*</span>{/if}
		{#if optional}<span class="text-xs font-normal text-slate-500">({m.common_optional()})</span
			>{/if}
	</label>
	{@render children(control)}
	{#if hint}
		<p id="{id}-hint" class="mt-1 text-xs text-slate-500">{hint}</p>
	{/if}
	{#if error}
		<p id="{id}-error" class="mt-1 text-xs font-medium text-red-600">{error}</p>
	{/if}
</div>
