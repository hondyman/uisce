#!/usr/bin/env python3
"""mismatch_helper.py: report alpha columns not present in crims.

Reads /tmp/mh_alpha.txt and /tmp/mh_crims.txt; lines are 'name:type:len'
triples.  Outputs any alpha column for which no crims column has exactly
the same 'name:type:len'.
"""
import sys

def parse(path):
    txt = open(path).read().strip()
    return set(txt.split('\n')) if txt else set()

alpha = parse('/tmp/mh_alpha.txt')
crims = parse('/tmp/mh_crims.txt')
missing = sorted(alpha - crims)
if missing:
    print('\n'.join(missing))

