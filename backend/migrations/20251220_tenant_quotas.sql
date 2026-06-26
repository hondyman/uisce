-- Create table for storing tenant-specific resource quotas
CREATE TABLE IF NOT EXISTS tenant_quotas (
    tenant_id VARCHAR(255) NOT NULL,
    resource_name VARCHAR(255) NOT NULL, -- e.g., 'analytics_requests', 'storage_gb'
    limit_value BIGINT NOT NULL,         -- The maximum allowed value
    window_seconds INT NOT NULL DEFAULT 0, -- 0 for static limits (storage), >0 for rate limits (requests/window)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (tenant_id, resource_name)
);

-- Seed some default quotas for the 'default' tenant
INSERT INTO tenant_quotas (tenant_id, resource_name, limit_value, window_seconds)
VALUES 
    ('default', 'analytics_requests', 100, 60), -- 100 requests per minute
    ('default', 'storage_gb', 10, 0)            -- 10 GB limit (static)
ON CONFLICT (tenant_id, resource_name) DO NOTHING;
