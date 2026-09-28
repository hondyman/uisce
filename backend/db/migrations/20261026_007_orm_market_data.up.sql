-- 20261026_007_orm_market_data.up.sql
-- quote is partitioned by RANGE(quote_time) with default partition only.
-- Follow-up: partition management via pg_partman or manual monthly partitions.

CREATE TABLE IF NOT EXISTS orm.quote (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    security_id uuid NOT NULL,
    venue_mic varchar(4),
    quote_time timestamptz NOT NULL,
    bid_price numeric(18,9),
    bid_size numeric(18,4),
    ask_price numeric(18,9),
    ask_size numeric(18,4),
    mid_price numeric(18,9),
    last_price numeric(18,9),
    last_size numeric(18,4),
    spread_bps numeric(12,6),
    quote_currency varchar(3),
    is_indicative bool DEFAULT false NOT NULL,
    source varchar(50),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT quote_pkey PRIMARY KEY (id, quote_time)
) PARTITION BY RANGE (quote_time);

CREATE TABLE IF NOT EXISTS orm.quote_default PARTITION OF orm.quote DEFAULT;
CREATE INDEX IF NOT EXISTS idx_quote_sec ON orm.quote_default (security_id, quote_time);
CREATE INDEX IF NOT EXISTS idx_quote_mic ON orm.quote_default (venue_mic, quote_time);
CREATE INDEX IF NOT EXISTS idx_quote_tenant ON orm.quote_default (tenant_id);

CREATE TABLE IF NOT EXISTS orm.market_data_snapshot (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    security_id uuid NOT NULL,
    as_of_date date NOT NULL,
    open_price numeric(18,9),
    high_price numeric(18,9),
    low_price numeric(18,9),
    close_price numeric(18,9),
    vwap numeric(18,9),
    volume numeric(24,4),
    trade_count int4,
    adjusted_close numeric(18,9),
    currency varchar(3),
    source varchar(50),
    is_final bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT market_data_snapshot_pkey PRIMARY KEY (id),
    CONSTRAINT uq_mds UNIQUE (tenant_id, security_id, as_of_date)
);
CREATE INDEX IF NOT EXISTS idx_mds_sec ON orm.market_data_snapshot (security_id, as_of_date);
