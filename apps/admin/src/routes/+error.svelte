<script lang="ts">
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import { m } from '$lib/paraglide/messages.js';

	const notFound = $derived(page.status === 404);
	const forbidden = $derived(page.status === 403);
	const title = $derived(
		notFound ? m.error_not_found_title() : forbidden ? m.error_forbidden_title() : m.error_title()
	);
	const body = $derived(
		notFound ? m.error_not_found_body() : forbidden ? m.error_forbidden_body() : m.error_generic()
	);
</script>

<svelte:head>
	<title>{title} · {m.app_title()}</title>
</svelte:head>

<main class="flex min-h-screen items-center justify-center p-4">
	<div class="w-full max-w-md card p-8 text-center">
		<p class="text-5xl font-bold text-brand-800">{page.status}</p>
		<h1 class="mt-4 text-xl font-semibold">{title}</h1>
		<p class="mt-2 text-sm text-slate-600">{body}</p>
		<a class="mt-6 btn-primary" href={resolve('/')}>{m.error_back_home()}</a>
	</div>
</main>
