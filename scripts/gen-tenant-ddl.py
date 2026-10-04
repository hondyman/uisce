#!/usr/bin/env python3
"""Generate the tenant structure from the crims database (schema only, no data).

The template is the gold-copy tenant's CRIMS datasource: its `config.schema` lists the schemas a tenant's
copy of it contains (today `orm,vend,ref,mdm,cash_flow,wlth`). `--schemas` takes that list; the default is it.

    scripts/gen-tenant-ddl.py --dsn "host=... dbname=crims user=postgres sslmode=verify-full ..."  [--out DIR]
    scripts/gen-tenant-ddl.py --from-dump crims_orm_mdm.sql                                        [--out DIR]

It runs `pg_dump -s -n orm -n mdm --no-owner --no-privileges --no-comments` (or reads that file) and
applies the transforms below, so the result is reproducible and every departure from crims is listed
here, not hidden in an edit:

  1. Row-level security is removed (ENABLE/FORCE ROW LEVEL SECURITY, CREATE POLICY). A tenant has its
     own database, ADR-042; the policies key on a session setting the tenant router never sets, so
     keeping FORCE would make every table read as empty. The lock in a tenant database is the
     database itself, the per-tenant role, and (separately) a CHECK on tenant_id.
  2. Foreign keys that leave the chosen schemas are removed and reported. With the gold copy's six
     schemas there are none; the report says so.
  3. The psql meta-commands pg_dump 18 emits (\\restrict, \\unrestrict) are removed.
  4. CREATE SCHEMA becomes IF NOT EXISTS, and the one extension the schemas use is created first.
  5. Statements are split into one file per schema, in APPLY_ORDER. A foreign key goes in the file of the
     LATER schema it touches (the owner's or the referenced one's), so no file refers to a schema that does
     not exist yet. orm and mdm refer to each other: the orm file never refers to mdm, and the reference
     back (orm.account -> mdm.party) is added at the end of the mdm file.

Output: NN_<schema>.up.sql in APPLY_ORDER (apply in that order), and REPORT.txt with the counts and every
statement that was removed.
"""
import argparse, os, re, subprocess, sys

HDR = re.compile(r"^-- Name: (?P<name>.*?); Type: (?P<type>[^;]+); Schema: (?P<schema>[^;]*); Owner: -", re.M)
# Dependency order: schemas nothing refers to first; orm refers to cash_flow; mdm refers to orm and is referred to by orm.
APPLY_ORDER = ["ref", "vend", "wlth", "cash_flow", "orm", "mdm"]
EXT_PREAMBLE = 'CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public;\n'

def read_dump(a, schemas):
    if a.from_dump:
        return open(a.from_dump).read()
    cmd = ["pg_dump", "-d", a.dsn, "-s", "--no-owner", "--no-privileges", "--no-tablespaces", "--no-comments"]
    for sc in schemas:
        cmd += ["-n", sc]
    return subprocess.run(cmd, check=True, capture_output=True, text=True).stdout

def split_blocks(text):
    """Yield (schema, type, name, body) for each object block, in dump order."""
    pos = [m for m in HDR.finditer(text)]
    for i, m in enumerate(pos):
        start = m.start()
        end = pos[i + 1].start() if i + 1 < len(pos) else len(text)
        yield m.group("schema"), m.group("type"), m.group("name"), text[start:end]

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--dsn"); ap.add_argument("--from-dump")
    ap.add_argument("--schemas", default=",".join(APPLY_ORDER), help="the gold copy datasource's config.schema")
    ap.add_argument("--out", default="backend/db/tenant_migrations/_candidates/crims")
    a = ap.parse_args()
    if not (a.dsn or a.from_dump):
        sys.exit("give --dsn or --from-dump")
    schemas = [x.strip() for x in a.schemas.split(",") if x.strip()]
    unknown = set(schemas) - set(APPLY_ORDER)
    if unknown:
        sys.exit(f"no apply order known for {sorted(unknown)}: add them to APPLY_ORDER with their dependencies")
    order = [x for x in APPLY_ORDER if x in schemas]
    rank = {s_: i for i, s_ in enumerate(order)}
    text = re.sub(r"^\\(un)?restrict .*\n", "", read_dump(a, schemas), flags=re.M)
    removed = []
    files = {s_: [] for s_ in order}
    stats = {"tables": {s_: 0 for s_ in order}, "policies": 0, "rls": 0}
    for schema, typ, name, body in split_blocks(text):
        if typ == "POLICY":
            stats["policies"] += 1; continue
        if typ == "SCHEMA":
            schema = name   # pg_dump labels a schema's own block with Schema: "-"
        if schema not in rank:
            removed.append(f"-- (outside the chosen schemas) {typ} {name}"); continue
        target = schema
        if typ == "FK CONSTRAINT":
            refs = set(re.findall(r"REFERENCES ([a-z_]+)\.", body))
            outside = refs - set(order)
            if outside:
                removed.append(f"-- removed FK {schema}.{name}: references {sorted(outside)}\n" + "\n".join("--   " + l for l in body.strip().splitlines() if not l.startswith("--")))
                continue
            target = max([schema] + sorted(refs), key=lambda x: rank[x])
        if typ == "SCHEMA":
            body = re.sub(r"CREATE SCHEMA (\w+);", r"CREATE SCHEMA IF NOT EXISTS \1;", body)
        def drop_rls(m):
            stats["rls"] += 1; return ""
        body = re.sub(r"^ALTER TABLE [^\n]* (ENABLE|FORCE) ROW LEVEL SECURITY;\n", drop_rls, body, flags=re.M)
        if typ == "TABLE":
            stats["tables"][schema] += len(re.findall(r"^CREATE TABLE ", body, re.M))
        files[target].append(body)
    # every schema's own CREATE SCHEMA must come first in its own file, whatever the dump order was
    for s_ in order:
        own = [b for b in files[s_] if re.search(rf"CREATE SCHEMA IF NOT EXISTS {s_};", b)]
        rest = [b for b in files[s_] if b not in own]
        files[s_] = own + rest
    os.makedirs(a.out, exist_ok=True)
    for f in os.listdir(a.out):
        if f.endswith(".up.sql"):
            os.remove(os.path.join(a.out, f))
    head = ("-- Generated by scripts/gen-tenant-ddl.py from the crims database. Do not edit by hand: change the\n"
            "-- generator or regenerate. Schema only; no data. Row-level security is removed (see the generator).\n\n")
    for i, s_ in enumerate(order):
        with open(os.path.join(a.out, f"{i + 1:02d}_{s_}.up.sql"), "w") as f:
            f.write(head + (EXT_PREAMBLE + "\n" if i == 0 else "") + "\n".join(b.rstrip() + "\n" for b in files[s_]))
    with open(os.path.join(a.out, "REPORT.txt"), "w") as f:
        f.write("apply order: " + " -> ".join(f"{i + 1:02d}_{s_}" for i, s_ in enumerate(order)) + "\n")
        f.write("tables: " + " ".join(f"{s_}={stats['tables'][s_]}" for s_ in order) + "\n")
        f.write(f"policies removed: {stats['policies']}\nrow-level-security statements removed: {stats['rls']}\n\nremoved objects:\n")
        f.write(("\n".join(removed) if removed else "(none)") + "\n")
    print(open(os.path.join(a.out, "REPORT.txt")).read())

if __name__ == "__main__":
    main()
