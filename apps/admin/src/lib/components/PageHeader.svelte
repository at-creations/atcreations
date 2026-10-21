<script lang="ts">
	import type { Snippet } from 'svelte';

	let { title, subtitle, actions }: { title: string; subtitle?: string; actions?: Snippet } =
		$props();
</script>

<svelte:head>
	<title>{title} · AT Creations Admin</title>
</svelte:head>

<header class="mb-6 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
	<div>
		<h1 class="text-2xl font-bold text-slate-900">{title}</h1>
		{#if subtitle}<p class="mt-1 text-sm text-slate-600">{subtitle}</p>{/if}
	</div>
	{#if actions}
		<div class="flex flex-wrap gap-2">{@render actions()}</div>
	{/if}
</header>
