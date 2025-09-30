-- 000001_init.up.sql
-- Create database, user and initial tables for miniblog
-- This is a simplified, idempotent migration without stored procedures.

CREATE DATABASE IF NOT EXISTS `miniblog` DEFAULT CHARACTER SET = 'utf8mb4' COLLATE = 'utf8mb4_general_ci';

-- Create application user if not exists (MySQL 5.7+ supports CREATE USER IF NOT EXISTS)
-- Adjust username/password as needed; this migration expects the runner has sufficient privileges.

CREATE USER IF NOT EXISTS 'miniblog'@'%' IDENTIFIED BY 'miniblog_password';
GRANT ALL PRIVILEGES ON `miniblog`.* TO 'miniblog'@'%';
FLUSH PRIVILEGES;

USE `miniblog`;

CREATE TABLE IF NOT EXISTS `users` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `username` VARCHAR(128) NOT NULL,
  `email` VARCHAR(255),
  `password` VARCHAR(255) NOT NULL,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_users_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Create additional tables as needed by application. Keep indexes created here.

-- If you need to add an index conditionally for older MySQL that doesn't support IF NOT EXISTS on ALTER,
-- you can rely on migration tooling which records applied versions (recommended).


-- Module table
CREATE TABLE IF NOT EXISTS `module` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `code` VARCHAR(128) NOT NULL,
  `title` VARCHAR(255),
  `status` INT,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_module_code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Section table
CREATE TABLE IF NOT EXISTS `section` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `code` VARCHAR(128) NOT NULL,
  `title` VARCHAR(255),
  `sort` INT,
  `module_code` VARCHAR(128),
  `status` INT,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_section_code` (`code`),
  KEY `idx_section_module_code` (`module_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Article table
CREATE TABLE IF NOT EXISTS `article` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `title` VARCHAR(255),
  `content` LONGTEXT,
  `external_link` VARCHAR(255),
  `section_code` VARCHAR(128),
  `author` VARCHAR(128),
  `tags` VARCHAR(255),
  `pos` INT,
  `status` INT,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_article_section_code` (`section_code`),
  KEY `idx_article_pos` (`pos`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- User table (rename to keep consistent with model.TableName)
CREATE TABLE IF NOT EXISTS `user` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `username` VARCHAR(128) NOT NULL,
  `password` VARCHAR(255) NOT NULL,
  `nickname` VARCHAR(128),
  `avatar` VARCHAR(255),
  `email` VARCHAR(255),
  `phone` VARCHAR(64),
  `introduction` TEXT,
  `status` INT,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_user_username` (`username`),
  UNIQUE KEY `idx_user_email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
