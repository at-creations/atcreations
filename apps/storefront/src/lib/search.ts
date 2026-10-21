export const SORTS = [
	'relevance',
	'newest',
	'price_asc',
	'price_desc',
	'name_asc',
	'name_desc'
] as const;
export type Sort = (typeof SORTS)[number];

export const PRICE_RANGES = {
	all: {},
	'0-50': { maxPrice: 5000 },
	'50-100': { minPrice: 5000, maxPrice: 10000 },
	'100-250': { minPrice: 10000, maxPrice: 25000 },
	'250-up': { minPrice: 25000 }
} as const satisfies Record<string, { minPrice?: number; maxPrice?: number }>;
export type PriceRange = keyof typeof PRICE_RANGES;
export const PRICE_RANGE_KEYS = Object.keys(PRICE_RANGES) as PriceRange[];

export const MAX_PAGE = 10_000;
export const MAX_QUERY_LENGTH = 100;

/** Parse a `page` query value: integer ≥ 1 (clamped), defaulting to 1. */
export function parsePage(value: string | null | undefined): number {
	if (!value || !/^\d{1,6}$/.test(value)) return 1;
	return Math.min(Math.max(1, Number(value)), MAX_PAGE);
}

export function parseSort(value: string | null | undefined): Sort {
	return (SORTS as readonly string[]).includes(value ?? '') ? (value as Sort) : 'relevance';
}

export function parsePriceRange(value: string | null | undefined): PriceRange {
	return value && Object.hasOwn(PRICE_RANGES, value) ? (value as PriceRange) : 'all';
}

/** Trim, collapse whitespace and cap the search term at the API's max length. */
export function parseQuery(value: string | null | undefined): string {
	return (value ?? '').replace(/\s+/g, ' ').trim().slice(0, MAX_QUERY_LENGTH);
}

export function priceBounds(range: PriceRange): { minPrice?: number; maxPrice?: number } {
	return PRICE_RANGES[range];
}

export interface SearchState {
	q: string;
	category: string;
	sort: Sort;
	price: PriceRange;
	page: number;
}

/** Serialize search state to URL params, omitting defaults. */
export function searchStateToParams(state: SearchState): URLSearchParams {
	const params = new URLSearchParams();
	if (state.q) params.set('q', state.q);
	if (state.category) params.set('category', state.category);
	if (state.sort !== 'relevance') params.set('sort', state.sort);
	if (state.price !== 'all') params.set('price', state.price);
	if (state.page > 1) params.set('page', String(state.page));
	return params;
}
