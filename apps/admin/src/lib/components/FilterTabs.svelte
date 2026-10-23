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
	import { ToggleGroup } from 'bits-ui';

	let {
		tabs,
		value,
		label,
		onselect
	}: {
		tabs: FilterTab<T>[];
		value: T | '';
		/** Accessible name of the group. */
		label: string;
		onselect: (value: T | '') => void;
	} = $props();

	// ToggleGroup needs a non-empty value per item; "All" is '' in the app.
	const ALL = '__all__';
</script>

<!-- A filter, not ARIA tabs: each button re-queries the list below. -->
<ToggleGroup.Root
	type="single"
	bind:value={
		() => value || ALL,
		(next) => {
			// Always one tab selected: clicking the active one changes nothing.
			if (!next || next === (value || ALL)) return;
			onselect((next === ALL ? '' : next) as T | '');
		}
	}
	aria-label={label}
	class="-mb-px flex gap-1 overflow-x-auto"
>
	{#each tabs as tab (tab.value)}
		<ToggleGroup.Item
			value={tab.value || ALL}
			class="group flex shrink-0 items-center gap-2 border-b-2 border-transparent px-3 py-2.5 text-sm font-medium whitespace-nowrap text-zinc-500 transition-colors hover:border-zinc-300 hover:text-zinc-800 focus-visible:bg-zinc-50 focus-visible:outline-none data-[state=on]:border-brand-400 data-[state=on]:text-brand-800"
		>
			{tab.label}
			{#if tab.count !== undefined}
				<span
					class="min-w-6 rounded-full bg-zinc-100 px-1.5 py-px text-center text-xs text-zinc-500 tabular-nums group-data-[state=on]:bg-brand-100 group-data-[state=on]:text-brand-800"
				>
					{#if tab.count === null}
						<span class="inline-block h-2 w-3 animate-pulse rounded bg-zinc-200"></span>
					{:else}
						{tab.count}
					{/if}
				</span>
			{/if}
		</ToggleGroup.Item>
	{/each}
</ToggleGroup.Root>
