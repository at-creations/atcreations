import { describe, expect, it } from 'vitest';
import type { PublicProduct, StoreInfo } from '@atcreations/api-types';
import { breadcrumbJsonLd, productJsonLd, storeJsonLd, websiteJsonLd } from './structured-data';

const SITE = 'https://atcreations.ca';

const product: PublicProduct = {
	id: '665f00000000000000000001',
	slug: 'wooden-boat',
	name: { en: 'Wooden boat', vi: 'Thuyền gỗ' },
	description: { en: 'Carved by hand.', vi: '' },
	priceCents: 5000,
	finalPriceCents: 4500,
	currency: 'CAD',
	discountPercent: 10,
	inStock: true,
	featured: false,
	categoryIds: [],
	categories: [{ id: 'c1', slug: 'models', name: { en: 'Models', vi: 'Mô hình' } }],
	images: [{ id: 'i1', key: 'k', url: 'https://cdn.example/boat.webp', width: 800, height: 600 }],
	createdAt: '2026-01-01T00:00:00Z',
	updatedAt: '2026-01-02T00:00:00Z'
};

describe('productJsonLd', () => {
	it('describes the product and its offer with absolute, localized URLs', () => {
		const ld = productJsonLd({ siteUrl: SITE, locale: 'vi', product, brand: 'AT Creations' });
		expect(ld.name).toBe('Thuyền gỗ');
		expect(ld.url).toBe(`${SITE}/vi/products/wooden-boat`);
		expect(ld.description).toBe('Carved by hand.'); // falls back to English
		expect(ld.category).toBe('Mô hình');
		expect(ld.image).toEqual(['https://cdn.example/boat.webp']);
		expect(ld.offers).toMatchObject({
			'@type': 'Offer',
			price: '45.00',
			priceCurrency: 'CAD',
			availability: 'https://schema.org/InStock'
		});
	});

	it('has no offer for "contact for price" products', () => {
		const ld = productJsonLd({
			siteUrl: SITE,
			locale: 'en',
			product: { ...product, priceCents: null, finalPriceCents: null },
			brand: 'AT Creations'
		});
		expect(ld.offers).toBeUndefined();
	});
});

describe('breadcrumbJsonLd', () => {
	it('numbers the items and uses the current URL for the last one', () => {
		const ld = breadcrumbJsonLd(
			SITE,
			[{ label: 'Home', href: '/en' }, { label: 'Boat' }],
			`${SITE}/en/products/wooden-boat`
		);
		expect(ld.itemListElement).toEqual([
			{ '@type': 'ListItem', position: 1, name: 'Home', item: `${SITE}/en` },
			{ '@type': 'ListItem', position: 2, name: 'Boat', item: `${SITE}/en/products/wooden-boat` }
		]);
	});
});

describe('storeJsonLd / websiteJsonLd', () => {
	it('lists contact details and only the open days', () => {
		const storeInfo: StoreInfo = {
			email: 'hi@example.com',
			phone: ' ',
			address: '123 Main St, Vancouver',
			businessHours: [
				{ day: 'mon', closed: false, open: '10:00', close: '18:00' },
				{ day: 'sun', closed: true }
			],
			updatedAt: '2026-01-01T00:00:00Z'
		};
		const ld = storeJsonLd({
			siteUrl: SITE,
			assetBaseUrl: 'https://cdn.example/site/',
			locale: 'en',
			name: 'AT Creations',
			description: 'Shop',
			storeInfo
		});
		expect(ld.email).toBe('hi@example.com');
		expect(ld.telephone).toBeUndefined();
		expect(ld.logo).toBe('https://cdn.example/site/logo-square.png');
		expect(ld.openingHoursSpecification).toEqual([
			{
				'@type': 'OpeningHoursSpecification',
				dayOfWeek: 'https://schema.org/Monday',
				opens: '10:00',
				closes: '18:00'
			}
		]);
	});

	it('points the search action at the localized search page', () => {
		const ld = websiteJsonLd({ siteUrl: SITE, locale: 'vi', name: 'AT Creations' });
		expect(ld.potentialAction).toMatchObject({
			target: { urlTemplate: `${SITE}/vi/search?q={search_term_string}` }
		});
	});
});
