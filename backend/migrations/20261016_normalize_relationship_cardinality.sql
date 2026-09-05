-- Normalize entity_relationship.cardinality to the canonical vocabulary
-- (ONE_TO_ONE / ONE_TO_MANY / MANY_TO_ONE / MANY_TO_MANY) already enforced
-- by entity_relationship_valid_cardinality (006_relationship_discovery_schema.sql).
--
-- Older/legacy discovery code wrote loose strings ("one-to-many", "1:N",
-- lowercase variants) directly into catalog_edge.properties (a JSONB blob,
-- not covered by this migration) and, in a few code paths, values that
-- never actually satisfied the CHECK constraint above were rejected at
-- write time — so any row already in entity_relationship should already be
-- canonical. This migration is a defensive backfill for any legacy/manual
-- data that slipped in before the CHECK constraint existed, or via direct
-- inserts that used a different case/separator.

BEGIN;

UPDATE public.entity_relationship
SET cardinality = CASE
    WHEN cardinality IS NULL THEN NULL
    WHEN upper(regexp_replace(cardinality, '[-: ]', '_', 'g')) IN ('ONE_TO_ONE', '1_1') THEN 'ONE_TO_ONE'
    WHEN upper(regexp_replace(cardinality, '[-: ]', '_', 'g')) IN ('ONE_TO_MANY', '1_M', '1_N') THEN 'ONE_TO_MANY'
    WHEN upper(regexp_replace(cardinality, '[-: ]', '_', 'g')) IN ('MANY_TO_ONE', 'M_1', 'N_1') THEN 'MANY_TO_ONE'
    WHEN upper(regexp_replace(cardinality, '[-: ]', '_', 'g')) IN ('MANY_TO_MANY', 'M_M', 'N_M', 'M_N', 'N_N') THEN 'MANY_TO_MANY'
    ELSE cardinality
END
WHERE cardinality IS NOT NULL
  AND cardinality NOT IN ('ONE_TO_ONE', 'ONE_TO_MANY', 'MANY_TO_ONE', 'MANY_TO_MANY');

-- Rows the CASE above couldn't map to a canonical value would otherwise
-- violate entity_relationship_valid_cardinality; null them out rather than
-- leaving an invalid value; they will be re-resolved by the FK/relationship
-- discovery engines (see backend/internal/api/cardinality_resolver.go) on
-- next scan.
UPDATE public.entity_relationship
SET cardinality = NULL
WHERE cardinality IS NOT NULL
  AND cardinality NOT IN ('ONE_TO_ONE', 'ONE_TO_MANY', 'MANY_TO_ONE', 'MANY_TO_MANY');

COMMIT;
