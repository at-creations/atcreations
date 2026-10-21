import { paraglideVitePlugin } from '@inlang/paraglide-js';
import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vitest/config';

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit(),
		paraglideVitePlugin({
			project: './project.inlang',
			outdir: './src/lib/paraglide',
			emitTsDeclarations: true,
			// URL first (/en/..., /vi/...), then the cookie written by the locale switcher,
			// then Accept-Language, then English.
			strategy: ['url', 'cookie', 'preferredLanguage', 'baseLocale'],
			urlPatterns: [
				{
					pattern: ':protocol://:domain(.*)::port?/:path(.*)?',
					localized: [
						['vi', ':protocol://:domain(.*)::port?/vi/:path(.*)?'],
						['en', ':protocol://:domain(.*)::port?/en/:path(.*)?']
					]
				}
			],
			routeStrategies: [
				{ match: '/sitemap.xml', exclude: true },
				{ match: '/robots.txt', exclude: true }
			]
		})
	],
	test: {
		expect: { requireAssertions: true },
		environment: 'node',
		include: ['src/**/*.{test,spec}.{js,ts}']
	}
});
