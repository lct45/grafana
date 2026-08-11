#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

. "$ROOT_DIR/scripts/devenv-postgres/env.defaults.sh"

PORT="${GRAFANA_VERIFY_PORT:-3000}"
HEALTH_URL="http://127.0.0.1:${PORT}/api/health"
PID_FILE="/tmp/grafana-postgres-verify.pid"
LOG_FILE="/tmp/grafana-postgres-verify.log"

cleanup() {
  if [[ -f "$PID_FILE" ]]; then
    kill "$(cat "$PID_FILE")" 2>/dev/null || true
    rm -f "$PID_FILE"
  fi
}
trap cleanup EXIT

echo "==> Starting Postgres"
"$ROOT_DIR/scripts/devenv-postgres/start-postgres.sh"

echo "==> Preparing conf/custom.ini"
"$ROOT_DIR/scripts/devenv-postgres/setup-custom-ini.sh"

echo "==> Building Grafana"
make GO_BUILD_DEV=1 build-go

echo "==> Starting Grafana against Postgres"
rm -f "$LOG_FILE"
./bin/grafana server \
  -packaging=dev \
  cfg:app_mode=development \
  cfg:server.http_port="$PORT" \
  >"$LOG_FILE" 2>&1 &
echo $! >"$PID_FILE"

echo "==> Waiting for Grafana health endpoint"
for _ in $(seq 1 120); do
  if curl -sf "$HEALTH_URL" >/dev/null 2>&1; then
    break
  fi
  sleep 2
done

if ! curl -sf "$HEALTH_URL" >/dev/null; then
  echo "Grafana health check failed. Recent logs:" >&2
  tail -50 "$LOG_FILE" >&2 || true
  exit 1
fi

HEALTH_JSON="$(curl -sf "$HEALTH_URL")"
echo "Health response: $HEALTH_JSON"

if ! echo "$HEALTH_JSON" | grep -q '"database"[[:space:]]*:[[:space:]]*"ok"'; then
  echo "Expected database health to be ok in $HEALTH_URL response." >&2
  exit 1
fi

echo "==> Checking root redirect"
ROOT_STATUS="$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:${PORT}/")"
if [[ "$ROOT_STATUS" != "302" && "$ROOT_STATUS" != "200" ]]; then
  echo "Expected HTTP 302 or 200 from /, got $ROOT_STATUS" >&2
  exit 1
fi

echo "==> Verifying Postgres configuration in logs"
if ! grep -q 'GF_DATABASE_TYPE=postgres' "$LOG_FILE"; then
  echo "Expected GF_DATABASE_TYPE=postgres in Grafana logs." >&2
  tail -30 "$LOG_FILE" >&2 || true
  exit 1
fi

echo "Postgres verification succeeded."
