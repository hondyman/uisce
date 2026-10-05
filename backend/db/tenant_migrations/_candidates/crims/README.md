# Candidate tenant structure, generated from the gold copy's CRIMS datasource

**Not wired in.** The directory name starts with `_`, which `TenantRunner` rejects as an app name, so nothing can
apply it by accident. It exists so the structure can be reviewed before it replaces or sits beside
`db/tenant_migrations/orm/0001`.

## Where the template comes from
The gold-copy tenant (`alpha.public.tenants` where `gold_copy = true`: `northwind`, `99e99e99-...`) owns the
templates. Its CRIMS datasource (`CRIMS ORM Database`, `441f62c9-aad1-481d-9aab-62943fa11cd3`; a second one,
`... (bo binding consolidation)`, points at the same place) is `crims` on the dev host with
`config.schema = orm,vend,ref,mdm,cash_flow,wlth`. That list is the template, and `scripts/gen-tenant-ddl.py` defaults
to it. (The gold copy has seven more datasources; two others carry a database in their config, `Northwinds` ->
`northwinds`, and the rest have an empty config. Only the CRIMS one is generated here.)

Files are `NN_<schema>.up.sql` in apply order. They come from `pg_dump -s` of `crims`: schema only, no data. Do not
edit them: change the generator and regenerate (`REPORT.txt` lists everything the generator removed).

| File | Tables | Notes |
|---|---|---|
| `01_ref`, `02_vend`, `03_wlth` | 0 each | Declared by the datasource and empty in `crims`; created as empty schemas |
| `04_cash_flow` | 1 | `cash_flow.settlement`, which `orm.fund_order_settlement` refers to |
| `05_orm` | 80 | 78 tables (5 of them partitions) and 2 partitioned parents |
| `06_mdm` | 441 | Also adds the one reference from `orm` back into `mdm` |

## Source of truth: alpha metadata wins (owner direction, 2026-10-04)
A tenant structure is built from what **alpha** holds for the gold copy's datasource, **after the gold copy's scan is
synced**; never straight from the source database. The files in this directory are therefore a **fidelity oracle**
(`pg_dump` of `crims`), not the deploy source. `scripts/tenant-ddl-scan-coverage.py` compares alpha's scan with the
live source, read-only, and is the prototype of the "sync first" gate. Against the dev host on 2026-10-04 (scan of
2026-09-26):

| | Result |
|---|---|
| **Stale** (a rescan fixes it) | 16 tables and 238 columns in the source that the scan lacks (all `mdm`, 19 tables); 13 columns in the scan that no longer exist (`mdm.rating_scale`) |
| **Fidelity** where both have the column | **no mismatch** on 7,798 columns: data type, nullability, length, precision, scale, default |
| **Foreign keys / unique keys** | Recorded richly: composite columns, `ON DELETE`/`ON UPDATE`, deferrability, constraint names; unique groups with names and columns |
| **Not recorded at all** (a rescan does *not* add these) | 312 check constraints, 1,604 non-primary-key indexes, 2 partitioned tables (and their partitions), 7 functions, 2 triggers |

So a deploy built only from today's scan would create the tables, columns, keys and foreign keys, and **would lose every
check constraint, every secondary index, the partitioning, the functions and the triggers**. Until the scanner records
those too, they need a source; that is a decision, not a default.

## What the generator changes, and why
| Change | Count | Reason |
|---|---|---|
| Row-level security removed (`ENABLE`/`FORCE ROW LEVEL SECURITY`, `CREATE POLICY`) | 967 statements, 976 policies | One tenant per database (ADR-042). The policies key on `app.current_tenant`, which the tenant router never sets, so keeping `FORCE` would make every table read as empty. |
| psql meta-commands (`\restrict`) removed | | pg_dump 18 emits them; they are not SQL. |
| `CREATE SCHEMA IF NOT EXISTS`, `uuid-ossp` created first | | The only extension used (`uuid_generate_*`; the rest is core `gen_random_uuid()`). |

No foreign key leaves the six schemas, so none is removed.

## Order, and why mdm is last
`mdm` refers to `orm` (18 foreign keys) and `orm.account` refers back to `mdm.party`. A foreign key goes in the file of
the later schema it touches, so `05_orm` never refers to `mdm` and `06_mdm` ends with the reference back.

## Fidelity (checked against `crims` on an empty scratch database, all six schemas)
Columns 8,036, indexes 2,124 and function bodies identical; constraints identical except one foreign key on
`orm.fix_alert` whose auto-generated name differs (same columns, same target, same `ON DELETE CASCADE`), a `pg_dump`
artefact of a foreign key declared on a partitioned table. NOT NULL constraints were not compared (PG18 stores them as
constraint rows, PG16 does not). Zero rows, zero tables with RLS.

## What is not here
Data (reference data such as `mdm.calendar_day` is seeded separately, later), the `tenant_id` CHECK, grants, and the
wiring: provisioning takes one `App` today and would need an ordered list.
