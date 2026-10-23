<script lang="ts" module>
	export interface BarListItem {
		key: string;
		label: string;
		value: number;
		/** Optional link for the row (already resolved). */
		href?: string;
	}
</script>

<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		items,
		format = (n: number) => n.toLocaleString(),
		showShare = false,
		empty,
		leading
	}: {
		items: BarListItem[];
		format?: (n: number) => string;
		/** Show each row's share of the total next to the value. */
		showShare?: boolean;
		/** Shown when there are no rows. */
		empty: string;
		/** Optional content before the label (icon, thumbnail). */
		leading?: Snippet<[BarListItem]>;
	} = $props();

	const max = $derived(Math.max(1, ...items.map((i) => i.value)));
	const total = $derived(items.reduce((sum, i) => sum + i.value, 0));
	const pct = (v: number) => (total > 0 ? Math.round((v / total) * 100) : 0);
</script>

{#if items.length === 0}
	<p class="py-6 text-center text-sm text-zinc-400">{empty}</p>
{:else}
	<ul class="space-y-3">
		{#each items as item (item.key)}
			<li>
				<!-- eslint-disable svelte/no-navigation-without-resolve -- callers pass resolved hrefs -->
				<svelte:element
					this={item.href ? 'a' : 'div'}
					href={item.href}
					class={['group block', item.href && 'rounded-md']}
				>
					<div class="mb-1 flex items-center justify-between gap-3 text-sm">
						<span
							class={[
								'flex min-w-0 items-center gap-2 text-zinc-700',
								item.href && 'group-hover:text-brand-700'
							]}
						>
							{@render leading?.(item)}
							<span class="truncate">{item.label}</span>
						</span>
						<span class="shrink-0 text-zinc-900 tabular-nums">
							<span class="font-medium">{format(item.value)}</span>
							{#if showShare}<span class="ml-1 text-xs text-zinc-400">{pct(item.value)}%</span>{/if}
						</span>
					</div>
					<div class="h-1.5 overflow-hidden rounded-full bg-zinc-100">
						<div
							class="h-full rounded-full bg-chart-1 transition-[width] duration-500"
							style:width="{(item.value / max) * 100}%"
						></div>
					</div>
				</svelte:element>
				<!-- eslint-enable svelte/no-navigation-without-resolve -->
			</li>
		{/each}
	</ul>
{/if}
