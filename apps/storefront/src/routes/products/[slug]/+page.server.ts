import { error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { CATALOGUE_CACHE_CONTROL, getProduct } from '$lib/server/api';
import { isSlug } from '$lib/urls';

export const load: PageServerLoad = async ({ params, setHeaders }) => {
	if (!isSlug(params.slug)) error(404, 'Not found');
	const product = await getProduct(params.slug);
	if (product.status !== 'published') error(404, 'Not found');
	setHeaders({ 'cache-control': CATALOGUE_CACHE_CONTROL });
	return { product };
};
