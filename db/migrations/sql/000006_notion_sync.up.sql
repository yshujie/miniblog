-- Expand only. Retained by the compatibility rollback; no account or API work occurs here.
SET @notion_sql = IF(EXISTS(SELECT 1 FROM information_schema.COLUMNS
 WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'article' AND COLUMN_NAME = 'tags_json'),
 'SELECT 1', 'ALTER TABLE article ADD COLUMN tags_json LONGTEXT DEFAULT NULL');
PREPARE notion_expand FROM @notion_sql;
EXECUTE notion_expand;
DEALLOCATE PREPARE notion_expand;

CREATE TABLE IF NOT EXISTS notion_sync_control (
 id BIGINT UNSIGNED PRIMARY KEY,
 paused BOOLEAN NOT NULL DEFAULT TRUE,
 source_writes_paused BOOLEAN NOT NULL DEFAULT FALSE,
 baseline_frozen BOOLEAN NOT NULL DEFAULT FALSE,
 baseline_at DATETIME(6) NULL,
 lease_owner VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL DEFAULT '',
 lease_epoch BIGINT UNSIGNED NOT NULL DEFAULT 0,
 lease_until DATETIME(6) NULL,
 current_run_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL DEFAULT '',
 next_run_at DATETIME(6) NULL,
 cooldown_until DATETIME(6) NULL,
 last_complete_scan_at DATETIME(6) NULL,
 last_success_at DATETIME(6) NULL,
 updated_at DATETIME(6) NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
INSERT IGNORE INTO notion_sync_control (id,paused) VALUES (1,TRUE);

CREATE TABLE IF NOT EXISTS notion_sync_sources (
 source_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 label VARCHAR(255) NOT NULL DEFAULT '',
 data_source_id VARCHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 module_code VARCHAR(128) NOT NULL,
 property_mapping_json LONGTEXT,
 status_mapping_json LONGTEXT,
 config_revision BIGINT UNSIGNED NOT NULL DEFAULT 1,
 enabled BOOLEAN NOT NULL DEFAULT FALSE,
 health VARCHAR(32) NOT NULL DEFAULT 'unconfigured',
 last_attempt_at DATETIME(6) NULL,
 last_complete_scan_at DATETIME(6) NULL,
 last_success_at DATETIME(6) NULL,
 last_error TEXT,
 created_at DATETIME(6) NULL,
 updated_at DATETIME(6) NULL,
 UNIQUE KEY uq_notion_source_data_source(data_source_id),
 KEY idx_notion_source_module(module_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE IF NOT EXISTS notion_catalog_bindings (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 source_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 data_source_id VARCHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 theme_property_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 option_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 option_name VARCHAR(255) NOT NULL,
 section_code VARCHAR(128) NULL,
 status VARCHAR(32) NOT NULL DEFAULT 'blocked',
 reason TEXT,
 created_at DATETIME(6) NULL,
 updated_at DATETIME(6) NULL,
 UNIQUE KEY uq_notion_catalog_identity(data_source_id,theme_property_id,option_id),
 KEY idx_notion_catalog_source(source_id),
 KEY idx_notion_catalog_section(section_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE IF NOT EXISTS notion_page_bindings (
 page_id VARCHAR(36) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 article_id BIGINT NULL,
 legacy_source_key CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
 source_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 management_state VARCHAR(32) NOT NULL DEFAULT 'baseline_pending',
 revision BIGINT UNSIGNED NOT NULL DEFAULT 0,
 desired_state INT NOT NULL DEFAULT 1,
 snapshot_json LONGTEXT,
 metadata_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL DEFAULT '',
 applied_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL DEFAULT '',
 page_url TEXT,
 public_url TEXT,
 publish_block_reason VARCHAR(128) NOT NULL DEFAULT '',
 publication_held BOOLEAN NOT NULL DEFAULT FALSE,
 publication_hold_reason VARCHAR(255) NOT NULL DEFAULT '',
 needs_revalidation BOOLEAN NOT NULL DEFAULT TRUE,
 native_archived BOOLEAN NOT NULL DEFAULT FALSE,
 in_trash BOOLEAN NOT NULL DEFAULT FALSE,
 notion_last_edited_at DATETIME(6) NULL,
 last_seen_run_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL DEFAULT '',
 last_apply_run_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL DEFAULT '',
 last_success_at DATETIME(6) NULL,
 last_error TEXT,
 bootstrap_state VARCHAR(32) NOT NULL DEFAULT '',
 bootstrap_expected_state INT NULL,
 bootstrap_expected_fingerprint CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL DEFAULT '',
 confirmed_by VARCHAR(128) NOT NULL DEFAULT '',
 confirmed_at DATETIME(6) NULL,
 local_before_json LONGTEXT,
 created_at DATETIME(6) NULL,
 updated_at DATETIME(6) NULL,
 UNIQUE KEY uq_notion_page_article(article_id),
 UNIQUE KEY uq_notion_page_legacy_source(legacy_source_key),
 KEY idx_notion_page_source(source_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE IF NOT EXISTS notion_sync_runs (
 run_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 mode VARCHAR(32) NOT NULL,
 phase VARCHAR(32) NOT NULL,
 status VARCHAR(32) NOT NULL,
 counts_json LONGTEXT,
 started_at DATETIME(6) NOT NULL,
 finished_at DATETIME(6) NULL,
 error TEXT,
 lease_epoch BIGINT UNSIGNED NOT NULL DEFAULT 0
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE IF NOT EXISTS notion_sync_run_items (
 run_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 item_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 page_id VARCHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 article_id BIGINT NULL,
 phase VARCHAR(32) NOT NULL DEFAULT '',
 outcome VARCHAR(32) NOT NULL,
 reason VARCHAR(255) NOT NULL DEFAULT '',
 before_json LONGTEXT,
 after_json LONGTEXT,
 error TEXT,
 bootstrap_journal_json LONGTEXT,
 created_at DATETIME(6) NULL,
 updated_at DATETIME(6) NULL,
 PRIMARY KEY(run_id,item_id),
 KEY idx_notion_item_page(page_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
