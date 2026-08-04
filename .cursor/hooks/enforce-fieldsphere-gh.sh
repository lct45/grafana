#!/usr/bin/env bash
set -euo pipefail

INPUT_JSON="$(cat)"
COMMAND="$(python3 -c 'import json,sys; print(json.load(sys.stdin).get("command",""))' <<< "$INPUT_JSON" 2>/dev/null || true)"
COMMAND_LC="$(printf '%s' "$COMMAND" | tr '[:upper:]' '[:lower:]')"

# Only enforce policy for gh commands. Other commands are always allowed.
if [[ ! "$COMMAND_LC" =~ (^|[[:space:]])gh([[:space:]]|$) ]]; then
  printf '%s\n' '{"continue":true,"permission":"allow"}'
  exit 0
fi

is_mutating="false"
if [[ "$COMMAND_LC" =~ (^|[[:space:]])(create|edit|merge|close|reopen|delete|transfer|lock|unlock|pin|unpin|upload|archive|unarchive|review|ready)([[:space:]]|$) ]]; then
  is_mutating="true"
fi

targets_upstream="false"
if [[ "$COMMAND_LC" =~ (--repo|-r)[[:space:]]*grafana/grafana ]] || [[ "$COMMAND_LC" =~ github\.com/grafana/grafana ]] || [[ "$COMMAND_LC" =~ (^|[[:space:]])grafana/grafana([[:space:]]|$) ]]; then
  targets_upstream="true"
fi

targets_fork_explicit="false"
if [[ "$COMMAND_LC" =~ (--repo|-r)[[:space:]]*lct45/grafana ]] || [[ "$COMMAND_LC" =~ github\.com/lct45/grafana ]]; then
  targets_fork_explicit="true"
fi

if [[ "$is_mutating" == "true" && "$targets_upstream" == "true" ]]; then
  cat <<'EOF'
{"continue":true,"permission":"deny","user_message":"Blocked: write actions to grafana/grafana are not allowed from this repo. Target lct45/grafana instead.","agent_message":"This gh command appears to perform a write operation against grafana/grafana. Use --repo lct45/grafana for write actions."}
EOF
  exit 0
fi

if [[ "$is_mutating" == "true" && "$targets_fork_explicit" != "true" ]]; then
  cat <<'EOF'
{"continue":true,"permission":"deny","user_message":"Blocked: mutating gh commands must include --repo lct45/grafana.","agent_message":"For safety, mutating gh commands must explicitly target the fork via --repo lct45/grafana."}
EOF
  exit 0
fi

printf '%s\n' '{"continue":true,"permission":"allow"}'
