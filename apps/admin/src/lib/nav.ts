import type { Permission } from '@atcreations/api-types';
import type { Pathname } from '$app/types';
import type { IconName } from '$lib/components/Icon.svelte';
import { m } from '$lib/paraglide/messages.js';

export interface NavItem {
	href: Pathname;
	label: () => string;
	icon: IconName;
	permission?: Permission;
	/** Shows the pending-orders counter next to the item. */
	badge?: 'pendingOrders';
}

/** Sidebar sections (separated by a divider) and the command menu's page list. */
export const NAV: NavItem[][] = [
	[
		{ href: '/', label: m.nav_dashboard, icon: 'dashboard' },
		{
			href: '/orders',
			label: m.nav_orders,
			icon: 'inbox',
			permission: 'orders:read',
			badge: 'pendingOrders'
		},
		{ href: '/products', label: m.nav_products, icon: 'box' },
		{ href: '/categories', label: m.nav_categories, icon: 'tag' }
	],
	[
		{ href: '/store', label: m.nav_store, icon: 'store', permission: 'store:write' },
		{ href: '/staff', label: m.nav_staff, icon: 'users', permission: 'staff:manage' },
		{ href: '/audit', label: m.nav_audit, icon: 'history', permission: 'audit:read' }
	]
];

export function isActivePath(href: string, pathname: string): boolean {
	return href === '/' ? pathname === '/' : pathname === href || pathname.startsWith(`${href}/`);
}
