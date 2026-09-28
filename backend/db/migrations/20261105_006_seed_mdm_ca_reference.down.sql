DO $down$
DECLARE gold uuid := '00000000-0000-0000-0000-000000000001'::uuid;
BEGIN
    DELETE FROM mdm.ca_entitlement_basis  WHERE tenant_id = gold;
    DELETE FROM mdm.ca_mandatory_type     WHERE tenant_id = gold;
    DELETE FROM mdm.ca_regulatory_regime  WHERE tenant_id = gold;
    DELETE FROM mdm.ca_tax_treatment      WHERE tenant_id = gold;
    DELETE FROM mdm.ca_payment_type       WHERE tenant_id = gold;
    DELETE FROM mdm.ca_election_type      WHERE tenant_id = gold;
    DELETE FROM mdm.ca_status             WHERE tenant_id = gold;
    DELETE FROM mdm.ca_event_type         WHERE tenant_id = gold;
END $down$;
