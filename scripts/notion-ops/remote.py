#!/usr/bin/env python3
"""Server-only, fixed read operations. No credential or report contents on stdout."""
import argparse
import contextlib
import fcntl
import json
import os
from pathlib import Path
import re
import secrets
import signal
import stat
import subprocess
import tempfile
import time

READ_MODES = ("schema_check", "dry_run", "bootstrap_preview")
FEATURE_KEYS = (
    "MINIBLOG_NOTION_TOKEN", "MINIBLOG_NOTION_SYNC_ENABLED",
    "MINIBLOG_NOTION_SYNC_INTERVAL", "MINIBLOG_NOTION_SYNC_AUTHOR",
    "MINIBLOG_CONTENT_REGISTER_ENABLED",
)
SOURCE_IDS = {
    "2bf330bd-ddf1-80a6-aa49-000bbd1e154b", "d579f619-4f1c-4e7e-9593-ab6529fc63d0",
    "cfa962f3-12be-40aa-95eb-9122186bd4ba", "03d24108-5172-4e8f-8cfd-e3bd52e437af",
    "d13097e3-b78a-4b6e-8848-979749d4b2f3",
}
DB_KEYS = {
    "MYSQL_HOST": "MINIBLOG_DATABASE_HOST", "MYSQL_PORT": "MINIBLOG_DATABASE_PORT",
    "MYSQL_DATABASE": "MINIBLOG_DATABASE_DBNAME", "MYSQL_USERNAME": "MINIBLOG_DATABASE_USERNAME",
    "MYSQL_PASSWORD": "MINIBLOG_DATABASE_PASSWORD",
}

class SafeError(Exception):
    """Only constant, operator-safe descriptions are raised."""


def private_file(path, allow_missing=False):
    path = Path(path)
    try:
        value = path.lstat()
    except FileNotFoundError:
        if allow_missing:
            return
        raise SafeError("required private file is missing") from None
    if not stat.S_ISREG(value.st_mode) or value.st_uid != os.getuid() or value.st_mode & 0o077:
        raise SafeError("file must be owned by the operator, regular and private")


def token_from(path):
    private_file(path)
    value = Path(path).read_text(encoding="utf-8")
    # Both current ntn_ and legacy secret_ tokens fit this single-line alphabet.
    # A strict alphabet also prevents Compose interpolation and dotenv injection.
    if value and not re.fullmatch(r"[A-Za-z0-9_-]{10,4096}", value):
        raise SafeError("invalid read credential format")
    return value


def atomic_private(path, value):
    path = Path(path)
    private_file(path, allow_missing=True)
    fd, temporary = tempfile.mkstemp(prefix=".notion-ops-", dir=path.parent)
    try:
        os.fchmod(fd, 0o600)
        with os.fdopen(fd, "w", encoding="utf-8") as output:
            output.write(value)
            output.flush()
            os.fsync(output.fileno())
        os.replace(temporary, path)
        directory = os.open(path.parent, os.O_RDONLY | os.O_DIRECTORY)
        try:
            os.fsync(directory)
        finally:
            os.close(directory)
    finally:
        if os.path.exists(temporary):
            os.unlink(temporary)


def env_key(line):
    match = re.match(r"^\s*(?:export\s+)?([A-Z][A-Z0-9_]*)\s*=", line)
    return match.group(1) if match else ""


def runtime_env(app_dir, candidate, credential):
    """Called inside the deployment flock; keep existing feature settings verbatim."""
    target = Path(app_dir) / ".env"
    private_file(target, allow_missing=True)
    private_file(candidate)
    new_token = token_from(credential)
    current = target.read_text(encoding="utf-8") if target.exists() else ""
    base = Path(candidate).read_text(encoding="utf-8")
    base_lines = [line for line in base.splitlines() if env_key(line) not in FEATURE_KEYS
                  and env_key(line) != "MINIBLOG_NOTION_BOOTSTRAP_TOKEN"]
    features = {}
    for line in current.splitlines():
        if env_key(line) in FEATURE_KEYS:
            features[env_key(line)] = line
    if new_token:
        features["MINIBLOG_NOTION_TOKEN"] = "MINIBLOG_NOTION_TOKEN=" + new_token
    # A backup is private too; never perpetuate an accidentally present writer token.
    if target.exists():
        backup = "\n".join(line for line in current.splitlines()
                           if env_key(line) != "MINIBLOG_NOTION_BOOTSTRAP_TOKEN") + "\n"
        atomic_private(Path(app_dir) / ".env.previous", backup)
    content = "\n".join(base_lines + [features[k] for k in FEATURE_KEYS if k in features]) + "\n"
    atomic_private(target, content)
    print("Runtime environment updated; feature switches preserved.")


@contextlib.contextmanager
def operation_lock(app_dir):
    path = Path(app_dir) / ".operations.lock"
    descriptor = os.open(path, os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600)
    try:
        if os.fstat(descriptor).st_uid != os.getuid():
            raise SafeError("operation lock owner is invalid")
        os.fchmod(descriptor, 0o600)
        deadline = time.monotonic() + 900
        while True:
            try:
                fcntl.flock(descriptor, fcntl.LOCK_EX | fcntl.LOCK_NB)
                break
            except BlockingIOError:
                if time.monotonic() >= deadline:
                    raise SafeError("another deployment or maintenance operation is active") from None
                time.sleep(1)
        yield
    finally:
        os.close(descriptor)


def docker_json(args):
    result = subprocess.run(["docker", *args], stdout=subprocess.PIPE, stderr=subprocess.DEVNULL,
                            text=True, timeout=30, check=False)
    if result.returncode:
        raise SafeError("running backend inspection failed")
    try:
        return json.loads(result.stdout)
    except (ValueError, TypeError):
        raise SafeError("running backend inspection returned invalid metadata") from None


def running_image():
    value = docker_json(["inspect", "--format", '{"image":{{json .Image}},"running":{{json .State.Running}}}',
                         "miniblog-backend"])
    if value.get("running") is not True or not re.fullmatch(r"sha256:[0-9a-f]{64}", value.get("image", "")):
        raise SafeError("backend must be running with an immutable local image")
    return value["image"]


def database_environment(require_scheduler_off=True):
    values = docker_json(["inspect", "--format", "{{json .Config.Env}}", "miniblog-backend"])
    if not isinstance(values, list):
        raise SafeError("backend environment metadata is invalid")
    current = {}
    for value in values:
        if isinstance(value, str) and "=" in value:
            key, entry = value.split("=", 1)
            current[key] = entry
    if require_scheduler_off and current.get("MINIBLOG_NOTION_SYNC_ENABLED", "").lower() not in ("false", "0"):
        raise SafeError("disable the runtime scheduler before rollout auditing")
    database = {}
    for key, fallback in DB_KEYS.items():
        if key in current and fallback in current and current[key] != current[fallback]:
            raise SafeError("backend database environment aliases disagree")
        # Viper consumes MINIBLOG_DATABASE_*; prefer the actual service mapping.
        value = current[fallback] if fallback in current else current.get(key)
        if value is None or any(character in value for character in "\r\n\x00"):
            raise SafeError("backend database environment is incomplete or unsupported")
        if key != "MYSQL_PASSWORD" and not value:
            raise SafeError("backend database environment is incomplete or unsupported")
        database[key] = value
    return database


def sanitized_task_output(raw, env_file):
    if isinstance(raw, bytes):
        raw = raw.decode("utf-8", errors="replace")
    if not raw:
        return "No task diagnostics captured.\n"
    # Never write unsanitized diagnostics, including TimeoutExpired.output.
    private_file(env_file)
    secret_values = []
    for line in Path(env_file).read_text(encoding="utf-8").splitlines():
        if "=" not in line:
            continue
        key, value = line.split("=", 1)
        if key in ("MINIBLOG_NOTION_TOKEN", "MINIBLOG_NOTION_BOOTSTRAP_TOKEN", "MYSQL_PASSWORD") and value:
            secret_values.append(value)
    for value in sorted(secret_values, key=len, reverse=True):
        raw = raw.replace(value, "[REDACTED]")
    return raw


def docker_task(image, directory, env_file, binary, args, label, network=False):
    log = directory / (label + ".log")
    name = "miniblog-notion-ops-" + directory.name + "-" + label
    owner = secrets.token_hex(16)
    # Durable, private ownership receipt lets the cancellation step clean only
    # this run when SSH dies before the local finally block completes.
    receipt = directory / ("container-" + label + ".json")
    atomic_private(receipt, json.dumps({"name": name, "owner": owner}) + "\n")
    # Never reuse or remove a pre-existing task, even with the same run identifier.
    exists = subprocess.run(["docker", "container", "inspect", name],
                            stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=30, check=False)
    if exists.returncode == 0:
        raise SafeError("a container already exists for this maintenance task")
    fd = os.open(log, os.O_CREAT | os.O_EXCL | os.O_WRONLY | os.O_NOFOLLOW, 0o600)
    command = ["docker", "run", "--rm", "--log-driver", "none", "--name", name,
               "--label", "miniblog.notion-ops.owner=" + owner,
               "--pull=never", "--user", f"{os.getuid()}:{os.getgid()}",
               "--env-file", str(env_file), "--volume", str(directory) + ":/ops",
               "--entrypoint", binary]
    if network:
        command += ["--network", "infra-network"]
    command += [image, *args]
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as output:
            diagnostics = b""
            try:
                result = subprocess.run(command, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=720, check=False)
                diagnostics = result.stdout
            except subprocess.TimeoutExpired as error:
                diagnostics = error.output or b""
                raise SafeError("read task timed out; inspect restricted server logs") from None
            finally:
                # CLI --report sends metadata to JSON; the short diagnostic output
                # stays in memory until credential redaction has completed.
                output.write(sanitized_task_output(diagnostics, env_file))
                output.flush()
                os.fsync(output.fileno())
        if result.returncode:
            raise SafeError("read task failed; inspect restricted server reports and logs")
    finally:
        # Killing Docker's client does not stop a daemon container. Cancel only
        # the container created by this invocation, guarded by its random label.
        owned = subprocess.run(["docker", "container", "inspect", "--format",
                                '{{index .Config.Labels "miniblog.notion-ops.owner"}}', name],
                               stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, timeout=30, check=False)
        if owned.returncode == 0 and owned.stdout.decode("utf-8").strip() == owner:
            stopped = subprocess.run(["docker", "rm", "-f", name], stdout=subprocess.DEVNULL,
                                     stderr=subprocess.DEVNULL, timeout=30, check=False)
            if stopped.returncode:
                raise SafeError("maintenance container cleanup failed")


def report_json(path):
    private_file(path)
    try:
        return json.loads(Path(path).read_text(encoding="utf-8"))
    except ValueError:
        raise SafeError("read task report is invalid") from None


def validate_schema(report):
    items = report.get("sources", [])
    if report.get("complete") is not True or len(items) != 5 or {s.get("source_id") for s in items} != SOURCE_IDS:
        raise SafeError("five-source schema check is incomplete")
    if any(s.get("status") != "valid" or s.get("actual_data_source_id") != s.get("source_id") for s in items):
        raise SafeError("five-source schema identities or fields do not match")


def validate_status(report, mode):
    if report.get("paused") is not True or report.get("current_run_id"):
        raise SafeError("pause synchronization and wait for the current run before auditing")
    sources = report.get("sources", [])
    if len(sources) != 5 or {s.get("source_id") for s in sources} != SOURCE_IDS:
        raise SafeError("five-source configuration is incomplete")
    if any(not s.get("module_code") for s in sources):
        raise SafeError("configure all five source modules before baseline scanning")
    if report.get("baseline_frozen") is not True:
        if mode == "bootstrap_preview":
            raise SafeError("complete the initial five-source dry run before historical preview")
        if any(s.get("enabled") is not False for s in sources):
            raise SafeError("disable all five sources before the initial baseline scan")
    # Read modes never change source_writes_paused; manual registration stays available.


def read_only(app_dir, credential, mode, run_id):
    if mode not in READ_MODES or not re.fullmatch(r"[0-9]+-[0-9]+", run_id):
        raise SafeError("unsupported maintenance mode or run identifier")
    token = token_from(credential)
    if not token:
        raise SafeError("configure the dedicated read credential before running maintenance")
    root = Path(app_dir) / "ops" / "notion"
    # Reject symlink directories before making private report storage.
    for directory in (Path(app_dir) / "ops", root, root / run_id):
        if directory.is_symlink():
            raise SafeError("report directory must not be a symbolic link")
        if directory.exists():
            if not directory.is_dir() or directory.stat().st_uid != os.getuid():
                raise SafeError("report directory owner or type is invalid")
        else:
            directory.mkdir(mode=0o700)
        os.chmod(directory, 0o700)
    directory = root / run_id
    if any(directory.iterdir()):
        raise SafeError("report directory already contains a maintenance result")
    with operation_lock(app_dir):
        image = running_image()
        atomic_private(directory / "operation.json", json.dumps({"mode": mode, "image_id": image, "run_id": run_id}) + "\n")
        env_file = directory / "task.env"
        try:
            atomic_private(env_file, "MINIBLOG_NOTION_TOKEN=" + token + "\n")
            schema_args = ["--mode", "schema_check", "--report", "/ops/schema.json", "--timeout", "10m"]
            docker_task(image, directory, env_file, "/app/notion-sync", schema_args, "schema")
            schema = report_json(directory / "schema.json")
            validate_schema(schema)
            if mode != "schema_check":
                database = database_environment()
                atomic_private(env_file, "\n".join(k + "=" + v for k, v in database.items()) +
                               "\nMINIBLOG_NOTION_TOKEN=" + token + "\n")
                docker_task(image, directory, env_file, "/app/content-preflight", [], "preflight", network=True)
                docker_task(image, directory, env_file, "/app/notion-sync",
                            ["--mode", "status", "--report", "/ops/status-before.json"], "status-before", network=True)
                validate_status(report_json(directory / "status-before.json"), mode)
                docker_task(image, directory, env_file, "/app/notion-sync",
                            ["--mode", mode, "--report", "/ops/result.json", "--timeout", "10m"], "result", network=True)
                docker_task(image, directory, env_file, "/app/notion-sync",
                            ["--mode", "status", "--report", "/ops/status-after.json"], "status-after", network=True)
                after = report_json(directory / "status-after.json")
                if after.get("baseline_frozen") is not True:
                    raise SafeError("scan did not produce a complete frozen baseline")
                # No report fields, error details, page text, links or credentials reach CI.
            print("Read maintenance completed: " + mode)
            print("Restricted server reports: " + str(directory))
            print("Five-source schema: complete; immutable image verified.")
        finally:
            if env_file.exists():
                env_file.unlink()


class SafeParser(argparse.ArgumentParser):
    def error(self, _message):
        raise SafeError("operation arguments are invalid")


def main():
    parser = SafeParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    runtime = sub.add_parser("runtime-env")
    runtime.add_argument("--app-dir", required=True)
    runtime.add_argument("--candidate", required=True)
    runtime.add_argument("--credential", required=True)
    read = sub.add_parser("read-only")
    read.add_argument("--app-dir", required=True)
    read.add_argument("--credential", required=True)
    read.add_argument("--mode", choices=READ_MODES, required=True)
    read.add_argument("--run-id", required=True)
    try:
        args = parser.parse_args()
        def cancelled(_signum, _frame):
            raise SafeError("maintenance task cancelled")
        for signum in (signal.SIGTERM, signal.SIGINT, signal.SIGHUP):
            signal.signal(signum, cancelled)
        os.umask(0o077)
        if args.command == "runtime-env":
            runtime_env(args.app_dir, args.candidate, args.credential)
        else:
            read_only(args.app_dir, args.credential, args.mode, args.run_id)
    except SafeError as error:
        print("Notion maintenance blocked: " + str(error), file=__import__("sys").stderr)
        return 1
    except Exception:
        # OS/process exceptions may contain values or command output; never echo them.
        print("Notion maintenance failed; inspect restricted server files.", file=__import__("sys").stderr)
        return 1
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
