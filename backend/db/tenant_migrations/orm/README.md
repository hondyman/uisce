# orm tenant migrations

Intentionally empty. The `orm` schema still lives in alpha (`alpha.orm.*`, defined across many
files in `backend/db/migrations`), and some of its tables reference alpha tables, which cannot be
foreign keys across databases. How the schema is carried into a tenant's own database, and what
happens to those references, is decided with the ORM data move (Phase 4b), not here.

Until then this directory only lets `App = "orm"` provisioning complete: the runner reports an
empty, done migration set, and the saga still gives the tenant database its role, binding and a
probe. The first `NNNN_*.up.sql` added here is applied to every tenant database provisioned with
the `orm` app; see `../README.md` for the rules (append-only, drift is fatal).
