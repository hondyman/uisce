-- Allow cube-backed saved queries (PR0 / Page Designer consume).
-- source_kind already exists (DEFAULT 'business_object'); constrain known kinds.

ALTER TABLE data_explorer.saved_query
  DROP CONSTRAINT IF EXISTS saved_query_source_kind_check;

ALTER TABLE data_explorer.saved_query
  ADD CONSTRAINT saved_query_source_kind_check
  CHECK (source_kind IN ('business_object', 'cube'));

COMMENT ON COLUMN data_explorer.saved_query.source_kind IS
  'business_object (source_id=bo id) or cube (source_id=cube id; binding_id null)';
