private_markets DB seed

This folder contains SQL to create and seed a `private_markets` schema and sample data.

How to run (local dev):

1. Create the database (one-time):

```bash
# if database doesn't exist
PGPASSWORD=postgres psql "postgresql://postgres:postgres@localhost:5432/postgres?sslmode=disable" -c "CREATE DATABASE private_markets;"
```

2. Apply the seed to the database:

```bash
PGPASSWORD=postgres psql "postgresql://postgres:postgres@localhost:5432/private_markets?sslmode=disable" -f db/private_markets/seed_private_markets.sql
```

3. Verify:

```bash
PGPASSWORD=postgres psql "postgresql://postgres:postgres@localhost:5432/private_markets?sslmode=disable" -c "SELECT * FROM private_markets.funds;" -c "SELECT * FROM private_markets.cash_flows ORDER BY cf_date;"
```

Notes:
- Adjust host/port/credentials as appropriate for your environment.
- The seed file is idempotent (uses IF NOT EXISTS and ON CONFLICT DO NOTHING).
