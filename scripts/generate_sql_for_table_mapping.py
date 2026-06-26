#!/usr/bin/env python3
"""
Generate INSERT INTO public.metrics statements from a mapping of schema.table -> industry_uuid.

Usage:
  python3 scripts/generate_sql_for_table_mapping.py --mapping migrations/table_mapping_suggested.json

This prints SQL to stdout (dry-run). Review before applying.
"""
import argparse
import json
from textwrap import dedent


def gen_sql(source, industry_uuid):
    schema, table = source.split('.', 1)
    # conservative exclusion list — these columns will be removed from details
    exclude = ['id','created_at','updated_at','created_on','updated_on']
    # build the to_jsonb exclusion array text
    exc_list = ",".join([f"'{c}'" for c in exclude])
    sql = dedent("""
    -- Migrate from {schema}.{table} -> industry {industry_uuid}
    INSERT INTO public.metrics (industry_id, metric_type, metric_time, value, tags, details, created_at, updated_at)
    SELECT
      '{industry_uuid}'::uuid AS industry_id,
      -- metric_type: try common candidates, fallback to table name
      COALESCE(m.metric_name::text, m.metric_type::text, m.name::text, '{table}') AS metric_type,
      -- timestamp: try common candidates
      COALESCE(m.metric_time, m.collected_at, m.timestamp, m.created_at, now()) AS metric_time,
      -- value: try common numeric candidates
      COALESCE(m.metric_value::double precision, m.value::double precision, m.total_value::double precision, m.count::double precision, NULL) AS value,
      -- tags: keep any jsonb-like column called tags/labels/labels
      COALESCE(m.tags::jsonb, m.labels::jsonb, '{{}}'::jsonb) AS tags,
      -- details: everything else
      to_jsonb(m) - ARRAY[{exc_array}] AS details,
      COALESCE(m.created_at, now()) AS created_at,
      COALESCE(m.updated_at, now()) AS updated_at
    FROM {schema}.{table} m;
    """)
    sql = sql.format(schema=schema, table=table, industry_uuid=industry_uuid, exc_array=exc_list)
    return sql


def main():
    p = argparse.ArgumentParser()
    p.add_argument('--mapping', required=True)
    args = p.parse_args()

    with open(args.mapping) as f:
        mapping = json.load(f)

    for source, uid in mapping.items():
        print(gen_sql(source, uid))


if __name__ == '__main__':
    main()
