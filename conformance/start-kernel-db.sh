#!/bin/sh
# Starts a disposable PostgreSQL with a pinned helix-kernel checkout's
# extension installed, so sql_catalog_drift.sh can compare sql/api.md
# against the ACTUAL installed function signatures rather than another
# document. Mirrors helix-cortex's ci/start-test-db.sh.
#
# Usage: KERNEL_DIR=/path/to/helix-kernel sh conformance/start-kernel-db.sh
set -eu
KERNEL_DIR=${KERNEL_DIR:?set KERNEL_DIR to a helix-kernel checkout}
PG_MAJOR=${PG_MAJOR:-17}
PORT=${PORT:-55434}
NAME=${NAME:-protocol-ci-postgres}

docker rm -f "$NAME" >/dev/null 2>&1 || true
docker run -d --name "$NAME" -e POSTGRES_HOST_AUTH_METHOD=trust -p "127.0.0.1:$PORT:5432" -v "$KERNEL_DIR":/src:ro "postgres:$PG_MAJOR" postgres >/dev/null
docker exec "$NAME" sh -c "apt-get update -qq && apt-get install -y -qq build-essential postgresql-server-dev-$PG_MAJOR >/dev/null" >&2
docker exec "$NAME" sh -c "mkdir /build && cp -r /src/. /build/ && cd /build && make >/dev/null && make install >/dev/null" >&2
docker restart "$NAME" >/dev/null
i=0
until docker exec "$NAME" pg_isready -U postgres >/dev/null 2>&1; do
    i=$((i + 1))
    [ "$i" -le 60 ] || { echo "start-kernel-db.sh: postgres never became ready" >&2; exit 1; }
    sleep 1
done

docker exec -u postgres "$NAME" psql -v ON_ERROR_STOP=1 -c "CREATE EXTENSION helix;" >&2

echo "export PGHOST=127.0.0.1"
echo "export PGPORT=$PORT"
echo "export PGUSER=postgres"
echo "export PROTOCOL_CI_CONTAINER=$NAME"
