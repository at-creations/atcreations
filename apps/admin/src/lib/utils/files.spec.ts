import { describe, expect, it } from 'vitest';
import { checkImageFile, MAX_IMAGE_BYTES, moveItem } from './files';

describe('checkImageFile', () => {
	it('accepts JPEG, PNG and WebP up to 10 MB', () => {
		expect(checkImageFile({ type: 'image/jpeg', size: 1000 })).toBeNull();
		expect(checkImageFile({ type: 'image/png', size: MAX_IMAGE_BYTES })).toBeNull();
		expect(checkImageFile({ type: 'image/webp', size: 1 })).toBeNull();
	});
	it('rejects other types and oversized files', () => {
		expect(checkImageFile({ type: 'image/gif', size: 10 })).toBe('type');
		expect(checkImageFile({ type: 'image/svg+xml', size: 10 })).toBe('type');
		expect(checkImageFile({ type: 'image/jpeg', size: MAX_IMAGE_BYTES + 1 })).toBe('size');
	});
});

describe('moveItem', () => {
	it('moves items and ignores out-of-range moves', () => {
		expect(moveItem(['a', 'b', 'c'], 0, 2)).toEqual(['b', 'c', 'a']);
		expect(moveItem(['a', 'b', 'c'], 2, 0)).toEqual(['c', 'a', 'b']);
		expect(moveItem(['a', 'b'], 0, 5)).toEqual(['a', 'b']);
	});
});
