DO $down$
DECLARE t text;
    tables text[] := ARRAY[
        'ca_conversion','ca_consent_solicitation','ca_sinking_fund',
        'ca_default_event','ca_bond_put','ca_bond_call',
        'ca_election_instruction','ca_election_deadline','ca_election_option'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format('DROP TABLE IF EXISTS mdm.%I CASCADE', t);
    END LOOP;
END $down$;
