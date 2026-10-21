# Deployment (self-hosted, single host)

Everything — Caddy, storefront, admin, API and MongoDB — runs on one Linux host with Docker Compose. MongoDB is self-hosted (no Atlas) as a single-node replica set with authentication.

## First-time setup

1. Host: Ubuntu LTS (or similar), Docker Engine + Compose plugin, firewall allowing only 22/80/443, a non-root `deploy` user in the `docker` group, SSH key auth only.
2. DNS: `A/AAAA` records for the storefront, `admin.` and `api.` domains → host. CDN domain (e.g. `cdn.atcreations.ca`) → the R2 bucket's custom domain in Cloudflare.
3. Copy this `deploy/` directory to `/opt/atcreations/deploy` and:
   ```bash
   cp .env.example .env              # domains, ACME email, registry, TAG
   ./scripts/init-secrets.sh         # generates Mongo keyfile/passwords, env files, hash pepper
   $EDITOR env/api.env env/admin.env env/storefront.env   # Clerk, R2, SendGrid values
   docker login ghcr.io              # read-only token (read:packages)
   docker compose pull && docker compose up -d --wait
   ```
4. On first boot MongoDB initialises the replica set (healthcheck) and creates the least-privilege `atcreations_app` user (`mongo/init/01-app-user.sh`).
5. In Clerk, set your own user's `publicMetadata` to `{"role":"admin"}`, then invite the rest of the team from the admin dashboard.

## Deploys

Push a tag `vX.Y.Z` → `.github/workflows/deploy.yml` builds and pushes images (with SBOM + provenance) and, after approval in the `production` environment, runs `scripts/deploy.sh sha-<commit>` over SSH. The script pulls, waits for health checks, and reverts automatically on failure. Manual rollback:

```bash
./scripts/deploy.sh "$(cat .previous-tag)"
```

## Backups

```bash
# crontab -e (deploy user)
15 3 * * * /opt/atcreations/deploy/scripts/backup.sh >> /var/log/atcreations-backup.log 2>&1
```

Archives land in `/opt/atcreations/backups`. **Copy them off-host** (e.g. `rclone copy` to a separate bucket with a write-only token) and test a restore regularly:

```bash
docker compose exec -T mongo bash -c 'mongorestore -u root -p "$(cat /run/secrets/mongo_root_password)" --authenticationDatabase admin --gzip --archive=/backups/<file> --drop'
```

## Operations

| Task | Command |
|---|---|
| Logs | `docker compose logs -f api` |
| Status | `docker compose ps` |
| Mongo shell (admin) | `docker compose exec mongo bash -c 'mongosh -u root -p "$(cat /run/secrets/mongo_root_password)"'` |
| Rotate app DB password | update `secrets/mongo_app_password`, `db.changeUserPassword(...)`, update `MONGO_URI` in `env/api.env`, `docker compose up -d api` |

Secrets: `deploy/.env`, `deploy/env/*.env`, `deploy/secrets/*` are git-ignored and `chmod 600`. Never commit them.
