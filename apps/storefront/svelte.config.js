import adapter from '@sveltejs/adapter-node';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import { loadEnv } from 'vite';

/*
 * Content-Security-Policy.
 *
 * `kit.csp` directives are static: they are computed here, when the config is loaded
 * (i.e. at `vite build` / `vite dev` time), NOT at runtime. Origins are therefore read from
 * the build environment (process.env, falling back to .env files):
 *
 *   PUBLIC_API_URL         -> connect-src (+ img-src, in case the API serves images)
 *   PUBLIC_ASSET_BASE_URL  -> img-src (CDN for marketing images)
 *   CSP_IMG_ORIGINS        -> optional, extra space/comma separated img-src origins
 *                             (e.g. a separate product-image CDN)
 *
 * Rebuild the app if any of these origins change.
 */
const mode = process.env.NODE_ENV === 'production' ? 'production' : 'development';
const env = { ...loadEnv(mode, process.cwd(), ''), ...process.env };
const isDev = process.argv.includes('dev');

/** @param {string | undefined} value @param {string} fallback */
function originOf(value, fallback) {
	try {
		return value ? new URL(value).origin : fallback;
	} catch {
		return fallback;
	}
}

const apiOrigin = originOf(env.PUBLIC_API_URL, 'http://localhost:8080');
const assetOrigin = originOf(env.PUBLIC_ASSET_BASE_URL, 'https://cdn.atcreations.ca');
const extraImgOrigins = (env.CSP_IMG_ORIGINS ?? '')
	.split(/[\s,]+/)
	.filter(Boolean)
	.map((v) => originOf(v, ''))
	.filter(Boolean);

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess(),
	compilerOptions: {
		// Force runes mode for the project, except for libraries.
		runes: ({ filename }) => (filename.split(/[/\\]/).includes('node_modules') ? undefined : true)
	},
	kit: {
		adapter: adapter(),
		csp: {
			mode: 'auto',
			directives: {
				'default-src': ['self'],
				'script-src': ['self'],
				'style-src': ['self'],
				'img-src': ['self', 'data:', assetOrigin, apiOrigin, ...extraImgOrigins],
				'font-src': ['self'],
				'connect-src': ['self', apiOrigin, ...(isDev ? ['ws://localhost:*'] : [])],
				'frame-src': ['https://www.google.com', 'https://maps.google.com'],
				'frame-ancestors': ['none'],
				'base-uri': ['self'],
				'form-action': ['self'],
				'object-src': ['none'],
				'upgrade-insecure-requests': !isDev
			}
		}
	}
};

export default config;
