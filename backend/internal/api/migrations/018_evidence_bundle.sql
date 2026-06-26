-- Migration 018: Evidence Bundle System
-- Creates tables for regulator-facing upgrade evidence bundles

-- Evidence bundles track complete upgrade lifecycle
CREATE TABLE IF NOT EXISTS metadata.evidence_bundles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    upgrade_request_id UUID NOT NULL,
    old_version TEXT NOT NULL,
    new_version TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('in_progress', 'completed', 'failed', 'rolled_back')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    
    CONSTRAINT fk_upgrade_request FOREIGN KEY (upgrade_request_id) 
        REFERENCES metadata.upgrade_requests(id) ON DELETE CASCADE
);

CREATE INDEX idx_evidence_bundles_upgrade ON metadata.evidence_bundles(upgrade_request_id);
CREATE INDEX idx_evidence_bundles_status ON metadata.evidence_bundles(status);
CREATE INDEX idx_evidence_bundles_created ON metadata.evidence_bundles(created_at DESC);

COMMENT ON TABLE metadata.evidence_bundles IS 'Immutable evidence bundles for upgrade lifecycle auditing';
COMMENT ON COLUMN metadata.evidence_bundles.upgrade_request_id IS 'Links to the upgrade request that triggered this evidence collection';
COMMENT ON COLUMN metadata.evidence_bundles.status IS 'Lifecycle state: in_progress, completed, failed, rolled_back';

-- Stage evidence stores artifacts from each upgrade stage
CREATE TABLE IF NOT EXISTS metadata.stage_evidence (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bundle_id UUID NOT NULL,
    stage_name TEXT NOT NULL CHECK (stage_name IN ('diff', 'rebase', 'test', 'approval', 'deploy', 'rollback', 'audit')),
    status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'success', 'failed', 'skipped')),
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    actor_id TEXT,
    artifacts JSONB NOT NULL DEFAULT '[]'::jsonb,
    
    CONSTRAINT fk_evidence_bundle FOREIGN KEY (bundle_id) 
        REFERENCES metadata.evidence_bundles(id) ON DELETE CASCADE
);

CREATE INDEX idx_stage_evidence_bundle ON metadata.stage_evidence(bundle_id);
CREATE INDEX idx_stage_evidence_stage ON metadata.stage_evidence(stage_name);
CREATE INDEX idx_stage_evidence_status ON metadata.stage_evidence(status);
CREATE INDEX idx_stage_evidence_artifacts ON metadata.stage_evidence USING GIN (artifacts);

COMMENT ON TABLE metadata.stage_evidence IS 'Evidence artifacts collected at each upgrade stage';
COMMENT ON COLUMN metadata.stage_evidence.stage_name IS 'Upgrade stage: diff, rebase, test, approval, deploy, rollback, audit';
COMMENT ON COLUMN metadata.stage_evidence.artifacts IS 'Array of artifact metadata (type, storage_path, checksum, metadata)';

-- Approval requests for maker-checker governance
CREATE TABLE IF NOT EXISTS metadata.approval_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bundle_id UUID NOT NULL,
    requested_by TEXT NOT NULL,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    required_role TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'approved', 'rejected', 'expired')),
    approver_id TEXT,
    decision TEXT CHECK (decision IN ('approved', 'rejected')),
    justification TEXT,
    decided_at TIMESTAMPTZ,
    
    CONSTRAINT fk_approval_bundle FOREIGN KEY (bundle_id) 
        REFERENCES metadata.evidence_bundles(id) ON DELETE CASCADE
);

CREATE INDEX idx_approval_requests_bundle ON metadata.approval_requests(bundle_id);
CREATE INDEX idx_approval_requests_status ON metadata.approval_requests(status);
CREATE INDEX idx_approval_requests_role ON metadata.approval_requests(required_role);
CREATE INDEX idx_approval_requests_decided ON metadata.approval_requests(decided_at DESC);

COMMENT ON TABLE metadata.approval_requests IS 'Maker-checker approval workflow for deployment governance';
COMMENT ON COLUMN metadata.approval_requests.required_role IS 'Role required to approve (e.g., data_steward, compliance_officer)';
COMMENT ON COLUMN metadata.approval_requests.decision IS 'Final decision: approved or rejected';

-- Create upgrade_requests table if it doesn't exist (referenced by FK)
CREATE TABLE IF NOT EXISTS metadata.upgrade_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    old_version TEXT NOT NULL,
    new_version TEXT NOT NULL,
    target_tenants TEXT[] NOT NULL,
    requested_by TEXT NOT NULL,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status TEXT NOT NULL CHECK (status IN ('pending', 'in_progress', 'completed', 'failed')),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_upgrade_requests_status ON metadata.upgrade_requests(status);
CREATE INDEX idx_upgrade_requests_created ON metadata.upgrade_requests(requested_at DESC);

COMMENT ON TABLE metadata.upgrade_requests IS 'Upgrade pipeline requests initiated by users';
