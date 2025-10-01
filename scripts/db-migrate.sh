#!/usr/bin/env bash
set -euo pipefail

# Resolve DB connection parameters with sensible defaults.
if [[ -n "${PIPELINE_ENV_FILE:-}" && -f "${PIPELINE_ENV_FILE}" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "${PIPELINE_ENV_FILE}"
  set +a
fi

db_host="${DB_HOST:-${MYSQL_HOST:-infra-mysql}}"
db_port="${DB_PORT:-${MYSQL_PORT:-3306}}"
db_user="${DB_USER:-${MYSQL_USERNAME:-miniblog}}"
db_password="${DB_PASSWORD:-${MYSQL_PASSWORD:-miniblog123}}"
db_name="${DB_NAME:-${MYSQL_DATABASE:-miniblog}}"

# Debug: print resolved DB connection params (do not print password)
echo "[db-migrate] Resolved DB_HOST=${db_host}, DB_PORT=${db_port}, DB_USER=${db_user}, DB_NAME=${db_name}" >&2

DB_HOST="$db_host" \
DB_PORT="$db_port" \
DB_USER="$db_user" \
DB_PASSWORD="$db_password" \
DB_NAME="$db_name" \
make db-migrate
