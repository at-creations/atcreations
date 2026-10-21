# @atcreations/admin

Staff dashboard for AT Creations: SvelteKit 2 + Svelte 5 (runes), Tailwind CSS 4, Paraglide (en/vi), Clerk auth via [`svelte-clerk`](https://github.com/wobsoriano/svelte-clerk). It talks to the Go API (`apps/api`, contract in `apps/api/openapi.yaml`) with a typed `openapi-fetch` client.

## Setup

```bash
pnpm install                       # from the repo root
cp apps/admin/.env.example apps/admin/.env   # then fill in the values
pnpm dev:admin                     # http://localhost:5173 (or the next free port)
```

| Variable                       | Where           | Purpose                                                                            |
| ------------------------------ | --------------- | ---------------------------------------------------------------------------------- |
| `PUBLIC_CLERK_PUBLISHABLE_KEY` | runtime         | Clerk publishable key (`pk_test_…` / `pk_live_…`)                                  |
| `CLERK_SECRET_KEY`             | runtime, server | Clerk secret key. Server only — never prefix with `PUBLIC_`.                       |
| `PUBLIC_API_URL`               | runtime + build | Go API origin used by the browser (e.g. `https://api.atcreations.ca`)              |
| `PUBLIC_STOREFRONT_URL`        | runtime         | Storefront origin for "View on site" links                                         |
| `CSP_CLERK_ORIGINS`            | build           | Extra Clerk Frontend API origins for the CSP (e.g. `https://clerk.atcreations.ca`) |
| `CSP_IMG_ORIGINS`              | build           | Image CDN origins for the CSP `img-src` (e.g. `https://cdn.atcreations.ca`)        |

Scripts (run with `pnpm --filter @atcreations/admin <script>`): `dev`, `build`, `preview`, `start` (runs `build/` with Node), `check`, `lint`, `format`, `test`. `check` and `lint` first compile the Paraglide messages (`i18n:compile`).

`build` needs no secrets: Clerk keys are read at runtime through `$env/dynamic/*`. Only the CSP origins are baked in at build time (see below), so build with the production `PUBLIC_API_URL` / `PUBLIC_CLERK_PUBLISHABLE_KEY` / `CSP_*` in the environment. Without them the CSP falls back to `http://localhost:8080` and `https://*.clerk.accounts.dev`.

## Clerk dashboard

1. **Restrict sign-ups** — _Configure → Restrictions_: set sign-up mode to **Restricted** (invitation only). The app only renders `<SignIn/>`; staff are invited from the Staff page (`POST /v1/admin/staff/invitations`).
2. **Paths** — sign-in URL `/sign-in`, after sign-out `/sign-in`.
3. **First admin** — create/invite your own account, then in _Users → (you) → Metadata_ set public metadata to:
   ```json
   { "role": "admin" }
   ```
   Roles are `staff`, `manager` and `admin`. A signed-in account without a role sees a "No access" page.
4. **Session token** — _Configure → Sessions → Customize session token_:
   ```json
   {
   	"role": "{{user.public_metadata.role}}",
   	"email": "{{user.primary_email_address}}",
   	"name": "{{user.full_name}}"
   }
   ```
5. **Allowed origins** — add the admin origin to the instance's allowed origins / domains (production instances use `clerk.<your-domain>`; add that origin to `CSP_CLERK_ORIGINS`).

## How auth and authorization work

- `src/hooks.server.ts` runs `withClerkHandler()`, then an auth guard: every route except `/sign-in/*` requires `locals.auth().userId`, otherwise `303 → /sign-in?redirect_url=<path>`. The redirect target is validated by `safeRedirectPath` (same-origin path, single leading `/`, no `//`, `\`, schemes or control characters) both in the hook and on the sign-in page.
- All pages are client-rendered (`ssr = false`); the server only serialises the Clerk auth state. Data is fetched in the browser with `Authorization: Bearer <clerk session token>` (`credentials: 'omit'`, token never stored).
- `(app)/+layout.svelte` calls `GET /v1/admin/me`. A `403` shows the "No access — ask an administrator to invite you" page; a `401` reloads into sign-in. `me.permissions` drives `session.can(permission)`, which hides nav items/actions and renders a 403 view for pages you can't use. **This is UX only — the API enforces every permission.**
- API errors are RFC 9457 problem+json, parsed into `ApiError { status, title, detail, errors[] }`; field errors are mapped onto forms and `detail` (e.g. 409 "cannot demote the last admin") is shown in a toast as plain text.

## Content-Security-Policy

Configured in `svelte.config.js` (`kit.csp`, mode `auto` → nonces for SvelteKit's inline bootstrap), following [Clerk's CSP guide](https://clerk.com/docs/security/clerk-csp):

- `script-src 'self'` + Clerk Frontend API (derived from `PUBLIC_CLERK_PUBLISHABLE_KEY` at build time, plus `CSP_CLERK_ORIGINS`, fallback `https://*.clerk.accounts.dev`) + `https://challenges.cloudflare.com` + `https://*.protect.clerk.com`
- `connect-src 'self'` + Clerk FAPI + `PUBLIC_API_URL` origin + `https://*.protect.clerk.com:*` + Clerk telemetry hosts
- `img-src 'self' https://img.clerk.com data: blob:` + `CSP_IMG_ORIGINS`
- `style-src 'self' 'unsafe-inline'` (Clerk injects runtime styles), `worker-src 'self' blob:`, `frame-src 'self' https://challenges.cloudflare.com https://*.protect.clerk.com`
- `frame-ancestors 'none'`, `base-uri 'self'`, `form-action 'self'`, `object-src 'none'`, `default-src 'self'`

The hook also sets `X-Content-Type-Options`, `Referrer-Policy: same-origin`, `Permissions-Policy`, `Cross-Origin-Opener-Policy: same-origin`, `X-Frame-Options: DENY`, `X-Robots-Tag: noindex, nofollow` and `Cache-Control: no-store` on HTML. `static/robots.txt` disallows everything.

## i18n

Messages live in `messages/en.json` and `messages/vi.json` (a unit test keeps keys and parameters in sync). The locale comes from the `PARAGLIDE_LOCALE` cookie, then the browser's preferred language, then `en`; the switcher in the sidebar sets the cookie. URLs are not localized.

## Tests

`pnpm --filter @atcreations/admin test` covers money/cents conversion, the redirect validator, permission helpers, the order-status state machine, slugify, problem+json parsing, URL query helpers, image checks, business hours and message-catalog parity.
