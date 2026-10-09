#!/usr/bin/env bash
# Local-only acceptance. Existing containers, databases and credentials are never reused.
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$repo_root"
report_dir="${1:-/tmp/miniblog-admin-ui-integration-$(date +%Y%m%d-%H%M%S)}"
mkdir -p "$report_dir"
report_dir="$(cd "$report_dir" && pwd)"
container_name="miniblog-admin-ui-test-$$"
database_name="miniblog_refactor_test_admin_ui_$$"
container_id=""
cleanup() {
  if [[ -n "$container_id" ]]; then
    # Remove only the temporary container created by this invocation, with no persistent volume.
    if [[ "$(docker inspect --format '{{ index .Config.Labels "miniblog.local-acceptance" }}' "$container_id" 2>/dev/null || true)" == "$container_name" ]]; then
      docker rm -f "$container_id" >/dev/null
    fi
  fi
}
trap cleanup EXIT INT TERM
endpoint="$(docker context inspect --format '{{.Endpoints.docker.Host}}')"
case "$endpoint" in unix://*) ;; *) echo 'Refusing a non-local Docker endpoint' >&2; exit 1;; esac
docker image inspect mysql:8.0.36 >/dev/null
container_id="$(docker run --detach --rm --pull never --name "$container_name" --label "miniblog.local-acceptance=$container_name" --publish 127.0.0.1::3306 --tmpfs /var/lib/mysql:rw --env MYSQL_ALLOW_EMPTY_PASSWORD=yes --env "MYSQL_DATABASE=$database_name" mysql:8.0.36)"
ready=false
for ((attempt=0;attempt<90;attempt++)); do
  if docker exec "$container_id" mysqladmin --protocol=tcp --host=127.0.0.1 --user=root ping --silent >/dev/null 2>&1; then ready=true; break; fi
  sleep 2
done
if [[ "$ready" != true ]]; then echo 'Isolated MySQL did not become ready' >&2; exit 1; fi
port="$(docker port "$container_id" 3306/tcp | sed -n 's/^127\.0\.0\.1://p')"
if [[ ! "$port" =~ ^[0-9]+$ ]]; then echo 'Missing loopback MySQL port' >&2; exit 1; fi
export MINIBLOG_TEST_MYSQL_DSN="root@tcp(127.0.0.1:$port)/$database_name?parseTime=true&multiStatements=true"
unset MINIBLOG_NOTION_TOKEN MINIBLOG_NOTION_BOOTSTRAP_TOKEN MYSQL_DSN MINIBLOG_ROLLOUT_INIT_TEST_DSN
printf 'Running real Go HTTP/MySQL contracts with an isolated local database.\n'
set +e
go test -race -count=1 -json -timeout=10m ./scripts/content-integration >"$report_dir/go-test.jsonl" 2>"$report_dir/go-test.stderr"
test_status=$?
set -e
python3 - "$report_dir" "$test_status" "$port" "$database_name" <<'PY'
import datetime,json,pathlib,sys
out=pathlib.Path(sys.argv[1]); tests={};packages=[]
for line in (out/'go-test.jsonl').read_text().splitlines():
    try: event=json.loads(line)
    except json.JSONDecodeError: continue
    action=event.get('Action');name=event.get('Test')
    if action in ('pass','fail','skip'):
        if name: tests[name]={'status':action,'duration_seconds':event.get('Elapsed',0)}
        else: packages.append({'package':event.get('Package'),'status':action,'duration_seconds':event.get('Elapsed',0)})
summary={'observed_at':datetime.datetime.now(datetime.timezone.utc).isoformat(),'exit_code':int(sys.argv[2]),'environment':{'database':'isolated MySQL 8.0.36','host':'127.0.0.1','port':int(sys.argv[3]),'schema':sys.argv[4],'persistent_volume':False,'provider':'local Notion HTTP substitute; no real provider access'},'scope':'HTTP test router wired to existing production controllers, Authn/Authz, business services and MySQL store. Production router installation and browser behavior are separate checks.','known_limits':['Authz denial currently does not abort handlers','logout does not revoke issued JWT','myinfo currently returns fixed admin role','anonymous create/change-password route wiring remains a static finding; not exercised here'],'counts':{state:sum(t['status']==state for t in tests.values()) for state in ('pass','fail','skip')},'packages':packages,'tests':tests}
(out/'summary.json').write_text(json.dumps(summary,ensure_ascii=False,indent=2)+'\n')
print(json.dumps({'exit_code':summary['exit_code'],'counts':summary['counts'],'evidence':str(out/'summary.json')},ensure_ascii=False))
PY
exit "$test_status"
