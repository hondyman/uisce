-- Auth Schema Migration
-- Creates tables for production-ready JWT authentication
-- Generated: 2026-02-13

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    role VARCHAR(50) DEFAULT 'user',
    organization VARCHAR(255),
    permissions TEXT[], -- Array of permission strings
    is_active BOOLEAN DEFAULT true,
    is_core_admin BOOLEAN DEFAULT false,
    email_verified BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    last_login_at TIMESTAMP,
    metadata JSONB DEFAULT '{}'::jsonb
);

-- Create index on email for fast lookups
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_active ON users(is_active);

-- Refresh tokens table for token rotation
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(512) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    revoked BOOLEAN DEFAULT false,
    revoked_at TIMESTAMP,
    ip_address VARCHAR(45), -- Support IPv6
    user_agent TEXT
);

-- Create indexes for token lookups
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token ON refresh_tokens(token);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires ON refresh_tokens(expires_at);

-- Revoked tokens table for logout and security
CREATE TABLE IF NOT EXISTS revoked_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    jti VARCHAR(255) UNIQUE NOT NULL, -- JWT ID
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    revoked_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL, -- Original token expiry
    reason VARCHAR(100) -- 'logout', 'security', 'password_change', etc.
);

-- Create index for fast revocation checks
CREATE INDEX IF NOT EXISTS idx_revoked_tokens_jti ON revoked_tokens(jti);
CREATE INDEX IF NOT EXISTS idx_revoked_tokens_expires ON revoked_tokens(expires_at);

-- API keys table for service-to-service authentication
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_hash VARCHAR(255) UNIQUE NOT NULL, -- bcrypt hash of the key
    name VARCHAR(255) NOT NULL,
    tenant_id UUID,
    permissions TEXT[],
    rate_limit INTEGER DEFAULT 60, -- requests per minute
    created_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP,
    is_active BOOLEAN DEFAULT true,
    last_used_at TIMESTAMP,
    created_by UUID REFERENCES users(id)
);

-- Create indexes for API key lookups
CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_active ON api_keys(is_active);

-- Audit log table for authentication events
CREATE TABLE IF NOT EXISTS auth_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    event_type VARCHAR(50) NOT NULL, -- 'login', 'logout', 'register', 'token_refresh', 'password_reset', etc.
    ip_address VARCHAR(45),
    user_agent TEXT,
    success BOOLEAN NOT NULL,
    error_message TEXT,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes for audit queries
CREATE INDEX IF NOT EXISTS idx_auth_audit_user ON auth_audit_log(user_id);
CREATE INDEX IF NOT EXISTS idx_auth_audit_event ON auth_audit_log(event_type);
CREATE INDEX IF NOT EXISTS idx_auth_audit_created ON auth_audit_log(created_at);

-- Function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to update users.updated_at
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Function to clean up expired tokens
CREATE OR REPLACE FUNCTION cleanup_expired_tokens()
RETURNS void AS $$
BEGIN
    -- Delete expired refresh tokens
    DELETE FROM refresh_tokens WHERE expires_at < NOW();
    
    -- Delete expired revoked tokens (no longer needed after expiry)
    DELETE FROM revoked_tokens WHERE expires_at < NOW();
END;
$$ LANGUAGE plpgsql;

-- Insert default admin user (password: Admin123!)
-- Only insert if no users exist
INSERT INTO users (email, password_hash, name, role, permissions, is_active, is_core_admin, email_verified)
SELECT 
    'admin@semlayer.com',
    '$2a$10$rJ5Z8qO8zZhN0YqO8qN0Ye8qO8qO8qO8qO8qO8qO8qO8qO8qO8qO', -- bcrypt hash of 'Admin123!'
    'System Administrator',
    'admin',
    ARRAY['read', 'write', 'admin', 'delete', 'manage_users'],
    true,
    true,
    true
WHERE NOT EXISTS (SELECT 1 FROM users LIMIT 1);

-- Grant necessary permissions
GRANT SELECT, INSERT, UPDATE, DELETE ON users TO PUBLIC;
GRANT SELECT, INSERT, UPDATE, DELETE ON refresh_tokens TO PUBLIC;
GRANT SELECT, INSERT, UPDATE, DELETE ON revoked_tokens TO PUBLIC;
GRANT SELECT, INSERT, UPDATE, DELETE ON api_keys TO PUBLIC;
GRANT SELECT, INSERT ON auth_audit_log TO PUBLIC;

-- Add comment
COMMENT ON TABLE users IS 'User accounts for authentication and authorization';
COMMENT ON TABLE refresh_tokens IS 'Refresh tokens for JWT token rotation';
COMMENT ON TABLE revoked_tokens IS 'Revoked JWT tokens for logout and security';
COMMENT ON TABLE api_keys IS 'API keys for service-to-service authentication';
COMMENT ON TABLE auth_audit_log IS 'Audit log for authentication events';
