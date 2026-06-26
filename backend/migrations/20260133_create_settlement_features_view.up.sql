DO $do$
BEGIN
  -- Only create view/materialized view if required source tables/columns exist
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'orders')
     AND EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'orders' AND column_name = 'order_id')
     AND EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'audit_ledger')
     AND EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'audit_ledger' AND column_name = 'entity_id') THEN

    -- Ensure we have an audit_ledger table if not exists
    CREATE TABLE IF NOT EXISTS audit_ledger (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        entity_id TEXT NOT NULL,
        entity_type TEXT NOT NULL,
        event_type TEXT NOT NULL,
        status TEXT,
        payload JSONB,
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

    CREATE INDEX IF NOT EXISTS idx_audit_ledger_entity ON audit_ledger(entity_id, event_type);

    -- Create the settlement features view
    CREATE OR REPLACE VIEW settlement_features_view AS
    SELECT
        o.order_id,
        o.customer_id,
        o.order_date,

        COALESCE((CASE WHEN al.status = 'SETTLEMENT_FAILED' THEN 1 ELSE 0 END), 0) AS settlement_failed,

        COALESCE((SELECT COUNT(*) FROM order_details od WHERE od.order_id = o.order_id), 0)::INTEGER AS line_item_count,
        (CASE WHEN o.ship_country <> c.country THEN 1 ELSE 0 END)::INTEGER AS is_cross_border,
        COALESCE(EXTRACT(EPOCH FROM (o.shipped_date - o.order_date)) / 86400.0, 0)::FLOAT AS order_to_ship_days,

        COALESCE(c.country, 'UNKNOWN') AS customer_country,
        COALESCE((SELECT COUNT(*) FROM orders o2 WHERE o2.customer_id = o.customer_id AND o2.order_date < o.order_date), 0)::INTEGER AS customer_trade_history_count,

        COALESCE((SELECT COUNT(*) FROM audit_ledger al2 WHERE al2.entity_id = o.customer_id::text AND al2.event_type = 'SETTLEMENT_STATUS' AND al2.status = 'SETTLEMENT_FAILED' AND al2.created_at < o.order_date), 0)::INTEGER AS customer_previous_fails,

        (CASE WHEN o.ship_postal_code IS NULL OR o.ship_postal_code = '' THEN 1 ELSE 0 END)::INTEGER AS is_missing_postal_code,
        (CASE WHEN o.shipped_date IS NULL THEN 1 ELSE 0 END)::INTEGER AS is_missing_ship_date,
        (CASE WHEN o.ship_address IS NULL OR o.ship_address = '' THEN 1 ELSE 0 END)::INTEGER AS is_missing_address,

        COALESCE(o.freight, 0)::FLOAT AS order_freight_cost,
        COALESCE((SELECT SUM(unit_price * quantity * (1 - COALESCE(discount, 0))) FROM order_details od WHERE od.order_id = o.order_id), 0)::FLOAT AS order_total_value,

        COALESCE(o.ship_via, 0)::INTEGER AS shipper_id,
        EXTRACT(DOW FROM o.order_date)::INTEGER AS order_day_of_week,
        EXTRACT(MONTH FROM o.order_date)::INTEGER AS order_month,
        COALESCE(EXTRACT(EPOCH FROM (o.required_date - o.order_date)) / 86400.0, 30)::FLOAT AS days_until_required
    FROM orders o
    LEFT JOIN customers c ON o.customer_id = c.customer_id
    LEFT JOIN audit_ledger al ON o.order_id::text = al.entity_id AND al.event_type = 'SETTLEMENT_STATUS'
    WHERE o.order_date < NOW() - INTERVAL '30 days';

    -- Create materialized view and supporting objects
    CREATE MATERIALIZED VIEW IF NOT EXISTS settlement_features_materialized AS
    SELECT * FROM settlement_features_view;

    CREATE UNIQUE INDEX IF NOT EXISTS idx_sfm_order_id ON settlement_features_materialized(order_id);

    CREATE OR REPLACE FUNCTION refresh_settlement_features()
    RETURNS void AS $$
    BEGIN
        REFRESH MATERIALIZED VIEW CONCURRENTLY settlement_features_materialized;
    END;
    $$ LANGUAGE plpgsql;

    COMMENT ON VIEW settlement_features_view IS 
    'Feature engineering view for ML-based settlement risk prediction. Contains trade complexity, counterparty history, data quality, and value features.';

  ELSE
    RAISE NOTICE 'Skipping settlement features creation: required tables/columns missing.';
  END IF;
END
$do$;
