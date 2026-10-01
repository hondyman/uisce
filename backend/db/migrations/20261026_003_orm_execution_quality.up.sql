-- 20261026_003_orm_execution_quality.up.sql

CREATE TABLE IF NOT EXISTS orm.execution_quality (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    execution_id uuid NOT NULL,
    arrival_price numeric(18,9),
    arrival_price_time timestamptz,
    interval_vwap numeric(18,9),
    market_vwap numeric(18,9),
    slippage_bps numeric(12,6),
    slippage_amount numeric(18,4),
    implementation_shortfall_bps numeric(12,6),
    implementation_shortfall_amount numeric(18,4),
    opportunity_cost_bps numeric(12,6),
    opportunity_cost_amount numeric(18,4),
    participation_rate_pct numeric(7,4),
    benchmark_ref varchar(50),
    measurement_notes text,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT execution_quality_pkey PRIMARY KEY (id),
    CONSTRAINT uq_eq UNIQUE (tenant_id, execution_id),
    CONSTRAINT fk_eq_exec FOREIGN KEY (execution_id) REFERENCES orm.execution(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_eq_exec ON orm.execution_quality (execution_id);

CREATE TABLE IF NOT EXISTS orm.order_benchmark (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    order_id uuid NOT NULL,
    benchmark_type varchar(30) NOT NULL,
    benchmark_time timestamptz,
    benchmark_price numeric(18,9),
    benchmark_quantity numeric(18,4),
    benchmark_currency varchar(3),
    source varchar(50),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT order_benchmark_pkey PRIMARY KEY (id),
    CONSTRAINT chk_ob_type CHECK (benchmark_type IN (
        'ARRIVAL','DECISION','CLOSE','OPEN','VWAP','TWAP','INTERVAL',
        'IMPLEMENTATION_SHORTFALL','PRE_TRADE','POST_TRADE')),
    CONSTRAINT fk_ob_order FOREIGN KEY (order_id) REFERENCES orm."order"(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_ob_order ON orm.order_benchmark (order_id, benchmark_type);
