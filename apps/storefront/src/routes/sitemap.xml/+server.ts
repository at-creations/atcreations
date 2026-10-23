import type { RequestHandler } from './$types';
import type { Product } from '@atcreations/api-types';
import { listCategories, listProducts } from '$lib/server/api';
import { config } from '$lib/server/env';
import { LOCALES, localizedPath } from '$lib/i18n';

const STATIC_PATHS = [
	'/',
	'/products',
	'/search',
	'/workshops',
	'/workshops/plaster-painting',
	'/about',
	'/contact',
	'/privacy',
	'/terms'
];
const PAGE_SIZE = 100;
const MAX_PAGES = 100; // the API pages lists 10k rows deep: 10k products (20k URLs, under the 50k limit)

function escapeXml(value: string): string {
	return value
		.replace(/&/g, '&amp;')
		.replace(/</g, '&lt;')
		.replace(/>/g, '&gt;')
		.replace(/"/g, '&quot;')
		.replace(/'/g, '&apos;');
}

async function allProducts(): Promise<Product[]> {
	const products: Product[] = [];
	for (let page = 1; page <= MAX_PAGES; page++) {
		const { data, meta } = await listProducts({ page, pageSize: PAGE_SIZE, sort: 'newest' });
		products.push(...data);
		if (page >= meta.totalPages || data.length === 0) break;
	}
	return products;
}

/** One <url> per locale, each listing every language version (incl. x-default). */
function entry(siteUrl: string, path: string, lastmod?: string): string {
	const alternates = [
		...LOCALES.map(
			(l) =>
				`<xhtml:link rel="alternate" hreflang="${l}" href="${escapeXml(siteUrl + localizedPath(l, path))}"/>`
		),
		`<xhtml:link rel="alternate" hreflang="x-default" href="${escapeXml(siteUrl + localizedPath('en', path))}"/>`
	].join('');
	return LOCALES.map(
		(l) =>
			`<url><loc>${escapeXml(siteUrl + localizedPath(l, path))}</loc>${
				lastmod ? `<lastmod>${escapeXml(lastmod)}</lastmod>` : ''
			}${alternates}</url>`
	).join('\n');
}

export const GET: RequestHandler = async () => {
	const { siteUrl } = config();
	const [categories, products] = await Promise.all([listCategories(), allProducts()]);

	const urls = [
		...STATIC_PATHS.map((p) => entry(siteUrl, p)),
		...categories.map((c) =>
			entry(siteUrl, `/products?category=${encodeURIComponent(c.slug)}`, c.updatedAt)
		),
		...products.map((p) => entry(siteUrl, `/products/${encodeURIComponent(p.slug)}`, p.updatedAt))
	];

	const xml = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:xhtml="http://www.w3.org/1999/xhtml">
${urls.join('\n')}
</urlset>`;

	return new Response(xml, {
		headers: {
			'content-type': 'application/xml; charset=utf-8',
			'cache-control': 'public, max-age=3600, s-maxage=3600'
		}
	});
};
