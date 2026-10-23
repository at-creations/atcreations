# Architecture

## Overview

```
                 ┌──────────────────────── single host (Docker Compose) ────────────────────────┐
 browser ──TLS──▶│ Caddy ──▶ storefront (SvelteKit SSR) ──http──▶ api (Go) ──▶ MongoDB 8 (rs0)    │
                 │       ──▶ admin (SvelteKit + Clerk)                  │      internal network  │
                 │       ──▶ api                                        │                        │
                 └──────────────────────────────────────────────────────┼────────────────────────┘
                                                                        ├──▶ Clerk (auth, staff users)
                                                                        ├──▶ Cloudflare R2 (images) ──▶ CDN
                                                                        └──▶ SendGrid (order email)
```

| Component | Responsibility |
|---|---|
| **api** (`apps/api`) | Single source of truth. Public catalogue endpoints (`/v1/...`, no credentials, cacheable) and admin endpoints (`/v1/admin/...`, Clerk bearer token + role permission). |
| **storefront** (`apps/storefront`) | Bilingual (en/vi) catalogue. Server-side renders every page by calling the API over the internal network, so pages have real SEO metadata. |
| **admin** (`apps/admin`) | Staff dashboard. Clerk handles sign-in (invitation-only); the browser calls the API with a short-lived Clerk session token. |
| **MongoDB** | Self-hosted, single-node replica set (needed for transactions), auth + keyfile, no published port. |
| **Redis** | Self-hosted, ephemeral (no persistence). Public response cache, shared rate-limit counters, Clerk identity cache and distributed locks. ACL user without dangerous commands, internal network only. |
| **R2** | New bucket; objects are content-addressed by random keys and served via the CDN domain. |

## API design

* Contract: `apps/api/openapi.yaml` (OpenAPI 3.1). TypeScript types are generated into `packages/api-types`.
* Responses: `{ "data": ..., "meta": {page,pageSize,total,totalPages} }`; errors are RFC 9457 `application/problem+json` with a `requestId`.
* Money is integer cents (CAD). `priceCents: null` means "contact for price". Discounts are percentages; `finalPriceCents` is computed server-side and stored for filtering/sorting.
* Localized text is `{en, vi}`; `en` is required.

### Code layout (`apps/api/internal`)

| Package | Purpose |
|---|---|
| `config` | Env parsing with fail-fast validation (refuses unsafe production settings) |
| `platform/httpx` | Problem responses, strict JSON decoding, query parsing, middleware (request id, real client IP, access log, recover, security headers) |
| `platform/mongox` | Connection, index declarations, regex quoting, job leases |
| `platform/textx` | Vietnamese accent folding, slugs |
| `auth` | Clerk JWT verification, role → permission map, `Require` middleware |
| `catalog` | Products, categories, images, view counting, orphaned-media sweep |
| `orders` | State machine, transactional stock reservation, tracking, emails |
| `staff` | Invitations, roles, ban/delete via Clerk Backend API with safety guards |
| `storeinfo` | Store contact info and business hours |
| `audit` | Append-only audit log of admin mutations (2-year TTL) |
| `analytics` | Cookieless storefront page views, daily traffic rollups, the admin sales and traffic report |
| `media` | Image validation/re-encoding, S3/R2 storage |
| `jobs` | Interval jobs guarded by MongoDB leases (multi-replica safe) |
| `platform/redisx` | Redis connection, versioned response cache middleware, distributed lock |

## Roles and permissions

Roles live in Clerk `publicMetadata.role`. The API maps them to permissions (`internal/auth/auth.go`); `GET /v1/admin/me` returns the effective list so the UI can hide actions it would be refused.

| Permission | staff | manager | admin |
|---|:-:|:-:|:-:|
| products:write (create/edit/images) | ✓ | ✓ | ✓ |
| products:delete | | ✓ | ✓ |
| categories:write | | ✓ | ✓ |
| orders:read / orders:write | ✓ | ✓ | ✓ |
| orders:shipping | | ✓ | ✓ |
| store:write | | | ✓ |
| staff:manage, audit:read | | | ✓ |

## Data model (MongoDB, database `atcreations`)

| Collection | Notes |
|---|---|
| `products` | `slug` unique; `status` draft/published/archived; `name/description {en,vi}`; `priceCents`, `discountPercent`, `finalPriceCents`, `hasPrice`; `stock`; `featured`; `categoryIds[]`; ordered `images[] {id,key,w,h,alt}` (max 10, first = cover); `views`, `recentViews`; folded `search`, `sortNameEn/Vi`. |
| `categories` | `slug` unique; `name/description {en,vi}`; optional `image`; `sortOrder`. |
| `product_views` | One doc per visitor/product/15-min bucket, keyed by HMAC(pepper, ip+ua) — no raw IPs. TTL 7 days. |
| `orders` | `number` unique (`AT-YYMMDD-XXXXXX`); `accessCodeHash` (HMAC, never returned); item snapshots with server prices; totals; `history[]` events. |
| `store_info` | Singleton (`_id: "store"`). |
| `audit_log` | Actor, action, resource, request id, IP; TTL 2 years. |
| `job_leases` | Distributed locks for scheduled jobs. |
| `analytics_daily` | One doc per store-local day (`_id: "YYYY-MM-DD"`): `pageViews`, `visitors` (HyperLogLog estimate), `productViews`. TTL 400 days. |
| `analytics_dims` | Per-day counters by dimension (`path`, `ref` host, `device`, `country`, `product`), unique on `{day, dim, key}`. TTL 400 days. |

Indexes are created idempotently at startup (`EnsureIndexes`).

## Search

The old service generated Gemini embeddings but never queried them; search was unescaped regex. Now each product stores an accent-folded `search` field (EN + VI name/description). Queries are folded the same way, split into terms (AND semantics) and each term is regex-quoted. This is fast enough for a catalogue of thousands of items; if it grows, swap in a `$text` index or Atlas-compatible search without changing the API.

## Scheduled jobs

| Job | Interval | What |
|---|---|---|
| `recent-views` | 1h | Recomputes 7-day view counts (drives "relevance" sort) |
| `media-sweep` | 24h | Deletes bucket objects under `products/` and `categories/` that no product, category or order references and are older than 24h (protects in-flight uploads and images shown on past orders) |

## Orders

Online checkout is **off** by default (`FEATURE_PUBLIC_CHECKOUT=false`) because the storefront is catalogue-only today. The API side is complete and safe to enable: prices come from the catalogue, stock is decremented inside a transaction with a `stock >= qty` guard, cancelling restocks, and transitions follow a strict state machine:

```
pending → confirmed → processing → shipped → delivered → completed
   ↓          ↓            ↓                     ↓           ↓
cancelled  cancelled   cancelled              refunded    refunded
```

## Analytics

First-party and cookieless; no third-party script.

- **Collection.** The storefront posts `POST /v1/events/pageview` (`{path, referrer}`) after every navigation; product pages keep posting to `/v1/products/{id}/views`, which now also feeds the daily rollup. The API drops bots, invalid paths and requests with `DNT: 1` or `Sec-GPC: 1` (the storefront also checks them before sending). It stores only counters: the path, the referrer **host**, a device class derived from the user agent, and the country from Cloudflare's `CF-IPCountry` (trusted only from `TRUSTED_PROXIES`).
- **Unique visitors.** HMAC(pepper, day + IP + user agent) is added to a Redis HyperLogLog (`analytics:uv:<day>`, 48 h TTL); only the day's count is written to MongoDB. Without Redis, visitors are not counted. A visitor seen on two days counts on both.
- **Report.** `GET /v1/admin/analytics?days=7|30|90` (`orders:read`) returns revenue and orders (excluding cancelled/refunded) per day, totals plus the previous period for comparison, orders by status, top-selling and most-viewed products, and top pages, sources, devices and countries. Days follow `ANALYTICS_TIMEZONE` (default `America/Vancouver`).

## Caching

| Layer | What | Invalidation |
|---|---|---|
| Redis response cache (API) | `GET /v1/categories*`, `/v1/products*` (5 min), `/v1/store-info` (1 h) | Every catalogue write (products, categories, images, stock changes from orders, the recent-views job) bumps the `catalog` namespace version; store-info writes bump `store`. Old entries are orphaned instantly and expire by TTL. Responses carry `X-Cache: HIT/MISS`. |
| HTTP `Cache-Control` | Public catalogue responses `max-age=60` | Time-based |
| TanStack Query (admin) | Every admin API read, keyed by a typed query-key factory | Cached data is shown at once and revalidated on mount, focus and reconnect; orders poll every 30 s and the analytics report every 60 s while visible; mutations invalidate or set the affected queries and are broadcast to the admin's other tabs; data is cleared on sign-out |

The storefront renders on the server (SEO), so it relies on the API cache rather than a client-side query cache.

## Local development stack

`docker-compose.dev.yml` runs MongoDB, Redis, the API, the storefront and the admin with live reload (`pnpm dev:stack`, which is `docker compose ... up --build --watch`):

| Service | URL | Reload |
|---|---|---|
| storefront | http://localhost:5173 | Vite HMR (source synced into the container) |
| admin | http://localhost:5174 | Vite HMR |
| api | http://localhost:8080 | Rebuilt and restarted on Go file changes |
| mongo | 127.0.0.1:27017 | — |
| redis | 127.0.0.1:6379 (password `dev-redis-password`) | — |
| media (tunnel only) | read-only Caddy proxy in front of RustFS so HTTPS pages can load images; GET/HEAD of `products/`, `categories/`, `site/` images only | — |
| rustfs (S3, stands in for R2) | API http://localhost:9000, console http://localhost:9001/rustfs/console/ (`atcreations-dev` / `dev-rustfs-secret-key`); bucket `atcreations-dev`, created by `rustfs-init` with public read for `products/*`, `categories/*` and `site/*` only; marketing images from `docker/dev/site-assets/` (filled by `fetch-site-assets.sh`) are synced to `site/` | — |
| mongo-express (optional, `--profile tools`) | http://localhost:8088 (dev/dev) | — |

With `pnpm dev:stack:tunnel`, a Cloudflare Tunnel (`docker/dev/cloudflared/`) also publishes the storefront, admin and API on `dev.`, `dev-admin.` and `dev-api.atcreations.ca`; the apps then use those public URLs (CORS, Clerk authorized parties, canonical URLs, Vite host allow-list) via the `DEV_*` variables that `setup.sh` writes to the root `.env`.

Real development keys go in `apps/api/.env` and `apps/admin/.env`; both are optional for the compose stack, which starts with placeholders (sign-in and image uploads need real Clerk/R2 keys).
