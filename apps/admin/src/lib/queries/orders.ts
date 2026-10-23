import type { Address, Order, OrderStatus } from '@atcreations/api-types';
import {
	createMutation,
	queryOptions,
	useQueryClient,
	type QueryClient
} from '@tanstack/svelte-query';
import { api, unwrap } from '$lib/api';
import { keys, type OrderListParams } from './keys';

/** How often order data is polled while the tab is visible: new orders arrive without a reload. */
export const ORDERS_POLL_MS = 30_000;

export function orderListQuery(params: OrderListParams) {
	return queryOptions({
		queryKey: keys.orders.list(params),
		refetchInterval: ORDERS_POLL_MS,
		queryFn: ({ signal }) =>
			unwrap(api().GET('/v1/admin/orders', { params: { query: params }, signal }))
	});
}

export function orderDetailQuery(id: string) {
	return queryOptions({
		queryKey: keys.orders.detail(id),
		refetchInterval: ORDERS_POLL_MS,
		queryFn: async ({ signal }) =>
			(await unwrap(api().GET('/v1/admin/orders/{id}', { params: { path: { id } }, signal }))).data
	});
}

async function putOrder(queryClient: QueryClient, order: Order, refreshLists: boolean) {
	const queryKey = keys.orders.detail(order.id);
	await queryClient.cancelQueries({ queryKey, exact: true });
	queryClient.setQueryData<Order>(queryKey, order);
	if (refreshLists) void queryClient.invalidateQueries({ queryKey: keys.orders.lists() });
}

export interface StatusChange {
	id: string;
	status: OrderStatus;
	note?: string;
	trackingNumber?: string;
	notify: boolean;
}

export function createOrderStatusMutation() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: async ({ id, ...body }: StatusChange) =>
			(await unwrap(api().POST('/v1/admin/orders/{id}/status', { params: { path: { id } }, body })))
				.data,
		onSuccess: async (order) => {
			await putOrder(queryClient, order, true);
			// Cancellations/refunds can restock products (and bump their version).
			void queryClient.invalidateQueries({ queryKey: keys.products.all });
		}
	}));
}

export function createOrderNoteMutation() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: async ({ id, staffNote }: { id: string; staffNote: string }) =>
			(
				await unwrap(
					api().PUT('/v1/admin/orders/{id}/notes', {
						params: { path: { id } },
						body: { staffNote }
					})
				)
			).data,
		onSuccess: (order) => putOrder(queryClient, order, false)
	}));
}

export function createOrderShippingMutation() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: async ({ id, address }: { id: string; address: Address }) =>
			(
				await unwrap(
					api().PUT('/v1/admin/orders/{id}/shipping', { params: { path: { id } }, body: address })
				)
			).data,
		onSuccess: (order) => putOrder(queryClient, order, false)
	}));
}

/** Number of orders waiting for a decision (sidebar badge, dashboard). */
export function pendingOrderCountQuery() {
	return queryOptions({
		...orderListQuery({ status: 'pending', pageSize: 1 }),
		select: (res) => res.meta.total
	});
}
