-- Subsection table: section 下的子分类层级
CREATE TABLE IF NOT EXISTS `subsection` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `code` VARCHAR(128) NOT NULL,
  `title` VARCHAR(255),
  `sort` INT,
  `section_code` VARCHAR(128),
  `status` INT,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_subsection_code` (`code`),
  KEY `idx_subsection_section_code` (`section_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

ALTER TABLE `article`
  ADD COLUMN `subsection_code` VARCHAR(128) DEFAULT NULL AFTER `section_code`,
  ADD KEY `idx_article_subsection_code` (`subsection_code`);
