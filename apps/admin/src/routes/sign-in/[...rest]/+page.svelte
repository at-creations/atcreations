<script lang="ts">
	import { SignIn } from 'svelte-clerk';
	import { page } from '$app/state';
	import { m } from '$lib/paraglide/messages.js';
	import LocaleSwitcher from '$lib/components/LocaleSwitcher.svelte';
	import { safeRedirectPath } from '$lib/utils/redirect';

	// Only same-origin relative paths are honoured as post-sign-in targets.
	const redirectTo = $derived(safeRedirectPath(page.url.searchParams.get('redirect_url')));
	const year = new Date().getFullYear();
</script>

<svelte:head>
	<title>{m.signin_title()} · {m.app_title()}</title>
</svelte:head>

<main class="flex min-h-screen flex-col items-center justify-center bg-[#172554] px-4 py-10">
	<div class="mb-8 text-center text-white">
		<p class="flex items-center justify-center gap-2 text-2xl font-bold">
			{m.app_name()}
			<span class="rounded bg-white/15 px-1.5 py-0.5 text-xs font-semibold uppercase"
				>{m.admin_badge()}</span
			>
		</p>
		<h1 class="mt-3 text-lg font-medium">{m.signin_title()}</h1>
		<p class="mt-1 text-sm text-blue-200">{m.signin_subtitle()}</p>
	</div>
	<!-- Sign-up is intentionally not offered: staff are invited by administrators. -->
	<SignIn
		path="/sign-in"
		routing="path"
		forceRedirectUrl={redirectTo}
		withSignUp={false}
		appearance={{ elements: { footerAction: { display: 'none' } } }}
	/>
	<div class="mt-8 w-48">
		<LocaleSwitcher />
	</div>
	<p class="mt-6 text-xs text-blue-200">© {year} AT Creations. {m.signin_copyright()}</p>
</main>
