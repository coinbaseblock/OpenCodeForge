#!/usr/bin/env bash
# Stop OpenCodeForge and remove its containers and volumes.
#
# This is destructive: it deletes Ollama's downloaded models and Open WebUI's
# user data. Workspace files on the host are NOT affected. The script prompts
# for confirmation unless --force is supplied.
set -euo pipefail

force=false
for arg in "$@"; do
  case "$arg" in
    -f|--force) force=true ;;
    *) echo "unknown flag: $arg" >&2; exit 2 ;;
  esac
done

if ! $force; then
  read -r -p "This will delete Ollama models and Open WebUI data volumes. Continue? (yes) " answer
  if [[ "$answer" != "yes" ]]; then
    echo "aborted."
    exit 1
  fi
fi

docker compose down -v
echo "reset complete. Run 'docker compose up -d --build' to start fresh."
