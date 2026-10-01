#!/usr/bin/env python3
"""gen_fk_recreate.py: compute per-FK post-migration actions (Step 3b).

Inputs:
  - step0_fk_inventory.tsv  (fk_name, child_schema.table, parent_schema.table, fk_clause)
  - crims.migration.plan    (where each alpha table ends up)

Outputs:
  - migration.fk_actions table (populated)
  - 005_fk_recreate.sql   (idempotent DDL for RECREATE actions, run against crims)

Outcomes:
  RECREATE  child -> crims, parent -> crims   (same home; rewrite schemas if needed)
  DROP      child -> crims, parent stays alpha (cross-DB) or parent dropped
  KEEP      child stays alpha (FK untouched in alpha; no-op for crims)
"""
from __future__ import annotations

import os
import sys
from pathlib import Path

import psycopg2

HERE = Path(__file__).resolve().parent
INVENTORY = HERE / "step0_fk_inventory.tsv"
OUT_SQL = HERE / "005_fk_recreate.sql"

PG = dict(
    host=os.environ.get("PGHOST", "100.84.50.65"),
    user=os.environ.get("PGUSER", "postgres"),
)


def load_inventory() -> list[dict]:
    rows = []
    for line in INVENTORY.read_text().splitlines():
        if not line.strip():
            continue
        fk_name, child, parent, clause = line.split("\t")
        rows.append(
            dict(
                fk_name=fk_name,
                child_schema=child.split(".")[0],
                child_table=child.split(".", 1)[1],
                parent_schema=parent.split(".")[0],
                parent_table=parent.split(".", 1)[1],
                clause=clause.strip(),
            )
        )
    return rows


def load_plan_targets(conn) -> dict[tuple[str, str], tuple[str, str, str]]:
    """(src_schema, src_table) -> (target_db, target_schema, target_table) or
    (target_db='-', ...) if dropped / no target."""
    out: dict[tuple[str, str], tuple[str, str, str]] = {}
    with conn.cursor() as cur:
        cur.execute(
            """
            SELECT source_schema, source_table,
                   COALESCE(target_db,'-'), COALESCE(target_schema,'-'), COALESCE(target_table,'-')
            FROM migration.plan
            WHERE source_db='alpha'
            """
        )
        for ss, st, td, ts, tt in cur.fetchall():
            out[(ss, st)] = (td, ts, tt)
    return out


def home_of(
    schema: str,
    table: str,
    plan: dict[tuple[str, str], tuple[str, str, str]],
    alpha_tables: set[tuple[str, str]],
    crims_tables: set[tuple[str, str]],
) -> str | None:
    """Return 'crims', 'alpha', or None (gone / unknown)."""
    key = (schema, table)
    if key in plan:
        td, ts, tt = plan[key]
        if td == "crims" and tt != "-":
            return "crims"
        if td == "alpha":
            return "alpha"
        # target_db='-' or null target -> dropped from crims; may still exist in alpha
        # but the plan intent is DROP for the alpha copy later. For FK purposes the
        # table still exists in alpha today -> treat as alpha (KEEP) unless also
        # missing from alpha.
        return "alpha" if key in alpha_tables else None
    # Not in plan: table already lives where it lives.
    if key in crims_tables:
        return "crims"
    if key in alpha_tables:
        return "alpha"
    return None


def qualify(schema: str, table: str) -> str:
    return f'"{schema}"."{table}"'


def rewrite_clause_to_home(clause: str, parent_schema: str, home: str) -> str:
    """Clause looks like: FOREIGN KEY (col) REFERENCES schema.table(col) [...]

    When home == 'crims' both DBs use the same schema names (mdm/public), so no
    textual rewrite is required — the clause is schema-qualified relative to the
    connected database.  We only normalize double-quoting for safety.
    """
    return clause


def ensure_fk_actions(cur) -> None:
    cur.execute(
        """
        CREATE TABLE IF NOT EXISTS migration.fk_actions (
            fk_name        text   NOT NULL,
            child_schema   text   NOT NULL,
            child_table    text   NOT NULL,
            parent_schema  text   NOT NULL,
            parent_table   text   NOT NULL,
            action         text   NOT NULL CHECK (action IN ('RECREATE','DROP','KEEP')),
            ddl            text,
            notes          text,
            PRIMARY KEY (fk_name, child_schema, child_table)
        )
        """
    )


def main() -> int:
    inv = load_inventory()
    alpha_conn = psycopg2.connect(dbname="alpha", **PG)
    crims_conn = psycopg2.connect(dbname="crims", **PG)

    with alpha_conn.cursor() as cur:
        cur.execute(
            "SELECT table_schema, table_name FROM information_schema.tables "
            "WHERE table_schema NOT IN ('pg_catalog','information_schema')"
        )
        alpha_tables = set(cur.fetchall())
    with crims_conn.cursor() as cur:
        cur.execute(
            "SELECT table_schema, table_name FROM information_schema.tables "
            "WHERE table_schema NOT IN ('pg_catalog','information_schema')"
        )
        crims_tables = set(cur.fetchall())
        plan = load_plan_targets(crims_conn)

    actions: list[dict] = []
    for fk in inv:
        ch = home_of(fk["child_schema"], fk["child_table"], plan, alpha_tables, crims_tables)
        pa = home_of(fk["parent_schema"], fk["parent_table"], plan, alpha_tables, crims_tables)

        if ch == "alpha":
            action, ddl, notes = "KEEP", None, "child stays in alpha"
        elif ch == "crims" and pa == "crims":
            action = "RECREATE"
            # Rewrite REFERENCES parent if parent moved schemas (edm.* -> mdm.*).
            parent_key = (fk["parent_schema"], fk["parent_table"])
            if parent_key in plan:
                _td, p_ts, p_tt = plan[parent_key]
                if _td == "crims" and p_tt != "-":
                    clause = fk["clause"]
                    clause = clause.replace(
                        f"REFERENCES {fk['parent_schema']}.{fk['parent_table']}",
                        f"REFERENCES {p_ts}.{p_tt}",
                    )
                else:
                    clause = fk["clause"]
            else:
                clause = fk["clause"]
            ddl = (
                f"ALTER TABLE {qualify(fk['child_schema'], fk['child_table'])} "
                f"ADD CONSTRAINT \"{fk['fk_name']}\" {clause};"
            )
            notes = "same-home (crims)"
        elif ch == "crims" and pa == "alpha":
            action, ddl = "DROP", None
            notes = "cross-DB (parent stays alpha) — app/RLS enforced"
        elif ch == "crims" and pa is None:
            action, ddl = "DROP", None
            notes = "parent gone"
        else:
            action, ddl = "DROP", None
            notes = f"child home={ch}, parent home={pa} — unresolved, drop in alpha later"

        actions.append(dict(fk, action=action, ddl=ddl, notes=notes))

    counts: dict[str, int] = {}
    for a in actions:
        counts[a["action"]] = counts.get(a["action"], 0) + 1

    with crims_conn.cursor() as cur:
        ensure_fk_actions(cur)
        cur.execute("TRUNCATE migration.fk_actions")
        for a in actions:
            cur.execute(
                """
                INSERT INTO migration.fk_actions
                    (fk_name, child_schema, child_table, parent_schema, parent_table,
                     action, ddl, notes)
                VALUES (%s,%s,%s,%s,%s,%s,%s,%s)
                """,
                (
                    a["fk_name"],
                    a["child_schema"],
                    a["child_table"],
                    a["parent_schema"],
                    a["parent_table"],
                    a["action"],
                    a["ddl"],
                    a["notes"],
                ),
            )
    crims_conn.commit()

    # Idempotent SQL file for RECREATE actions (skip constraints that already exist).
    lines = [
        "-- 005_fk_recreate.sql — Step 3b: re-add same-home FKs in crims.",
        "-- Auto-generated by gen_fk_recreate.py. Idempotent: skips existing constraints.",
        "-- DROP actions are intentionally absent (cross-DB / parent-gone FKs are",
        "-- enforced by app logic + RLS, per FABRIC_FK_DROPPED blocker resolution).",
        "",
        "DO $$",
        "DECLARE r record; n int := 0;",
        "BEGIN",
        "  FOR r IN SELECT child_schema, child_table, fk_name, ddl FROM migration.fk_actions WHERE action='RECREATE' ORDER BY child_schema, child_table, fk_name",
        "  LOOP",
        "    IF NOT EXISTS (",
        "      SELECT 1 FROM pg_constraint c",
        "        JOIN pg_class t ON t.oid = c.conrelid",
        "        JOIN pg_namespace s ON s.oid = t.relnamespace",
        "       WHERE c.conname = r.fk_name",
        "         AND s.nspname = r.child_schema",
        "         AND t.relname = r.child_table",
        "         AND c.contype = 'f'",
        "    ) THEN",
        "      EXECUTE r.ddl; n := n + 1;",
        "    END IF;",
        "  END LOOP;",
        f"  RAISE NOTICE 'fk_recreate: % constraints added', n;",
        "END $$;",
        "",
    ]
    OUT_SQL.write_text("\n".join(lines))

    print(f"fk_actions: RECREATE={counts.get('RECREATE',0)} "
          f"DROP={counts.get('DROP',0)} KEEP={counts.get('KEEP',0)} "
          f"(total {len(actions)})")
    print(f"wrote {OUT_SQL}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
