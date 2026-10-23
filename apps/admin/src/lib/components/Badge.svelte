<script lang="ts" module>
	export type BadgeTone = 'gray' | 'green' | 'amber' | 'red' | 'blue' | 'indigo' | 'purple' | 'sky';
</script>

<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		tone = 'gray',
		dot = false,
		children
	}: { tone?: BadgeTone; dot?: boolean; children: Snippet } = $props();

	const tones: Record<BadgeTone, string> = {
		gray: 'bg-zinc-100 text-zinc-600',
		green: 'bg-emerald-100 text-emerald-800',
		amber: 'bg-amber-100 text-amber-800',
		red: 'bg-rose-100 text-rose-800',
		blue: 'bg-sky-100 text-sky-800',
		indigo: 'bg-brand-100 text-brand-800',
		purple: 'bg-fuchsia-100 text-fuchsia-800',
		sky: 'bg-cyan-100 text-cyan-800'
	};
</script>

<span class={['badge', tones[tone]]}>
	{#if dot}<span class="size-1.5 rounded-full bg-current" aria-hidden="true"></span>{/if}
	{@render children()}
</span>
