-- Drift detector: fields in the live structure facet with no corresponding
-- bo_field_key_registry row. Non-empty result means registry ownership
-- (currently: nobody — see backfill_bo_field_key_registry.sql header) has
-- let the registry fall behind. Run in CI or as a scheduled check; wire into
-- a future drift-checker once one exists, this is the starter version.
--
-- Exit convention for CI: a non-empty result set should fail the job.
-- Example (psql): fail if row count > 0
--   psql "$DATABASE_URL" -tAc "$(cat check_field_key_registry_drift.sql)" | grep -q . && exit 1
--
-- RECOVERY NOTE (2026-09-06): this file was deleted, pre-commit, by a
-- concurrent session's git-clean-class operation on a shared working tree.
-- Reconstructed and re-run against live alpha: confirmed empty result
-- (zero drift as of this writing).

SELECT bo.bo_name, f.field_name
FROM public.business_object_fields f
JOIN public.business_objects bo ON bo.id = f.bo_id
LEFT JOIN public.bo_field_key_registry r
    ON r.bo_name = bo.bo_name AND r.field_name = f.field_name
WHERE r.id IS NULL;
