INSERT INTO swift_field_map (id, tenant_id, swift_version, msg_type, field_tag, semantic_field, required, transform_fn)
SELECT gen_random_uuid(), t.id, 'MT', 'MT541', ':20:', 'transaction_ref', true, NULL
FROM public.tenants t WHERE t.gold_copy = true
ON CONFLICT (tenant_id, swift_version, msg_type, field_tag) DO NOTHING;

INSERT INTO swift_field_map (id, tenant_id, swift_version, msg_type, field_tag, semantic_field, required, transform_fn)
SELECT gen_random_uuid(), t.id, 'MT', 'MT541', ':35B:', 'isin', true, 'parse_isin'
FROM public.tenants t WHERE t.gold_copy = true
ON CONFLICT (tenant_id, swift_version, msg_type, field_tag) DO NOTHING;

INSERT INTO swift_field_map (id, tenant_id, swift_version, msg_type, field_tag, semantic_field, required, transform_fn)
SELECT gen_random_uuid(), t.id, 'MT', 'MT541', ':36:', 'quantity', true, NULL
FROM public.tenants t WHERE t.gold_copy = true
ON CONFLICT (tenant_id, swift_version, msg_type, field_tag) DO NOTHING;

INSERT INTO swift_field_map (id, tenant_id, swift_version, msg_type, field_tag, semantic_field, required, transform_fn)
SELECT gen_random_uuid(), t.id, 'MT', 'MT541', ':90A:PRCT', 'settlement_price', false, NULL
FROM public.tenants t WHERE t.gold_copy = true
ON CONFLICT (tenant_id, swift_version, msg_type, field_tag) DO NOTHING;

INSERT INTO swift_field_map (id, tenant_id, swift_version, msg_type, field_tag, semantic_field, required, transform_fn)
SELECT gen_random_uuid(), t.id, 'MT', 'MT541', ':98A:SETT', 'settlement_date', true, 'parse_swift_date'
FROM public.tenants t WHERE t.gold_copy = true
ON CONFLICT (tenant_id, swift_version, msg_type, field_tag) DO NOTHING;

INSERT INTO swift_field_map (id, tenant_id, swift_version, msg_type, field_tag, semantic_field, required, transform_fn)
SELECT gen_random_uuid(), t.id, 'MT', 'MT541', ':95P:BUYR', 'counterparty_bic', true, NULL
FROM public.tenants t WHERE t.gold_copy = true
ON CONFLICT (tenant_id, swift_version, msg_type, field_tag) DO NOTHING;

INSERT INTO swift_field_map (id, tenant_id, swift_version, msg_type, field_tag, semantic_field, required, transform_fn)
SELECT gen_random_uuid(), t.id, 'MT', 'MT541', ':97A:SAFE', 'safekeeping_account', true, NULL
FROM public.tenants t WHERE t.gold_copy = true
ON CONFLICT (tenant_id, swift_version, msg_type, field_tag) DO NOTHING;

INSERT INTO swift_field_map (id, tenant_id, swift_version, msg_type, field_tag, semantic_field, required, transform_fn)
SELECT gen_random_uuid(), t.id, 'MT', 'MT541', ':19A:SETT', 'settlement_amount', true, 'parse_amount_ccy'
FROM public.tenants t WHERE t.gold_copy = true
ON CONFLICT (tenant_id, swift_version, msg_type, field_tag) DO NOTHING;

-- MT543 Mappings
INSERT INTO swift_field_map (id, tenant_id, swift_version, msg_type, field_tag, semantic_field, required, transform_fn)
SELECT gen_random_uuid(), t.id, 'MT', 'MT543', ':20:', 'transaction_ref', true, NULL
FROM public.tenants t WHERE t.gold_copy = true
ON CONFLICT (tenant_id, swift_version, msg_type, field_tag) DO NOTHING;

INSERT INTO swift_field_map (id, tenant_id, swift_version, msg_type, field_tag, semantic_field, required, transform_fn)
SELECT gen_random_uuid(), t.id, 'MT', 'MT543', ':35B:', 'isin', true, 'parse_isin'
FROM public.tenants t WHERE t.gold_copy = true
ON CONFLICT (tenant_id, swift_version, msg_type, field_tag) DO NOTHING;

INSERT INTO swift_field_map (id, tenant_id, swift_version, msg_type, field_tag, semantic_field, required, transform_fn)
SELECT gen_random_uuid(), t.id, 'MT', 'MT543', ':36:', 'quantity', true, NULL
FROM public.tenants t WHERE t.gold_copy = true
ON CONFLICT (tenant_id, swift_version, msg_type, field_tag) DO NOTHING;

INSERT INTO swift_field_map (id, tenant_id, swift_version, msg_type, field_tag, semantic_field, required, transform_fn)
SELECT gen_random_uuid(), t.id, 'MT', 'MT543', ':90A:PRCT', 'settlement_price', false, NULL
FROM public.tenants t WHERE t.gold_copy = true
ON CONFLICT (tenant_id, swift_version, msg_type, field_tag) DO NOTHING;

INSERT INTO swift_field_map (id, tenant_id, swift_version, msg_type, field_tag, semantic_field, required, transform_fn)
SELECT gen_random_uuid(), t.id, 'MT', 'MT543', ':98A:SETT', 'settlement_date', true, 'parse_swift_date'
FROM public.tenants t WHERE t.gold_copy = true
ON CONFLICT (tenant_id, swift_version, msg_type, field_tag) DO NOTHING;

INSERT INTO swift_field_map (id, tenant_id, swift_version, msg_type, field_tag, semantic_field, required, transform_fn)
SELECT gen_random_uuid(), t.id, 'MT', 'MT543', ':95P:SELL', 'counterparty_bic', true, NULL
FROM public.tenants t WHERE t.gold_copy = true
ON CONFLICT (tenant_id, swift_version, msg_type, field_tag) DO NOTHING;

INSERT INTO swift_field_map (id, tenant_id, swift_version, msg_type, field_tag, semantic_field, required, transform_fn)
SELECT gen_random_uuid(), t.id, 'MT', 'MT543', ':97A:SAFE', 'safekeeping_account', true, NULL
FROM public.tenants t WHERE t.gold_copy = true
ON CONFLICT (tenant_id, swift_version, msg_type, field_tag) DO NOTHING;

INSERT INTO swift_field_map (id, tenant_id, swift_version, msg_type, field_tag, semantic_field, required, transform_fn)
SELECT gen_random_uuid(), t.id, 'MT', 'MT543', ':19A:SETT', 'settlement_amount', true, 'parse_amount_ccy'
FROM public.tenants t WHERE t.gold_copy = true
ON CONFLICT (tenant_id, swift_version, msg_type, field_tag) DO NOTHING;

-- MT548 Mappings
INSERT INTO swift_field_map (id, tenant_id, swift_version, msg_type, field_tag, semantic_field, required, transform_fn)
SELECT gen_random_uuid(), t.id, 'MT', 'MT548', ':20:', 'transaction_ref', true, NULL
FROM public.tenants t WHERE t.gold_copy = true
ON CONFLICT (tenant_id, swift_version, msg_type, field_tag) DO NOTHING;

INSERT INTO swift_field_map (id, tenant_id, swift_version, msg_type, field_tag, semantic_field, required, transform_fn)
SELECT gen_random_uuid(), t.id, 'MT', 'MT548', ':25D:PSTA', 'settlement_status', true, NULL
FROM public.tenants t WHERE t.gold_copy = true
ON CONFLICT (tenant_id, swift_version, msg_type, field_tag) DO NOTHING;

INSERT INTO swift_field_map (id, tenant_id, swift_version, msg_type, field_tag, semantic_field, required, transform_fn)
SELECT gen_random_uuid(), t.id, 'MT', 'MT548', ':20C:SEME', 'sender_msg_ref', true, NULL
FROM public.tenants t WHERE t.gold_copy = true
ON CONFLICT (tenant_id, swift_version, msg_type, field_tag) DO NOTHING;

-- pacs.008 Mappings
INSERT INTO swift_field_map (id, tenant_id, swift_version, msg_type, field_tag, semantic_field, required, transform_fn)
SELECT gen_random_uuid(), t.id, 'MX', 'pacs.008', 'CdtTrfTxInf/EndToEndId', 'transaction_ref', true, NULL
FROM public.tenants t WHERE t.gold_copy = true
ON CONFLICT (tenant_id, swift_version, msg_type, field_tag) DO NOTHING;

INSERT INTO swift_field_map (id, tenant_id, swift_version, msg_type, field_tag, semantic_field, required, transform_fn)
SELECT gen_random_uuid(), t.id, 'MX', 'pacs.008', 'CdtTrfTxInf/Amt/InstdAmt', 'settlement_amount', true, 'parse_amount_ccy'
FROM public.tenants t WHERE t.gold_copy = true
ON CONFLICT (tenant_id, swift_version, msg_type, field_tag) DO NOTHING;

INSERT INTO swift_field_map (id, tenant_id, swift_version, msg_type, field_tag, semantic_field, required, transform_fn)
SELECT gen_random_uuid(), t.id, 'MX', 'pacs.008', 'CdtTrfTxInf/CdtrAgt/FinInstnId/BICFI', 'bic_receiver', true, NULL
FROM public.tenants t WHERE t.gold_copy = true
ON CONFLICT (tenant_id, swift_version, msg_type, field_tag) DO NOTHING;

INSERT INTO swift_field_map (id, tenant_id, swift_version, msg_type, field_tag, semantic_field, required, transform_fn)
SELECT gen_random_uuid(), t.id, 'MX', 'pacs.008', 'GrpHdr/MsgId', 'msg_id', true, NULL
FROM public.tenants t WHERE t.gold_copy = true
ON CONFLICT (tenant_id, swift_version, msg_type, field_tag) DO NOTHING;

INSERT INTO swift_field_map (id, tenant_id, swift_version, msg_type, field_tag, semantic_field, required, transform_fn)
SELECT gen_random_uuid(), t.id, 'MX', 'pacs.008', 'GrpHdr/CreDtTm', 'created_at', true, 'parse_iso_datetime'
FROM public.tenants t WHERE t.gold_copy = true
ON CONFLICT (tenant_id, swift_version, msg_type, field_tag) DO NOTHING;
