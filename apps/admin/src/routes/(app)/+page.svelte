<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { resolve } from '$app/paths';
	import Icon from '$lib/components/Icon.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import StatusBadge from '$lib/components/StatusBadge.svelte';
	import Thumb from '$lib/components/ui/Thumb.svelte';
	import { formatDateTime, localized } from '$lib/i18n';
	import { m } from '$lib/paraglide/messages.js';
	import { orderListQuery, pendingOrderCountQuery, productListQuery } from '$lib/queries';
	import { getAdminSession } from '$lib/state/session.svelte';
	import { formatCAD } from '$lib/utils/money';

	const session = getAdminSession();
	const canOrders = $derived(session.can('orders:read'));

	/** Stock scan size: enough to count out-of-stock items for a small catalogue. */
	const STOCK_SCAN = 50;
	const LOW_STOCK = 5;

	const total = (res: { meta: { total: number } }) => res.meta.total;

	const totalProducts = createQuery(() => ({
		...productListQuery({ pageSize: 1 }),
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
			lowest: res.data.filter((p) => p.stock < LOW_STOCK).slice(0, 6)
		})
	}));
	const outOfStock = $derived({ isError: stock.isError, data: stock.data?.outOfStock });
	const pendingOrders = createQuery(() => ({
		...pendingOrderCountQuery(),
		enabled: canOrders
	}));
	const recentOrders = createQuery(() => ({
		...orderListQuery({ sort: 'newest', pageSize: 6 }),
		select: (res) => res.data,
		enabled: canOrders
	}));

	const productsHref = resolve('/products');
	const ordersHref = resolve('/orders');
</script>

{#snippet stat(
	href: string,
	label: string,
	tone: string,
	query: { isError: boolean; data: number | undefined }
)}
	<!-- eslint-disable svelte/no-navigation-without-resolve -- resolved path + query -->
	<a {href} class={['rounded-2xl p-4 transition-[filter] hover:brightness-[0.98]', tone]}>
		<span class="text-sm opacity-75">{label}</span>
		<span class="mt-1 block text-2xl font-semibold tabular-nums">
			{#if query.data !== undefined}
				{query.data}
			{:else if query.isError}
				<span class="opacity-40">—</span>
			{:else}
				<span class="inline-block h-7 w-10 animate-pulse rounded bg-white/60"></span>
			{/if}
		</span>
	</a>
	<!-- eslint-enable svelte/no-navigation-without-resolve -->
{/snippet}

<PageHeader title={m.nav_dashboard()}>
	{#snippet actions()}
		{#if session.can('products:write')}
			<a class="btn-primary" href={resolve('/products/new')}
				><Icon name="plus" size={16} />{m.products_add()}</a
			>
		{/if}
	{/snippet}
</PageHeader>

<section class="grid grid-cols-2 gap-3 lg:grid-cols-4" aria-label={m.nav_dashboard()}>
	{#if canOrders}
		{@render stat(
			`${ordersHref}?status=pending`,
			m.dash_pending_orders(),
			'bg-amber-100 text-amber-900',
			pendingOrders
		)}
	{/if}
	{@render stat(productsHref, m.dash_products(), 'bg-sky-100 text-sky-900', totalProducts)}
	{@render stat(
		`${productsHref}?status=draft`,
		m.dash_draft(),
		'bg-brand-100 text-brand-900',
		drafts
	)}
	{@render stat(
		`${productsHref}?sort=stock_asc`,
		m.dash_out_of_stock(),
		'bg-rose-100 text-rose-900',
		outOfStock
	)}
</section>

<div class="mt-6 grid items-start gap-6 lg:grid-cols-3">
	{#if canOrders}
		<section class="card lg:col-span-2" aria-labelledby="recent-orders">
			<header class="card-header">
				<h2 id="recent-orders" class="card-title">{m.dash_recent_orders()}</h2>
				<a class="text-sm text-brand-600 hover:text-brand-700" href={ordersHref}
					>{m.dash_view_all()}</a
				>
			</header>
			{#if recentOrders.isPending}
				<div class="space-y-2 p-4" role="status" aria-label={m.common_loading()}>
					{#each [0, 1, 2] as i (i)}
						<div class="h-10 animate-pulse rounded-lg bg-zinc-50"></div>
					{/each}
				</div>
			{:else if recentOrders.isError}
				<p class="p-5 text-sm text-red-600">{m.orders_load_failed()}</p>
			{:else if recentOrders.data.length === 0}
				<p class="p-5 text-sm text-zinc-500">{m.dash_no_orders()}</p>
			{:else}
				<ul class="divide-y divide-zinc-100">
					{#each recentOrders.data as order (order.id)}
						<li>
							<a
								href={resolve(`/orders/${order.id}`)}
								class="flex items-center gap-4 px-5 py-3 text-sm hover:bg-zinc-50"
							>
								<span class="w-20 shrink-0 font-mono text-zinc-500">{order.number}</span>
								<span class="min-w-0 flex-1 truncate font-medium text-zinc-900"
									>{order.customer.name}</span
								>
								<span class="hidden text-zinc-500 sm:inline">{formatDateTime(order.createdAt)}</span
								>
								<StatusBadge kind="order" value={order.status} />
								<span class="w-20 text-right font-medium tabular-nums"
									>{formatCAD(order.totalCents)}</span
								>
							</a>
						</li>
					{/each}
				</ul>
			{/if}
		</section>
	{/if}

	<section class="card" aria-labelledby="low-stock">
		<header class="card-header">
			<h2 id="low-stock" class="card-title">{m.dash_low_stock()}</h2>
		</header>
		{#if stock.isPending}
			<div class="space-y-2 p-4" role="status" aria-label={m.common_loading()}>
				{#each [0, 1, 2] as i (i)}
					<div class="h-9 animate-pulse rounded-lg bg-zinc-50"></div>
				{/each}
			</div>
		{:else if stock.isError}
			<p class="p-5 text-sm text-red-600">{m.products_load_failed()}</p>
		{:else if stock.data.lowest.length === 0}
			<p class="p-5 text-sm text-zinc-500">—</p>
		{:else}
			<ul class="divide-y divide-zinc-100">
				{#each stock.data.lowest as product (product.id)}
					<li>
						<a
							href={resolve(`/products/${product.id}`)}
							class="flex items-center gap-3 px-5 py-2.5 text-sm hover:bg-zinc-50"
						>
							<Thumb
								src={product.images[0]?.url}
								class="size-8"
								rounded="rounded-md"
								iconSize={12}
							/>
							<span class="min-w-0 flex-1 truncate text-zinc-800">{localized(product.name)}</span>
							<span
								class={[
									'font-medium tabular-nums',
									product.stock === 0 ? 'text-red-600' : 'text-amber-600'
								]}>{product.stock}</span
							>
						</a>
					</li>
				{/each}
			</ul>
		{/if}
	</section>
</div>
