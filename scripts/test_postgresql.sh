#!/usr/bin/env bash
# Run commands against a disposable real PostgreSQL server, never a host cluster.
set -euo pipefail
AIDEX_PG_BIN="${AIDEX_POSTGRES_BIN:-$(pg_config --bindir)}"
AIDEX_PG_TEST_DIR="$(mktemp -d /tmp/aidex-pg-test.XXXXXX)"
AIDEX_PG_TEST_DIR="$(cd "$AIDEX_PG_TEST_DIR" && pwd -P)"
chmod 700 "$AIDEX_PG_TEST_DIR"
cleanup() {
  "$AIDEX_PG_BIN/pg_ctl" -D "$AIDEX_PG_TEST_DIR/data" -m immediate -w stop >/dev/null 2>&1 || true
  rm -rf "$AIDEX_PG_TEST_DIR"
}
trap cleanup EXIT INT TERM
"$AIDEX_PG_BIN/initdb" -D "$AIDEX_PG_TEST_DIR/data" --username=aidex_test --auth=trust --encoding=UTF8 --locale=C --data-checksums >"$AIDEX_PG_TEST_DIR/init.log"
# Trust is confined to an owner-only Unix socket; TCP is disabled in this harness.
# The product provisioning test separately checks its SCRAM-authenticated cluster.
"$AIDEX_PG_BIN/pg_ctl" -D "$AIDEX_PG_TEST_DIR/data" -l "$AIDEX_PG_TEST_DIR/server.log" -o "-k $AIDEX_PG_TEST_DIR -h '' -p 55439" -w start >/dev/null
export AIDEX_TEST_POSTGRES_DSN="postgresql://aidex_test@/postgres?host=$AIDEX_PG_TEST_DIR&port=55439"
export AIDEX_POSTGRES_BIN="$AIDEX_PG_BIN"
if [[ "$#" -eq 0 ]]; then
  go test -race -timeout 20m ./...
  go test -race -timeout 20m -tags development ./...
else
  "$@"
fi
