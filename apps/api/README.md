# AT Creations API

Go HTTP API serving the storefront (public catalogue) and the admin dashboard (Clerk-authenticated). The contract is [`openapi.yaml`](openapi.yaml).

## Run locally

```bash
docker compose -f ../../docker-compose.dev.yml up -d   # MongoDB rs0 on localhost
cp .env.example .env                                    # fill CLERK_SECRET_KEY, S3_* values
set -a; source .env; set +a
go run ./cmd/api
curl localhost:8080/healthz
```

## Test & lint

```bash
go vet ./...
go test ./...                 # add -race on Linux/macOS (needs cgo)
golangci-lint run
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

## Configuration

All configuration is environment variables, validated at startup (`internal/config`). The process exits with a list of problems if anything required is missing or unsafe for `APP_ENV=production` (wildcard CORS, test Clerk keys, `MAIL_PROVIDER=log`, http CDN, short `HASH_PEPPER`, missing `CLERK_AUTHORIZED_PARTIES`). See [`.env.example`](.env.example) for every variable.

## Endpoints

| Area | Routes |
|---|---|
| Health | `GET /healthz`, `GET /readyz` |
| Public | `GET /v1/categories[/{slug}]`, `GET /v1/products[/{slug}]`, `POST /v1/products/{id}/views`, `GET /v1/store-info`, `POST /v1/orders` (feature-flagged), `POST /v1/orders/track` |
| Admin | `/v1/admin/me`, `products` (+ `images`), `categories` (+ `image`), `store-info`, `orders` (+ `status`, `notes`, `shipping`), `staff` (+ `invitations`, `role`, `ban`, `unban`), `audit` |

Admin requests need `Authorization: Bearer <Clerk session token>` and a role in the Clerk user's `publicMetadata.role`.

## Container

`Dockerfile` builds a static binary into `distroless/static:nonroot`; `/api healthcheck` is the container health probe (no shell needed).
