CREATE TABLE IF NOT EXISTS ip_whitelist_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID,
    ip_address TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_tenant
        FOREIGN KEY(tenant_id)
        REFERENCES tenants(id)
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_ip_whitelist_tenant_id ON ip_whitelist_entries(tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_ip_whitelist_tenant_ip_unique ON ip_whitelist_entries(tenant_id, ip_address) WHERE tenant_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_ip_whitelist_global_ip_unique ON ip_whitelist_entries(ip_address) WHERE tenant_id IS NULL;

COMMENT ON TABLE ip_whitelist_entries IS 'Stores IP addresses for global and tenant-specific whitelists.';
COMMENT ON COLUMN ip_whitelist_entries.tenant_id IS 'NULL for global entries, or references a specific tenant.';
COMMENT ON COLUMN ip_whitelist_entries.ip_address IS 'The whitelisted IP address, CIDR range, or wildcard pattern.';