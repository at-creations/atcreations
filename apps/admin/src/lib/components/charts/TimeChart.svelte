<script lang="ts" module>
	export interface ChartSeries {
		key: string;
		label: string;
		/** CSS color of the mark (a --color-chart-* token). Never used for text. */
		color: string;
		values: number[];
	}
</script>

<script lang="ts">
	import { compactNumber, niceTicks } from '$lib/utils/chart';

	let {
		series,
		labels,
		tooltipLabels = labels,
		type = 'lines',
		format = (n: number) => n.toLocaleString(),
		axisFormat = (n: number) => compactNumber(n),
		height = 220,
		label,
		dimmed = false
	}: {
		series: ChartSeries[];
		/** Short x-axis labels, one per point. */
		labels: string[];
		/** Longer labels for the tooltip (defaults to `labels`). */
		tooltipLabels?: string[];
		/** Columns: one series of daily totals. Lines: one or more series with a crosshair. */
		type?: 'columns' | 'lines';
		format?: (n: number) => string;
		axisFormat?: (n: number) => string;
		height?: number;
		/** Accessible summary of the chart. */
		label: string;
		/** Held at reduced opacity while new data loads. */
		dimmed?: boolean;
	} = $props();

	const PAD = { top: 8, right: 8, bottom: 24, left: 44 };
	const BAR_MAX = 24;
	const RADIUS = 4;

	let width = $state(0);
	let active = $state<number | null>(null);

	const n = $derived(labels.length);
	const plotW = $derived(Math.max(0, width - PAD.left - PAD.right));
	const plotH = $derived(height - PAD.top - PAD.bottom);
	const ticks = $derived(niceTicks(Math.max(0, ...series.flatMap((s) => s.values))));
	const top = $derived(ticks[ticks.length - 1] ?? 1);
	const band = $derived(n > 0 ? plotW / n : 0);
	const barW = $derived(Math.max(2, Math.min(BAR_MAX, band * 0.62)));

	const y = (v: number) => PAD.top + plotH * (1 - v / top);
	const base = $derived(PAD.top + plotH);
	const x = (i: number) =>
		type === 'columns'
			? PAD.left + band * (i + 0.5)
			: PAD.left + (n <= 1 ? plotW / 2 : (plotW * i) / (n - 1));

	/** Column with a 4px rounded data end, square at the baseline. */
	function columnPath(i: number, v: number): string {
		const x0 = x(i) - barW / 2;
		const x1 = x0 + barW;
		const yTop = y(v);
		const h = base - yTop;
		if (h <= 0) return '';
		const r = Math.min(RADIUS, h, barW / 2);
		return `M${x0},${base}V${yTop + r}Q${x0},${yTop} ${x0 + r},${yTop}H${x1 - r}Q${x1},${yTop} ${x1},${yTop + r}V${base}Z`;
	}

	function linePath(values: number[]): string {
		return values.map((v, i) => `${i === 0 ? 'M' : 'L'}${x(i)},${y(v)}`).join('');
	}

	function areaPath(values: number[]): string {
		if (values.length === 0) return '';
		return `${linePath(values)}L${x(values.length - 1)},${base}L${x(0)},${base}Z`;
	}

	/** About six evenly spread x labels, always including the last day. */
	const labelEvery = $derived(Math.max(1, Math.ceil(n / 6)));
	const showLabel = (i: number) => (n - 1 - i) % labelEvery === 0;

	function indexAt(px: number): number | null {
		if (n === 0 || px < PAD.left - 8 || px > width - PAD.right + 8) return null;
		const i =
			type === 'columns'
				? Math.floor((px - PAD.left) / band)
				: Math.round(((px - PAD.left) / Math.max(1, plotW)) * (n - 1));
		return Math.min(n - 1, Math.max(0, i));
	}

	function onkeydown(e: KeyboardEvent) {
		if (n === 0) return;
		if (e.key === 'ArrowRight' || e.key === 'ArrowLeft') {
			e.preventDefault();
			const cur = active ?? (e.key === 'ArrowRight' ? -1 : n);
			active = Math.min(n - 1, Math.max(0, cur + (e.key === 'ArrowRight' ? 1 : -1)));
		} else if (e.key === 'Home' || e.key === 'End') {
			e.preventDefault();
			active = e.key === 'Home' ? 0 : n - 1;
		} else if (e.key === 'Escape') {
			active = null;
		}
	}

	const tipLeft = $derived(active === null ? 0 : x(active));
	const tipOnRight = $derived(tipLeft < width / 2);
</script>

<div class="relative">
	{#if series.length > 1}
		<ul class="mb-3 flex flex-wrap gap-x-4 gap-y-1 text-xs text-zinc-500">
			{#each series as s (s.key)}
				<li class="flex items-center gap-1.5">
					<span class="h-0.5 w-3 rounded-full" style:background={s.color} aria-hidden="true"></span>
					{s.label}
				</li>
			{/each}
		</ul>
	{/if}

	<div class={['relative transition-opacity', dimmed && 'opacity-50']} bind:clientWidth={width}>
		{#if width > 0}
			<!-- Focusable so arrow keys can step through the days; the table view carries the same data. -->
			<!-- svelte-ignore a11y_no_noninteractive_tabindex, a11y_no_noninteractive_element_interactions -->
			<svg
				{width}
				{height}
				role="img"
				aria-label={label}
				tabindex="0"
				class="block touch-pan-y overflow-visible rounded-md focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-brand-400"
				onpointermove={(e) => (active = indexAt(e.offsetX))}
				onpointerleave={() => (active = null)}
				onblur={() => (active = null)}
				{onkeydown}
			>
				<!-- Gridlines and y-axis ticks -->
				{#each ticks as t (t)}
					<line
						x1={PAD.left}
						x2={width - PAD.right}
						y1={y(t)}
						y2={y(t)}
						class={t === 0 ? 'stroke-zinc-300' : 'stroke-zinc-100'}
						stroke-width="1"
						shape-rendering="crispEdges"
					/>
					<text
						x={PAD.left - 8}
						y={y(t)}
						dy="0.32em"
						text-anchor="end"
						class="fill-zinc-400 text-[11px] tabular-nums">{axisFormat(t)}</text
					>
				{/each}

				<!-- X labels -->
				{#each labels as l, i (i)}
					{#if showLabel(i)}
						<text
							x={x(i)}
							y={height - 6}
							text-anchor={type === 'lines' && i === n - 1 ? 'end' : 'middle'}
							class="fill-zinc-400 text-[11px]">{l}</text
						>
					{/if}
				{/each}

				{#if type === 'columns'}
					{#each series[0]?.values ?? [] as v, i (i)}
						<path
							d={columnPath(i, v)}
							fill={series[0]!.color}
							class={['transition-opacity', active !== null && active !== i && 'opacity-50']}
						/>
					{/each}
				{:else}
					{#if active !== null}
						<line
							x1={x(active)}
							x2={x(active)}
							y1={PAD.top}
							y2={base}
							class="stroke-zinc-300"
							stroke-width="1"
						/>
					{/if}
					{#each series as s, si (s.key)}
						{#if si === 0}
							<path d={areaPath(s.values)} fill={s.color} fill-opacity="0.1" />
						{/if}
						<path
							d={linePath(s.values)}
							fill="none"
							stroke={s.color}
							stroke-width="2"
							stroke-linejoin="round"
							stroke-linecap="round"
						/>
						{@const i = active ?? s.values.length - 1}
						{#if s.values.length > 0}
							<circle
								cx={x(i)}
								cy={y(s.values[i] ?? 0)}
								r="4"
								fill={s.color}
								stroke="white"
								stroke-width="2"
							/>
						{/if}
					{/each}
				{/if}
			</svg>
		{/if}

		{#if active !== null}
			<div
				class="pointer-events-none absolute top-0 z-10 min-w-36 rounded-lg border border-zinc-200 bg-white px-3 py-2 text-xs shadow-popover"
				style:left="{tipOnRight ? tipLeft + 12 : tipLeft - 12}px"
				style:transform={tipOnRight ? 'none' : 'translateX(-100%)'}
				role="status"
			>
				<p class="mb-1 text-zinc-500">{tooltipLabels[active]}</p>
				{#each series as s (s.key)}
					<p class="flex items-center justify-between gap-4">
						<span class="flex items-center gap-1.5 text-zinc-500">
							<span class="h-0.5 w-3 rounded-full" style:background={s.color} aria-hidden="true"
							></span>
							{s.label}
						</span>
						<span class="font-semibold text-zinc-900 tabular-nums"
							>{format(s.values[active] ?? 0)}</span
						>
					</p>
				{/each}
			</div>
		{/if}
	</div>
</div>
