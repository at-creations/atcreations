<script lang="ts">
	import '../app.css';
	import { ClerkProvider } from 'svelte-clerk';
	import ConfirmHost from '$lib/components/ConfirmHost.svelte';
	import Toaster from '$lib/components/Toaster.svelte';
	import { m } from '$lib/paraglide/messages.js';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();
</script>

<svelte:head>
	<title>{m.app_title()}</title>
</svelte:head>

<ClerkProvider signInUrl="/sign-in" signUpUrl="/sign-up" afterSignOutUrl="/sign-in">
	{@render children()}
</ClerkProvider>

<Toaster />
<ConfirmHost />
