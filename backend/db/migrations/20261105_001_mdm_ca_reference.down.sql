DO $down$
DECLARE t text;
    tables text[] := ARRAY[
        'ca_entitlement_basis','ca_mandatory_type','ca_regulatory_regime',
        'ca_tax_treatment','ca_payment_type','ca_election_type',
        'ca_status','ca_event_type'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format('DROP TABLE IF EXISTS mdm.%I CASCADE', t);
    END LOOP;
END $down$;
