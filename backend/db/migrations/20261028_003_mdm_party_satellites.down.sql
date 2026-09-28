DO $down$
DECLARE t text;
    tables text[] := ARRAY[
        'party_name','party_identifier','party_address','party_contact',
        'party_individual','party_legal_entity','party_tax_residency'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format('DROP TABLE IF EXISTS mdm.%I CASCADE', t);
    END LOOP;
END $down$;
