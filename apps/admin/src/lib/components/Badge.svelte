<script lang="ts" module>
	export type BadgeTone = 'gray' | 'green' | 'amber' | 'red' | 'blue' | 'indigo' | 'purple' | 'sky';
</script>

<script lang="ts">
	import type { Snippet } from 'svelte';

	let { tone = 'gray', children }: { tone?: BadgeTone; children: Snippet } = $props();

	const tones: Record<BadgeTone, string> = {
		gray: 'bg-slate-100 text-slate-700',
		green: 'bg-emerald-100 text-emerald-800',
		amber: 'bg-amber-100 text-amber-800',
		red: 'bg-red-100 text-red-800',
		blue: 'bg-blue-100 text-blue-800',
		indigo: 'bg-indigo-100 text-indigo-800',
		purple: 'bg-purple-100 text-purple-800',
		sky: 'bg-sky-100 text-sky-800'
	};
</script>

<span class={['badge', tones[tone]]}>{@render children()}</span>
