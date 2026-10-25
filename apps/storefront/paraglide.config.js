// Paraglide compiler options, shared by the Vite plugin (vite.config.ts) and
// `pnpm i18n:compile` (scripts/i18n-compile.js), which check and lint run first
// because the generated src/lib/paraglide is not committed.
/** @type {import('@inlang/paraglide-js').CompilerOptions} */
export const paraglideOptions = {
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
};
