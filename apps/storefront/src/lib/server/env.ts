import { building, dev } from '$app/environment';
import { env as privateEnv } from '$env/dynamic/private';
import { env as publicEnv } from '$env/dynamic/public';

export interface ServerConfig {
	/** Canonical public origin, no trailing slash (e.g. https://atcreations.ca). */
	siteUrl: string;
	/** Browser-facing API origin, used by the client-side view counter. */
	publicApiUrl: string;
	/** Base URL used for SSR API calls (private network address when available). */
	apiInternalUrl: string;
	/** CDN base for static marketing images, no trailing slash. */
	assetBaseUrl: string;
}

const DEV_DEFAULTS = {
	PUBLIC_SITE_URL: 'http://localhost:5173',
	PUBLIC_API_URL: 'http://localhost:8080',
	PUBLIC_ASSET_BASE_URL: 'https://cdn.atcreations.ca/site'
} as const;

function parseHttpUrl(name: string, value: string | undefined): string {
	if (!value) throw new Error(`Missing required environment variable ${name}`);
	let url: URL;
	try {
		url = new URL(value);
	} catch {
		throw new Error(`Environment variable ${name} is not a valid URL`);
	}
	if (url.protocol !== 'http:' && url.protocol !== 'https:') {
		throw new Error(`Environment variable ${name} must be an http(s) URL`);
	}
	return url.href.replace(/\/+$/, '');
}

function read(name: keyof typeof DEV_DEFAULTS): string | undefined {
	const value = publicEnv[name]?.trim();
	if (value) return value;
	if (dev || building) return DEV_DEFAULTS[name];
	return undefined;
}

export function loadConfig(): ServerConfig {
	const publicApiUrl = parseHttpUrl('PUBLIC_API_URL', read('PUBLIC_API_URL'));
	const internal = privateEnv.API_INTERNAL_URL?.trim();
	return {
		siteUrl: new URL(parseHttpUrl('PUBLIC_SITE_URL', read('PUBLIC_SITE_URL'))).origin,
		publicApiUrl: new URL(publicApiUrl).origin,
		apiInternalUrl: internal ? parseHttpUrl('API_INTERNAL_URL', internal) : publicApiUrl,
		assetBaseUrl: parseHttpUrl('PUBLIC_ASSET_BASE_URL', read('PUBLIC_ASSET_BASE_URL'))
	};
}

let cached: ServerConfig | undefined;

/** Validated configuration. Throws (crashing startup via hooks.server.ts) when invalid. */
export function config(): ServerConfig {
	cached ??= loadConfig();
	return cached;
}
