#!/bin/bash
# Daily encrypted backup of the production MongoDB that Coolify runs for AT Creations.
# Installed on the app server as /usr/local/bin/atc-backup and run by atc-backup.timer
# (setup: docs/deploy-coolify.md, "Backups").
#
#  - mongodump runs inside the mongo container; the root password stays there (written to a
#    temporary config file, never on a command line).
#  - The archive is encrypted with age to a public key. The private key is kept off the server,
#    so this host cannot read its own backups.
#  - Kept KEEP_LOCAL_DAYS here and copied to RCLONE_DEST (Cloudflare R2), kept KEEP_REMOTE_DAYS.
set -euo pipefail
# APP_UUID, AGE_RECIPIENT, RCLONE_DEST, KEEP_LOCAL_DAYS, KEEP_REMOTE_DAYS
. /etc/atc-backup/config
# --s3-no-head: R2 doesn't implement the HEAD ?versionId= check rclone makes after an upload
# (501); integrity is still checked by R2 against the Content-MD5 rclone sends.
RCLONE=(rclone --config /etc/atc-backup/rclone.conf --s3-no-check-bucket --s3-no-head)
DIR=/var/backups/atcreations
install -d -m 0700 "$DIR"

c="$(docker ps --filter "name=^mongo-$APP_UUID" --filter status=running --format '{{.Names}}' | head -1)"
[ -n "$c" ] || { echo "no running mongo container for $APP_UUID" >&2; exit 1; }

# A random part makes the names unguessable: the R2 bucket is publicly readable through the CDN.
name="atcreations-$(date -u +%Y%m%dT%H%M%SZ)-$(od -An -N6 -tx1 /dev/urandom | tr -d ' \n').archive.gz.age"
out="$DIR/$name"
trap 'rm -f "$out.part"' EXIT

docker exec -i "$c" bash -s <<'EOS' | age -r "$AGE_RECIPIENT" >"$out.part"
set -e
umask 077
conf="$(mktemp)"
trap 'rm -f "$conf"' EXIT
pw="$MONGO_INITDB_ROOT_PASSWORD"
q="'"
printf 'password: %s\n' "$q${pw//$q/$q$q}$q" >"$conf"
mongodump --config="$conf" -u root --authenticationDatabase admin --db atcreations --gzip --archive --quiet
EOS

size="$(stat -c %s "$out.part")"
[ "$size" -gt 1024 ] || { echo "backup too small ($size bytes)" >&2; exit 1; }
mv "$out.part" "$out"
echo "wrote $name ($size bytes)"

"${RCLONE[@]}" copyto "$out" "$RCLONE_DEST/$name"
echo "copied to $RCLONE_DEST"

find "$DIR" -name 'atcreations-*.archive.gz.age' -mtime +"$KEEP_LOCAL_DAYS" -delete
"${RCLONE[@]}" delete --min-age "${KEEP_REMOTE_DAYS}d" --include 'atcreations-*.archive.gz.age' "$RCLONE_DEST"
echo "pruned: local > $KEEP_LOCAL_DAYS days, remote > $KEEP_REMOTE_DAYS days"
