import contextlib
from datetime import datetime, timedelta, timezone
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import stat
import subprocess
import sys
import tempfile
import unittest
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

    def test_scheduler_restart_health_failure_recovers_off_env_without_external_retry(self):
        target = self.app / ".env"
        target.write_text("MYSQL_PASSWORD=keep\nMINIBLOG_NOTION_SYNC_ENABLED=false\n")
        target.chmod(0o600)
        runner = self.runner("scheduler_on", {"validated_sync_run_id": RUN})
        runner.task = mock.Mock(side_effect=[{"author": "昵称"}, drain(), status()])
        with mock.patch.object(rollout, "no_active_tasks"), mock.patch.object(rollout, "restart_backend") as restart, \
                mock.patch.object(rollout, "healthy_backend", side_effect=rollout.SafeError("unverified")):
            with self.assertRaises(rollout.SafeError):
                runner.scheduler(True, self.runtime)
        self.assertEqual(restart.call_count, 2)
        self.assertIn("MINIBLOG_NOTION_SYNC_ENABLED=false", target.read_text())
        self.assertIn("MYSQL_PASSWORD=keep", target.read_text())
        for path in (target, self.app / ".env.previous", self.directory / "runtime-env-before"):
            self.assertEqual(stat.S_IMODE(path.stat().st_mode), 0o600)
        self.assertEqual(runner.task.call_count, 3)

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
