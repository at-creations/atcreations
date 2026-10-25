#!/bin/bash
# Forced command for the GitHub Actions deploy key, installed on the Coolify host as
# /usr/local/bin/atc-deploy (setup: docs/deploy-coolify.md, step 3).
# The only thing the key can do: `ssh ghdeploy@sr1 sha-<40 hex>` deploys that image tag of
# the AT Creations stack through Coolify's local API, so the call never crosses Cloudflare.
set -euo pipefail
. /etc/atc-deploy/config # COOLIFY_API, APP_UUID
HDR=/etc/atc-deploy/auth.hdr # "Authorization: Bearer <token>", never on a command line

TAG="${SSH_ORIGINAL_COMMAND:-}"
if [[ ! "$TAG" =~ ^sha-[0-9a-f]{40}$ ]]; then
	echo "refused: expected sha-<40 hex commit>" >&2
	exit 2
fi

# One deploy at a time; a second caller waits for the first.
exec 9>/tmp/atc-deploy.lock
flock -w 1200 9

api() { curl -fsS --max-time 30 -H @"$HDR" "$@"; }
field() { python3 -c "import sys, json; print($1)"; }

api -X PATCH "$COOLIFY_API/applications/$APP_UUID/envs" -H 'Content-Type: application/json' \
	-d "{\"key\": \"TAG\", \"value\": \"$TAG\"}" >/dev/null
id="$(api -X POST "$COOLIFY_API/deploy?uuid=$APP_UUID&force=false" | field 'json.load(sys.stdin)["deployments"][0]["deployment_uuid"]')"
echo "Deploying $TAG (Coolify deployment $id)"

for _ in $(seq 1 90); do
	status="$(api "$COOLIFY_API/deployments/$id" | field 'json.load(sys.stdin).get("status", "")')"
	case "$status" in
	finished)
		echo "Deployed $TAG"
		exit 0
		;;
	failed | cancelled*)
		echo "Coolify deployment $id $status; see its logs in Coolify" >&2
		exit 1
		;;
	esac
	sleep 10
done
echo "Timed out waiting for Coolify deployment $id" >&2
exit 1
