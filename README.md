# AT Creations

Monorepo for the AT Creations store: a bilingual (EN/VI) product catalogue, a staff dashboard, and the API behind both.

| App | Path | Stack |
|---|---|---|
| API | [`apps/api`](apps/api) | Go 1.27 · chi · MongoDB 8 · Clerk · Cloudflare R2 |
| Storefront | [`apps/storefront`](apps/storefront) | SvelteKit 2 · Svelte 5 · Tailwind 4 · Paraglide |
| Admin | [`apps/admin`](apps/admin) | SvelteKit 2 · Svelte 5 · Clerk (`svelte-clerk`) |
| API types | [`packages/api-types`](packages/api-types) | Generated from [`openapi.yaml`](apps/api/openapi.yaml) |

Docs: [Architecture](docs/ARCHITECTURE.md) · [Security model](docs/SECURITY.md) · [Deployment](deploy/README.md) · [Data migration](docs/MIGRATION.md)

## Prerequisites

- Go 1.27+, Node 22.20+, pnpm 11 (`corepack enable`), Docker
- A Clerk development instance, an R2 bucket (or any S3-compatible store) for images

## Local development

```bash
pnpm install
docker compose -f docker-compose.dev.yml up -d        # MongoDB replica set on 127.0.0.1:27017

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
