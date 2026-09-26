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

- A staging table is an **additional, non-core binding** of the business object
  it feeds: `business_object_binding` (driving node = the staging table) plus
  `field_bindings` (BO field → staging column, `RESOLVED`). Resolution uses
  `ResolveSemanticFieldMapForBinding`, which fails loud on uncovered fields.
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
2. **Staging bindings.** Register staging tables and columns in the catalog;
   binding drafts, approval, audit; API + MCP. The rule-check step aliases
   rows through the binding of its downstream staging sink.
3. **Product business object.** Scan `mdm.product` into the catalog, define
   the Product BO (fields, terms, `MAPS_TO`), bind `staging.ff_product`.
4. **UI.** The staging load step shows the object it is checked as and its
   field mapping, with suggestions; an approval queue for binding changes.
