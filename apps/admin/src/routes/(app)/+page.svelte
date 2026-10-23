<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { resolve } from '$app/paths';
	import StatTile from '$lib/components/charts/StatTile.svelte';
	import TimeChart from '$lib/components/charts/TimeChart.svelte';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import Icon, { type IconName } from '$lib/components/Icon.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import StockLabel, { LOW_STOCK } from '$lib/components/products/StockLabel.svelte';
	import StatusBadge from '$lib/components/StatusBadge.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Thumb from '$lib/components/ui/Thumb.svelte';
	import { formatDateTime, localized } from '$lib/i18n';
	import { m } from '$lib/paraglide/messages.js';
	import { getLocale } from '$lib/paraglide/runtime.js';
	import {
		analyticsQuery,
		orderListQuery,
		pendingOrderCountQuery,
		productListQuery
	} from '$lib/queries';
	import { getAdminSession } from '$lib/state/session.svelte';
	import { change } from '$lib/utils/chart';
	import { formatCAD } from '$lib/utils/money';

	const session = getAdminSession();
	const canOrders = $derived(session.can('orders:read'));

	/** The headline period for sales figures. */
	const DAYS = 7;
	/** Stock scan size: enough to count out-of-stock items for a small catalogue. */
	const STOCK_SCAN = 50;

	const total = (res: { meta: { total: number } }) => res.meta.total;

	const report = createQuery(() => ({ ...analyticsQuery(DAYS), enabled: canOrders }));
	const pendingOrders = createQuery(() => ({ ...pendingOrderCountQuery(), enabled: canOrders }));
	const recentOrders = createQuery(() => ({
		...orderListQuery({ sort: 'newest', pageSize: 6 }),
		select: (res) => res.data,
		enabled: canOrders
	}));
	const totalProducts = createQuery(() => ({
		...productListQuery({ pageSize: 1 }),
		select: total
	}));
	const published = createQuery(() => ({
		...productListQuery({ pageSize: 1, status: 'published' }),
		select: total
	}));
	const drafts = createQuery(() => ({
		...productListQuery({ pageSize: 1, status: 'draft' }),
		select: total
	}));
	const stock = createQuery(() => ({
		...productListQuery({ sort: 'stock_asc', pageSize: STOCK_SCAN }),
		select: (res) => ({
			outOfStock: res.data.filter((p) => p.stock === 0).length,
			low: res.data.filter((p) => p.stock <= LOW_STOCK).slice(0, 6)
		})
	}));
	const newest = createQuery(() => ({
		...productListQuery({ sort: 'newest', pageSize: 4 }),
		select: (res) => res.data,
		enabled: !canOrders
	}));

	const intl = $derived(getLocale() === 'vi' ? 'vi-VN' : 'en-CA');
	const today = $derived(
		new Intl.DateTimeFormat(intl, { weekday: 'long', month: 'long', day: 'numeric' }).format(
			new Date()
		)
	);
	const dayShort = $derived(new Intl.DateTimeFormat(intl, { weekday: 'short', timeZone: 'UTC' }));
	const dayLong = $derived(
		new Intl.DateTimeFormat(intl, {
			weekday: 'long',
			month: 'short',
			day: 'numeric',
			timeZone: 'UTC'
		})
	);
	const asDate = (d: string) => new Date(`${d}T00:00:00Z`);
	const cadCompact = (cents: number) =>
		new Intl.NumberFormat(intl, {
			style: 'currency',
			currency: 'CAD',
			notation: 'compact',
			maximumFractionDigits: 1
		}).format(cents / 100);

	const t = $derived(report.data?.totals);
	const p = $derived(report.data?.previous);
	const vs = $derived(m.an_vs_previous({ days: DAYS }));

	interface Todo {
		key: string;
		icon: IconName;
		tone: string;
		label: string;
		count: number;
		href: string;
	}
	const todos = $derived.by<Todo[]>(() => {
		const out: Todo[] = [];
		if (canOrders && (pendingOrders.data ?? 0) > 0)
			out.push({
				key: 'pending',
				icon: 'inbox',
				tone: 'bg-amber-100 text-amber-700',
				label: m.dash_todo_pending(),
				count: pendingOrders.data ?? 0,
				href: `${resolve('/orders')}?status=pending`
			});
		if ((stock.data?.outOfStock ?? 0) > 0)
			out.push({
				key: 'stock',
				icon: 'alert',
				tone: 'bg-rose-100 text-rose-700',
				label: m.dash_out_of_stock(),
				count: stock.data?.outOfStock ?? 0,
				href: `${resolve('/products')}?sort=stock_asc`
			});
		if ((drafts.data ?? 0) > 0)
			out.push({
				key: 'drafts',
				icon: 'edit',
				tone: 'bg-brand-100 text-brand-700',
				label: m.dash_todo_drafts(),
				count: drafts.data ?? 0,
				href: `${resolve('/products')}?status=draft`
			});
		return out;
	});
	const todosLoading = $derived(
		stock.isPending || drafts.isPending || (canOrders && pendingOrders.isPending)
	);
</script>

<PageHeader title={m.nav_dashboard()} subtitle={today}>
	{#snippet actions()}
		{#if session.can('products:write')}
			<a class="btn-primary" href={resolve('/products/new')}
				><Icon name="plus" size={16} />{m.products_add()}</a
			>
		{/if}
	{/snippet}
</PageHeader>

<!-- Headline numbers -->
<section class="grid grid-cols-2 gap-3 xl:grid-cols-4" aria-label={m.nav_dashboard()}>
	{#if canOrders}
		<StatTile
			label={m.dash_revenue_week()}
			value={t ? formatCAD(t.revenueCents) : ''}
			delta={t && p ? change(t.revenueCents, p.revenueCents) : null}
			deltaLabel={vs}
			loading={!t}
			href={`${resolve('/analytics')}?days=7`}
			icon="chart"
			iconTone="bg-brand-100 text-brand-700"
		/>
		<StatTile
			label={m.dash_orders_week()}
			value={t ? t.orders.toLocaleString(intl) : ''}
			delta={t && p ? change(t.orders, p.orders) : null}
			deltaLabel={vs}
			loading={!t}
			href={resolve('/orders')}
			icon="inbox"
			iconTone="bg-amber-100 text-amber-700"
		/>
		<StatTile
			label={m.dash_visitors_week()}
			value={t ? t.visitors.toLocaleString(intl) : ''}
			delta={t && p ? change(t.visitors, p.visitors) : null}
			deltaLabel={vs}
			loading={!t}
			href={`${resolve('/analytics')}?days=7`}
			icon="users"
			iconTone="bg-sky-100 text-sky-700"
		/>
	{:else}
		<StatTile
			label={m.dash_products()}
			value={totalProducts.data?.toLocaleString(intl) ?? ''}
			loading={totalProducts.isPending}
			href={resolve('/products')}
			icon="box"
			iconTone="bg-sky-100 text-sky-700"
		/>
		<StatTile
			label={m.dash_draft()}
			value={drafts.data?.toLocaleString(intl) ?? ''}
			loading={drafts.isPending}
			href={`${resolve('/products')}?status=draft`}
			icon="edit"
			iconTone="bg-brand-100 text-brand-700"
		/>
		<StatTile
			label={m.dash_out_of_stock()}
			value={stock.data?.outOfStock.toLocaleString(intl) ?? ''}
			loading={stock.isPending}
			href={`${resolve('/products')}?sort=stock_asc`}
			icon="alert"
			iconTone="bg-rose-100 text-rose-700"
		/>
	{/if}
	<StatTile
		label={m.dash_live_products()}
		value={published.data?.toLocaleString(intl) ?? ''}
		loading={published.isPending}
		href={`${resolve('/products')}?status=published`}
		icon="check"
		iconTone="bg-emerald-100 text-emerald-700"
	/>
</section>

<div class="mt-6 grid items-start gap-6 lg:grid-cols-3">
	<!-- Sales trend (or newest products for roles without order access) -->
	{#if canOrders}
		<Card title={m.dash_sales_week()} class="lg:col-span-2">
			{#snippet actions()}
				<a class="text-sm text-brand-600 hover:text-brand-700" href={resolve('/analytics')}
					>{m.dash_view_analytics()}</a
				>
			{/snippet}
			{#if report.data}
				<TimeChart
					type="columns"
					height={200}
					label={m.an_revenue_chart({ days: DAYS })}
					labels={report.data.daily.map((d) => dayShort.format(asDate(d.date)))}
					tooltipLabels={report.data.daily.map((d) => dayLong.format(asDate(d.date)))}
					format={formatCAD}
					axisFormat={cadCompact}
					series={[
						{
							key: 'revenue',
							label: m.an_revenue(),
							color: 'var(--color-chart-1)',
							values: report.data.daily.map((d) => d.revenueCents)
						}
					]}
				/>
			{:else if report.isError}
				<p class="py-16 text-center text-sm text-zinc-400">{m.an_no_data()}</p>
			{:else}
				<div class="h-[200px] animate-pulse rounded-lg bg-zinc-50"></div>
			{/if}
		</Card>
	{:else}
		<Card title={m.dash_newest_products()} class="lg:col-span-2">
			{#if newest.data && newest.data.length > 0}
				<ul class="grid grid-cols-2 gap-3 sm:grid-cols-4">
					{#each newest.data as product (product.id)}
						<li>
							<a href={resolve(`/products/${product.id}`)} class="group block">
								<Thumb
									src={product.images[0]?.url}
									class="aspect-square w-full"
									rounded="rounded-xl"
									iconSize={24}
								/>
								<p
									class="mt-2 truncate text-sm font-medium text-zinc-800 group-hover:text-brand-700"
								>
									{localized(product.name)}
								</p>
							</a>
						</li>
					{/each}
				</ul>
			{:else}
				<p class="py-10 text-center text-sm text-zinc-400">{m.products_empty_title()}</p>
			{/if}
		</Card>
	{/if}

	<!-- To do -->
	<Card title={m.dash_todo()} padded={false}>
		{#if todosLoading}
			<div class="space-y-2 p-4" role="status" aria-label={m.common_loading()}>
				{#each [0, 1, 2] as i (i)}
					<div class="h-11 animate-pulse rounded-lg bg-zinc-50"></div>
				{/each}
			</div>
		{:else if todos.length === 0}
			<EmptyState icon="check" title={m.dash_all_done()} />
		{:else}
			<ul class="divide-y divide-zinc-100">
				{#each todos as todo (todo.key)}
					<li>
						<!-- eslint-disable svelte/no-navigation-without-resolve -- resolved path + query -->
						<a href={todo.href} class="group flex items-center gap-3 px-5 py-3.5 hover:bg-zinc-50">
							<span class={['flex size-9 items-center justify-center rounded-lg', todo.tone]}>
								<Icon name={todo.icon} size={16} />
							</span>
							<span class="flex-1 text-sm font-medium text-zinc-800">{todo.label}</span>
							<span class="text-lg font-semibold text-zinc-900 tabular-nums">{todo.count}</span>
							<Icon name="right" size={16} class="text-zinc-300 group-hover:text-zinc-500" />
						</a>
						<!-- eslint-enable svelte/no-navigation-without-resolve -->
					</li>
				{/each}
			</ul>
		{/if}
	</Card>

	<!-- Recent orders -->
	{#if canOrders}
		<Card title={m.dash_recent_orders()} padded={false} class="lg:col-span-2">
			{#snippet actions()}
				<a class="text-sm text-brand-600 hover:text-brand-700" href={resolve('/orders')}
					>{m.dash_view_all()}</a
				>
			{/snippet}
			{#if recentOrders.isPending}
				<div class="space-y-2 p-4" role="status" aria-label={m.common_loading()}>
					{#each [0, 1, 2, 3] as i (i)}
						<div class="h-11 animate-pulse rounded-lg bg-zinc-50"></div>
					{/each}
				</div>
			{:else if recentOrders.isError}
				<p class="p-5 text-sm text-red-600">{m.orders_load_failed()}</p>
			{:else if recentOrders.data.length === 0}
				<EmptyState icon="inbox" title={m.dash_no_orders()} />
			{:else}
				<ul class="divide-y divide-zinc-100">
					{#each recentOrders.data as order (order.id)}
						<li>
							<a
								href={resolve(`/orders/${order.id}`)}
								class="flex items-center gap-3 px-5 py-3 text-sm hover:bg-zinc-50"
							>
								<span
									class="flex size-9 shrink-0 items-center justify-center rounded-full bg-zinc-100 text-xs font-semibold text-zinc-600 uppercase"
									aria-hidden="true">{order.customer.name.slice(0, 2)}</span
								>
								<span class="min-w-0 flex-1">
									<span class="block truncate font-medium text-zinc-900">{order.customer.name}</span
									>
									<span class="block truncate text-xs text-zinc-500"
										><span class="font-mono">{order.number}</span> · {formatDateTime(
											order.createdAt
										)}</span
									>
								</span>
								<StatusBadge kind="order" value={order.status} />
								<span
									class="hidden w-24 text-right font-semibold text-zinc-900 tabular-nums sm:block"
									>{formatCAD(order.totalCents)}</span
								>
							</a>
						</li>
					{/each}
				</ul>
			{/if}
		</Card>
	{/if}

	<!-- Low stock -->
	<Card title={m.dash_low_stock()} padded={false} class={!canOrders ? 'lg:col-span-3' : ''}>
		{#snippet actions()}
			<!-- eslint-disable svelte/no-navigation-without-resolve -- resolved path + query -->
			<a
				class="text-sm text-brand-600 hover:text-brand-700"
				href={`${resolve('/products')}?sort=stock_asc`}>{m.dash_view_all()}</a
			>
			<!-- eslint-enable svelte/no-navigation-without-resolve -->
		{/snippet}
		{#if stock.isPending}
			<div class="space-y-2 p-4" role="status" aria-label={m.common_loading()}>
				{#each [0, 1, 2, 3] as i (i)}
					<div class="h-10 animate-pulse rounded-lg bg-zinc-50"></div>
				{/each}
			</div>
		{:else if stock.isError}
			<p class="p-5 text-sm text-red-600">{m.products_load_failed()}</p>
		{:else if stock.data.low.length === 0}
			<EmptyState icon="check" title={m.dash_stock_ok()} />
		{:else}
			<ul class="divide-y divide-zinc-100">
				{#each stock.data.low as product (product.id)}
					<li>
						<a
							href={resolve(`/products/${product.id}`)}
							class="flex items-center gap-3 px-5 py-2.5 hover:bg-zinc-50"
						>
							<Thumb src={product.images[0]?.url} class="size-9" />
							<span class="min-w-0 flex-1 truncate text-sm text-zinc-800"
								>{localized(product.name)}</span
							>
							<StockLabel stock={product.stock} />
						</a>
					</li>
				{/each}
			</ul>
		{/if}
	</Card>
</div>
