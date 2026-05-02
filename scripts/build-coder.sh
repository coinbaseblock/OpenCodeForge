#!/usr/bin/env bash
# Build the `opencodeforge-coder` model inside the Ollama container from
# ollama/Modelfile.coder. Pulls the FROM base model first if it's missing.
#
# Usage:
#   scripts/build-coder.sh
#   MODEL_NAME=my-coder MODELFILE=/modelfiles/Modelfile.coder scripts/build-coder.sh
set -euo pipefail

CONTAINER="${OLLAMA_CONTAINER:-opencodeforge-ollama}"
MODEL_NAME="${MODEL_NAME:-opencodeforge-coder}"
MODELFILE="${MODELFILE:-/modelfiles/Modelfile.coder}"

if ! docker inspect "$CONTAINER" >/dev/null 2>&1; then
  echo "error: container '$CONTAINER' not running. Run 'docker compose up -d' first." >&2
  exit 1
fi

base=$(docker exec -t "$CONTAINER" sh -c "grep -E '^FROM ' '$MODELFILE' | awk '{print \$2}'" | tr -d '\r\n ')
if [[ -z "$base" ]]; then
  echo "error: could not parse FROM line in $MODELFILE" >&2
  exit 1
fi

if ! docker exec -t "$CONTAINER" ollama list | grep -Fq "$base"; then
  echo "base model $base missing — pulling first ..."
  docker exec -t "$CONTAINER" ollama pull "$base"
fi

echo "creating $MODEL_NAME from $MODELFILE (base: $base) ..."
docker exec -t "$CONTAINER" ollama create "$MODEL_NAME" -f "$MODELFILE"

echo "done."
docker exec -t "$CONTAINER" ollama list
