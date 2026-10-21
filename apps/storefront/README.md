# @atcreations/storefront

Public, catalogue-only storefront for AT Creations (no cart, checkout or login).
SvelteKit 2 + Svelte 5 (runes), SSR with `@sveltejs/adapter-node`, Tailwind CSS 4,
Paraglide JS (en/vi), data from the Go API (`apps/api/openapi.yaml`) via `openapi-fetch`.

## Setup

```bash
pnpm install                       # from the repo root
cp apps/storefront/.env.example apps/storefront/.env
pnpm dev:storefront                # http://localhost:5173 → redirects to /en or /vi
```

The API must be running (default `http://localhost:8080`). After changing the API contract,
run `pnpm gen:api` so `@atcreations/api-types` is up to date.

## Environment

| Variable                | When    | Purpose                                                                         |
| ----------------------- | ------- | ------------------------------------------------------------------------------- |
| `PUBLIC_SITE_URL`       | runtime | Canonical origin (canonical/hreflang/OG URLs, sitemap, robots).                 |
| `PUBLIC_API_URL`        | both    | Browser-facing API origin (product view counter). **Also baked into the CSP.**  |
| `API_INTERNAL_URL`      | runtime | Optional SSR base URL (e.g. `http://api:8080`); falls back to `PUBLIC_API_URL`. |
| `PUBLIC_ASSET_BASE_URL` | both    | CDN base for marketing images (`hero.jpg`, `logo-nav.png`, …). **Also in CSP.** |
| `CSP_IMG_ORIGINS`       | build   | Optional extra `img-src` origins (space/comma separated).                       |

Values are validated at server start (`src/lib/server/env.ts`, imported by
`hooks.server.ts`); the server refuses to start in production if a required URL is missing
or not http(s). In dev they default to localhost / the production CDN.
Nothing secret is ever exposed: the client only receives the three public origins via the
root layout data.

### Content-Security-Policy

`kit.csp` (mode `auto`, nonces for SvelteKit's inline bootstrap) is configured in
`svelte.config.js`. Because SvelteKit's CSP directives are static, origins are read from the
**build** environment (`process.env`, then `.env*`). Rebuild if `PUBLIC_API_URL`,
`PUBLIC_ASSET_BASE_URL` or `CSP_IMG_ORIGINS` change. Product/category images must be served
from one of those origins (the API's `url` fields).

Policy: `default-src 'self'; script-src 'self' + nonce; style-src 'self'` (no
`unsafe-inline`: no Svelte transitions or inline `style` attributes are used);
`img-src 'self' data: <asset origin> <api origin> [extra]`; `connect-src 'self' <api origin>`;
`frame-src https://www.google.com https://maps.google.com`; `frame-ancestors 'none'`;
`base-uri 'self'`; `form-action 'self'`; `object-src 'none'`; `upgrade-insecure-requests`
(omitted in `vite dev`). Other security headers are added in `hooks.server.ts`; HSTS is set
by the reverse proxy.

## Scripts

| Script         | What                                          |
| -------------- | --------------------------------------------- |
| `pnpm dev`     | Vite dev server                               |
| `pnpm build`   | Production build (`build/`, run `node build`) |
| `pnpm preview` | Preview the production build                  |
| `pnpm check`   | `svelte-kit sync && svelte-check`             |
| `pnpm lint`    | `prettier --check . && eslint .`              |
| `pnpm format`  | Prettier write                                |
| `pnpm test`    | Vitest unit tests (`src/**/*.spec.ts`)        |

## i18n & routing

- Messages: `messages/en.json`, `messages/vi.json` (flat keys, compiled by Paraglide into
  `src/lib/paraglide/`, git-ignored). All UI copy lives there.
- Every page is served under `/en/...` and `/vi/...`. Routes in `src/routes` are
  unprefixed; `hooks.ts` (`reroute`) strips the prefix and Paraglide's middleware sets the
  locale. Any unprefixed GET (including `/`) is redirected (307) using the
  `PARAGLIDE_LOCALE` cookie (set by the locale switcher), then `Accept-Language`, then `en`.
- `/sitemap.xml` and `/robots.txt` are unlocalized.
- Note: the Paraglide compiler loads inlang plugins from `cdn.jsdelivr.net` at build time
  (see `project.inlang/settings.json`), so the build needs network access.

## Structure

```
src/
  app.css               Tailwind 4 theme tokens + shared utilities (btn, card, field)
  hooks.server.ts       env validation, security headers, locale redirect, Paraglide, handleError
  hooks.ts              reroute (de-localize URLs)
  lib/
    server/api.ts       typed openapi-fetch client (5s timeout), 404 → error(404), else 503
    server/env.ts       env validation
    components/         Header, Footer, LocaleSwitcher, ProductCard, Price, ProductBadges,
                        CategoryPills, Pagination, Gallery, Lightbox, Seo, JsonLd, …
    i18n.ts             locales, t(), localized paths
    format.ts hours.ts pagination.ts product.ts search.ts seo.ts urls.ts maps.ts (+ *.spec.ts)
  routes/
    +layout.server.ts   store info + public config
    +page.*             home
    products/ products/[slug]/ search/ about/ contact/ workshops/ workshops/plaster-painting/
    privacy/ terms/ sitemap.xml/ robots.txt/ +error.svelte
```

## Notes

- Catalogue responses send `cache-control: public, max-age=60, s-maxage=300, stale-while-revalidate=600`.
- Product descriptions are plain text rendered with `whitespace-pre-line`; `{@html}` is
  banned by ESLint. Product JSON-LD is serialized with `<`, `>` and `&` escaped.
- `svelte/no-navigation-without-resolve` is disabled: links are locale-prefixed paths built by
  `$lib/i18n`/`$lib/urls` and the app uses no `paths.base`.
