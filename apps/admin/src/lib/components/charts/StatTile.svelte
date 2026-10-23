<script lang="ts">
	import Icon, { type IconName } from '../Icon.svelte';

	let {
		label,
		value,
		delta,
		deltaLabel,
		loading = false,
		href,
		icon,
		iconTone = 'bg-zinc-100 text-zinc-500'
	}: {
		label: string;
		/** Already formatted. */
		value: string;
		/** Relative change vs the previous period (0.12 = +12 %); null = no baseline. */
		delta?: number | null;
		/** e.g. "vs previous 30 days"; omit to hide the delta line. */
		deltaLabel?: string;
		loading?: boolean;
		/** Makes the whole tile a link (already resolved). */
		href?: string;
		icon?: IconName;
		/** Background + text classes of the icon chip. */
		iconTone?: string;
	} = $props();

	const pct = $derived(
		delta === null || delta === undefined ? null : Math.round(Math.abs(delta) * 1000) / 10
	);
	const up = $derived((delta ?? 0) > 0);
	const flat = $derived(pct === 0);
</script>

<!-- eslint-disable svelte/no-navigation-without-resolve -- callers pass resolved hrefs -->
<svelte:element
	this={href ? 'a' : 'div'}
	{href}
	class={[
		'block card p-4',
		href &&
			'transition hover:border-zinc-300 hover:shadow-md focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-400'
	]}
>
	<div class="flex items-start justify-between gap-3">
		<p class="text-sm text-zinc-500">{label}</p>
		{#if icon}
			<span class={['flex size-8 shrink-0 items-center justify-center rounded-lg', iconTone]}>
				<Icon name={icon} size={16} />
			</span>
		{/if}
	</div>
	{#if loading}
		<div class="mt-2 h-8 w-24 animate-pulse rounded bg-zinc-100"></div>
		{#if deltaLabel}<div class="mt-2 h-3 w-32 animate-pulse rounded bg-zinc-100"></div>{/if}
	{:else}
		<p class="mt-1 text-2xl font-semibold tracking-tight text-zinc-900">{value}</p>
		{#if deltaLabel}
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
	{/if}
</svelte:element>
<!-- eslint-enable svelte/no-navigation-without-resolve -->
