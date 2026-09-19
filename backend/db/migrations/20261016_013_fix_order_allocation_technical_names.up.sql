-- order_allocation's business_object_fields rows were authored with
-- technical_name copied verbatim from field_name/display_name (PascalCase,
-- e.g. "AccountID") instead of the real physical snake_case column on
-- orm.order_allocation (e.g. "account_id"). These fields have no
-- field_bindings row, so boresolver's resolveCatalogPhysicalColumn falls
-- back to technical_name to find the physical column - the malformed value
-- meant every query against this BO's fields (beyond the driving table's
-- own PK) failed to resolve, and the frontend's physical-column -> semantic
-- label lookup (case-insensitive on technical_name) also missed, leaving
-- Table/Form widgets showing raw uppercase DB column names instead of
-- display names. Confirmed against a live catalog_node scan of
-- /orm/order_allocation/* that these are the only 8 physical columns.
UPDATE business_object_fields
SET technical_name = v.correct_name
FROM (VALUES
    ('AccountID', 'account_id'),
    ('AllocatedQuantity', 'allocated_qty'),
    ('CreatedAt', 'created_at'),
    ('ID', 'id'),
    ('OrderId', 'order_id'),
    ('Status', 'status'),
    ('TargetQuantity', 'target_qty'),
    ('UpdatedAt', 'updated_at')
) AS v(field_name, correct_name)
WHERE business_object_fields.bo_id = '0b5d5b5b-cef9-49d2-82ab-13457b90ee8c'
  AND business_object_fields.field_name = v.field_name;
