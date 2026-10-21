#!/usr/bin/env bash
# Rolls the stack to an image tag. Run on the server from deploy/:
#   ./scripts/deploy.sh sha-<commit>
# The previous tag is kept in .previous-tag so `./scripts/deploy.sh "$(cat .previous-tag)"` rolls back.
set -euo pipefail
cd "$(dirname "$0")/.."

NEW_TAG="${1:?usage: deploy.sh <image-tag>}"
[[ "$NEW_TAG" =~ ^[A-Za-z0-9._-]+$ ]] || { echo "invalid tag" >&2; exit 1; }

CURRENT_TAG="$(grep -E '^TAG=' .env | cut -d= -f2- || true)"

# Pull first, without touching .env: a failed pull leaves the running
# configuration exactly as it was.
TAG="$NEW_TAG" docker compose pull api storefront admin

set_tag() { sed -i "s/^TAG=.*/TAG=$1/" .env; }
set_tag "$NEW_TAG"
if ! docker compose up -d --remove-orphans --wait --wait-timeout 180; then
  echo "rollout failed; reverting to ${CURRENT_TAG}" >&2
  set_tag "$CURRENT_TAG"
  docker compose up -d --remove-orphans
  exit 1
fi

# Only remember a rollback target when the tag actually changed.
if [[ -n "$CURRENT_TAG" && "$CURRENT_TAG" != "$NEW_TAG" ]]; then
  echo "$CURRENT_TAG" > .previous-tag
fi
docker image prune -f >/dev/null
echo "deployed ${NEW_TAG}"
