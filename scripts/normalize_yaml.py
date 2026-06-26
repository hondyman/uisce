#!/usr/bin/env python3
"""Normalize YAML files: load and re-dump with consistent indentation and quoting.

This script rewrites the provided YAML files in-place. It preserves mapping order
and sets a wide width to avoid line-wrapping so long URLs/strings stay on one
line. It also ensures a document start marker '---' is present for linters.
"""
import sys
from pathlib import Path
import yaml


def normalize(path: Path) -> None:
    text = path.read_text()
    try:
        docs = list(yaml.safe_load_all(text))
    except Exception as exc:
        print(f"ERROR loading {path}: {exc}")
        return

    # Dump with wide width to avoid automatic wrapping, preserve order
    dumped = yaml.safe_dump_all(
        docs,
        default_flow_style=False,
        sort_keys=False,
        width=1000,
        indent=2,
        explicit_start=True,
    )

    # Ensure trailing newline
    if not dumped.endswith("\n"):
        dumped += "\n"

    path.write_text(dumped)
    print(f"Normalized {path}")


def main(paths):
    for p in paths:
        normalize(Path(p))


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: normalize_yaml.py file1.yaml [file2.yaml ...]")
        sys.exit(1)
    main(sys.argv[1:])
