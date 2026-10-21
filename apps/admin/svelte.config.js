import adapter from '@sveltejs/adapter-node';
import { loadEnv } from 'vite';

/**
 * Content-Security-Policy origins are resolved once, at config/build time.
 * Values come from process.env first, then from .env files (via Vite's loadEnv).
 * See README.md → "Content-Security-Policy" for details.
 */
const env = {
	...loadEnv(
		process.env.NODE_ENV === 'production' ? 'production' : 'development',
		process.cwd(),
		''
	),
	...process.env
};

/** @param {string | undefined} value */
function originOf(value) {
	if (!value) return undefined;
	try {
		const url = new URL(value);
		return url.protocol === 'https:' || url.protocol === 'http:' ? url.origin : undefined;
	} catch {
		return undefined;
	}
}

/**
 * Clerk publishable keys are `pk_(test|live)_<base64("<frontend-api-host>$")>`.
 * @param {string | undefined} key
 */
function clerkFrontendApiFromKey(key) {
	const encoded = key?.split('_')[2];
	if (!encoded) return undefined;
	try {
		const host = Buffer.from(encoded, 'base64').toString('utf8').replace(/\$$/, '');
		return /^[a-z0-9.-]+$/i.test(host) && host.includes('.') ? `https://${host}` : undefined;
	} catch {
		return undefined;
	}
}

/** @param {string | undefined} list */
function splitList(list) {
	return (list ?? '')
		.split(/[\s,]+/)
		.map((v) => v.trim())
		.filter(Boolean);
}

const clerkFapi = [
	...new Set([
		...splitList(env.CSP_CLERK_ORIGINS),
		clerkFrontendApiFromKey(env.PUBLIC_CLERK_PUBLISHABLE_KEY) ?? 'https://*.clerk.accounts.dev'
	])
];
const apiOrigin = originOf(env.PUBLIC_API_URL) ?? 'http://localhost:8080';
const imageOrigins = splitList(env.CSP_IMG_ORIGINS);

/** @type {import('@sveltejs/kit').Config} */
const config = {
	compilerOptions: {
		runes: true
	},
	kit: {
		adapter: adapter(),
		csp: {
			mode: 'auto',
			directives: {
				'default-src': ['self'],
				'script-src': [
					'self',
					...clerkFapi,
					'https://challenges.cloudflare.com',
					'https://*.protect.clerk.com'
				],
				'connect-src': [
					'self',
					...clerkFapi,
					apiOrigin,
					'https://*.protect.clerk.com:*',
					'https://clerk-telemetry.com',
					'https://*.clerk-telemetry.com'
				],
				'img-src': ['self', 'https://img.clerk.com', ...imageOrigins, 'data:', 'blob:'],
				'worker-src': ['self', 'blob:'],
				'style-src': ['self', 'unsafe-inline'],
				'font-src': ['self', 'data:'],
				'frame-src': ['self', 'https://challenges.cloudflare.com', 'https://*.protect.clerk.com'],
				'frame-ancestors': ['none'],
				'base-uri': ['self'],
				'form-action': ['self'],
				'object-src': ['none']
			}
		}
	}
};

export default config;
