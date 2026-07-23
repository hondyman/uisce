Edge table naming (singular vs plural)

Background
- During investigation we found an inconsistency between repository migrations and the actual database in the `alpha` dev DB.
- Migrations include a `catalog_edge_types` (plural) table definition, but the running DB contained `catalog_edge_type` (singular).

Impact
- Code and APIs that expect an `is_active` column on one of these tables failed with SQL errors (column does not exist).

Actions taken
- Added `backend/migrations/001000_add_is_active_to_catalog_edge_type.sql` to add `is_active` to the singular table, and applied it to the `alpha` DB.
- Made `backend/migrations/000999_add_is_active_to_catalog_edge_types.sql` conditional on table existence so it will not error if the plural table is absent.

Recommendations
1. Pick a canonical table name (singular or plural) and standardize project migrations to that name.
   - If you want the plural `catalog_edge_types` table, create a migration to create/rename tables accordingly and update code references.
   - If you want to keep the singular `catalog_edge_type`, update repository migrations and documentation to reference that name.
2. Run the project's full migrations against a fresh dev DB to ensure the intended schema is reproducible.
3. Add a short CI step that runs migrations against a temporary DB to catch these inconsistencies early.

How to reconcile (quick options)
- To keep the singular table and consolidate migrations, update repo migrations to target `catalog_edge_type`.
- To migrate to the plural table name, add a migration that renames `catalog_edge_type` -> `catalog_edge_types` (and adjust constraints/indexes), then update code and migrations.

If you want I can:
- Create a rename migration to convert the singular table to the plural one.
- Update migration tooling to run in CI to catch future divergence.
