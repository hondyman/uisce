-- Migration for JIT Add-On Grant Audit Table
CREATE TABLE IF NOT EXISTS jit_addon_grant_audit (
    id UUID PRIMARY KEY,
    grant_id UUID REFERENCES jit_addon_grant(id),
    user_id TEXT,
    event_type TEXT, -- 'granted', 'expired', 'revoked', 'renewed'
    reason TEXT,
    occurred_at TIMESTAMP
);
