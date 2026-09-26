# MDM mastering blueprint — ingestion to golden record

Product is the first entity mastered end to end. Everything here is
configuration-driven so the next entity (Party, Security, Benchmark, …) needs
definitions, not code. It realises the Product Master ingest design
(`backend/db/manual_fixes/ingest_pipeline_design.md`) with platform pieces.

## The flow

```
vendor file ─▶ data pipeline ─▶ staging.<vendor>_<entity> ─▶ canonicalize ─▶ staging.<entity>_incoming
 (FactSet,      file → map →         (_load_run,              via the approved      rows keyed by
  BBG, RDP)     rule check* →         _source_row_num)        staging binding       business-object
                staging load                                   + named transforms    field names
                                                                                          │
      ┌───────────────────────────────────────────────────────────────────────────────────┘
      ▼
  match ─────────────▶ survivorship ────────────▶ publish ──────────────▶ golden record
  match rules,          per-field strategy +       versioned golden         mdm.<entity> (what the
  entity_xref,          source priority            record + per-field       business object reads),
  candidates → steward  (internal/mdm               provenance, dq score     <entity>_golden_record
  review queue           SurvivorshipEngine)

 * the rule check reads staging rows as the business object through the approved staging binding
```

One Temporal workflow per (tenant, entity) run chains the steps. Each step is
idempotent per load run. The run is scheduled by the one scheduler on business
calendars, or triggered from an enterprise scheduler (Tidal/Control-M).

## What is shared vs per tenant

| Gold-copy tenant (core, inherited read-only) | Each tenant |
|---|---|
| Business object, semantic terms, `MAPS_TO` column mappings | Its own vendor feeds and staging rows |
| Validation rules | Match candidates, steward decisions |
| Staging bindings (vendor table → object fields) | Golden records (`mdm.<entity>`, versioned) |
| Match rules, survivorship rules, source priority | Overrides and extensions of the core config |

A tenant binding or rule takes precedence over the core one for the same key.
Tenants never write back to the gold copy.

## One mapping

The **staging binding** (business-object field → staging column,
maker-checker, audited) is the only vendor mapping. Rules, canonical rows,
survivorship and lineage all speak business-object field names. Bindings gain
named transforms (`trim`, `upper`, `to_date`, `to_numeric`, type-map lookup,
vendor-specific parsers such as the Bloomberg ticker). The seeded
`mdm.product_field_mapping` rows are imported into bindings and that table is
retired.

## MDM validation rules at every stage

Rules live in the one rule engine (`internal/rules/vm`), scoped to the
business object (`domain: mdm`), and read business-object field names, so the
same rule applies wherever the entity's data is:

| Stage | Reads rows as | BLOCK | WARN |
|---|---|---|---|
| Staging (pipeline rule check) | the object, through the approved staging binding | row rejected, kept with its reason | row kept, warning recorded |
| Canonical (`<entity>_incoming`), before matching | the object's fields | not matched; exception for a steward | matched; lowers confidence |
| Golden record, before publishing | the survived values | version not published; exception | published; lowers the DQ score |

A rule that reads a field the rows don't carry fails loud: never an ordinary
pass or fail.

## Workflow approvals

One **Approvals inbox** for everything that needs a second person, fed by
pluggable approval sources rather than one queue per feature:

- message catalog changes, staging binding changes (already maker-checker)
- MDM validation rule changes
- golden-record overrides and steward match decisions (merge / split)

Each item shows who proposed it, what it changes (a diff), and the risk level.
Risk decides the stages (the `internal/approvals` workflow): low = one peer,
medium = peer + domain owner, high = peer + domain + compliance. Stages run as
a Temporal workflow so an approval survives restarts, can time out and can be
delegated. Every decision is audited. The proposer can never approve.

## Governance

- Changes to bindings, rules and golden overrides go through maker-checker
  (one person proposes, another approves) and are audited.
- Errors are catalog messages (en/es/fr) with a correlation id.
- Row-level security on every table. The tenant comes from the token only.
- Golden records are immutable: a re-run creates version N+1.

## Adding an entity — checklist

1. Scan its MDM tables into the catalog.
2. Create semantic terms and map them to the columns (`MAPS_TO`).
3. Create the business object in the gold copy (core).
4. Write its validation rules.
5. Build one data pipeline per vendor file into its staging table.
6. Bind each vendor staging table to the object (maker-checker).
7. Configure match rules, survivorship rules and source priority.
8. Schedule the mastering run.

## Slices (Product)

| # | Slice | Status |
|---|---|---|
| 1 | Rule checks fail loud; staging bindings with maker-checker | done (#157, #159) |
| 2 | Product business object over `mdm.product` (30 fields); FactSet binding approved; rules on staging rows | done |
| 3 | **Staging bindings UI**: bindings, mapping editor with suggestions, approval queue, history | this change |
| 4 | **Approvals inbox** (pluggable sources, risk-based stages on the approvals workflow); MDM rule changes under approval | |
| 4b | Binding transforms + type-map lookups; import `product_field_mapping` | |
| 5 | Canonicalize: staging → `product_incoming` through the binding; MDM rules on canonical rows | |
| 6 | Match engine: match rules, `entity_xref`, candidates, steward review queue | |
| 7 | Survivorship + versioned golden publish with provenance → `mdm.product`; MDM rules gate publishing | |
| 8 | Mastering workflow + schedule + feed-health dashboard | |
| 9 | Second and third source (BBG, RDP) to prove multi-source survivorship | |
| 10 | Steward overrides (maker-checker), exceptions, backfill tooling | |
