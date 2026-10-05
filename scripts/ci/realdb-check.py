#!/usr/bin/env python3
"""Fails unless the real-database tests actually ran: listed set == passed set (minus named exclusions).

The expected set comes from `go test -list <pattern> <package>` over scripts/ci/realdb-packages.txt, so nothing is
counted by hand. Fails on: a failed or skipped test, a listed test that did not pass (deleted, renamed, never ran), a
go test failure, empty input. Usage: realdb-check.py <go-test.json> <realdb-packages.txt> <backend dir>
"""
import json, subprocess, sys, collections

events_file, cfg, backend = sys.argv[1:4]
packages, exclude = [], set()
for ln in open(cfg):
    ln = ln.strip()
    if not ln or ln.startswith("#"):
        continue
    kind, rest = ln.split(None, 1)
    if kind == "package":
        d, pat = rest.split(None, 1)
        packages.append((d, pat))
    elif kind == "exclude":
        exclude.add(rest.split()[0])
    else:
        sys.exit(f"FAIL: unknown line in {cfg}: {ln}")

expected = {}
for d, pat in packages:
    r = subprocess.run(["go", "test", "-list", pat, "./" + d + "/"], cwd=backend, capture_output=True, text=True)
    if r.returncode != 0:
        sys.exit(f"FAIL: go test -list {d}: {r.stderr[-500:]}")
    names = {l.strip() for l in r.stdout.splitlines() if l.startswith("Test")} - exclude
    if not names:
        sys.exit(f"FAIL: no tests listed for {d} (pattern {pat!r}): zero tests matched is a failure")
    expected[d] = names

events = []
for line in open(events_file):
    if line.startswith("{"):
        try:
            events.append(json.loads(line))
        except json.JSONDecodeError:
            pass
if not any(e.get("Test") for e in events):
    sys.exit("FAIL: no test events: nothing ran")

passed = collections.defaultdict(set)
skipped, failed, pkg_failed, out = set(), set(), set(), collections.defaultdict(list)
for e in events:
    a, pkg, t = e.get("Action"), e.get("Package", ""), e.get("Test")
    if a == "output" and t:
        out[f"{pkg}.{t}"].append(e.get("Output", ""))
    if t and "/" not in t:
        if a == "pass":
            passed[pkg].add(t)
        elif a == "skip":
            skipped.add(f"{pkg}.{t}")
        elif a == "fail":
            failed.add(f"{pkg}.{t}")
    elif t and a == "fail":
        failed.add(f"{pkg}.{t}")
    elif not t and a == "fail":
        pkg_failed.add(pkg)

for name in sorted(failed)[:5]:
    print(f"--- output of {name}\n" + "".join(out[name])[-3000:])

problems = []
if failed or pkg_failed:
    problems.append(f"failed: {sorted(failed | pkg_failed)}")
if skipped:
    problems.append(f"skipped (a skip is a failure here): {sorted(skipped)}")
for d, names in expected.items():
    got = {t for p, ts in passed.items() if p.endswith("/" + d) for t in ts}
    missing = sorted(names - got)
    print(f"{d}: {len(names & got)}/{len(names)} listed tests passed")
    if missing:
        problems.append(f"{d}: listed but did not pass: {missing}")
if problems:
    sys.exit("FAIL:\n  " + "\n  ".join(problems))
print("ok: every listed real-database test ran and passed")
