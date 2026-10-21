#!/usr/bin/env bash
# Generates first-time secrets for the production host. Idempotent: existing
# files are never overwritten. Run from the deploy/ directory on the server.
set -euo pipefail
cd "$(dirname "$0")/.."
umask 077
mkdir -p secrets env

# Strip CR/LF: some openssl builds (e.g. Git Bash on Windows) emit CRLF, and a
# stray carriage return makes MongoDB reject the password (SASLprep) or keyfile.
oneline() { tr -d '\r\n'; }
gen() { [ -s "$1" ] || { "${@:2}" | oneline > "$1"; echo "created $1"; }; }

gen secrets/mongo_keyfile        openssl rand -base64 756
gen secrets/mongo_root_password  openssl rand -hex 32
gen secrets/mongo_app_password   openssl rand -hex 32

for app in api storefront admin; do
  [ -s "env/$app.env" ] || { cp "env/$app.env.example" "env/$app.env"; echo "created env/$app.env (edit it!)"; }
done

if grep -q "__MONGO_APP_PASSWORD__" env/api.env; then
  sed -i "s/__MONGO_APP_PASSWORD__/$(cat secrets/mongo_app_password)/" env/api.env
fi
if grep -q "__HASH_PEPPER__" env/api.env; then
  sed -i "s/__HASH_PEPPER__/$(openssl rand -hex 32 | oneline)/" env/api.env
fi
chmod 600 secrets/* env/*.env
echo "Done. Fill in the remaining placeholders in deploy/env/*.env and deploy/.env."
