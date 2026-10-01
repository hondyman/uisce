DO $down$
DECLARE t text;
    tables text[] := ARRAY[
        'party_type','party_sub_type','party_status','party_segment',
        'party_role','party_relationship_type','kyc_status','risk_rating',
        'source_of_wealth_type','sanctions_list_source','party_document_type',
        'fatca_crs_status'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format('DROP TABLE IF EXISTS mdm.%I CASCADE', t);
    END LOOP;
END $down$;
