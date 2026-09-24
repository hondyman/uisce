# Migration loss and reconstruction record

Audit date: 2026-09-21. Companion to `reconciliation-2026-09-21.tsv` (the 33 migration
files imported byte-for-byte in commit `83a3fc8c1`).

Six migrations are recorded as applied in `oms.migration_log` on the shared dev DB
whose SQL is not in any git ref or worktree, or whose content cannot be matched to
what was applied. Each is captured here so the next audit does not rediscover it.

| Migration | Applied (UTC) | Status | Notes |
|---|---|---|---|
| `20260824_001_catalog_sti_unique_constraints` | 2026-09-04 | **Reconstructed** | Live: `UNIQUE CONSTRAINT catalog_node_unique (tenant_id, node_type_id, qualified_path)`. Corroborated by the header of `20260904_001_capture_unmanaged_schema_drift`. High confidence for that constraint; cannot prove the original held nothing else. |
| `20261020_004_gold_copy_sync_datasource_grants` | 2026-09-18 21:35 | **Reconstructed** | Live delta beyond `20261016_002`/`_004`: `SELECT` on all `public` relations (1001/1001), default privileges (`uisce_gold_copy_sync=r/postgres`), `USAGE` on `public`. Reconstructed faithfully, not narrowed. See security follow-up. |
| `20261001_012_swift_inbound_channel` | 2026-09-18 | **Lost, accepted** | Live object: `vend.swift_inbound_channel` (10 columns). 0 Go references on this branch. |
| `20261001_013_swift_webhook_inbox` | 2026-09-18 | **Lost, accepted** | Live object: `vend.swift_webhook_inbox` (13 columns). 0 Go references on this branch. |
| `20261001_014_swift_refund_reversal_rows` | 2026-09-18 | **Lost, accepted** | No dedicated table found live (likely row seeding into an existing table). 0 Go references. |
| `20261019_workflow_definitions` | 2026-09-03 | **Unresolved** | A ref holds this filename with content that does not match the applied hash. Needs comparison against the live schema to decide which version is live. |

## How to read the statuses

- **Reconstructed**: the file on disk was rebuilt from live database state, not
  recovered. Its content hash will never match the log, so the parity guard lists it
  as `different` in `backend/db/PARITY_ALLOWLIST.tsv`. On the shared dev DB the runner
  logs "content has changed" and skips it; it only ever executes on a fresh database.
- **Lost, accepted**: no file is provided. The swift feature is partly present on this
  branch (migrations 001-011, 12 Go files) but nothing here references these three
  migrations' objects, so a reconstruction would create tables nothing uses. The shared
  DB still has the objects if they are needed as reference. If the swift inbound work
  resurfaces, start from `vend.swift_inbound_channel` and `vend.swift_webhook_inbox`.

## Open follow-ups (named so they do not evaporate)

1. **`uisce_gold_copy_sync` is broader than its purpose.** It is `LOGIN` + `BYPASSRLS` with
   `SELECT` on every table in `public` plus default privileges. It is the structurally
   cross-tenant role used by the datasource resolver, gold-copy sync and tenant
   provisioning, and global-admin cross-tenant support depends on it, so the
   reconstruction preserves it exactly. Narrowing it to the tables it needs is a
   separate, reviewed change.
2. **13 files whose disk content differs from the applied hash** (source of every
   "content has changed" startup warning). Three are explained (`20261021_001`,
   `20261021_002`, `20260914_014`); the other ten are untriaged. Triage each: fix the
   file to match what was applied, or record the edit as intentional.
3. **Unmanaged schema.** Objects in the live DB that no migration in any directory
   creates, including the constraints the MDM seed relies on (`uq_bob_tenant_bo_backend`,
   `chk_bob_temporal_mode`, `fk_bob_backend`, `uq_business_object_fields_term`, the
   `supports_multi_source` column) and the `physical_backend` table. Core tables come from
   the legacy `backend/migrations/` directory, which the runner does not track. Bringing
   these under managed migrations is a project of its own.
4. **`tenant_id_uuid_migration_backup`** is a schema-change record, not a data backup or a
   restore path. Plan to drop it about two weeks after 2026-09-21.
