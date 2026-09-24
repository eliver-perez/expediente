#!/usr/bin/env python3
"""Validate the H1 design in disposable SQLite databases; no product bootstrap."""

import json
from pathlib import Path
import re
import sqlite3
import tempfile
import unittest
from urllib.parse import unquote


PROJECT_DIRECTORY = Path(__file__).resolve().parent.parent
SCHEMA_PATH = PROJECT_DIRECTORY / "db" / "schema.proposed.sql"
FIXED_TIME = "2026-09-23T12:00:00.000000Z"


class ProposedSchemaChecks(unittest.TestCase):
    def setUp(self):
        self.temporary_directory = tempfile.TemporaryDirectory(prefix="document-design-")
        self.addCleanup(self.temporary_directory.cleanup)
        database_path = Path(self.temporary_directory.name) / "design.sqlite"
        self.connection = sqlite3.connect(database_path, isolation_level=None)
        self.addCleanup(self.connection.close)
        self.connection.executescript(SCHEMA_PATH.read_text(encoding="utf-8"))
        self.insert("users", id="user-a", username="owner", username_key="owner",
                    display_name="Design fixture", password_hash="not-a-real-password-hash",
                    created_at=FIXED_TIME, updated_at=FIXED_TIME)
        for library_id in ("library-a", "library-b"):
            self.insert("libraries", id=library_id, name=library_id, mode="linked",
                        created_at=FIXED_TIME, updated_at=FIXED_TIME)

    def insert(self, table_name, **fields):
        # All identifiers are test constants; all values use bound parameters.
        columns = ", ".join(fields)
        placeholders = ", ".join("?" for _ in fields)
        return self.connection.execute(
            f"INSERT INTO {table_name} ({columns}) VALUES ({placeholders})",
            tuple(fields.values()),
        )

    def create_session(self, session_id):
        self.insert("sessions", id=session_id, user_id="user-a",
                    token_digest=f"fixture-digest-{session_id}", csrf_token_digest="fixture-csrf",
                    credential_version=1, started_at=FIXED_TIME, last_activity_at=FIXED_TIME,
                    expires_at="2026-09-24T12:00:00.000000Z",
                    observed_ip="127.0.0.1", user_agent="Design test")

    def create_file(self, file_id="file-a", library_id="library-a", source="linked"):
        self.insert("physical_files", id=file_id, library_id=library_id,
                    storage_source=source, availability="available",
                    integrity_status="verified", created_at=FIXED_TIME)

    def create_document(self, document_id="document-a", file_id="file-a", library_id="library-a", **extra):
        self.insert("documents", id=document_id, library_id=library_id,
                    physical_file_id=file_id, original_filename="plan.pdf", filename_search_key="plan.pdf",
                    approval_status="draft", created_at=FIXED_TIME, **extra)

    def create_extraction(self, generation=1, page_text="Estructura metálica aprobada"):
        version_id = f"content-{generation}"
        extraction_id = f"extraction-{generation}"
        self.insert("content_versions", id=version_id, physical_file_id="file-a",
                    generation=generation, size_bytes=100, os_modified_at=FIXED_TIME,
                    sha256="a" * 64, observed_change_key=f"observation-{generation}",
                    observed_at=FIXED_TIME, change_origin="scan")
        self.insert("extraction_runs", id=extraction_id, physical_file_id="file-a",
                    content_version_id=version_id, extractor_revision="design-fixture",
                    language_codes="spa", status="complete", page_count=1, completed_at=FIXED_TIME)
        self.insert("extraction_pages", extraction_id=extraction_id, page_number=1,
                    extraction_method="native", page_text=page_text)
        return extraction_id

    def publish_extraction(self, extraction_id):
        self.connection.execute("DELETE FROM indexed_pages WHERE physical_file_id = ?", ("file-a",))
        self.connection.execute(
            """INSERT INTO indexed_pages (physical_file_id, extraction_id, page_number, page_text)
               SELECT ?, extraction_id, page_number, page_text FROM extraction_pages WHERE extraction_id = ?""",
            ("file-a", extraction_id),
        )
        self.connection.execute(
            "UPDATE physical_files SET indexed_extraction_id = ? WHERE id = ?",
            (extraction_id, "file-a"),
        )

    def search_count(self, query):
        return self.connection.execute(
            "SELECT count(*) FROM page_text_fts WHERE page_text_fts MATCH ?", (query,)
        ).fetchone()[0]

    def test_schema_foreign_keys_integrity_and_wal_support(self):
        self.assertEqual(self.connection.execute("PRAGMA foreign_keys").fetchone()[0], 1)
        self.assertEqual(self.connection.execute("PRAGMA journal_mode=WAL").fetchone()[0], "wal")
        self.assertEqual(self.connection.execute("PRAGMA foreign_key_check").fetchall(), [])
        self.assertEqual(self.connection.execute("PRAGMA integrity_check").fetchone()[0], "ok")

    def test_only_one_open_session_per_user(self):
        self.create_session("session-a")
        with self.assertRaises(sqlite3.IntegrityError):
            self.create_session("session-b")

    def test_session_replacement_commits_and_keeps_history(self):
        self.create_session("session-a")
        self.connection.execute("BEGIN IMMEDIATE")
        self.connection.execute(
            "UPDATE sessions SET closed_at = ?, close_reason = 'new_login' WHERE id = ?",
            (FIXED_TIME, "session-a"),
        )
        self.create_session("session-b")
        self.connection.execute("COMMIT")
        self.assertEqual(self.connection.execute(
            "SELECT id FROM sessions WHERE closed_at IS NULL"
        ).fetchall(), [("session-b",)])
        self.assertEqual(self.connection.execute(
            "SELECT close_reason FROM sessions WHERE id = 'session-a'"
        ).fetchone()[0], "new_login")

    def test_session_replacement_rollback_preserves_previous_session(self):
        self.create_session("session-a")
        self.connection.execute("BEGIN IMMEDIATE")
        self.connection.execute(
            "UPDATE sessions SET closed_at = ?, close_reason = 'new_login' WHERE id = ?",
            (FIXED_TIME, "session-a"),
        )
        self.create_session("session-b")
        self.connection.execute("ROLLBACK")
        self.assertEqual(self.connection.execute(
            "SELECT id FROM sessions WHERE closed_at IS NULL"
        ).fetchall(), [("session-a",)])

    def test_document_can_be_unclassified_but_cannot_duplicate_file(self):
        self.create_file()
        self.create_document()
        self.assertIsNone(self.connection.execute("SELECT case_id FROM documents").fetchone()[0])
        with self.assertRaises(sqlite3.IntegrityError):
            self.create_document(document_id="document-b")

    def test_document_cannot_reference_another_library_file_or_case(self):
        self.create_file()
        with self.assertRaises(sqlite3.IntegrityError):
            self.create_document(library_id="library-b")
        self.insert("cases", id="case-b", library_id="library-b", identifier="B-001",
                    identifier_key="b-001", created_at=FIXED_TIME)
        with self.assertRaises(sqlite3.IntegrityError):
            self.create_document(case_id="case-b")

    def test_storage_source_is_immutable_and_os_identity_is_unique(self):
        self.create_file()
        self.create_file(file_id="file-b")
        with self.assertRaises(sqlite3.IntegrityError):
            self.connection.execute("UPDATE physical_files SET storage_source = 'managed' WHERE id = 'file-a'")
        self.connection.execute("UPDATE physical_files SET os_identity_key = 'verified-volume:file' WHERE id = 'file-a'")
        with self.assertRaises(sqlite3.IntegrityError):
            self.connection.execute("UPDATE physical_files SET os_identity_key = 'verified-volume:file' WHERE id = 'file-b'")

    def test_equal_hashes_do_not_merge_distinct_files(self):
        self.create_file()
        self.create_extraction()
        self.create_file(file_id="file-b")
        self.insert("content_versions", id="other-content", physical_file_id="file-b",
                    generation=1, size_bytes=100, os_modified_at=FIXED_TIME, sha256="a" * 64,
                    observed_change_key="other-observation", observed_at=FIXED_TIME, change_origin="scan")
        self.assertEqual(self.connection.execute("SELECT count(*) FROM content_versions WHERE sha256 = ?",
                                                ("a" * 64,)).fetchone()[0], 2)

    def test_missing_original_keeps_searchable_text(self):
        self.create_file()
        self.create_document()
        self.publish_extraction(self.create_extraction())
        self.connection.execute("UPDATE physical_files SET availability = 'missing'")
        self.assertEqual(self.search_count('"estructura metalica"'), 1)

    def test_fts_replacement_is_atomic_and_updates_old_terms(self):
        self.create_file()
        previous_extraction = self.create_extraction()
        self.publish_extraction(previous_extraction)
        next_extraction = self.create_extraction(generation=2, page_text="Hormigón nuevo")
        self.assertEqual(self.search_count("metalica"), 1)
        self.assertEqual(self.search_count("hormigon"), 0)
        self.connection.execute("BEGIN IMMEDIATE")
        self.publish_extraction(next_extraction)
        self.connection.execute("ROLLBACK")
        self.assertEqual(self.search_count("metalica"), 1)
        self.assertEqual(self.search_count("hormigon"), 0)
        self.connection.execute("BEGIN IMMEDIATE")
        self.publish_extraction(next_extraction)
        self.connection.execute("COMMIT")
        self.assertEqual(self.search_count("metalica"), 0)
        self.assertEqual(self.search_count("hormigon"), 1)
        self.connection.execute("INSERT INTO page_text_fts(page_text_fts, rank) VALUES ('integrity-check', 1)")

    def test_audit_is_append_only_and_search_keeps_exact_query(self):
        self.insert("audit_events", id="event-a", occurred_at=FIXED_TIME, actor_kind="user",
                    actor_user_id="user-a", event_type="search.executed", details_json="{}")
        exact_query = '  PR-008 "estructura metálica"  '
        self.insert("search_audit_details", event_id="event-a", exact_query_text=exact_query,
                    search_type="general", consulted_library_ids_json='["library-a"]',
                    applied_filters_json='{"availability":"missing"}', result_count=1)
        self.assertEqual(self.connection.execute("SELECT exact_query_text FROM search_audit_details").fetchone()[0],
                         exact_query)
        for statement in ("UPDATE audit_events SET event_type = 'changed'", "DELETE FROM audit_events"):
            with self.assertRaises(sqlite3.IntegrityError):
                self.connection.execute(statement)

    def test_jobs_limit_attempts_and_concurrent_file_work(self):
        self.create_file()
        fields = dict(physical_file_id="file-a", job_type="extract", target_version="1",
                      payload_json="{}", status="running", available_at=FIXED_TIME, created_at=FIXED_TIME)
        self.insert("jobs", id="job-a", idempotency_key="extract:file-a:1", **fields)
        with self.assertRaises(sqlite3.IntegrityError):
            self.insert("jobs", id="job-b", idempotency_key="verify:file-a:1", **fields)
        with self.assertRaises(sqlite3.IntegrityError):
            self.connection.execute("UPDATE jobs SET attempt_count = 6 WHERE id = 'job-a'")


class DocumentationChecks(unittest.TestCase):
    def test_local_markdown_links_exist(self):
        missing_links = []
        for document_path in PROJECT_DIRECTORY.rglob("*.md"):
            if any(part in {"node_modules", ".git", ".cache", "build", "test-results", "playwright-report"}
                   for part in document_path.relative_to(PROJECT_DIRECTORY).parts):
                continue
            for target in re.findall(r"\[[^\]]*\]\(([^)]+)\)", document_path.read_text(encoding="utf-8")):
                if target.startswith(("https://", "http://", "#", "mailto:")):
                    continue
                relative_path = unquote(target.split("#", 1)[0].strip("<>"))
                if not (document_path.parent / relative_path).exists():
                    missing_links.append(f"{document_path.relative_to(PROJECT_DIRECTORY)} -> {target}")
        self.assertEqual(missing_links, [])

    def test_license_example_preserves_required_v1_fields(self):
        contract = (PROJECT_DIRECTORY / "LICENSE_CONTRACT.md").read_text(encoding="utf-8")
        payload_section = contract.split("Payload firmado EXACTO de licencia V1", 1)[1]
        payload, _ = json.JSONDecoder().raw_decode(payload_section[payload_section.index("{"):])
        required_fields = {
            "schema_version", "product_id", "license_id", "activation_id", "installation_id",
            "installation_public_key", "fingerprint_version", "fingerprint_hash", "license_type",
            "license_status", "license_revision", "issued_at", "expires_at", "grace_days",
            "maintenance_until", "entitled_release_until", "features", "limits",
        }
        self.assertEqual(set(payload), required_fields)
        self.assertEqual(payload["product_id"], "gestor_documental")
        self.assertEqual(payload["fingerprint_version"], "1")
        self.assertEqual(set(payload["features"]), {
            "linked_libraries", "managed_libraries", "ocr", "expedientes", "review_workflow"
        })
        self.assertIsNone(payload["expires_at"])
        self.assertEqual(payload["grace_days"], 0)


if __name__ == "__main__":
    print(f"H1 design checks — SQLite {sqlite3.sqlite_version}; not a production runtime certification.", flush=True)
    unittest.main(verbosity=2)
