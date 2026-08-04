#!/usr/bin/env bash
set -euo pipefail

# Stable HTTP port in 3001–3999 from worktree path (leaves 3000 for the main checkout).
PORT=$((3001 + $(printf '%s' "$(pwd)" | cksum | awk '{print $1}') % 999))

if [[ -n "${ROOT_WORKTREE_PATH:-}" && -f "$ROOT_WORKTREE_PATH/conf/custom.ini" ]]; then
  cp "$ROOT_WORKTREE_PATH/conf/custom.ini" conf/custom.ini
  grep -vE '^[[:space:]]*;?http_port[[:space:]]*=' conf/custom.ini > conf/custom.ini.tmp || true
  mv conf/custom.ini.tmp conf/custom.ini
fi

if [[ -f conf/custom.ini ]] && grep -q '^\[server\]' conf/custom.ini; then
  awk -v p="$PORT" '
    BEGIN { done = 0 }
    /^\[server\]/ {
      print
      if (!done) { print "http_port = " p; done = 1 }
      next
    }
    { print }
    END {
      if (!done) print "\n[server]\nhttp_port = " p
    }
  ' conf/custom.ini > conf/custom.ini.tmp
  mv conf/custom.ini.tmp conf/custom.ini
else
  printf '[server]\nhttp_port = %s\n' "$PORT" > conf/custom.ini
fi

mkdir -p .cursor
printf '%s\n' "$PORT" > .cursor/worktree-port
echo "Worktree HTTP port: ${PORT} (conf/custom.ini + .cursor/worktree-port)"

corepack enable
corepack install
yarn install --immutable
