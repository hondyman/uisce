DO $down$
DECLARE t text;
    tables text[] := ARRAY[
        'counterparty_exposure','counterparty_due_diligence',
        'counterparty_agreement_term','counterparty_agreement',
        'counterparty_credit_limit','counterparty_credit_profile'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format('DROP TABLE IF EXISTS mdm.%I CASCADE', t);
    END LOOP;
END $down$;
