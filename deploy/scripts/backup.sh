#!/usr/bin/env bash
# Nightly MongoDB backup. Schedule on the host, e.g.:
#   15 3 * * * /opt/atcreations/deploy/scripts/backup.sh >> /var/log/atcreations-backup.log 2>&1
# Copy the resulting archives off-host (e.g. rclone to an R2 bucket with a
# separate, write-only token) — a backup on the same disk is not a backup.
set -euo pipefail
cd "$(dirname "$0")/.."
KEEP_DAYS="${KEEP_DAYS:-14}"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"

docker compose exec -T mongo bash -c '
  mongodump --quiet -u root -p "$(cat /run/secrets/mongo_root_password)" \
    --authenticationDatabase admin --db "$MONGO_APP_DB" --gzip --archive="/backups/atcreations-'"$STAMP"'.archive.gz"'

find ../backups -name "atcreations-*.archive.gz" -mtime +"$KEEP_DAYS" -delete
echo "backup ok: atcreations-$STAMP.archive.gz"
