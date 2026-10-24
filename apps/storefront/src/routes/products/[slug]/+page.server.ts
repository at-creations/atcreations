import { error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { CATALOGUE_CACHE_CONTROL, getProduct } from '$lib/server/api';
import { isSlug } from '$lib/urls';

export const load: PageServerLoad = async ({ params, setHeaders }) => {
	if (!isSlug(params.slug)) error(404, 'Not found');
	// The public endpoint only returns published products (404 otherwise).
	const product = await getProduct(params.slug);
	setHeaders({ 'cache-control': CATALOGUE_CACHE_CONTROL });
	return { product };
};
