<script lang="ts">
	import type { Snippet } from 'svelte';
	import Icon, { type IconName } from './Icon.svelte';

	let {
		icon = 'inbox',
		title,
		body,
		children
	}: { icon?: IconName; title: string; body?: string; children?: Snippet } = $props();
</script>

<div class="flex flex-col items-center px-6 py-14 text-center">
	<span
		class="flex size-12 items-center justify-center rounded-full bg-zinc-100 text-zinc-400 ring-8 ring-zinc-50"
	>
		<Icon name={icon} size={22} />
	</span>
	<p class="mt-5 text-sm font-semibold text-zinc-900">{title}</p>
	{#if body}<p class="mt-1 max-w-sm text-sm text-zinc-500">{body}</p>{/if}
	{#if children}<div class="mt-5 flex flex-wrap justify-center gap-2">
			{@render children()}
		</div>{/if}
</div>
