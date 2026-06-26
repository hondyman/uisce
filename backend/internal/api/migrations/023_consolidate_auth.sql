-- Add password fields to public.users to consolidate authentication
ALTER TABLE public.users
ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255),
ADD COLUMN IF NOT EXISTS salt VARCHAR(255);

-- Optional: Migrate existing passwords if needed
-- This is a best-effort migration for known users if they exist in both tables
UPDATE public.users u
SET 
    password_hash = a.password_hash,
    salt = a.salt
FROM private_markets_user_auth a
WHERE u.id = a.user_id;
