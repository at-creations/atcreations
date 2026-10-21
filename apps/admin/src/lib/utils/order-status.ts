import type { OrderStatus } from '@atcreations/api-types';

export const ORDER_STATUSES: readonly OrderStatus[] = [
	'pending',
	'confirmed',
	'processing',
	'shipped',
	'delivered',
	'completed',
	'cancelled',
	'refunded'
];

/**
 * State machine from the OpenAPI `OrderStatus` description:
 * pending→confirmed|cancelled, confirmed→processing|cancelled, processing→shipped|cancelled,
 * shipped→delivered, delivered→completed|refunded, completed→refunded.
 */
const TRANSITIONS: Record<OrderStatus, readonly OrderStatus[]> = {
	pending: ['confirmed', 'cancelled'],
	confirmed: ['processing', 'cancelled'],
	processing: ['shipped', 'cancelled'],
	shipped: ['delivered'],
	delivered: ['completed', 'refunded'],
	completed: ['refunded'],
	cancelled: [],
	refunded: []
};

export function allowedNextStatuses(status: OrderStatus): readonly OrderStatus[] {
	return TRANSITIONS[status] ?? [];
}

export function canTransition(from: OrderStatus, to: OrderStatus): boolean {
	return allowedNextStatuses(from).includes(to);
}

/** The shipping address may only be edited before the order ships (and not when cancelled/refunded). */
export function isShippingEditable(status: OrderStatus): boolean {
	return !['shipped', 'delivered', 'completed', 'cancelled', 'refunded'].includes(status);
}

export function isOrderStatus(value: unknown): value is OrderStatus {
	return typeof value === 'string' && (ORDER_STATUSES as readonly string[]).includes(value);
}
