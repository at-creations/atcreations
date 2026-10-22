<script lang="ts">
	import { SignUp } from 'svelte-clerk';
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import { m } from '$lib/paraglide/messages.js';
	import LocaleSwitcher from '$lib/components/LocaleSwitcher.svelte';

	// Accounts are invitation-only. Clerk's invitation email links here with a
	// __clerk_ticket parameter, which <SignUp/> redeems. Without a ticket we do
	// not render the form at all (Clerk's "Restricted" sign-up mode would also
	// reject it server-side).
	const hasTicket = $derived(page.url.searchParams.has('__clerk_ticket'));
	const year = new Date().getFullYear();
</script>

<svelte:head>
	<title>{m.signup_title()} · {m.app_title()}</title>
</svelte:head>

<main class="flex min-h-screen flex-col items-center justify-center bg-[#172554] px-4 py-10">
	<div class="mb-8 text-center text-white">
		<p class="flex items-center justify-center gap-2 text-2xl font-bold">
			{m.app_name()}
			<span class="rounded bg-white/15 px-1.5 py-0.5 text-xs font-semibold uppercase"
				>{m.admin_badge()}</span
			>
		</p>
		<h1 class="mt-3 text-lg font-medium">{m.signup_title()}</h1>
		<p class="mt-1 text-sm text-blue-200">
			{hasTicket ? m.signup_subtitle() : m.signup_invite_only()}
		</p>
	</div>
	{#if hasTicket}
		<SignUp path="/sign-up" routing="path" signInUrl="/sign-in" forceRedirectUrl="/" />
	{:else}
		<a
			href={resolve('/sign-in')}
			class="rounded-lg bg-white px-4 py-2 text-sm font-semibold text-[#172554] hover:bg-blue-50 focus-visible:ring-2 focus-visible:ring-white focus-visible:outline-none"
			>{m.signup_go_to_signin()}</a
		>
	{/if}
	<div class="mt-8 w-48">
		<LocaleSwitcher />
	</div>
	<p class="mt-6 text-xs text-blue-200">© {year} AT Creations. {m.signin_copyright()}</p>
</main>
