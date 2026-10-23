<script lang="ts">
	import { SignUp } from 'svelte-clerk';
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import AuthShell from '$lib/components/AuthShell.svelte';
	import Icon from '$lib/components/Icon.svelte';
	import { m } from '$lib/paraglide/messages.js';

	// Accounts are invitation-only. Clerk's invitation email links here with a
	// __clerk_ticket parameter, which <SignUp/> redeems. Without a ticket we do
	// not render the form at all (Clerk's "Restricted" sign-up mode would also
	// reject it server-side).
	const hasTicket = $derived(page.url.searchParams.has('__clerk_ticket'));
</script>

<AuthShell
	title={m.signup_title()}
	subtitle={hasTicket ? m.signup_subtitle() : m.signup_invite_only()}
	waitForClerk={hasTicket}
>
	{#if hasTicket}
		<SignUp path="/sign-up" routing="path" signInUrl="/sign-in" forceRedirectUrl="/" />
	{:else}
		<a href={resolve('/sign-in')} class="btn-primary h-10 w-full"
			>{m.signup_go_to_signin()}<Icon name="arrowRight" size={16} /></a
		>
	{/if}
</AuthShell>
