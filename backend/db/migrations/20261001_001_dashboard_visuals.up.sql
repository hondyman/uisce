-- Migration: dashboard_visuals table for NL conversation -> dashboard persistence
-- Purpose: Store individual visuals created via natural language dashboard conversations

BEGIN;

-- Dashboard visuals table: stores each visual (chart) created in a conversation
CREATE TABLE IF NOT EXISTS dashboard_visuals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dashboard_id VARCHAR(255) NOT NULL REFERENCES dashboards(id) ON DELETE CASCADE,
    visual_type VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    query_spec JSONB NOT NULL DEFAULT '{}',
    visual_config JSONB NOT NULL DEFAULT '{}',
    position JSONB NOT NULL DEFAULT '{"x":0,"y":0,"width":4,"height":3}',
    compliance JSONB,
    decision_trace JSONB,
    created_from_conversation_id VARCHAR(255),
    created_from_visual_id VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for fast lookups by dashboard
CREATE INDEX IF NOT EXISTS idx_dashboard_visuals_dashboard ON dashboard_visuals(dashboard_id);

-- Index for conversation lineage (tracing visuals back to source conversation)
CREATE INDEX IF NOT EXISTS idx_dashboard_visuals_conversation ON dashboard_visuals(created_from_conversation_id);

-- Composite index for dashboard + position ordering
CREATE INDEX IF NOT EXISTS idx_dashboard_visuals_dashboard_position ON dashboard_visuals(dashboard_id, (position->>'y'), (position->>'x'));

-- Add constraint to ensure visual_type is valid
ALTER TABLE dashboard_visuals DROP CONSTRAINT IF EXISTS valid_visual_type;
ALTER TABLE dashboard_visuals ADD CONSTRAINT valid_visual_type
    CHECK (visual_type IN ('line', 'bar', 'pie', 'table', 'area', 'scatter', 'heatmap', 'metric'));

COMMENT ON TABLE dashboard_visuals IS 'Individual chart/visual instances created via NL dashboard conversations';
COMMENT ON COLUMN dashboard_visuals.query_spec IS 'JSONB: metrics, dimensions, filters, aggregation, SQL generated for this visual';
COMMENT ON COLUMN dashboard_visuals.visual_config IS 'JSONB: chart configuration (xAxis, yAxis, chartType, showLegend, etc.)';
COMMENT ON COLUMN dashboard_visuals.compliance IS 'JSONB: governance compliance status and violations for this visual';
COMMENT ON COLUMN dashboard_visuals.created_from_conversation_id IS 'Links visual back to the NL conversation that created it';
COMMENT ON COLUMN dashboard_visuals.created_from_visual_id IS 'Original visual ID from the conversation (before being saved)';

COMMIT;
