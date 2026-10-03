# Alpha audit — 2026-10

**Purpose.** This file is where an alpha session records what the database
*actually* says. The finding that produced it is that the schema dump lied about
the database, so the database's real state is not recoverable from the
repository and must be captured here as a first-class artifact.

**Rule for whoever runs this: record raw output, not interpretation.** Paste
query results verbatim. If a query errors, paste the error. Do not summarise a
result into a conclusion — the conclusions go in the last section, separately,
so a later reader can check them against the raw output still sitting above.

**Also required:** for every question, mark how you determined the answer as
one of

- `verified-live` — ran the query, have the output
- `couldn't-look` — could not obtain it (no access, table missing, permission
  denied). This is an honest and useful answer. It is **not** the same as
  "the answer is no."

---

## Session header

Fill these in first.

| Field | Value |
|---|---|
| Date run | |
| Operator | |
| Environment (alpha / staging / other) | |
| Schema version or migration head | |
| `origin/main` SHA at session time | |

---

## Phase 1 — Q5, the invalidation check

Run **this first**. It can dissolve the entire finding, and if it does, phases
2 and 3 are not needed and the whole approach is re-derived.

```sql
SELECT edge_type, count(*) FROM public.catalog_edge
 WHERE upper(edge_type) IN ('METRIC_OF','USES_TERM','DERIVED_FROM')
 GROUP BY 1;
```

**Raw output:**

```

```

**Answer:** `verified-live` / `couldn't-look`

**If non-zero:** STOP. Some path writes these edges, the caller search missed
it, and reachability — not schema — is the open question. Do not proceed to
phases 2–3 on the assumption that the writer is dead.

---

## Phase 2 — Q1 + Q2, the schema mismatch

Settles whether the snapshot is faithful to the live database.

```sql
-- Q1
SELECT column_name, data_type, is_nullable, column_default
  FROM information_schema.columns
 WHERE table_schema = 'public' AND table_name = 'catalog_node'
 ORDER BY ordinal_position;

-- Q2
SELECT column_name, data_type, is_nullable, column_default
  FROM information_schema.columns
 WHERE table_schema = 'public' AND table_name = 'catalog_edge'
 ORDER BY ordinal_position;
```

**Q1 raw output:**

```

```

**Q2 raw output:**

```

```

**Answers:** `verified-live` / `couldn't-look`

### Questions these settle

- Does `catalog_node` have `node_id`? Does it have `node_key`?
- Does `catalog_edge` have a **text** `edge_type` column, or only
  `edge_type_id uuid`? Does it have `source_id`/`target_id`, or
  `source_node_id`/`target_node_id`? Does it have `from_node_id`/`to_node_id`?
- Is `catalog_edge` still partitioned?
- Are `id`, `source_node_id`, `target_node_id`, `edge_type_id` NOT NULL with no
  default? (Determines whether a partial INSERT can succeed at all.)

---

## Phase 3 — Q3 + Q4, input to the fix

Not needed for the diagnosis; needed to write a corrected writer.

```sql
-- Q3
SELECT id, edge_type_name, is_active
  FROM public.catalog_edge_types
 WHERE upper(edge_type_name) IN ('METRIC_OF','USES_TERM','DERIVED_FROM');

-- Q4
SELECT c.relname, pg_get_expr(c.relpartbound, c.oid) AS bounds
  FROM pg_inherits i
  JOIN pg_class c ON c.oid = i.inhrelid
  JOIN pg_class p ON p.oid = i.inhparent
 WHERE p.relname = 'catalog_edge'
 ORDER BY c.relname;
```

**Q3 raw output (book the `edge_type_id` UUIDs if returned):**

```

```

**Q4 raw output:**

```

```

**Answers:** `verified-live` / `couldn't-look`

---

## Four-file sweep

Same root cause as the main finding: pre-glossary `catalog_node` /
`catalog_edge` generation. **Two questions per file**, because a file can
reference a missing column and never be called — schema match alone is not
enough, and liveness must not be assumed.

Citations verified against `11b114167`.

| # | File | What it uses |
|---|---|---|
| 1 | `backend/internal/semanticmatch/resolver.go:75,77,80` | `INSERT INTO catalog_node (tenant_id, node_id, node_type, node_key, node_name, properties)`, `ON CONFLICT (tenant_id, node_key)`, `RETURNING node_id` |
| 2 | `backend/internal/catalog/sti_column_scanner.go:56,70` | `INSERT INTO catalog_node (node_id, tenant_id, node_type, node_key, node_name, qualified_path[, properties])` |
| 2b | `backend/internal/catalog/sti_column_scanner.go:80,83` | also `INSERT INTO catalog_edge (tenant_id, source_node_id, target_node_id, edge_type)` with text `edge_type` — **`COLUMN_OF`** |
| 3 | `backend/internal/catalog/subtype_bo_builder.go:36,50` | `INSERT INTO catalog_node (node_id, …, node_type, node_key, …)` |
| 3b | `backend/internal/catalog/subtype_bo_builder.go:60,63` | also `INSERT INTO catalog_edge (…, edge_type)` with text `edge_type` — **`ATTRIBUTE_OF`** |
| 4 | `backend/internal/bo/layout_service.go:104,108,110,112,114,116` | reads `tax.node_key`; joins `st.node_id`, `e_bt.from_node_id`, `e_bt.edge_type IN ('DEFINED_BY','DESCRIBES')`, `bt.node_id`, `e_bt.to_node_id`, `e_tax.from_node_id`, `e_tax.edge_type = 'MEMBER_OF'`, `tax.node_id` |

**Note on file 1:** `resolver.go:18-20` carries its own comment —
*"node_id TEXT PK (if yours is bigserial, drop node_id from the INSERT and keep
RETURNING node_id)"* and *"UNIQUE (tenant_id, node_key) (if absent, swap to
SELECT-then-INSERT)"*. The author was aware the schema varied and left
adaptation instructions. That is schema-adaptive debt rather than a plain
mistake, and it should affect the disposition.

**Note on files 2 and 3:** each writes **both** tables, so each fails on
`catalog_node` *and* `catalog_edge`. The original report cited only their
`catalog_node` inserts; the `catalog_edge` writes are `COLUMN_OF` and
`ATTRIBUTE_OF`, both text-`edge_type`.

### Worksheet — fill one row per row above

| # | Schema match? | Reachable? (cite a non-test `file:line`) | Cell | Disposition |
|---|---|---|---|---|
| 1 | | | | |
| 2 / 2b | | | | |
| 3 / 3b | | | | |
| 4 | | | | |

### Dispositions

| Cell | Meaning | Action |
|---|---|---|
| mismatched + reachable | **live defect** | fix or delete; each needs its own decision |
| mismatched + unreachable | dead code targeting a dead schema | **delete** |
| matched + reachable | healthy | no action |
| matched + unreachable | dead code | delete on general principle, or note and leave |

Two of the four cells end in deletion. `archguard` now catches recreation of a
classified database opener, so deletion of that file class needs no replacement
guard.

---

## Cube DDL validation

Same risk class: generated SQL validated by unit tests against a dump rather
than a live database. Run in this same session because the access is open.

- [ ] Confirm the DDL generator's target tables/columns against **Q1/Q2 live
      output**, not the snapshot.
- [ ] Run the generated DDL against alpha (or a scratch database built from
      Q1/Q2's shape) and record the result.
- [ ] **Capture the cube DDL golden corpus** in this pass, so the session leaves
      permanent assets rather than a description of one.

**Raw output / errors:**

```

```

---

## Scheduler observation

- [ ] Record what the scheduler currently runs, and whether anything invokes
      `MetricCatalogReconciler.ReconcileAll` from outside Go (a CronJob, a CLI,
      a startup hook). This closes the *couldn't-look* on external invocation
      that the Go-only search could not exclude.

**Findings:**

```

```

---

## Conclusions

Fill in **after** the raw output above is complete, and keep the reasoning
visible so it can be checked.

**Did the main finding survive?** (expected shape: Q5 zero, Q1/Q2 matching the
snapshot, Q3 returning three rows)

**If the snapshot was stale:** the dump and the migration log are one artifact
pair — re-run `backend/db/snapshots/regenerate.sh` and commit both.

**Does the metric lineage writer get wired, ported, or deleted?**

**What happened to the C2 calc-term lineage and reconciler backfill scope?** It
was parked pending these answers and may now be reshaped or dropped.

**Sweep dispositions:**

**Cube DDL status:**

**Follow-up work created by this session:**
