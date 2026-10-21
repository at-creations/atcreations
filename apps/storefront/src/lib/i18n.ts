import type { LocalizedText } from '@atcreations/api-types';

export const LOCALES = ['en', 'vi'] as const;
export type Locale = (typeof LOCALES)[number];
export const DEFAULT_LOCALE: Locale = 'en';

export function isLocale(value: unknown): value is Locale {
	return typeof value === 'string' && (LOCALES as readonly string[]).includes(value);
}

/** Pick the text for `locale`, falling back to English (and finally to an empty string). */
export function t(text: LocalizedText | null | undefined, locale: Locale): string {
	if (!text) return '';
	const value = locale === 'vi' ? text.vi : text.en;
	return value && value.trim() !== '' ? value : (text.en ?? '');
}

/** BCP 47 tag used for Intl formatting. */
export function intlLocale(locale: Locale): string {
	return locale === 'vi' ? 'vi-VN' : 'en-CA';
}

/** Open Graph locale (`og:locale`). */
export function ogLocale(locale: Locale): string {
	return locale === 'vi' ? 'vi_VN' : 'en_CA';
}

/** Remove a leading `/en` or `/vi` segment from a pathname. */
export function stripLocale(pathname: string): string {
	const match = /^\/(en|vi)(?=\/|$)/.exec(pathname);
	const rest = match ? pathname.slice(match[0].length) : pathname;
	return rest === '' ? '/' : rest;
}

/**
 * Build a locale-prefixed path. `path` is an unlocalized, app-relative path such as
 * `/products` or `/products?page=2` (a leading slash is added when missing).
 */
export function localizedPath(locale: Locale, path = '/'): string {
	const normalized = path.startsWith('/') ? path : `/${path}`;
	if (normalized === '/' || normalized.startsWith('/?') || normalized.startsWith('/#')) {
		return `/${locale}${normalized.slice(1)}`;
	}
	return `/${locale}${normalized}`;
}

/** Same page (path + query + hash) in another locale. */
export function switchLocaleHref(url: URL, locale: Locale): string {
	return localizedPath(locale, stripLocale(url.pathname)) + url.search + url.hash;
}
