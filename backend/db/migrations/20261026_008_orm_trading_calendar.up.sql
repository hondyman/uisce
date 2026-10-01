-- 20261026_008_orm_trading_calendar.up.sql
-- market_id left as plain UUID (no orm.market).

CREATE TABLE IF NOT EXISTS orm.trading_session (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    market_id uuid,
    mic varchar(4),
    session_date date NOT NULL,
    session_type varchar(30) NOT NULL,
    open_time time,
    close_time time,
    time_zone varchar(100),
    is_early_close bool DEFAULT false NOT NULL,
    early_close_reason varchar(255),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT trading_session_pkey PRIMARY KEY (id),
    CONSTRAINT uq_ts UNIQUE (tenant_id, session_date, session_type, mic),
    CONSTRAINT chk_ts_type CHECK (session_type IN (
        'PRE_MARKET','OPENING_AUCTION','CONTINUOUS','CLOSING_AUCTION',
        'POST_MARKET','CLOSED','HALT','EARLY_CLOSE','LATE_OPEN'))
);
-- Replaces COALESCE(mic,'') expression index (PG rejects expr in UNIQUE)
CREATE UNIQUE INDEX IF NOT EXISTS uq_ts_mic_null ON orm.trading_session (tenant_id, session_date, session_type) WHERE mic IS NULL;
CREATE INDEX IF NOT EXISTS idx_ts_date ON orm.trading_session (session_date, mic);

CREATE TABLE IF NOT EXISTS orm.trading_halt (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    security_id uuid,
    mic varchar(4),
    halt_start timestamptz NOT NULL,
    halt_end timestamptz,
    halt_type varchar(30) NOT NULL,
    halt_reason varchar(500),
    is_resumed bool DEFAULT false NOT NULL,
    resumed_at timestamptz,
    source varchar(50),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT trading_halt_pkey PRIMARY KEY (id),
    CONSTRAINT chk_th_type CHECK (halt_type IN (
        'NEWS_PENDING','VOLATILITY','REGULATORY','TECHNICAL',
        'CIRCUIT_BREAKER','LULD','IPO','MERGER'))
);
CREATE INDEX IF NOT EXISTS idx_th_sec ON orm.trading_halt (security_id, halt_start) WHERE security_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_th_mic ON orm.trading_halt (mic, halt_start) WHERE mic IS NOT NULL;
