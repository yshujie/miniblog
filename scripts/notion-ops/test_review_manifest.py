"""Only synthetic records; no private review files or external services are used."""
import ast
import contextlib
from datetime import datetime, timedelta, timezone
import copy
import io
import json
import os
import re
from pathlib import Path
import tempfile
import unittest
from unittest import mock

import review_manifest as compiler

SOURCE = "2bf330bd-ddf1-80a6-aa49-000bbd1e154b"
OTHER_SOURCE = "d579f619-4f1c-4e7e-9593-ab6529fc63d0"
IMAGE = "a" * 40


def fixture(count=38):
    approved = {"source_id": SOURCE, "reviewer": "synthetic reviewer", "entries": []}
    latest = {"run_id": "12345678-1234-1234-1234-123456789abc", "run": {"run_id": "12345678-1234-1234-1234-123456789abc", "mode": "bootstrap_preview", "status": "completed", "started_at": "2026-10-08T23:59:00Z", "finished_at": "2026-10-09T00:00:00Z", "counts": {"seen": count, "failed": 0, "blocked": 0}}, "items": [], "run_items": []}
    for index in range(count):
        page = f"{index + 1:032x}"
        published = index < 14
        snap = {"page_id": page, "source_id": SOURCE, "title": f"PRIVATE_TITLE_{index}", "tags": ["synthetic"], "topic_option_id": "option", "topic_option_name": "synthetic topic", "desired_state": 0, "state_option_id": "", "page_url": f"https://www.notion.so/{page}", "public_url": f"https://synthetic.invalid/{page}", "native_archived": False, "in_trash": False, "last_edited_at": "old", "difficulty": None, "reason": ""}
        entry = {"page_id": page, "expected_state": "published" if published else "draft", "snapshot": copy.deepcopy(snap), "allow_legacy_alias": False}
        candidate = {"page_id": page, "source_id": SOURCE, "title": snap["title"], "topic": snap["topic_option_name"], "notion_state": "", "new_page": not published, "requires_legacy_alias": False, "match_method": "trusted_page_id" if published else "unmatched", "public_condition": "public_url_available", "proposed_section_code": "section", "expected_fingerprint": f"{index + 1:064x}", "candidate_article_ids": [], "title_hint_article_ids": []}
        if published:
            article = str(9007199254741993 + index)
            local = {"article_id": article, "local_state": "published", "local_title": f"synthetic local {index}", "local_section_code": "historical-section"}
            entry.update(article_id=article, local=copy.deepcopy(local))
            candidate.update(local)
            candidate["candidate_article_ids"] = [article]
        else:
            entry["new_page"] = True
        approved["entries"].append(entry)
        latest["items"].append(candidate)
        latest["run_items"].append({"page_id": page, "outcome": "pending", "before": {"management_state": "baseline_pending"}, "after": snap})
        latest["run_items"].append({"page_id": page, "outcome": "bootstrap_preview", "after": {"bootstrap_preview": copy.deepcopy(candidate)}})
    return approved, latest


def compile_fixture(approved, latest, **kwargs):
    return compiler.compile_manifests(approved, latest, 9, IMAGE, **kwargs)


def update_candidate(latest, index, **fields):
    latest["items"][index].update(fields)
    latest["run_items"][index * 2 + 1]["after"]["bootstrap_preview"].update(fields)



class RunTimeTests(unittest.TestCase):
    def test_zero_through_nine_fractional_digits_remain_valid(self):
        for fraction in ("", "1", "12", "123", "1234", "12345", "123456", "1234567", "12345678", "123456789"):
            value = "2026-10-09T05:30:00" + ("." + fraction if fraction else "") + "Z"
            with self.subTest(fraction=fraction):
                self.assertTrue(compiler.valid_run_times({"started_at": value, "finished_at": value}))
                whole = compiler.run_instant_ns("2026-10-09T05:30:00Z")
                self.assertEqual(int(fraction.ljust(9, "0") or "0"), compiler.run_instant_ns(value) - whole)

    def test_offset_equivalence_leap_day_and_pre_epoch_are_exact(self):
        for first, second in (("2026-10-09T05:30:00.123456789Z", "2026-10-09T13:30:00.123456789+08:00"),
                              ("2026-10-09T05:30:00.1Z", "2026-10-09T00:00:00.1-05:30"),
                              ("2024-02-29T00:00:00Z", "2024-02-28T23:00:00-01:00")):
            self.assertEqual(compiler.run_instant_ns(first), compiler.run_instant_ns(second))
            self.assertTrue(compiler.valid_run_times({"started_at": first, "finished_at": second}))
        self.assertEqual(-1, compiler.run_instant_ns("1969-12-31T23:59:59.999999999Z"))

    def test_one_nanosecond_reverse_order_cannot_compile_approval(self):
        approved, latest = fixture()
        latest["run"].update(started_at="2026-10-09T05:30:00.123456789Z", finished_at="2026-10-09T05:30:00.123456788Z")
        self.assertFalse(compiler.valid_run_times(latest["run"]))
        with self.assertRaises(compiler.SafeError) as caught:
            compile_fixture(approved, latest)
        self.assertEqual("incomplete_preview_run", caught.exception.category)
        latest["run"]["finished_at"] = "2026-10-09T13:30:00.123456790+08:00"
        self.assertEqual(38, sum(len(batch["input"]["confirmations"]) for batch in compile_fixture(approved, latest).values()))

    def test_malformed_dates_zones_precision_and_missing_values_are_rejected(self):
        values = [None, 1, True, "", "PRIVATE_TIMESTAMP", "2026-10-09", "2026-10-09T05:30:00", "2026-10-09 05:30:00Z",
                  "2026-10-09T05:30:00z", "2026-10-09T05:30:00.Z", "2026-10-09T05:30:00.1234567890Z",
                  "2026-10-09T05:30:00,123Z", "2026-10-09T05:30:00+0800", "2026-10-09T05:30:00+08:00:00",
                  "2026-10-09T05:30:00+24:00", "2026-10-09T05:30:00+00:60", "2026-02-29T05:30:00Z",
                  "2026-13-01T05:30:00Z", "2026-10-09T24:00:00Z", "2026-10-09T05:30:60Z", "0000-10-09T05:30:00Z",
                  "2026-10-09T05:30:00Z\n"]
        for value in values:
            with self.subTest(value=value):
                self.assertFalse(compiler.valid_run_times({"started_at": value, "finished_at": "2026-10-09T06:00:00Z"}))
        self.assertFalse(compiler.valid_run_times({}))

    def test_python310_fraction_contract_and_syntax(self):
        class Python310DateTime(datetime):
            @classmethod
            def fromisoformat(cls, value):
                fraction = re.search(r"\.([0-9]+)", value)
                if fraction and len(fraction[1]) not in (3, 6):
                    raise ValueError("Python 3.10 fractional precision contract")
                return datetime.fromisoformat(value)
        # Explicitly demonstrate the historical contract rejects Go's 1/2/4/5/7/8/9 digits.
        for digits in (1, 2, 4, 5, 7, 8, 9):
            value = "2026-10-09T05:30:00." + "1" * digits + "+00:00"
            with self.assertRaises(ValueError):
                Python310DateTime.fromisoformat(value)
        with mock.patch.object(compiler, "datetime", Python310DateTime):
            for digits in range(1, 10):
                value = "2026-10-09T05:30:00." + "1" * digits + "Z"
                self.assertTrue(compiler.valid_run_times({"started_at": value, "finished_at": value}))
            self.assertFalse(compiler.valid_run_times({"started_at": "2026-10-09T05:30:00.000000002Z", "finished_at": "2026-10-09T05:30:00.000000001Z"}))
        ast.parse(Path(compiler.__file__).read_text(), feature_version=(3, 10))

class ManifestTests(unittest.TestCase):
    def assert_drift(self, approved, latest, field=None):
        with self.assertRaises(compiler.SafeError) as caught:
            compile_fixture(approved, latest)
        report = caught.exception.report()
        self.assertEqual(report["error"], "review_drift")
        self.assertNotIn("PRIVATE_TITLE", json.dumps(report))
        if field:
            self.assertIn(field, [item for fields in report["fields"].values() for item in fields])
        return report

    def test_exact_38_14_published_24_draft_and_large_ids(self):
        approved, latest = fixture()
        result = compile_fixture(approved, latest, expected_publish_count=14, expected_draft_count=24)
        self.assertEqual(set(result), {"published", "draft"})
        self.assertEqual(len(result["published"]["input"]["confirmations"]), 14)
        self.assertEqual(len(result["draft"]["input"]["confirmations"]), 24)
        first = result["published"]["input"]["confirmations"][0]
        self.assertEqual(first["article_id"], "9007199254741993")
        self.assertEqual(first["confirmed_by"], approved["reviewer"])
        self.assertEqual(set(result["draft"]), {"version", "mode", "source_id", "expected_config_revision", "expected_image_sha", "input"})
        self.assertEqual(result["draft"]["input"]["confirmations"][0]["article_id"], "")

    def test_current_combined_audit_contract_checks_all_38_fresh_snapshots(self):
        approved, latest = fixture()
        combined = []
        for index in range(38):
            scan, final = latest["run_items"][index * 2:index * 2 + 2]
            final["before"] = scan["before"]
            final["after"]["snapshot"] = scan["after"]
            combined.append(final)
        latest["run_items"] = combined
        result = compile_fixture(approved, latest, expected_publish_count=14, expected_draft_count=24)
        self.assertEqual(sum(len(batch["input"]["confirmations"]) for batch in result.values()), 38)
        del latest["run_items"][0]["after"]["snapshot"]
        self.assert_drift(approved, latest, "snapshot.missing_or_invalid")
        latest["run_items"][0]["after"]["snapshot"] = approved["entries"][0]["snapshot"]
        del latest["run_items"][0]["before"]
        self.assert_drift(approved, latest, "management_state.not_baseline_pending")

    def test_same_historical_article_cannot_be_confirmed_for_two_pages(self):
        approved, latest = fixture()
        approved["entries"][1]["article_id"] = approved["entries"][0]["article_id"]
        report = self.assert_drift(approved, latest, "article_id.duplicate")
        self.assertEqual(len(report["affectedPageIDs"]), 2)

    def test_new_config_fingerprint_and_readonly_fields_are_allowed(self):
        approved, latest = fixture()
        update_candidate(latest, 0, expected_fingerprint="b" * 64, proposed_section_code="explicitly-reused-section")
        latest["run_items"][0]["after"].update(last_edited_at="new", difficulty="new difficulty", description="readonly change")
        result = compiler.compile_manifests(approved, latest, 10, "b" * 40)
        self.assertEqual(result["published"]["expected_config_revision"], 10)
        self.assertEqual(result["published"]["input"]["confirmations"][0]["expected_fingerprint"], "b" * 64)

    def test_every_blog_snapshot_drift_is_rejected_without_printing_values(self):
        values = {"title": "PRIVATE_NEW_TITLE", "tags": ["PRIVATE_TAG"], "source_id": OTHER_SOURCE, "page_id": "f" * 32,
                  "topic_option_id": "PRIVATE_OPTION", "topic_option_name": "PRIVATE_TOPIC", "page_url": "https://private.invalid/page",
                  "public_url": None, "desired_state": 2, "state_option_id": "PRIVATE_STATE", "native_archived": True, "in_trash": True}
        for field, value in values.items():
            with self.subTest(field=field):
                approved, latest = fixture()
                latest["run_items"][0]["after"][field] = value
                self.assert_drift(approved, latest, "snapshot." + field)

    def test_missing_snapshot_fields_fail_closed(self):
        for field in compiler.SNAPSHOT_FIELDS:
            with self.subTest(field=field):
                approved, latest = fixture()
                del latest["run_items"][0]["after"][field]
                self.assert_drift(approved, latest, "snapshot.missing_or_invalid")

    def test_article_identity_status_and_local_fields_must_remain_approved(self):
        for field, value in (("article_id", "9007199254742999"), ("local_state", "draft"), ("local_title", "PRIVATE_CHANGED"), ("local_section_code", "changed")):
            with self.subTest(field=field):
                approved, latest = fixture()
                update_candidate(latest, 0, **{field: value})
                self.assert_drift(approved, latest)
        approved, latest = fixture()
        update_candidate(latest, 14, new_page=False, article_id="123")
        self.assert_drift(approved, latest, "article_identity")

    def test_rich_protected_local_fields_require_an_independent_private_snapshot(self):
        approved, latest = fixture()
        approved["entries"][0]["local"].update(author="PRIVATE_AUTHOR", content="PRIVATE_BODY", external_link="https://private.invalid/original", pos=7)
        self.assert_drift(approved, latest, "local.content")
        local = {approved["entries"][0]["article_id"]: copy.deepcopy(approved["entries"][0]["local"])}
        self.assertTrue(compile_fixture(approved, latest, latest_local=local))
        local[approved["entries"][0]["article_id"]]["pos"] = 8
        with self.assertRaises(compiler.SafeError):
            compile_fixture(approved, latest, latest_local=local)

    def test_added_removed_duplicate_baseline_pages_reject_approval_expansion(self):
        approved, latest = fixture()
        extra_approved, extra_latest = fixture(39)
        latest["items"].append(extra_latest["items"][-1])
        latest["run_items"].extend(extra_latest["run_items"][-2:])
        latest["run"]["counts"]["seen"] = 39
        self.assert_drift(approved, latest, "scope.added_baseline_page")
        approved, latest = fixture()
        latest["items"].pop()
        self.assert_drift(approved, latest, "scope.missing_or_moved_page")
        for which in ("approved", "candidate", "observation", "saved"):
            with self.subTest(which=which):
                approved, latest = fixture()
                if which == "approved":
                    approved["entries"].append(copy.deepcopy(approved["entries"][0]))
                elif which == "candidate":
                    latest["items"].append(copy.deepcopy(latest["items"][0]))
                else:
                    latest["run_items"].append(copy.deepcopy(latest["run_items"][0 if which == "observation" else 1]))
                self.assert_drift(approved, latest)

    def test_nonbaseline_new_observation_is_not_frozen_or_confirmed(self):
        approved, latest = fixture()
        extra = copy.deepcopy(latest["run_items"][28])
        extra["page_id"] = "f" * 32
        extra["after"]["page_id"] = extra["page_id"]
        extra["before"] = {}
        latest["run_items"].append(extra)
        latest["run"]["counts"]["seen"] = 39
        result = compile_fixture(approved, latest)
        pages = [row["page_id"] for batch in result.values() for row in batch["input"]["confirmations"]]
        self.assertEqual(len(pages), 38)
        self.assertNotIn(extra["page_id"], pages)

    def test_other_libraries_are_never_auto_approved(self):
        approved, latest = fixture()
        _, other = fixture(1)
        page = "e" * 32
        other["items"][0].update(page_id=page, source_id=OTHER_SOURCE)
        other["run_items"][0].update(page_id=page)
        other["run_items"][0]["after"].update(page_id=page, source_id=OTHER_SOURCE)
        other["run_items"][1].update(page_id=page, after={"bootstrap_preview": copy.deepcopy(other["items"][0])})
        latest["items"].extend(other["items"])
        latest["run_items"].extend(other["run_items"])
        latest["run"]["counts"]["seen"] = 39
        self.assertTrue(all(batch["source_id"] == SOURCE for batch in compile_fixture(approved, latest).values()))

    def test_run_completion_and_full_audit_are_mandatory(self):
        for change in ({"started_at": None}, {"started_at": "2026-10-10T00:00:00Z"}, {"finished_at": "not-a-date"}, {"status": "completed_with_errors"}, {"mode": "dry_run"}, {"finished_at": None}, {"run_id": "different"}, {"error": "PRIVATE_FAILURE"}):
            with self.subTest(change=change):
                approved, latest = fixture()
                latest["run"].update(change)
                with self.assertRaises(compiler.SafeError):
                    compile_fixture(approved, latest)
        for field in ("failed", "blocked", "seen"):
            approved, latest = fixture()
            latest["run"]["counts"][field] += 1
            with self.assertRaises(compiler.SafeError):
                compile_fixture(approved, latest)
        approved, latest = fixture()
        latest["run_items"].pop(1)
        self.assert_drift(approved, latest, "saved_candidate.missing_or_mismatched")

    def test_publication_requires_unblocked_bound_public_url_and_valid_fingerprint(self):
        for change in ({"publish_block_reason": "publication_hold"}, {"public_condition": "public_url_missing"}, {"proposed_section_code": ""}, {"expected_fingerprint": "old"}, {"expected_fingerprint": None}, {"reason": "notion_state_conflict"}, {"match_method": "unknown"}, {"requires_legacy_alias": True}):
            with self.subTest(change=change):
                approved, latest = fixture()
                update_candidate(latest, 0, **change)
                self.assert_drift(approved, latest)
        approved, latest = fixture()
        update_candidate(latest, 0, proposed_section_code=True)
        self.assert_drift(approved, latest, "candidate.missing_or_invalid")
        approved, latest = fixture()
        latest["run_items"][0]["before"]["management_state"] = "managed"
        self.assert_drift(approved, latest, "management_state.not_baseline_pending")

    def test_empty_unmapped_notion_state_is_reviewable_only_with_original_approval(self):
        approved, latest = fixture()
        for entry in approved["entries"]:
            entry["snapshot"]["reason"] = "unknown_state"
        for item in latest["run_items"]:
            if "page_id" in item["after"]:
                item["after"]["reason"] = "unknown_state"
        self.assertEqual(len(compile_fixture(approved, latest)["published"]["input"]["confirmations"]), 14)
        latest["run_items"][0]["after"]["state_option_id"] = "unknown-option"
        self.assert_drift(approved, latest, "snapshot.unusable")
        approved, latest = fixture()
        latest["run_items"][0]["after"]["reason"] = "unknown_state"
        self.assert_drift(approved, latest, "snapshot.unusable")

    def test_required_candidate_fields_missing_do_not_default_to_approval(self):
        for field in ("requires_legacy_alias", "match_method", "candidate_article_ids", "title_hint_article_ids", "public_condition"):
            approved, latest = fixture()
            del latest["items"][14][field]
            del latest["run_items"][29]["after"]["bootstrap_preview"][field]
            self.assert_drift(approved, latest, "candidate.missing_or_invalid")

    def test_subsection_drift_is_detected_even_when_original_go_field_was_omitted(self):
        approved, latest = fixture()
        update_candidate(latest, 0, local_subsection_code="changed-subsection")
        self.assert_drift(approved, latest, "local.local_subsection_code")
        update_candidate(latest, 0, local_subsection_code="")
        self.assertTrue(compile_fixture(approved, latest))
        approved["entries"][0]["local"]["local_subsection_code"] = ""
        del latest["items"][0]["local_subsection_code"]
        del latest["run_items"][1]["after"]["bootstrap_preview"]["local_subsection_code"]
        self.assertTrue(compile_fixture(approved, latest))

    def test_missing_page_drift_is_reported_before_optional_count_guard(self):
        approved, latest = fixture()
        page = latest["items"].pop()["page_id"]
        with self.assertRaises(compiler.SafeError) as caught:
            compile_fixture(approved, latest, expected_publish_count=14, expected_draft_count=24)
        self.assertIn(page, caught.exception.report()["affectedPageIDs"])
        self.assertEqual(caught.exception.category, "review_drift")

    def test_numeric_or_missing_approved_ids_and_boolean_revision_are_rejected(self):
        approved, latest = fixture()
        approved["entries"][0]["article_id"] = 9007199254741993
        with self.assertRaises(compiler.SafeError):
            compile_fixture(approved, latest)
        approved, latest = fixture()
        del approved["entries"][0]["local"]
        with self.assertRaises(compiler.SafeError):
            compile_fixture(approved, latest)
        with self.assertRaises(compiler.SafeError):
            compiler.compile_manifests(*fixture(), True, IMAGE)


class PrivateFileTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(dir=Path(tempfile.gettempdir()).resolve())
        self.directory = Path(self.temporary.name)
        self.addCleanup(self.temporary.cleanup)

    def file(self, name, value):
        path = self.directory / name
        path.write_text(json.dumps(value))
        path.chmod(0o600)
        return path

    def test_success_outputs_are_atomic_private_and_never_overwritten(self):
        path = self.directory / "output.json"
        compiler.write_private_reports({path: {"synthetic": True}})
        self.assertEqual(path.stat().st_mode & 0o777, 0o600)
        self.assertEqual(json.loads(path.read_text()), {"synthetic": True})
        with self.assertRaises(compiler.SafeError):
            compiler.write_private_reports({path: {"overwritten": True}})
        self.assertEqual(json.loads(path.read_text()), {"synthetic": True})

    def test_symlink_input_output_and_parent_are_rejected(self):
        target = self.file("target.json", {})
        link = self.directory / "link.json"
        link.symlink_to(target)
        with self.assertRaises(compiler.SafeError):
            compiler.read_private(link)
        with self.assertRaises(compiler.SafeError):
            compiler.write_private_reports({link: {}})
        parent = self.directory / "alias"
        parent.symlink_to(self.directory, target_is_directory=True)
        with self.assertRaises(compiler.SafeError):
            compiler.write_private_reports({parent / "new.json": {}})
        target.chmod(0o644)
        with self.assertRaises(compiler.SafeError):
            compiler.read_private(target)

    def test_install_race_rolls_back_only_own_output(self):
        first, second = self.directory / "first.json", self.directory / "second.json"
        real_link = os.link
        def raced_link(source, target, **kwargs):
            if target == second:
                second.write_text("other writer")
            return real_link(source, target, **kwargs)
        with mock.patch.object(compiler.os, "link", side_effect=raced_link):
            with self.assertRaises(FileExistsError):
                compiler.write_private_reports({first: {}, second: {}})
        self.assertFalse(first.exists())
        self.assertEqual(second.read_text(), "other writer")
        self.assertFalse(list(self.directory.glob(".review-manifest-*")))

    def test_cli_outputs_only_counts_and_drift_ids_fields(self):
        approved, latest = fixture()
        approval = self.file("approved.json", approved)
        preview = self.file("latest.json", latest)
        prefix = self.directory / "review"
        args = ["--approved", str(approval), "--latest", str(preview), "--config-revision", "9", "--image-sha", IMAGE, "--out-prefix", str(prefix), "--expected-publish-count", "14", "--expected-draft-count", "24"]
        stdout, stderr = io.StringIO(), io.StringIO()
        with contextlib.redirect_stdout(stdout), contextlib.redirect_stderr(stderr):
            self.assertEqual(compiler.main(args), 0)
        self.assertEqual(json.loads(stdout.getvalue())["counts"], {"draft": 24, "published": 14})
        self.assertNotIn("PRIVATE_TITLE", stdout.getvalue() + stderr.getvalue())
        latest["run_items"][0]["after"]["title"] = "PRIVATE_CHANGED"
        preview.write_text(json.dumps(latest))
        drift = self.directory / "drift.json"
        stdout, stderr = io.StringIO(), io.StringIO()
        with contextlib.redirect_stdout(stdout), contextlib.redirect_stderr(stderr):
            self.assertEqual(compiler.main(args + ["--drift-report", str(drift)]), 1)
        self.assertNotIn("PRIVATE_CHANGED", stderr.getvalue())
        self.assertEqual(drift.stat().st_mode & 0o777, 0o600)
        self.assertIn("snapshot.title", drift.read_text())
        self.assertEqual(stdout.getvalue(), "")

    def test_failed_serialization_does_not_leave_private_staging_files(self):
        with self.assertRaises(TypeError):
            compiler.write_private_reports({self.directory / "failure.json": {"unsupported": object()}})
        self.assertFalse(list(self.directory.glob(".review-manifest-*")))
        self.assertFalse((self.directory / "failure.json").exists())

    def test_duplicate_json_keys_and_argument_errors_do_not_echo_input(self):
        path = self.file("duplicate.json", {})
        path.write_text('{"source_id":"PRIVATE_SECRET","source_id":"PRIVATE_SECRET"}')
        with self.assertRaises(compiler.SafeError):
            compiler.read_private(path)
        stderr = io.StringIO()
        with contextlib.redirect_stderr(stderr):
            self.assertEqual(compiler.main(["--config-revision", "PRIVATE_SECRET"]), 1)
        self.assertNotIn("PRIVATE_SECRET", stderr.getvalue())


if __name__ == "__main__":
    unittest.main()
