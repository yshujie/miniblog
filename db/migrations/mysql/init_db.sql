-- init_db.sql -- idempotent initialization for miniblog database
-- Created: 2025-09-30
-- This script is safe to run multiple times.

-- create database if not exists
CREATE DATABASE IF NOT EXISTS `${APP_DB_NAME}` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;

-- create user if not exists (MySQL 5.7+ compatible)
CREATE USER IF NOT EXISTS '${APP_DB_USER}'@'%' IDENTIFIED BY '${APP_DB_PASSWORD}';

-- grant privileges on database
GRANT ALL PRIVILEGES ON `${APP_DB_NAME}`.* TO '${APP_DB_USER}'@'%';
FLUSH PRIVILEGES;

-- example table (idempotent)
USE `${APP_DB_NAME}`;
CREATE TABLE IF NOT EXISTS users (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  username VARCHAR(64) NOT NULL UNIQUE,
  password_hash VARCHAR(256) NOT NULL,
  email VARCHAR(128),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- add an index if not exists (MySQL doesn't support CREATE INDEX IF NOT EXISTS pre-8.0, so use a safe check)
-- This attempts to create the index and ignores error if it already exists.
DELIMITER $$
CREATE PROCEDURE __create_index_if_not_exists()
BEGIN
  DECLARE v_count INT DEFAULT 0;
  SELECT COUNT(1) INTO v_count FROM INFORMATION_SCHEMA.STATISTICS
    WHERE TABLE_SCHEMA = '${APP_DB_NAME}' AND TABLE_NAME = 'users' AND INDEX_NAME = 'idx_users_username';
  IF v_count = 0 THEN
    ALTER TABLE users ADD INDEX idx_users_username (username);
  END IF;
END$$
DELIMITER ;
CALL __create_index_if_not_exists();
DROP PROCEDURE IF EXISTS __create_index_if_not_exists;
