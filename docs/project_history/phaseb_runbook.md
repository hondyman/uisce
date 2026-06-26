Phase-B Canonicalization Runbook

Purpose
-------
This runbook accompanies `phaseb_playbook.sql`. It contains operator steps, pre-run checklist, maintenance window guidance, and rollback notes. Use it to safely perform Phase-B on staging or production.

Pre-Run Checklist
-----------------
- Take a full DB snapshot (pg_dump or physical snapshot). Store it off-host.
- Notify stakeholders and schedule maintenance window for destructive operations (DROP COLUMN requires ACCESS EXCLUSIVE locks).
- Ensure application traffic is quiesced or in read-only mode during the maintenance window (recommended).
- Copy `phaseb_playbook.sql` to the DB host or run via psql with a direct connection string.
- Edit `phaseb_playbook.sql` to fill any mapping placeholders and confirm target tables list.

How to run (staging/test)
-------------------------
1. Run only the pre-checks and backfill section first, verify counts:

   psql -U <user> -d <db> -f phaseb_playbook.sql

   Or run selectively:

   psql -U <user> -d <db> -c "\i phaseb_playbook.sql"

2. Verify the backfill results with the SELECT counts in the script. If there are remaining NULLs, either provide mapping rules or identify exemptions.

How to run (production - full destructive)
------------------------------------------
1. During maintenance window, ensure you have console access to the DB host and fast restore procedure.
2. Run the full playbook:

   psql -U <user> -d <db> -f phaseb_playbook.sql

3. Monitor psql output for notices, errors, and validation messages. If `VALIDATE CONSTRAINT` is deferred, finish validation during the window or plan to validate later.

Rollback & Recovery
-------------------
- Preferred rollback: restore DB from snapshot taken before running Phase-B.
- Partial rollback: for a specific new tenant_product_datasource created and applied, run the reverse UPDATE/DELETE snippet shown in the playbook.

Notes & Gotchas
----------------
- DROP COLUMN acquires an ACCESS EXCLUSIVE lock and may block/kill concurrent queries. Plan the maintenance window accordingly.
- For very large tables, perform backfills in batches to reduce WAL and avoid long-running transactions.
- Use `ALTER TABLE ... ADD CONSTRAINT ... NOT VALID` then `ALTER TABLE ... VALIDATE CONSTRAINT` to reduce initial locks; VALIDATE still may require scanning the table.

Post-Run Verification
---------------------
- Run the post-checks at the end of `phaseb_playbook.sql`.
- Run a few application-level smoke tests to confirm behavior.
- Observe logs for referential integrity or query errors.

If you want, I can now:
- (1) Tidy the test DB further (drop duplicated FK constraints), or
- (2) Create a PR with `phaseb_playbook.sql` and `phaseb_runbook.md` and a short PR description, or
- (3) Generate a one-shot production-ready SQL file with your chosen mapping rules and TPD ids filled in.

Tell me which you prefer and which environment (staging/production) to target for the final playbook.
