# Data migration notes (old → new)

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
| `thumbnail` + `images[]` (R2 keys) | `images[]` ordered, cover first | Download each object, run through `media.Process`, upload under `products/{id}/{random}.jpg` |
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
