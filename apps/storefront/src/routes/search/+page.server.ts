import type { PageServerLoad } from './$types';
import { CATALOGUE_CACHE_CONTROL, listCategories, listProducts } from '$lib/server/api';
import { currentLocale } from '$lib/locale';
import {
	parsePage,
	parsePriceRange,
	parseQuery,
	parseSort,
	priceBounds,
	type SearchState
} from '$lib/search';
import { isSlug } from '$lib/urls';

const PAGE_SIZE = 24;

export const load: PageServerLoad = async ({ url, setHeaders }) => {
	const locale = currentLocale();
	const params = url.searchParams;
	const rawCategory = params.get('category');

	// Validate and clamp everything before it reaches the API.
	const state: SearchState = {
		q: parseQuery(params.get('q')),
		category: isSlug(rawCategory) ? rawCategory : '',
		sort: parseSort(params.get('sort')),
		price: parsePriceRange(params.get('price')),
		page: parsePage(params.get('page'))
	};

	const [categories, results] = await Promise.all([
		listCategories(),
		listProducts({
			q: state.q || undefined,
			category: state.category || undefined,
			sort: state.sort,
			...priceBounds(state.price),
			locale,
			page: state.page,
			pageSize: PAGE_SIZE
		})
	]);

	setHeaders({ 'cache-control': CATALOGUE_CACHE_CONTROL });
	return { state, categories, products: results.data, meta: results.meta };
};
