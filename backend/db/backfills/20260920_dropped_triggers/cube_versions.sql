-- Backfill: cube_custom_model_versions + cube_security_policy_versions
--
-- The deleted cube_custom_model_version and cube_security_policy_version
-- triggers inserted version rows on every UPDATE. This script does NOT
-- reconstruct the version history (impossible without WAL); it just
-- confirms the version-row tables are empty and ready for the consumer to
-- begin versioning from a clean slate.
--
-- If you need historical version rows, do it from your application logs
-- or from a previous snapshot — not from the live DB.

BEGIN;

SELECT 'cube_custom_model_versions pre-backfill' AS label, count(*) FROM cube_custom_model_versions
UNION ALL
SELECT 'cube_security_policy_versions pre-backfill', count(*) FROM cube_security_policy_versions;

-- Verify the latest version on each parent matches max version in versions table
-- (a sanity check that the trigger never drifted before being dropped).
SELECT ccm.id AS custom_model_id,
       ccm.version AS parent_version,
       max(ccmv.version) AS versions_table_max
FROM   cube_custom_models ccm
LEFT JOIN cube_custom_model_versions ccmv ON ccmv.custom_model_id = ccm.id
GROUP  BY ccm.id, ccm.version
HAVING ccm.version IS DISTINCT FROM max(ccmv.version);

COMMIT;
