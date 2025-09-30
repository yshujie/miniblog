-- 000001_init.down.sql
-- Down migration for 000001_init.up.sql
-- Warning: destructive operations. Only run in safe environments or CI that intentionally rolls back.

DROP TABLE IF EXISTS `users`;
DROP TABLE IF EXISTS `article`;
DROP TABLE IF EXISTS `section`;
DROP TABLE IF EXISTS `module`;
DROP TABLE IF EXISTS `user`;
DROP DATABASE IF EXISTS `miniblog`;
