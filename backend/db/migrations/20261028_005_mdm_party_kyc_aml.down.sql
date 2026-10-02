DO $down$
DECLARE t text;
    tables text[] := ARRAY[
        'kyc_profile','kyc_document','sanctions_screening','sanctions_hit',
        'pep_screening','adverse_media_check','fatca_crs_classification',
        'party_consent'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format('DROP TABLE IF EXISTS mdm.%I CASCADE', t);
    END LOOP;
END $down$;
