-- Reverts technical_name back to the malformed PascalCase values it had
-- before 20261016_013_fix_order_allocation_technical_names.up.sql. Only
-- provided for symmetry with this migration set's convention; there is no
-- reason to actually run this except to reproduce the pre-fix bug.
UPDATE business_object_fields
SET technical_name = field_name
WHERE bo_id = '0b5d5b5b-cef9-49d2-82ab-13457b90ee8c'
  AND field_name IN ('AccountID', 'AllocatedQuantity', 'CreatedAt', 'ID', 'OrderId', 'Status', 'TargetQuantity', 'UpdatedAt');
