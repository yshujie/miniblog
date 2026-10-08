-- Cutover gate: stop old writers and run audit-content -apply first.
-- A duplicate primary key intentionally aborts migration before any ALTER when
-- source identities are incomplete. This also works without CHECK enforcement.
DROP TEMPORARY TABLE IF EXISTS `_miniblog_content_cutover_gate`;
CREATE TEMPORARY TABLE `_miniblog_content_cutover_gate` (`id` INT PRIMARY KEY);
INSERT INTO `_miniblog_content_cutover_gate` VALUES (1);
INSERT INTO `_miniblog_content_cutover_gate`
  SELECT 1 FROM `article`
  WHERE COALESCE(TRIM(`external_link`), '') <> ''
    AND (`source_key` IS NULL OR `source_key` NOT REGEXP '^[0-9a-f]{64}$'
      OR COALESCE(TRIM(`provider`), '') = '' OR COALESCE(TRIM(`canonical_url`), '') = '')
  LIMIT 1;
INSERT INTO `_miniblog_content_cutover_gate`
  SELECT 1 FROM (
    SELECT `source_key` FROM `article`
    WHERE `source_key` IS NOT NULL
    GROUP BY `source_key` HAVING COUNT(*) > 1
  ) AS `duplicate_sources` LIMIT 1;
DROP TEMPORARY TABLE `_miniblog_content_cutover_gate`;

-- The safe down migration retains this index. Reapplying up must therefore be
-- idempotent while still checking data completeness on every attempt.
SET @content_unique = (SELECT COUNT(*) FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'article'
    AND INDEX_NAME = 'uq_article_source_key' AND NON_UNIQUE = 0);
SET @content_plain = (SELECT COUNT(*) FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'article'
    AND INDEX_NAME = 'idx_article_source_key');
SET @content_sql = IF(@content_unique > 0, 'SELECT 1',
  IF(@content_plain > 0,
    'ALTER TABLE article DROP KEY idx_article_source_key, ADD UNIQUE KEY uq_article_source_key (source_key)',
    'ALTER TABLE article ADD UNIQUE KEY uq_article_source_key (source_key)'));
PREPARE content_cutover FROM @content_sql;
EXECUTE content_cutover;
DEALLOCATE PREPARE content_cutover;
