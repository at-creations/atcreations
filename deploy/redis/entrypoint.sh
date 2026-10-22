#!/bin/sh
# Builds the ACL file from the mounted secret at start (the secret is only
# readable by root), then drops privileges and starts Redis.
#  - the passwordless "default" user is disabled
#  - the app user may run everything except @dangerous commands
#    (FLUSHALL, CONFIG, DEBUG, KEYS, SHUTDOWN, MODULE, ...)
set -eu
umask 077
PASSWORD="$(tr -d '\r\n' < /run/secrets/redis_password)"
cat > /tmp/users.acl <<EOF
user default off
user atcreations on >${PASSWORD} ~* &* +@all -@dangerous
EOF
chown redis:redis /tmp/users.acl
exec /bin/setpriv --reuid redis --regid redis --clear-groups \
  redis-server /usr/local/etc/redis/redis.conf --aclfile /tmp/users.acl
