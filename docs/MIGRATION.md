# Data migration (old → new)

## Running it

`apps/api/cmd/migrate` imports categories, products (with images) and store info from the legacy database. It only **reads** the legacy database (secondary-preferred) and downloads legacy images from the public CDN; it never writes to the old system.

```bash
cd apps/api
export OLD_MONGO_URI='<legacy DATABASE_URL, with tlsCertificateKeyFile pointing at db-key.pem>'
export MONGO_URI='mongodb://127.0.0.1:27017/?replicaSet=rs0&directConnection=true'   # target
export REDIS_URL='redis://:dev-redis-password@127.0.0.1:6379/0'                    # optional
export S3_ENDPOINT=http://127.0.0.1:9000 S3_BUCKET=atcreations-dev S3_ACCESS_KEY_ID=atcreations-dev S3_SECRET_ACCESS_KEY=dev-rustfs-secret-key
export CDN_BASE_URL=http://localhost:9000/atcreations-dev
go run ./cmd/migrate          # dry run: reads everything, checks every image, writes nothing
go run ./cmd/migrate -apply   # migrate
```

- Re-runnable: categories and products keep their ids (replaced, not duplicated); images already migrated are reused (`migration_media` collection), so re-runs don't upload again.
- Store info is only imported when the new system has none, so later edits are never overwritten.
- Not migrated: users (invite staff via Clerk), refresh tokens, API keys, raw page views (they contain IP addresses; view totals are kept on products).

## Field mapping


The new system starts on a fresh MongoDB and a fresh R2 bucket. Nothing reads the old database or bucket. When you are ready to migrate, write a one-off script (e.g. `apps/api/cmd/migrate`) that reads an export of the old data and writes through the same validation rules. This page records the mapping.

## Products (`products` → `products`)

| Old field | New field | Notes |
|---|---|---|
| `_id` | `_id` | Can keep, or generate new ids (old storefront URLs used `/{id}/{slug}`; new URLs are `/products/{slug}`) |
| `name` / `nameVI` | `name.en` / `name.vi` | |
| `description` / `descriptionVI` | `description.en` / `description.vi` | |
| `price` (dollars, float; ≤0 meant "contact") | `priceCents` (int) / `null` | `round(price*100)`; `≤ 0 → null` |
| `discount` (%) | `discountPercent` | Clamp to 0–90; recompute `finalPriceCents`, `hasPrice` |
| `stock` | `stock` | Clamp ≥ 0 |
| `featured` | `featured` | |
| `categoryIds` | `categoryIds` | Remap if category ids change |
| `thumbnail` + `images[]` (R2 keys) | `images[]` ordered, cover first | Download each object, run through `media.Process` (WebP), upload under `products/{id}/{random}.webp` |
| `views` | `views` | |
| — | `slug` | `textx.Slugify(name.en)` with numeric suffix on collision |
| — | `status` | `published` |
| — | `version` | `1` |
| `normalized*`, `embedding`, `recentViews` | — | Derived / dropped; `search`, `sortName*` recomputed |

## Categories

`name/nameVI → name.{en,vi}`, `description/descriptionVI → description.{en,vi}`, `slug` kept (validate pattern), `thumbnail → image` (re-processed), `parent` dropped (was unused), `sortOrder` = 0.

## Store info

`email`, `phone`, `address` copy over. `businessHours[{day: "Monday", openTime, closeTime | "closed"}]` → `{day: "mon", closed, open, close}`.

## Orders

Old orders can be imported for history only: map statuses (`CREATED`/`PENDING` → `pending`, `SHIPPING` → `shipped`, others lowercase), convert money to cents, set `accessCodeHash` to the HMAC of the old plaintext code (customers keep their codes), keep `history` as `{type, at, actor, note}`.

## Users

Old staff accounts are **not** migrated (passwords cannot move to Clerk safely). Invite each staff member from the admin dashboard; their role is set on the invitation.

## Static site images

Marketing images (`data/hero.jpg`, logos, workshop photos) move to the new bucket under `site/` (see `apps/storefront/README.md` for the expected names).
