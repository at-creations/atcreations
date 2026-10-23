<script lang="ts">
	import { m } from '$lib/paraglide/messages.js';
	import Icon from '../Icon.svelte';

	let { label, onremove }: { label: string; onremove?: () => void } = $props();
</script>

<span
	class="inline-flex items-center gap-1 rounded-full bg-brand-100 py-0.5 pr-1 pl-2.5 text-xs font-medium text-brand-900"
>
	{label}
	{#if onremove}
		<button
			type="button"
			class="rounded-full p-0.5 text-brand-700 hover:bg-brand-200"
			aria-label="{m.common_clear()}: {label}"
			onclick={onremove}
		>
			<Icon name="x" size={12} />
		</button>
	{/if}
</span>
