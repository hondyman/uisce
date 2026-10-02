DO $down$
DECLARE gold uuid := '00000000-0000-0000-0000-000000000001'::uuid;
BEGIN
    DELETE FROM mdm.agreement_type      WHERE tenant_id = gold;
    DELETE FROM mdm.counterparty_role   WHERE tenant_id = gold;
    DELETE FROM mdm.counterparty_status WHERE tenant_id = gold;
    DELETE FROM mdm.counterparty_type   WHERE tenant_id = gold;
END $down$;
