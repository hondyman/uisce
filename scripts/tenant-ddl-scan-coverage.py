#!/usr/bin/env python3
"""Compare what alpha's catalog scan holds for a datasource with the live source database. Read-only.

    scripts/tenant-ddl-scan-coverage.py --alpha "<dsn>" --source "<dsn>" --datasource <uuid> --schemas orm,mdm,cash_flow

The rule it supports: alpha metadata wins, so a tenant structure is built from what alpha holds, after the gold
copy's scan has been synced. That is only sound if (1) the scan is current and (2) it records enough. This reports both:

  staleness   tables/columns in the source that the scan does not have, and the reverse (a rescan fixes these)
  fidelity    field mismatches on columns both have (data type, nullability, length, precision, scale, default)
  not in scan classes of object the scan does not record at all, counted in the source (a rescan does NOT fix these)

Exit status 0 only when there is no staleness and no fidelity mismatch; 2 otherwise. Unrecorded classes are
reported, never counted as failure, because the scanner has to be extended to record them.
"""
import argparse, collections, csv, subprocess, sys

def psql(dsn, sql):
    out = subprocess.run(["psql", dsn, "-AtqF", "\t", "-c", sql], check=True, capture_output=True, text=True).stdout
    return list(csv.reader(out.splitlines(), delimiter="\t"))

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--alpha", required=True); ap.add_argument("--source", required=True)
    ap.add_argument("--datasource", required=True); ap.add_argument("--schemas", default="orm,mdm,cash_flow")
    a = ap.parse_args()
    sch = [x.strip() for x in a.schemas.split(",")]
    inl = ",".join("'" + x + "'" for x in sch)
    scan = {tuple(r[:3]): r[3:] for r in psql(a.alpha, f"""
        select split_part(qualified_path,'/',2), split_part(qualified_path,'/',3), split_part(qualified_path,'/',4),
               coalesce(properties->>'data_type',''), case when coalesce(properties->>'is_nullable','')='true' then 'true' else 'false' end,
               regexp_replace(coalesce(properties->>'default_value',''),'^public\\.',''), coalesce(properties->>'max_length',''),
               coalesce(properties->>'precision',''), coalesce(properties->>'scale','')
          from catalog_node where tenant_datasource_id='{a.datasource}' and (properties->>'is_physical_column')='true'
           and split_part(qualified_path,'/',2) in ({inl})""")}
    real = {tuple(r[:3]): r[3:] for r in psql(a.source, f"""
        select table_schema, table_name, column_name, data_type, case when is_nullable='YES' then 'true' else 'false' end,
               regexp_replace(coalesce(column_default,''),'^public\\.',''), coalesce(character_maximum_length::text,''),
               coalesce(numeric_precision::text,''), coalesce(numeric_scale::text,'')
          from information_schema.columns where table_schema in ({inl})""")}
    rt, st = {k[:2] for k in real}, {k[:2] for k in scan}
    missing_cols, extra_cols = set(real) - set(scan), set(scan) - set(real)
    names = ["data_type", "nullable", "default", "max_length", "precision", "scale"]
    mism = collections.Counter(); ex = {}
    for k in set(scan) & set(real):
        for i, n in enumerate(names):
            if scan[k][i] != real[k][i]:
                mism[n] += 1; ex.setdefault(n, (k, scan[k][i], real[k][i]))
    print(f"source: {len(rt)} tables, {len(real)} columns   scan: {len(st)} tables, {len(scan)} columns")
    print(f"STALE   tables in source not in scan: {len(rt - st)}  {sorted(rt - st)[:6]}")
    print(f"STALE   tables in scan not in source: {len(st - rt)}  {sorted(st - rt)[:6]}")
    byt = collections.Counter(k[:2] for k in missing_cols)
    print(f"STALE   columns in source not in scan: {len(missing_cols)} across {len(byt)} tables {[f'{x}.{y}({c})' for (x, y), c in byt.most_common(4)]}")
    print(f"STALE   columns in scan not in source: {len(extra_cols)}  {sorted('/'.join(k) for k in extra_cols)[:4]}")
    print(f"FIDELITY mismatches on the {len(set(scan) & set(real))} columns both have: {dict(mism) or 'none'}")
    for n, (k, x, y) in ex.items():
        print(f"           e.g. {n} {'/'.join(k)}: scan={x!r} source={y!r}")
    cnt = lambda q: psql(a.source, q)[0][0]
    nsoid = f"(select oid from pg_namespace where nspname in ({inl}))"
    print("NOT IN SCAN (counted in the source; a rescan does not add these):")
    print("   check constraints       " + cnt(f"select count(*) from pg_constraint where contype = 'c' and connamespace in {nsoid}"))
    print("   non-primary-key indexes " + cnt(f"select count(*) from pg_indexes where schemaname in ({inl}) and indexname !~ '_pkey$'"))
    print("   partitioned tables      " + cnt(f"select count(*) from pg_class where relkind = 'p' and relnamespace in {nsoid}"))
    print("   functions / triggers    " + cnt(f"select count(*) from pg_proc where pronamespace in {nsoid}") + " / "
          + cnt(f"select count(*) from pg_trigger t join pg_class c on c.oid = t.tgrelid where not t.tgisinternal and c.relnamespace in {nsoid}"))
    sys.exit(0 if not (rt ^ st or missing_cols or extra_cols or mism) else 2)

if __name__ == "__main__":
    main()
