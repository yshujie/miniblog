#!/usr/bin/env python3
"""Restricted, reviewed server maintenance. Never print private input or reports."""
import argparse
from datetime import datetime, timedelta, timezone
import hashlib
import json
import os
from pathlib import Path
import re
import signal
import stat
import subprocess
import sys
import time
import urllib.error
import urllib.request

import remote

ACTIONS = ("pause", "resume", "catalog_prepare", "catalog_bind", "catalog_activate",
           "source_update", "bootstrap_preview", "bootstrap_apply", "sync",
           "scheduler_on", "scheduler_off")
SCOPED = set(ACTIONS) - {"pause", "resume", "scheduler_off"}
READ_ACTIONS = {"catalog_prepare", "bootstrap_preview", "bootstrap_apply", "sync", "scheduler_on"}
EMPTY_INPUT = {"pause", "resume", "sync", "scheduler_off"}
IMAGE_REPOSITORY = "ghcr.io/yshujie/miniblog-backend:"
SafeError = remote.SafeError


def private_directory(path, create=False):
    path = Path(path)
    if create and not path.exists():
        path.mkdir(mode=0o700)
    value = path.lstat()
    if not stat.S_ISDIR(value.st_mode) or value.st_uid != os.getuid() or value.st_mode & 0o077:
        raise SafeError("maintenance directory must be owned, regular and private")


def strict_object(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise SafeError("review JSON contains duplicate fields")
        result[key] = value
    return result


def reviewed_manifest(app_dir, identifier, digest, action, image_sha, revision):
    if (action not in ACTIONS or not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9_-]{0,79}", identifier)
            or not re.fullmatch(r"[0-9a-f]{64}", digest)
            or not re.fullmatch(r"[0-9a-f]{40}", image_sha)
            or type(revision) is not int or revision < 0):
        raise SafeError("review operation identifiers are invalid")
    if not stat.S_ISDIR(Path(app_dir).lstat().st_mode):
        raise SafeError("application directory cannot be a symlink")
    directory = Path(app_dir) / "ops" / "notion" / "reviews"
    for path in (directory.parent.parent, directory.parent, directory):
        private_directory(path)
    path = directory / (identifier + ".json")
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW)
    with os.fdopen(fd, "rb") as stream:
        value = os.fstat(stream.fileno())
        if (not stat.S_ISREG(value.st_mode) or value.st_uid != os.getuid()
                or value.st_mode & 0o077 or value.st_size > 2 * 1024 * 1024):
            raise SafeError("review file must be owned, regular, private and bounded")
        raw = stream.read(2 * 1024 * 1024 + 1)
    if hashlib.sha256(raw).hexdigest() != digest:
        raise SafeError("review file hash does not match approval")
    try:
        envelope = json.loads(raw, object_pairs_hook=strict_object, parse_constant=lambda _: (_ for _ in ()).throw(SafeError("review JSON contains non-finite values")))
    except (ValueError, UnicodeError):
        raise SafeError("review JSON is invalid") from None
    fields = {"version", "mode", "source_id", "expected_config_revision", "expected_image_sha", "input"}
    if (not isinstance(envelope, dict) or set(envelope) != fields
            or type(envelope["version"]) is not int or envelope["version"] != 1
            or envelope["mode"] != action or envelope["expected_image_sha"] != image_sha
            or type(envelope["expected_config_revision"]) is not int
            or envelope["expected_config_revision"] != revision
            or not isinstance(envelope["input"], dict)):
        raise SafeError("review envelope does not match the requested operation")
    source = envelope["source_id"]
    if action in SCOPED:
        if source not in remote.SOURCE_IDS or revision <= 0:
            raise SafeError("review requires a fixed source and positive configuration revision")
    elif source != "" or revision != 0:
        raise SafeError("global review cannot carry a source or configuration revision")
    validate_input(action, envelope["input"])
    return envelope


def validate_input(action, value):
    if action in EMPTY_INPUT and value:
        raise SafeError("this reviewed operation accepts no input fields")
    allowed = {
        "catalog_prepare": {"reuse_map"},
        "catalog_bind": {"binding_id", "option_id", "section_code"},
        "catalog_activate": {"items", "expected_newly_public_article_ids"},
        "source_update": {"label", "module_code", "enabled", "config", "expected_config_revision"},
        "bootstrap_preview": {"manual_matches"},
        "bootstrap_apply": {"confirmations"},
        "scheduler_on": {"validated_sync_run_id"},
    }
    if set(value) - allowed.get(action, set()):
        raise SafeError("review input contains unsupported fields")
    if action == "scheduler_on" and (set(value) != {"validated_sync_run_id"} or
            not re.fullmatch(r"[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}", str(value["validated_sync_run_id"]))):
        raise SafeError("scheduler approval requires a completed sync run identifier")
    if action == "bootstrap_apply" and (set(value) != {"confirmations"}
            or not isinstance(value["confirmations"], list) or not value["confirmations"]):
        raise SafeError("bootstrap approval requires explicit per-page confirmations")
    if action in {"catalog_bind", "catalog_activate", "source_update"} and not value:
        raise SafeError("reviewed mutation input cannot be empty")
    # Credentials never belong in durable reviews, even nested configuration.
    def check_keys(entry):
        if isinstance(entry, dict):
            for key, child in entry.items():
                if any(word in key.lower() for word in ("token", "secret", "password", "credential")):
                    raise SafeError("review input cannot contain credentials")
                check_keys(child)
        elif isinstance(entry, list):
            for child in entry:
                check_keys(child)
    check_keys(value)


def runtime_metadata(expected_sha):
    value = remote.docker_json(["inspect", "--format", '{"image":{{json .Image}},"tag":{{json .Config.Image}},"running":{{json .State.Running}},"health":{{json .State.Health.Status}},"env":{{json .Config.Env}}}', "miniblog-backend"])
    expected_tag = IMAGE_REPOSITORY + expected_sha
    if (not isinstance(value, dict) or value.get("running") is not True
            or value.get("tag") != expected_tag
            or not re.fullmatch(r"sha256:[0-9a-f]{64}", value.get("image", ""))
            or not isinstance(value.get("env"), list)):
        raise SafeError("running backend does not match the approved immutable revision")
    image = remote.docker_json(["image", "inspect", "--format", "{{json .Id}}", expected_tag])
    if image != value["image"]:
        raise SafeError("approved image tag and running image ID disagree")
    current = {}
    for entry in value["env"]:
        if not isinstance(entry, str) or "=" not in entry:
            raise SafeError("backend runtime environment metadata is invalid")
        key, content = entry.split("=", 1)
        if key in current:
            raise SafeError("backend runtime environment contains duplicate fields")
        current[key] = content
    switch = current.get("MINIBLOG_NOTION_SYNC_ENABLED", "").lower()
    if switch not in ("false", "0", "true", "1"):
        raise SafeError("backend runtime scheduler flag must be explicit")
    if current.get("MINIBLOG_NOTION_BOOTSTRAP_TOKEN"):
        raise SafeError("writer credentials cannot be resident in the backend")
    return value["image"], current, value


def no_active_tasks():
    result = subprocess.run(["docker", "ps", "-q"], stdout=subprocess.PIPE,
                            stderr=subprocess.DEVNULL, timeout=30, check=False, text=True)
    if result.returncode:
        raise SafeError("maintenance task inventory could not be inspected")
    for identifier in result.stdout.splitlines():
        if not re.fullmatch(r"[0-9a-f]{12,64}", identifier):
            raise SafeError("maintenance task inventory is invalid")
        row = remote.docker_json(["inspect", "--format", '{"name":{{json .Name}},"entrypoint":{{json .Config.Entrypoint}},"command":{{json .Config.Cmd}}}', identifier])
        name = str(row.get("name", "")).lstrip("/")
        command = (row.get("entrypoint") or []) + (row.get("command") or [])
        if (name.startswith(("miniblog-notion-ops-", "miniblog-notion-rollout-"))
                or any(isinstance(arg, str) and Path(arg).name == "notion-sync" for arg in command)):
            raise SafeError("another runtime or maintenance task is still running")


def drained(value, dual_pause=True, frozen=True, no_journal=False):
    required = {"paused", "source_writes_paused", "baseline_frozen", "current_run_id", "lease_owner", "lease_until", "lease_epoch", "unresolved_bootstrap_count"}
    if (not isinstance(value, dict) or not required.issubset(value)
            or type(value["paused"]) is not bool or type(value["source_writes_paused"]) is not bool
            or type(value["baseline_frozen"]) is not bool
            or type(value["unresolved_bootstrap_count"]) is not int or value["unresolved_bootstrap_count"] < 0
            or not isinstance(value["lease_epoch"], str) or not value["lease_epoch"].isdigit()):
        raise SafeError("drain evidence is incomplete")
    if dual_pause and (value["paused"] is not True or value["source_writes_paused"] is not True):
        raise SafeError("maintenance requires both pause switches")
    if frozen and value["baseline_frozen"] is not True:
        raise SafeError("maintenance requires a frozen historical baseline")
    if value["current_run_id"] != "" or value["lease_owner"] != "" or value["lease_until"] is not None:
        raise SafeError("runtime lease or current task has not drained")
    if no_journal and value["unresolved_bootstrap_count"] != 0:
        raise SafeError("unresolved bootstrap journals require explicit review")


def source_status(status, manifest):
    rows = status.get("sources") if isinstance(status, dict) else None
    if (not isinstance(rows, list) or len(rows) != 5
            or {row.get("source_id") for row in rows if isinstance(row, dict)} != remote.SOURCE_IDS):
        raise SafeError("fixed source inventory is incomplete")
    if manifest["source_id"]:
        row = next(row for row in rows if row["source_id"] == manifest["source_id"])
        if (type(row.get("config_revision")) is not int
                or row["config_revision"] != manifest["expected_config_revision"] or not row.get("module_code")):
            raise SafeError("reviewed source configuration has changed")
        return row
    return None


def instant(value):
    if not isinstance(value, str):
        raise SafeError("sync completion timestamps are missing")
    try:
        result = datetime.fromisoformat(value.replace("Z", "+00:00"))
        if result.tzinfo is None:
            raise ValueError()
        return result
    except ValueError:
        raise SafeError("sync completion timestamps are invalid") from None


def validate_sync_evidence(run, source, identifier, now=None):
    if (not isinstance(run, dict) or run.get("run_id") != identifier
            or run.get("mode") != "sync" or run.get("status") != "completed"
            or run.get("error") or source.get("enabled") is not True
            or not isinstance(run.get("counts"), dict)
            or type(run["counts"].get("failed")) is not int or run["counts"]["failed"] != 0
            or type(run["counts"].get("blocked")) is not int or run["counts"]["blocked"] != 0):
        raise SafeError("scheduler approval lacks a successful enabled-source sync")
    start, finish, success = instant(run.get("started_at")), instant(run.get("finished_at")), instant(source.get("last_success_at"))
    now = now or datetime.now(timezone.utc)
    if not start <= success <= finish or finish < now - timedelta(minutes=30) or finish > now + timedelta(seconds=30):
        raise SafeError("scheduler sync validation is stale or unrelated")


def immutable_private(path, value):
    descriptor = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    with os.fdopen(descriptor, "w", encoding="utf-8") as output:
        output.write(json.dumps(value, ensure_ascii=False) + "\n")
        output.flush()
        os.fsync(output.fileno())


def sync_receipt(app_dir, identifier, value=None):
    if not re.fullmatch(r"[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}", identifier):
        raise SafeError("sync receipt run identifier is invalid")
    directory = Path(app_dir) / "ops" / "notion" / "sync-receipts"
    for parent in (directory.parent.parent, directory.parent):
        private_directory(parent)
    private_directory(directory, create=value is not None)
    path = directory / (identifier + ".json")
    if value is not None:
        try:
            immutable_private(path, value)
        except FileExistsError:
            raise SafeError("sync success receipt cannot be replaced") from None
        return value
    descriptor = os.open(path, os.O_RDONLY | os.O_NOFOLLOW)
    with os.fdopen(descriptor, "rb") as stream:
        info = os.fstat(stream.fileno())
        if not stat.S_ISREG(info.st_mode) or info.st_uid != os.getuid() or info.st_mode & 0o077 or info.st_size > 8192:
            raise SafeError("sync receipt is not a private bounded file")
        try:
            result = json.loads(stream.read(8193), object_pairs_hook=strict_object)
        except (ValueError, UnicodeError):
            raise SafeError("sync receipt is invalid") from None
    return result


def validate_sync_receipt(receipt, run, source, manifest, image, reader):
    expected = {"version": 1, "run_id": run["run_id"], "source_id": manifest["source_id"],
                "config_revision": manifest["expected_config_revision"], "image_id": image,
                "expected_image_sha": manifest["expected_image_sha"], "reader_sha256": hashlib.sha256(reader.encode()).hexdigest(),
                "started_at": run["started_at"], "finished_at": run["finished_at"],
                "last_success_at": source["last_success_at"]}
    if (not isinstance(receipt, dict) or type(receipt.get("version")) is not int
            or type(receipt.get("config_revision")) is not int or receipt != expected):
        raise SafeError("sync success receipt does not match current configuration and image")


def validate_recovery_scope(evidence, confirmations):
    if evidence["unresolved_bootstrap_count"] == 0:
        return
    pages = evidence.get("unresolved_bootstrap_page_ids")
    if (not isinstance(pages, list) or len(pages) != evidence["unresolved_bootstrap_count"]
            or len(set(pages)) != len(pages)
            or any(not isinstance(page, str) or not re.fullmatch(r"[0-9a-f]{32}", page) for page in pages)):
        raise SafeError("bootstrap recovery page evidence is incomplete")
    requested = [item.get("page_id") for item in confirmations if isinstance(item, dict)]
    if (len(requested) != len(confirmations) or not requested
            or len(set(requested)) != len(requested) or not set(requested).issubset(pages)):
        raise SafeError("bootstrap recovery is restricted to unresolved reviewed pages")


def scheduler_content(original, enabled, author):
    if not isinstance(author, str) or not author.strip() or len(author) > 128 or any(c in author for c in "\r\n\x00\\"):
        raise SafeError("resolved default author is unsupported")
    # Compose single quotes prevent interpolation. Backslash is rejected to avoid
    # escape ambiguity; a literal apostrophe is supported by Compose's quote escape.
    quoted = "'" + author.replace("'", "\\'") + "'"
    keys = {"MINIBLOG_NOTION_SYNC_ENABLED", "MINIBLOG_NOTION_SYNC_AUTHOR", "MINIBLOG_NOTION_BOOTSTRAP_TOKEN"}
    lines = [line for line in original.splitlines() if remote.env_key(line) not in keys]
    return "\n".join(lines + ["MINIBLOG_NOTION_SYNC_ENABLED=" + str(enabled).lower(), "MINIBLOG_NOTION_SYNC_AUTHOR=" + quoted]) + "\n"


class SchedulerDiagnostics:
    """A bounded private trace containing only fixed categories and comparisons."""
    STAGES = {"env_write", "env_written", "compose_restart", "metadata", "runtime_check",
              "http_local", "http_public", "health_verified", "health_timeout", "operation_failed",
              "operation_completed", "rollback_env", "rollback_local_verified", "rollback_verified",
              "rollback_failed"}
    ERRORS = {"http_error", "url_error", "timeout", "os_error", "validation_error", "cancelled", "unexpected_error"}
    BOOLEANS = {"running", "tag_matches", "image_matches", "scheduler_matches", "author_matches"}

    def __init__(self, path):
        self.path, self.events, self.available = Path(path), [], True

    def record(self, stage, phase="apply", **fields):
        if stage not in self.STAGES or phase not in {"apply", "rollback"}:
            raise SafeError("scheduler diagnostic category is invalid")
        for key, value in fields.items():
            valid = ((key in self.BOOLEANS and type(value) is bool)
                     or (key == "docker_health" and value in {"starting", "healthy", "unhealthy", "missing", "unknown"})
                     or (key == "exception_class" and value in self.ERRORS)
                     or (key == "http_status" and type(value) is int and 100 <= value <= 599)
                     or (key == "returncode" and type(value) is int and -255 <= value <= 255))
            if not valid:
                raise SafeError("scheduler diagnostic field is invalid")
        events = (self.events + [{"phase": phase, "stage": stage, **fields}])[-256:]
        try:
            remote.atomic_private(self.path, json.dumps({"events": events}) + "\n")
        except (OSError, SafeError):
            # A trace failure must not prevent the safety rollback. The caller
            # refuses successful activation if evidence could not be persisted.
            self.available = False
            return False
        self.events = events
        return True


def scheduler_exception(error):
    if isinstance(error, urllib.error.HTTPError):
        return "http_error"
    if isinstance(error, urllib.error.URLError):
        return "url_error"
    if isinstance(error, (TimeoutError, subprocess.TimeoutExpired)):
        return "timeout"
    if isinstance(error, OSError):
        return "os_error"
    if isinstance(error, SafeError):
        return "validation_error"
    if isinstance(error, (KeyboardInterrupt, SystemExit)):
        return "cancelled"
    return "unexpected_error"


def scheduler_record(diagnostics, stage, phase="apply", **fields):
    if diagnostics is not None:
        try:
            return diagnostics.record(stage, phase=phase, **fields)
        except BaseException:
            # Even a cancellation during diagnostic I/O cannot skip rollback.
            diagnostics.available = False
            return False
    return True


def restart_backend(app_dir, sha, diagnostics=None, phase="apply"):
    environment = dict(os.environ, BACKEND_IMAGE_TAG=IMAGE_REPOSITORY + sha)
    scheduler_record(diagnostics, "compose_restart", phase)
    try:
        result = subprocess.run(["docker", "compose", "--env-file", ".env", "-f", "docker-compose.yml", "-f", "docker-compose.prod.yml", "up", "-d", "--no-deps", "--no-build", "--pull", "never", "miniblog-backend"],
                                cwd=app_dir, env=environment, stdout=subprocess.DEVNULL,
                                stderr=subprocess.DEVNULL, timeout=180, check=False)
    except (OSError, subprocess.TimeoutExpired) as error:
        scheduler_record(diagnostics, "compose_restart", phase, exception_class=scheduler_exception(error))
        raise SafeError("backend scheduler restart did not complete; inspect private evidence") from None
    scheduler_record(diagnostics, "compose_restart", phase, returncode=result.returncode)
    if result.returncode:
        raise SafeError("backend scheduler restart failed; inspect private evidence")


def healthy_backend(sha, image, enabled, author, diagnostics=None, phase="apply", check_public=True):
    if not check_public and (phase != "rollback" or enabled is not False):
        raise SafeError("public health may only be omitted when verifying an off rollback")
    deadline = time.monotonic() + 90
    while True:
        stage = "metadata"
        try:
            current_image, environment, row = runtime_metadata(sha)
            image_matches = current_image == image
            scheduler_matches = environment.get("MINIBLOG_NOTION_SYNC_ENABLED", "").lower() == str(enabled).lower()
            author_matches = environment.get("MINIBLOG_NOTION_SYNC_AUTHOR") == author
            health = row.get("health", "missing")
            if health not in {"starting", "healthy", "unhealthy", "missing"}:
                health = "unknown"
            scheduler_record(diagnostics, "metadata", phase, running=row.get("running") is True,
                             tag_matches=row.get("tag") == IMAGE_REPOSITORY + sha, image_matches=image_matches)
            stage = "runtime_check"
            scheduler_record(diagnostics, stage, phase, docker_health=health, image_matches=image_matches,
                             scheduler_matches=scheduler_matches, author_matches=author_matches)
            if image_matches and health == "healthy" and scheduler_matches and author_matches:
                probes = [("http_local", "http://127.0.0.1:8090/health")]
                if check_public:
                    probes.append(("http_public", "https://api.yangshujie.com/health"))
                for stage, url in probes:
                    with urllib.request.urlopen(url, timeout=8) as response:
                        status = response.status
                        if type(status) is not int or not 100 <= status <= 599:
                            raise SafeError("backend health response metadata is invalid")
                        scheduler_record(diagnostics, stage, phase, http_status=status)
                        if status != 200:
                            raise SafeError("backend public health verification failed")
                scheduler_record(diagnostics, "health_verified" if check_public else "rollback_local_verified", phase)
                return
        except subprocess.TimeoutExpired as error:
            scheduler_record(diagnostics, stage, phase, exception_class=scheduler_exception(error))
            raise
        except (SafeError, OSError) as error:
            fields = {"exception_class": scheduler_exception(error)}
            if isinstance(error, urllib.error.HTTPError) and type(error.code) is int and 100 <= error.code <= 599:
                fields["http_status"] = error.code
            scheduler_record(diagnostics, stage, phase, **fields)
        if time.monotonic() >= deadline:
            scheduler_record(diagnostics, "health_timeout", phase)
            raise SafeError("backend scheduler health verification did not complete")
        time.sleep(3)


class Runner:
    def __init__(self, app_dir, directory, manifest, image, database, reader, writer):
        self.app_dir, self.directory, self.manifest, self.image = Path(app_dir), Path(directory), manifest, image
        self.database, self.reader, self.writer = database, reader, writer
        self.counter = 0

    def task(self, mode, extra=None, write=False):
        self.counter += 1
        label = f"{self.counter:02d}-{mode}"
        env_file = self.directory / (label + ".env")
        fields = dict(self.database)
        if mode in {"catalog_prepare", "bootstrap_preview", "bootstrap_apply", "sync"}:
            fields["MINIBLOG_NOTION_TOKEN"] = self.reader
        if write:
            if mode != "bootstrap_apply" or not self.writer:
                raise SafeError("writer credential is restricted to approved bootstrap apply")
            fields["MINIBLOG_NOTION_BOOTSTRAP_TOKEN"] = self.writer
        remote.atomic_private(env_file, "".join(key + "=" + value + "\n" for key, value in fields.items()))
        args = ["--mode", mode, "--report", "/ops/" + label + ".json"] + (extra or [])
        try:
            remote.docker_task(self.image, self.directory, env_file, "/app/notion-sync", args, label, network=True)
            return remote.report_json(self.directory / (label + ".json"))
        finally:
            env_file.unlink(missing_ok=True)

    def input_file(self, name, value):
        path = self.directory / name
        remote.atomic_private(path, json.dumps(value, ensure_ascii=False) + "\n")
        return "/ops/" + name

    def scope(self):
        return ["--source-id", self.manifest["source_id"], "--expected-config-revision", str(self.manifest["expected_config_revision"])]

    def status(self):
        value = self.task("status")
        return value, source_status(value, self.manifest)

    def drain(self, **kwargs):
        value = self.task("drain_status")
        drained(value, **kwargs)
        no_active_tasks()
        return value

    def control(self, paused):
        path = self.input_file("control-" + str(self.counter) + ".json", {"paused": paused, "source_writes_paused": paused})
        return self.task("control_update", ["--input", path])

    def scheduler(self, enabled, runtime):
        author_result = self.task("author_resolve")
        author = author_result.get("author") if isinstance(author_result, dict) else None
        self.drain(no_journal=enabled)
        self.status()
        target = self.app_dir / ".env"
        remote.private_file(target)
        original = target.read_text(encoding="utf-8")
        # Persist an off recovery environment, never a writer or an unsafe on backup.
        backup = scheduler_content(original, False, author)
        diagnostics = SchedulerDiagnostics(self.directory / "scheduler-diagnostics.json")
        if not diagnostics.record("env_write"):
            raise SafeError("scheduler private evidence could not be persisted")
        try:
            remote.atomic_private(self.app_dir / ".env.previous", backup)
            remote.atomic_private(target, scheduler_content(original, enabled, author))
            if not diagnostics.record("env_written"):
                raise SafeError("scheduler private evidence could not be persisted")
            restart_backend(self.app_dir, self.manifest["expected_image_sha"], diagnostics)
            healthy_backend(self.manifest["expected_image_sha"], self.image, enabled, author, diagnostics)
            if not diagnostics.available:
                raise SafeError("scheduler private evidence could not be persisted")
            self.drain(no_journal=enabled)
            self.status()
            if not diagnostics.available or not diagnostics.record("operation_completed"):
                raise SafeError("scheduler private evidence could not be persisted")
        except BaseException as error:
            scheduler_record(diagnostics, "operation_failed", exception_class=scheduler_exception(error))
            if enabled:
                # Rollback verifies the off switch and local backend recovery.
                # A failed public check still fails this operation; it cannot be
                # used to claim successful activation or public availability.
                try:
                    remote.atomic_private(target, backup)
                    scheduler_record(diagnostics, "rollback_env", phase="rollback")
                    restart_backend(self.app_dir, self.manifest["expected_image_sha"], diagnostics, phase="rollback")
                    healthy_backend(self.manifest["expected_image_sha"], self.image, False, author,
                                    diagnostics, phase="rollback", check_public=False)
                    scheduler_record(diagnostics, "rollback_verified", phase="rollback")
                except BaseException as rollback_error:
                    scheduler_record(diagnostics, "rollback_failed", phase="rollback", exception_class=scheduler_exception(rollback_error))
            raise
        return {"scheduler_enabled": enabled, "image_id": self.image}

    def execute(self, runtime):
        action, value = self.manifest["mode"], self.manifest["input"]
        if action == "pause":
            self.control(True)
            # Runtime requests check cancellation periodically; a cleared DB lease
            # alone cannot prove its HTTP operation or CLI container has ended.
            time.sleep(20)
            deadline = time.monotonic() + 40
            while True:
                try:
                    self.drain(frozen=False)
                    break
                except SafeError:
                    if time.monotonic() >= deadline:
                        raise SafeError("pause applied but runtime has not drained") from None
                    time.sleep(3)
            return {"paused": True, "source_writes_paused": True}
        if action == "sync":
            status, source = self.status()
            if (status.get("paused") is not False or status.get("source_writes_paused") is not False
                    or status.get("baseline_frozen") is not True or source.get("enabled") is not True):
                raise SafeError("immediate sync requires resumed control, frozen baseline and enabled source")
            self.drain(dual_pause=False, no_journal=True)
            author = self.task("author_resolve").get("author")
            if not isinstance(author, str) or not author.strip():
                raise SafeError("default author could not be resolved")
            result = self.task("sync", ["--enable-sync", "--author", author])
            run = result.get("run", {})
            if run.get("status") != "completed" or run.get("counts", {}).get("failed") != 0 or run.get("counts", {}).get("blocked") != 0:
                raise SafeError("immediate sync has unresolved errors; inspect private report")
            _, current_source = self.status()
            identifier = run.get("run_id", "")
            validate_sync_evidence(run, current_source, identifier)
            receipt = {"version": 1, "run_id": identifier, "source_id": self.manifest["source_id"],
                       "config_revision": self.manifest["expected_config_revision"], "image_id": self.image,
                       "expected_image_sha": self.manifest["expected_image_sha"], "reader_sha256": hashlib.sha256(self.reader.encode()).hexdigest(),
                       "started_at": run["started_at"], "finished_at": run["finished_at"],
                       "last_success_at": current_source["last_success_at"]}
            sync_receipt(self.app_dir, identifier, receipt)
            return result
        evidence = self.drain(no_journal=action in {"resume", "scheduler_on", "catalog_prepare", "catalog_bind", "catalog_activate", "source_update"})
        status, source = self.status()
        if action == "resume":
            result = self.control(False)
            if result.get("paused") is not False or result.get("source_writes_paused") is not False:
                raise SafeError("resume could not be verified")
            return result
        if action == "scheduler_on":
            if runtime.get("MINIBLOG_NOTION_SYNC_ENABLED", "").lower() not in ("false", "0"):
                raise SafeError("scheduler is already enabled; an on operation cannot be replayed")
            if not self.reader or runtime.get("MINIBLOG_NOTION_TOKEN") != self.reader:
                raise SafeError("resident read credential does not match the validated dedicated reader")
            identifier = value["validated_sync_run_id"]
            run = self.task("run_status", ["--run-id", identifier])
            validate_sync_evidence(run, source, identifier)
            validate_sync_receipt(sync_receipt(self.app_dir, identifier), run, source, self.manifest, self.image, self.reader)
            return self.scheduler(True, runtime)
        if action == "scheduler_off":
            return self.scheduler(False, runtime)
        if action == "catalog_prepare":
            extra = self.scope()
            if value:
                extra += ["--catalog-plan", self.input_file("catalog-plan.json", value)]
            return self.task(action, extra)
        if action in {"catalog_bind", "catalog_activate", "source_update"}:
            if action == "source_update" and "expected_config_revision" in value and value["expected_config_revision"] != self.manifest["expected_config_revision"]:
                raise SafeError("source update input revision disagrees with the review envelope")
            return self.task(action, self.scope() + ["--input", self.input_file("input.json", value)])
        if action == "bootstrap_preview":
            extra = []
            if value:
                extra += ["--manual-matches", self.input_file("manual-matches.json", value)]
            return self.task(action, extra)
        if action == "bootstrap_apply":
            validate_recovery_scope(evidence, value["confirmations"])
            return self.task(action, self.scope() + ["--allow-notion-write", "--confirmations", self.input_file("confirmations.json", value)], write=True)
        raise SafeError("unsupported reviewed operation")


def maintenance(app_dir, action, identifier, digest, sha, revision, run_id, reader_path=None, writer_path=None):
    if not re.fullmatch(r"[0-9]+-[0-9]+", run_id):
        raise SafeError("maintenance run identifier is invalid")
    app_dir = Path(app_dir)
    reader = remote.token_from(reader_path) if reader_path else ""
    writer = remote.token_from(writer_path) if writer_path else ""
    if action in READ_ACTIONS and not reader:
        raise SafeError("dedicated read credential is required for this operation")
    if (action == "bootstrap_apply") != bool(writer):
        raise SafeError("writer credential is required only for bootstrap apply")
    with remote.operation_lock(app_dir):
        manifest = reviewed_manifest(app_dir, identifier, digest, action, sha, revision)
        image, runtime, _ = runtime_metadata(sha)
        database = remote.database_environment(require_scheduler_off=False)
        root = app_dir / "ops" / "notion"
        directory = root / ("rollout-" + run_id)
        try:
            directory.mkdir(mode=0o700)
        except FileExistsError:
            raise SafeError("maintenance run report directory cannot be reused") from None
        private_directory(directory)
        remote.atomic_private(directory / "operation.json", json.dumps({"manifest_id": identifier, "manifest_sha256": digest, "action": action, "expected_image_sha": sha, "image_id": image, "run_id": run_id}) + "\n")
        runner = Runner(app_dir, directory, manifest, image, database, reader, writer)
        try:
            result = runner.execute(runtime)
            remote.atomic_private(directory / "result.json", json.dumps(result, ensure_ascii=False) + "\n")
        except BaseException:
            remote.atomic_private(directory / "operation-failed.json", '{"status":"failed","review_required":true}\n')
            raise
    print("Reviewed maintenance completed: " + action)
    print("Restricted server reports: /opt/miniblog/ops/notion/rollout-" + run_id)


def main():
    parser = remote.SafeParser(description=__doc__)
    parser.add_argument("--app-dir", required=True)
    parser.add_argument("--action", required=True, choices=ACTIONS)
    parser.add_argument("--manifest-id", required=True)
    parser.add_argument("--manifest-sha256", required=True)
    parser.add_argument("--expected-image-sha", required=True)
    parser.add_argument("--expected-config-revision", required=True, type=int)
    parser.add_argument("--run-id", required=True)
    parser.add_argument("--reader")
    parser.add_argument("--writer")
    try:
        os.umask(0o077)
        def cancelled(_signum, _frame):
            raise SafeError("reviewed operation cancelled; inspect private report")
        for signum in (signal.SIGTERM, signal.SIGINT, signal.SIGHUP):
            signal.signal(signum, cancelled)
        args = parser.parse_args()
        maintenance(args.app_dir, args.action, args.manifest_id, args.manifest_sha256, args.expected_image_sha,
                    args.expected_config_revision, args.run_id, args.reader, args.writer)
        return 0
    except SafeError as error:
        print("Reviewed operation blocked: " + str(error), file=sys.stderr)
    except Exception:
        print("Reviewed operation failed; inspect restricted server evidence.", file=sys.stderr)
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
