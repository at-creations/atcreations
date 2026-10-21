import type { Product } from '@atcreations/api-types';

const NEW_WINDOW_MS = 30 * 24 * 60 * 60 * 1000;

type PriceFields = Pick<Product, 'priceCents' | 'finalPriceCents' | 'discountPercent'>;

/** "New" badge: created less than 30 days ago. */
export function isNewProduct(product: Pick<Product, 'createdAt'>, now = Date.now()): boolean {
	const created = Date.parse(product.createdAt);
	return Number.isFinite(created) && now >= created && now - created < NEW_WINDOW_MS;
}

export function isDiscounted(product: PriceFields): boolean {
	return (
		product.priceCents !== null &&
		product.finalPriceCents !== null &&
		product.discountPercent > 0 &&
		product.finalPriceCents < product.priceCents
	);
}

export function savingsCents(product: PriceFields): number {
	return isDiscounted(product) ? (product.priceCents ?? 0) - (product.finalPriceCents ?? 0) : 0;
}
