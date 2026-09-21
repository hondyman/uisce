# MDM validation rules (tier 1 and 2)

The catalog lives in `backend/internal/mdmrules`; `backend/cmd/seed_mdm_rules` writes it to the
gold-copy tenant through `ValidationRuleService.UpsertValidationRule` (domain `mdm`).

## How the rules are authored

* **Semantic terms, not columns.** A rule references terms such as `ThresholdAutoMatch`. At execution
  the engine resolves each term to the column that represents it under the active binding
  (`analytics.ResolveSemanticFieldMap`), so one rule applies consistently to every binding of a BO. An
  unresolvable term is a persisted `rule_error`, never a silent pass.
* **Binding scope.** A rule may carry `binding_ids` (`business_object_binding.bo_binding_id`); empty
  means every binding. The seeder's `ScopeCurrentBindings` scopes a rule to the bindings the BO has at
  seed time, so a binding added later does not inherit it. Today one rule uses it
  (`mdm.issuer.sourced_from_a_system`); the issuer BO has a single binding, so it behaves like an
  unscoped rule until a second binding exists.
* **Vocabulary.** `mdmrules.Vocabulary` is the term list per BO; the tests check every referenced term
  against it and evaluate every rule against passing and failing example records.

## What is deliberately not a rule

* `Status`, `Priority` and any `*IsActive` term: enumerations and UX controls own those.
* `IssuerId`: every `issuer_*` table maps its own `id` column to this term, so on tables that also
  have an issuer FK (`issuer_change_request`, `issuer_exception`) it is ambiguous which column a rule
  would read.
* `attribute_def`: no mapped terms, so no rule can reference it.

## Running it

```bash
DATABASE_URL=... go run ./cmd/seed_mdm_rules            # plan (writes nothing)
DATABASE_URL=... go run ./cmd/seed_mdm_rules -apply     # write; idempotent (keyed by rule name)
```

Rules are created with `governance_status = draft` by the service.

## Regenerating the vocabulary

```sql
SELECT bo.bo_key, f.field_name, f.field_role
FROM public.business_object_fields f
JOIN public.business_objects bo ON bo.id = f.bo_id
WHERE bo.tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1)
  AND bo.bo_key IN (<the BO keys>)
ORDER BY 1, f.display_order;
```

## Engine gaps (tracked as future work)

The engine (`vm.AdvancedEvaluator`) cannot express these yet, so no rule exists for them:

1. **String length / regex**: issuer `Lei` (`^[A-Z0-9]{20}$`, present in the custom_attributes schema),
   `Domicile`/`CountryCode` (2 characters).
2. **Date comparison**: `DissolutionDate >= IncorporationDate`, `ValidTo >= ValidFrom`,
   `EffectiveTo >= EffectiveFrom`. Expression operands must be numeric.
3. **String term-to-term comparison**: `IssuerIdA != IssuerIdB`, `EntityId1 != EntityId2`. Conditions
   compare a term to a literal; expressions are numeric only.
4. **Set membership (`in`)**: expressible only as an OR of equals today.

## Open items

* **Published golden record** (`Status = 'PUBLISHED'` requires `PublishedAt` and `PublishedBy`): not
  written. If the status string is not exactly right the rule would silently never fire. Confirm the
  golden-record status vocabulary first.
* **Mapping hygiene**: `is_active` maps to per-BO names (`IssuerIsActive` on five unrelated BOs,
  `MatchIsActive`, `DqIsActive`, ...), and the `issuer_*` `id` columns map to `IssuerId`. Neither blocks
  the current rules; both weaken "one term, one meaning".
