import { describe, expect, it } from 'vitest';
import { getPageItems } from './pagination';

describe('getPageItems', () => {
	it('hides pagination for a single page or none', () => {
		expect(getPageItems(1, 0)).toEqual([]);
		expect(getPageItems(1, 1)).toEqual([]);
	});

	it('shows every page when there are 5 or fewer', () => {
		expect(getPageItems(1, 2)).toEqual([1, 2]);
		expect(getPageItems(3, 5)).toEqual([1, 2, 3, 4, 5]);
	});

	it('shows 1 … cur±1 … last for many pages', () => {
		expect(getPageItems(1, 10)).toEqual([1, 2, 'ellipsis-end', 10]);
		expect(getPageItems(3, 10)).toEqual([1, 2, 3, 4, 'ellipsis-end', 10]);
		expect(getPageItems(4, 10)).toEqual([1, 'ellipsis-start', 3, 4, 5, 'ellipsis-end', 10]);
		expect(getPageItems(8, 10)).toEqual([1, 'ellipsis-start', 7, 8, 9, 10]);
		expect(getPageItems(10, 10)).toEqual([1, 'ellipsis-start', 9, 10]);
	});

	it('clamps out-of-range current pages', () => {
		expect(getPageItems(99, 6)).toEqual([1, 'ellipsis-start', 5, 6]);
		expect(getPageItems(-3, 6)).toEqual([1, 2, 'ellipsis-end', 6]);
	});
});
