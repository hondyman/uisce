-- 20261026_013_orm_intraday_pnl.up.sql

CREATE TABLE IF NOT EXISTS orm.pnl_intraday (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    account_id uuid NOT NULL,
    portfolio_id uuid,
    as_of_timestamp timestamptz NOT NULL,
    currency varchar(3) NOT NULL,
    realized_pnl numeric(24,4) DEFAULT 0,
    unrealized_pnl numeric(24,4) DEFAULT 0,
    total_pnl numeric(24,4) DEFAULT 0,
    income numeric(24,4) DEFAULT 0,
    commissions numeric(24,4) DEFAULT 0,
    fees numeric(24,4) DEFAULT 0,
    net_pnl numeric(24,4),
    market_value numeric(24,4),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT pnl_intraday_pkey PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_pnl_acct ON orm.pnl_intraday (account_id, as_of_timestamp);
CREATE INDEX IF NOT EXISTS idx_pnl_portfolio ON orm.pnl_intraday (portfolio_id, as_of_timestamp) WHERE portfolio_id IS NOT NULL;
