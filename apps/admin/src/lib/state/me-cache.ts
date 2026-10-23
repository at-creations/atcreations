import type { Me } from '@atcreations/api-types';

/**
 * Last known `/me` per Clerk user, kept in sessionStorage (this tab only, gone when it closes) so
 * a reload paints the dashboard shell immediately while the server confirms the role in the
 * background. It only gates the UI; the API checks every permission on every request.
 */
const KEY = 'atc-admin:me';

export function readCachedMe(userId: string | null | undefined): Me | undefined {
	if (!userId) return undefined;
	try {
		const raw = sessionStorage.getItem(KEY);
		if (!raw) return undefined;
		const parsed = JSON.parse(raw) as { userId?: unknown; me?: Me };
		return parsed.userId === userId && parsed.me ? parsed.me : undefined;
	} catch {
		return undefined;
	}
}

export function writeCachedMe(userId: string, me: Me) {
	try {
		sessionStorage.setItem(KEY, JSON.stringify({ userId, me }));
	} catch {
		// Storage unavailable (private mode, quota): the cache is only an optimisation.
	}
}

export function clearCachedMe() {
	try {
		sessionStorage.removeItem(KEY);
	} catch {
		// ignore
	}
}
