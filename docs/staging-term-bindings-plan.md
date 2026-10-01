# Rule coverage for staging and file loads

## Problem

Validation rules belong to a business object and read its field names
(e.g. `ReviewedBy` on `security_term_sheet`). A BO write aliases each field to
its bound physical column (`analytics.ResolveSemanticFieldMap`) before the
rules run. A pipeline `rule_check` step did not: rows mapped to staging columns
or straight from a file reached the rules in a different shape, and the engine
reads a missing field as `false` - so staging and file loads had no real rule
coverage, silently.

## Decisions (2026-09-26)

- A staging table binds to the business object it feeds in **dedicated
  tables** - `staging_bindings` (BO field → staging column) and
  `staging_binding_changes` - used only by rule checks and lineage.
  Not `business_object_binding`: that is where an object's records are read
  and written (one binding per tenant, object and backend, chosen by the
  record resolvers), so a staging binding there could become a record source.
- Creating or changing a staging binding goes through **maker-checker**
  (the message-catalog pattern): drafted by one person, approved by another,
  audited.
- `staging.ff_product` gets a **Product business object** over the MDM product
  golden table, then a binding to it.
- File loads need no mapping of their own: the Map step targets the sink's
  columns, and the sink's binding maps those to fields.
- Gold-copy bindings are inherited read-only; tenants add their own.

## Steps

1. **Fail loud (this change).**
   - `vm.FieldRefs` lists the fields a rule reads; BO writes and pipelines
     share it.
   - At run time a row missing a field a rule reads is rejected (BLOCK) with
     the field named - never evaluated.
   - At design time the grounding check compares each picked rule's fields
     with the fields reaching the step (map targets, file columns or BO
     fields) and flags rules of one object in front of a load into another.
2. **Staging bindings (this change).**
   - Tables with RLS: a tenant reads/writes its own and reads the gold copy's;
     its own binding wins. Messages in set 9300 (en/es/fr).
   - Maker-checker: propose → a second person approves (applied, re-checked
     against the binding it was proposed on) or rejects; the proposer can
     withdraw. Core (gold-copy) bindings need a platform administrator,
     tenant bindings a tenant administrator; impersonating administrators
     can't propose or approve.
   - API: `/api/staging-bindings` (list, resolve, changes, propose,
     approve, reject, withdraw).
   - A rule check in front of a staging load reads rows through the table's
     binding to its object; without an approved binding the editor flags it
     and a run doesn't start. The load still receives the rows unchanged.
3. **Product business object.** Scan `mdm.product` into the catalog, define
   the Product BO (fields, terms, `MAPS_TO`), bind `staging.ff_product`.
4. **UI.** The staging load step shows the object it is checked as and its
   field mapping, with suggestions; an approval queue for binding changes.
