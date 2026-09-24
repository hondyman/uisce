-- Migration 011: Align pacs.008 field_map tags to ParseMX's emitted paths.
--
-- Contract verified against TestParseMX_Pacs008_FieldExtraction.
-- Three tags previously did not match parser output at all:
--   - EndToEndId (missing PmtId level)
--   - Amt/InstdAmt (wrong element; parser emits IntrBkSttlmAmt)
--   - CdtrAgt/FinInstnId/BICFI (parser emits Cdtr/FinInstnId/BICFI)
-- Two critical fields (uetr, settlement_date) were completely missing.

UPDATE vend.swift_field_map SET field_tag = 'Document/FIToFICstmrCdtTrf/CdtTrfTxInf/PmtId/EndToEndId'
WHERE msg_type = 'pacs.008' AND field_tag = 'CdtTrfTxInf/EndToEndId'
  AND tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1);

UPDATE vend.swift_field_map SET field_tag = 'Document/FIToFICstmrCdtTrf/CdtTrfTxInf/IntrBkSttlmAmt'
WHERE msg_type = 'pacs.008' AND field_tag = 'CdtTrfTxInf/Amt/InstdAmt'
  AND tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1);

UPDATE vend.swift_field_map SET field_tag = 'Document/FIToFICstmrCdtTrf/CdtTrfTxInf/Cdtr/FinInstnId/BICFI'
WHERE msg_type = 'pacs.008' AND field_tag = 'CdtTrfTxInf/CdtrAgt/FinInstnId/BICFI'
  AND tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1);

UPDATE vend.swift_field_map SET field_tag = 'Document/FIToFICstmrCdtTrf/GrpHdr/MsgId'
WHERE msg_type = 'pacs.008' AND field_tag = 'GrpHdr/MsgId'
  AND tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1);

UPDATE vend.swift_field_map SET field_tag = 'Document/FIToFICstmrCdtTrf/GrpHdr/CreDtTm'
WHERE msg_type = 'pacs.008' AND field_tag = 'GrpHdr/CreDtTm'
  AND tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1);

-- Insert missing settlement-critical fields for pacs.008
INSERT INTO vend.swift_field_map (id, tenant_id, swift_version, msg_type, field_tag, semantic_field, required, transform_fn)
SELECT gen_random_uuid(), t.id, 'MX', 'pacs.008', 'Document/FIToFICstmrCdtTrf/CdtTrfTxInf/PmtId/UETR', 'uetr', true, NULL
FROM public.tenants t WHERE t.gold_copy = true
ON CONFLICT (tenant_id, swift_version, msg_type, field_tag) DO NOTHING;

INSERT INTO vend.swift_field_map (id, tenant_id, swift_version, msg_type, field_tag, semantic_field, required, transform_fn)
SELECT gen_random_uuid(), t.id, 'MX', 'pacs.008', 'Document/FIToFICstmrCdtTrf/CdtTrfTxInf/IntrBkSttlmDt', 'settlement_date', true, 'parse_swift_date'
FROM public.tenants t WHERE t.gold_copy = true
ON CONFLICT (tenant_id, swift_version, msg_type, field_tag) DO NOTHING;
