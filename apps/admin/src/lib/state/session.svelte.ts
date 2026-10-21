import { getContext, setContext } from 'svelte';
import type { Me, Permission } from '@atcreations/api-types';
import { hasPermission } from '$lib/utils/permissions';

const KEY = Symbol('admin-session');

/** The signed-in staff member as returned by GET /v1/admin/me. */
export class AdminSession {
	me = $state<Me | null>(null);

	/** UI gating only — the API enforces every permission as well. */
	can = (...permissions: Permission[]): boolean => hasPermission(this.me, ...permissions);
}

export function setAdminSession(session: AdminSession) {
	return setContext(KEY, session);
}

export function getAdminSession(): AdminSession {
	const session = getContext<AdminSession | undefined>(KEY);
	if (!session) throw new Error('AdminSession context is missing');
	return session;
}
