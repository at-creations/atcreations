<script lang="ts">
	import type { Snippet } from 'svelte';
	import { useClerkContext } from 'svelte-clerk';
	import { m } from '$lib/paraglide/messages.js';
	import LocaleSwitcher from './LocaleSwitcher.svelte';

	let {
		title,
		subtitle,
		/** Hold a placeholder until Clerk has loaded (for pages that render a Clerk widget). */
		waitForClerk = true,
		children
	}: { title: string; subtitle?: string; waitForClerk?: boolean; children: Snippet } = $props();

	const clerk = useClerkContext();
	const loading = $derived(waitForClerk && !clerk.isLoaded);
</script>

<svelte:head>
	<title>{title} · {m.app_title()}</title>
</svelte:head>

<main class="flex min-h-screen flex-col items-center justify-center px-4 py-10">
	<div class="w-full max-w-sm">
		<div class="mb-8 flex flex-col items-center text-center">
			<img src="/apple-touch-icon.png" alt="" class="size-11 rounded-xl" />
			<h1 class="mt-5 text-xl font-semibold text-zinc-900">{title}</h1>
			{#if subtitle}<p class="mt-1 text-sm text-zinc-500">{subtitle}</p>{/if}
		</div>

		<div class="card p-6 sm:p-8">
			{#if loading}
				<!-- Same shape as the widget, so nothing jumps when Clerk arrives. -->
				<div class="space-y-4" role="status" aria-label={m.common_loading()}>
					<div class="h-10 animate-pulse rounded-lg bg-zinc-100"></div>
					<div class="h-px bg-zinc-100"></div>
					<div class="h-10 animate-pulse rounded-lg bg-zinc-100"></div>
					<div class="h-10 animate-pulse rounded-lg bg-zinc-200"></div>
				</div>
			{/if}
			<div class={loading ? 'hidden' : 'animate-pop-in'}>
				{@render children()}
			</div>
		</div>

		<div class="mt-6 flex justify-center">
			<LocaleSwitcher />
		</div>
	</div>
</main>
