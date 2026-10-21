#!/bin/bash
# Runs once, on first initialisation of an empty data directory, creating a
# least-privilege application user limited to the app database.
#
# NOTE: docker-entrypoint.sh *sources* *.sh init files into its own shell, so
# this file must not change shell options (set -e etc.) or export variables.
# The entrypoint has already read MONGO_INITDB_ROOT_PASSWORD_FILE into
# MONGO_INITDB_ROOT_PASSWORD (and unset the _FILE variable).
# Passwords are passed via the environment, never interpolated into JS.

APP_DB="${MONGO_APP_DB:?MONGO_APP_DB must be set}" \
APP_PASSWORD="$(cat /run/secrets/mongo_app_password)" \
mongosh --quiet \
  -u "$MONGO_INITDB_ROOT_USERNAME" \
  -p "$MONGO_INITDB_ROOT_PASSWORD" \
  --authenticationDatabase admin \
  --eval '
    const appDb = db.getSiblingDB(process.env.APP_DB);
    appDb.createUser({
      user: "atcreations_app",
      pwd: process.env.APP_PASSWORD,
      roles: [{ role: "readWrite", db: process.env.APP_DB }]
    });
  '
