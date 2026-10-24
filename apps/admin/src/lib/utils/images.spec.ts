import { describe, expect, it } from 'vitest';
import { imageUrl } from './images';

describe('imageUrl', () => {
	const img = {
		url: 'full.webp',
		variants: [
			{ url: 'v400.webp', width: 400 },
			{ url: 'v800.webp', width: 800 }
		]
	};
	it('picks the smallest copy that is large enough', () => {
		expect(imageUrl(img, 96)).toBe('v400.webp');
		expect(imageUrl(img, 600)).toBe('v800.webp');
		expect(imageUrl(img, 1200)).toBe('full.webp');
	});
	it('falls back to the image, or nothing', () => {
		expect(imageUrl({ url: 'only.webp' }, 96)).toBe('only.webp');
		expect(imageUrl(undefined, 96)).toBeUndefined();
	});
});
