import { describe, expect, it } from 'vitest';
import { sized, srcset } from './images';

const img = {
	url: 'https://cdn.test/a.webp',
	width: 2400,
	variants: [
		{ url: 'https://cdn.test/a-400.webp', width: 400 },
		{ url: 'https://cdn.test/a-800.webp', width: 800 }
	]
};

describe('responsive images', () => {
	it('lists every copy with its width', () => {
		expect(srcset(img)).toBe(
			'https://cdn.test/a-400.webp 400w, https://cdn.test/a-800.webp 800w, https://cdn.test/a.webp 2400w'
		);
		expect(srcset({ url: 'x', width: 10 })).toBeUndefined();
	});
	it('picks the smallest copy that is large enough', () => {
		expect(sized(img, 160)).toBe('https://cdn.test/a-400.webp');
		expect(sized(img, 500)).toBe('https://cdn.test/a-800.webp');
		expect(sized(img, 1000)).toBe('https://cdn.test/a.webp');
		expect(sized({ url: 'x' }, 100)).toBe('x');
	});
});
