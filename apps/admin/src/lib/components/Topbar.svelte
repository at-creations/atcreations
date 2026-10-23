<script lang="ts">
	import { UserButton } from 'svelte-clerk';
	import { storefrontUrl } from '$lib/links';
	import { m } from '$lib/paraglide/messages.js';
	import Icon from './Icon.svelte';
	import LocaleSwitcher from './LocaleSwitcher.svelte';

	let {
		drawerOpen,
		ontoggledrawer,
		onsearch
	}: { drawerOpen: boolean; ontoggledrawer: () => void; onsearch: () => void } = $props();

	const siteUrl = storefrontUrl();
	const isMac = typeof navigator !== 'undefined' && /Mac|iPhone|iPad/.test(navigator.platform);
</script>

<header
	class="sticky top-0 z-30 flex h-14 items-center gap-2 border-b border-zinc-200 bg-white px-3 sm:px-4 lg:px-8"
>
	<button
		type="button"
		class="btn-icon btn-ghost lg:hidden"
		aria-controls="admin-sidebar"
		aria-expanded={drawerOpen}
		aria-label={drawerOpen ? m.nav_close() : m.nav_open()}
		onclick={ontoggledrawer}
	>
		<Icon name={drawerOpen ? 'x' : 'menu'} size={20} />
	</button>

	<button
		type="button"
		class="flex h-9 w-full max-w-64 items-center gap-2 rounded-lg bg-zinc-100 px-3 text-sm text-zinc-500 transition-colors hover:bg-zinc-200/70"
		aria-haspopup="dialog"
		aria-keyshortcuts={isMac ? 'Meta+K' : 'Control+K'}
		onclick={onsearch}
	>
		<Icon name="search" size={16} />
		<span class="flex-1 text-left">{m.cmd_open()}</span>
		<kbd class="kbd hidden bg-white sm:inline-flex" aria-hidden="true"
			>{isMac ? '⌘K' : 'Ctrl K'}</kbd
		>
	</button>

	<div class="ml-auto flex items-center gap-1">
		{#if siteUrl}
			<!-- eslint-disable svelte/no-navigation-without-resolve -- external storefront URL -->
			<a
				href={siteUrl}
				target="_blank"
				rel="noopener noreferrer"
				class="btn-icon btn-ghost"
				title={m.nav_view_store()}
				aria-label={m.nav_view_store()}
			>
				<Icon name="external" size={18} />
			</a>
			<!-- eslint-enable svelte/no-navigation-without-resolve -->
		{/if}
		<LocaleSwitcher compact />
		<div class="ml-1 flex size-9 items-center justify-center">
			<UserButton />
		</div>
	</div>
</header>
