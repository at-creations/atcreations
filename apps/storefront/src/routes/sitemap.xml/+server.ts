import type { RequestHandler } from './$types';
import type { PublicProduct } from '@atcreations/api-types';
import { listCategories, listProducts } from '$lib/server/api';
import { config } from '$lib/server/env';
import { LOCALES, localizedPath } from '$lib/i18n';

// /search is left out: its pages are noindex.
const STATIC_PATHS = [
	'/',
	'/products',
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

async function allProducts(): Promise<PublicProduct[]> {
	const products: PublicProduct[] = [];
	for (let page = 1; page <= MAX_PAGES; page++) {
		const { data, meta } = await listProducts({ page, pageSize: PAGE_SIZE, sort: 'newest' });
		products.push(...data);
		if (page >= meta.totalPages || data.length === 0) break;
	}
	return products;
}

/** One <url> per locale, each listing every language version (incl. x-default) and its images. */
function entry(siteUrl: string, path: string, lastmod?: string, images: string[] = []): string {
	const alternates = [
		...LOCALES.map(
			(l) =>
				`<xhtml:link rel="alternate" hreflang="${l}" href="${escapeXml(siteUrl + localizedPath(l, path))}"/>`
		),
		`<xhtml:link rel="alternate" hreflang="x-default" href="${escapeXml(siteUrl + localizedPath('en', path))}"/>`
	].join('');
	const imageTags = images
		.map((url) => `<image:image><image:loc>${escapeXml(url)}</image:loc></image:image>`)
		.join('');
	return LOCALES.map(
		(l) =>
			`<url><loc>${escapeXml(siteUrl + localizedPath(l, path))}</loc>${
				lastmod ? `<lastmod>${escapeXml(lastmod)}</lastmod>` : ''
			}${alternates}${imageTags}</url>`
	).join('\n');
}

export const GET: RequestHandler = async () => {
	const { siteUrl } = config();
	const [categories, products] = await Promise.all([listCategories(), allProducts()]);

	const urls = [
		...STATIC_PATHS.map((p) => entry(siteUrl, p)),
		...categories.map((c) =>
			entry(
				siteUrl,
				`/products?category=${encodeURIComponent(c.slug)}`,
				c.updatedAt,
				c.image ? [c.image.url] : []
			)
		),
		...products.map((p) =>
			entry(
				siteUrl,
				`/products/${encodeURIComponent(p.slug)}`,
				p.updatedAt,
				p.images.map((i) => i.url)
			)
		)
	];

	const xml = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:xhtml="http://www.w3.org/1999/xhtml" xmlns:image="http://www.google.com/schemas/sitemap-image/1.1">
${urls.join('\n')}
</urlset>`;

	return new Response(xml, {
		headers: {
			'content-type': 'application/xml; charset=utf-8',
			'cache-control': 'public, max-age=3600, s-maxage=3600'
		}
	});
};
