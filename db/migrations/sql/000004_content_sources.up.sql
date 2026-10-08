-- Expand only. Backfill source identities before enabling the unique index.
-- Expand only. Rerunnable because safe rollback retains all expanded data.
-- Backfill source identities before enabling the unique index.
ALTER TABLE `article` MODIFY COLUMN `external_link` TEXT;

SET @content_sql = IF(EXISTS(SELECT 1 FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'article' AND COLUMN_NAME = 'provider'),
  'SELECT 1', 'ALTER TABLE article ADD COLUMN provider VARCHAR(16) DEFAULT NULL');
PREPARE content_expand FROM @content_sql;
EXECUTE content_expand;
DEALLOCATE PREPARE content_expand;

SET @content_sql = IF(EXISTS(SELECT 1 FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'article' AND COLUMN_NAME = 'canonical_url'),
  'SELECT 1', 'ALTER TABLE article ADD COLUMN canonical_url TEXT DEFAULT NULL');
PREPARE content_expand FROM @content_sql;
EXECUTE content_expand;
DEALLOCATE PREPARE content_expand;

SET @content_sql = IF(EXISTS(SELECT 1 FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'article' AND COLUMN_NAME = 'source_key'),
  'SELECT 1', 'ALTER TABLE article ADD COLUMN source_key CHAR(64) CHARACTER SET ascii COLLATE ascii_bin DEFAULT NULL');
PREPARE content_expand FROM @content_sql;
EXECUTE content_expand;
DEALLOCATE PREPARE content_expand;

SET @content_sql = IF(EXISTS(SELECT 1 FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'module' AND COLUMN_NAME = 'sort'),
  'SELECT 1', 'ALTER TABLE module ADD COLUMN sort INT NOT NULL DEFAULT 0');
PREPARE content_expand FROM @content_sql;
EXECUTE content_expand;
DEALLOCATE PREPARE content_expand;

SET @content_sql = IF(EXISTS(SELECT 1 FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'article' AND INDEX_NAME IN ('idx_article_source_key', 'uq_article_source_key')),
  'SELECT 1', 'ALTER TABLE article ADD KEY idx_article_source_key (source_key)');
PREPARE content_expand FROM @content_sql;
EXECUTE content_expand;
DEALLOCATE PREPARE content_expand;

SET @content_sql = IF(EXISTS(SELECT 1 FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'article' AND INDEX_NAME = 'idx_article_placement'),
  'SELECT 1', 'ALTER TABLE article ADD KEY idx_article_placement (section_code, subsection_code, pos, id)');
PREPARE content_expand FROM @content_sql;
EXECUTE content_expand;
DEALLOCATE PREPARE content_expand;
