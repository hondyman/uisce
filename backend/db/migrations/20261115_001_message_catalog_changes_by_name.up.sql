-- Who, readably (the token's email at the time); the *_by IDs stay the
-- audit key.
--
-- Moved out of 20261027_001_message_catalog, where it was added after that
-- migration had been applied (fd73f661f), so existing databases never ran it.
-- On alpha the columns were added by hand; IF NOT EXISTS makes this a no-op there.
ALTER TABLE public.message_catalog_changes
    ADD COLUMN IF NOT EXISTS requested_by_name text,
    ADD COLUMN IF NOT EXISTS reviewed_by_name text;
