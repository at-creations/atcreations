<script lang="ts">
	import type { Order, OrderStatus } from '@atcreations/api-types';
	import { createQuery, keepPreviousData } from '@tanstack/svelte-query';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import FilterTabs from '$lib/components/FilterTabs.svelte';
	import SearchInput from '$lib/components/ui/SearchInput.svelte';
	import LoadState from '$lib/components/LoadState.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Pagination from '$lib/components/Pagination.svelte';
	import RequirePermission from '$lib/components/RequirePermission.svelte';
	import StatusBadge from '$lib/components/StatusBadge.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import { formatDateTime, orderStatusLabel } from '$lib/i18n';
	import { m } from '$lib/paraglide/messages.js';
	import { orderListQuery, pendingOrderCountQuery, type OrderListParams } from '$lib/queries';
	import { getAdminSession } from '$lib/state/session.svelte';
	import { formatCAD } from '$lib/utils/money';
	import { ORDER_STATUSES } from '$lib/utils/order-status';
	import { ISO_DATE, readEnum, readInt, readString, withParams } from '$lib/utils/query';

	const session = getAdminSession();
	const SORTS = ['newest', 'oldest', 'total_desc', 'total_asc'] as const;
	type Sort = (typeof SORTS)[number];
	const sortLabels: Record<Sort, () => string> = {
		newest: m.sort_newest,
		oldest: m.sort_oldest,
		total_desc: m.sort_total_desc,
		total_asc: m.sort_total_asc
	};
	const PAGE_SIZES = [10, 25, 50] as const;
	const uid = $props.id();

	const filters = $derived.by(() => {
		const p = page.url.searchParams;
		const from = readString(p, 'from', 10);
		const to = readString(p, 'to', 10);
		return {
			page: readInt(p, 'page', 1),
			pageSize: readInt(p, 'pageSize', 25, PAGE_SIZES),
			q: readString(p, 'q'),
			status: readEnum<OrderStatus>(p, 'status', ORDER_STATUSES),
			from: ISO_DATE.test(from) ? from : '',
			to: ISO_DATE.test(to) ? to : '',
			sort: readEnum(p, 'sort', SORTS, 'newest') ?? 'newest'
		};
	});

	const dateError = $derived(
		filters.from && filters.to && filters.from > filters.to ? m.orders_date_invalid() : ''
	);

	const listParams = $derived<OrderListParams>({
		page: filters.page,
		pageSize: filters.pageSize,
		q: filters.q || undefined,
		status: filters.status,
		from: filters.from || undefined,
		to: filters.to && !(filters.from && filters.from > filters.to) ? filters.to : undefined,
		sort: filters.sort
	});

	const ordersQuery = createQuery(() => ({
		...orderListQuery(listParams),
		placeholderData: keepPreviousData,
		enabled: session.can('orders:read')
	}));
	const orders = $derived(ordersQuery.data?.data ?? []);
	const meta = $derived(ordersQuery.data?.meta ?? null);
	const showError = $derived(ordersQuery.isError && !ordersQuery.data);

	function navigate(changes: Record<string, string | number | null | undefined>) {
		// Same page, only the query string changes (built from page.url).
		// eslint-disable-next-line svelte/no-navigation-without-resolve
		goto(withParams(page.url, changes), { keepFocus: true, noScroll: true });
	}

	let searchInput = $derived(filters.q);

	const itemCount = (order: Order) => order.items.reduce((n, i) => n + i.quantity, 0);

	const pendingCount = createQuery(() => ({
		...pendingOrderCountQuery(),
		enabled: session.can('orders:read')
	}));
	const statusTabs = $derived([
		{ value: '' as const, label: m.common_all() },
		...ORDER_STATUSES.map((status) => ({
			value: status,
			label: orderStatusLabel(status),
			// Only "pending" needs action, so only it carries a counter.
			count: status === 'pending' ? (pendingCount.data ?? null) : undefined
		}))
	]);
	const hasFilters = $derived(!!(filters.q || filters.status || filters.from || filters.to));

	function clearFilters() {
		searchInput = '';
		goto(resolve('/orders'), { keepFocus: true, noScroll: true });
	}

	/** Row click opens the order, unless the click was on a link or text was selected. */
	function openRow(event: MouseEvent, order: Order) {
		const target = event.target as HTMLElement;
		if (target.closest('a, button') || window.getSelection()?.toString()) return;
		goto(resolve(`/orders/${order.id}`));
	}
</script>

<RequirePermission permission="orders:read">
	<PageHeader title={m.orders_title()} />

	<section class="card" aria-busy={ordersQuery.isFetching}>
		<div class="border-b border-zinc-200 px-3">
			<FilterTabs
				tabs={statusTabs}
				value={filters.status ?? ''}
				label={m.orders_col_status()}
				onselect={(status) => navigate({ status })}
			/>
		</div>

		<div
			class="flex flex-col gap-2 p-3 lg:flex-row lg:items-center"
			aria-label={m.common_filters()}
		>
			<SearchInput
				bind:value={searchInput}
				class="flex-1 lg:max-w-sm"
				placeholder={m.orders_search_placeholder()}
				onsearch={(q) => navigate({ q })}
			/>
			<div class="flex flex-wrap items-center gap-2">
				<div
					class="flex h-9 items-center rounded-lg border border-zinc-300 bg-white shadow-xs focus-within:border-brand-600 focus-within:ring-3 focus-within:ring-brand-600/15"
				>
					<label for="{uid}-from" class="sr-only">{m.orders_from()}</label>
					<input
						id="{uid}-from"
						type="date"
						class="h-full rounded-l-lg bg-transparent px-2.5 text-sm focus:outline-none"
						title={m.orders_from()}
						value={filters.from}
						max={filters.to || undefined}
						aria-invalid={dateError ? 'true' : undefined}
						aria-describedby={dateError ? `${uid}-date-err` : undefined}
						onchange={(e) => navigate({ from: e.currentTarget.value })}
					/>
					<span class="text-zinc-300" aria-hidden="true">→</span>
					<label for="{uid}-to" class="sr-only">{m.orders_to()}</label>
					<input
						id="{uid}-to"
						type="date"
						class="h-full rounded-r-lg bg-transparent px-2.5 text-sm focus:outline-none"
						title={m.orders_to()}
						value={filters.to}
						min={filters.from || undefined}
						aria-invalid={dateError ? 'true' : undefined}
						onchange={(e) => navigate({ to: e.currentTarget.value })}
					/>
				</div>
				<Select
					label={m.common_sort_by()}
					class="w-auto min-w-40"
					value={filters.sort}
					items={SORTS.map((s) => ({ value: s, label: sortLabels[s]() }))}
					onchange={(v) => navigate({ sort: v === 'newest' ? null : v })}
				/>
				{#if hasFilters}
					<button type="button" class="btn-ghost" onclick={clearFilters}
						>{m.products_clear_filters()}</button
					>
				{/if}
			</div>
		</div>
		{#if dateError}
			<p id="{uid}-date-err" class="px-3 pb-3 text-xs text-red-600">{dateError}</p>
		{/if}

		<div class="border-t border-zinc-200">
			{#if showError}
				<LoadState
					error={ordersQuery.error}
					fallback={m.orders_load_failed()}
					onretry={() => ordersQuery.refetch()}
				/>
			{:else if ordersQuery.isPending}
				<div class="divide-y divide-zinc-100" role="status" aria-label={m.common_loading()}>
					{#each [0, 1, 2, 3, 4] as i (i)}
						<div class="flex items-center gap-3 px-5 py-3.5">
							<div class="size-9 animate-pulse rounded-full bg-zinc-100"></div>
							<div class="h-3.5 w-1/3 animate-pulse rounded bg-zinc-100"></div>
						</div>
					{/each}
				</div>
			{:else if orders.length === 0}
				{#if hasFilters}
					<EmptyState icon="search" title={m.orders_none()}>
						<button type="button" class="btn-secondary" onclick={clearFilters}
							>{m.products_clear_filters()}</button
						>
					</EmptyState>
				{:else}
					<EmptyState icon="inbox" title={m.dash_no_orders()} />
				{/if}
			{:else}
				<div class="overflow-x-auto">
					<table
						class={[
							'w-full min-w-[720px] text-left text-sm',
							ordersQuery.isPlaceholderData && 'opacity-60'
						]}
					>
						<thead class="border-b border-zinc-200 bg-zinc-50/80">
							<tr>
								<th scope="col" class="th pl-5">{m.orders_col_number()}</th>
								<th scope="col" class="th">{m.orders_col_customer()}</th>
								<th scope="col" class="th">{m.orders_col_status()}</th>
								<th scope="col" class="th text-right">{m.orders_col_items()}</th>
								<th scope="col" class="th text-right">{m.orders_col_total()}</th>
								<th scope="col" class="th pr-5">{m.orders_col_created()}</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-zinc-100">
							{#each orders as order (order.id)}
								<tr
									class={[
										'group cursor-pointer transition-colors hover:bg-zinc-50',
										order.status === 'pending' && 'bg-amber-50/40'
									]}
									onclick={(e) => openRow(e, order)}
								>
									<td class="td pl-5">
										<a
											href={resolve(`/orders/${order.id}`)}
											class="font-mono text-sm font-medium text-zinc-900 group-hover:text-brand-700"
											>{order.number}</a
										>
									</td>
									<td class="td">
										<div class="flex items-center gap-3">
											<span
												class="flex size-8 shrink-0 items-center justify-center rounded-full bg-zinc-100 text-[11px] font-semibold text-zinc-600 uppercase"
												aria-hidden="true">{order.customer.name.slice(0, 2)}</span
											>
											<div class="min-w-0">
												<span class="block truncate font-medium text-zinc-900"
													>{order.customer.name}</span
												>
												<span class="block truncate text-xs text-zinc-500"
													>{order.customer.email}</span
												>
											</div>
										</div>
									</td>
									<td class="td"><StatusBadge kind="order" value={order.status} /></td>
									<td class="td text-right text-zinc-600 tabular-nums">{itemCount(order)}</td>
									<td class="td text-right font-semibold text-zinc-900 tabular-nums"
										>{formatCAD(order.totalCents)}</td
									>
									<td class="td pr-5 whitespace-nowrap text-zinc-500"
										>{formatDateTime(order.createdAt)}</td
									>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</div>
		{#if meta && !showError && orders.length > 0}
			<Pagination
				{meta}
				pageSizes={[...PAGE_SIZES]}
				onpage={(p) => navigate({ page: p })}
				onpagesize={(size) => navigate({ pageSize: size })}
			/>
		{/if}
	</section>
</RequirePermission>
