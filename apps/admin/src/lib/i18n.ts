import type { LocalizedText, OrderStatus, Permission, ProductStatus } from '@atcreations/api-types';
import { m } from '$lib/paraglide/messages.js';
import { getLocale } from '$lib/paraglide/runtime.js';
import { ApiError } from '$lib/api-error';
import type { StaffRole } from '$lib/utils/permissions';
import type { ParsedDollars } from '$lib/utils/money';

export function localized(text: LocalizedText | null | undefined): string {
	if (!text) return '';
	return (getLocale() === 'vi' ? text.vi : undefined) || text.en || '';
}

function intlLocale() {
	return getLocale() === 'vi' ? 'vi-VN' : 'en-CA';
}

/** The API stops counting at this many matches (list totals, product stats). */
export const MAX_COUNT = 10_000;

/** A list total; "10,000+" when the API stopped counting (meta.totalCapped). */
export function formatTotal(meta: { total: number; totalCapped?: boolean }): string {
	const n = new Intl.NumberFormat(intlLocale()).format(meta.total);
	return meta.totalCapped ? `${n}+` : n;
}

export function formatDate(iso: string | null | undefined): string {
	if (!iso) return '';
	const d = new Date(iso);
	if (Number.isNaN(d.getTime())) return '';
	return new Intl.DateTimeFormat(intlLocale(), { dateStyle: 'medium' }).format(d);
}

export function formatDateTime(iso: string | null | undefined): string {
	if (!iso) return '';
	const d = new Date(iso);
	if (Number.isNaN(d.getTime())) return '';
	return new Intl.DateTimeFormat(intlLocale(), { dateStyle: 'medium', timeStyle: 'short' }).format(
		d
	);
}

export function productStatusLabel(status: ProductStatus): string {
	return { draft: m.status_draft, published: m.status_published, archived: m.status_archived }[
		status
	]();
}

export function orderStatusLabel(status: OrderStatus): string {
	const map: Record<OrderStatus, () => string> = {
		pending: m.order_status_pending,
		confirmed: m.order_status_confirmed,
		processing: m.order_status_processing,
		shipped: m.order_status_shipped,
		delivered: m.order_status_delivered,
		completed: m.order_status_completed,
		cancelled: m.order_status_cancelled,
		refunded: m.order_status_refunded
	};
	return map[status]?.() ?? status;
}

export function roleLabel(role: StaffRole): string {
	const map: Record<StaffRole, () => string> = {
		staff: m.role_staff,
		manager: m.role_manager,
		admin: m.role_admin,
		none: m.role_none
	};
	return map[role]?.() ?? role;
}

export function roleSummary(role: StaffRole): string {
	const map: Record<StaffRole, () => string> = {
		staff: m.role_staff_summary,
		manager: m.role_manager_summary,
		admin: m.role_admin_summary,
		none: m.role_none_summary
	};
	return map[role]?.() ?? '';
}

export function permissionLabel(permission: Permission): string {
	const map: Record<Permission, () => string> = {
		'products:write': m.perm_products_write,
		'products:delete': m.perm_products_delete,
		'categories:write': m.perm_categories_write,
		'orders:read': m.perm_orders_read,
		'orders:write': m.perm_orders_write,
		'orders:shipping': m.perm_orders_shipping,
		'store:write': m.perm_store_write,
		'staff:manage': m.perm_staff_manage,
		'audit:read': m.perm_audit_read
	};
	return map[permission]?.() ?? permission;
}

export function priceErrorMessage(result: ParsedDollars): string | undefined {
	if (result.ok) return undefined;
	return {
		invalid: m.price_error_invalid,
		negative: m.price_error_negative,
		precision: m.price_error_precision,
		too_large: m.price_error_too_large
	}[result.reason]();
}

/**
 * Human-readable message for a failed request. Problem `detail` (e.g. 409 "cannot demote the
 * last admin") is surfaced verbatim as text — never as HTML.
 */
export function describeError(
	error: unknown,
	fallback?: string
): { message: string; detail?: string } {
	if (error instanceof ApiError) {
		if (error.status === 0) return { message: m.error_network() };
		const detail = [
			error.detail ?? (fallback ? error.title : undefined),
			error.requestId ? m.error_request_id({ id: error.requestId }) : undefined
		]
			.filter(Boolean)
			.join(' · ');
		return { message: fallback ?? error.detail ?? error.title, detail: detail || undefined };
	}
	return { message: fallback ?? m.error_generic() };
}
