-- Retires (not deletes) the full catalog_validation_rules corpus.
--
-- Why: docs/validation_rules_migration_report.json shows all 233 rows
-- target entities absent from every tenant's business_objects catalog
-- (business_objects has exactly 5 rows system-wide - execution,
-- execution_allocation, order, order_allocation, placement - none of
-- which any of the 233 rules reference). The corpus was authored against
-- a schema (Customer/Employee/Product/trade_order/trade_execution) that
-- no longer exists. Separately, nothing in the running server ever
-- evaluated condition_json in the first place (TriggerValidationEngine,
-- the only reader, is never mounted - see the Phase 0 investigation in
-- session notes) - so retiring these changes zero live behavior.
--
-- Archive, not delete: is_active is flipped to false, condition_json and
-- every other column are left untouched, and one row per rule is written
-- to catalog_validation_rules_audit recording the retirement with a
-- reason and a pointer to the report that justified it. Fully reversible
-- by flipping is_active back and reading the audit trail for why it was
-- set false in the first place.

INSERT INTO catalog_validation_rules_audit (rule_id, tenant_id, action, old_values, new_values, changed_by)
SELECT
  id,
  tenant_id,
  'retired',
  jsonb_build_object('is_active', is_active),
  jsonb_build_object('is_active', false),
  'migration:20260909_retire_validation_rule_corpus - see docs/validation_rules_migration_report.json: corpus targets entities absent from the current business_objects catalog (0/233 field-resolvable), and no code path ever evaluated condition_json (TriggerValidationEngine unmounted)'
FROM catalog_validation_rules
WHERE is_active = true;

UPDATE catalog_validation_rules
SET is_active = false,
    updated_at = now()
WHERE is_active = true;

