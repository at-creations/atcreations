import type { paths } from '@atcreations/api-types';

type QueryOf<P extends keyof paths> = paths[P] extends { get: { parameters: { query?: infer Q } } }
	? NonNullable<Q>
	: never;

export type ProductListParams = QueryOf<'/v1/admin/products'>;
export type OrderListParams = QueryOf<'/v1/admin/orders'>;
export type StaffListParams = QueryOf<'/v1/admin/staff'>;
export type AuditListParams = QueryOf<'/v1/admin/audit'>;
export type AnalyticsDays = NonNullable<QueryOf<'/v1/admin/analytics'>['days']>;

/**
 * Drop `undefined`, `null` and '' so equivalent filter objects produce the same key
 * (`{ q: '' }` and `{}` both mean "no search"). Keys are sorted for readability; TanStack
 * hashes object keys order-independently anyway.
 */
export function normalizeParams<T extends object>(params: T): Partial<T> {
	const out: Record<string, unknown> = {};
	for (const key of Object.keys(params).sort()) {
		const value = (params as Record<string, unknown>)[key];
		if (value === undefined || value === null || value === '') continue;
		out[key] = value;
	}
	return out as Partial<T>;
}

/**
 * Query-key factory. Hierarchical so invalidation can be as broad (`keys.products.all`) or as
 * narrow (`keys.products.detail(id)`) as needed.
 */
export const keys = {
	me: ['me'] as const,
	products: {
		all: ['products'] as const,
		lists: () => ['products', 'list'] as const,
		list: (params: ProductListParams = {}) =>
			['products', 'list', normalizeParams(params)] as const,
		details: () => ['products', 'detail'] as const,
		detail: (id: string) => ['products', 'detail', id] as const
	},
	categories: {
		all: ['categories'] as const
	},
	orders: {
		all: ['orders'] as const,
		lists: () => ['orders', 'list'] as const,
		list: (params: OrderListParams = {}) => ['orders', 'list', normalizeParams(params)] as const,
		details: () => ['orders', 'detail'] as const,
		detail: (id: string) => ['orders', 'detail', id] as const
	},
	staff: {
		all: ['staff'] as const,
		lists: () => ['staff', 'list'] as const,
		list: (params: StaffListParams = {}) => ['staff', 'list', normalizeParams(params)] as const,
		invitations: ['staff', 'invitations'] as const
	},
	storeInfo: ['store-info'] as const,
	uploads: {
		all: ['uploads'] as const,
		list: (kind: 'product' | 'category', id: string) => ['uploads', kind, id] as const
	},
	analytics: {
		all: ['analytics'] as const,
		report: (days: AnalyticsDays) => ['analytics', 'report', days] as const
	},
	audit: {
		all: ['audit'] as const,
		list: (params: AuditListParams = {}) => ['audit', 'list', normalizeParams(params)] as const
	}
};
