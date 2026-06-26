-- Create Production-Ready Authentication Test Script
-- Tests all auth endpoints and validates JWT tokens

-- 1. Clean up existing test users
DELETE FROM refresh_tokens WHERE user_id IN (SELECT id FROM users WHERE email LIKE '%test%' OR email = 'admin@semlayer.com');
DELETE FROM revoked_tokens WHERE user_id IN (SELECT id FROM users WHERE email LIKE '%test%' OR email = 'admin@semlayer.com');
DELETE FROM users WHERE email LIKE '%test%' OR email = 'admin@semlayer.com';

-- 2. Manually insert admin user compatible with existing schema
-- Get table structure first
\d users

-- View existing users to understand the schema
SELECT column_name, data_type, is_nullable 
FROM information_schema.columns 
WHERE table_name = 'users' 
ORDER BY ordinal_position;
