#!/usr/bin/env bash
# Same-origin UI -> real Go -> isolated MySQL, with a local Notion HTTP substitute.
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$repo_root"
report_dir="${1:-/tmp/miniblog-admin-ui-real-$(date +%Y%m%d-%H%M%S)}"
: "${MINIBLOG_ADMIN_REAL_DIST:?Build the new admin UI with VITE_API_ROOT=/v1 and set its absolute dist path}"
[[ "$MINIBLOG_ADMIN_REAL_DIST" == /* && -f "$MINIBLOG_ADMIN_REAL_DIST/index.html" ]] || { echo 'Admin dist must be an absolute directory with index.html' >&2; exit 1; }
mkdir -p "$report_dir"
report_dir="$(cd "$report_dir" && pwd)"
container_name="miniblog-admin-ui-serve-$$"
database_name="miniblog_refactor_test_admin_ui_serve_$$"
container_id=""
server_pid=""
cleanup() {
  if [[ -n "$server_pid" ]] && kill -0 "$server_pid" 2>/dev/null; then kill -TERM "$server_pid"; wait "$server_pid" || true; fi
  if [[ -n "$container_id" && "$(docker inspect --format '{{ index .Config.Labels "miniblog.local-acceptance" }}' "$container_id" 2>/dev/null || true)" == "$container_name" ]]; then docker rm -f "$container_id" >/dev/null; fi
}
trap cleanup EXIT INT TERM
endpoint="$(docker context inspect --format '{{.Endpoints.docker.Host}}')"
case "$endpoint" in unix://*) ;; *) echo 'Refusing a non-local Docker endpoint' >&2; exit 1;; esac
go build -o "$report_dir/admin-local-server" ./scripts/admin-ui-integration/server
docker image inspect mysql:8.0.36 >/dev/null
container_id="$(docker run --detach --rm --pull never --name "$container_name" --label "miniblog.local-acceptance=$container_name" --publish 127.0.0.1::3306 --tmpfs /var/lib/mysql:rw --env MYSQL_ALLOW_EMPTY_PASSWORD=yes --env "MYSQL_DATABASE=$database_name" mysql:8.0.36)"
ready=false
for ((attempt=0;attempt<90;attempt++)); do if docker exec "$container_id" mysqladmin --protocol=tcp --host=127.0.0.1 --user=root ping --silent >/dev/null 2>&1; then ready=true; break; fi; sleep 2; done
[[ "$ready" == true ]] || { echo 'Isolated MySQL did not become ready' >&2; exit 1; }
port="$(docker port "$container_id" 3306/tcp | sed -n 's/^127\.0\.0\.1://p')"
[[ "$port" =~ ^[0-9]+$ ]] || { echo 'Missing loopback MySQL port' >&2; exit 1; }
export MINIBLOG_TEST_MYSQL_DSN="root@tcp(127.0.0.1:$port)/$database_name?parseTime=true&multiStatements=true"
unset MINIBLOG_NOTION_TOKEN MINIBLOG_NOTION_BOOTSTRAP_TOKEN MYSQL_DSN MINIBLOG_ROLLOUT_INIT_TEST_DSN
"$report_dir/admin-local-server" --listen "${MINIBLOG_ADMIN_REAL_LISTEN:-127.0.0.1:0}" --ready "$report_dir/ready.json" >"$report_dir/service.log" 2>&1 &
server_pid=$!
for ((attempt=0;attempt<90;attempt++)); do
  if [[ -f "$report_dir/ready.json" ]]; then
    python3 - "$report_dir/ready.json" <<'PY'
import json,sys
r=json.load(open(sys.argv[1]));print('Same-origin admin UI and real Go API ready: '+r['url']);print('Non-secret fixture metadata: '+sys.argv[1])
PY
    break
  fi
  kill -0 "$server_pid" 2>/dev/null || { echo 'Local validation service failed; inspect the local service.log' >&2; exit 1; }
  sleep 1
done
[[ -f "$report_dir/ready.json" ]] || { echo 'Local validation service did not become ready' >&2; exit 1; }
wait "$server_pid"
server_pid=""
