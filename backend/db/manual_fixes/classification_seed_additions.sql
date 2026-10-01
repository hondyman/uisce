-- classification_seed_additions.sql
-- Purpose: the delta rows agreed in the G1-G12 review, corrected against Step 0
--          pre-flight reality (2026-09-23). These are ADDITIONS to the base
--          classification. The base itself (261 rows) does not exist on disk —
--          see step0_report.md, G20.
--
-- Corrections applied vs. the original proposal:
--   * 'oms.subtype_registry_seed' REMOVED  — table does not exist (G15)
--   * notes updated for issuer_master target = crims.mdm (G16: crims has no edm schema)
--   * classification_scheme rows NOT added here — they exist in crims, not alpha (G14)
--   * party / party_sub_type / party_relationship_type added because no base exists (G20)
--
-- NOT YET APPLIED. The migration schema does not exist on either DB.
-- This file is a deliverable artifact, pending the G20 decision.

-- ---------------------------------------------------------------------------
-- G1/G2: control-plane metadata that stays in alpha.oms
-- ---------------------------------------------------------------------------
-- ('oms','subtype_registry',      'oms','subtype_registry',      'METADATA','CATALOG',    100,'stays in alpha.oms; catalog pipeline reads it (backend/internal/catalog/subtype_registry.go)')
-- ('oms','migration_log',         'oms','migration_log',         'METADATA','MIGRATION',  100,'stays in alpha.oms; SHA-256 ledger for cmd/migrate')

-- ---------------------------------------------------------------------------
-- G6: issuer_master — the 7 cross-schema FKs from alpha.mdm point here.
--     Target corrected to crims.mdm (G16): crims has no edm schema.
--     Unifies the split issuer domain (G19): alpha holds the master,
--     crims already holds the 20 issuer_* satellites.
-- ---------------------------------------------------------------------------
-- ('edm','issuer_master',         'mdm','issuer_master',         'DATA','ISSUER',        300,'moves to crims.mdm; FK target for product/CA/counterparty; 7 cross-schema FKs rewritten to mdm.issuer_master')

-- ---------------------------------------------------------------------------
-- G12/G7: recon tables — destination deferred; stays in crims.mdm for now
-- ---------------------------------------------------------------------------
-- ('mdm','recon_statement_batches','mdm','recon_statement_batches','DATA_RECON','RECON', 300,'TBD: crims.reconciliation vs crims.mdm')
-- ('mdm','recon_breaks',          'mdm','recon_breaks',          'DATA_RECON','RECON',   310,'TBD')
-- ('mdm','recon_operations_config','mdm','recon_operations_config','DATA_RECON','RECON', 600,'TBD')
-- ('mdm','recon_operations_runs', 'mdm','recon_operations_runs', 'DATA_RECON','RECON',   600,'TBD')

-- ---------------------------------------------------------------------------
-- G4: party anchor + lookups — crims.mdm.party is empty (0 rows), so the full
--     seed path (Step B + Step C) is mandatory before any satellite insert.
--     NOTE: crims.mdm.xref already exists (G21) — inspect before creating
--     migration.party_xref; it may already be the cross-reference mechanism.
-- ---------------------------------------------------------------------------
-- ('mdm','party',                 'mdm','party',                 'DATA','PARTY',         300,'INSERT with xref mapping; see party_xref; crims.mdm.party currently 0 rows — full seed required')
-- ('mdm','party_sub_type',        'mdm','party_sub_type',        'DATA','PARTY',         310,'FK to party_type')
-- ('mdm','party_relationship_type','mdm','party_relationship_type','DATA','PARTY',        310,'')

-- ---------------------------------------------------------------------------
-- Open decisions these additions do NOT yet cover (see step0_report.md):
--   G18  fund_hierarchy -> public.tenants cross-DB FK
--   G20  base classification missing — full rebuild likely required
--   G22  singular vs plural fabric table names in alpha.public
--   G13  83 pre-existing crims.mdm tables; 2 name collisions + 2 near-collisions
--   G21  crims.mdm.xref may subsume migration.party_xref
-- ---------------------------------------------------------------------------

SELECT 1; -- placeholder so the file is valid SQL if executed; no rows written
