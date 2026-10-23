<script lang="ts">
	import { describeError } from '$lib/i18n';
	import { m } from '$lib/paraglide/messages.js';
	import Icon from './Icon.svelte';

	let {
		loading = false,
		error = null,
		fallback,
		onretry
	}: {
		loading?: boolean;
		error?: unknown;
		fallback?: string;
		onretry?: () => void;
	} = $props();

	const described = $derived(error ? describeError(error, fallback) : null);
</script>

{#if loading}
	<div class="flex items-center justify-center gap-3 p-10 text-zinc-500" role="status">
		<span
			class="size-5 animate-spin rounded-full border-2 border-zinc-300 border-t-brand-800"
			aria-hidden="true"
		></span>
		<span>{m.common_loading()}</span>
	</div>
{:else if described}
	<div class="flex flex-col items-center gap-3 p-10 text-center" role="alert">
		<Icon name="alert" size={28} class="text-red-500" />
		<p class="font-medium text-zinc-800">{described.message}</p>
		{#if described.detail}<p class="text-sm text-zinc-500">{described.detail}</p>{/if}
		{#if onretry}
			<button type="button" class="btn-secondary" onclick={onretry}>{m.common_try_again()}</button>
		{/if}
	</div>
{/if}
