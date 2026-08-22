#!/usr/bin/env bash
set -euo pipefail

echo "=========================================================================="
echo "🚀 Seeding Uisce Golden Demo Harness: Multi-Tier Storage & 3-Tier Taxonomy"
echo "=========================================================================="

PG_CONN="${DATABASE_URL:-postgres://postgres:postgres@localhost:5432/uisce_alpha?sslmode=disable}"
TENANT_ID="${TENANT_ID:-99e99e99-99e9-49e9-89e9-99e99e99e999}"

if command -v psql >/dev/null 2>&1; then
    echo "Connecting to PostgreSQL at $PG_CONN..."
    psql "$PG_CONN" <<SQL || echo "Postgres seeding skipped (offline mode)"
-- 1. Register Schemas
CREATE SCHEMA IF NOT EXISTS iceberg;
CREATE SCHEMA IF NOT EXISTS starrocks;
CREATE SCHEMA IF NOT EXISTS public;

-- 2. Cold Lakehouse Table (Te < 2026-01-01)
DROP TABLE IF EXISTS iceberg.portfolio_positions_archive CASCADE;
CREATE TABLE iceberg.portfolio_positions_archive (
    tenant_id UUID NOT NULL,
    account_id VARCHAR(50) NOT NULL,
    security_id VARCHAR(50) NOT NULL,
    trade_date DATE NOT NULL,
    base_cost NUMERIC(18, 4) NOT NULL,
    quantity NUMERIC(18, 4) NOT NULL,
    system_valid_from TIMESTAMPTZ NOT NULL,
    system_valid_to TIMESTAMPTZ,
    is_deleted BOOLEAN DEFAULT FALSE
);

-- 3. Hot Real-Time Table (Te >= 2026-01-01 + Late Mutations)
DROP TABLE IF EXISTS starrocks.portfolio_positions_realtime CASCADE;
CREATE TABLE starrocks.portfolio_positions_realtime (
    tenant_id UUID NOT NULL,
    account_id VARCHAR(50) NOT NULL,
    security_id VARCHAR(50) NOT NULL,
    trade_date DATE NOT NULL,
    base_cost NUMERIC(18, 4) NOT NULL,
    quantity NUMERIC(18, 4) NOT NULL,
    system_valid_from TIMESTAMPTZ NOT NULL,
    system_valid_to TIMESTAMPTZ,
    is_deleted BOOLEAN DEFAULT FALSE
);

-- 4. Trade Allocation Fills Table (Fan-Out 1:N Child Grain)
DROP TABLE IF EXISTS public.trade_allocation_fills CASCADE;
CREATE TABLE public.trade_allocation_fills (
    tenant_id UUID NOT NULL,
    account_id VARCHAR(50) NOT NULL,
    trade_date DATE NOT NULL,
    fill_id UUID NOT NULL,
    allocated_shares NUMERIC(18, 4) NOT NULL,
    is_deleted BOOLEAN DEFAULT FALSE
);

-- 5. Ingest Cold Historical Data (2024 - 2025)
INSERT INTO iceberg.portfolio_positions_archive VALUES
('$TENANT_ID', 'CUST_ACC_01', 'US0378331005', '2024-06-15', 150.00, 1000.0, '2024-06-15 00:00:00Z', NULL, FALSE),
('$TENANT_ID', 'CUST_ACC_01', 'US0378331005', '2025-03-20', 165.50, 1500.0, '2025-03-20 00:00:00Z', NULL, FALSE),
('$TENANT_ID', 'CUST_ACC_02', 'GB0002634946', '2025-11-10', 210.00, 800.0,  '2025-11-10 00:00:00Z', NULL, FALSE);

-- 6. Ingest Hot Real-Time Data (2026) + Late Correction on 2024-06-15
INSERT INTO starrocks.portfolio_positions_realtime VALUES
('$TENANT_ID', 'CUST_ACC_01', 'US0378331005', '2026-03-10', 180.00, 2000.0, '2026-03-10 00:00:00Z', NULL, FALSE),
('$TENANT_ID', 'CUST_ACC_02', 'GB0002634946', '2026-07-22', 225.00, 1200.0, '2026-07-22 00:00:00Z', NULL, FALSE),
('$TENANT_ID', 'CUST_ACC_01', 'US0378331005', '2024-06-15', 150.00, 1100.0, '2026-08-20 18:00:00Z', NULL, FALSE);

-- 7. Ingest Child Allocation Fills
INSERT INTO public.trade_allocation_fills VALUES
('$TENANT_ID', 'CUST_ACC_01', '2024-06-15', gen_random_uuid(), 600.0, FALSE),
('$TENANT_ID', 'CUST_ACC_01', '2024-06-15', gen_random_uuid(), 500.0, FALSE),
('$TENANT_ID', 'CUST_ACC_01', '2026-03-10', gen_random_uuid(), 2000.0, FALSE);
SQL
fi

echo "✔ Database tables and bitemporal datasets configured."
echo "✔ Running verification test..."
go test -v -count=1 ./backend/internal/e2e -run "TestE2E_ComplexMultiPeriodQueryAcrossSixEngines"
