import type { BusinessHours, PublicProduct, StoreInfo } from '@atcreations/api-types';
import { localizedPath, t, type Locale } from './i18n';
import { truncate } from './seo';
import { assetUrl } from './urls';

/** schema.org JSON-LD builders. Every URL they return is absolute. */

type JsonLd = Record<string, unknown>;

const SCHEMA_DAYS: Record<BusinessHours['day'], string> = {
	mon: 'https://schema.org/Monday',
	tue: 'https://schema.org/Tuesday',
	wed: 'https://schema.org/Wednesday',
	thu: 'https://schema.org/Thursday',
	fri: 'https://schema.org/Friday',
	sat: 'https://schema.org/Saturday',
	sun: 'https://schema.org/Sunday'
};

/** Stable `@id`s so the pages' nodes link to one shop and one site. */
export const storeId = (siteUrl: string) => `${siteUrl}/#store`;
export const websiteId = (siteUrl: string) => `${siteUrl}/#website`;

/** The shop: a `Store` (an Organization and a LocalBusiness) with its contact details and hours. */
export function storeJsonLd(opts: {
	siteUrl: string;
	assetBaseUrl: string;
	locale: Locale;
	name: string;
	description: string;
	storeInfo: StoreInfo | null;
}): JsonLd {
	const { siteUrl, assetBaseUrl, locale, storeInfo } = opts;
	const ld: JsonLd = {
		'@type': 'Store',
		'@id': storeId(siteUrl),
		name: opts.name,
		description: opts.description,
		url: `${siteUrl}${localizedPath(locale)}`,
		logo: assetUrl(assetBaseUrl, 'logo-square.png'),
		image: assetUrl(assetBaseUrl, 'og-image.png'),
		currenciesAccepted: 'CAD'
	};
	const email = storeInfo?.email?.trim();
	const phone = storeInfo?.phone?.trim();
	const address = storeInfo?.address?.trim();
	if (email) ld.email = email;
	if (phone) ld.telephone = phone;
	if (address) ld.address = address;
	if (storeInfo?.mapUrl) ld.hasMap = storeInfo.mapUrl;
	const hours = (storeInfo?.businessHours ?? []).filter((h) => !h.closed && h.open && h.close);
	if (hours.length > 0) {
		ld.openingHoursSpecification = hours.map((h) => ({
			'@type': 'OpeningHoursSpecification',
			dayOfWeek: SCHEMA_DAYS[h.day],
			opens: h.open,
			closes: h.close
		}));
	}
	return ld;
}

/** The site, with a sitelinks search box pointing at the localized search page. */
export function websiteJsonLd(opts: { siteUrl: string; locale: Locale; name: string }): JsonLd {
	const { siteUrl, locale } = opts;
	return {
		'@type': 'WebSite',
		'@id': websiteId(siteUrl),
		name: opts.name,
		url: `${siteUrl}${localizedPath(locale)}`,
		inLanguage: locale,
		publisher: { '@id': storeId(siteUrl) },
		potentialAction: {
			'@type': 'SearchAction',
			target: {
				'@type': 'EntryPoint',
				urlTemplate: `${siteUrl}${localizedPath(locale, '/search')}?q={search_term_string}`
			},
			'query-input': 'required name=search_term_string'
		}
	};
}

export interface Crumb {
	label: string;
	/** App-relative (localized) path; the last crumb defaults to `currentUrl`. */
	href?: string;
}

export function breadcrumbJsonLd(siteUrl: string, items: Crumb[], currentUrl: string): JsonLd {
	return {
		'@context': 'https://schema.org',
		'@type': 'BreadcrumbList',
		itemListElement: items.map((item, i) => ({
			'@type': 'ListItem',
			position: i + 1,
			name: item.label,
			item: item.href ? `${siteUrl}${item.href}` : currentUrl
		}))
	};
}

/** A product with its offer. Products without a price ("contact for price") get no offer. */
export function productJsonLd(opts: {
	siteUrl: string;
	locale: Locale;
	product: PublicProduct;
	brand: string;
}): JsonLd {
	const { siteUrl, locale, product } = opts;
	const name = t(product.name, locale);
	const description = t(product.description, locale);
	const url = `${siteUrl}${localizedPath(locale, `/products/${product.slug}`)}`;
	const ld: JsonLd = {
		'@context': 'https://schema.org',
		'@type': 'Product',
		'@id': `${url}#product`,
		name,
		sku: product.id,
		url,
		image: product.images.map((i) => i.url),
		brand: { '@type': 'Brand', name: opts.brand }
	};
	if (description) ld.description = truncate(description, 5000);
	if (product.categories[0]) ld.category = t(product.categories[0].name, locale);
	if (product.finalPriceCents !== null) {
		ld.offers = {
			'@type': 'Offer',
			url,
			priceCurrency: product.currency,
			price: (product.finalPriceCents / 100).toFixed(2),
			availability: product.inStock
				? 'https://schema.org/InStock'
				: 'https://schema.org/OutOfStock',
			itemCondition: 'https://schema.org/NewCondition',
			seller: { '@id': storeId(siteUrl) }
		};
	}
	return ld;
}
