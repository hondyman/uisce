-- Maps a calculation (a semantic.objects row of type 'calculation') to the
-- StarRocks pre-aggregation/materialized view that must be refreshed when
-- that calculation's definition changes. Populated when a calculation is
-- registered against a pre-agg; consumed by the CDC-driven calc-refresh
-- worker (backend/cmd/security-sync-worker) on semantic.objects change
-- events, per PLAN_STUDIO_EVENTS_AUDIT.md item 8.
CREATE TABLE IF NOT EXISTS calculation_preagg_map (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL,
    calculation_id TEXT NOT NULL,
    preagg_id      TEXT NOT NULL,
    starrocks_mv   TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, calculation_id, preagg_id)
);

CREATE INDEX IF NOT EXISTS idx_calculation_preagg_map_lookup
    ON calculation_preagg_map (tenant_id, calculation_id);
