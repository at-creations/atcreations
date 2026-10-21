import { describe, expect, it } from 'vitest';
import { safeRedirectPath } from './redirect';

describe('safeRedirectPath', () => {
	it('accepts same-origin relative paths', () => {
		expect(safeRedirectPath('/products')).toBe('/products');
		expect(safeRedirectPath('/products?page=2&status=draft')).toBe('/products?page=2&status=draft');
		expect(safeRedirectPath('/orders/abc#history')).toBe('/orders/abc#history');
	});
	it('falls back for empty or missing values', () => {
		expect(safeRedirectPath(null)).toBe('/');
		expect(safeRedirectPath(undefined)).toBe('/');
		expect(safeRedirectPath('')).toBe('/');
		expect(safeRedirectPath('', '/home')).toBe('/home');
	});
	it('rejects protocol-relative and absolute URLs', () => {
		expect(safeRedirectPath('//evil.com')).toBe('/');
		expect(safeRedirectPath('///evil.com')).toBe('/');
		expect(safeRedirectPath('https://evil.com')).toBe('/');
		expect(safeRedirectPath('javascript:alert(1)')).toBe('/');
		expect(safeRedirectPath('data:text/html,hi')).toBe('/');
		expect(safeRedirectPath('evil.com/path')).toBe('/');
	});
	it('rejects backslashes and encoded tricks', () => {
		expect(safeRedirectPath('/\\evil.com')).toBe('/');
		expect(safeRedirectPath('\\\\evil.com')).toBe('/');
		expect(safeRedirectPath('/%2F%2Fevil.com')).toBe('/');
		expect(safeRedirectPath('/%5Cevil.com')).toBe('/');
		expect(safeRedirectPath('/%E0%A4%A')).toBe('/');
	});
	it('rejects control characters', () => {
		expect(safeRedirectPath('/\tevil')).toBe('/');
		expect(safeRedirectPath('/\nfoo')).toBe('/');
	});
	it('avoids redirect loops to the sign-in page', () => {
		expect(safeRedirectPath('/sign-in')).toBe('/');
		expect(safeRedirectPath('/sign-in/factor-one')).toBe('/');
		expect(safeRedirectPath('/sign-in-help')).toBe('/sign-in-help');
	});
});
