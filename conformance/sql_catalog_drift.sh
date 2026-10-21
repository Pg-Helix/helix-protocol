#!/bin/sh
# Executable drift check between sql/api.md's documented signatures and the
# ACTUAL helix_ai functions in a running PostgreSQL with helix-kernel
# installed. Run after `make installcheck` in helix-kernel's CI (which
# checks out this script/fixture at a pinned ref) -- this is the review's
# "real SQL catalog/signature drift check," not another doc that can go
# stale the same way sql/api.md itself already did once.
#
# Usage: PGHOST=/tmp PGDATABASE=contrib_regression ./sql_catalog_drift.sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
EXPECTED="$SCRIPT_DIR/sql_catalog_signatures.txt"

ACTUAL=$(psql -tAq -c "
    SELECT p.proname || '(' || pg_get_function_arguments(p.oid) || ') RETURNS ' || pg_get_function_result(p.oid)
    FROM pg_proc p
    JOIN pg_namespace n ON n.oid = p.pronamespace
    WHERE n.nspname = 'helix_ai'
      AND p.proname IN ('register_source', 'request_refresh', 'claim_job', 'publish_result', 'get_job_input')
    ORDER BY p.proname;
")

if [ "$ACTUAL" != "$(cat "$EXPECTED")" ]; then
    echo "SQL catalog drift between helix-kernel's installed functions and $EXPECTED:" >&2
    ACTUAL_FILE=$(mktemp)
    printf '%s\n' "$ACTUAL" > "$ACTUAL_FILE"
    diff "$ACTUAL_FILE" "$EXPECTED" >&2 || true
    rm -f "$ACTUAL_FILE"
    exit 1
fi

echo "sql_catalog_drift: helix_ai function signatures match $EXPECTED"
