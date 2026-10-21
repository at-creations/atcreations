import type { Locale } from './i18n';

const EMBED_HOSTS = new Set(['www.google.com', 'maps.google.com']);

/**
 * URL for the Google Maps iframe. Uses `mapUrl` from store info when it is an https Google
 * Maps URL (the only hosts allowed by the CSP frame-src), otherwise builds an embed URL
 * from the address. Returns null when neither is usable.
 */
export function mapEmbedUrl(
	mapUrl: string | undefined | null,
	address: string | undefined | null,
	locale: Locale
): string | null {
	if (mapUrl) {
		try {
			const url = new URL(mapUrl);
			if (url.protocol === 'https:' && EMBED_HOSTS.has(url.hostname)) return url.href;
		} catch {
			// fall through to the address
		}
	}
	const query = address?.trim();
	if (!query) return null;
	const params = new URLSearchParams({ q: query, hl: locale, output: 'embed' });
	return `https://maps.google.com/maps?${params.toString()}`;
}

/** Google Maps directions link for an address. */
export function directionsUrl(address: string | undefined | null): string | null {
	const query = address?.trim();
	if (!query) return null;
	return `https://www.google.com/maps/dir/?${new URLSearchParams({ api: '1', destination: query }).toString()}`;
}
