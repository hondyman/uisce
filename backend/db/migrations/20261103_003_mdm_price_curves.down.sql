-- 20261103_003_mdm_price_curves.down.sql
DO $down$
DECLARE t text;
    tables text[] := ARRAY[
        'fx_rate_master','vol_surface_point','vol_surface',
        'curve_snapshot','curve_point','curve_master'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format('DROP TABLE IF EXISTS mdm.%I CASCADE', t);
    END LOOP;
    DROP FUNCTION IF EXISTS mdm.immutable_tstz_text(timestamptz);
END $down$;
