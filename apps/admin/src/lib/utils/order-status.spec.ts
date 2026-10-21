import { describe, expect, it } from 'vitest';
import {
	allowedNextStatuses,
	canTransition,
	isOrderStatus,
	isShippingEditable
} from './order-status';

describe('allowedNextStatuses', () => {
	it('follows the documented state machine', () => {
		expect(allowedNextStatuses('pending')).toEqual(['confirmed', 'cancelled']);
		expect(allowedNextStatuses('confirmed')).toEqual(['processing', 'cancelled']);
		expect(allowedNextStatuses('processing')).toEqual(['shipped', 'cancelled']);
		expect(allowedNextStatuses('shipped')).toEqual(['delivered']);
		expect(allowedNextStatuses('delivered')).toEqual(['completed', 'refunded']);
		expect(allowedNextStatuses('completed')).toEqual(['refunded']);
	});
	it('treats cancelled and refunded as terminal', () => {
		expect(allowedNextStatuses('cancelled')).toEqual([]);
		expect(allowedNextStatuses('refunded')).toEqual([]);
	});
});

describe('canTransition', () => {
	it('rejects skipping or reversing states', () => {
		expect(canTransition('pending', 'shipped')).toBe(false);
		expect(canTransition('shipped', 'cancelled')).toBe(false);
		expect(canTransition('delivered', 'pending')).toBe(false);
		expect(canTransition('processing', 'shipped')).toBe(true);
	});
});

describe('isShippingEditable', () => {
	it('locks the address once shipped, cancelled or refunded', () => {
		expect(isShippingEditable('pending')).toBe(true);
		expect(isShippingEditable('processing')).toBe(true);
		expect(isShippingEditable('shipped')).toBe(false);
		expect(isShippingEditable('cancelled')).toBe(false);
		expect(isShippingEditable('refunded')).toBe(false);
	});
});

describe('isOrderStatus', () => {
	it('validates values', () => {
		expect(isOrderStatus('pending')).toBe(true);
		expect(isOrderStatus('lost')).toBe(false);
	});
});
