-- Intentionally retain schema, JSON tags, source aliases and ownership on rollback.
-- Stop/revoke the sync lease and run a compatibility binary that understands these fields.
SELECT 1;
