-- Migration 007: IBOR Event Sourcing

-- 1. Position Events (The Source of Truth for Changes)
-- This table logs every change to a position (deltas).
CREATE TABLE IF NOT EXISTS position_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    account_id TEXT NOT NULL,
    asset_id TEXT NOT NULL,
    quantity_change DECIMAL(20, 8) NOT NULL,
    event_type TEXT NOT NULL, -- 'Trade', 'CorporateAction', 'Correction'
    event_date TIMESTAMP WITH TIME ZONE NOT NULL, -- Business Date (Valid Time)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() -- System Time
);

CREATE INDEX idx_position_events_account_asset ON position_events (account_id, asset_id, event_date);

-- 2. Position Snapshots (Daily Materialized View)
-- Stores the state at the end of each day.
-- In a real system, a nightly job would populate this table.
CREATE TABLE IF NOT EXISTS position_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    account_id TEXT NOT NULL,
    asset_id TEXT NOT NULL,
    quantity DECIMAL(20, 8) NOT NULL,
    snapshot_date DATE NOT NULL, -- The date this snapshot represents (EOD)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, account_id, asset_id, snapshot_date)
);

-- 3. Folding Function: get_position_as_of
-- Returns the position for an account/asset at a specific point in time.
-- Logic: Find latest snapshot <= target_date, then sum events between snapshot and target_date.
CREATE OR REPLACE FUNCTION get_position_as_of(
    p_account_id TEXT,
    p_asset_id TEXT,
    p_as_of_time TIMESTAMP WITH TIME ZONE
)
RETURNS DECIMAL AS $$
DECLARE
    v_snapshot_qty DECIMAL := 0;
    v_events_qty DECIMAL := 0;
    v_snapshot_date DATE;
BEGIN
    -- 1. Get latest snapshot on or before the target date
    SELECT quantity, snapshot_date
    INTO v_snapshot_qty, v_snapshot_date
    FROM position_snapshots
    WHERE account_id = p_account_id
      AND asset_id = p_asset_id
      AND snapshot_date <= p_as_of_time::DATE
    ORDER BY snapshot_date DESC
    LIMIT 1;

    -- If no snapshot found, assume 0 and start from beginning of time
    IF v_snapshot_date IS NULL THEN
        v_snapshot_date := '-infinity';
    END IF;

    -- 2. Sum events since snapshot up to target time
    -- Note: We cast snapshot_date to timestamp (start of day) and assume snapshot includes EOD,
    -- so we look for events strictly AFTER the snapshot date's EOD (or just use date comparison).
    -- For simplicity: snapshot_date is EOD. Events > snapshot_date are new.
    SELECT COALESCE(SUM(quantity_change), 0)
    INTO v_events_qty
    FROM position_events
    WHERE account_id = p_account_id
      AND asset_id = p_asset_id
      AND event_date > v_snapshot_date::TIMESTAMP WITH TIME ZONE
      AND event_date <= p_as_of_time;

    RETURN v_snapshot_qty + v_events_qty;
END;
$$ LANGUAGE plpgsql STABLE;
