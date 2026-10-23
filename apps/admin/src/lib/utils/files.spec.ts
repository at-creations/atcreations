import { describe, expect, it } from 'vitest';
import { moveItem } from './files';

describe('moveItem', () => {
	it('moves items and ignores out-of-range moves', () => {
		expect(moveItem(['a', 'b', 'c'], 0, 2)).toEqual(['b', 'c', 'a']);
		expect(moveItem(['a', 'b', 'c'], 2, 0)).toEqual(['c', 'a', 'b']);
		expect(moveItem(['a', 'b'], 0, 5)).toEqual(['a', 'b']);
	});
});
