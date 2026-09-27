-- 0015_survivorship_selection.up.sql
-- Survivorship, layered (see docs/mdm-mastering-blueprint.md):
--   1. the entity's source hierarchy (mdm.<prefix>_source_priority, by field group) is the default for
--      every attribute - no rule needed;
--   2. mdm.survivorship_rule overrides it per attribute (ranking, most recent, most frequent, ...);
--   3. a rule may name a SELECTION rule: a catalog rule (domain 'survivorship', the one rule engine)
--      each candidate value must satisfy to be chosen - evaluated against the other sources' values,
--      e.g. "within 20% of the median of the other sources".
--   4. stewards override / merge for true exceptions.
-- Run against crims. Additive.
--
-- Apply:
--   psql "$CRIMS_DSN" -1 -v ON_ERROR_STOP=1 -f 0015_survivorship_selection.up.sql

\set ON_ERROR_STOP on
BEGIN;

ALTER TABLE mdm.survivorship_rule
    ADD COLUMN IF NOT EXISTS selection_rule_id uuid,
    ADD COLUMN IF NOT EXISTS selection_mode    varchar(10) NOT NULL DEFAULT 'ENFORCE',
    ADD COLUMN IF NOT EXISTS on_none_selected  varchar(10) NOT NULL DEFAULT 'HOLD',
    ADD COLUMN IF NOT EXISTS selection_min_peers int NOT NULL DEFAULT 0;

DO $c$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'survivorship_rule_selection_mode_ck') THEN
        ALTER TABLE mdm.survivorship_rule ADD CONSTRAINT survivorship_rule_selection_mode_ck
            CHECK (selection_mode IN ('ENFORCE', 'FLAG'));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'survivorship_rule_none_selected_ck') THEN
        ALTER TABLE mdm.survivorship_rule ADD CONSTRAINT survivorship_rule_none_selected_ck
            CHECK (on_none_selected IN ('HOLD', 'ALLOW'));
    END IF;
END
$c$;

COMMENT ON COLUMN mdm.survivorship_rule.selection_rule_id IS
    'Catalog rule (domain survivorship) a candidate value must satisfy to be chosen; evaluated per candidate with value, source, age_hours, stale, rank, peers[], all[], previous, record.';
COMMENT ON COLUMN mdm.survivorship_rule.selection_mode IS
    'ENFORCE: a candidate failing the selection rule cannot win. FLAG: it still can, and an exception is raised.';
COMMENT ON COLUMN mdm.survivorship_rule.selection_min_peers IS
    'The selection rule applies only when at least this many other sources have a value (e.g. 2: no consensus check against a single peer).';
COMMENT ON COLUMN mdm.survivorship_rule.on_none_selected IS
    'When no candidate passes: HOLD keeps the previous value and asks a steward; ALLOW chooses among all and raises an exception.';

-- Product: which source-priority field group ranks each attribute when it has no survivorship rule.
SELECT set_config('app.current_tenant', '99e99e99-99e9-49e9-89e9-99e99e99e999', true);
UPDATE mdm.mastering_entity
   SET settings = settings || '{
         "default_field_group": "IDENTITY",
         "field_groups": {
           "name": "NAME", "legal_name": "NAME", "short_name": "NAME", "marketing_name": "NAME",
           "inception_date": "DATES", "live_date": "DATES", "closed_date": "DATES", "closed_to_new_date": "DATES",
           "product_type_cd": "CLASSIFICATION", "product_category_cd": "CLASSIFICATION",
           "product_sub_type_cd": "CLASSIFICATION", "strategy_cd": "CLASSIFICATION", "product_family_cd": "CLASSIFICATION",
           "aum": "AUM", "benchmark_name": "BENCHMARK"
         }
       }'::jsonb,
       updated_at = now()
 WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND entity_cd = 'PRODUCT';

COMMIT;
