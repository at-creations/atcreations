<script lang="ts">
	import type { Snippet } from 'svelte';
	import Icon from './Icon.svelte';

	let {
		title,
		subtitle,
		back,
		meta,
		actions
	}: {
		title: string;
		subtitle?: string;
		/** Link to the parent list ("← Products"). */
		back?: { href: string; label: string };
		/** Inline badges next to the title (status, featured…). */
		meta?: Snippet;
		actions?: Snippet;
	} = $props();
</script>

<svelte:head>
	<title>{title} · AT Creations Admin</title>
</svelte:head>

<header class="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
	<div class="flex min-w-0 items-start gap-3">
		{#if back}
			<!-- eslint-disable svelte/no-navigation-without-resolve -- resolved or external URL -->
			<a
				href={back.href}
				class="btn-secondary mt-0.5 size-8 shrink-0 px-0"
				aria-label={back.label}
				title={back.label}
			>
				<Icon name="left" size={16} />
			</a>
			<!-- eslint-enable svelte/no-navigation-without-resolve -->
		{/if}
		<div class="min-w-0">
			<div class="flex flex-wrap items-center gap-x-3 gap-y-1">
				<h1 class="truncate text-xl font-semibold tracking-tight text-zinc-900 sm:text-2xl">
					{title}
				</h1>
				{@render meta?.()}
			</div>
			{#if subtitle}<p class="mt-1 text-sm text-zinc-500">{subtitle}</p>{/if}
		</div>
	</div>
	{#if actions}
		<div class="flex shrink-0 flex-wrap items-center gap-2">{@render actions()}</div>
	{/if}
</header>
