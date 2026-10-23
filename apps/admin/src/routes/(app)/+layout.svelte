<script lang="ts">
	import { createQuery, QueryClientProvider } from '@tanstack/svelte-query';
	import { Dialog } from 'bits-ui';
	import { untrack } from 'svelte';
	import { MediaQuery } from 'svelte/reactivity';
	import { useClerkContext } from 'svelte-clerk';
	import { afterNavigate } from '$app/navigation';
	import { page } from '$app/state';
	import { ApiError, configureApi } from '$lib/api';
	import CommandPalette from '$lib/components/CommandPalette.svelte';
	import Icon from '$lib/components/Icon.svelte';
	import LoadState from '$lib/components/LoadState.svelte';
	import Sidebar from '$lib/components/Sidebar.svelte';
	import Topbar from '$lib/components/Topbar.svelte';
	import { m } from '$lib/paraglide/messages.js';
	import { createAdminQueryClient, meQuery } from '$lib/queries';
	import { clearCachedMe, readCachedMe, writeCachedMe } from '$lib/state/me-cache';
	import { AdminSession, setAdminSession } from '$lib/state/session.svelte';
	import { toast } from '$lib/state/toast.svelte';
	import { safeRedirectPath } from '$lib/utils/redirect';
	import type { LayoutProps } from './$types';

	let { data, children }: LayoutProps = $props();

	const clerk = useClerkContext();
	// One cache per browser session (this layout lives for the whole signed-in session; SSR is off).
	const queryClient = createAdminQueryClient();

	// The server already verified the session (hooks.server.ts); its user id lets us paint the
	// shell from the last known /me while Clerk's script is still loading.
	const serverUserId = untrack(
		() => (data.initialState as { userId?: string | null } | undefined)?.userId ?? null
	);

	let drawerOpen = $state(false);
	let paletteOpen = $state(false);
	let leaving = false;

	const isDesktop = new MediaQuery('min-width: 1024px');

	function signInUrl() {
		const target = safeRedirectPath(`${page.url.pathname}${page.url.search}`);
		return target === '/' ? '/sign-in' : `/sign-in?redirect_url=${encodeURIComponent(target)}`;
	}

	function goToSignIn() {
		if (leaving) return;
		leaving = true;
		clearCachedMe();
		// Full reload so the server-side guard and Clerk handshake run again.
		window.location.assign(signInUrl());
	}

	/** The API rejected our token: end the Clerk session too, or /sign-in would bounce us back. */
	async function endSession() {
		if (leaving) return;
		leaving = true;
		queryClient.clear();
		clearCachedMe();
		try {
			await clerk.clerk?.signOut({ redirectUrl: signInUrl() });
		} catch {
			window.location.assign(signInUrl());
		}
	}

	// Requests made before Clerk has loaded wait for it instead of going out without a token
	// (which would bounce the user to sign-in). Clerk caches and refreshes the token itself.
	let markClerkReady: () => void = () => {};
	const clerkReady = new Promise<void>((resolve) => (markClerkReady = resolve));
	$effect(() => {
		if (clerk.isLoaded) markClerkReady();
	});

	configureApi({
		getToken: async () => {
			await Promise.race([clerkReady, new Promise((r) => setTimeout(r, 15_000))]);
			return (await clerk.session?.getToken()) ?? null;
		},
		onUnauthorized: () => {
			if (leaving) return;
			toast.error(m.error_session_expired());
			void endSession();
		}
	});

	const me = createQuery(
		() => ({
			...meQuery(),
			enabled: clerk.isLoaded && !!clerk.auth.userId,
			initialData: () => readCachedMe(serverUserId),
			// Cached copy is shown at once but always re-checked with the server.
			initialDataUpdatedAt: 0
		}),
		() => queryClient
	);
	setAdminSession(new AdminSession(() => me.data ?? null));

	$effect(() => {
		const current = me.data;
		const userId = clerk.auth.userId;
		if (current && userId && me.isSuccess && me.dataUpdatedAt > 0) writeCachedMe(userId, current);
	});

	const status = $derived.by((): 'loading' | 'ready' | 'noaccess' | 'error' => {
		const error = me.error;
		if (error instanceof ApiError && error.status === 403) return 'noaccess';
		if (me.data) return 'ready';
		// 401 keeps the skeleton: onUnauthorized is already redirecting to sign-in.
		if (me.isError && !(error instanceof ApiError && error.status === 401)) return 'error';
		return 'loading';
	});

	// Drop every cached response when the Clerk user signs out or changes, so no data survives
	// a user switch.
	let lastUserId: string | null | undefined;
	$effect(() => {
		if (!clerk.isLoaded) return;
		const userId = clerk.auth.userId ?? null;
		untrack(() => {
			if (lastUserId !== undefined && lastUserId !== userId) {
				queryClient.clear();
				clearCachedMe();
			}
			lastUserId = userId;
			if (!userId) {
				queryClient.clear();
				goToSignIn();
			}
		});
	});

	afterNavigate(() => {
		drawerOpen = false;
	});

	async function signOut() {
		queryClient.clear();
		clearCachedMe();
		await clerk.clerk?.signOut({ redirectUrl: '/sign-in' });
	}
</script>

{#snippet skeleton()}
	<div class="min-h-screen lg:pl-60" role="status" aria-label={m.auth_verifying()}>
		<div class="fixed inset-y-0 left-0 hidden w-60 border-r border-zinc-200 bg-white p-4 lg:block">
			<div class="flex items-center gap-2.5">
				<img src="/apple-touch-icon.png" alt="" class="size-8 rounded-lg" />
				<div class="h-3.5 w-28 animate-pulse rounded bg-zinc-200"></div>
			</div>
			<div class="mt-8 space-y-2.5">
				{#each [70, 55, 62, 48, 66, 52] as w, i (i)}
					<div class="h-3 animate-pulse rounded bg-zinc-100" style:width="{w}%"></div>
				{/each}
			</div>
		</div>
		<div class="h-14 border-b border-zinc-200 bg-white"></div>
		<div class="mx-auto max-w-7xl space-y-6 p-4 sm:p-6 lg:p-8">
			<div class="h-7 w-56 animate-pulse rounded bg-zinc-200"></div>
			<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
				{#each [0, 1, 2, 3] as i (i)}
					<div class="h-28 animate-pulse card bg-white"></div>
				{/each}
			</div>
			<div class="h-72 animate-pulse card"></div>
		</div>
		<span class="sr-only">{m.auth_verifying()}</span>
	</div>
{/snippet}

<QueryClientProvider client={queryClient}>
	{#if status === 'loading'}
		{@render skeleton()}
	{:else if status === 'noaccess'}
		<main class="flex min-h-screen items-center justify-center p-4">
			<div class="w-full max-w-md card p-8 text-center">
				<span
					class="mx-auto flex size-12 items-center justify-center rounded-full bg-zinc-100 text-zinc-500"
				>
					<Icon name="shield" size={24} />
				</span>
				<h1 class="mt-4 text-lg font-semibold">{m.noaccess_title()}</h1>
				<p class="mt-2 text-sm text-zinc-600">{m.noaccess_body()}</p>
				{#if clerk.user?.primaryEmailAddress?.emailAddress}
					<p class="mt-4 text-sm text-zinc-500">
						{m.noaccess_signed_in_as({ email: clerk.user.primaryEmailAddress.emailAddress })}
					</p>
				{/if}
				<button type="button" class="btn-primary mt-6" onclick={signOut}>{m.sign_out()}</button>
			</div>
		</main>
	{:else if status === 'error'}
		<main class="flex min-h-screen items-center justify-center p-4">
			<div class="w-full max-w-md card">
				<LoadState error={me.error} onretry={() => me.refetch()} />
			</div>
		</main>
	{:else}
		<a
			href="#main-content"
			class="sr-only z-50 rounded bg-white px-3 py-2 focus:not-sr-only focus:fixed focus:top-2 focus:left-2"
			>{m.skip_to_content()}</a
		>
		<div class="min-h-screen lg:pl-60">
			{#if isDesktop.current}
				<aside id="admin-sidebar" class="fixed inset-y-0 left-0 z-40 w-60">
					<Sidebar />
				</aside>
			{:else}
				<!-- Phones and tablets: the sidebar is a modal sheet (focus trap, Escape, outside click). -->
				<Dialog.Root bind:open={drawerOpen}>
					<Dialog.Portal>
						<Dialog.Overlay class="fixed inset-0 z-50 bg-zinc-900/40 backdrop-blur-[1px]" />
						<Dialog.Content
							id="admin-sidebar"
							class="fixed inset-y-0 left-0 z-50 w-72 max-w-[85vw] animate-slide-in shadow-popover focus:outline-none"
						>
							<Dialog.Title class="sr-only">{m.nav_main()}</Dialog.Title>
							<Sidebar onnavigate={() => (drawerOpen = false)} />
							<Dialog.Close
								class="btn-icon btn-ghost absolute top-2.5 right-2.5 size-8"
								aria-label={m.nav_close()}
							>
								<Icon name="x" size={18} />
							</Dialog.Close>
						</Dialog.Content>
					</Dialog.Portal>
				</Dialog.Root>
			{/if}

			<Topbar
				{drawerOpen}
				ontoggledrawer={() => (drawerOpen = !drawerOpen)}
				onsearch={() => (paletteOpen = true)}
			/>

			<main
				id="main-content"
				tabindex="-1"
				class="mx-auto max-w-7xl px-4 py-6 focus:outline-none sm:px-6 lg:px-8 lg:py-8"
			>
				{@render children()}
			</main>
		</div>
		<CommandPalette bind:open={paletteOpen} />
	{/if}

	{#if import.meta.env.DEV}
		{#await import('@tanstack/svelte-query-devtools') then { SvelteQueryDevtools }}
			<SvelteQueryDevtools buttonPosition="bottom-right" />
		{/await}
	{/if}
</QueryClientProvider>
