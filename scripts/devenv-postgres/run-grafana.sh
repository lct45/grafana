#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

. "$ROOT_DIR/scripts/devenv-postgres/env.defaults.sh"
"$ROOT_DIR/scripts/devenv-postgres/setup-custom-ini.sh"

echo "Starting Grafana with Postgres backend (${GF_DATABASE_USER}@${GF_DATABASE_HOST}/${GF_DATABASE_NAME})..."
exec make run
