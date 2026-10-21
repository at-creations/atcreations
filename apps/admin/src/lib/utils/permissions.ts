import type { Me, Permission, Role, StaffMember } from '@atcreations/api-types';

/** A staff account's role; `none` = Clerk account without a staff role (no access). */
export type StaffRole = StaffMember['role'];

export const ROLES: readonly Role[] = ['staff', 'manager', 'admin'];

/**
 * Permission ladder (mirrors the API; the API remains the source of truth via GET /v1/admin/me).
 * staff   = products & orders
 * manager = + categories, product delete, shipping edits
 * admin   = + staff, store info, audit
 */
const STAFF: Permission[] = ['products:write', 'orders:read', 'orders:write'];
const MANAGER: Permission[] = [...STAFF, 'categories:write', 'products:delete', 'orders:shipping'];
const ADMIN: Permission[] = [...MANAGER, 'staff:manage', 'store:write', 'audit:read'];

export const ROLE_PERMISSIONS: Record<StaffRole, readonly Permission[]> = {
	none: [],
	staff: STAFF,
	manager: MANAGER,
	admin: ADMIN
};

export function permissionsForRole(role: StaffRole): readonly Permission[] {
	return ROLE_PERMISSIONS[role] ?? [];
}

/** Permissions gained (positive) or lost (negative) when moving from one role to another. */
export function permissionDiff(
	from: StaffRole,
	to: StaffRole
): { added: Permission[]; removed: Permission[] } {
	const a = new Set(permissionsForRole(from));
	const b = new Set(permissionsForRole(to));
	return {
		added: [...b].filter((p) => !a.has(p)),
		removed: [...a].filter((p) => !b.has(p))
	};
}

/** True when `me` holds every permission listed. A missing `me` never has permissions. */
export function hasPermission(
	me: Pick<Me, 'permissions'> | null | undefined,
	...required: Permission[]
): boolean {
	if (!me) return false;
	return required.every((p) => me.permissions.includes(p));
}

export function isRole(value: unknown): value is Role {
	return typeof value === 'string' && (ROLES as readonly string[]).includes(value);
}
