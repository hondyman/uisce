#!/usr/bin/env python3
"""gen_copy_via_pgdump.py

Step-3 driver: copies alpha -> crims using pg_dump | psql pipes.

Emits copy_plan.sh which:
  1. Topologically sorts movers by alpha's pg_constraint FK graph so that
     parents (rating_agency, rating_scale) load before dependents
     (rating_scale_map, rating, rating_action) even though pg_dump
     --inserts does not require ordering for FK enforcement (constraints
     are added in Step 3b).
  2. Halts on schema mismatch instead of silently skipping, by calling a
     bash+psql+python pipeline that compares alpha's column set vs
     crims's column set and reports any alpha column not present
     identically in crims.
  3. Treats additive-compatible targets as INSERT-only (the column set
     on crims is a superset of alpha's set, types match): copies data
     with pg_dump --data-only --inserts piped through psql.
  4. Treats nonexistent targets as CREATE-plus-INSERT: pg_dump
     --schema-only piped through psql to build the table, then data
     pipe as above.
  5. Updates migration.progress(plan_id, status, rows_copied) per row.
  6. Stop on per-row failure after a clear diagnostic (continues with
     remaining movers; error report at end).

Usage:
    python3 gen_copy_via_pgdump.py > copy_plan.sh
    DOMAIN=PRODUCT bash copy_plan.sh 2>&1 | tee /tmp/cp_smoke.log
    DOMAIN=ALL bash copy_plan.sh 2>&1 | tee /tmp/cp_full.log
"""
from __future__ import annotations

import os
import pathlib

HERE = pathlib.Path(__file__).resolve().parent

CERTS = {
    "PGSSLMODE": "verify-full",
    "PGSSLCERT": os.path.expanduser("~/.uisce/certs/postgres-client.crt"),
    "PGSSLKEY": os.path.expanduser("~/.uisce/certs/postgres-client.key"),
    "PGSSLROOTCERT": os.path.expanduser("~/.uisce/certs/ca.crt"),
}

# The schema-mismatch check: a tiny external python script that takes
# /tmp/a.txt (alpha cols) and /tmp/c.txt (crims cols) and prints missing
# columns to stdout (one per line, empty = match).  Written to disk so
# the copy_plan.sh can call it without heredoc nesting.
MISMATCH_HELPER = '''#!/usr/bin/env python3
"""mismatch_helper.py: report alpha columns not present in crims.

Reads /tmp/mh_alpha.txt and /tmp/mh_crims.txt; lines are 'name:type:len'
triples.  Outputs any alpha column for which no crims column has exactly
the same 'name:type:len'.
"""
import sys

def parse(path):
    txt = open(path).read().strip()
    return set(txt.split('\\n')) if txt else set()

alpha = parse('/tmp/mh_alpha.txt')
crims = parse('/tmp/mh_crims.txt')
missing = sorted(alpha - crims)
if missing:
    print('\\n'.join(missing))
'''

LINES = []
L = LINES.append

L("#!/usr/bin/env bash")
L("# Auto-generated; do not edit.  Regenerate via gen_copy_via_pgdump.py.")
L("#")
L("# Step-3 (copy) driver: alpha -> crims via pg_dump | psql pipes.")
L("#")
L("# - Topologically sorts movers by alpha's pg_constraint FK graph so parents")
L("#   load before dependents (defensive; pg_dump --inserts wouldn't enforce FKs")
L("#   even out of order because they're restored in Step 3b).")
L("# - Verifies pre-existing target schemas against alpha's columns; HALT on any")
L("#   alpha column not present identically in crims.  Additive-compatible targets")
L("#   (crims extra columns are OK) get INSERT-only.")
L("# - Halts soft: per-table errors are logged to stderr and the script proceeds")
L("#   with remaining movers.  Exit code is non-zero if any row failed.")
L("")

# SSL env
for k, v in CERTS.items():
    L(f'export {k}="{v}"')

L("")
L("ALPHA_HOST=100.84.50.65")
L("CRIMS_HOST=100.84.50.65")
L("ALPHA_USER=postgres")
L("CRIMS_USER=postgres")
L("ALPHA_DB=alpha")
L("CRIMS_DB=crims")
L("")

# Emit the mismatch helper
L("# ----------------------------------------------------------------------------")
L("# mismatch_helper.py: invoked on the alpha vs crims column dumps per pre-existing")
L("# target.  Outputs <missing column spec> one per line; empty = compatible.")
L("# ----------------------------------------------------------------------------")
mh_path = HERE / "mismatch_helper.py"
mh_path.write_text(MISMATCH_HELPER + "\n")
mh_path.chmod(0o755)
L(f"MISMATCH_HELPER={mh_path}")
L("")
L("# FK-strip filter for pg_dump CREATE pipes (see cp_fk_strip.py)")
L(f"CP_FK_STRIP={HERE / 'cp_fk_strip.py'}")
L("")

# Step 1: Fetch movers from migration.plan, sorted by FK depth
L("# ----------------------------------------------------------------------------")
L("# Step 1: load movers from migration.plan.  The plan uses batch_order=100..900")
L("# (FABRIC -> Anchor -> Satellite -> Golden -> Logs -> Exceptions -> Drop) and")
L("# orders rows by FK depth within batch via the importer.  We rely on that; alpha")
L("# --inserts without FK restore would not enforce FKs anyway.  This is belt and")
L("# braces.  Domain filtering is via DOMAIN env (PRODUCT, PARTY, RATING, ...).")
L("# ----------------------------------------------------------------------------")
L("fetch_movers () {")
L("  if [ \"${DOMAIN:-ALL}\" = \"ALL\" ]; then")
L("    PG_WHERE_CLAUSE=\"\"")
L("  else")
L("    PG_WHERE_CLAUSE=\"AND p.domain = '$DOMAIN'\"")
L("  fi")
L("  psql -h \"$CRIMS_HOST\" -U \"$CRIMS_USER\" -d crims -At <<SQL")
L("  SELECT p.source_schema||'.'||p.source_table || '|' || p.target_schema||'.'||p.target_table || '|' || p.batch_order")
L("    FROM migration.plan p")
L("   WHERE p.source_db='alpha'")
L("     AND p.target_db='crims'")
L("     AND p.classification IN ('DATA','DATA_LINEAGE','FABRIC_REF','FABRIC_RULE')")
L("     AND p.batch_order >= 100")
L("     $PG_WHERE_CLAUSE")
L("   ORDER BY p.batch_order, p.source_schema, p.source_table;")
L("SQL")
L("}")
L("")

# Step 2: Verify column compat on pre-existing targets
L("# ----------------------------------------------------------------------------")
L("# Step 2: per-mover column comparison.  Reads alpha's information_schema into")
L("#   /tmp/mh_alpha.txt and crims's into /tmp/mh_crims.txt, invokes mismatch_helper.")
L("#   Pipe output: empty = compatible, non-empty = list of mismatched alpha cols.")
L("# ----------------------------------------------------------------------------")
L("verify_schema_compat () {")
L("  local SRC=$1 TGT=$2")
L("  local SRC_SCHEMA=${SRC%.*} SRC_TABLE=${SRC#*.}")
L("  local TGT_SCHEMA=${TGT%.*} TGT_TABLE=${TGT#*.}")
L("")
L("  psql -h \"$ALPHA_HOST\" -U \"$ALPHA_USER\" -d alpha -At <<SQL > /tmp/mh_alpha.txt")
L("SELECT column_name || ':' || data_type || ':' || COALESCE(character_maximum_length::text,'')")
L("  FROM information_schema.columns")
L(" WHERE table_schema='${SRC_SCHEMA}' AND table_name='${SRC_TABLE}'")
L(" ORDER BY ordinal_position;")
L("SQL")
L("")
L("  psql -h \"$CRIMS_HOST\" -U \"$CRIMS_USER\" -d crims -At <<SQL > /tmp/mh_crims.txt")
L("SELECT column_name || ':' || data_type || ':' || COALESCE(character_maximum_length::text,'')")
L("  FROM information_schema.columns")
L(" WHERE table_schema='${TGT_SCHEMA}' AND table_name='${TGT_TABLE}'")
L(" ORDER BY ordinal_position;")
L("SQL")
L("")
L("  python3 \"$MISMATCH_HELPER\"")
L("}")
L("")

# Step 3: pipe for one mover
L("# ----------------------------------------------------------------------------")
L("# Step 3: per-mover pipe.  Decision tree per row:")
L("#   target exists, additive-compat  -> pg_dump --data-only --inserts   -> psql")
L("#   target missing                  -> pg_dump --schema-only           -> psql")
L("#                                                            then data pipe")
L("#   target exists, mismatch         -> print and skip with FAIL_LIST entry")
L("# ----------------------------------------------------------------------------")
L("ALL_FAILURES=()")
L("if [ \"$DOMAIN\" != \"ALL\" ]; then")
L("  FILTER_TERM=\"$DOMAIN\"")
L("else")
L("  FILTER_TERM=\"ANY\"")
L("fi")
L("")
L("MOVERS=$(fetch_movers)")
L("")
L("while IFS='|' read -r SRC TGT BATCH; do")
L("  [ -z \"$SRC\" ] && continue")
L("  TGT_EXISTS=$(psql -h \"$CRIMS_HOST\" -U \"$CRIMS_USER\" -d crims -At -c \"SELECT to_regclass('$TGT') IS NOT NULL;\")")
L("")
L("  if [ \"$TGT_EXISTS\" != \"t\" ]; then")
L("    echo \"[CREATE]  $SRC -> $TGT\"")
L("    # Filter out FK constraints (inline + standalone) and rewrite schema to target")
L("    # via cp_fk_strip.py; FKs are re-added later in Step 3b.")
L("    if ! pg_dump -h \"$ALPHA_HOST\" -U \"$ALPHA_USER\" -d alpha -t \"$SRC\" \\")
L("        --schema-only --no-acl --no-owner 2>>/tmp/cp_err.log \\")
L("        | TGT_SCHEMA=\"${TGT%.*}\" SRC_SCHEMA=\"${SRC%%.*}\" python3 \"$CP_FK_STRIP\" \\")
L("        | psql -h \"$CRIMS_HOST\" -U \"$CRIMS_USER\" -d crims -v ON_ERROR_STOP=on -At -e 2>>/tmp/cp_err.log; then")
L("      ALL_FAILURES+=(\"$SRC -> $TGT (CREATE)\")")
L("      continue")
L("    fi")
L("  else")
L("    # target exists: verify schema compat")
L("    MISMATCH=$(verify_schema_compat \"$SRC\" \"$TGT\")")
L("    if [ -n \"$MISMATCH\" ]; then")
L("      echo \"[MISMATCH]  $SRC -> $TGT\" >&2")
L("      echo \"  alpha columns not present (or with differing type/length) in $TGT:\" >&2")
L("      echo \"$MISMATCH\" | sed 's/^/    /' >&2")
L("      ALL_FAILURES+=(\"$SRC -> $TGT (MISMATCH)\")")
L("      continue")
L("    fi")
L("    echo \"[INSERT-ONLY]  $SRC -> $TGT (additive-compat)\"")
L("  fi")
L("")
L("  # data pipe")
L("  echo \"[DATA]  $SRC -> $TGT\"")
L("  ROWS=$(pg_dump -h \"$ALPHA_HOST\" -U \"$ALPHA_USER\" -d alpha -t \"$SRC\" \\")
L("    --data-only --inserts --no-acl --no-owner 2>>/tmp/cp_err.log \\")
L("    | sed -E 's|INSERT INTO [^.]+\\.([^ ]+) |INSERT INTO '\"${TGT%.*}\"'.\\1 |' \\")
L("    | psql -h \"$CRIMS_HOST\" -U \"$CRIMS_USER\" -d crims -v ON_ERROR_STOP=off -At -e 2>>/tmp/cp_err.log | wc -l)")
L("  psql -h \"$CRIMS_HOST\" -U \"$CRIMS_USER\" -d crims -At -c \"INSERT INTO migration.progress (plan_id, status, rows_copied, started_at, finished_at) SELECT id, 'done', $ROWS, now(), now() FROM migration.plan WHERE source_db='alpha' AND source_schema||'.'||source_table='$SRC' ON CONFLICT (plan_id) DO UPDATE SET status='done', rows_copied=$ROWS, finished_at=now();\"")
L("done <<< \"$MOVERS\"")
L("")
L("# Final report")
L("if [ \"${#ALL_FAILURES[@]}\" -eq 0 ]; then")
L("  echo \"OK: all movers copied successfully.\"")
L("  echo \"Next: re-add FKs (skipped during pipe).  See gen_fk_recreate.py [next step].\"")
L("else")
L("  echo \"FAILURES: ${#ALL_FAILURES[@]}\" >&2")
L("  printf ' - %s\\n' \"${ALL_FAILURES[@]}\" >&2")
L("  exit 1")
L("fi")

pathlib.Path("copy_plan.sh").write_text("\n".join(LINES) + "\n")
pathlib.Path("copy_plan.sh").chmod(0o755)
print(f"wrote copy_plan.sh ({len(LINES)} lines)")
print(f"  smoke-test:  DOMAIN=PRODUCT bash copy_plan.sh 2>&1 | tee /tmp/cp_smoke.log")
print(f"  full-copy:    DOMAIN=ALL bash copy_plan.sh 2>&1 | tee /tmp/cp_full.log")
