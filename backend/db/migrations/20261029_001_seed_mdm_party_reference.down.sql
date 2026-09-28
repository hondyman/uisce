-- Removes the seeded reference rows. Only the gold-copy tenant rows are
-- deleted; if any tenant added their own reference rows, those are kept.

DELETE FROM mdm.fatca_crs_status       WHERE tenant_id = '00000000-0000-0000-0000-000000000001';
DELETE FROM mdm.party_document_type    WHERE tenant_id = '00000000-0000-0000-0000-000000000001';
DELETE FROM mdm.sanctions_list_source  WHERE tenant_id = '00000000-0000-0000-0000-000000000001';
DELETE FROM mdm.source_of_wealth_type  WHERE tenant_id = '00000000-0000-0000-0000-000000000001';
DELETE FROM mdm.risk_rating            WHERE tenant_id = '00000000-0000-0000-0000-000000000001';
DELETE FROM mdm.kyc_status             WHERE tenant_id = '00000000-0000-0000-0000-000000000001';
DELETE FROM mdm.party_relationship_type WHERE tenant_id = '00000000-0000-0000-0000-000000000001';
DELETE FROM mdm.party_role             WHERE tenant_id = '00000000-0000-0000-0000-000000000001';
DELETE FROM mdm.party_segment          WHERE tenant_id = '00000000-0000-0000-0000-000000000001';
DELETE FROM mdm.party_status           WHERE tenant_id = '00000000-0000-0000-0000-000000000001';
DELETE FROM mdm.party_sub_type         WHERE tenant_id = '00000000-0000-0000-0000-000000000001';
DELETE FROM mdm.party_type             WHERE tenant_id = '00000000-0000-0000-0000-000000000001';
