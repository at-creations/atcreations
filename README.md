# AT Creations

Monorepo for the AT Creations store: a bilingual (EN/VI) product catalogue, a staff dashboard, and the API behind both.

| App | Path | Stack |
|---|---|---|
| API | [`apps/api`](apps/api) | Go 1.27 · chi · MongoDB 8 · Redis 8 · Clerk · Cloudflare R2 |
| Storefront | [`apps/storefront`](apps/storefront) | SvelteKit 2 · Svelte 5 · Tailwind 4 · Paraglide |
| Admin | [`apps/admin`](apps/admin) | SvelteKit 2 · Svelte 5 · Clerk (`svelte-clerk`) · TanStack Query |
| API types | [`packages/api-types`](packages/api-types) | Generated from [`openapi.yaml`](apps/api/openapi.yaml) |

Docs: [Architecture](docs/ARCHITECTURE.md) · [Security model](docs/SECURITY.md) · [Deployment](deploy/README.md) · [Data migration](docs/MIGRATION.md)

## Prerequisites

- Go 1.27+, Node 22.20+, pnpm 11 (`corepack enable`), Docker
- A Clerk development instance, an R2 bucket (or any S3-compatible store) for images

## Local development

### Everything in Docker (recommended)

```bash
pnpm dev:stack        # MongoDB, Redis, RustFS (S3), API, storefront, admin — with live reload
pnpm dev:stack:down   # stop (add -v to also wipe the dev database)
```

Storefront http://localhost:5173 · Admin http://localhost:5174 · API http://localhost:8080 (`/readyz`). Put your Clerk development keys in `apps/api/.env` (`CLERK_SECRET_KEY`) and `apps/admin/.env` to sign in. Image uploads go to a local [RustFS](https://rustfs.com) bucket (console http://localhost:9001/rustfs/console/, `atcreations-dev` / `dev-rustfs-secret-key`); add `S3_*`/`CDN_BASE_URL` to `apps/api/.env` only to use a real R2 bucket instead. Ports can be changed with `API_PORT`, `STOREFRONT_PORT`, `ADMIN_PORT`, `MONGO_PORT`, `REDIS_PORT`. Optional DB UI: `docker compose -f docker-compose.dev.yml --profile tools up -d mongo-express` → http://localhost:8088 (dev/dev).

### Public dev URLs (Cloudflare Tunnel)

Expose the dev stack on `*.atcreations.ca` — useful for testing on phones, sharing previews and Clerk flows on a real domain:

```bash
bash docker/dev/cloudflared/setup.sh   # once: login, tunnel, DNS routes, config, root .env
pnpm dev:stack:tunnel                  # dev stack + cloudflared
```

| Service | URL |
|---|---|
| Storefront | https://dev.atcreations.ca |
| Admin | https://dev-admin.atcreations.ca |
| API | https://dev-api.atcreations.ca |

- Runs cloudflared in Docker; nothing to install. Override names with `ZONE`, `TUNNEL_NAME`, `STOREFRONT_HOST`, `ADMIN_HOST`, `API_HOST`. Names are single-level (`dev-admin.`) because Cloudflare's free certificate covers `*.atcreations.ca` but not `*.dev.atcreations.ca`; production names (`atcreations.ca`, `www.`, `admin.`, `api.`, `cdn.`) are refused.
- Config and credentials live in `docker/dev/cloudflared/` and are git-ignored (`cert.pem`, `credentials.json`, `config.yml`). Only `credentials.json` and `config.yml` are mounted into the container; `cert.pem` is only needed to run `setup.sh` again.
- Only the storefront, admin and API are routed. MongoDB, Redis and mongo-express are never exposed; other hostnames return 404.
- **Protect the hostnames with Cloudflare Access** (Zero Trust → Access → Applications → self-hosted, allow only your team's emails). The dev servers are Vite development servers and are not hardened for the public internet. `config.example.yml` shows how to make cloudflared verify the Access token too.
- `localhost` keeps working while the tunnel is up. Delete the `DEV_*` lines from the root `.env` to go back to localhost-only URLs.

### Apps on the host

```bash
pnpm install
docker compose -f docker-compose.dev.yml up -d mongo redis   # 127.0.0.1:27017 and :6379

cp apps/api/.env.example apps/api/.env                # fill in Clerk + R2 values
cp apps/storefront/.env.example apps/storefront/.env
cp apps/admin/.env.example apps/admin/.env

(cd apps/api && go run ./cmd/api)                     # http://localhost:8080
pnpm dev:storefront                                   # http://localhost:5173
pnpm dev:admin                                        # http://localhost:5174
```

The API loads its environment from the shell; use `set -a; source .env; set +a` (bash) or your IDE's env-file support.

## Quality gates

```bash
(cd apps/api && go vet ./... && go test ./...)
pnpm gen:api            # regenerate TS types after editing openapi.yaml
pnpm -r check && pnpm -r lint && pnpm -r test && pnpm -r build
```

CI runs all of the above plus golangci-lint, govulncheck, `pnpm audit`, Trivy and CodeQL.

## AI tooling

This repo is set up for Claude Code with the [Ruflo](https://github.com/ruvnet/ruflo) agent harness (`.claude/`, `.mcp.json`, pinned to `ruflo@3.59.12`) and project skills installed with [`skills`](https://github.com/vercel-labs/skills) (`skills-lock.json`). Project rules for agents are in [`CLAUDE.md`](CLAUDE.md). Find more skills with the `find-skills` skill or `npx skills find <query>`; restore the locked set with `npx skills experimental_install`.
