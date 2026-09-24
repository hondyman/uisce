-- 20260826_001_subtype_relationship_allowlist.down.sql

DROP INDEX IF EXISTS oms.idx_subtype_registry_rel_allowlist_gin;

ALTER TABLE oms.subtype_registry
  DROP COLUMN IF EXISTS relationship_allowlist;
