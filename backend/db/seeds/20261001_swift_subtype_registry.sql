-- SWIFT settlement subtype registry seed.
-- Mirrors backend/db/seeds/20260823_oms_subtype_registry.sql pattern.
-- These rows enable the catalog STI pipeline (Stage 2 in AGENTS.md) to
-- emit BUSINESS_OBJECT catalog nodes for dvp_securities and free_of_payment.
--
-- Idempotent: ON CONFLICT DO NOTHING on (root_object, subtype_code).
-- Applied by: POST /api/catalog/admin/sync-swift-subtypes

INSERT INTO oms.subtype_registry (
    id, root_object, subtype_code, display_name, description, field_allowlist
)
SELECT
    gen_random_uuid(),
    'cash_flow.settlement',
    'dvp_securities',
    'DVP Securities Settlement',
    'Delivery versus Payment securities settlement instruction (MT541 receive, MT543 deliver)',
    '["transaction_ref","isin","quantity","settlement_price","settlement_amount","currency",
      "settlement_date","counterparty_bic","safekeeping_account","uetr","swift_version",
      "msg_type","bic_sender","bic_receiver","custodian_id","settlement_status",
      "trade_date","place_of_settlement","common_reference"]'::jsonb
WHERE NOT EXISTS (
    SELECT 1 FROM oms.subtype_registry
    WHERE root_object = 'cash_flow.settlement' AND subtype_code = 'dvp_securities'
);

INSERT INTO oms.subtype_registry (
    id, root_object, subtype_code, display_name, description, field_allowlist
)
SELECT
    gen_random_uuid(),
    'cash_flow.settlement',
    'free_of_payment',
    'Free of Payment Settlement',
    'Securities delivery or receipt without a simultaneous cash leg (MT540 receive FOP, MT542 deliver FOP)',
    '["transaction_ref","isin","quantity","settlement_date","counterparty_bic","safekeeping_account",
      "uetr","swift_version","msg_type","bic_sender","bic_receiver","custodian_id",
      "settlement_status","trade_date","place_of_settlement","reason_code"]'::jsonb
WHERE NOT EXISTS (
    SELECT 1 FROM oms.subtype_registry
    WHERE root_object = 'cash_flow.settlement' AND subtype_code = 'free_of_payment'
);
