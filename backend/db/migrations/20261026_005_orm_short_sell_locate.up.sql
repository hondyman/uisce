-- 20261026_005_orm_short_sell_locate.up.sql
-- account_id and broker_id left as plain UUID (orm.account PK is varchar;
-- orm.broker PK is varchar).

CREATE TABLE IF NOT EXISTS orm.short_sell_locate (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    security_id uuid NOT NULL,
    account_id uuid NOT NULL,
    locate_type varchar(20) NOT NULL,
    locate_status varchar(20) NOT NULL,
    requested_at timestamptz NOT NULL,
    confirmed_at timestamptz,
    expiry_at timestamptz,
    quantity_requested numeric(18,4),
    quantity_confirmed numeric(18,4),
    quantity_used numeric(18,4) DEFAULT 0,
    locate_reference varchar(100),
    broker_id uuid,
    locate_fee_bps numeric(10,4),
    is_easy_to_borrow bool,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT short_sell_locate_pkey PRIMARY KEY (id),
    CONSTRAINT chk_ssl_type CHECK (locate_type IN (
        'EASY_TO_BORROW','HARD_TO_BORROW','PRE_BORROW','RECALL',
        'BROKER_LOCATE','INTERNAL_LOCATE')),
    CONSTRAINT chk_ssl_status CHECK (locate_status IN (
        'REQUESTED','CONFIRMED','REJECTED','EXPIRED','USED','CANCELLED'))
);
CREATE INDEX IF NOT EXISTS idx_ssl_sec ON orm.short_sell_locate (security_id, locate_status);
CREATE INDEX IF NOT EXISTS idx_ssl_acct ON orm.short_sell_locate (account_id, locate_status);
CREATE INDEX IF NOT EXISTS idx_ssl_expiry ON orm.short_sell_locate (expiry_at) WHERE locate_status = 'CONFIRMED';
