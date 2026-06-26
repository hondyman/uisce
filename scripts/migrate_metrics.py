#!/usr/bin/env python3
"""
Migration helper: generate INSERT statements to move per-schema metrics into public.metrics.

Usage:
  python3 scripts/migrate_metrics.py --mapping migrations/mapping.json

Options:
  --mapping  Path to JSON mapping file: {"schema_name":"industry_uuid", ...}
  --create-table  Print the CREATE TABLE DDL
  --apply    Actually execute statements against DATABASE_URL (requires psycopg2 and DATABASE_URL env var). Default is dry-run (print SQL).
  --preserve-ids  Attempt to preserve source ids (will include id in INSERT and set sequence afterwards). Use with caution.

The script defaults to printing SQL to stdout so you can review before applying.
"""
import argparse
import json
import os
import sys
from textwrap import dedent

COMMON_COLS = ['metric_type','metric_time','value','tags','id','created_at','updated_at']

DDL = '''
CREATE SCHEMA IF NOT EXISTS public;

CREATE TABLE IF NOT EXISTS public.metrics (
  id bigserial PRIMARY KEY,
  industry_id uuid NOT NULL,
  metric_type text NOT NULL,
  metric_time timestamptz NOT NULL,
  value double precision,
  tags jsonb DEFAULT '{}'::jsonb,
  details jsonb DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
'''


def gen_insert(schema, industry_uuid, preserve_ids=False):
    # conservative generator: assumes source table has at least metric_type, metric_time
    # and optional value, tags, created_at, updated_at. All other fields get bundled into details.
    common = [c for c in COMMON_COLS]
    detail_expr = "to_jsonb(m) - ARRAY['%s']" % "','".join([c for c in common if c != 'id'])
    if preserve_ids:
        cols = 'id, industry_id, metric_type, metric_time, value, tags, details, created_at, updated_at'
        insert_vals = "m.id, '%s'::uuid, m.metric_type, m.metric_time, m.value, m.tags::jsonb, %s, m.created_at, m.updated_at" % (industry_uuid, detail_expr)
    else:
        cols = 'industry_id, metric_type, metric_time, value, tags, details, created_at, updated_at'
        insert_vals = "'%s'::uuid, m.metric_type, m.metric_time, m.value, m.tags::jsonb, %s, m.created_at, m.updated_at" % (industry_uuid, detail_expr)

    sql = dedent(f"""
    -- Migrate schema: {schema} -> industry_id {industry_uuid}
    INSERT INTO public.metrics ({cols})
    SELECT {insert_vals}
    FROM {schema}.metrics m;
    """
    )
    return sql


def main():
    p = argparse.ArgumentParser()
    p.add_argument('--mapping', required=True)
    p.add_argument('--create-table', action='store_true')
    p.add_argument('--apply', action='store_true')
    p.add_argument('--preserve-ids', action='store_true')
    args = p.parse_args()

    try:
        with open(args.mapping) as f:
            mapping = json.load(f)
    except Exception as e:
        print('Error reading mapping file:', e, file=sys.stderr)
        sys.exit(2)

    if args.create_table:
        print(DDL)

    for schema, uuid in mapping.items():
        print(gen_insert(schema, uuid, preserve_ids=args.preserve_ids))

    if args.apply:
        try:
            import psycopg2
        except Exception:
            print('\n-- ERROR: psycopg2 is required to use --apply. Install it and ensure DATABASE_URL is set.', file=sys.stderr)
            sys.exit(3)

        dburl = os.getenv('DATABASE_URL')
        if not dburl:
            print('\n-- ERROR: set DATABASE_URL environment variable to apply changes', file=sys.stderr)
            sys.exit(4)

        print('\n-- Applying migrations to database...')
        conn = psycopg2.connect(dburl)
        try:
            with conn:
                with conn.cursor() as cur:
                    cur.execute(DDL)
                    for schema, uuid in mapping.items():
                        sql = gen_insert(schema, uuid, preserve_ids=args.preserve_ids)
                        cur.execute(sql)
            print('-- Done')
        finally:
            conn.close()


if __name__ == '__main__':
    main()
