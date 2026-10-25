# Deployment (self-hosted, single host)

Everything — Caddy, storefront, admin, API, MongoDB and Redis — runs on one Linux host with Docker Compose. MongoDB is self-hosted (no Atlas) as a single-node replica set with authentication. Redis is ephemeral (cache, rate limits, locks) with an ACL user and no persistence.

## First-time setup

1. Host: Ubuntu LTS (or similar), Docker Engine + Compose plugin, firewall allowing only 22/80/443, a non-root `deploy` user in the `docker` group, SSH key auth only.
2. DNS: `A/AAAA` records for the storefront, `admin.` and `api.` domains → host. CDN domain (e.g. `cdn.atcreations.ca`) → the R2 bucket's custom domain in Cloudflare.
3. Copy this `deploy/` directory to `/opt/atcreations/deploy` and:
   ```bash
   cp .env.example .env              # domains, ACME email, registry, TAG
   ./scripts/init-secrets.sh         # generates Mongo keyfile/passwords, Redis password, env files, hash pepper
   $EDITOR env/api.env env/admin.env env/storefront.env   # Clerk, R2, SendGrid values
   docker login ghcr.io              # read-only token (read:packages)
   docker compose pull && docker compose up -d --wait
   ```
4. On first boot MongoDB initialises the replica set (healthcheck) and creates the least-privilege `atcreations_app` user (`mongo/init/01-app-user.sh`).
5. In Clerk, set your own user's `publicMetadata` to `{"role":"admin"}`, then invite the rest of the team from the admin dashboard.

## Deploys

Production deploys run on Coolify: after CI passes on `main`, `.github/workflows/deploy.yml` builds and pushes the images and deploys `deploy/coolify/docker-compose.yml` there. Setup: [docs/deploy-coolify.md](../docs/deploy-coolify.md).

On a plain Docker host with this directory, roll to an image tag with `scripts/deploy.sh sha-<commit>`. The script pulls every image (api, imgworker, storefront, admin), waits for health checks, and reverts automatically on failure. Manual rollback:

```bash
./scripts/deploy.sh "$(cat .previous-tag)"
```

The frontends compile their public URLs into the CSP at build time: set the repository variables `PUBLIC_API_URL`, `PUBLIC_ASSET_BASE_URL`, `CSP_IMG_ORIGINS`, `PUBLIC_CLERK_PUBLISHABLE_KEY` and `CSP_CLERK_ORIGINS` (the build fails if one it needs is empty). To check that an image was built by this repository's workflow:

```bash
gh attestation verify oci://ghcr.io/<owner>/atcreations-api:sha-<commit> --owner <owner>
```

Base images are pinned by digest in the Dockerfiles and `docker-compose.yml`; Dependabot proposes updates.

## Backups

Backups are encrypted with [age](https://age-encryption.org) to a public key; the private key never goes on the server. One-time setup:

```bash
sudo apt install age rclone
age-keygen -o backup.key            # on your own machine: keep backup.key offline / in a password manager
cp env/backup.env.example env/backup.env
# set BACKUP_AGE_RECIPIENT to the "public key: age1..." line, and optionally BACKUP_RCLONE_REMOTE
rclone config                       # if using the off-host copy: a remote with a write-only token
```

```bash
# crontab -e (deploy user)
15 3 * * * /opt/atcreations/deploy/scripts/backup.sh >> /var/log/atcreations-backup.log 2>&1
```

Archives (`*.archive.gz.age`) land in `/opt/atcreations/backups` (kept `KEEP_DAYS`, default 14) and are copied to `BACKUP_RCLONE_REMOTE`. The script refuses to run without an age recipient unless `BACKUP_ALLOW_PLAINTEXT=1`. The database password is never put on a command line. Test a restore regularly (decrypt on a machine that has the key, copy the archive into `/opt/atcreations/backups`, then):

```bash
age -d -i backup.key -o atcreations-<stamp>.archive.gz atcreations-<stamp>.archive.gz.age
docker compose exec -T mongo bash -s <<'EOS'
umask 077; conf="$(mktemp)"; trap 'rm -f "$conf"' EXIT
pw="$(cat /run/secrets/mongo_root_password)"; q="'"
printf 'password: %s\n' "$q${pw//$q/$q$q}$q" > "$conf"
mongorestore --config="$conf" -u root --authenticationDatabase admin --gzip --archive=/backups/atcreations-<stamp>.archive.gz --drop
EOS
```

## Operations

| Task | Command |
|---|---|
| Logs | `docker compose logs -f api` |
| Status | `docker compose ps` |
| Mongo shell (admin) | `docker compose exec mongo bash -c 'mongosh -u root -p "$(cat /run/secrets/mongo_root_password)"'` |
| Rotate app DB password | update `secrets/mongo_app_password`, `db.changeUserPassword(...)`, update `MONGO_URI` in `env/api.env`, `docker compose up -d api` |

Secrets: `deploy/.env`, `deploy/env/*.env`, `deploy/secrets/*` are git-ignored and `chmod 600`. Never commit them.
