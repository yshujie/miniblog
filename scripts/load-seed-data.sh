#!/bin/bash
# scripts/load-seed-data.sh
# Load seed data into database

set -e

# Source environment variables if available
if [ -f "${PIPELINE_ENV_FILE}" ]; then
  source "${PIPELINE_ENV_FILE}"
fi

# Safety guard: only load seed data when explicitly enabled. Set ENABLE_DB_SEED=true to run.
if [ "${ENABLE_DB_SEED:-false}" != "true" ]; then
    echo "[load-seed-data] Skipping seed load because ENABLE_DB_SEED != true"
    exit 0
fi

# Database connection parameters
DB_HOST="${DB_HOST:-${MYSQL_HOST:-mysql}}"
DB_PORT="${DB_PORT:-${MYSQL_PORT:-3306}}"
DB_USER="${DB_USER:-${MYSQL_USERNAME:-miniblog}}"
DB_PASSWORD="${DB_PASSWORD:-${MYSQL_PASSWORD:-miniblog123}}"
DB_NAME="${DB_NAME:-${MYSQL_DBNAME:-${MYSQL_DATABASE:-miniblog}}}"

SEED_DATA_DIR="$(cd "$(dirname "$0")/../db/migrations/sql" && pwd)"

echo "[load-seed-data] Loading seed data into database: ${DB_NAME}"
echo "[load-seed-data] Using DB_HOST=${DB_HOST}, DB_PORT=${DB_PORT}, DB_USER=${DB_USER}"

# Check if running in Docker or local
if command -v docker >/dev/null 2>&1 && docker ps --format '{{.Names}}' | grep -q "^mysql$"; then
    echo "-> Using docker exec to load data"
    MYSQL_CMD="docker exec -i mysql mysql -h${DB_HOST} -P${DB_PORT} -u${DB_USER} -p${DB_PASSWORD} ${DB_NAME}"
else
    echo "-> Using local mysql client"
    MYSQL_CMD="mysql -h${DB_HOST} -P${DB_PORT} -u${DB_USER} -p${DB_PASSWORD} ${DB_NAME}"
fi

# Load data files in order
for sql_file in user.sql module.sql section.sql article.sql casbin_rule.sql; do
    if [ -f "${SEED_DATA_DIR}/${sql_file}" ]; then
        echo "Loading ${sql_file}..."
        $MYSQL_CMD < "${SEED_DATA_DIR}/${sql_file}"
        echo "✓ ${sql_file} loaded successfully"
    else
        echo "⚠ ${sql_file} not found, skipping..."
    fi
done

echo "✅ All seed data loaded successfully!"
