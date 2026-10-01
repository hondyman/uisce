-- Ensure vend.swift_field_map exists before 20261001_011, which updates it.
--
-- 20261001_002/003 create and seed an unqualified swift_field_map. alpha applied
-- them while its search_path was (vend, public), so the table landed in vend,
-- where 011 and the swift adapter expect it. The runner now pins
-- search_path to (public, oms), so on a fresh database (backend-gated-tests) the
-- table lands in public and 011 fails with
-- relation "vend.swift_field_map" does not exist.
--
-- Only when vend.swift_field_map is absent: create it with public's shape and
-- RLS policy, and copy public's seed rows. On alpha the table exists, so this is
-- a no-op. Already-applied 002/003/011 are not edited.
DO $$
BEGIN
    IF to_regclass('vend.swift_field_map') IS NOT NULL THEN
        RETURN;
    END IF;

    CREATE SCHEMA IF NOT EXISTS vend;
    CREATE TABLE vend.swift_field_map (LIKE public.swift_field_map INCLUDING ALL);
    ALTER TABLE vend.swift_field_map ENABLE ROW LEVEL SECURITY;
    CREATE POLICY swift_field_map_isolation ON vend.swift_field_map
        USING (
            tenant_id = NULLIF(current_setting('app.tenant_id', 't'), '')::uuid
            OR tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1)
        );
    INSERT INTO vend.swift_field_map SELECT * FROM public.swift_field_map;
END
$$;
