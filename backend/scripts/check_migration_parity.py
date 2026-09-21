#!/usr/bin/env python3
"""Migration parity guard.

Two modes (combine freely); both honour backend/db/PARITY_ALLOWLIST.tsv.

--db
    Compares backend/db/migrations against oms.migration_log (needs DATABASE_URL
    and psql):
      1. applied-but-missing   every filename in the log must exist on disk
      2. applied-but-different for files that exist, disk sha256 must equal the
                               sha256 the runner recorded
    Needs the real (shared) DB, so run it where that is reachable: locally, on
    backend start, or from a runner with DB access. It is meaningless against an
    ephemeral CI database.

--git-base REF
    DB-free, CI-safe. Fails if any *.up.sql that already exists at REF was
    modified, renamed or deleted since. Editing a shipped migration is the root
    cause of "content has changed" drift.

Allowlist (backend/db/PARITY_ALLOWLIST.tsv), tab-separated:
    filename <TAB> kind <TAB> reason
kind is one of: missing | different | modified. The reason is REQUIRED (min 15
characters) so an exception is always explained. Stale entries (no longer
needed) are reported so the list shrinks over time.

Exit codes: 0 clean, 1 findings not allowlisted, 2 usage/config error.
"""
import argparse
import hashlib
import os
import subprocess
import sys

KINDS = {"missing", "different", "modified"}


def repo_root():
    r = subprocess.run(["git", "rev-parse", "--show-toplevel"], capture_output=True, text=True)
    if r.returncode != 0:
        sys.exit("error: not inside a git repository")
    return r.stdout.strip()


def load_allowlist(path):
    allow = {}
    if not os.path.exists(path):
        return allow
    for n, line in enumerate(open(path, encoding="utf-8"), 1):
        raw = line.rstrip("\n")
        if not raw.strip() or raw.lstrip().startswith("#"):
            continue
        parts = raw.split("\t")
        if len(parts) < 3:
            print(f"error: {path}:{n}: expected 'filename<TAB>kind<TAB>reason'", file=sys.stderr)
            sys.exit(2)
        name, kind, reason = parts[0].strip(), parts[1].strip(), "\t".join(parts[2:]).strip()
        if kind not in KINDS:
            print(f"error: {path}:{n}: kind must be one of {sorted(KINDS)}, got {kind!r}", file=sys.stderr)
            sys.exit(2)
        if len(reason) < 15:
            print(f"error: {path}:{n}: a reason of at least 15 characters is required for {name}", file=sys.stderr)
            sys.exit(2)
        allow[(name, kind)] = reason
    return allow


def disk_hashes(mig_dir):
    out = {}
    for dp, _dn, fns in os.walk(mig_dir):
        for f in fns:
            if f.endswith(".up.sql"):
                p = os.path.join(dp, f)
                with open(p, "rb") as fh:
                    out[os.path.relpath(p, mig_dir)] = hashlib.sha256(fh.read()).hexdigest()
    return out


def log_hashes():
    url = os.environ.get("DATABASE_URL")
    if not url:
        print("error: --db needs DATABASE_URL set", file=sys.stderr)
        sys.exit(2)
    r = subprocess.run(["psql", url, "-X", "-A", "-t", "-F", "\t", "-c",
                        "SELECT filename, sha256 FROM oms.migration_log"], capture_output=True, text=True)
    if r.returncode != 0:
        detail = r.stderr.strip().splitlines()[-1] if r.stderr.strip() else "no output"
        print(f"error: could not read oms.migration_log: {detail}", file=sys.stderr)
        sys.exit(2)
    out = {}
    for line in r.stdout.splitlines():
        if "\t" in line:
            f, s = line.split("\t", 1)
            out[f] = s.strip().lower()
    return out


def git_changed(root, base, mig_rel):
    # cwd=root: the pathspec is repo-relative and must not depend on where this runs from.
    r = subprocess.run(["git", "diff", "--name-status", "--diff-filter=MDR", f"{base}...HEAD", "--", mig_rel],
                       capture_output=True, text=True, cwd=root)
    if r.returncode != 0:
        print(f"error: git diff against {base} failed: {r.stderr.strip()}", file=sys.stderr)
        sys.exit(2)
    out = []
    for line in r.stdout.splitlines():
        parts = line.split("\t")
        path = parts[-1]  # for renames the new path is last; the old path is parts[1]
        old = parts[1]
        if old.startswith(mig_rel + "/") and old.endswith(".up.sql"):
            out.append((old[len(mig_rel) + 1:], parts[0][0]))
    return out


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--db", action="store_true", help="compare disk against oms.migration_log")
    ap.add_argument("--git-base", metavar="REF", help="fail if shipped migrations changed since REF")
    ap.add_argument("--allowlist", help="override allowlist path")
    a = ap.parse_args()
    if not a.db and not a.git_base:
        ap.error("choose --db and/or --git-base REF")

    root = repo_root()
    mig_rel = "backend/db/migrations"
    mig_dir = os.path.join(root, mig_rel)
    allow = load_allowlist(a.allowlist or os.path.join(root, "backend/db/PARITY_ALLOWLIST.tsv"))
    used = set()
    failures, allowed = [], []

    def record(name, kind, detail):
        key = (name, kind)
        if key in allow:
            used.add(key)
            allowed.append((name, kind, detail, allow[key]))
        else:
            failures.append((name, kind, detail))

    if a.db:
        disk, log = disk_hashes(mig_dir), log_hashes()
        for name in sorted(log):
            if name not in disk:
                record(name, "missing", "recorded as applied, no file on disk")
            elif disk[name] != log[name]:
                record(name, "different", f"disk {disk[name][:12]} != applied {log[name][:12]}")
    if a.git_base:
        for name, status in git_changed(root, a.git_base, mig_rel):
            what = {"M": "modified", "D": "deleted", "R": "renamed"}.get(status, status)
            record(name, "modified", f"shipped migration {what} since {a.git_base}")

    stale = sorted(k for k in allow if k not in used and (
        (k[1] in ("missing", "different") and a.db) or (k[1] == "modified" and a.git_base)))

    for name, kind, detail, reason in allowed:
        print(f"  allowed  [{kind}] {name}: {reason}")
    for k in stale:
        print(f"  STALE    [{k[1]}] {k[0]}: no longer needed, remove from allowlist")
    if failures:
        print(f"\nFAIL: {len(failures)} migration parity finding(s) not in the allowlist:")
        for name, kind, detail in failures:
            print(f"  [{kind}] {name}: {detail}")
        print("\nFix the file to match what was applied, or add it to backend/db/PARITY_ALLOWLIST.tsv "
              "with a reason (filename<TAB>kind<TAB>reason).")
        sys.exit(1)
    print(f"OK: no unallowlisted findings ({len(allowed)} allowed, {len(stale)} stale allowlist entries).")


if __name__ == "__main__":
    main()
