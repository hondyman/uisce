-- Local `orm` schema in `alpha` — a provisional, platform-local instance
-- of the 5-BO OMS model the catalog was already built against
-- (business_objects.driver_table_name = '/orm/<bo_key>', the MAPS_TO
-- catalog-edge column vocabulary under /orm/<bo_key>/<column>). Column
-- names/types/FK constraint names below were read directly off the live
-- catalog_node.properties for tenant 99e99e99-99e9-49e9-89e9-99e99e99e999
-- (data_type/precision/scale/is_nullable/foreign_key_constraints), not
-- guessed — this schema did not physically exist anywhere before this
-- migration, only its metadata did.
--
-- Explicitly NOT the same thing as:
--   - crims.orm (external tenant OLTP, Debezium CDC source — see
--     docs/oms-calc-engine-handoff.md)
--   - the `orm` *database* candidate under backend/db/orm/ (a separate,
--     from-scratch single-tenant "soul_trader" design, never verified live)
-- This is a third, clearly-scoped thing: dev/proof data for the unified
-- validation engine, matching the vocabulary the catalog already commits
-- to. See docs/unified-rule-engine-handoff.md item 10 for the full
-- stratum discussion — this migration resolves nothing about which
-- stratum is canonical for production; it only makes the platform-local
-- one real.

BEGIN;

CREATE SCHEMA IF NOT EXISTS orm;

CREATE TABLE orm."order" (
    id              UUID          NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    sec_id          NUMERIC(18,0) NOT NULL,
    side            VARCHAR(10)   NOT NULL,
    order_type      VARCHAR(20)   NOT NULL,
    status          VARCHAR(20)   NOT NULL DEFAULT 'NEW',
    target_qty      NUMERIC(18,4) NOT NULL,
    executed_qty    NUMERIC(18,4) NOT NULL DEFAULT 0,
    leaves_qty      NUMERIC(18,4) NOT NULL,
    limit_price     NUMERIC(18,9),
    avg_price       NUMERIC(18,9) DEFAULT 0,
    time_in_force   VARCHAR(10)   DEFAULT 'DAY',
    trade_date      DATE          NOT NULL,
    manager_id      UUID,
    trader_id       UUID,
    custom_attributes JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE orm.placement (
    id              UUID          NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    order_id        UUID          NOT NULL,
    broker_id       VARCHAR(20)   NOT NULL,
    venue_id        VARCHAR(20),
    routed_qty      NUMERIC(18,4) NOT NULL,
    executed_qty    NUMERIC(18,4) NOT NULL DEFAULT 0,
    leaves_qty      NUMERIC(18,4) NOT NULL,
    status          VARCHAR(20)   NOT NULL DEFAULT 'NEW',
    fix_clordid     VARCHAR(120),
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT placement_order_id_fkey FOREIGN KEY (order_id) REFERENCES orm."order"(id)
);

CREATE TABLE orm.order_allocation (
    id              UUID          NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    order_id        UUID          NOT NULL,
    account_id      VARCHAR(20)   NOT NULL,
    target_qty      NUMERIC(18,4) NOT NULL,
    allocated_qty   NUMERIC(18,4) NOT NULL DEFAULT 0,
    status          VARCHAR(20)   NOT NULL DEFAULT 'NEW',
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT order_allocation_order_id_fkey FOREIGN KEY (order_id) REFERENCES orm."order"(id)
);

ALTER TABLE orm."order" ADD CONSTRAINT chk_order_target_qty_positive CHECK (target_qty > 0);

CREATE TABLE orm.execution (
    id              UUID          NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    placement_id    UUID          NOT NULL,
    order_id        UUID          NOT NULL,
    exec_qty        NUMERIC(18,4) NOT NULL,
    exec_price      NUMERIC(18,9) NOT NULL,
    broker_id       VARCHAR(50),
    exec_time       TIMESTAMPTZ   NOT NULL,
    transact_time   TIMESTAMPTZ   NOT NULL,
    status          VARCHAR(20)   NOT NULL,
    broker_exec_id  VARCHAR(120),
    last_capacity   VARCHAR(1),
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ   DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT execution_placement_id_fkey FOREIGN KEY (placement_id) REFERENCES orm.placement(id),
    CONSTRAINT execution_order_id_fkey FOREIGN KEY (order_id) REFERENCES orm."order"(id)
);

CREATE TABLE orm.execution_allocation (
    id                  UUID          NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    execution_id        UUID          NOT NULL,
    order_allocation_id UUID          NOT NULL,
    alloc_exec_qty      NUMERIC(18,4) NOT NULL,
    alloc_exec_price    NUMERIC(18,9) NOT NULL,
    created_at          TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT execution_allocation_execution_id_fkey FOREIGN KEY (execution_id) REFERENCES orm.execution(id),
    CONSTRAINT execution_allocation_order_allocation_id_fkey FOREIGN KEY (order_allocation_id) REFERENCES orm.order_allocation(id)
);

-- Supporting reference data for the account-status/discretion compliance
-- rule (Order BO rule set). Not part of the catalog's 5-BO model — no
-- "account" BO exists in business_objects for this tenant — this is
-- minimal reference data the rule needs, analogous to a lookup table,
-- not a sixth BO.
CREATE TABLE orm.account (
    account_id      VARCHAR(20) NOT NULL PRIMARY KEY,
    status          VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    is_discretionary BOOLEAN    NOT NULL DEFAULT true
);

CREATE INDEX idx_orm_placement_order ON orm.placement(order_id);
CREATE INDEX idx_orm_order_allocation_order ON orm.order_allocation(order_id);
CREATE INDEX idx_orm_execution_placement ON orm.execution(placement_id);
CREATE INDEX idx_orm_execution_order ON orm.execution(order_id);
CREATE INDEX idx_orm_execution_allocation_execution ON orm.execution_allocation(execution_id);
CREATE INDEX idx_orm_execution_allocation_order_allocation ON orm.execution_allocation(order_allocation_id);

COMMIT;
