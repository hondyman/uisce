-- Minimal broker reference table for the OMS validation rule set (see
-- docs/oms-validation-implementation-spec section "Placement: Broker
-- active"). Same pattern as orm.account in 20260909_create_local_orm_schema.sql:
-- a small lookup table, not a BO - no "broker" BO exists in this catalog,
-- and this rule only needs a status column, not a full entity.
--
-- IMPORTANT: run this with search_path forced to public explicitly
-- (psql ... -c "SET search_path=public;" -f this_file), not a bare
-- psql -f - see docs/unified-rule-engine-handoff.md Session 5 addendum
-- (item 17) for why alpha's database-level search_path default
-- (`vend, public`) is a trap for exactly this kind of one-off migration.

BEGIN;

CREATE TABLE IF NOT EXISTS orm.broker (
    broker_id VARCHAR(20) NOT NULL PRIMARY KEY,
    status    VARCHAR(20) NOT NULL DEFAULT 'ACTIVE'
);

COMMIT;
