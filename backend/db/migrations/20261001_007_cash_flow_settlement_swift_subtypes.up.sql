-- Ticket: deploy-blocked-20261001_007-subtype_registry-description
-- Live oms.subtype_registry has no "description" column (cols: id, root_object,
-- subtype_code, display_name, field_allowlist, parent_subtype_code, is_active,
-- tenant_id, created_at). Prior INSERT listed description and fatally blocked
-- ApplyMigrations / server boot. Dropped that column from the INSERT list.
DO $$ 
BEGIN
    IF EXISTS (
        SELECT FROM information_schema.tables 
        WHERE table_schema = 'oms' AND table_name = 'subtype_registry'
    ) THEN
        INSERT INTO oms.subtype_registry (
            id, root_object, subtype_code, display_name, field_allowlist
        )
        SELECT
            gen_random_uuid(),
            'cash_flow.settlement',
            'dvp_securities',
            'DVP Securities Settlement',
            '["transaction_ref", "isin", "quantity", "settlement_price", "settlement_amount", "currency", "settlement_date", "counterparty_bic", "safekeeping_account", "uetr", "swift_version", "msg_type", "bic_sender", "bic_receiver", "custodian_id", "settlement_status"]'::jsonb
        WHERE NOT EXISTS (
            SELECT 1 FROM oms.subtype_registry 
            WHERE root_object = 'cash_flow.settlement' AND subtype_code = 'dvp_securities'
        );

        INSERT INTO oms.subtype_registry (
            id, root_object, subtype_code, display_name, field_allowlist
        )
        SELECT
            gen_random_uuid(),
            'cash_flow.settlement',
            'free_of_payment',
            'Free of Payment Settlement',
            '["transaction_ref", "isin", "quantity", "settlement_date", "counterparty_bic", "safekeeping_account", "uetr", "swift_version", "msg_type", "bic_sender", "bic_receiver", "custodian_id", "settlement_status"]'::jsonb
        WHERE NOT EXISTS (
            SELECT 1 FROM oms.subtype_registry 
            WHERE root_object = 'cash_flow.settlement' AND subtype_code = 'free_of_payment'
        );
    END IF;
END $$;
