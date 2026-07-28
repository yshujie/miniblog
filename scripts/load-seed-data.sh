#!/usr/bin/env bash
set -euo pipefail

if [[ -n "${PIPELINE_ENV_FILE:-}" && -f "${PIPELINE_ENV_FILE}" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "${PIPELINE_ENV_FILE}"
  set +a
fi

if [[ "${ENABLE_DB_SEED:-false}" != "true" ]]; then
  echo "[load-seed-data] Skipping seed load because ENABLE_DB_SEED != true"
  exit 0
fi

db_host="${DB_HOST:-${MYSQL_HOST:-mysql}}"
db_port="${DB_PORT:-${MYSQL_PORT:-3306}}"
db_user="${DB_USER:-${MYSQL_USERNAME:-miniblog}}"
db_password="${DB_PASSWORD:-${MYSQL_PASSWORD:-miniblog123}}"
db_name="${DB_NAME:-${MYSQL_DBNAME:-${MYSQL_DATABASE:-miniblog}}}"
docker_network="${DOCKER_NETWORK:-infra-network}"
seed_data_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../db/migrations/sql" && pwd)"
mysql_args=(-h "${db_host}" -P "${db_port}" -u "${db_user}" "${db_name}")

echo "[load-seed-data] Loading seed data into database: ${db_name}"
echo "[load-seed-data] Using DB_HOST=${db_host}, DB_PORT=${db_port}, DB_USER=${db_user}"

if command -v mysql >/dev/null 2>&1; then
  echo "-> Using local mysql client"
  run_mysql() {
    MYSQL_PWD="${db_password}" mysql "${mysql_args[@]}"
  }
else
  if ! command -v docker >/dev/null 2>&1; then
    echo "❌ mysql client and docker are both unavailable" >&2
    exit 1
  fi

  echo "-> Local mysql client not found, using mysql:8.0 on ${docker_network}"
  run_mysql() {
    MYSQL_PWD="${db_password}" docker run --rm -i --network "${docker_network}" \
      -e MYSQL_PWD mysql:8.0 \
      mysql "${mysql_args[@]}"
  }
fi

for sql_file in user.sql module.sql section.sql article.sql casbin_rule.sql; do
  sql_path="${seed_data_dir}/${sql_file}"
  if [[ -f "${sql_path}" ]]; then
    echo "Loading ${sql_file}..."
    run_mysql < "${sql_path}"
    echo "✓ ${sql_file} loaded successfully"
  else
    echo "⚠ ${sql_file} not found, skipping..."
  fi
done

echo "✅ All seed data loaded successfully!"
