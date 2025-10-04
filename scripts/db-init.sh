#!/usr/bin/env bash
set -euo pipefail

: "${DB_ROOT_PASSWORD:=}"

# Safety guard: only run DB init when explicitly enabled. This prevents accidental
# database creation during CI/builds. To force, set ENABLE_DB_INIT=true in environment.
if [ "${ENABLE_DB_INIT:-false}" != "true" ]; then
  echo "[db-init] Skipping DB init because ENABLE_DB_INIT != true"
  exit 0
fi

if [[ -n "${PIPELINE_ENV_FILE:-}" && -f "${PIPELINE_ENV_FILE}" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "${PIPELINE_ENV_FILE}"
  set +a
fi

# 导出所有 Makefile db-init 需要的环境变量
export DB_ROOT_PASSWORD
export MYSQL_HOST
export MYSQL_PORT
export MYSQL_USERNAME
export MYSQL_PASSWORD
export MYSQL_DBNAME

echo "[db-init] Using MYSQL_HOST=${MYSQL_HOST:-mysql}, MYSQL_USERNAME=${MYSQL_USERNAME:-miniblog}, MYSQL_DBNAME=${MYSQL_DBNAME:-miniblog}"

make db-init
