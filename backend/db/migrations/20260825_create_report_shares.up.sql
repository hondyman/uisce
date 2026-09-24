-- =============================================================================
-- Report Shares table: stores direct / link / public share records
-- for vend.report_definitions reports.
-- Includes Phase-C columns: suspended_at, watermark, allow_export, allow_print
-- =============================================================================

CREATE TABLE IF NOT EXISTS public.report_shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Tenant + report scoping
    tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    report_id     UUID NOT NULL REFERENCES vend.report_definitions(id) ON DELETE CASCADE,
    shared_by     TEXT NOT NULL,   -- app_user.id of the owner who shared

    -- Share type
    share_type    VARCHAR(20) NOT NULL DEFAULT 'direct',  -- direct | link | public

    -- Recipient (for direct shares)
    recipient_id  TEXT,    -- app_user.id
    recipient_type VARCHAR(20),  -- user | team | role

    -- Link sharing
    share_link    VARCHAR(100),
    link_expiry   TIMESTAMP WITH TIME ZONE,
    password_hash VARCHAR(255),

    -- Permissions (restricted to 'view' for Phase 1; comment/edit/admin available later)
    permission    VARCHAR(20) NOT NULL DEFAULT 'view',  -- view | comment | edit | admin

    -- Restrictions
    allow_export  BOOLEAN NOT NULL DEFAULT true,
    allow_print   BOOLEAN NOT NULL DEFAULT true,
    watermark     BOOLEAN NOT NULL DEFAULT false,

    -- Soft-suspend (Phase C #5)
    suspended_at   TIMESTAMP WITH TIME ZONE,

    -- Timestamps
    created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at    TIMESTAMP WITH TIME ZONE,

    -- Unique constraint: one direct user-share per report
    CONSTRAINT report_shares_direct_uniq UNIQUE (report_id, recipient_id)
);

-- Indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_report_shares_tenant      ON public.report_shares(tenant_id);
CREATE INDEX IF NOT EXISTS idx_report_shares_report      ON public.report_shares(report_id);
CREATE INDEX IF NOT EXISTS idx_report_shares_recipient    ON public.report_shares(recipient_id);
CREATE INDEX IF NOT EXISTS idx_report_shares_shared_by    ON public.report_shares(shared_by);
CREATE INDEX IF NOT EXISTS idx_report_shares_share_link   ON public.report_shares(share_link)
    WHERE share_link IS NOT NULL;

-- =============================================================================
-- Report Share Audit Log: timestamped event log for all share actions
-- Phase C #14
-- =============================================================================

CREATE TABLE IF NOT EXISTS public.report_share_audit_log (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    share_id      UUID REFERENCES public.report_shares(id) ON DELETE CASCADE,
    tenant_id     UUID NOT NULL,
    report_id     UUID NOT NULL,
    actor_id      TEXT NOT NULL,   -- app_user.id who performed the action
    action        VARCHAR(30) NOT NULL,  -- share_created | share_revoked | share_suspended |
                                         -- share_resumed | permission_changed | report_cloned
    details       JSONB DEFAULT '{}',
    ip_address    INET,
    created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_share_audit_report  ON public.report_share_audit_log(report_id);
CREATE INDEX IF NOT EXISTS idx_share_audit_actor   ON public.report_share_audit_log(actor_id);
CREATE INDEX IF NOT EXISTS idx_share_audit_created ON public.report_share_audit_log(created_at DESC);

COMMENT ON TABLE public.report_shares          IS 'Report sharing configurations — direct, link, and public shares';
COMMENT ON TABLE public.report_share_audit_log IS 'Audit trail for all report share actions';
