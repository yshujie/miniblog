#!/usr/bin/env python3
"""Offline compiler of an explicitly approved scope; never contacts Notion or a DB.

Candidate-local fields are checked here. Body/author/link/order protection not present
in a candidate must be checked by the caller's private preflight, or supplied through
--latest-local when included in approved.local. This tool grants no new approval.
"""
import argparse
from datetime import datetime
import json
import os
from pathlib import Path
import re
import stat
import sys
import tempfile

SOURCE_IDS = {
    "2bf330bd-ddf1-80a6-aa49-000bbd1e154b", "d579f619-4f1c-4e7e-9593-ab6529fc63d0",
    "cfa962f3-12be-40aa-95eb-9122186bd4ba", "03d24108-5172-4e8f-8cfd-e3bd52e437af",
    "d13097e3-b78a-4b6e-8848-979749d4b2f3",
}
PAGE_ID = re.compile(r"[0-9a-f]{32}\Z")
ARTICLE_ID = re.compile(r"[1-9][0-9]*\Z")
RUN_ID = re.compile(r"[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}\Z")
FINGERPRINT = re.compile(r"[0-9a-f]{64}\Z")
SNAPSHOT_TYPES = {
    "page_id": str, "source_id": str, "title": str, "tags": list,
    "topic_option_id": str, "topic_option_name": str, "desired_state": int,
    "page_url": str, "native_archived": bool, "in_trash": bool,
    "state_option_id": str,
}
SNAPSHOT_FIELDS = (*SNAPSHOT_TYPES, "public_url")
LOCAL_FIELDS = {"article_id", "local_state", "local_title", "local_section_code", "local_subsection_code"}
PROTECTED_FIELDS = {"author", "content", "external_link", "pos", "protected_hash"}
STATE_NAMES = {0: "", 1: "draft", 2: "published", 3: "unpublished", 4: "archived"}


class SafeError(Exception):
    def __init__(self, category, fields=None):
        super().__init__(category)
        self.category = category
        self.fields = fields or {}

    def report(self):
        return {"error": self.category, "affectedPageIDs": sorted(self.fields),
                "fields": {page: sorted(set(fields)) for page, fields in sorted(self.fields.items())}}


def require(condition, category="invalid_input"):
    if not condition:
        raise SafeError(category)


def no_duplicate_keys(pairs):
    result = {}
    for key, value in pairs:
        require(key not in result, "duplicate_json_key")
        result[key] = value
    return result


def no_symlink_parents(path):
    path = Path(path).absolute()
    require(not any(parent.is_symlink() for parent in (path, *path.parents)), "unsafe_path")
    return path


def read_private(path):
    path = no_symlink_parents(path)
    descriptor = os.open(path, os.O_RDONLY | os.O_NOFOLLOW)
    try:
        value = os.fstat(descriptor)
        require(stat.S_ISREG(value.st_mode) and value.st_uid == os.getuid()
                and not value.st_mode & 0o077, "unsafe_private_file")
        with os.fdopen(descriptor, encoding="utf-8") as stream:
            descriptor = None
            return json.load(stream, object_pairs_hook=no_duplicate_keys)
    finally:
        if descriptor is not None:
            os.close(descriptor)


def write_private_reports(reports):
    """Stage all files then atomically install without overwriting any existing path."""
    staged = []
    installed = []
    try:
        for path, report in reports.items():
            path = no_symlink_parents(path)
            parent = path.parent.stat()
            require(parent.st_uid == os.getuid() and not parent.st_mode & 0o077, "unsafe_report_directory")
            require(not path.exists(), "report_already_exists")
            fd, temporary = tempfile.mkstemp(prefix=".review-manifest-", dir=path.parent)
            staged.append((path, temporary))
            with os.fdopen(fd, "w", encoding="utf-8") as output:
                os.fchmod(output.fileno(), 0o600)
                json.dump(report, output, ensure_ascii=False, indent=2)
                output.write("\n")
                output.flush()
                os.fsync(output.fileno())
        for path, temporary in staged:
            os.link(temporary, path, follow_symlinks=False)
            installed.append((path, os.stat(temporary).st_ino))
        for parent in {path.parent for path, _ in staged}:
            fd = os.open(parent, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW)
            try:
                os.fsync(fd)
            finally:
                os.close(fd)
    except Exception:
        for path, inode in installed:
            if path.lstat().st_ino == inode:
                path.unlink()
        raise
    finally:
        for _, temporary in staged:
            if os.path.exists(temporary):
                os.unlink(temporary)


def snapshot_shape(value):
    return (isinstance(value, dict)
            and all(type(value.get(key)) is kind for key, kind in SNAPSHOT_TYPES.items())
            and "public_url" in value and (value["public_url"] is None or type(value["public_url"]) is str)
            and all(type(tag) is str for tag in value["tags"])
            and value["desired_state"] in STATE_NAMES)


def valid_run_times(run):
    try:
        start, finish = [datetime.fromisoformat(run[key].replace("Z", "+00:00")) for key in ("started_at", "finished_at")]
        return start.tzinfo is not None and finish.tzinfo is not None and start <= finish
    except (KeyError, ValueError, TypeError, AttributeError):
        return False


def compile_manifests(approved, latest, revision, image_sha, *, latest_local=None,
                      expected_publish_count=None, expected_draft_count=None):
    require(latest_local is None or (isinstance(latest_local, dict) and all(isinstance(value, dict) for value in latest_local.values())), "invalid_local_snapshot")
    require(type(revision) is int and revision > 0, "invalid_config_revision")
    require(type(image_sha) is str and re.fullmatch(r"[0-9a-f]{40}", image_sha), "invalid_image_sha")
    require(isinstance(approved, dict) and type(approved.get("source_id")) is str and approved["source_id"] in SOURCE_IDS
            and type(approved.get("reviewer")) is str and approved["reviewer"].strip()
            and isinstance(approved.get("entries"), list) and approved["entries"], "invalid_approval")
    source = approved["source_id"]
    entries = {}
    article_pages = {}
    for entry in approved["entries"]:
        require(isinstance(entry, dict) and type(entry.get("page_id")) is str
                and PAGE_ID.fullmatch(entry["page_id"]), "invalid_approval")
        page = entry["page_id"]
        if page in entries:
            raise SafeError("review_drift", {page: ["approved.page_id.duplicate"]})
        require(type(entry.get("expected_state")) is str and entry["expected_state"] in {"draft", "published"}, "invalid_approval")
        require(type(entry.get("allow_legacy_alias", False)) is bool, "invalid_approval")
        new = entry.get("new_page", False)
        require(type(new) is bool and (entry.get("article_id", "") == "" if new else
                type(entry.get("article_id")) is str and ARTICLE_ID.fullmatch(entry["article_id"])), "invalid_approval")
        if not new:
            other = article_pages.get(entry["article_id"])
            if other:
                raise SafeError("review_drift", {page: ["article_id.duplicate"], other: ["article_id.duplicate"]})
            article_pages[entry["article_id"]] = page
        require(snapshot_shape(entry.get("snapshot")), "invalid_approved_snapshot")
        if not new:
            local = entry.get("local")
            require(isinstance(local, dict) and {"article_id", "local_state", "local_title", "local_section_code"} <= set(local)
                    and not set(local) - LOCAL_FIELDS - PROTECTED_FIELDS, "invalid_approved_local")
        entries[page] = entry
    require(isinstance(latest, dict) and isinstance(latest.get("run"), dict), "invalid_preview_run")
    run = latest["run"]
    counts = run.get("counts", {})
    require(run.get("run_id") == latest.get("run_id") and type(run.get("run_id")) is str
            and RUN_ID.fullmatch(run["run_id"]) and valid_run_times(run) and run.get("mode") == "bootstrap_preview"
            and run.get("status") == "completed"
            and not run.get("error") and isinstance(counts, dict)
            and all(type(counts.get(key)) is int and counts[key] >= 0 for key in ("seen", "failed", "blocked"))
            and counts["failed"] == counts["blocked"] == 0, "incomplete_preview_run")
    require(isinstance(latest.get("items"), list) and isinstance(latest.get("run_items"), list), "incomplete_preview_audit")
    drift = {}

    def flag(page, field):
        if type(page) is str and PAGE_ID.fullmatch(page):
            drift.setdefault(page, []).append(field)
        else:
            raise SafeError("invalid_preview_identity")

    candidates = {}
    for candidate in latest["items"]:
        require(isinstance(candidate, dict) and type(candidate.get("page_id")) is str
                and PAGE_ID.fullmatch(candidate["page_id"]), "invalid_preview_identity")
        page = candidate["page_id"]
        if page in candidates:
            flag(page, "candidate.page_id.duplicate")
        candidates[page] = candidate
    snapshots = {}
    before = {}
    saved = {}
    for item in latest["run_items"]:
        require(isinstance(item, dict), "invalid_preview_audit")
        page = item.get("page_id")
        if not page:
            continue  # Catalog audit rows have no page identity.
        require(type(page) is str and PAGE_ID.fullmatch(page), "invalid_preview_identity")
        after = item.get("after")
        if isinstance(after, dict) and "bootstrap_preview" in after:
            if page in saved:
                flag(page, "saved_candidate.duplicate")
            saved[page] = after["bootstrap_preview"]
            if "snapshot" in after:
                if page in snapshots:
                    flag(page, "snapshot.page_id.duplicate")
                snapshots[page] = after["snapshot"]
                before[page] = item.get("before")
        else:
            if page in snapshots:
                flag(page, "snapshot.page_id.duplicate")
            snapshots[page] = after
            before[page] = item.get("before")
    scoped = {page for page, candidate in candidates.items() if candidate.get("source_id") == source}
    for page in scoped - set(entries):
        flag(page, "scope.added_baseline_page")
    for page in set(entries) - scoped:
        flag(page, "scope.missing_or_moved_page")
    confirmations = {"draft": [], "published": []}
    for page, entry in entries.items():
        candidate = candidates.get(page)
        snap = snapshots.get(page)
        old = entry["snapshot"]
        if not candidate:
            continue
        candidate_fields = {"source_id": str, "title": str, "topic": str, "notion_state": str, "new_page": bool,
                            "requires_legacy_alias": bool, "match_method": str, "public_condition": str,
                            "candidate_article_ids": list, "title_hint_article_ids": list}
        optional_strings = ("article_id", "proposed_section_code", "publish_block_reason", "reason", "local_state", "local_title", "local_section_code", "local_subsection_code")
        if (not all(type(candidate[key]) is str for key in optional_strings if key in candidate)
                or not all(type(candidate.get(key)) is kind for key, kind in candidate_fields.items())
                or not all(type(identity) is str and ARTICLE_ID.fullmatch(identity)
                           for key in ("candidate_article_ids", "title_hint_article_ids") for identity in candidate.get(key, []) if isinstance(candidate.get(key), list))):
            flag(page, "candidate.missing_or_invalid")
        if saved.get(page) != candidate:
            flag(page, "saved_candidate.missing_or_mismatched")
        if not snapshot_shape(snap):
            flag(page, "snapshot.missing_or_invalid")
            continue
        for key in SNAPSHOT_FIELDS:
            if old[key] != snap[key]:
                flag(page, "snapshot." + key)
        if old["page_id"] != page or snap["page_id"] != page or old["source_id"] != source or snap["source_id"] != source:
            flag(page, "snapshot.identity")
        for key, value in (("source_id", source), ("title", snap["title"]), ("topic", snap["topic_option_name"]), ("notion_state", STATE_NAMES[snap["desired_state"]])):
            if candidate.get(key) != value:
                flag(page, "candidate." + key)
        if not isinstance(before.get(page), dict) or before[page].get("management_state") != "baseline_pending":
            flag(page, "management_state.not_baseline_pending")
        new = entry.get("new_page", False)
        if type(candidate.get("new_page")) is not bool or candidate["new_page"] != new or (candidate.get("article_id") or "") != entry.get("article_id", ""):
            flag(page, "article_identity")
        allowed_matches = {"unmatched"} if new else {"trusted_page_id", "known_reading_url", "explicit_alias_review"}
        if candidate.get("reason") or candidate.get("match_method") not in allowed_matches:
            flag(page, "candidate.reason_or_match_conflict")
        if candidate.get("requires_legacy_alias") and not entry.get("allow_legacy_alias", False):
            flag(page, "allow_legacy_alias")
        empty_state_review = (snap.get("reason") == old.get("reason") == "unknown_state"
                              and snap["desired_state"] == old["desired_state"] == 0
                              and snap["state_option_id"] == old["state_option_id"] == "")
        if snap["native_archived"] or snap["in_trash"] or (snap.get("reason") and not empty_state_review):
            flag(page, "snapshot.unusable")
        if not new:
            local = entry["local"]
            # Only Go's omitempty local_subsection_code has an approved empty default.
            if candidate.get("local_subsection_code", "") != local.get("local_subsection_code", ""):
                flag(page, "local.local_subsection_code")
            for key, value in local.items():
                actual = (candidate.get(key, "") if key == "local_subsection_code" else candidate.get(key)) if key in LOCAL_FIELDS else (latest_local or {}).get(entry["article_id"], {}).get(key)
                if actual != value:
                    flag(page, "local." + key)
            if local["article_id"] != entry["article_id"] or local["local_state"] != entry["expected_state"]:
                flag(page, "local.approved_identity_or_state")
        elif candidate.get("candidate_article_ids"):
            flag(page, "new_page.historical_identity")
        fingerprint = candidate.get("expected_fingerprint")
        if type(fingerprint) is not str or not FINGERPRINT.fullmatch(fingerprint):
            flag(page, "expected_fingerprint")
        if entry["expected_state"] == "published" and (candidate.get("publish_block_reason")
                or candidate.get("public_condition") != "public_url_available"
                or type(candidate.get("proposed_section_code")) is not str or not candidate["proposed_section_code"] or not snap["public_url"]):
            flag(page, "publication.not_ready")
        confirmations[entry["expected_state"]].append({
            "page_id": page, "article_id": entry.get("article_id", ""), "new_page": new,
            "allow_legacy_alias": entry.get("allow_legacy_alias", False),
            "expected_state": entry["expected_state"], "expected_fingerprint": fingerprint,
            "confirmed_by": approved["reviewer"],
        })
    if drift:
        raise SafeError("review_drift", drift)
    require(counts["seen"] == len(snapshots), "incomplete_preview_audit")
    for state, expected in (("published", expected_publish_count), ("draft", expected_draft_count)):
        if expected is not None:
            require(type(expected) is int and expected >= 0, "invalid_expected_count")
            require(len(confirmations[state]) == expected, "approved_count_mismatch")
    return {state: {"version": 1, "mode": "bootstrap_apply", "source_id": source,
                    "expected_config_revision": revision, "expected_image_sha": image_sha,
                    "input": {"confirmations": rows}}
            for state, rows in confirmations.items() if rows}


class SafeParser(argparse.ArgumentParser):
    def error(self, message):
        raise SafeError("invalid_arguments")


def main(argv=None):
    parser = SafeParser(description=__doc__)
    parser.add_argument("--approved", required=True)
    parser.add_argument("--latest", required=True)
    parser.add_argument("--latest-local", help="Private article-ID map for optional protected local fields")
    parser.add_argument("--config-revision", type=int, required=True)
    parser.add_argument("--image-sha", required=True)
    parser.add_argument("--out-prefix", required=True)
    parser.add_argument("--drift-report")
    parser.add_argument("--expected-publish-count", type=int)
    parser.add_argument("--expected-draft-count", type=int)
    args = None
    try:
        args = parser.parse_args(argv)
        reports = compile_manifests(read_private(args.approved), read_private(args.latest), args.config_revision,
                                    args.image_sha, latest_local=read_private(args.latest_local) if args.latest_local else None,
                                    expected_publish_count=args.expected_publish_count, expected_draft_count=args.expected_draft_count)
        outputs = {Path(args.out_prefix + "." + state + ".json"): report for state, report in reports.items()}
        write_private_reports(outputs)
        print(json.dumps({"compiled": True, "counts": {state: len(report["input"]["confirmations"]) for state, report in reports.items()}}))
        return 0
    except SafeError as error:
        if args and args.drift_report:
            try:
                write_private_reports({Path(args.drift_report): error.report()})
            except (OSError, SafeError):
                print(json.dumps({"error": "drift_report_write_failed"}), file=sys.stderr)
                return 1
        print(json.dumps(error.report()), file=sys.stderr)
        return 1
    except (OSError, ValueError, TypeError, KeyError):
        print(json.dumps({"error": "invalid_private_input_or_output"}), file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
