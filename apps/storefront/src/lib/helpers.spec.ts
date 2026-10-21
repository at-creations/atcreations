import { describe, expect, it } from 'vitest';
import { localizedPath, stripLocale, switchLocaleHref, t } from './i18n';
import { formatMoney } from './format';
import { isDiscounted, isNewProduct, savingsCents } from './product';
import { assetUrl, isSlug, productHref, productsHref } from './urls';
import { parsePage, parsePriceRange, parseQuery, parseSort, searchStateToParams } from './search';
import { serializeJsonLd, truncate } from './seo';
import { directionsUrl, mapEmbedUrl } from './maps';

describe('t()', () => {
	it('picks the requested locale and falls back to English', () => {
		expect(t({ en: 'Vase', vi: 'Bình' }, 'vi')).toBe('Bình');
		expect(t({ en: 'Vase', vi: 'Bình' }, 'en')).toBe('Vase');
		expect(t({ en: 'Vase' }, 'vi')).toBe('Vase');
		expect(t({ en: 'Vase', vi: '  ' }, 'vi')).toBe('Vase');
		expect(t(undefined, 'vi')).toBe('');
	});
});

describe('locale paths', () => {
	it('prefixes and strips locales', () => {
		expect(localizedPath('en')).toBe('/en');
		expect(localizedPath('vi', '/products')).toBe('/vi/products');
		expect(localizedPath('vi', 'products?page=2')).toBe('/vi/products?page=2');
		expect(localizedPath('en', '/?x=1')).toBe('/en?x=1');
		expect(stripLocale('/vi/products/abc')).toBe('/products/abc');
		expect(stripLocale('/en')).toBe('/');
		expect(stripLocale('/envelope')).toBe('/envelope');
	});

	it('switches locale preserving path and query', () => {
		const url = new URL('https://x.test/en/search?q=vase&page=2');
		expect(switchLocaleHref(url, 'vi')).toBe('/vi/search?q=vase&page=2');
	});
});

describe('formatMoney', () => {
	it('formats CAD cents per locale', () => {
		expect(formatMoney(12345, 'en')).toBe('$123.45');
		const vi = formatMoney(12345, 'vi');
		expect(vi).toContain('123,45');
		expect(vi).toMatch(/CA\$|CAD/);
	});
});

describe('product helpers', () => {
	const now = Date.parse('2026-10-21T00:00:00Z');

	it('detects new products (< 30 days)', () => {
		expect(isNewProduct({ createdAt: '2026-10-01T00:00:00Z' }, now)).toBe(true);
		expect(isNewProduct({ createdAt: '2026-09-01T00:00:00Z' }, now)).toBe(false);
		expect(isNewProduct({ createdAt: 'garbage' }, now)).toBe(false);
	});

	it('computes discounts', () => {
		const p = { priceCents: 10000, finalPriceCents: 8000, discountPercent: 20 };
		expect(isDiscounted(p)).toBe(true);
		expect(savingsCents(p)).toBe(2000);
		expect(isDiscounted({ priceCents: null, finalPriceCents: null, discountPercent: 20 })).toBe(
			false
		);
		expect(isDiscounted({ priceCents: 100, finalPriceCents: 100, discountPercent: 0 })).toBe(false);
	});
});

describe('url builders', () => {
	it('validates slugs', () => {
		expect(isSlug('bamboo-vase-2')).toBe(true);
		expect(isSlug('Bad Slug')).toBe(false);
		expect(isSlug('../etc')).toBe(false);
		expect(isSlug(null)).toBe(false);
	});

	it('builds product URLs', () => {
		expect(productHref('vi', 'bamboo-vase')).toBe('/vi/products/bamboo-vase');
		expect(productsHref('en')).toBe('/en/products');
		expect(productsHref('en', { category: 'pottery', page: 1 })).toBe(
			'/en/products?category=pottery'
		);
		expect(productsHref('en', { category: 'pottery', page: 3 })).toBe(
			'/en/products?category=pottery&page=3'
		);
		expect(assetUrl('https://cdn.test/site/', '/hero.jpg')).toBe('https://cdn.test/site/hero.jpg');
	});
});

describe('search params', () => {
	it('clamps page', () => {
		expect(parsePage(null)).toBe(1);
		expect(parsePage('0')).toBe(1);
		expect(parsePage('-2')).toBe(1);
		expect(parsePage('abc')).toBe(1);
		expect(parsePage('7')).toBe(7);
		expect(parsePage('999999')).toBe(10000);
	});

	it('accepts only known sort and price values', () => {
		expect(parseSort('price_asc')).toBe('price_asc');
		expect(parseSort('popular')).toBe('relevance');
		expect(parseSort('$where')).toBe('relevance');
		expect(parsePriceRange('50-100')).toBe('50-100');
		expect(parsePriceRange('toString')).toBe('all');
	});

	it('normalizes the query', () => {
		expect(parseQuery('  bamboo   vase ')).toBe('bamboo vase');
		expect(parseQuery('x'.repeat(200))).toHaveLength(100);
	});

	it('serializes state without defaults', () => {
		const base = { q: '', category: '', sort: 'relevance', price: 'all', page: 1 } as const;
		expect(searchStateToParams(base).toString()).toBe('');
		expect(
			searchStateToParams({
				q: 'vase',
				category: 'pottery',
				sort: 'newest',
				price: '0-50',
				page: 2
			}).toString()
		).toBe('q=vase&category=pottery&sort=newest&price=0-50&page=2');
	});
});

describe('seo helpers', () => {
	it('escapes JSON-LD so it cannot break out of a script tag', () => {
		const payload = { name: '</script><script>alert(1)</script>&' };
		const out = serializeJsonLd(payload);
		expect(out).not.toContain('<');
		expect(out).not.toContain('>');
		expect(out).not.toContain('&');
		expect(JSON.parse(out)).toEqual(payload);
	});

	it('truncates on word boundaries', () => {
		expect(truncate('short')).toBe('short');
		const out = truncate('word '.repeat(60), 50);
		expect(out.length).toBeLessThanOrEqual(50);
		expect(out.endsWith('…')).toBe(true);
	});
});

describe('maps', () => {
	it('uses a Google mapUrl when provided, otherwise the address', () => {
		expect(mapEmbedUrl('https://www.google.com/maps/embed?pb=abc', 'x', 'en')).toBe(
			'https://www.google.com/maps/embed?pb=abc'
		);
		expect(mapEmbedUrl('https://evil.test/embed', '4305 Main St', 'vi')).toBe(
			'https://maps.google.com/maps?q=4305+Main+St&hl=vi&output=embed'
		);
		expect(mapEmbedUrl(undefined, '  ', 'en')).toBeNull();
		expect(directionsUrl('4305 Main St')).toBe(
			'https://www.google.com/maps/dir/?api=1&destination=4305+Main+St'
		);
	});
});
