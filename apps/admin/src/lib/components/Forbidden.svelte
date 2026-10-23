<script lang="ts">
	import { resolve } from '$app/paths';
	import { m } from '$lib/paraglide/messages.js';
	import Icon from './Icon.svelte';
</script>

<div class="mx-auto mt-10 flex max-w-lg flex-col items-center gap-3 card p-10 text-center">
	<Icon name="shield" size={36} class="text-zinc-400" />
	<h1 class="text-xl font-semibold">{m.error_forbidden_title()}</h1>
	<p class="text-sm text-zinc-600">{m.error_forbidden_body()}</p>
	<a class="btn-primary mt-2" href={resolve('/')}>{m.error_back_home()}</a>
</div>
