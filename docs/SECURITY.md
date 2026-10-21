# Security model

## What changed from the old apps

| Old behaviour (atstore-*) | Risk | New behaviour |
|---|---|---|
| Guest `POST /order` accepted `price`, `discount`, `total`, `status`, `history` from the body | Anyone could buy at $0 or mark orders completed | Body may only contain product ids + quantities; prices, totals and status are server-computed (`orders.Service.Create`) |
| Password-reset / verification emails linked to a `url` taken from the request body | Account takeover via attacker-controlled link | No first-party passwords at all — Clerk owns sign-in, reset and verification |
| `REGISTER_SECRET` let anyone self-register with any role (incl. admin), compared with `===` | Privilege escalation | Invitation-only staff via Clerk; roles only settable by admins through the API (`staff`), with last-admin and self-change guards |
| Profile/staff/product/category updates passed raw bodies to `findByIdAndUpdate` (incl. `$set`, `apiKeys`, `role`) | Mass assignment, operator injection | Typed DTOs decoded with `DisallowUnknownFields`; explicit `$set` of whitelisted fields |
| Unescaped regex from query params (staff, orders) | ReDoS / regex injection | All user text goes through `regexp.QuoteMeta`; query params validated with bounds; `pageSize ≤ 100` |
| Order emails interpolated customer input into HTML | HTML injection from the store's own domain | `html/template` auto-escaping (tested) |
| Order access code stored and compared in plaintext, returned to staff | Credential exposure | HMAC-SHA256 with server pepper, constant-time compare, never returned; tracking uses `POST` and returns a uniform 404 |
| Stock check-then-decrement, no transaction, no restock on cancel | Overselling, drift | Conditional `$inc` inside a MongoDB transaction; cancel restocks atomically; CAS on status |
| Login/user responses leaked encrypted API-key blobs; product lists leaked 768-float embeddings | Data exposure | DTOs map only public fields; API keys and embeddings removed |
| Uploads trusted client MIME type and original filename; page media overwrote by filename | Stored XSS / polyglots, overwrites | Format sniffed from bytes, size and pixel caps (decompression bombs), re-encoded (strips EXIF/GPS), random object keys |
| Redis bound to `0.0.0.0`, `protected-mode no`, port published, default password | Remote takeover | Redis removed; MongoDB has auth + keyfile, runs on an `internal` Docker network with no published port |
| CSRF bound to `req.ip`; rate limits keyed on proxy IP or spoofable `X-Forwarded-For` | Bypass / shared buckets | No cookies → no CSRF surface; client IP resolved only through configured trusted proxies (right-most untrusted hop) |
| Weak default secrets (`JWT_SECRET="secret"`), no startup validation | Forgeable tokens | No first-party JWTs; config refuses to start with missing/unsafe values (`config.validate`) |
| Public `/health` exposed env, memory, uptime | Recon | `/healthz` and `/readyz` return only status |
| Admin route protection was client-side only | Data exposure / UX bypass | Server-side guard in SvelteKit hooks **and** every API route enforces auth + permission |
| CI deployed with `ssh-keyscan` (TOFU), `rsync --delete`, built on the server; AI bot sent full diffs to a third party | Supply chain / MITM | Pinned actions (SHA), CodeQL, Trivy, govulncheck, `pnpm audit`, signed provenance + SBOM, pinned host key, immutable image tags, rollback |

## Controls

**Authentication** — Clerk session JWTs verified against Clerk JWKS (`clerk-sdk-go`), `azp` restricted to the admin origin (`CLERK_AUTHORIZED_PARTIES`), 5s leeway. No cookies are accepted by the API (`AllowCredentials: false`), which removes CSRF entirely.

**Authorization** — `auth.Require(permission)` on every admin route; role from the session token's custom claim, or from the Clerk API (30s cache) when the claim is absent. A role change therefore takes effect when the user's current session token expires (Clerk default ≈60s); a **ban** revokes all sessions immediately, so ban first when removing access urgently. Staff management refuses self-modification and removing the last active admin (mutations are serialized in-process; the API is deployed as a single replica). Clerk outages surface as 503, not as sign-outs.

**Concurrency** — Product edits use optimistic concurrency (`version`): a save based on stale data (including stock changed by an order) is rejected with 409 instead of overwriting live stock.

**Input** — 1 MiB JSON limit, single-object bodies, unknown fields rejected, struct validation, query parameter bounds, 12 MB request cap at Caddy and 10 MB per image in the API.

**Output** — Problem details never include internal errors (logged with request id instead). Security headers on every response: `nosniff`, `frame-ancestors 'none'`, `default-src 'none'` CSP on the API, COOP/CORP, `Referrer-Policy`, HSTS (preload) at Caddy. Admin is `noindex` and `no-store`.

**Client IP** — The API trusts `X-Forwarded-For` only from Caddy and the storefront (fixed container IPs), never the whole Docker subnet. If the host serves IPv6, enable IPv6 on the `edge` network (or disable Docker's userland proxy) so IPv6 visitors are not all seen as the Docker gateway.

**Rate limits** (per real client IP) — global 300/min; view counter 30/min; order placement 5/10 min; order tracking 10/15 min; uploads 60/min; staff management 60/min.

**Data** — Least-privilege Mongo user (`readWrite` on the app DB only); visitor fingerprints are keyed hashes; audit log of every admin mutation; nightly `mongodump` with retention (copy off-host).

**Uploads** — At most two images are decoded at once; header-based limits reject anything over 32 MP or 128 MB decoded (catches 16-bit "decompression bombs"); `GOMEMLIMIT` keeps the Go heap under the container limit. Product image objects are removed by the nightly sweep, which keeps any image still shown on past orders.

**Containers** — Distroless non-root API image; read-only root filesystems, `cap_drop: ALL`, `no-new-privileges`, memory limits, log rotation; MongoDB on an internal-only network.

**Supply chain** — Lockfiles, `pnpm` `minimumReleaseAge` (3 days) and `onlyBuiltDependencies`, Dependabot (grouped), GitHub Actions pinned by SHA, Trivy (deps, secrets, IaC, images), CodeQL (`security-extended`), govulncheck. Ruflo's MCP server is pinned (`ruflo@3.59.12`) and its remote promo/"funnel" fetches are disabled (`RUFLO_FUNNEL=0`).

## Operational checklist

- [ ] Clerk: restrict sign-ups to invitations only; enable MFA (recommended: required for admins); set allowed origins; configure session token claims (see `apps/admin/README.md`).
- [ ] R2: dedicated bucket + API token scoped to that bucket (Object Read & Write); public access only through the CDN custom domain.
- [ ] SendGrid: domain authentication (SPF/DKIM), DMARC on the sending domain.
- [ ] GitHub: branch protection on `main` (required checks: CI, CodeQL), `production` environment with required reviewers, secrets `DEPLOY_SSH_KEY` (deploy-only user), `DEPLOY_KNOWN_HOSTS`, `DEPLOY_HOST`, `DEPLOY_USER`.
- [ ] Host: firewall allows only 22/80/443, SSH keys only, unattended security upgrades, Docker log rotation, off-host backup copy, restore drill.

## Reporting a vulnerability

See [`/SECURITY.md`](../SECURITY.md).
