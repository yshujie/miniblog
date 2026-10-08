#!/usr/bin/env python3
"""Actions runner: private-file credential transport, never secret-valued SSH arguments."""
import argparse
import base64
import hashlib
import os
from pathlib import Path
import re
import shutil
import signal
import subprocess
import tempfile
import sys

MODES = ("deploy_token", "cleanup_deploy", "cleanup_ops", "schema_check", "dry_run", "bootstrap_preview")

class SafeError(Exception):
    pass


def private_write(path, value):
    descriptor = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    with os.fdopen(descriptor, "w", encoding="utf-8") as output:
        output.write(value)


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
        token = os.environ.get("MINIBLOG_NOTION_TOKEN", "")
        if args.mode not in ("cleanup_deploy", "cleanup_ops") and token and not re.fullmatch(r"[A-Za-z0-9_-]{10,4096}", token):
            raise SafeError("invalid read credential format")
        if args.mode in ("schema_check", "dry_run", "bootstrap_preview") and not token:
            raise SafeError("dedicated read credential is not configured")
        key = os.environ.get("SVRD_SSH_KEY", "")
        if not key:
            raise SafeError("SSH credential is not configured")
        os.umask(0o077)
        with tempfile.TemporaryDirectory(prefix="miniblog-notion-", dir=os.environ.get("RUNNER_TEMP")) as temporary:
            temp = Path(temporary)
            private_write(temp / "key", key)
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
            stage = "/tmp/miniblog-notion-" + ("deploy-" if args.mode in ("deploy_token", "cleanup_deploy") else "ops-") + run
            def ssh(command, timeout=1800):
                result = subprocess.run(["ssh", *common, "-p", port, destination, command],
                                        stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, timeout=timeout, check=False)
                return result.returncode
            if args.mode in ("cleanup_deploy", "cleanup_ops"):
                if ssh("rm -rf -- " + stage, 60):
                    raise SafeError("remote private staging cleanup failed")
                print("Private operation staging removed.")
                return 0
            try:
                # Refuse to reuse an existing directory, including a symlink.
                if ssh("umask 077; mkdir -- " + stage, 60):
                    raise SafeError("remote private staging creation failed")
                private_write(temp / "credential", token)
                shutil.copyfile(Path(__file__).with_name("remote.py"), temp / "remote.py")
                os.chmod(temp / "remote.py", 0o600)
                for name in ("remote.py", "credential"):
                    result = subprocess.run(["scp", *common, "-P", port, str(temp / name), destination + ":" + stage + "/" + name],
                                            stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=120, check=False)
                    if result.returncode:
                        raise SafeError("private operation files could not be transferred")
                if args.mode == "deploy_token":
                    print("Read credential staged privately for deployment.")
                    return 0  # The next deployment step consumes it; always() cleanup removes staging.
                print("Restricted server reports: /opt/miniblog/ops/notion/" + run, flush=True)
                command = ("python3 " + stage + "/remote.py read-only --app-dir /opt/miniblog --credential " +
                           stage + "/credential --mode " + args.mode + " --run-id " + run)
                if ssh(command):
                    raise SafeError("read maintenance failed; inspect restricted server reports")
                print("Read maintenance completed: " + args.mode)
            finally:
                if args.mode != "deploy_token":
                    if ssh("rm -rf -- " + stage, 60):
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
