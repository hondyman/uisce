DO $down$
DECLARE gold uuid := '00000000-0000-0000-0000-000000000001'::uuid;
BEGIN
    DELETE FROM mdm.product_lifecycle_event_type    WHERE tenant_id = gold;
    DELETE FROM mdm.product_target_market_type      WHERE tenant_id = gold;
    DELETE FROM mdm.product_share_class_type        WHERE tenant_id = gold;
    DELETE FROM mdm.product_document_type           WHERE tenant_id = gold;
    DELETE FROM mdm.product_registration_type       WHERE tenant_id = gold;
    DELETE FROM mdm.product_distribution_channel    WHERE tenant_id = gold;
    DELETE FROM mdm.product_status                  WHERE tenant_id = gold;
    DELETE FROM mdm.product_sub_type                WHERE tenant_id = gold;
    DELETE FROM mdm.product_type                    WHERE tenant_id = gold;
END $down$;
