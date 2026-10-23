<script lang="ts">
	import { createQuery, keepPreviousData } from '@tanstack/svelte-query';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import BarList from '$lib/components/charts/BarList.svelte';
	import StatTile from '$lib/components/charts/StatTile.svelte';
	import TimeChart from '$lib/components/charts/TimeChart.svelte';
	import LoadState from '$lib/components/LoadState.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import RequirePermission from '$lib/components/RequirePermission.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Segmented from '$lib/components/ui/Segmented.svelte';
	import Thumb from '$lib/components/ui/Thumb.svelte';
	import { localized, orderStatusLabel } from '$lib/i18n';
	import { m } from '$lib/paraglide/messages.js';
	import { getLocale } from '$lib/paraglide/runtime.js';
	import { analyticsQuery, type AnalyticsDays } from '$lib/queries';
	import { getAdminSession } from '$lib/state/session.svelte';
	import { change, compactNumber } from '$lib/utils/chart';
	import { formatCAD } from '$lib/utils/money';
	import { readEnum, withParams } from '$lib/utils/query';

	const session = getAdminSession();
	const RANGES = ['7', '30', '90'] as const;

	const days = $derived(
		Number(readEnum(page.url.searchParams, 'days', RANGES, '30')) as AnalyticsDays
	);
	const report = createQuery(() => ({
		...analyticsQuery(days),
		enabled: session.can('orders:read'),
		placeholderData: keepPreviousData
	}));
	const data = $derived(report.data);
	const dimmed = $derived(report.isPlaceholderData);

	function setRange(next: string) {
		// Same page, only the query string changes.
		// eslint-disable-next-line svelte/no-navigation-without-resolve
		goto(withParams(page.url, { days: next === '30' ? null : next }), {
			keepFocus: true,
			noScroll: true
		});
	}

	const intl = $derived(getLocale() === 'vi' ? 'vi-VN' : 'en-CA');
	const dayShort = $derived(
		new Intl.DateTimeFormat(intl, { month: 'short', day: 'numeric', timeZone: 'UTC' })
	);
	const dayLong = $derived(
		new Intl.DateTimeFormat(intl, {
			weekday: 'short',
			month: 'short',
			day: 'numeric',
			timeZone: 'UTC'
		})
	);
	// Dates are plain store-local days ("2026-10-20"); format them as UTC so they never shift.
	const asDate = (d: string) => new Date(`${d}T00:00:00Z`);
	const countries = $derived(new Intl.DisplayNames([intl], { type: 'region' }));

	const labels = $derived(data?.daily.map((d) => dayShort.format(asDate(d.date))) ?? []);
	const tooltipLabels = $derived(data?.daily.map((d) => dayLong.format(asDate(d.date))) ?? []);

	const conversion = (orders: number, visitors: number) => (visitors > 0 ? orders / visitors : 0);
	const pct = (f: number) => `${(Math.round(f * 1000) / 10).toLocaleString(intl)}%`;
	const cadCompact = (cents: number) =>
		new Intl.NumberFormat(intl, {
			style: 'currency',
			currency: 'CAD',
			notation: 'compact',
			maximumFractionDigits: 1
		}).format(cents / 100);

	const deviceLabels: Record<string, () => string> = {
		mobile: m.an_device_mobile,
		tablet: m.an_device_tablet,
		desktop: m.an_device_desktop
	};
	function countryName(code: string) {
		try {
			return countries.of(code) ?? code;
		} catch {
			return code;
		}
	}
	const flag = (code: string) =>
		String.fromCodePoint(...[...code.toUpperCase()].map((c) => 0x1f1a5 + c.charCodeAt(0)));

	const vsLabel = $derived(m.an_vs_previous({ days }));
</script>

<RequirePermission permission="orders:read">
	<PageHeader title={m.nav_analytics()}>
		{#snippet actions()}
			<Segmented
				label={m.an_range()}
				value={String(days)}
				onchange={setRange}
				options={RANGES.map((r) => ({ value: r, label: m.an_range_days({ days: r }) }))}
			/>
		{/snippet}
	</PageHeader>

	{#if report.isError && !data}
		<div class="card"><LoadState error={report.error} onretry={() => report.refetch()} /></div>
	{:else}
		{@const t = data?.totals}
		{@const p = data?.previous}
		<section
			class="grid grid-cols-2 gap-3 lg:grid-cols-3 xl:grid-cols-5"
			aria-label={m.nav_analytics()}
		>
			<StatTile
				label={m.an_revenue()}
				value={t ? formatCAD(t.revenueCents) : ''}
				delta={t && p ? change(t.revenueCents, p.revenueCents) : null}
				deltaLabel={vsLabel}
				loading={!t}
			/>
			<StatTile
				label={m.an_orders()}
				value={t ? t.orders.toLocaleString(intl) : ''}
				delta={t && p ? change(t.orders, p.orders) : null}
				deltaLabel={vsLabel}
				loading={!t}
			/>
			<StatTile
				label={m.an_avg_order()}
				value={t ? formatCAD(t.avgOrderCents) : ''}
				delta={t && p ? change(t.avgOrderCents, p.avgOrderCents) : null}
				deltaLabel={vsLabel}
				loading={!t}
			/>
			<StatTile
				label={m.an_visitors()}
				value={t ? t.visitors.toLocaleString(intl) : ''}
				delta={t && p ? change(t.visitors, p.visitors) : null}
				deltaLabel={vsLabel}
				loading={!t}
			/>
			<StatTile
				label={m.an_conversion()}
				value={t ? pct(conversion(t.orders, t.visitors)) : ''}
				delta={t && p
					? change(conversion(t.orders, t.visitors), conversion(p.orders, p.visitors))
					: null}
				deltaLabel={vsLabel}
				loading={!t}
			/>
		</section>

		<div class="mt-6 grid items-start gap-6 lg:grid-cols-3">
			<Card title={m.an_revenue()} class="lg:col-span-2">
				{#if data}
					<TimeChart
						type="columns"
						label={m.an_revenue_chart({ days })}
						{labels}
						{tooltipLabels}
						{dimmed}
						format={formatCAD}
						axisFormat={cadCompact}
						series={[
							{
								key: 'revenue',
								label: m.an_revenue(),
								color: 'var(--color-chart-1)',
								values: data.daily.map((d) => d.revenueCents)
							}
						]}
					/>
				{:else}
					<div class="h-[220px] animate-pulse rounded-lg bg-zinc-50"></div>
				{/if}
			</Card>

			<Card title={m.an_order_status()}>
				<BarList
					empty={m.an_no_data()}
					items={(data?.ordersByStatus ?? []).map((s) => ({
						key: s.status,
						label: orderStatusLabel(s.status),
						value: s.count,
						href: `${resolve('/orders')}?status=${s.status}`
					}))}
				/>
			</Card>

			<Card title={m.an_traffic()} class="lg:col-span-2">
				{#if data}
					<TimeChart
						label={m.an_traffic_chart({ days })}
						{labels}
						{tooltipLabels}
						{dimmed}
						series={[
							{
								key: 'visitors',
								label: m.an_visitors(),
								color: 'var(--color-chart-1)',
								values: data.daily.map((d) => d.visitors)
							},
							{
								key: 'pageViews',
								label: m.an_page_views(),
								color: 'var(--color-chart-2)',
								values: data.daily.map((d) => d.pageViews)
							}
						]}
					/>
				{:else}
					<div class="h-[240px] animate-pulse rounded-lg bg-zinc-50"></div>
				{/if}
			</Card>

			<Card title={m.an_sources()}>
				<BarList
					empty={m.an_no_data()}
					showShare
					items={(data?.referrers ?? []).map((r) => ({
						key: r.key,
						label: r.key === 'direct' ? m.an_direct() : r.key,
						value: r.count
					}))}
				/>
			</Card>

			<Card title={m.an_top_selling()} padded={false}>
				{#if (data?.topSelling.length ?? 0) === 0}
					<p class="py-10 text-center text-sm text-zinc-400">{m.an_no_data()}</p>
				{:else}
					<ul class="divide-y divide-zinc-100">
						{#each data?.topSelling ?? [] as item, i (item.productId)}
							<li>
								<a
									href={resolve(`/products/${item.productId}`)}
									class="flex items-center gap-3 px-5 py-3 hover:bg-zinc-50"
								>
									<span class="w-4 text-xs text-zinc-400 tabular-nums">{i + 1}</span>
									<Thumb src={item.imageUrl} class="size-9" />
									<span class="min-w-0 flex-1">
										<span class="block truncate text-sm font-medium text-zinc-900"
											>{localized(item.name)}</span
										>
										<span class="text-xs text-zinc-500"
											>{m.an_units_sold({ count: item.unitsSold })}</span
										>
									</span>
									<span class="text-sm font-semibold text-zinc-900 tabular-nums"
										>{formatCAD(item.revenueCents)}</span
									>
								</a>
							</li>
						{/each}
					</ul>
				{/if}
			</Card>

			<Card title={m.an_most_viewed()}>
				<BarList
					empty={m.an_no_data()}
					items={(data?.topViewed ?? []).map((v) => ({
						key: v.productId,
						label: localized(v.name),
						value: v.views,
						href: resolve(`/products/${v.productId}`)
					}))}
				>
					{#snippet leading(item)}
						<Thumb
							src={data?.topViewed.find((v) => v.productId === item.key)?.imageUrl}
							class="size-6"
							rounded="rounded"
							iconSize={10}
						/>
					{/snippet}
				</BarList>
			</Card>

			<Card title={m.an_top_pages()}>
				<BarList
					empty={m.an_no_data()}
					items={(data?.pages ?? []).map((pg) => ({ key: pg.key, label: pg.key, value: pg.count }))}
				/>
			</Card>

			<Card title={m.an_devices()}>
				<BarList
					empty={m.an_no_data()}
					showShare
					items={(data?.devices ?? []).map((d) => ({
						key: d.key,
						label: deviceLabels[d.key]?.() ?? d.key,
						value: d.count
					}))}
				/>
			</Card>

			<Card title={m.an_countries()}>
				<BarList
					empty={m.an_no_data()}
					showShare
					items={(data?.countries ?? []).map((c) => ({
						key: c.key,
						label: `${flag(c.key)} ${countryName(c.key)}`,
						value: c.count
					}))}
				/>
			</Card>
		</div>

		{#if data}
			<details class="group mt-6 card">
				<summary
					class="flex cursor-pointer list-none items-center justify-between px-5 py-3.5 text-sm font-semibold text-zinc-900"
				>
					{m.an_daily_table()}
					<span class="text-xs font-normal text-zinc-400 group-open:hidden">{m.an_show()}</span>
				</summary>
				<div class="overflow-x-auto border-t border-zinc-100">
					<table class="w-full text-left text-sm">
						<thead class="bg-zinc-50/80">
							<tr>
								<th scope="col" class="th pl-5">{m.an_date()}</th>
								<th scope="col" class="th text-right">{m.an_revenue()}</th>
								<th scope="col" class="th text-right">{m.an_orders()}</th>
								<th scope="col" class="th text-right">{m.an_visitors()}</th>
								<th scope="col" class="th pr-5 text-right">{m.an_page_views()}</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-zinc-100 tabular-nums">
							{#each [...data.daily].reverse() as d (d.date)}
								<tr>
									<td class="td pl-5 text-zinc-600">{dayLong.format(asDate(d.date))}</td>
									<td class="td text-right">{formatCAD(d.revenueCents)}</td>
									<td class="td text-right">{d.orders}</td>
									<td class="td text-right">{compactNumber(d.visitors)}</td>
									<td class="td pr-5 text-right">{compactNumber(d.pageViews)}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</details>
			<p class="mt-4 text-xs text-zinc-400">{m.an_privacy_note({ timezone: data.timezone })}</p>
		{/if}
	{/if}
</RequirePermission>
