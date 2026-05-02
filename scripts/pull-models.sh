#!/usr/bin/env bash
# Pull recommended OpenCodeForge coding models into the Ollama container.
#
# Usage:
#   scripts/pull-models.sh                # pulls the default (qwen2.5-coder:14b)
#   scripts/pull-models.sh ultralight     # 1.5b
#   scripts/pull-models.sh fast           # 3b
#   scripts/pull-models.sh light          # 7b
#   scripts/pull-models.sh heavy          # 32b
#   scripts/pull-models.sh deepseek       # deepseek-coder-v2:lite
#   scripts/pull-models.sh golang         # optimized set for Go
#   scripts/pull-models.sh python         # optimized set for Python
#   scripts/pull-models.sh all            # everything
set -euo pipefail

CONTAINER="${OLLAMA_CONTAINER:-opencodeforge-ollama}"

require_container() {
  if ! docker inspect "$CONTAINER" >/dev/null 2>&1; then
    echo "error: container '$CONTAINER' not running. Run 'docker compose up -d' first." >&2
    exit 1
  fi
}

pull() {
  if docker exec -t "$CONTAINER" ollama list | grep -Fq "$1"; then
    echo "skip $1 (already installed)"
    return
  fi
  echo "pulling $1 ..."
  docker exec -t "$CONTAINER" ollama pull "$1"
}

require_container

profile="${1:-default}"
case "$profile" in
  ultralight)
    pull "qwen2.5-coder:1.5b"
    ;;
  fast)
    pull "qwen2.5-coder:3b"
    ;;
  light)
    pull "qwen2.5-coder:7b"
    ;;
  default)
    pull "qwen2.5-coder:14b"
    ;;
  heavy)
    pull "qwen2.5-coder:32b"
    ;;
  deepseek)
    pull "deepseek-coder-v2:lite"
    ;;
  golang)
    pull "qwen2.5-coder:3b"
    pull "deepseek-coder-v2:lite"
    ;;
  python)
    pull "qwen2.5-coder:7b"
    pull "deepseek-coder-v2:lite"
    ;;
  all)
    pull "qwen2.5-coder:1.5b"
    pull "qwen2.5-coder:3b"
    pull "qwen2.5-coder:7b"
    pull "qwen2.5-coder:14b"
    pull "qwen2.5-coder:32b"
    pull "deepseek-coder-v2:lite"
    ;;
  *)
    echo "unknown profile: $profile" >&2
    echo "valid: ultralight | fast | light | default | heavy | deepseek | golang | python | all" >&2
    exit 2
    ;;
esac

echo "done."
docker exec -t "$CONTAINER" ollama list
