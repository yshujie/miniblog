#!/usr/bin/env bash
set -euo pipefail

files=$(echo "${DEPLOY_COMPOSE_FILES:-docker-compose.yml}" | xargs)
if [[ -z "$files" ]]; then
  files="docker-compose.yml"
fi

pull_flag=${PULL_IMAGES:-false}

echo "Deploying with compose files: ${files}"

FILES="$files" PULL="$pull_flag" make compose-up
