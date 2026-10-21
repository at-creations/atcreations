import { describe, expect, it } from 'vitest';
import { readEnum, readInt, withParams } from './query';

describe('query helpers', () => {
	const params = new URLSearchParams('page=3&pageSize=25&status=draft&bad=-1');
	it('reads validated integers', () => {
		expect(readInt(params, 'page', 1)).toBe(3);
		expect(readInt(params, 'pageSize', 10, [10, 25, 50])).toBe(25);
		expect(readInt(params, 'pageSize', 10, [10, 50])).toBe(10);
		expect(readInt(params, 'bad', 1)).toBe(1);
		expect(readInt(params, 'missing', 7)).toBe(7);
	});
	it('reads enums', () => {
		expect(readEnum(params, 'status', ['draft', 'published'])).toBe('draft');
		expect(readEnum(params, 'status', ['published'])).toBeUndefined();
	});
	it('builds URLs and resets the page on filter changes', () => {
		const url = new URL('http://x/products?page=3&q=tea');
		expect(withParams(url, { status: 'draft' })).toBe('/products?q=tea&status=draft');
		expect(withParams(url, { page: 4 })).toBe('/products?page=4&q=tea');
		expect(withParams(url, { q: '' })).toBe('/products');
	});
});
