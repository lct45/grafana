#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

. "$ROOT_DIR/scripts/devenv-postgres/env.defaults.sh"

wait_for_postgres() {
  for _ in $(seq 1 60); do
    if PGPASSWORD="$GF_DATABASE_PASSWORD" pg_isready -h 127.0.0.1 -p 5432 -U "$GF_DATABASE_USER" -d "$GF_DATABASE_NAME" >/dev/null 2>&1; then
      echo "Postgres is ready at ${GF_DATABASE_HOST}."
      return 0
    fi
    sleep 2
  done
  return 1
}

if wait_for_postgres; then
  exit 0
fi

if ! command -v docker >/dev/null 2>&1; then
  echo "Postgres is not reachable and Docker is unavailable." >&2
  exit 1
fi

echo "Starting Postgres via devenv (devenv/docker/blocks/postgres)..."
make devenv sources=postgres

echo "Waiting for Postgres to accept connections..."
if wait_for_postgres; then
  exit 0
fi

echo "Postgres did not become ready in time." >&2
exit 1
