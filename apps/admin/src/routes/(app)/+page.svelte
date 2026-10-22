<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { resolve } from '$app/paths';
	import Icon from '$lib/components/Icon.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import StatusBadge from '$lib/components/StatusBadge.svelte';
	import { formatDateTime, localized } from '$lib/i18n';
	import { safeImageUrl } from '$lib/links';
	import { m } from '$lib/paraglide/messages.js';
	import { orderListQuery, productListQuery } from '$lib/queries';
	import { getAdminSession } from '$lib/state/session.svelte';
	import { formatCAD } from '$lib/utils/money';

	const session = getAdminSession();
	const canOrders = $derived(session.can('orders:read'));

	const total = (res: { meta: { total: number } }) => res.meta.total;

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
	const lowStock = createQuery(() => ({
		...productListQuery({ sort: 'stock_asc', pageSize: 5 }),
		select: (res) => res.data
	}));
	const pendingOrders = createQuery(() => ({
		...orderListQuery({ status: 'pending', pageSize: 1 }),
		select: total,
		enabled: canOrders
	}));
	const recentOrders = createQuery(() => ({
		...orderListQuery({ sort: 'newest', pageSize: 5 }),
		select: (res) => res.data,
		enabled: canOrders
	}));

	const displayName = $derived(session.me?.name || session.me?.email || '');
</script>

{#snippet stat(query: { isError: boolean; data: number | undefined })}
	{#if query.isError && query.data === undefined}
		<span class="text-slate-400">—</span>
	{:else if query.data === undefined}
		<span class="inline-block h-8 w-12 animate-pulse rounded bg-slate-200"></span>
	{:else}
		{query.data}
	{/if}
{/snippet}

<PageHeader title={m.dash_welcome({ name: displayName })} subtitle={m.dash_subtitle()} />

<section class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4" aria-label={m.dash_title()}>
	<a href={resolve('/products')} class="card p-5 transition hover:shadow-md">
		<p class="flex items-center gap-2 text-sm font-medium text-slate-500">
			<Icon name="box" size={16} />{m.dash_products()}
		</p>
		<p class="mt-2 text-3xl font-bold text-slate-900">{@render stat(totalProducts)}</p>
	</a>
	<a href={resolve('/products?status=published')} class="card p-5 transition hover:shadow-md">
		<p class="text-sm font-medium text-slate-500">{m.dash_published()}</p>
		<p class="mt-2 text-3xl font-bold text-emerald-700">{@render stat(published)}</p>
	</a>
	<a href={resolve('/products?status=draft')} class="card p-5 transition hover:shadow-md">
		<p class="text-sm font-medium text-slate-500">{m.dash_draft()}</p>
		<p class="mt-2 text-3xl font-bold text-amber-700">{@render stat(drafts)}</p>
	</a>
	{#if canOrders}
		<a href={resolve('/orders?status=pending')} class="card p-5 transition hover:shadow-md">
			<p class="flex items-center gap-2 text-sm font-medium text-slate-500">
				<Icon name="cart" size={16} />{m.dash_pending_orders()}
			</p>
			<p class="mt-2 text-3xl font-bold text-slate-900">{@render stat(pendingOrders)}</p>
		</a>
	{/if}
</section>

<section class="mt-6" aria-labelledby="quick-actions">
	<h2 id="quick-actions" class="mb-3 text-sm font-semibold tracking-wide text-slate-500 uppercase">
		{m.dash_quick_actions()}
	</h2>
	<div class="flex flex-wrap gap-2">
		{#if session.can('products:write')}
			<a class="btn-primary" href={resolve('/products/new')}
				><Icon name="plus" />{m.dash_add_product()}</a
			>
		{/if}
		<a class="btn-secondary" href={resolve('/categories')}
			><Icon name="tag" />{m.dash_manage_categories()}</a
		>
		{#if session.can('staff:manage')}
			<a class="btn-secondary" href={resolve('/staff')}
				><Icon name="users" />{m.dash_invite_staff()}</a
			>
		{/if}
		{#if session.can('store:write')}
			<a class="btn-secondary" href={resolve('/store')}
				><Icon name="store" />{m.dash_edit_store()}</a
			>
		{/if}
	</div>
</section>

<div class="mt-6 grid gap-6 lg:grid-cols-2">
	<section class="card" aria-labelledby="low-stock">
		<header class="flex items-center justify-between border-b border-slate-200 px-5 py-3">
			<h2 id="low-stock" class="font-semibold">{m.dash_low_stock()}</h2>
			<a class="text-sm text-brand-700 hover:underline" href={resolve('/products?sort=stock_asc')}
				>{m.dash_view_all()}</a
			>
		</header>
		{#if lowStock.isPending}
			<p class="p-5 text-sm text-slate-500" role="status">{m.common_loading()}</p>
		{:else if lowStock.isError}
			<p class="p-5 text-sm text-red-600">{m.products_load_failed()}</p>
		{:else if lowStock.data.length === 0}
			<p class="p-5 text-sm text-slate-500">{m.dash_no_products()}</p>
		{:else}
			<ul class="divide-y divide-slate-100">
				{#each lowStock.data as product (product.id)}
					{@const cover = safeImageUrl(product.images[0]?.url)}
					<li>
						<a
							href={resolve(`/products/${product.id}`)}
							class="flex items-center gap-3 px-5 py-3 hover:bg-slate-50"
						>
							{#if cover}
								<img src={cover} alt="" class="size-10 rounded object-cover" loading="lazy" />
							{:else}
								<span class="flex size-10 items-center justify-center rounded bg-slate-100">
									<Icon name="image" size={16} class="text-slate-400" />
								</span>
							{/if}
							<span class="min-w-0 flex-1 truncate text-sm font-medium"
								>{localized(product.name)}</span
							>
							<span
								class={[
									'text-sm font-semibold',
									product.stock === 0
										? 'text-red-600'
										: product.stock < 5
											? 'text-amber-600'
											: 'text-slate-600'
								]}>{m.dash_in_stock({ count: product.stock })}</span
							>
						</a>
					</li>
				{/each}
			</ul>
		{/if}
	</section>

	{#if canOrders}
		<section class="card" aria-labelledby="recent-orders">
			<header class="flex items-center justify-between border-b border-slate-200 px-5 py-3">
				<h2 id="recent-orders" class="font-semibold">{m.dash_recent_orders()}</h2>
				<a class="text-sm text-brand-700 hover:underline" href={resolve('/orders')}
					>{m.dash_view_all()}</a
				>
			</header>
			{#if recentOrders.isPending}
				<p class="p-5 text-sm text-slate-500" role="status">{m.common_loading()}</p>
			{:else if recentOrders.isError}
				<p class="p-5 text-sm text-red-600">{m.orders_load_failed()}</p>
			{:else if recentOrders.data.length === 0}
				<p class="p-5 text-sm text-slate-500">{m.dash_no_orders()}</p>
			{:else}
				<ul class="divide-y divide-slate-100">
					{#each recentOrders.data as order (order.id)}
						<li>
							<a
								href={resolve(`/orders/${order.id}`)}
								class="flex flex-wrap items-center gap-x-3 gap-y-1 px-5 py-3 hover:bg-slate-50"
							>
								<span class="font-mono text-sm font-medium">{order.number}</span>
								<StatusBadge kind="order" value={order.status} />
								<span class="min-w-0 flex-1 truncate text-sm text-slate-600"
									>{order.customer.name}</span
								>
								<span class="text-sm font-semibold">{formatCAD(order.totalCents)}</span>
								<span class="w-full text-xs text-slate-500">{formatDateTime(order.createdAt)}</span>
							</a>
						</li>
					{/each}
				</ul>
			{/if}
		</section>
	{/if}
</div>
