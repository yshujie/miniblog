ALTER TABLE `article`
  DROP KEY `idx_article_subsection_code`,
  DROP COLUMN `subsection_code`;

DROP TABLE IF EXISTS `subsection`;
