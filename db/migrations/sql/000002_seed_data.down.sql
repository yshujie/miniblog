-- 000002_seed_data.down.sql
-- Rollback seed data

USE `miniblog`;

-- Remove seed data in reverse order
DELETE FROM `casbin_rule` WHERE id = 1;
DELETE FROM `article` WHERE id >= 573483961993409070;
DELETE FROM `section` WHERE id <= 19;
DELETE FROM `module` WHERE id <= 6;
DELETE FROM `user` WHERE id = 1;
