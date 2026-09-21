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

## Core and custom rules

* A rule authored in the **gold-copy tenant is core**: every tenant inherits it, read-only.
* A rule authored in a **tenant is custom**: it applies to that tenant only.
* `ValidationRuleService.ListByBO` returns the tenant's own rules plus the gold-copy tenant's, each marked
  `origin: core | custom`. The evaluator therefore holds a tenant to every core rule and to its own.
* A tenant can never override, shadow or switch off a core rule: writes only ever touch the caller's own
  nodes (`handleSetActive` filters on the caller's tenant), and a custom rule may not reuse a core rule's
  name or duplicate its conditions (`rejectCoreShadow`, `findDuplicateRuleAST`). If a same-name pair does
  exist, core wins.
* A tenant may add a custom rule on an inherited core BO, and scope it to the inherited gold-copy binding.
* Only the gold-copy tenant can retire a core rule (switch it off); the evaluator skips inactive rules.
  Before this the evaluator ignored `is_active`, so the UI switch did not actually stop enforcement.

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

## Verifying it end to end

`backend/cmd/verify_mdm_e2e` acts as a regular tenant against the real database and reports PASS/FAIL per
check: the tenant sees the gold-copy rules marked core; what else of the gold-copy tenant it can read; every
rule term resolves to a column under the BO's driving table; the stored rules accept and reject their
examples; a binding-scoped rule finds the inherited gold-copy binding; a tenant cannot shadow a core rule;
with `-write`, a temporary custom rule is created, seen only by its tenant, and removed.

```bash
DATABASE_URL='<the application role>' go run ./cmd/verify_mdm_e2e [-tenant <uuid>] [-write]
```

Use the application's role, not the owner: RLS does not apply to a superuser or a `BYPASSRLS` role, and
the command says so and skips the visibility checks if it is one.

If step 3 or 5 fails for many BOs at once, the tenant cannot read the gold-copy column nodes and `MAPS_TO`
targets it needs to resolve inherited terms; the rules themselves are fine.

## Open items

* **Golden-record status**: there is no rule keyed on a status value (`Status = 'PUBLISHED'`), because the
  status vocabulary is not defined anywhere in the repo and a wrong string would silently never fire.
  `mdm.issuer_golden_record.publication_recorded_completely` covers the integrity that matters without it:
  `PublishedAt` and `PublishedBy` must be present together or absent together.
* **Mapping hygiene**: `is_active` maps to per-BO names (`IssuerIsActive` on five unrelated BOs,
  `MatchIsActive`, `DqIsActive`, ...), and the `issuer_*` `id` columns map to `IssuerId`. Neither blocks
  the current rules; both weaken "one term, one meaning".
