/** Helpers for treating the URL query string as the source of truth for list pages. */

export function readInt(
	params: URLSearchParams,
	key: string,
	fallback: number,
	allowed?: readonly number[]
): number {
	const raw = params.get(key);
	if (raw === null || !/^\d+$/.test(raw)) return fallback;
	const n = Number(raw);
	if (!Number.isSafeInteger(n) || n < 1) return fallback;
	if (allowed && !allowed.includes(n)) return fallback;
	return n;
}

export function readEnum<T extends string>(
	params: URLSearchParams,
	key: string,
	allowed: readonly T[],
	fallback: T | undefined = undefined
): T | undefined {
	const raw = params.get(key);
	return raw !== null && (allowed as readonly string[]).includes(raw) ? (raw as T) : fallback;
}

export function readString(params: URLSearchParams, key: string, maxLength = 100): string {
	return (params.get(key) ?? '').slice(0, maxLength);
}

/**
 * Build a relative URL from `base` with the given params changed. `undefined`, `null` and ''
 * remove the key. Changing anything other than `page` resets pagination.
 */
export function withParams(
	base: URL,
	changes: Record<string, string | number | boolean | null | undefined>
): string {
	const params = new URLSearchParams(base.searchParams);
	for (const [key, value] of Object.entries(changes)) {
		if (value === undefined || value === null || value === '') params.delete(key);
		else params.set(key, String(value));
	}
	if (!('page' in changes)) params.delete('page');
	const qs = params.toString();
	return `${base.pathname}${qs ? `?${qs}` : ''}`;
}

export const ISO_DATE = /^\d{4}-\d{2}-\d{2}$/;
