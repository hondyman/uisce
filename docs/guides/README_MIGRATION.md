# Migration helper: consolidate per-industry metrics into public.metrics

This folder contains a schema DDL, a migration SQL template, a sample mapping, and a small Python helper that generates INSERT statements for moving rows from per-schema `metrics` tables into a consolidated `public.metrics` table.

Files added:
- `sql/create_public_metrics.sql` — DDL for `public.metrics` and example indexes.
- `sql/migrate_template.sql` — an example single-schema migration statement you can adapt.
- `migrations/mapping.json` — sample schema -> industry UUID mapping (edit with real values).
- `scripts/migrate_metrics.py` — generator that prints migration SQL for all mapped schemas. It supports `--apply` to run SQL (requires `psycopg2` and `DATABASE_URL` env var).

Quick dry-run (prints SQL only):

```bash
python3 scripts/migrate_metrics.py --mapping migrations/mapping.json
```

To print the CREATE TABLE DDL as well:

```bash
python3 scripts/migrate_metrics.py --mapping migrations/mapping.json --create-table
```

To execute the generated SQL against a database (dangerous — backup first):

1. Install dependency (if needed):

```bash
pip3 install psycopg2-binary
```

2. Set `DATABASE_URL` and run with `--apply`:

```bash
export DATABASE_URL="postgresql://postgres:postgres@host.docker.internal:5432/alpha"
python3 scripts/migrate_metrics.py --mapping migrations/mapping.json --apply
```

Notes & caution:
- The generator assumes a source table `metrics` in each schema and that `metric_type` and `metric_time` columns exist. Non-common columns get bundled into `details` via `to_jsonb(m) - ARRAY[...]`.
- Review generated SQL carefully before applying. Test on a staging copy of the DB first.
- If preserving original ids is required, use `--preserve-ids` and ensure sequence adjustments are made.
