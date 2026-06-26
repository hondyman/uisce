# Quick Start: Remove AGE from Local PostgreSQL

## Option 1: Run the Migration (Recommended)

If you have a migration tool set up:
```bash
cd /Users/eganpj/GitHub/semlayer/backend
# Your migration tool will find and run:
# migrations/20260123_drop_age_extension.up.sql
```

## Option 2: Manual SQL Execution

```bash
export PGPASSWORD=postgres
psql -h host.docker.internal -U postgres -d alpha -f backend/migrations/20260123_drop_age_extension.up.sql
```

## Option 3: Use the Convenience Script

```bash
cd /Users/eganpj/GitHub/semlayer
./scripts/drop_age_local.sh
```

## Verify AGE is Removed

```bash
export PGPASSWORD=postgres
psql -h host.docker.internal -U postgres -d alpha -c "\dx"
```

You should NOT see `age` in the list of extensions.

## Verify Your Data is Intact

Check that your catalog tables still have data:
```bash
psql -h host.docker.internal -U postgres -d alpha -c "SELECT count(*) FROM catalog_node;"
psql -h host.docker.internal -U postgres -d alpha -c "SELECT count(*) FROM catalog_edge;"
```

## Done!

Your application now uses only relational tables for lineage tracking. No graph database extension needed!
