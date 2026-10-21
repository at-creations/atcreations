import type { PageServerLoad } from './$types';
import { CATALOGUE_CACHE_CONTROL, listCategories, listProducts, orFallback } from '$lib/server/api';
import { currentLocale } from '$lib/locale';

export const load: PageServerLoad = async ({ setHeaders }) => {
	const locale = currentLocale();
	// The home page degrades gracefully: a failing section is hidden rather than erroring.
	const [featured, categories] = await Promise.all([
		orFallback(
			async () =>
				(await listProducts({ featured: true, pageSize: 6, sort: 'newest', locale })).data,
			[]
		),
		orFallback(listCategories, [])
	]);
	setHeaders({ 'cache-control': CATALOGUE_CACHE_CONTROL });
	return { featured, categories };
};
