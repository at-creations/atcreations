<script lang="ts">
	import { SignIn } from 'svelte-clerk';
	import { page } from '$app/state';
	import AuthShell from '$lib/components/AuthShell.svelte';
	import { m } from '$lib/paraglide/messages.js';
	import { safeRedirectPath } from '$lib/utils/redirect';

	// Only same-origin relative paths are honoured as post-sign-in targets.
	const redirectTo = $derived(safeRedirectPath(page.url.searchParams.get('redirect_url')));
</script>

<AuthShell title={m.signin_title()} subtitle={m.signin_subtitle()}>
	<!-- Sign-up is intentionally not offered: staff are invited by administrators. -->
	<SignIn path="/sign-in" routing="path" forceRedirectUrl={redirectTo} withSignUp={false} />
	{#snippet footer()}{m.auth_need_access()}{/snippet}
</AuthShell>
