#!/usr/bin/env bash
# One-time setup of the dev-stack Cloudflare Tunnel. Run from anywhere:
#
#   bash docker/dev/cloudflared/setup.sh
#
# It runs cloudflared in Docker (nothing to install) and:
#   1. logs in to Cloudflare (opens a browser link; writes cert.pem here)
#   2. creates the tunnel, or reuses it, and stores credentials.json here
#   3. creates proxied CNAMEs for the dev hostnames in your zone
#   4. renders config.yml and the DEV_* variables in the repo-root .env
#
# Override defaults with environment variables, e.g.
#   ZONE=atcreations.ca TUNNEL_NAME=atcreations-dev-bao bash docker/dev/cloudflared/setup.sh
#
# cert.pem, credentials.json and config.yml are git-ignored. cert.pem is only
# needed for setup (not at runtime); delete it afterwards if you like.
set -euo pipefail

IMAGE="cloudflare/cloudflared:2026.10.1"
ZONE="${ZONE:-atcreations.ca}"
TUNNEL_NAME="${TUNNEL_NAME:-atcreations-dev}"
# Single-level names: Cloudflare's Universal SSL covers *.zone but not *.*.zone.
STOREFRONT_HOST="${STOREFRONT_HOST:-dev.$ZONE}"
ADMIN_HOST="${ADMIN_HOST:-dev-admin.$ZONE}"
API_HOST="${API_HOST:-dev-api.$ZONE}"
MEDIA_HOST="${MEDIA_HOST:-dev-media.$ZONE}"

cd "$(dirname "$0")"
REPO_ROOT="$(cd ../../.. && pwd)"

# Never let the dev tunnel take over production names.
for host in "$STOREFRONT_HOST" "$ADMIN_HOST" "$API_HOST" "$MEDIA_HOST"; do
  case "$host" in
    "$ZONE" | "www.$ZONE" | "admin.$ZONE" | "api.$ZONE" | "cdn.$ZONE")
      echo "refusing to route production hostname $host to the dev tunnel" >&2
      exit 1 ;;
  esac
done

# Docker needs a host path it understands: D:/... under Git Bash, plain elsewhere.
DOCKER_ARGS=(--rm -i -e HOME=/home/nonroot)
if HOST_DIR="$(pwd -W 2>/dev/null)"; then
  export MSYS_NO_PATHCONV=1 # keep container paths intact in Git Bash
else
  HOST_DIR="$(pwd)"
  DOCKER_ARGS+=(--user "$(id -u):$(id -g)") # write files as you on Linux/macOS
fi
cf() { docker run "${DOCKER_ARGS[@]}" -v "$HOST_DIR:/home/nonroot/.cloudflared" "$IMAGE" "$@"; }
CERT=/home/nonroot/.cloudflared/cert.pem

umask 077

if [ ! -s cert.pem ]; then
  echo "==> Logging in to Cloudflare: open the URL below and pick the $ZONE zone"
  cf tunnel login
fi

tunnel_id() {
  # Prefer the id recorded in our credentials file; otherwise take the id of
  # the first tunnel object (not a nested connection id) from the list.
  if [ -s credentials.json ]; then
    tr -d '\n ' < credentials.json | sed -n 's/.*"TunnelID":"\([0-9a-f-]\{36\}\)".*/\1/p'
    return
  fi
  cf tunnel --origincert "$CERT" list --name "$TUNNEL_NAME" --output json 2>/dev/null |
    tr -d '\n ' | sed -n 's/^\[{"id":"\([0-9a-f-]\{36\}\)".*/\1/p'
}

ID="$(tunnel_id)"
if [ -z "$ID" ]; then
  echo "==> Creating tunnel $TUNNEL_NAME"
  cf tunnel --origincert "$CERT" create --credentials-file /home/nonroot/.cloudflared/credentials.json "$TUNNEL_NAME" >/dev/null
  ID="$(tunnel_id)"
elif [ ! -s credentials.json ]; then
  echo "==> Tunnel $TUNNEL_NAME exists; fetching its credentials"
  cf tunnel --origincert "$CERT" token --cred-file /home/nonroot/.cloudflared/credentials.json "$TUNNEL_NAME" >/dev/null
else
  echo "==> Reusing tunnel $TUNNEL_NAME ($ID)"
fi
[ -n "$ID" ] || { echo "could not determine the tunnel id" >&2; exit 1; }

for host in "$STOREFRONT_HOST" "$ADMIN_HOST" "$API_HOST" "$MEDIA_HOST"; do
  echo "==> DNS route $host → $TUNNEL_NAME"
  # Without --overwrite-dns this fails safely if a different record already exists.
  cf tunnel --origincert "$CERT" route dns "$TUNNEL_NAME" "$host" ||
    echo "    (skipped: a record for $host already exists — check it points to $ID.cfargotunnel.com)"
done

sed -e "s/__TUNNEL_ID__/$ID/" \
  -e "s/__STOREFRONT_HOST__/$STOREFRONT_HOST/" \
  -e "s/__ADMIN_HOST__/$ADMIN_HOST/" \
  -e "s/__API_HOST__/$API_HOST/" \
  -e "s/__MEDIA_HOST__/$MEDIA_HOST/" \
  config.example.yml > config.yml
chmod 600 cert.pem credentials.json config.yml 2>/dev/null || true

# Point the dev stack at the public URLs (repo-root .env is git-ignored and
# read automatically by docker compose). Existing values are replaced.
ENV_FILE="$REPO_ROOT/.env"
touch "$ENV_FILE"
set_env() { # portable in-place update (BSD and GNU sed differ on -i)
  grep -v "^$1=" "$ENV_FILE" > "$ENV_FILE.tmp" || true
  printf '%s=%s\n' "$1" "$2" >> "$ENV_FILE.tmp"
  mv "$ENV_FILE.tmp" "$ENV_FILE"
}
set_env DEV_STOREFRONT_URL "https://$STOREFRONT_HOST"
set_env DEV_ADMIN_URL "https://$ADMIN_HOST"
set_env DEV_API_URL "https://$API_HOST"
set_env DEV_TUNNEL_HOSTS "$STOREFRONT_HOST,$ADMIN_HOST"
set_env DEV_MEDIA_ORIGIN "https://$MEDIA_HOST"

cat <<EOF

Tunnel ready: $TUNNEL_NAME ($ID)
  storefront  https://$STOREFRONT_HOST
  admin       https://$ADMIN_HOST
  api         https://$API_HOST
  media       https://$MEDIA_HOST (read-only images)

Start the stack with the tunnel:   pnpm dev:stack:tunnel
Strongly recommended: protect these hostnames with Cloudflare Access (see README).
EOF
