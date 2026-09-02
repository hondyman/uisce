-- 20260901000004_drop_unused_business_object_tables.sql
-- Cleanup pass: this product has no legacy users to preserve compatibility
-- for, so drop every business-object/binding table confirmed to have zero
-- live Go code references (grep -rn "FROM|INTO|UPDATE|JOIN <table>" across
-- backend/, cross-checked against routed HTTP handlers). Two dead Go files
-- that only queried now-dropped tables and were never instantiated by any
-- live caller (internal/ai/graph_rag.go, internal/boresolver/
-- relationship_scope_service.go) were deleted in the same change.
--
-- Kept (confirmed live): business_objects, business_object_bindings,
-- business_object_binding (singular — live via GET /api/business-objects/{id}/scope),
-- business_object_relationships, business_object_fields, bo_fields,
-- bo_instances, bo_subtypes, field_bindings, relationship_bindings.

DROP TABLE IF EXISTS business_object CASCADE;
DROP TABLE IF EXISTS business_object_def CASCADE;
DROP TABLE IF EXISTS business_object_rules CASCADE;
DROP TABLE IF EXISTS business_object_access_rules CASCADE;
DROP TABLE IF EXISTS business_object_validation_rule CASCADE;
DROP TABLE IF EXISTS business_object_relationship CASCADE;
DROP TABLE IF EXISTS business_object_field CASCADE;
DROP TABLE IF EXISTS legacy_business_objects CASCADE;
DROP TABLE IF EXISTS report_business_objects CASCADE;
DROP TABLE IF EXISTS field_binding CASCADE;
DROP TABLE IF EXISTS relationship_binding CASCADE;
DROP TABLE IF EXISTS env_binding CASCADE;
DROP TABLE IF EXISTS binding_related_table CASCADE;
DROP TABLE IF EXISTS bp_role_semantic_policy_bindings CASCADE;
-- cell_footnote_bindings lives in the catalog_doc schema — unrelated
-- subsystem (name coincidence only), left alone.
