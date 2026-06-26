# Health check

This repository includes a lightweight health check script to validate the local development stack.

- `scripts/health_check.sh` checks:
  - API Gateway: `http://localhost:8000/health`
  - Hasura: `http://localhost:8081/healthz` (falls back to `http://localhost:8080/healthz`)
  - Minimal GraphQL query via the gateway at `http://localhost:8000/api/graphql`

Usage:

```bash
chmod +x ./scripts/health_check.sh
./scripts/health_check.sh
```

Migrations:

- `hasura/migrations/20250914_add_fks/up.sql` contains idempotent SQL to add the missing foreign keys that Hasura expected (fabric_defn → tenants and fabric_defn → tenant_product_datasource).

CI:

- `.github/workflows/health_check.yml` runs docker compose and the health script on push to `main` (or via manual dispatch).
