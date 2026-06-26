-- Insert default admin user with properly hashed password
-- Password: Admin123!
-- Only insert if admin doesn't exist
INSERT INTO users (email, password_hash, name, role, permissions, is_active, is_core_admin, email_verified)
SELECT 
    'admin@semlayer.com',
    '$2a$10$7tGk5tDQKmmnQ7AKzOjlWufdFNgueXG.q4zRKPr8uZEWb4uoDeNhe',
    'System Administrator',
    'admin',
    ARRAY['read', 'write', 'admin', 'delete', 'manage_users'],
    true,
    true,
    true
WHERE NOT EXISTS (SELECT 1 FROM users WHERE email = 'admin@semlayer.com');
