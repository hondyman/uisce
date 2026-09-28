-- 20261026_011_orm_baskets.up.sql
-- basket_item.security_id → oms.security(id) (cross-schema).

CREATE TABLE IF NOT EXISTS orm.basket (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    basket_cd varchar(50) NOT NULL,
    name varchar(250) NOT NULL,
    basket_type varchar(30) NOT NULL,
    source_type varchar(30),
    source_id uuid,
    total_notional numeric(24,4),
    currency varchar(3),
    status varchar(20) DEFAULT 'DRAFT' NOT NULL,
    created_by uuid,
    approved_by uuid,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT basket_pkey PRIMARY KEY (id),
    CONSTRAINT uq_basket_cd UNIQUE (tenant_id, basket_cd),
    CONSTRAINT chk_b_type CHECK (basket_type IN (
        'CUSTOM','INDEX_REPLICATION','REBALANCE','PAIR','BASKET',
        'PORTFOLIO_TRANSITION','ETF_CREATE','ETF_REDEEM')),
    CONSTRAINT chk_b_source CHECK (source_type IS NULL OR source_type IN (
        'MODEL','INDEX','CUSTOM_UPLOAD','PORTFOLIO')),
    CONSTRAINT chk_b_status CHECK (status IN (
        'DRAFT','READY','ROUTED','PARTIAL','COMPLETED','CANCELLED'))
);

CREATE TABLE IF NOT EXISTS orm.basket_item (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    basket_id uuid NOT NULL,
    security_id uuid NOT NULL,
    side varchar(10) NOT NULL,
    target_qty numeric(18,4),
    target_notional numeric(18,4),
    target_weight_pct numeric(7,4),
    order_id uuid,
    sequence_number int4,
    status varchar(20) DEFAULT 'PENDING' NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT basket_item_pkey PRIMARY KEY (id),
    CONSTRAINT chk_bi_side CHECK (side IN ('BUY','SELL','SELL_SHORT','BUY_TO_COVER')),
    CONSTRAINT chk_bi_status CHECK (status IN ('PENDING','ROUTED','PARTIAL','FILLED','CANCELLED')),
    CONSTRAINT fk_bi_basket FOREIGN KEY (basket_id) REFERENCES orm.basket(id) ON DELETE CASCADE,
    CONSTRAINT fk_bi_security FOREIGN KEY (security_id) REFERENCES oms.security(id)
);
CREATE INDEX IF NOT EXISTS idx_bi_basket ON orm.basket_item (basket_id);
