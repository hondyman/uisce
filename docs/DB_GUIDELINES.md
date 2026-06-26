DB Access Guidelines

Short version
- Prefer Hasura for straightforward CRUD endpoints and when you want rapid GraphQL exposure of tables, relationships, and row-level security.
- Prefer sqlx for performance-sensitive, complex queries, reporting, and places where fine-grained SQL control matters.
- Avoid introducing new GORM-based data access. Existing GORM usage may remain for low-risk areas but plan incremental replacement.

Why sqlx
- sqlx provides a thin, explicit mapping between SQL and Go structs, which makes it easier to reason about query performance and generated SQL.
- It reduces magic ORM behavior and gives you full control over indexes, joins, CTEs, and vendor-specific features.

Migration strategy (recommended)
1. Audit: identify services using GORM and their critical DB code paths.
2. Introduce an sqlx DB helper (see `services/compliance-engine/db/sqlx_db.go`).
3. Convert a single, high-value codepath to use sqlx (example: workflow ABAC policy initialization in compliance-engine).
4. Add tests for behavior parity and performance benchmarks.
5. Incrementally replace other GORM usages, one service/function at a time.

Notes
- Hasura is the right tool when you want GraphQL-first development and automatic schema tracking; use sqlx behind Hasura when you need complex logic that is awkward to express as remote schemas/actions.
- Keep migrations as authoritative SQL files (we already maintain `backend/db/migrations/*.sql`) so sqlx and Hasura map to the same schema.

Example
- See `services/compliance-engine/workflow_abac_sqlx.go` for an example of using sqlx to insert JSONB fields safely with `ON CONFLICT DO NOTHING`.

If you want, I can:
- Create a complete checklist and PR template for converting a service from GORM to sqlx.
- Start converting another specific service (name one: e.g., `portfolio-management` or `backend` handlers).
