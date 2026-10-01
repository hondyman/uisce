#!/usr/bin/env python3
"""cp_fk_strip.py: filter pg_dump --schema-only output so CREATE TABLE on crims works.

Strips:
  - inline CREATE TABLE column-level FOREIGN KEY references (the REFERENCES ... trailer)
  - post-CREATE-TABLE `ALTER TABLE ONLY x.t ADD CONSTRAINT ... FOREIGN KEY ...` blocks
  - rewrites schema.table -> tgt_schema.table in CREATE TABLE statements

Reads from stdin, emits to stdout.  Uses `TGT_SCHEMA` env var as the rewrite target.

Invocation:
    pg_dump ... | python3 cp_fk_strip.py | psql ...
"""
import os
import re
import sys

TGT_SCHEMA = os.environ.get("TGT_SCHEMA", "crims")
# Source schema of the table being dumped (e.g. "edm" or "mdm").  When it
# differs from TGT_SCHEMA (edm -> mdm moves), every `SRC_SCHEMA.` qualifier in
# DDL must be rewritten, not just CREATE TABLE (ALTER TABLE / CREATE POLICY /
# CREATE INDEX all carry the source schema).
SRC_SCHEMA = os.environ.get("SRC_SCHEMA", "")

# For each input line, do schema.table -> TGT_SCHEMA.table if appropriate.
def rewrite_inline(text: str) -> str:
    """Rewrite source-schema qualifiers to TGT_SCHEMA.

    Handles CREATE TABLE / INDEX / POLICY and any residual `SRC_SCHEMA.`
    qualifiers on ALTER TABLE etc.  When SRC_SCHEMA == TGT_SCHEMA this is a
    no-op except for normalization.
    """
    if SRC_SCHEMA and SRC_SCHEMA != TGT_SCHEMA:
        # Word-boundary replace of the source schema qualifier only.
        text = re.sub(rf"\b{re.escape(SRC_SCHEMA)}\.", f"{TGT_SCHEMA}.", text)

    # CREATE TABLE s.t <space-or-paren> (covers mdm->mdm normalization too)
    text = re.sub(r"\bCREATE\s+TABLE\s+\S+\.([^\s\(]+)", lambda m: f"CREATE TABLE {TGT_SCHEMA}.{m.group(1)}", text)
    # CREATE [UNIQUE] INDEX [name] ON s.t — IF NOT EXISTS guards against
    # schema-scoped index-name collisions with unrelated crims tables.
    text = re.sub(r"\bCREATE\s+(UNIQUE\s+)?INDEX\s+(?!IF NOT EXISTS)(\S+)\s+ON\s+\S+\.([^\s\(]+)",
                  lambda m: f"CREATE {m.group(1) or ''}INDEX IF NOT EXISTS {m.group(2)} ON {TGT_SCHEMA}.{m.group(3)}",
                  text)
    # CREATE INDEX [name] ON s.t
    text = re.sub(r"\bCREATE\s+INDEX\s+\S+\s+ON\s+\S+\.([^\s\(]+)", lambda m: re.sub(r"\bON\s+\S+\.([^\s\(]+)", f"ON {TGT_SCHEMA}.{m.group(1)}", m.group(0)), text)
    # CREATE POLICY [name] ON s.t
    text = re.sub(r"\bCREATE\s+POLICY\s+\S+\s+ON\s+\S+\.([^\s\(]+)", lambda m: re.sub(r"\bON\s+\S+\.([^\s\(]+)", f"ON {TGT_SCHEMA}.{m.group(1)}", m.group(0)), text)
    return text


def strip_fks(text: str) -> str:
    """Remove CREATE TABLE inline REFERENCES clauses (multi-line aware).

    The pg_dump format inside CREATE TABLE body is:
        <col_name>   <TYPE>   [CONSTRAINT name] REFERENCES tbl(col) [ON DELETE ...],
    Sometimes the REFERENCES is at end-of-line with comma; sometimes inline; sometimes
    on its own indented line.
    """
    lines = text.splitlines()
    out = []
    in_create_body = False
    body_indent = None
    for line in lines:
        stripped = line.strip()
        # Detect entering CREATE TABLE body: line ends with opening (
        if "CREATE TABLE " in stripped and stripped.rstrip().endswith("("):
            in_create_body = True
            body_indent = None
            out.append(line)
            continue
        if in_create_body and stripped == ");":
            in_create_body = False
            out.append(line)
            continue
        if in_create_body:
            # Any line that has 'REFERENCES' (and is not a standalone ALTER TABLE statement)
            if "REFERENCES" in stripped and not stripped.upper().startswith("ALTER TABLE"):
                # drop the line; preserve any leading/trailing comma on the line above.
                continue
            out.append(line)
            continue
        out.append(line)
    return "\n".join(out)


def strip_standalone_fk_blocks(text: str) -> str:
    """Remove standalone `ALTER TABLE ONLY x.t ADD CONSTRAINT c FOREIGN KEY ...;` blocks.

    pg_dump emits FK constraints as a header comment followed by the ALTER TABLE statement
    spanning 2 lines, plus optionally a `    ADD CONSTRAINT ... FOREIGN KEY` continuation.
    The block ends at the closing `);` of the ALTER statement.
    """
    # Find headers like `-- Name: tbl fk_name; Type: FK CONSTRAINT; Schema: ...` and
    # eat them plus the ALTER TABLE ... ; block that follows.
    pat = re.compile(
        r"(?:--\s*Name:\s+\S+\s+\S+;\s*Type:\s*FK\s+CONSTRAINT;[\s\S]*?"
        r"ALTER TABLE ONLY\s+[^;]+;\s*\n)",
        re.MULTILINE,
    )
    return pat.sub("", text)


def main() -> None:
    src = sys.stdin.read()
    src = strip_standalone_fk_blocks(src)
    src = strip_fks(src)
    src = rewrite_inline(src)
    sys.stdout.write(src)


if __name__ == "__main__":
    main()
