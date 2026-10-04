#!/usr/bin/env python3
"""Reads `go test -json` output and fails unless the real-database tests actually ran.

A skipped test is not a pass: "no database reachable" and "zero tests matched" are failures.
Fails when
  * go test itself failed (any fail event, build failure),
  * any test skipped (this job sets every variable the tests gate on, so a skip means it did not run),
  * a package listed in the minimums ran fewer passing tests than required (a rename or a typo in
    -run must not quietly shrink the gate),
  * the input has no test events at all.
Usage: realdb-check.py <go-test.json> <minimums.txt>   (minimums: "<package suffix> <min passes>")
"""
import json, sys, collections

events = []
for line in open(sys.argv[1]):
    line = line.strip()
    if not line.startswith("{"):
        continue
    try:
        events.append(json.loads(line))
    except json.JSONDecodeError:
        pass
if not any(e.get("Test") for e in events):
    sys.exit("FAIL: no test events: nothing ran")

passed, skipped, failed = collections.Counter(), [], []
pkg_failed = []
for e in events:
    a, pkg, t = e.get("Action"), e.get("Package", ""), e.get("Test")
    if t and a == "pass" and "/" not in t:
        passed[pkg] += 1
    elif t and a == "skip":
        skipped.append(f"{pkg}.{t}")
    elif t and a == "fail":
        failed.append(f"{pkg}.{t}")
    elif not t and a == "fail":
        pkg_failed.append(pkg)

problems = []
if failed or pkg_failed:
    problems.append(f"failed: {sorted(set(failed + pkg_failed))}")
if skipped:
    problems.append(f"skipped (a skip is a failure here): {sorted(set(skipped))}")
for ln in open(sys.argv[2]):
    ln = ln.split("#")[0].strip()
    if not ln:
        continue
    suffix, want = ln.rsplit(None, 1)
    got = sum(n for p, n in passed.items() if p.endswith(suffix))
    print(f"{suffix}: {got} passed (need >= {want})")
    if got < int(want):
        problems.append(f"{suffix}: {got} passed, need >= {want}")
if problems:
    sys.exit("FAIL:\n  " + "\n  ".join(problems))
print("ok: every real-database test ran and passed")
