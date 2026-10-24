#!/usr/bin/env bash
# Nightly MongoDB backup. Schedule on the host, e.g.:
#   15 3 * * * /opt/atcreations/deploy/scripts/backup.sh >> /var/log/atcreations-backup.log 2>&1
#
# Settings come from deploy/env/backup.env (see backup.env.example):
#   BACKUP_AGE_RECIPIENT   age public key; archives are encrypted to it (the private key stays
#                          off the server). Required unless BACKUP_ALLOW_PLAINTEXT=1.
#   BACKUP_RCLONE_REMOTE   optional rclone destination for the off-host copy, e.g.
#                          "r2-backups:atcreations-backups" (use a write-only token).
#   KEEP_DAYS              local retention (default 14).
set -euo pipefail
cd "$(dirname "$0")/.."
if [[ -f env/backup.env ]]; then
  set -a
  # shellcheck disable=SC1091
  . env/backup.env
  set +a
fi
KEEP_DAYS="${KEEP_DAYS:-14}"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
NAME="atcreations-$STAMP.archive.gz"

if [[ -z "${BACKUP_AGE_RECIPIENT:-}" && "${BACKUP_ALLOW_PLAINTEXT:-}" != "1" ]]; then
  echo "backup refused: set BACKUP_AGE_RECIPIENT in env/backup.env (or BACKUP_ALLOW_PLAINTEXT=1)" >&2
  exit 1
fi
if [[ -n "${BACKUP_AGE_RECIPIENT:-}" ]] && ! command -v age >/dev/null; then
  echo "backup refused: 'age' is not installed (apt install age)" >&2
  exit 1
fi

# The password goes into a private config file inside the container, never onto a command
# line (visible to every process on the host).
docker compose exec -T -e NAME="$NAME" mongo bash -s <<'EOS'
set -euo pipefail
umask 077
conf="$(mktemp)"
trap 'rm -f "$conf"' EXIT
pw="$(cat /run/secrets/mongo_root_password)"
q="'"
printf 'password: %s\n' "$q${pw//$q/$q$q}$q" > "$conf" # a YAML single-quoted string
mongodump --quiet --config="$conf" -u root --authenticationDatabase admin \
  --db "$MONGO_APP_DB" --gzip --archive="/backups/$NAME"
EOS

FILE="../backups/$NAME"
if [[ -n "${BACKUP_AGE_RECIPIENT:-}" ]]; then
  age -r "$BACKUP_AGE_RECIPIENT" -o "$FILE.age" "$FILE"
  rm -f "$FILE"
  FILE="$FILE.age"
fi

if [[ -n "${BACKUP_RCLONE_REMOTE:-}" ]]; then
  rclone copy --no-traverse "$FILE" "$BACKUP_RCLONE_REMOTE/"
fi

find ../backups -name "atcreations-*.archive.gz*" -mtime +"$KEEP_DAYS" -delete
echo "backup ok: $(basename "$FILE")${BACKUP_RCLONE_REMOTE:+ (copied to $BACKUP_RCLONE_REMOTE)}"
