#!/usr/bin/env bash
# Probe every OpenCodeForge service and print a one-line status for each.
set -u

OLLAMA_PORT="${OLLAMA_PORT:-11434}"
WEBUI_PORT="${WEBUI_PORT:-3000}"
TOOLS_PORT="${TOOLS_PORT:-8088}"

probe() {
  local name="$1" url="$2"
  local code
  code=$(curl -sS -o /dev/null -w '%{http_code}' --max-time 5 "$url" || echo 000)
  if [[ "$code" =~ ^[23] ]]; then
    printf '%-12s ok (%s)\n' "$name" "$url"
  else
    printf '%-12s down (%s, http=%s)\n' "$name" "$url" "$code"
  fi
}

probe 'ollama'     "http://localhost:${OLLAMA_PORT}/api/tags"
probe 'open-webui' "http://localhost:${WEBUI_PORT}/"
probe 'tools-api'  "http://localhost:${TOOLS_PORT}/health"
