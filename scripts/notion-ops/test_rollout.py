import contextlib
from datetime import datetime, timedelta, timezone
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import re
import stat
import subprocess
import sys
import tempfile
import unittest
import urllib.error
from unittest import mock

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import rollout
import transport

SHA = "b" * 40
IMAGE = "sha256:" + "a" * 64
SOURCE = "2bf330bd-ddf1-80a6-aa49-000bbd1e154b"
TOKEN = "ntn_READER_SENTINEL_0123456789"
WRITER = "ntn_WRITER_SENTINEL_0123456789"
PASSWORD = "DATABASE_SENTINEL_$private'"
RUN = "12345678-1234-1234-1234-123456789abc"


def envelope(action="catalog_prepare", value=None):
    scoped = action in rollout.SCOPED
    return {"version": 1, "mode": action, "source_id": SOURCE if scoped else "",
            "expected_config_revision": 7 if scoped else 0,
            "expected_image_sha": SHA, "input": value or {}}


def drain(**overrides):
    value = {"paused": True, "source_writes_paused": True, "baseline_frozen": True,
             "current_run_id": "", "lease_owner": "", "lease_until": None,
             "lease_epoch": "7", "unresolved_bootstrap_count": 0}
    value.update(overrides)
    return value


def status(**overrides):
    value = {"paused": True, "source_writes_paused": True, "baseline_frozen": True,
             "sources": [{"source_id": source, "config_revision": 7, "module_code": "go",
                          "enabled": True} for source in rollout.remote.SOURCE_IDS]}
    value.update(overrides)
    return value


class ReviewTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.app = Path(self.temp.name)
        self.reviews = self.app / "ops" / "notion" / "reviews"
        self.reviews.mkdir(parents=True)
        for path in (self.reviews, self.reviews.parent, self.reviews.parent.parent):
            path.chmod(0o700)
        self.path = self.reviews / "go-review.json"
        self.write(envelope())

    def write(self, value):
        self.path.write_text(json.dumps(value))
        self.path.chmod(0o600)

    def review(self, action="catalog_prepare", revision=7, digest=None):
        return rollout.reviewed_manifest(self.app, "go-review", digest or hashlib.sha256(self.path.read_bytes()).hexdigest(), action, SHA, revision)

    def test_exact_approved_bytes_scope_and_revision(self):
        self.assertEqual(self.review(), envelope())
        for key, value in (("version", True), ("source_id", "unknown"),
                           ("expected_config_revision", 8), ("expected_image_sha", "c" * 40),
                           ("unexpected", "field")):
            current = envelope()
            current[key] = value
            self.write(current)
            with self.assertRaises(rollout.SafeError):
                self.review()
        self.write(envelope())
        with self.assertRaises(rollout.SafeError):
            self.review(digest="a" * 64)

    def test_no_path_symlink_public_permission_or_duplicate_json(self):
        with self.assertRaises(rollout.SafeError):
            rollout.reviewed_manifest(self.app, "../go-review", "a" * 64, "catalog_prepare", SHA, 7)
        self.path.chmod(0o644)
        with self.assertRaises(rollout.SafeError):
            self.review()
        self.path.unlink()
        other = self.reviews / "other"
        other.write_text(json.dumps(envelope()))
        other.chmod(0o600)
        self.path.symlink_to(other)
        with self.assertRaises(OSError):
            self.review()
        self.path.unlink()
        self.path.write_text('{"version":1,"version":1}')
        self.path.chmod(0o600)
        with self.assertRaises(rollout.SafeError):
            self.review()

    def test_global_review_cannot_gain_source_or_input_authority(self):
        for action in ("pause", "resume", "scheduler_off"):
            self.write(envelope(action))
            self.assertEqual(self.review(action, 0)["source_id"], "")
            current = envelope(action)
            current["input"] = {"paused": False}
            self.write(current)
            with self.assertRaises(rollout.SafeError):
                self.review(action, 0)

    def test_private_review_cannot_persist_credentials(self):
        for action, value in (("source_update", {"config": {"token": TOKEN}}),
                              ("bootstrap_apply", {"confirmations": [{"secret": WRITER}]}),
                              ("catalog_prepare", {"reuse_map": [{"password": PASSWORD}]})):
            with self.assertRaises(rollout.SafeError):
                rollout.validate_input(action, value)

    def test_no_alias_bool_revision_nonfinite_or_unsafe_owner(self):
        current = envelope()
        current["expected_config_revision"] = True
        self.write(current)
        with self.assertRaises(rollout.SafeError):
            self.review(revision=1)
        self.path.write_text(json.dumps(envelope()).replace('"input": {}', '"input": {"reuse_map": NaN}'))
        with self.assertRaises(rollout.SafeError):
            self.review()
        self.write(envelope())
        with mock.patch.object(rollout.os, "getuid", return_value=os.getuid() + 1):
            with self.assertRaises(rollout.SafeError):
                self.review()


class GuardTests(unittest.TestCase):
    def test_drain_requires_complete_no_lease_and_unknown_journal_closed(self):
        rollout.drained(drain(), no_journal=True)
        for field in drain():
            current = drain()
            del current[field]
            with self.assertRaises(rollout.SafeError):
                rollout.drained(current)
        for changes in ({"lease_owner": "runtime"}, {"current_run_id": RUN},
                        {"lease_until": "2026-10-09T00:00:00Z"}, {"paused": False},
                        {"source_writes_paused": False}, {"baseline_frozen": False}):
            with self.assertRaises(rollout.SafeError):
                rollout.drained(drain(**changes))
        with self.assertRaises(rollout.SafeError):
            rollout.drained(drain(unresolved_bootstrap_count=1), no_journal=True)
        rollout.drained(drain(unresolved_bootstrap_count=1))

    def test_scheduler_evidence_is_fresh_exact_and_has_projection_success(self):
        now = datetime.now(timezone.utc)
        run = {"run_id": RUN, "mode": "sync", "status": "completed", "counts": {"failed": 0, "blocked": 0},
               "started_at": (now - timedelta(minutes=2)).isoformat(), "finished_at": now.isoformat()}
        source = {"enabled": True, "last_success_at": (now - timedelta(minutes=1)).isoformat()}
        rollout.validate_sync_evidence(run, source, RUN, now)
        for change in ({"status": "completed_with_errors"}, {"mode": "dry_run"},
                       {"counts": {"failed": 0, "blocked": 1}}, {"run_id": "different"},
                       {"finished_at": (now - timedelta(hours=1)).isoformat()}):
            with self.assertRaises(rollout.SafeError):
                rollout.validate_sync_evidence(dict(run, **change), source, RUN, now)
        with self.assertRaises(rollout.SafeError):
            rollout.validate_sync_evidence(run, dict(source, last_success_at=(now - timedelta(hours=1)).isoformat()), RUN, now)

    def test_tag_and_immutable_running_image_must_both_match(self):
        data = {"running": True, "tag": rollout.IMAGE_REPOSITORY + SHA, "image": IMAGE,
                "health": "healthy", "env": ["MINIBLOG_NOTION_SYNC_ENABLED=false"]}
        with mock.patch.object(rollout.remote, "docker_json", side_effect=[data, IMAGE]):
            self.assertEqual(rollout.runtime_metadata(SHA)[0], IMAGE)
        with mock.patch.object(rollout.remote, "docker_json", side_effect=[data, "sha256:" + "c" * 64]):
            with self.assertRaises(rollout.SafeError):
                rollout.runtime_metadata(SHA)
        for row in (dict(data, tag="latest"), dict(data, env=[]),
                    dict(data, env=data["env"] + ["MINIBLOG_NOTION_BOOTSTRAP_TOKEN=" + WRITER])):
            with mock.patch.object(rollout.remote, "docker_json", return_value=row):
                with self.assertRaises(rollout.SafeError):
                    rollout.runtime_metadata(SHA)

    def test_task_inventory_detects_runtime_cli_independent_of_db_lease(self):
        completed = subprocess.CompletedProcess([], 0, "a" * 12 + "\n")
        with mock.patch.object(rollout.subprocess, "run", return_value=completed), \
                mock.patch.object(rollout.remote, "docker_json", return_value={"name": "/some-other-cli", "entrypoint": ["/app/notion-sync"], "command": []}):
            with self.assertRaises(rollout.SafeError):
                rollout.no_active_tasks()


class RFC3339NanoTests(unittest.TestCase):
    def test_go_zero_to_nine_fraction_digits_are_exact_across_offsets(self):
        base = rollout.instant("2026-10-09T05:04:02Z")
        self.assertIs(type(base), int)
        self.assertEqual(base, rollout.instant("2026-10-09T13:04:02+08:00"))
        self.assertEqual(base, rollout.instant("2026-10-09T01:04:02-04:00"))
        self.assertEqual(base, rollout.instant("2026-10-09T05:04:02.000000000Z"))
        for digits in range(1, 10):
            fraction = "123456789"[:digits]
            with self.subTest(digits=digits):
                self.assertEqual(rollout.instant("2026-10-09T05:04:02." + fraction + "Z"),
                                 base + int(fraction.ljust(9, "0")))
                self.assertEqual(rollout.instant("2026-10-09T13:04:02." + fraction + "+08:00"),
                                 base + int(fraction.ljust(9, "0")))
        self.assertLess(rollout.instant("2026-10-09T05:04:02.123456788Z"),
                        rollout.instant("2026-10-09T05:04:02.123456789Z"))

    def test_go_timestamp_on_python310_fraction_parser(self):
        class Python310DateTime(datetime):
            @classmethod
            def fromisoformat(cls, value):
                fraction = re.search(r"\.([0-9]+)", value)
                if fraction and len(fraction.group(1)) not in (3, 6):
                    raise ValueError("Python 3.10 accepts only 3 or 6 fraction digits")
                return datetime.fromisoformat(value)
        run = {"run_id": RUN, "mode": "sync", "status": "completed",
               "counts": {"failed": 0, "blocked": 0},
               "started_at": "2026-10-09T13:03:47.382306+08:00",
               "finished_at": "2026-10-09T13:04:28.954917+08:00"}
        source = {"enabled": True, "last_success_at": "2026-10-09T13:04:02.04173+08:00"}
        now = datetime(2026, 10, 9, 5, 4, 29, tzinfo=timezone.utc)
        with self.assertRaises(ValueError):
            Python310DateTime.fromisoformat(source["last_success_at"])
        with mock.patch.object(rollout, "datetime", Python310DateTime):
            rollout.validate_sync_evidence(run, source, RUN, now)
            for digits in range(1, 10):
                self.assertIs(type(rollout.instant("2026-10-09T05:04:02." + "123456789"[:digits] + "Z")), int)

    def test_missing_timezone_malformed_date_offset_and_fraction_are_rejected(self):
        invalid = (None, False, 0, "", "2026-10-09T05:04:02",
                   "2026-10-09T05:04:02.123456", "2026-10-09 05:04:02Z",
                   "2026-10-09T05:04:02z", "2026-10-09T05:04:02UTC",
                   "2026-10-09T05:04:02+0800", "2026-10-09T05:04:02+24:00",
                   "2026-10-09T05:04:02+01:60", "2026-10-09T05:04:02.Z",
                   "2026-10-09T05:04:02.1234567890Z", "2026-02-30T05:04:02Z",
                   "2026-10-09T24:04:02Z", "2026-10-09T05:60:02Z",
                   "2026-10-09T05:04:60Z", "0000-10-09T05:04:02Z")
        for value in invalid:
            with self.subTest(value=value), self.assertRaises(rollout.SafeError):
                rollout.instant(value)

    def test_source_success_outside_run_by_one_nanosecond_is_rejected(self):
        now = datetime(2026, 10, 9, 5, 4, 29, tzinfo=timezone.utc)
        run = {"run_id": RUN, "mode": "sync", "status": "completed",
               "counts": {"failed": 0, "blocked": 0},
               "started_at": "2026-10-09T05:04:02.000000100Z",
               "finished_at": "2026-10-09T05:04:02.000000200Z"}
        for fraction in ("000000100", "000000150", "000000200"):
            rollout.validate_sync_evidence(run, {"enabled": True, "last_success_at":
                                           "2026-10-09T13:04:02." + fraction + "+08:00"}, RUN, now)
        for fraction in ("000000099", "000000201"):
            with self.subTest(fraction=fraction), self.assertRaises(rollout.SafeError):
                rollout.validate_sync_evidence(run, {"enabled": True, "last_success_at":
                                               "2026-10-09T05:04:02." + fraction + "Z"}, RUN, now)

    def test_stale_and_future_windows_preserve_nanosecond_boundaries(self):
        now = datetime(2026, 10, 9, 5, 4, 29, tzinfo=timezone.utc)
        for finish, allowed in (("2026-10-09T04:34:29Z", True),
                                ("2026-10-09T04:34:28.999999999Z", False),
                                ("2026-10-09T05:04:59Z", True),
                                ("2026-10-09T05:04:59.000000001Z", False)):
            run = {"run_id": RUN, "mode": "sync", "status": "completed",
                   "counts": {"failed": 0, "blocked": 0},
                   "started_at": "2026-10-09T04:00:00Z", "finished_at": finish}
            source = {"enabled": True, "last_success_at": finish}
            with self.subTest(finish=finish):
                if allowed:
                    rollout.validate_sync_evidence(run, source, RUN, now)
                else:
                    with self.assertRaises(rollout.SafeError):
                        rollout.validate_sync_evidence(run, source, RUN, now)

    def test_success_timestamp_compatibility_does_not_relax_other_guards(self):
        now = datetime(2026, 10, 9, 5, 4, 29, tzinfo=timezone.utc)
        run = {"run_id": RUN, "mode": "sync", "status": "completed",
               "counts": {"failed": 0, "blocked": 0},
               "started_at": "2026-10-09T05:03:47.382306Z",
               "finished_at": "2026-10-09T05:04:28.954917Z"}
        source = {"enabled": True, "last_success_at": "2026-10-09T05:04:02.04173Z"}
        for change in ({"status": "completed_with_errors"}, {"mode": "dry_run"},
                       {"run_id": "other"}, {"error": "source_unavailable"},
                       {"counts": {"failed": 1, "blocked": 0}},
                       {"counts": {"failed": 0, "blocked": 1}},
                       {"counts": {"failed": False, "blocked": 0}}):
            with self.subTest(change=change), self.assertRaises(rollout.SafeError):
                rollout.validate_sync_evidence(dict(run, **change), source, RUN, now)
        with self.assertRaises(rollout.SafeError):
            rollout.validate_sync_evidence(run, dict(source, enabled=False), RUN, now)


class SchedulerHealthTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.directory = Path(self.temp.name)
        self.diagnostics = rollout.SchedulerDiagnostics(self.directory / "scheduler-diagnostics.json")

    def metadata(self, health="healthy", enabled="true", author="author", image=IMAGE):
        return image, {"MINIBLOG_NOTION_SYNC_ENABLED": enabled, "MINIBLOG_NOTION_SYNC_AUTHOR": author,
                       "MINIBLOG_NOTION_TOKEN": TOKEN, "MYSQL_PASSWORD": PASSWORD}, {
                           "tag": rollout.IMAGE_REPOSITORY + SHA, "running": True, "health": health}

    def response(self, status=200):
        response = mock.MagicMock()
        response.__enter__.return_value.status = status
        return response

    def report(self):
        path = self.directory / "scheduler-diagnostics.json"
        self.assertEqual(stat.S_IMODE(path.stat().st_mode), 0o600)
        text = path.read_text()
        for value in (TOKEN, WRITER, PASSWORD, "private-author", "private exception detail"):
            self.assertNotIn(value, text)
        return json.loads(text)["events"]

    def test_compose_restart_uses_explicit_app_env_and_backend_only(self):
        with mock.patch.object(rollout.subprocess, "run", return_value=subprocess.CompletedProcess([], 0)) as run:
            rollout.restart_backend(self.directory, SHA, self.diagnostics)
        args, = run.call_args.args
        self.assertEqual(args, ["docker", "compose", "--env-file", ".env", "-f", "docker-compose.yml",
                                "-f", "docker-compose.prod.yml", "up", "-d", "--no-deps", "--no-build",
                                "--pull", "never", "miniblog-backend"])
        self.assertEqual(run.call_args.kwargs["cwd"], self.directory)
        self.assertEqual(run.call_args.kwargs["env"]["BACKEND_IMAGE_TAG"], rollout.IMAGE_REPOSITORY + SHA)
        self.assertEqual(run.call_args.kwargs["stdout"], subprocess.DEVNULL)
        self.assertEqual(run.call_args.kwargs["stderr"], subprocess.DEVNULL)
        self.assertEqual(self.report()[-1]["returncode"], 0)

    def test_health_starting_then_healthy_records_only_safe_metadata(self):
        with mock.patch.object(rollout, "runtime_metadata", side_effect=[self.metadata("starting", author="private-author"), self.metadata(author="private-author")]), \
                mock.patch.object(rollout.urllib.request, "urlopen", side_effect=[self.response(), self.response()]) as opened, \
                mock.patch.object(rollout.time, "monotonic", side_effect=[0, 1]), mock.patch.object(rollout.time, "sleep"):
            rollout.healthy_backend(SHA, IMAGE, True, "private-author", self.diagnostics)
        self.assertEqual([call.args[0] for call in opened.call_args_list],
                         ["http://127.0.0.1:8090/health", "https://api.yangshujie.com/health"])
        events = self.report()
        checks = [event for event in events if event["stage"] == "runtime_check"]
        self.assertEqual([event["docker_health"] for event in checks], ["starting", "healthy"])
        self.assertTrue(all(event["image_matches"] and event["author_matches"] and event["scheduler_matches"] for event in checks))
        self.assertEqual(events[-1]["stage"], "health_verified")

    def test_public_502_still_fails_after_successful_local_health(self):
        failure = urllib.error.HTTPError("https://api.yangshujie.com/health", 502, "private exception detail", {}, None)
        with mock.patch.object(rollout, "runtime_metadata", return_value=self.metadata()), \
                mock.patch.object(rollout.urllib.request, "urlopen", side_effect=[self.response(), failure]), \
                mock.patch.object(rollout.time, "monotonic", side_effect=[0, 100]):
            with self.assertRaises(rollout.SafeError):
                rollout.healthy_backend(SHA, IMAGE, True, "author", self.diagnostics)
        events = self.report()
        self.assertIn({"phase": "apply", "stage": "http_public", "http_status": 502, "exception_class": "http_error"}, events)
        self.assertIn({"phase": "apply", "stage": "http_local", "http_status": 200}, events)
        self.assertEqual(events[-1]["stage"], "health_timeout")

    def test_http_responses_other_than_200_never_verify_health(self):
        for status in (204, 301, 401, 502):
            with self.subTest(status=status), mock.patch.object(rollout, "runtime_metadata", return_value=self.metadata()), \
                    mock.patch.object(rollout.urllib.request, "urlopen", return_value=self.response(status)), \
                    mock.patch.object(rollout.time, "monotonic", side_effect=[0, 100]):
                with self.assertRaises(rollout.SafeError):
                    rollout.healthy_backend(SHA, IMAGE, True, "author", self.diagnostics)
                events = self.report()
                self.assertEqual(events[-1]["stage"], "health_timeout")
                self.assertFalse(any(event["stage"] == "health_verified" for event in events))

    def test_public_check_cannot_be_skipped_for_activation(self):
        for options in ({"check_public": False}, {"check_public": False, "phase": "rollback"}):
            with self.subTest(options=options), mock.patch.object(rollout, "runtime_metadata") as metadata:
                with self.assertRaises(rollout.SafeError):
                    rollout.healthy_backend(SHA, IMAGE, True, "author", self.diagnostics, **options)
                metadata.assert_not_called()

    def test_compose_failure_and_timeout_diagnostics_never_store_raw_output(self):
        for result in (subprocess.CompletedProcess([], 7, TOKEN, PASSWORD),
                       subprocess.TimeoutExpired(["docker", WRITER], 180, output=TOKEN, stderr=PASSWORD)):
            with self.subTest(result=type(result).__name__):
                with mock.patch.object(rollout.subprocess, "run", **({"side_effect": result} if isinstance(result, Exception) else {"return_value": result})):
                    with self.assertRaises(rollout.SafeError):
                        rollout.restart_backend(self.directory, SHA, self.diagnostics)
                events = self.report()
                self.assertEqual(events[-1]["stage"], "compose_restart")
                if isinstance(result, Exception):
                    self.assertEqual(events[-1]["exception_class"], "timeout")
                else:
                    self.assertEqual(events[-1]["returncode"], 7)

    def test_diagnostic_fields_reject_raw_values_and_evidence_failure_is_flagged(self):
        for fields in ({"author": "private-author"}, {"exception_class": PASSWORD}, {"docker_health": TOKEN}, {"http_status": "502"}):
            with self.subTest(fields=list(fields)):
                with self.assertRaises(rollout.SafeError):
                    self.diagnostics.record("metadata", **fields)
        with mock.patch.object(rollout.remote, "atomic_private", side_effect=OSError("private exception detail")):
            self.assertFalse(self.diagnostics.record("metadata"))
        self.assertFalse(self.diagnostics.available)
        self.assertEqual(self.diagnostics.events, [])

    def test_runtime_mismatch_never_reaches_http_checks(self):
        for overrides, field in (({"enabled": "false"}, "scheduler_matches"),
                                 ({"author": "private-author"}, "author_matches"),
                                 ({"image": "sha256:" + "c" * 64}, "image_matches")):
            with self.subTest(field=field), mock.patch.object(rollout, "runtime_metadata", return_value=self.metadata(**overrides)), \
                    mock.patch.object(rollout.urllib.request, "urlopen") as opened, \
                    mock.patch.object(rollout.time, "monotonic", side_effect=[0, 100]):
                with self.assertRaises(rollout.SafeError):
                    rollout.healthy_backend(SHA, IMAGE, True, "author", self.diagnostics)
                opened.assert_not_called()
                self.assertFalse([event for event in self.report() if event["stage"] == "runtime_check"][-1][field])

    def test_metadata_inspection_timeout_keeps_original_abort_and_safe_stage(self):
        failure = subprocess.TimeoutExpired(["docker", PASSWORD], 30, output=TOKEN, stderr=WRITER)
        with mock.patch.object(rollout, "runtime_metadata", side_effect=failure), \
                mock.patch.object(rollout.time, "monotonic", return_value=0):
            with self.assertRaises(subprocess.TimeoutExpired):
                rollout.healthy_backend(SHA, IMAGE, True, "author", self.diagnostics)
        self.assertEqual(self.report()[-1], {"phase": "apply", "stage": "metadata", "exception_class": "timeout"})

    def test_untrusted_health_and_exception_messages_are_never_recorded(self):
        with mock.patch.object(rollout, "runtime_metadata", return_value=self.metadata(health=PASSWORD)), \
                mock.patch.object(rollout.time, "monotonic", side_effect=[0, 100]):
            with self.assertRaises(rollout.SafeError):
                rollout.healthy_backend(SHA, IMAGE, True, "author", self.diagnostics)
        self.assertEqual([event for event in self.report() if event["stage"] == "runtime_check"][-1]["docker_health"], "unknown")
        with mock.patch.object(rollout, "runtime_metadata", side_effect=OSError("private exception detail")), \
                mock.patch.object(rollout.time, "monotonic", side_effect=[0, 100]):
            with self.assertRaises(rollout.SafeError):
                rollout.healthy_backend(SHA, IMAGE, True, "author", self.diagnostics)
        self.assertIn({"phase": "apply", "stage": "metadata", "exception_class": "os_error"}, self.report())


class RunnerTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.app = Path(self.temp.name)
        self.directory = self.app / "run"
        self.directory.mkdir(mode=0o700)
        (self.app / "ops" / "notion").mkdir(parents=True)
        (self.app / "ops").chmod(0o700)
        (self.app / "ops" / "notion").chmod(0o700)
        self.database = {"MYSQL_PASSWORD": PASSWORD, "MYSQL_DATABASE": "test"}
        self.runtime = {"MINIBLOG_NOTION_SYNC_ENABLED": "false", "MINIBLOG_NOTION_TOKEN": TOKEN}

    def runner(self, action="catalog_prepare", value=None, writer=""):
        return rollout.Runner(self.app, self.directory, envelope(action, value), IMAGE, self.database, TOKEN, writer)

    def fake_task(self, calls, fail=None):
        def run(image, directory, env_file, binary, args, label, network):
            mode = args[args.index("--mode") + 1]
            fields = env_file.read_text()
            calls.append((mode, args, fields))
            self.assertNotIn(TOKEN, " ".join(args))
            self.assertNotIn(WRITER, " ".join(args))
            self.assertNotIn(PASSWORD, " ".join(args))
            if mode == fail:
                raise rollout.SafeError("private task failed")
            result = {"drain_status": drain(), "status": status(), "author_resolve": {"author": "昵称"}}.get(mode, {"completed": True})
            (directory / (label + ".json")).write_text(json.dumps(result))
            (directory / (label + ".json")).chmod(0o600)
        return run

    def test_writer_is_only_in_apply_container_never_status_or_argv_and_removed_on_failure(self):
        calls = []
        runner = self.runner("bootstrap_apply", {"confirmations": [{"page_id": "a" * 32}]}, WRITER)
        with mock.patch.object(rollout.remote, "docker_task", side_effect=self.fake_task(calls, "bootstrap_apply")), \
                mock.patch.object(rollout, "no_active_tasks"):
            with self.assertRaises(rollout.SafeError):
                runner.execute(self.runtime)
        self.assertEqual([mode for mode, _, _ in calls], ["drain_status", "status", "bootstrap_apply"])
        for mode, args, fields in calls:
            self.assertEqual(WRITER in fields, mode == "bootstrap_apply")
            self.assertEqual(TOKEN in fields, mode == "bootstrap_apply")
        self.assertEqual(list(self.directory.glob("*.env")), [])
        self.assertIn("--allow-notion-write", calls[-1][1])
        self.assertEqual(json.loads((self.directory / "confirmations.json").read_text()), {"confirmations": [{"page_id": "a" * 32}]})

    def test_source_revision_conflict_blocks_before_mutation(self):
        runner = self.runner("source_update", {"enabled": True})
        runner.task = mock.Mock(side_effect=[drain(), status(sources=[dict(row, config_revision=8) for row in status()["sources"]])])
        with mock.patch.object(rollout, "no_active_tasks"):
            with self.assertRaises(rollout.SafeError):
                runner.execute(self.runtime)
        self.assertEqual([call.args[0] for call in runner.task.call_args_list], ["drain_status", "status"])

    def test_resume_unknown_journal_blocks_but_scheduler_off_can_recover(self):
        for action in ("resume", "scheduler_on"):
            runner = self.runner(action, {"validated_sync_run_id": RUN} if action == "scheduler_on" else {})
            runner.task = mock.Mock(return_value=drain(unresolved_bootstrap_count=1))
            with self.assertRaises(rollout.SafeError):
                runner.execute(self.runtime)
        runner = self.runner("scheduler_off")
        runner.task = mock.Mock(side_effect=[drain(unresolved_bootstrap_count=1), status()])
        runner.scheduler = mock.Mock(return_value={"scheduler_enabled": False})
        with mock.patch.object(rollout, "no_active_tasks"):
            runner.execute(self.runtime)
        runner.scheduler.assert_called_once_with(False, self.runtime)

    def test_pause_control_is_first_then_grace_and_independent_task_drain(self):
        runner = self.runner("pause")
        runner.task = mock.Mock(side_effect=[{}, drain(baseline_frozen=False)])
        with mock.patch.object(rollout.time, "sleep") as sleep, mock.patch.object(rollout, "no_active_tasks") as inventory:
            result = runner.execute(self.runtime)
        self.assertEqual(runner.task.call_args_list[0].args[0], "control_update")
        sleep.assert_called_once_with(20)
        inventory.assert_called_once()
        self.assertEqual(result, {"paused": True, "source_writes_paused": True})
        self.assertEqual(json.loads(next(self.directory.glob("control-*.json")).read_text()), result)

    def test_catalog_reuse_uses_same_reviewed_maintenance_cli(self):
        runner = self.runner(value={"reuse_map": [{"option_id": "actual%id", "section_code": "concurrency&sync"}]})
        runner.task = mock.Mock(side_effect=[drain(), status(), {"items": []}])
        with mock.patch.object(rollout, "no_active_tasks"):
            runner.execute({"MINIBLOG_NOTION_SYNC_ENABLED": "true"})
        last = runner.task.call_args_list[-1]
        self.assertEqual(last.args[0], "catalog_prepare")
        self.assertIn("--catalog-plan", last.args[1])
        self.assertIn(SOURCE, last.args[1])

    def test_immediate_sync_requires_resumed_and_reports_partial_failure(self):
        runner = self.runner("sync")
        runner.task = mock.Mock(return_value=status())
        with self.assertRaises(rollout.SafeError):
            runner.execute(self.runtime)
        good = status(paused=False, source_writes_paused=False)
        runner.task = mock.Mock(side_effect=[good, drain(paused=False, source_writes_paused=False), {"author": "昵称"},
                                            {"run": {"status": "completed_with_errors", "counts": {"failed": 1, "blocked": 0}}}])
        with mock.patch.object(rollout, "no_active_tasks"):
            with self.assertRaises(rollout.SafeError):
                runner.execute(self.runtime)
        self.assertEqual(runner.task.call_args_list[-1].args, ("sync", ["--enable-sync", "--author", "昵称"]))

    def test_scheduler_env_only_changes_approved_fields_never_keeps_writer(self):
        original = "MYSQL_PASSWORD='keep'\nMINIBLOG_NOTION_TOKEN=old_reader\nMINIBLOG_CONTENT_REGISTER_ENABLED=true\nMINIBLOG_NOTION_SYNC_INTERVAL=5m\nMINIBLOG_NOTION_SYNC_ENABLED=false\nMINIBLOG_NOTION_SYNC_AUTHOR=old\nMINIBLOG_NOTION_BOOTSTRAP_TOKEN=writer\n"
        result = rollout.scheduler_content(original, True, "O'Brien $昵称")
        self.assertIn("MYSQL_PASSWORD='keep'", result)
        self.assertIn("MINIBLOG_CONTENT_REGISTER_ENABLED=true", result)
        self.assertIn("MINIBLOG_NOTION_SYNC_INTERVAL=5m", result)
        self.assertIn("MINIBLOG_NOTION_SYNC_ENABLED=true", result)
        self.assertIn("MINIBLOG_NOTION_SYNC_AUTHOR='O\\'Brien $昵称'", result)
        self.assertNotIn("BOOTSTRAP", result)

    def test_scheduler_author_uses_article_rune_limit(self):
        self.assertIn("MINIBLOG_NOTION_SYNC_AUTHOR=", rollout.scheduler_content("", True, "名" * 128))
        with self.assertRaises(rollout.SafeError):
            rollout.scheduler_content("", True, "名" * 129)

    def test_scheduler_restart_health_failure_recovers_off_env_without_external_retry(self):
        target = self.app / ".env"
        target.write_text("MYSQL_PASSWORD=" + PASSWORD + "\nMINIBLOG_NOTION_TOKEN=" + TOKEN + "\nMINIBLOG_NOTION_SYNC_ENABLED=false\n")
        target.chmod(0o600)
        runner = self.runner("scheduler_on", {"validated_sync_run_id": RUN})
        runner.task = mock.Mock(side_effect=[{"author": "昵称"}, drain(), status()])
        with mock.patch.object(rollout, "no_active_tasks"), mock.patch.object(rollout, "restart_backend") as restart, \
                mock.patch.object(rollout, "healthy_backend", side_effect=rollout.SafeError("unverified")):
            with self.assertRaises(rollout.SafeError):
                runner.scheduler(True, self.runtime)
        self.assertEqual(restart.call_count, 2)
        self.assertIn("MINIBLOG_NOTION_SYNC_ENABLED=false", target.read_text())
        self.assertIn("MYSQL_PASSWORD=" + PASSWORD, target.read_text())
        for path in (target, self.app / ".env.previous"):
            self.assertEqual(stat.S_IMODE(path.stat().st_mode), 0o600)
        self.assertEqual(runner.task.call_count, 3)
        self.assertFalse((self.directory / "runtime-env-before").exists())
        for path in self.directory.iterdir():
            if path.is_file():
                self.assertNotIn(TOKEN, path.read_text())
                self.assertNotIn(PASSWORD, path.read_text())

    def test_public_failure_rolls_back_and_verifies_local_off_without_claiming_public_success(self):
        target = self.app / ".env"
        target.write_text("MYSQL_PASSWORD=" + PASSWORD + "\nMINIBLOG_NOTION_TOKEN=" + TOKEN + "\nMINIBLOG_NOTION_SYNC_ENABLED=false\n")
        target.chmod(0o600)
        runner = self.runner("scheduler_on", {"validated_sync_run_id": RUN})
        runner.task = mock.Mock(side_effect=[{"author": "private-author"}, drain(), status()])
        row = {"health": "healthy", "running": True, "tag": rollout.IMAGE_REPOSITORY + SHA}
        before = (IMAGE, {"MINIBLOG_NOTION_SYNC_ENABLED": "true", "MINIBLOG_NOTION_SYNC_AUTHOR": "private-author"}, row)
        recovered = (IMAGE, {"MINIBLOG_NOTION_SYNC_ENABLED": "false", "MINIBLOG_NOTION_SYNC_AUTHOR": "private-author"}, row)
        response = mock.MagicMock()
        response.__enter__.return_value.status = 200
        failure = urllib.error.HTTPError("https://api.yangshujie.com/health", 502, "private exception detail", {}, None)
        with mock.patch.object(rollout, "no_active_tasks"), mock.patch.object(rollout, "restart_backend") as restart, \
                mock.patch.object(rollout, "runtime_metadata", side_effect=[before, recovered]), \
                mock.patch.object(rollout.urllib.request, "urlopen", side_effect=[response, failure, response]) as opened, \
                mock.patch.object(rollout.time, "monotonic", side_effect=[0, 100, 200]):
            with self.assertRaises(rollout.SafeError):
                runner.scheduler(True, self.runtime)
        self.assertEqual(restart.call_count, 2)
        self.assertEqual(runner.task.call_count, 3)
        self.assertIn("MINIBLOG_NOTION_SYNC_ENABLED=false", target.read_text())
        self.assertEqual([call.args[0] for call in opened.call_args_list],
                         ["http://127.0.0.1:8090/health", "https://api.yangshujie.com/health", "http://127.0.0.1:8090/health"])
        path = self.directory / "scheduler-diagnostics.json"
        self.assertEqual(stat.S_IMODE(path.stat().st_mode), 0o600)
        text = path.read_text()
        for value in (PASSWORD, TOKEN, WRITER, "private-author", "private exception detail"):
            self.assertNotIn(value, text)
        events = json.loads(text)["events"]
        self.assertIn({"phase": "apply", "stage": "http_public", "http_status": 502, "exception_class": "http_error"}, events)
        self.assertIn({"phase": "rollback", "stage": "rollback_local_verified"}, events)
        self.assertEqual(events[-1]["stage"], "rollback_verified")
        self.assertFalse(any(event["stage"] in {"health_verified", "operation_completed"} for event in events))

    def test_failed_rollback_health_is_recorded_without_suppression_of_original_failure(self):
        target = self.app / ".env"
        target.write_text("MINIBLOG_NOTION_SYNC_ENABLED=false\n")
        target.chmod(0o600)
        runner = self.runner("scheduler_on", {"validated_sync_run_id": RUN})
        runner.task = mock.Mock(side_effect=[{"author": "private-author"}, drain(), status()])
        with mock.patch.object(rollout, "no_active_tasks"), mock.patch.object(rollout, "restart_backend") as restart, \
                mock.patch.object(rollout, "healthy_backend", side_effect=rollout.SafeError("private exception detail")) as health:
            with self.assertRaises(rollout.SafeError):
                runner.scheduler(True, self.runtime)
        self.assertEqual(restart.call_count, 2)
        self.assertEqual(health.call_count, 2)
        self.assertEqual(health.call_args.args[2], False)
        self.assertEqual(health.call_args.kwargs, {"phase": "rollback", "check_public": False})
        events = json.loads((self.directory / "scheduler-diagnostics.json").read_text())["events"]
        self.assertEqual(events[-1], {"phase": "rollback", "stage": "rollback_failed", "exception_class": "validation_error"})
        self.assertFalse(any(event["stage"] == "rollback_verified" for event in events))

    def test_transient_diagnostic_failure_never_records_completed_and_always_rolls_back(self):
        for failed_stage in ("runtime_check", "operation_completed"):
            with self.subTest(failed_stage=failed_stage):
                target = self.app / ".env"
                target.write_text("MINIBLOG_NOTION_SYNC_ENABLED=false\n")
                target.chmod(0o600)
                runner = self.runner("scheduler_on", {"validated_sync_run_id": RUN})
                runner.task = mock.Mock(side_effect=[{"author": "private-author"}, drain(), status(), drain(), status()])
                row = {"health": "healthy", "running": True, "tag": rollout.IMAGE_REPOSITORY + SHA}
                metadata = [(IMAGE, {"MINIBLOG_NOTION_SYNC_ENABLED": switch, "MINIBLOG_NOTION_SYNC_AUTHOR": "private-author"}, row)
                            for switch in ("true", "false")]
                response = mock.MagicMock()
                response.__enter__.return_value.status = 200
                write = rollout.remote.atomic_private
                failed = []
                def fail_once(path, value):
                    if Path(path).name == "scheduler-diagnostics.json" and not failed and json.loads(value)["events"][-1]["stage"] == failed_stage:
                        failed.append(True)
                        raise OSError("private exception detail")
                    return write(path, value)
                with mock.patch.object(rollout, "no_active_tasks"), mock.patch.object(rollout, "restart_backend") as restart, \
                        mock.patch.object(rollout.remote, "atomic_private", side_effect=fail_once), \
                        mock.patch.object(rollout, "runtime_metadata", side_effect=metadata), \
                        mock.patch.object(rollout.urllib.request, "urlopen", return_value=response), \
                        mock.patch.object(rollout.time, "monotonic", side_effect=[0, 100]):
                    with self.assertRaises(rollout.SafeError):
                        runner.scheduler(True, self.runtime)
                self.assertTrue(failed)
                self.assertEqual(restart.call_count, 2)
                self.assertIn("MINIBLOG_NOTION_SYNC_ENABLED=false", target.read_text())
                events = json.loads((self.directory / "scheduler-diagnostics.json").read_text())["events"]
                self.assertFalse(any(event["stage"] == "operation_completed" for event in events))
                self.assertEqual(events[-1]["stage"], "rollback_verified")

    def test_cancellation_during_failure_diagnostic_cannot_skip_off_rollback(self):
        target = self.app / ".env"
        target.write_text("MINIBLOG_NOTION_SYNC_ENABLED=false\n")
        target.chmod(0o600)
        runner = self.runner("scheduler_on", {"validated_sync_run_id": RUN})
        runner.task = mock.Mock(side_effect=[{"author": "private-author"}, drain(), status()])
        write = rollout.remote.atomic_private
        def interrupt_failed_trace(path, value):
            if Path(path).name == "scheduler-diagnostics.json" and json.loads(value)["events"][-1]["stage"] == "operation_failed":
                raise KeyboardInterrupt()
            return write(path, value)
        with mock.patch.object(rollout, "no_active_tasks"), mock.patch.object(rollout, "restart_backend") as restart, \
                mock.patch.object(rollout, "healthy_backend", side_effect=[rollout.SafeError("unverified"), None]), \
                mock.patch.object(rollout.remote, "atomic_private", side_effect=interrupt_failed_trace):
            with self.assertRaises(rollout.SafeError):
                runner.scheduler(True, self.runtime)
        self.assertEqual(restart.call_count, 2)
        self.assertIn("MINIBLOG_NOTION_SYNC_ENABLED=false", target.read_text())
        events = json.loads((self.directory / "scheduler-diagnostics.json").read_text())["events"]
        self.assertEqual(events[-1]["stage"], "rollback_verified")
        self.assertFalse(any(event["stage"] == "operation_completed" for event in events))

    def test_scheduler_on_remains_paused_and_requires_run_evidence(self):
        now = datetime.now(timezone.utc)
        sources = status()
        for row in sources["sources"]:
            row["last_success_at"] = now.isoformat()
        run = {"run_id": RUN, "mode": "sync", "status": "completed", "counts": {"failed": 0, "blocked": 0},
               "started_at": (now - timedelta(seconds=1)).isoformat(), "finished_at": now.isoformat()}
        runner = self.runner("scheduler_on", {"validated_sync_run_id": RUN})
        rollout.sync_receipt(self.app, RUN, {"version": 1, "run_id": RUN, "source_id": SOURCE, "config_revision": 7,
                            "image_id": IMAGE, "expected_image_sha": SHA, "reader_sha256": hashlib.sha256(TOKEN.encode()).hexdigest(), "started_at": run["started_at"],
                            "finished_at": run["finished_at"], "last_success_at": now.isoformat()})
        runner.task = mock.Mock(side_effect=[drain(), sources, run])
        runner.scheduler = mock.Mock(return_value={"scheduler_enabled": True})
        with mock.patch.object(rollout, "no_active_tasks"):
            runner.execute(self.runtime)
        self.assertEqual([call.args[0] for call in runner.task.call_args_list], ["drain_status", "status", "run_status"])
        runner.scheduler.assert_called_once_with(True, self.runtime)


    def test_success_receipt_is_immutable_private_and_invalid_after_config_change(self):
        receipt = {"version": 1, "run_id": RUN, "source_id": SOURCE, "config_revision": 7,
                   "image_id": IMAGE, "expected_image_sha": SHA, "reader_sha256": hashlib.sha256(TOKEN.encode()).hexdigest(), "started_at": "start", "finished_at": "finish", "last_success_at": "success"}
        rollout.sync_receipt(self.app, RUN, receipt)
        self.assertEqual(rollout.sync_receipt(self.app, RUN), receipt)
        with self.assertRaises(rollout.SafeError):
            rollout.sync_receipt(self.app, RUN, dict(receipt, config_revision=99))
        run = {"run_id": RUN, "started_at": "start", "finished_at": "finish"}
        source = {"last_success_at": "success"}
        rollout.validate_sync_receipt(receipt, run, source, envelope("scheduler_on"), IMAGE, TOKEN)
        changed = envelope("scheduler_on")
        changed["expected_config_revision"] = 99
        with self.assertRaises(rollout.SafeError):
            rollout.validate_sync_receipt(receipt, run, source, changed, IMAGE, TOKEN)
        path = self.app / "ops" / "notion" / "sync-receipts" / (RUN + ".json")
        self.assertEqual(stat.S_IMODE(path.stat().st_mode), 0o600)
        path.chmod(0o644)
        with self.assertRaises(rollout.SafeError):
            rollout.sync_receipt(self.app, RUN)

    def test_rotated_reader_or_resident_empty_cannot_reuse_old_sync_success(self):
        receipt = {"version": 1, "run_id": RUN, "source_id": SOURCE, "config_revision": 7,
                   "image_id": IMAGE, "expected_image_sha": SHA, "reader_sha256": hashlib.sha256(TOKEN.encode()).hexdigest(),
                   "started_at": "start", "finished_at": "finish", "last_success_at": "success"}
        with self.assertRaises(rollout.SafeError):
            rollout.validate_sync_receipt(receipt, {"run_id": RUN, "started_at": "start", "finished_at": "finish"},
                                          {"last_success_at": "success"}, envelope("scheduler_on"), IMAGE, TOKEN + "rotated")
        runner = self.runner("scheduler_on", {"validated_sync_run_id": RUN})
        runner.task = mock.Mock(side_effect=[drain(), status()])
        with mock.patch.object(rollout, "no_active_tasks"):
            with self.assertRaises(rollout.SafeError):
                runner.execute({"MINIBLOG_NOTION_SYNC_ENABLED": "false", "MINIBLOG_NOTION_TOKEN": ""})
        self.assertEqual(runner.task.call_count, 2)

    def test_unknown_journal_recovery_only_approved_unresolved_pages(self):
        page = "a" * 32
        evidence = drain(unresolved_bootstrap_count=1, unresolved_bootstrap_page_ids=[page])
        rollout.validate_recovery_scope(evidence, [{"page_id": page}])
        for confirmations in ([{"page_id": "b" * 32}], [{"page_id": page}, {"page_id": "b" * 32}], [{"page_id": page}, {"page_id": page}]):
            with self.assertRaises(rollout.SafeError):
                rollout.validate_recovery_scope(evidence, confirmations)
        with self.assertRaises(rollout.SafeError):
            rollout.validate_recovery_scope(drain(unresolved_bootstrap_count=1), [{"page_id": page}])
        runner = self.runner("bootstrap_apply", {"confirmations": [{"page_id": page}]}, WRITER)
        runner.task = mock.Mock(side_effect=[evidence, status(), {"items": [{"outcome": "adopted"}]}])
        with mock.patch.object(rollout, "no_active_tasks"):
            runner.execute(self.runtime)
        self.assertEqual(runner.task.call_args_list[-1].args[0], "bootstrap_apply")

    def test_config_mutations_unknown_journal_stop_before_task_but_preview_allowed(self):
        for action, value in (("catalog_prepare", {}), ("source_update", {"enabled": True}),
                              ("catalog_bind", {"binding_id": "1", "section_code": "a"}),
                              ("catalog_activate", {"items": []})):
            runner = self.runner(action, value)
            runner.task = mock.Mock(return_value=drain(unresolved_bootstrap_count=1))
            with self.assertRaises(rollout.SafeError):
                runner.execute(self.runtime)
            self.assertEqual(runner.task.call_count, 1)
        runner = self.runner("bootstrap_preview")
        runner.task = mock.Mock(side_effect=[drain(unresolved_bootstrap_count=1), status(), {}])
        with mock.patch.object(rollout, "no_active_tasks"):
            runner.execute(self.runtime)
        self.assertEqual(runner.task.call_args_list[-1].args[0], "bootstrap_preview")

    def test_on_config_changed_during_restart_restores_off(self):
        target = self.app / ".env"
        target.write_text("MINIBLOG_NOTION_SYNC_ENABLED=false\n")
        target.chmod(0o600)
        runner = self.runner("scheduler_on", {"validated_sync_run_id": RUN})
        changed = status(sources=[dict(row, config_revision=8) for row in status()["sources"]])
        runner.task = mock.Mock(side_effect=[{"author": "昵称"}, drain(), status(), drain(), changed])
        with mock.patch.object(rollout, "no_active_tasks"), mock.patch.object(rollout, "restart_backend") as restart, \
                mock.patch.object(rollout, "healthy_backend"):
            with self.assertRaises(rollout.SafeError):
                runner.scheduler(True, self.runtime)
        self.assertEqual(restart.call_count, 2)
        self.assertIn("MINIBLOG_NOTION_SYNC_ENABLED=false", target.read_text())


class TransportRolloutTests(unittest.TestCase):
    def env(self, action):
        return {"GITHUB_REF": "refs/heads/main", "GITHUB_RUN_ID": "1234", "GITHUB_RUN_ATTEMPT": "1",
                "SVRD_HOST": "example.test", "SVRD_USER": "operator", "SVRD_SSH_KEY": "fake",
                "MINIBLOG_NOTION_TOKEN": TOKEN, "MINIBLOG_NOTION_BOOTSTRAP_TOKEN": WRITER if action == "bootstrap_apply" else "",
                "NOTION_ROLLOUT_ACTION": action, "NOTION_MANIFEST_ID": "go-review", "NOTION_MANIFEST_SHA256": "a" * 64,
                "NOTION_EXPECTED_IMAGE_SHA": SHA, "NOTION_EXPECTED_CONFIG_REVISION": "7"}

    def test_writer_private_file_transport_never_ssh_argument_and_finally_cleanup(self):
        calls, files = [], []
        def command(args, **kwargs):
            calls.append(args)
            if args[0] == "scp":
                path = Path(args[-2])
                files.append((path.name, path.read_text(), stat.S_IMODE(path.stat().st_mode)))
            return subprocess.CompletedProcess(args, 0, b"", b"")
        with mock.patch.dict(os.environ, self.env("bootstrap_apply"), clear=True), \
                mock.patch.object(sys, "argv", ["transport.py", "--mode", "rollout"]), \
                mock.patch.object(transport, "write_validated_key"), \
                mock.patch.object(transport.subprocess, "run", side_effect=command), \
                contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
            self.assertEqual(transport.main(), 0)
        self.assertEqual(next(content for name, content, _ in files if name == "writer"), WRITER)
        self.assertTrue(all(mode == 0o600 for _, _, mode in files))
        self.assertNotIn(WRITER, " ".join(" ".join(args) for args in calls))
        self.assertNotIn(TOKEN, " ".join(" ".join(args) for args in calls))
        self.assertIn("--writer /tmp/miniblog-notion-rollout-1234-1/writer", next(args[-1] for args in calls if args[0] == "ssh" and "/rollout.py" in args[-1]))
        self.assertIn("shutil.rmtree", calls[-1][-1])

    def test_writer_rejected_in_non_bootstrap_job_before_any_network(self):
        value = self.env("sync")
        value["MINIBLOG_NOTION_BOOTSTRAP_TOKEN"] = WRITER
        with mock.patch.dict(os.environ, value, clear=True), mock.patch.object(sys, "argv", ["transport.py", "--mode", "rollout"]), \
                mock.patch.object(transport.subprocess, "run") as process, contextlib.redirect_stderr(io.StringIO()) as log:
            self.assertEqual(transport.main(), 1)
        process.assert_not_called()
        self.assertNotIn(WRITER, log.getvalue())

    def test_cleanup_removes_only_receipt_owner_and_private_env_preserves_evidence(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary) / "ops" / "notion" / "rollout-1234-1"
            root.mkdir(parents=True)
            for path in (root, root.parent, root.parent.parent):
                path.chmod(0o700)
            for path, value in ((root / "operation.json", {"run_id": "1234-1"}),
                                (root / "container-apply.json", {"name": "miniblog-notion-ops-rollout-1234-1-apply", "owner": "a" * 32})):
                path.write_text(json.dumps(value))
                path.chmod(0o600)
            secret = root / "task.env"
            secret.write_text("MINIBLOG_NOTION_BOOTSTRAP_TOKEN=" + WRITER)
            secret.chmod(0o600)
            code = transport.ROLLOUT_CLEANUP_CODE.replace("/opt/miniblog/ops/notion/rollout-", str(root.parent / "rollout-"))
            calls = []
            def command(args, **kwargs):
                calls.append(args)
                return subprocess.CompletedProcess(args, 0, ("a" * 32 + "\n") if "inspect" in args else "")
            with mock.patch.object(sys, "argv", ["cleanup", "1234-1"]), \
                    mock.patch.object(subprocess, "run", side_effect=command):
                exec(compile(code, "cleanup", "exec"), {})
            self.assertTrue(any(args[:3] == ["docker", "rm", "-f"] for args in calls))
            self.assertFalse(secret.exists())
            self.assertTrue((root / "operation.json").exists())
            calls.clear()
            def different_owner(args, **kwargs):
                calls.append(args)
                return subprocess.CompletedProcess(args, 0, "b" * 32 + "\n")
            with mock.patch.object(sys, "argv", ["cleanup", "1234-1"]), \
                    mock.patch.object(subprocess, "run", side_effect=different_owner):
                with self.assertRaises(SystemExit) as failure:
                    exec(compile(code, "cleanup", "exec"), {})
            self.assertEqual(failure.exception.code, 1)
            self.assertFalse(any("rm" in args for args in calls))

    def test_ssh_failure_cleans_owned_stage_and_keeps_logs_secret_free(self):
        calls = []
        def command(args, **kwargs):
            calls.append(args)
            failed = args[0] == "ssh" and "/rollout.py --app-dir" in args[-1]
            return subprocess.CompletedProcess(args, 1 if failed else 0, b"", (WRITER + TOKEN + PASSWORD).encode() if failed else b"")
        with mock.patch.dict(os.environ, self.env("bootstrap_apply"), clear=True), \
                mock.patch.object(sys, "argv", ["transport.py", "--mode", "rollout"]), \
                mock.patch.object(transport, "write_validated_key"), \
                mock.patch.object(transport.subprocess, "run", side_effect=command), \
                contextlib.redirect_stdout(io.StringIO()) as output, contextlib.redirect_stderr(io.StringIO()) as errors:
            self.assertEqual(transport.main(), 1)
        self.assertIn("shutil.rmtree", calls[-1][-1])
        for secret in (WRITER, TOKEN, PASSWORD):
            self.assertNotIn(secret, output.getvalue() + errors.getvalue())

    def test_workflow_has_dedicated_writer_job_and_no_artifact_report_upload(self):
        content = (HERE.parent.parent / ".github" / "workflows" / "notion-rollout.yml").read_text()
        regular, writer = content.split("  bootstrap-apply:", 1)
        self.assertNotIn("MINIBLOG_NOTION_BOOTSTRAP_TOKEN", regular)
        self.assertEqual(writer.count("secrets.MINIBLOG_NOTION_BOOTSTRAP_TOKEN"), 1)
        self.assertNotIn("upload-artifact", content)
        self.assertIn("group: miniblog-prod", content)
        self.assertIn("github.ref == 'refs/heads/main'", content)


if __name__ == "__main__":
    unittest.main()
