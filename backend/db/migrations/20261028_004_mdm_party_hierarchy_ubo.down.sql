DO $down$
DECLARE t text;
    tables text[] := ARRAY[
        'party_hierarchy','party_hierarchy_closure','party_ownership',
        'party_control','party_ubo','party_relationship','party_successor'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format('DROP TABLE IF EXISTS mdm.%I CASCADE', t);
    END LOOP;
END $down$;
