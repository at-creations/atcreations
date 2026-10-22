import { hashKey, partialMatchKey } from '@tanstack/svelte-query';
import { describe, expect, it } from 'vitest';
import { keys, normalizeParams } from './keys';

describe('query keys', () => {
	it('are stable for equal params regardless of order or empty values', () => {
		const a = keys.products.list({ page: 2, q: 'tea', status: 'draft' });
		const b = keys.products.list({ status: 'draft', q: 'tea', page: 2, category: undefined });
		const c = keys.products.list({ page: 2, q: 'tea', status: 'draft', category: '' as string });
		expect(a).toEqual(b);
		expect(hashKey(a)).toBe(hashKey(b));
		expect(hashKey(a)).toBe(hashKey(c));
		expect(keys.products.list()).toEqual(keys.products.list({}));
	});

	it('differ when params differ', () => {
		expect(hashKey(keys.products.list({ page: 1 }))).not.toBe(
			hashKey(keys.products.list({ page: 2 }))
		);
		expect(hashKey(keys.orders.detail('a'))).not.toBe(hashKey(keys.orders.detail('b')));
	});

	it('nest under their domain so prefix invalidation works', () => {
		const list = keys.products.list({ page: 1 });
		const detail = keys.products.detail('p1');
		expect(partialMatchKey(list, keys.products.all)).toBe(true);
		expect(partialMatchKey(list, keys.products.lists())).toBe(true);
		expect(partialMatchKey(detail, keys.products.all)).toBe(true);
		expect(partialMatchKey(detail, keys.products.details())).toBe(true);
		// Invalidating lists must not touch the product being edited.
		expect(partialMatchKey(detail, keys.products.lists())).toBe(false);
		expect(partialMatchKey(keys.staff.invitations, keys.staff.all)).toBe(true);
		expect(partialMatchKey(keys.staff.invitations, keys.staff.lists())).toBe(false);
		expect(partialMatchKey(keys.audit.list({ page: 1 }), keys.audit.all)).toBe(true);
		expect(partialMatchKey(keys.orders.list({}), keys.products.all)).toBe(false);
	});

	it('have the documented shape', () => {
		expect(keys.me).toEqual(['me']);
		expect(keys.categories.all).toEqual(['categories']);
		expect(keys.storeInfo).toEqual(['store-info']);
		expect(keys.products.detail('x')).toEqual(['products', 'detail', 'x']);
		expect(keys.orders.list({ status: 'pending', pageSize: 1 })).toEqual([
			'orders',
			'list',
			{ pageSize: 1, status: 'pending' }
		]);
		expect(keys.staff.list({ q: '' })).toEqual(['staff', 'list', {}]);
	});

	it('normalizeParams drops empty values and keeps false/0', () => {
		expect(
			normalizeParams({ a: undefined, b: null, c: '', d: 0, e: false, f: 'x' } as Record<
				string,
				unknown
			>)
		).toEqual({ d: 0, e: false, f: 'x' });
	});
});
