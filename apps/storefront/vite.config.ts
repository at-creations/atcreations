import { paraglideVitePlugin } from '@inlang/paraglide-js';
import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vitest/config';
import { paraglideOptions } from './paraglide.config.js';

// Extra Host headers the dev server accepts (e.g. Cloudflare Tunnel hostnames),
// comma separated. Vite rejects unknown hosts to block DNS-rebinding attacks.
const allowedHosts = (process.env.VITE_ALLOWED_HOSTS ?? '')
	.split(',')
	.map((h) => h.trim())
	.filter(Boolean);

export default defineConfig({
	server: {
		allowedHosts,
		// Dev only: when served through Cloudflare Tunnel, keep Cloudflare (and,
		// for unversioned modules such as the client entry, the browser) from
		// caching dev responses. A cached entry script from a previous dev-server
		// run loads stale SvelteKit/Svelte modules next to fresh ones and breaks
		// hydration (blank page). Vite's versioned deps keep their own headers.
		headers: {
			'Cache-Control': 'no-store',
			'Cloudflare-CDN-Cache-Control': 'no-store'
		}
	},
	plugins: [tailwindcss(), sveltekit(), paraglideVitePlugin(paraglideOptions)],
	test: {
		expect: { requireAssertions: true },
		environment: 'node',
		include: ['src/**/*.{test,spec}.{js,ts}']
	}
});
