-- Keep uniqueness on rollback: the compatible application understands it.
-- Removing it would make already-registered documents vulnerable to duplication.
SELECT 1;
