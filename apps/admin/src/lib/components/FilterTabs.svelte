<script lang="ts" module>
	export interface FilterTab<V extends string> {
		/** '' = "All". */
		value: V | '';
		label: string;
		/** undefined = no counter, null = still loading. */
		count?: number | null;
	}
</script>

<script lang="ts" generics="T extends string">
	let {
		tabs,
		value,
		label,
		onselect
	}: {
		tabs: FilterTab<T>[];
		value: T | '';
		/** Accessible name of the tab group. */
		label: string;
		onselect: (value: T | '') => void;
	} = $props();
</script>

<!-- A row of filter buttons (not ARIA tabs: each one changes the list's query, like a segmented filter). -->
<div class="-mb-px flex gap-1 overflow-x-auto" role="group" aria-label={label}>
	{#each tabs as tab (tab.value)}
		{@const active = tab.value === value}
		<button
			type="button"
			aria-pressed={active}
			class={[
				'flex shrink-0 items-center gap-2 border-b-2 px-3 py-2.5 text-sm font-medium whitespace-nowrap transition-colors',
				active
					? 'border-brand-400 text-brand-800'
					: 'border-transparent text-zinc-500 hover:border-zinc-300 hover:text-zinc-800'
			]}
			onclick={() => onselect(tab.value)}
		>
			{tab.label}
			{#if tab.count !== undefined}
				<span
					class={[
						'min-w-6 rounded-full px-1.5 py-px text-center text-xs tabular-nums',
						active ? 'bg-brand-100 text-brand-800' : 'bg-zinc-100 text-zinc-500'
					]}
				>
					{#if tab.count === null}
						<span class="inline-block h-2 w-3 animate-pulse rounded bg-zinc-200"></span>
					{:else}
						{tab.count}
					{/if}
				</span>
			{/if}
		</button>
	{/each}
</div>
