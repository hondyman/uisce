-- Add column to users table for storing language
ALTER TABLE public.app_user
  ADD COLUMN IF NOT EXISTS language VARCHAR(10) DEFAULT 'en';

-- Optional index for faster lookups if needed
CREATE INDEX IF NOT EXISTS idx_users_language ON public.app_user(language);
