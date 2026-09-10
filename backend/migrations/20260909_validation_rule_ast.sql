-- Adds rule_ast alongside condition_json on catalog_validation_rules.
--
-- Additive only: condition_json is never modified or dropped. rule_ast
-- holds the vm.RuleNode-shaped translation (see backend/cmd/
-- migrate_validation_rules) produced by the translator for rules whose
-- condition_json is a mechanically-translatable flat condition. Rules the
-- translator can't safely translate (needs-manual / field-unresolvable,
-- see the migration report) are left with rule_ast NULL - the evaluator
-- must treat NULL as "not yet on the unified engine", not as an error.
ALTER TABLE catalog_validation_rules
  ADD COLUMN IF NOT EXISTS rule_ast jsonb;

COMMENT ON COLUMN catalog_validation_rules.rule_ast IS
  'vm.RuleNode-shaped JSON (internal/rules/vm), translated from condition_json by cmd/migrate_validation_rules. NULL means not yet translated - condition_json remains the source of truth for those rows.';
