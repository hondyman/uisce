-- +goose Up
-- Add name column to public.users to satisfy backend login expectations.
ALTER TABLE public.app_user
ADD COLUMN IF NOT EXISTS name varchar(255);

-- Backfill name from existing fields where possible
UPDATE public.app_user SET name = email
WHERE name IS NULL OR name = '';

-- +goose Down
ALTER TABLE public.app_user DROP COLUMN IF EXISTS name;
