#!/usr/bin/env python3
"""Fails when two headings in docs/ARCHITECTURAL_DECISIONS.md share an ADR number.

Parallel sessions and branches have each taken "the next number"; a duplicate is cheap to prevent and expensive to
untangle once code comments cite it. Usage: adr-numbers-check.py [path]
"""
import re, sys, collections
path = sys.argv[1] if len(sys.argv) > 1 else "docs/ARCHITECTURAL_DECISIONS.md"
seen = collections.defaultdict(list)
for i, line in enumerate(open(path, encoding="utf-8"), 1):
    m = re.match(r"^#{2,4}\s+ADR-(\d+)\b", line)
    if m:
        seen[int(m.group(1))].append(i)
dups = {n: ls for n, ls in seen.items() if len(ls) > 1}
if dups:
    sys.exit("FAIL: duplicate ADR numbers (number: lines): " + ", ".join(f"ADR-{n:03d}: {ls}" for n, ls in sorted(dups.items())))
print(f"ok: {len(seen)} ADR numbers, no duplicates")
