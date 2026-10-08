# orm tenant migrations

The `orm` schema reaches a tenant's own database through the migrations in this
directory. **Resolved with the ORM data move (Phase 4b); see ADR-042.**

- `0001_orm_schema.up.sql` creates all 33 ORM tables plus the `quote_default`
  partition, in dependency order, with `tenant_id uuid NOT NULL` on every table.
- It creates **no row-level security**. The policies it replaces existed to let a
  tenant read its own rows *or* the shared reference tenant's; in a per-tenant
  database the database is the boundary (ADR-030), and the reference rows are
  copied in instead.
- It does **not** create the three foreign keys into `oms` (`fk_ama_account`,
  `fk_bi_security`, `fk_pl_position`). `oms` stays in `alpha` and a foreign key
  cannot cross a database boundary. The columns remain; ADR-042 records why.

**Provenance.** The file is derived from the migration set, which is the schema
authority (ADR-024): `migrations/20260909_create_local_orm_schema.sql`,
`migrations/20260910_create_orm_broker.sql` and `db/migrations/20261026_002..014`,
with the `tenant_id` of `db/migrations/20261017_001_orm_tenant_order_chain.up.sql`
applied. Note that `migrations/` (no `db/`) is a directory **no runner reads** —
`internal/migrations/runner.go` reads `db/migrations` only — so that file is a
source to copy from, not a migration that runs.

`internal/migrations/ormmove` copies the data (ADR-043); this directory is schema
only. `orm.quote` is deliberately **not** copied — it is shared, high-volume
market data, and per-tenant copies would multiply storage by the tenant count to
hold identical bytes. See `ormmove.ExcludedTables` for how to change that.

Applied files are never edited. Add `0002_*.up.sql`.

- `0002_replica_identity_full.up.sql` sets REPLICA IDENTITY FULL on every table in
  the tenant's `orm` schema. It is a schema property, not an operational tweak: under
  the default identity a CDC `DELETE` carries no `tenant_id`, so per-tenant routing
  cannot dispatch it and every delete in every tenant database dead-letters while
  inserts and updates work perfectly. Applied there from the start, the failure cannot
  occur; applied later as a fix, it has already left undeletable rows behind. It runs
  over the whole schema rather than reading a publication, because a tenant database
  has no `orm_cdc_publication` of its own and depending on connector configuration
  that does not exist there would make it a no-op exactly when it matters.
