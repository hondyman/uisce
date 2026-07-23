
CREATE TABLE IF NOT EXISTS tenant_ip_whitelist (
    tenant_id UUID NOT NULL,
    ip_address VARCHAR(45) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, ip_address)
);
