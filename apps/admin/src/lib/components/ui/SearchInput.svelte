<script lang="ts">
	import { m } from '$lib/paraglide/messages.js';
	import Icon from '../Icon.svelte';

	let {
		value = $bindable(''),
		placeholder,
		label = m.common_search(),
		/** Debounce for `onsearch` in ms. */
		delay = 300,
		onsearch,
		class: className = ''
	}: {
		value?: string;
		placeholder?: string;
		label?: string;
		delay?: number;
		/** Called with the trimmed value after typing pauses, or at once when cleared. */
		onsearch?: (value: string) => void;
		class?: string;
	} = $props();

	const uid = $props.id();
	let timer: ReturnType<typeof setTimeout> | undefined;

	function emit(next: string, immediate = false) {
		value = next;
		clearTimeout(timer);
		if (!onsearch) return;
		if (immediate) onsearch(next.trim());
		else timer = setTimeout(() => onsearch?.(next.trim()), delay);
	}

	$effect(() => () => clearTimeout(timer));
</script>

<div class={['relative', className]}>
	<label for="{uid}-q" class="sr-only">{label}</label>
	<Icon
		name="search"
		size={16}
		class="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-zinc-400"
	/>
	<input
		id="{uid}-q"
		type="search"
		class="input pr-8 pl-9 [&::-webkit-search-cancel-button]:hidden"
		maxlength="100"
		{placeholder}
		{value}
		oninput={(e) => emit(e.currentTarget.value)}
		onkeydown={(e) => {
			if (e.key === 'Escape' && value) {
				e.stopPropagation();
				emit('', true);
			}
		}}
	/>
	{#if value}
		<button
			type="button"
			class="absolute top-1/2 right-2 -translate-y-1/2 rounded p-0.5 text-zinc-400 hover:bg-zinc-100 hover:text-zinc-700"
			aria-label={m.common_clear()}
			onclick={() => emit('', true)}
		>
			<Icon name="x" size={14} />
		</button>
	{/if}
</div>
