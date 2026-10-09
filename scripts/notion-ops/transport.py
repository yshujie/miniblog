#!/usr/bin/env python3
"""Actions runner: private-file credential transport, never secret-valued SSH arguments."""
import argparse
import base64
import hashlib
import os
from pathlib import Path
import re
import shutil
import shlex
import signal
import subprocess
import tempfile
import sys

MODES = ("deploy_token", "cleanup_deploy", "cleanup_ops", "schema_check", "dry_run", "bootstrap_preview", "rollout", "cleanup_rollout")

class SafeError(Exception):
    pass


def private_write(path, value):
    descriptor = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    with os.fdopen(descriptor, "w", encoding="utf-8") as output:
        output.write(value)


def write_validated_key(path, value):
    # OpenSSH rejects otherwise valid private keys without the final LF.
    # Normalize pasted Windows line endings without altering key body contents.
    normalized = value.replace("\r\n", "\n").replace("\r", "\n").rstrip("\n") + "\n"
    private_write(path, normalized)
    try:
        parsed = subprocess.run(["ssh-keygen", "-y", "-P", "", "-f", str(path)],
                                stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL,
                                stderr=subprocess.PIPE, timeout=20, check=False)
    except (OSError, subprocess.TimeoutExpired):
        raise SafeError("local SSH key validation could not complete") from None
    if parsed.returncode:
        message = (parsed.stderr or b"").decode("utf-8", errors="replace").lower()
        if "passphrase" in message or "encrypted" in message:
            raise SafeError("SSH private key requires an unsupported passphrase")
        raise SafeError("SSH private key format is invalid")


def ssh_error_category(stderr, returncode, fallback):
    # Classify in memory, never return an excerpt, host, path or credential.
    message = (stderr or b"").decode("utf-8", errors="replace").lower()
    if any(value in message for value in ("load key", "invalid format", "error in libcrypto", "unprotected private key", "bad permissions")):
        return "SSH private key could not be loaded"
    if any(value in message for value in ("host key verification failed", "remote host identification has changed", "host key has changed")):
        return "SSH host verification failed"
    if any(value in message for value in ("could not resolve hostname", "name or service not known", "temporary failure in name resolution", "nodename nor servname")):
        return "SSH destination could not be resolved"
    if any(value in message for value in ("connection timed out", "operation timed out", "connection refused", "network is unreachable", "no route to host", "connection reset", "connection closed", "broken pipe")):
        # An authentication error may also be followed by 'Connection closed'.
        if "permission denied" not in message and "authentication failed" not in message:
            return "SSH connection failed"
    if returncode == 255 and any(value in message for value in ("permission denied", "authentication failed", "no more authentication methods")):
        return "SSH authentication was rejected"
    return fallback


STAGE_CREATE_CODE = r"""import os, sys
stage, run = sys.argv[1:]
os.umask(0o077)
os.mkdir(stage, 0o700)
fd = os.open(stage + '/.owner', os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
with os.fdopen(fd, 'w', encoding='ascii') as output:
    output.write(run + '\n')
"""
STAGE_CLEANUP_CODE = r"""import os, shutil, stat, sys
stage, run = sys.argv[1:]
try:
    info = os.lstat(stage)
except FileNotFoundError:
    sys.exit(0)
if not stat.S_ISDIR(info.st_mode) or info.st_uid != os.getuid():
    sys.exit(1)
try:
    fd = os.open(stage + '/.owner', os.O_RDONLY | os.O_NOFOLLOW)
except OSError:
    sys.exit(1)
with os.fdopen(fd, 'rb') as marker:
    info = os.fstat(marker.fileno())
    if not stat.S_ISREG(info.st_mode) or info.st_uid != os.getuid():
        sys.exit(1)
    expected_marker = (run + '\n').encode('ascii')
    if marker.read(len(expected_marker) + 1) != expected_marker:
        sys.exit(1)
shutil.rmtree(stage)
"""


def stage_command(operation, stage, run):
    code = STAGE_CREATE_CODE if operation == "create" else STAGE_CLEANUP_CODE
    return "python3 -c " + shlex.quote(code) + " " + shlex.quote(stage) + " " + shlex.quote(run)


ROLLOUT_CLEANUP_CODE = r"""import json, os, pathlib, re, stat, subprocess, sys
run = sys.argv[1]
if not re.fullmatch(r"[0-9]+-[0-9]+", run):
    sys.exit(1)
root = pathlib.Path('/opt/miniblog/ops/notion/rollout-' + run)
try:
    info = root.lstat()
except FileNotFoundError:
    sys.exit(0)
if not stat.S_ISDIR(info.st_mode) or info.st_uid != os.getuid() or info.st_mode & 0o077:
    sys.exit(1)
for parent in (root.parent.parent, root.parent):
    info = parent.lstat()
    if not stat.S_ISDIR(info.st_mode) or info.st_uid != os.getuid() or info.st_mode & 0o077:
        sys.exit(1)
def read(path):
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW)
    with os.fdopen(fd) as stream:
        info = os.fstat(stream.fileno())
        if not stat.S_ISREG(info.st_mode) or info.st_uid != os.getuid() or info.st_mode & 0o077:
            raise ValueError()
        return json.load(stream)
if read(root / 'operation.json').get('run_id') != run:
    sys.exit(1)
for path in root.glob('container-*.json'):
    row = read(path)
    name, owner = row.get('name', ''), row.get('owner', '')
    if not name.startswith('miniblog-notion-ops-rollout-' + run + '-') or not re.fullmatch(r'[0-9a-f]{32}', owner):
        sys.exit(1)
    result = subprocess.run(['docker', 'container', 'inspect', '--format', '{{index .Config.Labels "miniblog.notion-ops.owner"}}', name], stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, text=True, timeout=30)
    if result.returncode == 0:
        if result.stdout.strip() != owner:
            sys.exit(1)
        if subprocess.run(['docker', 'rm', '-f', name], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=30).returncode:
            sys.exit(1)
for path in root.glob('*.env'):
    info = path.lstat()
    if not stat.S_ISREG(info.st_mode) or info.st_uid != os.getuid() or info.st_mode & 0o077:
        sys.exit(1)
    path.unlink()
"""

def rollout_cleanup_command(run):
    return "python3 -c " + shlex.quote(ROLLOUT_CLEANUP_CODE) + " " + shlex.quote(run)


class SafeParser(argparse.ArgumentParser):
    def error(self, _message):
        raise SafeError("operation arguments are invalid")


def main():
    parser = SafeParser(description=__doc__)
    parser.add_argument("--mode", choices=MODES, required=True)
    try:
        args = parser.parse_args()
        def cancelled(_signum, _frame):
            raise SafeError("operation cancelled")
        for signum in (signal.SIGTERM, signal.SIGINT, signal.SIGHUP):
            signal.signal(signum, cancelled)
        if os.environ.get("GITHUB_REF") != "refs/heads/main":
            raise SafeError("server operations require the main branch")
        host = os.environ.get("SVRD_HOST", "")
        user = os.environ.get("SVRD_USER", "")
        port = os.environ.get("SVRD_PORT", "22")
        run = os.environ.get("GITHUB_RUN_ID", "") + "-" + os.environ.get("GITHUB_RUN_ATTEMPT", "")
        if (not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9.-]{0,252}", host)
                or not re.fullmatch(r"[A-Za-z_][A-Za-z0-9_-]{0,63}", user)
                or not port.isdigit() or not 1 <= int(port) <= 65535
                or not re.fullmatch(r"[0-9]+-[0-9]+", run)):
            raise SafeError("SSH destination or run identifier is invalid")
        rollout_action = os.environ.get("NOTION_ROLLOUT_ACTION", "")
        review_id = os.environ.get("NOTION_MANIFEST_ID", "")
        review_hash = os.environ.get("NOTION_MANIFEST_SHA256", "")
        image_sha = os.environ.get("NOTION_EXPECTED_IMAGE_SHA", "")
        revision = os.environ.get("NOTION_EXPECTED_CONFIG_REVISION", "")
        rollout_actions = {"pause", "resume", "catalog_prepare", "catalog_bind", "catalog_activate", "source_update", "bootstrap_preview", "bootstrap_apply", "sync", "scheduler_on", "scheduler_off"}
        if args.mode == "rollout" and (rollout_action not in rollout_actions
                or not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9_-]{0,79}", review_id)
                or not re.fullmatch(r"[0-9a-f]{64}", review_hash)
                or not re.fullmatch(r"[0-9a-f]{40}", image_sha)
                or not re.fullmatch(r"0|[1-9][0-9]{0,19}", revision)):
            raise SafeError("reviewed operation inputs are invalid")
        token = os.environ.get("MINIBLOG_NOTION_TOKEN", "")
        writer = os.environ.get("MINIBLOG_NOTION_BOOTSTRAP_TOKEN", "")
        if args.mode == "rollout":
            if rollout_action in {"catalog_prepare", "bootstrap_preview", "bootstrap_apply", "sync", "scheduler_on"} and not token:
                raise SafeError("dedicated read credential is not configured")
            if (rollout_action == "bootstrap_apply") != bool(writer):
                raise SafeError("writer credential is restricted to bootstrap apply")
            if writer and not re.fullmatch(r"[A-Za-z0-9_-]{10,4096}", writer):
                raise SafeError("invalid writer credential format")
        if args.mode not in ("cleanup_deploy", "cleanup_ops", "cleanup_rollout") and token and not re.fullmatch(r"[A-Za-z0-9_-]{10,4096}", token):
            raise SafeError("invalid read credential format")
        if args.mode in ("schema_check", "dry_run", "bootstrap_preview") and not token:
            raise SafeError("dedicated read credential is not configured")
        key = os.environ.get("SVRD_SSH_KEY", "")
        if not key:
            raise SafeError("SSH credential is not configured")
        os.umask(0o077)
        with tempfile.TemporaryDirectory(prefix="miniblog-notion-", dir=os.environ.get("RUNNER_TEMP")) as temporary:
            temp = Path(temporary)
            write_validated_key(temp / "key", key)
            private_write(temp / "known_hosts", "")
            destination = user + "@" + host
            # Current deployment actions do not require a fingerprint. An optional
            # SHA256 fingerprint upgrades this TOFU contract without a new required secret.
            fingerprint = os.environ.get("SVRD_HOST_FINGERPRINT", "")
            if fingerprint:
                if not re.fullmatch(r"SHA256:[A-Za-z0-9+/]{43}", fingerprint):
                    raise SafeError("SSH fingerprint format is invalid")
                scan = subprocess.run(["ssh-keyscan", "-T", "10", "-p", port, host],
                                      stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, timeout=30, check=False)
                matched = []
                for line in scan.stdout.decode("utf-8").splitlines():
                    parts = line.split()
                    if len(parts) < 3 or line.startswith("#"):
                        continue
                    digest = "SHA256:" + base64.b64encode(hashlib.sha256(base64.b64decode(parts[2])).digest()).decode().rstrip("=")
                    if digest == fingerprint:
                        matched.append(line)
                if not matched:
                    raise SafeError("SSH host fingerprint did not match")
                (temp / "known_hosts").write_text("\n".join(matched) + "\n", encoding="utf-8")
            common = ["-i", str(temp / "key"), "-o", "BatchMode=yes", "-o", "IdentitiesOnly=yes",
                      "-o", "ConnectTimeout=15", "-o", "LogLevel=ERROR", "-o", "StrictHostKeyChecking=" + ("yes" if fingerprint else "accept-new"),
                      "-o", "UserKnownHostsFile=" + str(temp / "known_hosts")]
            stage = "/tmp/miniblog-notion-" + ("deploy-" if args.mode in ("deploy_token", "cleanup_deploy") else ("rollout-" if args.mode in ("rollout", "cleanup_rollout") else "ops-")) + run
            def ssh(command, timeout=1800, failure="remote SSH command failed"):
                try:
                    result = subprocess.run(["ssh", *common, "-p", port, destination, command],
                                            stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=timeout, check=False)
                except subprocess.TimeoutExpired:
                    raise SafeError("SSH operation timed out") from None
                if result.returncode:
                    raise SafeError(ssh_error_category(result.stderr, result.returncode, failure))
                return 0
            if args.mode in ("cleanup_deploy", "cleanup_ops", "cleanup_rollout"):
                if args.mode == "cleanup_rollout":
                    ssh(rollout_cleanup_command(run), 120, "owned maintenance container cleanup failed")
                if ssh(stage_command("cleanup", stage, run), 60, "remote private staging cleanup failed"):
                    raise SafeError("remote private staging cleanup failed")
                print("Private operation staging removed.")
                return 0
            stage_created = False
            deployment_handoff = False
            try:
                # Refuse to reuse an existing directory, including a symlink.
                if ssh(stage_command("create", stage, run), 60, "remote private staging creation failed"):
                    raise SafeError("remote private staging creation failed")
                stage_created = True
                private_write(temp / "credential", token)
                shutil.copyfile(Path(__file__).with_name("remote.py"), temp / "remote.py")
                os.chmod(temp / "remote.py", 0o600)
                files = ["remote.py", "credential"]
                if args.mode == "rollout":
                    shutil.copyfile(Path(__file__).with_name("rollout.py"), temp / "rollout.py")
                    os.chmod(temp / "rollout.py", 0o600)
                    files.append("rollout.py")
                    if writer:
                        private_write(temp / "writer", writer)
                        files.append("writer")
                for name in files:
                    try:
                        result = subprocess.run(["scp", *common, "-P", port, str(temp / name), destination + ":" + stage + "/" + name],
                                                stdout=subprocess.DEVNULL, stderr=subprocess.PIPE, timeout=120, check=False)
                    except subprocess.TimeoutExpired:
                        raise SafeError("SSH file transfer timed out") from None
                    if result.returncode:
                        raise SafeError(ssh_error_category(result.stderr, result.returncode, "private operation files could not be transferred"))
                if args.mode == "deploy_token":
                    deployment_handoff = True
                    print("Read credential staged privately for deployment.")
                    return 0  # The next deployment step consumes it; always() cleanup removes staging.
                if args.mode == "rollout":
                    command = ("python3 " + stage + "/rollout.py --app-dir /opt/miniblog --action " + shlex.quote(rollout_action) +
                               " --manifest-id " + shlex.quote(review_id) + " --manifest-sha256 " + review_hash +
                               " --expected-image-sha " + image_sha + " --expected-config-revision " + revision +
                               " --run-id " + run + (" --reader " + stage + "/credential" if token else "") +
                               (" --writer " + stage + "/writer" if writer else ""))
                    print("Restricted server reports: /opt/miniblog/ops/notion/rollout-" + run, flush=True)
                    ssh(command, failure="reviewed maintenance failed; inspect restricted server reports")
                    print("Reviewed maintenance completed: " + rollout_action)
                    return 0
                print("Restricted server reports: /opt/miniblog/ops/notion/" + run, flush=True)
                command = ("python3 " + stage + "/remote.py read-only --app-dir /opt/miniblog --credential " +
                           stage + "/credential --mode " + args.mode + " --run-id " + run)
                if ssh(command, failure="read maintenance failed; inspect restricted server reports"):
                    raise SafeError("read maintenance failed; inspect restricted server reports")
                print("Read maintenance completed: " + args.mode)
            finally:
                if stage_created and not deployment_handoff:
                    if ssh(stage_command("cleanup", stage, run), 60, "remote private staging cleanup failed"):
                        raise SafeError("remote private staging cleanup failed")
    except SafeError as error:
        print("Notion operation blocked: " + str(error), file=sys.stderr)
        return 1
    except Exception:
        print("Notion operation failed; no credential values were logged.", file=sys.stderr)
        return 1
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
