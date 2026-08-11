#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
EXAMPLE="$ROOT_DIR/conf/custom.ini.postgres.example"
TARGET="$ROOT_DIR/conf/custom.ini"

if [[ ! -f "$EXAMPLE" ]]; then
  echo "Missing example config: $EXAMPLE" >&2
  exit 1
fi

if [[ -f "$TARGET" ]]; then
  echo "conf/custom.ini already exists; leaving it unchanged."
  echo "To use Postgres, copy database settings from conf/custom.ini.postgres.example"
  exit 0
fi

cp "$EXAMPLE" "$TARGET"
echo "Created conf/custom.ini from conf/custom.ini.postgres.example"
