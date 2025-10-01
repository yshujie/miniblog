-- 000002_seed_data.up.sql
-- Insert initial seed data for miniblog

USE `miniblog`;

-- Insert initial user
INSERT INTO `user` (id, username, password, nickname, avatar, email, phone, introduction, status, created_at, updated_at) 
VALUES (1, 'clack', '$2a$10$FAQPC0W5l.UVnrrZnp0yEu0QxseILstQdzdUu9uj2TI7Ndl1tpK5C', 'clack', ' ', 'yangshujie@gmail.com', '15711236163', ' ', 0, '2025-06-18 11:34:07', '2025-06-18 11:34:07')
ON DUPLICATE KEY UPDATE username=username;

-- Insert modules
INSERT INTO `module` (id, code, title, status, created_at, updated_at) VALUES 
(1, 'ai', 'AI', 0, '2025-06-17 11:51:38', '2025-06-17 11:51:38'),
(2, 'go', 'Golang', 1, '2025-06-17 11:51:38', '2025-06-17 11:51:38'),
(3, 'ddd', '领域驱动设计', 1, '2025-06-17 11:51:38', '2025-06-17 11:51:38'),
(4, 'refactor', '重构', 0, '2025-06-17 11:51:38', '2025-06-17 11:51:38'),
(5, 'database', '数据库', 0, '2025-06-17 11:51:38', '2025-06-23 19:38:38'),
(6, 'project', '项目', 1, '2025-06-23 14:03:22', '2025-06-23 14:03:22')
ON DUPLICATE KEY UPDATE title=VALUES(title), status=VALUES(status);

-- Note: Sections and articles data will be loaded from separate files
-- See: section.sql, article.sql for full data
