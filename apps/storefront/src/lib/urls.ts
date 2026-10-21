import { localizedPath, type Locale } from './i18n';

const SLUG_RE = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;

/** True when `value` is a well-formed slug (lowercase letters, digits, single dashes). */
export function isSlug(value: string | null | undefined): value is string {
	return typeof value === 'string' && value.length <= 120 && SLUG_RE.test(value);
}

type QueryValue = string | number | undefined | null;

export function withQuery(path: string, params: Record<string, QueryValue>): string {
	const qs = new URLSearchParams();
	for (const [key, value] of Object.entries(params)) {
		if (value === undefined || value === null || value === '') continue;
		qs.set(key, String(value));
	}
	const s = qs.toString();
	return s ? `${path}?${s}` : path;
}

export function productHref(locale: Locale, slug: string): string {
	return localizedPath(locale, `/products/${encodeURIComponent(slug)}`);
}

export function productsHref(
	locale: Locale,
	opts: { category?: string | null; page?: number } = {}
): string {
	return withQuery(localizedPath(locale, '/products'), {
		category: opts.category,
		page: opts.page && opts.page > 1 ? opts.page : undefined
	});
}

/** Join a CDN base URL and a file name. */
export function assetUrl(base: string, file: string): string {
	return `${base.replace(/\/+$/, '')}/${file.replace(/^\/+/, '')}`;
}
