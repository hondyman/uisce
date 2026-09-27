-- 20261026_012_orm_routing_rules.up.sql
-- broker_id left as plain UUID.

CREATE TABLE IF NOT EXISTS orm.routing_rule (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    rule_cd varchar(50) NOT NULL,
    name varchar(250) NOT NULL,
    priority int4 DEFAULT 100 NOT NULL,
    asset_class_cd varchar(20),
    sec_sub_typ_cd varchar(30),
    order_type varchar(20),
    side varchar(10),
    quantity_min numeric(18,4),
    quantity_max numeric(18,4),
    notional_min numeric(18,4),
    notional_max numeric(18,4),
    venue_mic varchar(4),
    broker_id uuid,
    routing_strategy varchar(30),
    algo_cd varchar(50),
    is_active bool DEFAULT true NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT routing_rule_pkey PRIMARY KEY (id),
    CONSTRAINT uq_rr_cd UNIQUE (tenant_id, rule_cd),
    CONSTRAINT chk_rr_side CHECK (side IS NULL OR side IN ('BUY','SELL','SHORT','COVER')),
    CONSTRAINT chk_rr_strategy CHECK (routing_strategy IS NULL OR routing_strategy IN (
        'DIRECT','SMART_ORDER_ROUTER','DARK_FIRST','LIT_FIRST',
        'VWAP','TWAP','IS','POV','MANUAL'))
);
CREATE INDEX IF NOT EXISTS idx_rr_priority ON orm.routing_rule (priority, is_active);
