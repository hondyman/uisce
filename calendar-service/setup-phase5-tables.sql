-- Create table for sync results tracking
CREATE TABLE IF NOT EXISTS google_sync_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    sync_id VARCHAR(255) UNIQUE NOT NULL,
    sync_status VARCHAR(50) NOT NULL DEFAULT 'pending',
    events_synced INTEGER DEFAULT 0,
    events_merged INTEGER DEFAULT 0,
    errors TEXT,
    started_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create index for faster lookups
CREATE INDEX IF NOT EXISTS idx_google_sync_user_id ON google_sync_results(user_id);
CREATE INDEX IF NOT EXISTS idx_google_sync_tenant_id ON google_sync_results(tenant_id);
CREATE INDEX IF NOT EXISTS idx_google_sync_status ON google_sync_results(sync_status);

-- Create table for token storage (alternative to in-memory)
CREATE TABLE IF NOT EXISTS oauth_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    provider VARCHAR(50) NOT NULL,
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    token_type VARCHAR(50),
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, provider)
);

-- Create index for token lookups
CREATE INDEX IF NOT EXISTS idx_oauth_tokens_user_provider ON oauth_tokens(user_id, provider);
