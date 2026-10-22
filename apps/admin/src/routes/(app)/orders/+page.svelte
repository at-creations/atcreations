<script lang="ts">
	import type { Order, OrderStatus } from '@atcreations/api-types';
	import { createQuery, keepPreviousData } from '@tanstack/svelte-query';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import Icon from '$lib/components/Icon.svelte';
	import LoadState from '$lib/components/LoadState.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Pagination from '$lib/components/Pagination.svelte';
	import RequirePermission from '$lib/components/RequirePermission.svelte';
	import StatusBadge from '$lib/components/StatusBadge.svelte';
	import { formatDateTime, orderStatusLabel } from '$lib/i18n';
	import { m } from '$lib/paraglide/messages.js';
	import { orderListQuery, type OrderListParams } from '$lib/queries';
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
	let timer: ReturnType<typeof setTimeout> | undefined;
	function onSearch(value: string) {
		searchInput = value;
		clearTimeout(timer);
		timer = setTimeout(() => navigate({ q: value.trim() }), 300);
	}
	$effect(() => () => clearTimeout(timer));

	const itemCount = (order: Order) => order.items.reduce((n, i) => n + i.quantity, 0);
</script>

<RequirePermission permission="orders:read">
	<PageHeader title={m.orders_title()} />

	<section class="mb-4 card p-4" aria-label={m.common_filters()}>
		<div class="grid gap-3 md:grid-cols-2 xl:grid-cols-6">
			<div class="relative md:col-span-2">
				<label for="{uid}-q" class="sr-only">{m.common_search()}</label>
				<Icon
					name="search"
					size={16}
					class="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-slate-400"
				/>
				<input
					id="{uid}-q"
					type="search"
					class="input pl-9"
					maxlength="100"
					placeholder={m.orders_search_placeholder()}
					value={searchInput}
					oninput={(e) => onSearch(e.currentTarget.value)}
				/>
			</div>
			<div>
				<label for="{uid}-status" class="sr-only">{m.orders_col_status()}</label>
				<select
					id="{uid}-status"
					class="input"
					value={filters.status ?? ''}
					onchange={(e) => navigate({ status: e.currentTarget.value })}
				>
					<option value="">{m.orders_all_statuses()}</option>
					{#each ORDER_STATUSES as status (status)}
						<option value={status}>{orderStatusLabel(status)}</option>
					{/each}
				</select>
			</div>
			<div>
				<label for="{uid}-from" class="sr-only">{m.orders_from()}</label>
				<input
					id="{uid}-from"
					type="date"
					class="input"
					title={m.orders_from()}
					value={filters.from}
					max={filters.to || undefined}
					aria-invalid={dateError ? 'true' : undefined}
					aria-describedby={dateError ? `${uid}-date-err` : undefined}
					onchange={(e) => navigate({ from: e.currentTarget.value })}
				/>
			</div>
			<div>
				<label for="{uid}-to" class="sr-only">{m.orders_to()}</label>
				<input
					id="{uid}-to"
					type="date"
					class="input"
					title={m.orders_to()}
					value={filters.to}
					min={filters.from || undefined}
					aria-invalid={dateError ? 'true' : undefined}
					onchange={(e) => navigate({ to: e.currentTarget.value })}
				/>
			</div>
			<div>
				<label for="{uid}-sort" class="sr-only">{m.common_sort_by()}</label>
				<select
					id="{uid}-sort"
					class="input"
					value={filters.sort}
					onchange={(e) =>
						navigate({ sort: e.currentTarget.value === 'newest' ? null : e.currentTarget.value })}
				>
					{#each SORTS as sort (sort)}
						<option value={sort}>{sortLabels[sort]()}</option>
					{/each}
				</select>
			</div>
		</div>
		{#if dateError}
			<p id="{uid}-date-err" class="mt-2 text-xs text-red-600">{dateError}</p>
		{/if}
	</section>

	<section class="overflow-hidden card" aria-busy={ordersQuery.isFetching}>
		{#if showError}
			<LoadState
				error={ordersQuery.error}
				fallback={m.orders_load_failed()}
				onretry={() => ordersQuery.refetch()}
			/>
		{:else if ordersQuery.isPending}
			<LoadState loading />
		{:else if orders.length === 0}
			<p class="p-10 text-center text-slate-500">{m.orders_none()}</p>
		{:else}
			<div class="overflow-x-auto">
				<table
					class={[
						'w-full min-w-[720px] text-left text-sm',
						ordersQuery.isPlaceholderData && 'opacity-60'
					]}
				>
					<thead class="bg-slate-50 text-xs font-semibold tracking-wide text-slate-500 uppercase">
						<tr>
							<th scope="col" class="px-4 py-3">{m.orders_col_number()}</th>
							<th scope="col" class="px-4 py-3">{m.orders_col_customer()}</th>
							<th scope="col" class="px-4 py-3">{m.orders_col_status()}</th>
							<th scope="col" class="px-4 py-3 text-right">{m.orders_col_total()}</th>
							<th scope="col" class="px-4 py-3 text-right">{m.orders_col_items()}</th>
							<th scope="col" class="px-4 py-3">{m.orders_col_created()}</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-slate-100">
						{#each orders as order (order.id)}
							<tr class="hover:bg-slate-50">
								<td class="px-4 py-3">
									<a
										href={resolve(`/orders/${order.id}`)}
										class="font-mono font-medium text-brand-700 hover:underline">{order.number}</a
									>
								</td>
								<td class="px-4 py-3">
									<span class="block font-medium">{order.customer.name}</span>
									<span class="block text-xs text-slate-500">{order.customer.email}</span>
								</td>
								<td class="px-4 py-3"><StatusBadge kind="order" value={order.status} /></td>
								<td class="px-4 py-3 text-right font-medium">{formatCAD(order.totalCents)}</td>
								<td class="px-4 py-3 text-right">{itemCount(order)}</td>
								<td class="px-4 py-3 text-slate-600">{formatDateTime(order.createdAt)}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
		{#if meta && !showError}
			<Pagination
				{meta}
				pageSizes={[...PAGE_SIZES]}
				onpage={(p) => navigate({ page: p })}
				onpagesize={(size) => navigate({ pageSize: size })}
			/>
		{/if}
	</section>
</RequirePermission>
