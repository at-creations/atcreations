import { env } from '$env/dynamic/public';

/** Public storefront URL for a product (used for "View on site"). Returns null when unset/invalid. */
export function storefrontProductUrl(slug: string): string | null {
	const base = env.PUBLIC_STOREFRONT_URL;
	if (!base) return null;
	try {
		const url = new URL(`/products/${encodeURIComponent(slug)}`, base);
		return url.protocol === 'https:' || url.protocol === 'http:' ? url.href : null;
	} catch {
		return null;
	}
}

/** Only render http(s) image URLs coming from the API. */
export function safeImageUrl(url: string | null | undefined): string | undefined {
	if (!url) return undefined;
	try {
		const parsed = new URL(url);
		return parsed.protocol === 'https:' || parsed.protocol === 'http:' ? parsed.href : undefined;
	} catch {
		return undefined;
	}
}

/** Public storefront home page (for "View store"). Returns null when unset/invalid. */
export function storefrontUrl(): string | null {
	const base = env.PUBLIC_STOREFRONT_URL;
	if (!base) return null;
	try {
		const url = new URL('/', base);
		return url.protocol === 'https:' || url.protocol === 'http:' ? url.href : null;
	} catch {
		return null;
	}
}
