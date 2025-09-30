#!/usr/bin/env bash
set -euo pipefail

: "${DB_ROOT_PASSWORD:=}"

if [[ -n "${PIPELINE_ENV_FILE:-}" && -f "${PIPELINE_ENV_FILE}" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "${PIPELINE_ENV_FILE}"
  set +a
fi

export DB_ROOT_PASSWORD

make db-init
