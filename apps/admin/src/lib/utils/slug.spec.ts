import { describe, expect, it } from 'vitest';
import { isValidSlug, slugify } from './slug';

describe('slugify', () => {
	it('lowercases and hyphenates', () => {
		expect(slugify('Hello World')).toBe('hello-world');
		expect(slugify('  Silk  Scarf -- Blue ')).toBe('silk-scarf-blue');
		expect(slugify('Tea & Cups')).toBe('tea-and-cups');
	});
	it('strips Vietnamese diacritics', () => {
		expect(slugify('Áo dài truyền thống')).toBe('ao-dai-truyen-thong');
		expect(slugify('Đèn lồng Hội An')).toBe('den-long-hoi-an');
	});
	it('handles empty and symbol-only input', () => {
		expect(slugify('')).toBe('');
		expect(slugify('!!!')).toBe('');
	});
	it('truncates without a trailing hyphen', () => {
		expect(slugify('abc def', 4)).toBe('abc');
		expect(slugify('a'.repeat(200)).length).toBe(120);
	});
});

describe('isValidSlug', () => {
	it('validates the API pattern', () => {
		expect(isValidSlug('silk-scarf')).toBe(true);
		expect(isValidSlug('a1')).toBe(true);
		expect(isValidSlug('Silk')).toBe(false);
		expect(isValidSlug('-silk')).toBe(false);
		expect(isValidSlug('silk--scarf')).toBe(false);
		expect(isValidSlug('')).toBe(false);
		expect(isValidSlug('abcde', 4)).toBe(false);
	});
});
