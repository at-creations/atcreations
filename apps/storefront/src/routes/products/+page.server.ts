import { error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { CATALOGUE_CACHE_CONTROL, listCategories, listProducts } from '$lib/server/api';
import { currentLocale } from '$lib/locale';
import { pageWithinListingLimit, parsePage } from '$lib/search';
import { isSlug } from '$lib/urls';

const PAGE_SIZE = 24;

export const load: PageServerLoad = async ({ url, setHeaders }) => {
	const locale = currentLocale();
	const rawCategory = url.searchParams.get('category');
	if (rawCategory !== null && rawCategory !== '' && !isSlug(rawCategory)) error(404, 'Not found');
	const category = rawCategory || null;
	const page = parsePage(url.searchParams.get('page'));
	if (!pageWithinListingLimit(page, PAGE_SIZE)) error(404, 'Not found');

	const [categories, products] = await Promise.all([
		listCategories(),
		listProducts({
			page,
			pageSize: PAGE_SIZE,
			category: category ?? undefined,
			sort: 'newest',
			locale
		})
	]);

	const selectedCategory = category ? categories.find((c) => c.slug === category) : undefined;
	if (category && !selectedCategory) error(404, 'Not found');
	// A page past the end is not a real listing (and would be an indexable empty page).
	if (page > 1 && page > products.meta.totalPages) error(404, 'Not found');

	setHeaders({ 'cache-control': CATALOGUE_CACHE_CONTROL });
	return {
		categories,
		selectedCategory: selectedCategory ?? null,
		products: products.data,
		meta: products.meta,
		page
	};
};
