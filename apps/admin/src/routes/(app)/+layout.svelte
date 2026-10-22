<script lang="ts">
	import { createQuery, QueryClientProvider } from '@tanstack/svelte-query';
	import { untrack } from 'svelte';
	import { MediaQuery } from 'svelte/reactivity';
	import { useClerkContext } from 'svelte-clerk';
	import { afterNavigate } from '$app/navigation';
	import { page } from '$app/state';
	import { ApiError, configureApi } from '$lib/api';
	import Icon from '$lib/components/Icon.svelte';
	import LoadState from '$lib/components/LoadState.svelte';
	import Sidebar from '$lib/components/Sidebar.svelte';
	import { m } from '$lib/paraglide/messages.js';
	import { createAdminQueryClient, meQuery } from '$lib/queries';
	import { AdminSession, setAdminSession } from '$lib/state/session.svelte';
	import { toast } from '$lib/state/toast.svelte';
	import { safeRedirectPath } from '$lib/utils/redirect';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();

	const clerk = useClerkContext();
	// One cache per browser session (this layout lives for the whole signed-in session; SSR is off).
	const queryClient = createAdminQueryClient();

	let drawerOpen = $state(false);
	let leaving = false;

	const isDesktop = new MediaQuery('min-width: 1024px');

	function goToSignIn() {
		if (leaving) return;
		leaving = true;
		const target = safeRedirectPath(`${page.url.pathname}${page.url.search}`);
		const query = target === '/' ? '' : `?redirect_url=${encodeURIComponent(target)}`;
		// Full reload so the server-side guard and Clerk handshake run again.
		window.location.assign(`/sign-in${query}`);
	}

	configureApi({
		getToken: async () => (await clerk.session?.getToken()) ?? null,
		onUnauthorized: () => {
			toast.error(m.error_session_expired());
			goToSignIn();
		}
	});

	const me = createQuery(
		() => ({ ...meQuery(), enabled: clerk.isLoaded && !!clerk.auth.userId }),
		() => queryClient
	);
	setAdminSession(new AdminSession(() => me.data ?? null));

	const status = $derived.by((): 'loading' | 'ready' | 'noaccess' | 'error' => {
		const error = me.error;
		if (error instanceof ApiError && error.status === 403) return 'noaccess';
		if (me.data) return 'ready';
		// 401 keeps the spinner: onUnauthorized is already redirecting to sign-in.
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
			if (lastUserId !== undefined && lastUserId !== userId) queryClient.clear();
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
		await clerk.clerk?.signOut({ redirectUrl: '/sign-in' });
	}

	const drawerHidden = $derived(!isDesktop.current && !drawerOpen);
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === 'Escape' && drawerOpen) drawerOpen = false;
	}}
/>

<QueryClientProvider client={queryClient}>
	{#if status === 'loading'}
		<div class="flex min-h-screen items-center justify-center gap-3 text-slate-600" role="status">
			<span
				class="size-6 animate-spin rounded-full border-2 border-slate-300 border-t-brand-800"
				aria-hidden="true"
			></span>
			{m.auth_verifying()}
		</div>
	{:else if status === 'noaccess'}
		<main class="flex min-h-screen items-center justify-center p-4">
			<div class="w-full max-w-md card p-8 text-center">
				<Icon name="shield" size={40} class="mx-auto text-slate-400" />
				<h1 class="mt-4 text-xl font-semibold">{m.noaccess_title()}</h1>
				<p class="mt-2 text-sm text-slate-600">{m.noaccess_body()}</p>
				{#if clerk.user?.primaryEmailAddress?.emailAddress}
					<p class="mt-4 text-sm text-slate-500">
						{m.noaccess_signed_in_as({ email: clerk.user.primaryEmailAddress.emailAddress })}
					</p>
				{/if}
				<button type="button" class="mt-6 btn-primary" onclick={signOut}>{m.sign_out()}</button>
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
		<div class="min-h-screen lg:pl-64">
			<!-- Mobile top bar -->
			<div
				class="sticky top-0 z-30 flex items-center gap-3 bg-[#172554] px-4 py-3 text-white lg:hidden"
			>
				<button
					type="button"
					class="rounded-md p-1.5 hover:bg-white/10"
					aria-controls="admin-sidebar"
					aria-expanded={drawerOpen}
					aria-label={drawerOpen ? m.nav_close() : m.nav_open()}
					onclick={() => (drawerOpen = !drawerOpen)}
				>
					<Icon name={drawerOpen ? 'x' : 'menu'} size={22} />
				</button>
				<span class="font-bold">{m.app_name()}</span>
				<span class="rounded bg-white/15 px-1.5 py-0.5 text-[10px] font-semibold uppercase"
					>{m.admin_badge()}</span
				>
			</div>

			{#if drawerOpen && !isDesktop.current}
				<button
					type="button"
					class="fixed inset-0 z-40 bg-slate-900/50 lg:hidden"
					aria-label={m.nav_close()}
					tabindex="-1"
					onclick={() => (drawerOpen = false)}
				></button>
			{/if}

			<aside
				id="admin-sidebar"
				class={[
					'fixed inset-y-0 left-0 z-50 w-64 transition-transform duration-200 lg:translate-x-0',
					drawerOpen ? 'translate-x-0' : '-translate-x-full'
				]}
				inert={drawerHidden}
				aria-hidden={drawerHidden ? 'true' : undefined}
			>
				<Sidebar onnavigate={() => (drawerOpen = false)} />
			</aside>

			<main
				id="main-content"
				tabindex="-1"
				class="mx-auto max-w-7xl p-4 focus:outline-none sm:p-6 lg:p-8"
			>
				{@render children()}
			</main>
		</div>
	{/if}

	{#if import.meta.env.DEV}
		{#await import('@tanstack/svelte-query-devtools') then { SvelteQueryDevtools }}
			<SvelteQueryDevtools buttonPosition="bottom-left" />
		{/await}
	{/if}
</QueryClientProvider>
