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
		/** Small print under the form (e.g. how to get access). */
		footer,
		children
	}: {
		title: string;
		subtitle?: string;
		waitForClerk?: boolean;
		footer?: Snippet;
		children: Snippet;
	} = $props();

	const clerk = useClerkContext();
	const loading = $derived(waitForClerk && !clerk.isLoaded);
</script>

<svelte:head>
	<title>{title} · {m.app_title()}</title>
</svelte:head>

<div class="grid min-h-dvh bg-white lg:grid-cols-[minmax(0,1.1fr)_minmax(0,1fr)]">
	<!-- The shop itself, so staff know at a glance where they are signing in. -->
	<figure class="relative h-44 overflow-hidden bg-zinc-200 sm:h-56 lg:h-auto">
		<img
			src="/auth/shop.webp"
			alt=""
			width="765"
			height="956"
			fetchpriority="high"
			class="absolute inset-0 size-full object-cover object-[50%_30%]"
		/>
		<!-- Scrim for caption legibility over a busy photo. -->
		<figcaption
			class="absolute inset-x-0 bottom-0 bg-linear-to-t from-zinc-950/65 to-transparent px-6 pt-14 pb-4 text-sm font-medium text-white sm:px-10 lg:px-12 lg:pb-10 lg:text-base"
		>
			{m.auth_shop_caption()}
		</figcaption>
	</figure>

	<main class="flex min-w-0 flex-col px-6 py-6 sm:px-10 lg:px-16 lg:py-10">
		<header class="flex items-center justify-between gap-4">
			<span class="flex items-center gap-2.5">
				<img
					src="/apple-touch-icon.png"
					alt=""
					width="36"
					height="36"
					class="size-9 rounded-lg ring-1 ring-zinc-200"
				/>
				<span class="text-sm font-semibold text-zinc-900">
					AT Creations <span class="font-normal text-zinc-500">{m.auth_wordmark_suffix()}</span>
				</span>
			</span>
			<LocaleSwitcher compact />
		</header>

		<div class="mx-auto flex w-full max-w-sm flex-1 flex-col justify-center py-10 lg:py-16">
			<h1 class="text-3xl font-semibold tracking-tight text-balance text-zinc-900">{title}</h1>
			{#if subtitle}
				<p class="mt-2 text-sm/6 text-pretty text-zinc-600">{subtitle}</p>
			{/if}

			<div class="mt-8">
				{#if loading}
					<!-- Same shape as the widget, so nothing jumps when Clerk arrives. -->
					<div class="space-y-5" role="status" aria-label={m.common_loading()}>
						<div class="h-11 animate-pulse rounded-lg bg-zinc-100"></div>
						<div class="h-px bg-zinc-100"></div>
						<div class="space-y-2">
							<div class="h-4 w-24 animate-pulse rounded bg-zinc-100"></div>
							<div class="h-11 animate-pulse rounded-lg bg-zinc-100"></div>
						</div>
						<div class="h-11 animate-pulse rounded-lg bg-brand-100"></div>
					</div>
				{/if}
				<div class={loading ? 'hidden' : 'animate-fade-in'}>
					{@render children()}
				</div>
			</div>

			{#if footer}
				<div class="mt-8 border-t border-zinc-100 pt-6 text-sm text-zinc-500">
					{@render footer()}
				</div>
			{/if}
		</div>
	</main>
</div>
