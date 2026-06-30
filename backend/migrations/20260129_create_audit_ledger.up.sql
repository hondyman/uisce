CREATE TABLE IF NOT EXISTS audit_ledger (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    transaction_type VARCHAR(255) NOT NULL,
    actor_id UUID NOT NULL,
    payload JSONB NOT NULL,
    previous_hash CHAR(64) NOT NULL,
    hash CHAR(64) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for fast retrieval of the last hash for a tenant
CREATE INDEX IF NOT EXISTS idx_audit_ledger_tenant_created ON audit_ledger (tenant_id, created_at DESC);

-- Index for lookup by ID
CREATE INDEX IF NOT EXISTS idx_audit_ledger_id ON audit_ledger (id);

ALTER TABLE audit_ledger DROP CONSTRAINT IF EXISTS chk_hash_length;
ALTER TABLE audit_ledger ADD CONSTRAINT chk_hash_length CHECK (length(hash) = 64);
ALTER TABLE audit_ledger DROP CONSTRAINT IF EXISTS chk_prev_hash_length;
ALTER TABLE audit_ledger ADD CONSTRAINT chk_prev_hash_length CHECK (length(previous_hash) = 64);
