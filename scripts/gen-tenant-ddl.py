#!/usr/bin/env python3
"""Generate the tenant `orm` and `mdm` DDL from the crims database (schema only, no data).

    scripts/gen-tenant-ddl.py --dsn "host=... dbname=crims user=postgres sslmode=verify-full ..."  [--out DIR]
    scripts/gen-tenant-ddl.py --from-dump crims_orm_mdm.sql                                        [--out DIR]

It runs `pg_dump -s -n orm -n mdm --no-owner --no-privileges --no-comments` (or reads that file) and
applies the transforms below, so the result is reproducible and every departure from crims is listed
here, not hidden in an edit:

  1. Row-level security is removed (ENABLE/FORCE ROW LEVEL SECURITY, CREATE POLICY). A tenant has its
     own database, ADR-042; the policies key on a session setting the tenant router never sets, so
     keeping FORCE would make every table read as empty. The lock in a tenant database is the
     database itself, the per-tenant role, and (separately) a CHECK on tenant_id.
  2. Foreign keys that leave the two schemas are removed and reported (today: orm.fund_order_settlement
     -> cash_flow.settlement). cash_flow is not part of a tenant database.
  3. The psql meta-commands pg_dump 18 emits (\\restrict, \\unrestrict) are removed.
  4. CREATE SCHEMA becomes IF NOT EXISTS, and the one extension the schemas use is created first.
  5. Statements are split into two files by the schema in their pg_dump header. Foreign keys that point
     into mdm go in the mdm file, because mdm has to come after orm and refers back to it. The orm
     file therefore never refers to mdm, and the mdm file may refer to orm.

Output: orm.up.sql then mdm.up.sql (apply in that order), and REPORT.txt with the counts and every
statement that was removed.
"""
import argparse, os, re, subprocess, sys

HDR = re.compile(r"^-- Name: (?P<name>.*?); Type: (?P<type>[^;]+); Schema: (?P<schema>[^;]*); Owner: -", re.M)
EXT_PREAMBLE = 'CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public;\n'

def read_dump(a):
    if a.from_dump:
        return open(a.from_dump).read()
    return subprocess.run(
        ["pg_dump", "-d", a.dsn, "-s", "-n", "orm", "-n", "mdm", "--no-owner", "--no-privileges", "--no-tablespaces", "--no-comments"],
        check=True, capture_output=True, text=True).stdout

def split_blocks(text):
    """Yield (schema, type, name, body) for each object block, in dump order."""
    pos = [m for m in HDR.finditer(text)]
    for i, m in enumerate(pos):
        start = m.start()
        end = pos[i + 1].start() if i + 1 < len(pos) else len(text)
        yield m.group("schema"), m.group("type"), m.group("name"), text[start:end]

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--dsn"); ap.add_argument("--from-dump"); ap.add_argument("--out", default="backend/db/tenant_migrations/_candidates/crims")
    a = ap.parse_args()
    if not (a.dsn or a.from_dump):
        sys.exit("give --dsn or --from-dump")
    text = read_dump(a)
    text = re.sub(r"^\\(un)?restrict .*\n", "", text, flags=re.M)
    removed, files = [], {"orm": [], "mdm": []}
    stats = {"tables": {"orm": 0, "mdm": 0}, "policies": 0, "rls": 0}
    for schema, typ, name, body in split_blocks(text):
        if typ == "POLICY":
            stats["policies"] += 1; continue
        # `-- Name: X; Type: ROW SECURITY` does not exist; the ALTER ... ROW LEVEL SECURITY statements sit
        # inside TABLE blocks' neighbours, so strip them by statement below.
        if schema not in ("orm", "mdm"):
            if typ == "SCHEMA":
                pass
            else:
                removed.append(f"-- (outside orm/mdm) {typ} {name}"); continue
        target = schema if schema in files else "orm"
        if typ == "FK CONSTRAINT":
            refs = set(re.findall(r"REFERENCES ([a-z_]+)\.", body))
            outside = refs - {"orm", "mdm"}
            if outside:
                removed.append(f"-- removed FK {schema}.{name}: references {sorted(outside)}\n" + "\n".join("--   " + l for l in body.strip().splitlines() if not l.startswith("--")))
                continue
            if "mdm" in refs:
                target = "mdm"   # mdm is applied second, so a reference into it can only be added there
        if typ == "SCHEMA":
            body = re.sub(r"CREATE SCHEMA (\w+);", r"CREATE SCHEMA IF NOT EXISTS \1;", body)
        # statement-level RLS removal
        def drop_rls(m):
            stats["rls"] += 1; return ""
        body = re.sub(r"^ALTER TABLE [^\n]* (ENABLE|FORCE) ROW LEVEL SECURITY;\n", drop_rls, body, flags=re.M)
        if typ == "TABLE":
            stats["tables"][schema] += len(re.findall(r"^CREATE TABLE ", body, re.M))
        files[target].append(body)
    # the CREATE SCHEMA mdm block lands in the orm file by dump order; move it to mdm
    orm_blocks, mdm_blocks = [], list(files["mdm"])
    for b in files["orm"]:
        if re.search(r"CREATE SCHEMA IF NOT EXISTS mdm;", b):
            mdm_blocks.insert(0, b)
        else:
            orm_blocks.append(b)
    os.makedirs(a.out, exist_ok=True)
    head = ("-- Generated by scripts/gen-tenant-ddl.py from the crims database. Do not edit by hand: change the\n"
            "-- generator or regenerate. Schema only; no data. Row-level security is removed (see the generator).\n\n")
    for name, blocks in (("orm", orm_blocks), ("mdm", mdm_blocks)):
        with open(os.path.join(a.out, f"{name}.up.sql"), "w") as f:
            f.write(head + (EXT_PREAMBLE + "\n" if name == "orm" else "") + "\n".join(b.rstrip() + "\n" for b in blocks))
    with open(os.path.join(a.out, "REPORT.txt"), "w") as f:
        f.write(f"tables: orm={stats['tables']['orm']} mdm={stats['tables']['mdm']}\n")
        f.write(f"policies removed: {stats['policies']}\nrow-level-security statements removed: {stats['rls']}\n\nremoved objects:\n")
        f.write("\n".join(removed) + "\n")
    print(open(os.path.join(a.out, "REPORT.txt")).read())

if __name__ == "__main__":
    main()
