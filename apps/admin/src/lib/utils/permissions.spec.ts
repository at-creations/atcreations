import { describe, expect, it } from 'vitest';
import { hasPermission, isRole, permissionDiff, permissionsForRole } from './permissions';

describe('hasPermission', () => {
	const me = { permissions: ['products:write', 'orders:read'] as const };
	it('checks single and multiple permissions', () => {
		expect(hasPermission({ permissions: [...me.permissions] }, 'products:write')).toBe(true);
		expect(hasPermission({ permissions: [...me.permissions] }, 'products:delete')).toBe(false);
		expect(
			hasPermission({ permissions: [...me.permissions] }, 'products:write', 'orders:read')
		).toBe(true);
		expect(
			hasPermission({ permissions: [...me.permissions] }, 'products:write', 'staff:manage')
		).toBe(false);
	});
	it('denies when there is no user', () => {
		expect(hasPermission(null, 'orders:read')).toBe(false);
		expect(hasPermission(undefined, 'orders:read')).toBe(false);
	});
	it('allows when no permission is required', () => {
		expect(hasPermission({ permissions: [] })).toBe(true);
	});
});

describe('role ladder', () => {
	it('is cumulative', () => {
		const staff = permissionsForRole('staff');
		const manager = permissionsForRole('manager');
		const admin = permissionsForRole('admin');
		expect(staff.every((p) => manager.includes(p))).toBe(true);
		expect(manager.every((p) => admin.includes(p))).toBe(true);
		expect(staff).not.toContain('categories:write');
		expect(manager).toContain('orders:shipping');
		expect(manager).not.toContain('staff:manage');
		expect(admin).toContain('audit:read');
	});
	it('computes diffs between roles', () => {
		expect(permissionDiff('staff', 'manager')).toEqual({
			added: ['categories:write', 'products:delete', 'orders:shipping'],
			removed: []
		});
		expect(permissionDiff('admin', 'staff').removed).toContain('staff:manage');
		expect(permissionDiff('admin', 'admin')).toEqual({ added: [], removed: [] });
		expect(permissionDiff('none', 'staff').added).toEqual([
			'products:write',
			'orders:read',
			'orders:write'
		]);
		expect(permissionsForRole('none')).toEqual([]);
	});
	it('validates roles', () => {
		expect(isRole('manager')).toBe(true);
		expect(isRole('owner')).toBe(false);
		expect(isRole(undefined)).toBe(false);
	});
});
