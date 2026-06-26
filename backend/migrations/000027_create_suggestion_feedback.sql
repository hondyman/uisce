-- Migration: 000027_create_suggestion_feedback.sql
-- Created: 2025-10-14
-- Purpose: Create table to store user feedback on business term suggestions for ML training

-- Ensure pgcrypto (for gen_random_uuid) is available. This is idempotent.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Table to store user feedback on suggestions (accepts/rejects)
CREATE TABLE IF NOT EXISTS public.suggestion_feedback (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL,
    tenant_datasource_id uuid NOT NULL,
    semantic_term_id uuid NOT NULL, -- The semantic term that was being mapped
    business_term_id uuid, -- The business term node_id if accepted (NULL if rejected)
    business_term_name text NOT NULL, -- Name of the suggested business term
    action text NOT NULL CHECK (action IN ('accept', 'reject')),
    confidence numeric(5, 4), -- Original confidence score of the suggestion
    reason text, -- Optional: why user rejected (e.g., 'not relevant', 'wrong category')
    created_at timestamptz DEFAULT now(),
    created_by uuid, -- User who provided the feedback
    
    -- Foreign key to catalog_node for semantic_term_id
    CONSTRAINT fk_semantic_term FOREIGN KEY (semantic_term_id) 
        REFERENCES public.catalog_node(id) ON DELETE CASCADE,
    
    -- Foreign key to catalog_node for business_term_id (when accepted)
    CONSTRAINT fk_business_term FOREIGN KEY (business_term_id) 
        REFERENCES public.catalog_node(id) ON DELETE SET NULL
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_feedback_tenant ON public.suggestion_feedback (tenant_id, tenant_datasource_id);
CREATE INDEX IF NOT EXISTS idx_feedback_semantic_term ON public.suggestion_feedback (semantic_term_id);
CREATE INDEX IF NOT EXISTS idx_feedback_business_term ON public.suggestion_feedback (business_term_id);
CREATE INDEX IF NOT EXISTS idx_feedback_action ON public.suggestion_feedback (action);
CREATE INDEX IF NOT EXISTS idx_feedback_created_at ON public.suggestion_feedback (created_at DESC);

COMMENT ON TABLE public.suggestion_feedback IS 'Stores user feedback (accept/reject) on business term suggestions to improve ML models and suggestion algorithms.';
COMMENT ON COLUMN public.suggestion_feedback.action IS 'User action: accept or reject';
COMMENT ON COLUMN public.suggestion_feedback.confidence IS 'Original confidence score of the suggestion when it was presented';
COMMENT ON COLUMN public.suggestion_feedback.reason IS 'Optional reason for rejection to help improve suggestions';

-- View to analyze feedback patterns for improving suggestions
CREATE OR REPLACE VIEW public.suggestion_feedback_stats AS
SELECT
    sf.business_term_name,
    COUNT(*) FILTER (WHERE sf.action = 'accept') as accept_count,
    COUNT(*) FILTER (WHERE sf.action = 'reject') as reject_count,
    COUNT(*) as total_feedback,
    ROUND(
        COUNT(*) FILTER (WHERE sf.action = 'accept')::numeric / 
        NULLIF(COUNT(*), 0) * 100, 
        2
    ) as acceptance_rate,
    AVG(sf.confidence) FILTER (WHERE sf.action = 'accept') as avg_confidence_accepted,
    AVG(sf.confidence) FILTER (WHERE sf.action = 'reject') as avg_confidence_rejected,
    array_agg(DISTINCT sf.reason) FILTER (WHERE sf.reason IS NOT NULL) as rejection_reasons
FROM public.suggestion_feedback sf
GROUP BY sf.business_term_name
ORDER BY total_feedback DESC;

COMMENT ON VIEW public.suggestion_feedback_stats IS 'Aggregated statistics on suggestion feedback to identify patterns and improve suggestion quality.';
