-- Ticket: deploy-blocked-20261001_007-subtype_registry-description
-- Live oms.subtype_registry has no "description" column and tenant_id NOT NULL
-- without default. Seed like 20260930_001: resolve gold_copy tenant, omit description.
DO $$
DECLARE
    gct UUID;
BEGIN
    IF EXISTS (
        SELECT FROM information_schema.tables
        WHERE table_schema = 'oms' AND table_name = 'subtype_registry'
    ) THEN
        SELECT id INTO gct FROM public.tenants WHERE gold_copy = true LIMIT 1;
        IF gct IS NULL THEN
            gct := '00000000-0000-0000-0000-000000000001'::UUID;
        END IF;

        INSERT INTO oms.subtype_registry (
            tenant_id, root_object, subtype_code, display_name, field_allowlist, is_active
        )
        SELECT
            gct,
            'cash_flow.settlement',
            'dvp_securities',
            'DVP Securities Settlement',
            '["transaction_ref", "isin", "quantity", "settlement_price", "settlement_amount", "currency", "settlement_date", "counterparty_bic", "safekeeping_account", "uetr", "swift_version", "msg_type", "bic_sender", "bic_receiver", "custodian_id", "settlement_status"]'::jsonb,
            true
        WHERE NOT EXISTS (
            SELECT 1 FROM oms.subtype_registry
            WHERE root_object = 'cash_flow.settlement' AND subtype_code = 'dvp_securities'
        );

        INSERT INTO oms.subtype_registry (
            tenant_id, root_object, subtype_code, display_name, field_allowlist, is_active
        )
        SELECT
            gct,
            'cash_flow.settlement',
            'free_of_payment',
            'Free of Payment Settlement',
            '["transaction_ref", "isin", "quantity", "settlement_date", "counterparty_bic", "safekeeping_account", "uetr", "swift_version", "msg_type", "bic_sender", "bic_receiver", "custodian_id", "settlement_status"]'::jsonb,
            true
        WHERE NOT EXISTS (
            SELECT 1 FROM oms.subtype_registry
            WHERE root_object = 'cash_flow.settlement' AND subtype_code = 'free_of_payment'
        );
    END IF;
END $$;
