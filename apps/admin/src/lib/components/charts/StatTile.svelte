<script lang="ts">
	import Icon from '../Icon.svelte';

	let {
		label,
		value,
		delta,
		deltaLabel,
		loading = false
	}: {
		label: string;
		/** Already formatted. */
		value: string;
		/** Relative change vs the previous period (0.12 = +12 %); null = no baseline. */
		delta?: number | null;
		/** e.g. "vs previous 30 days" */
		deltaLabel: string;
		loading?: boolean;
	} = $props();

	const pct = $derived(
		delta === null || delta === undefined ? null : Math.round(Math.abs(delta) * 1000) / 10
	);
	const up = $derived((delta ?? 0) > 0);
	const flat = $derived(pct === 0);
</script>

<div class="card p-4">
	<p class="text-sm text-zinc-500">{label}</p>
	{#if loading}
		<div class="mt-2 h-8 w-24 animate-pulse rounded bg-zinc-100"></div>
		<div class="mt-2 h-3 w-32 animate-pulse rounded bg-zinc-100"></div>
	{:else}
		<p class="mt-1 text-2xl font-semibold tracking-tight text-zinc-900">{value}</p>
		<p class="mt-1 flex items-center gap-1 text-xs text-zinc-500">
			{#if pct !== null && !flat}
				<span
					class={[
						'inline-flex items-center gap-0.5 font-medium',
						up ? 'text-emerald-700' : 'text-rose-700'
					]}
				>
					<Icon name={up ? 'up' : 'down'} size={12} />{pct}%
				</span>
			{:else}
				<span class="font-medium text-zinc-400">—</span>
			{/if}
			{deltaLabel}
		</p>
	{/if}
</div>
