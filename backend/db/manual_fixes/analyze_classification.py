#!/usr/bin/env python3
"""Step 0.5 analysis: classify alpha.mdm, quantify cross-home FK breakage.

Read-only analysis. Inputs are the Step 0 inventory artifacts.
Outputs: summary to stdout + draft classification CSV.
"""
from __future__ import annotations

import csv
import re
import sys
from collections import defaultdict
from pathlib import Path

HERE = Path(__file__).parent

# ---------------------------------------------------------------------------
# Load inventories
# ---------------------------------------------------------------------------

def load_names(p: Path) -> list[str]:
    return [ln.strip() for ln in p.read_text().splitlines() if ln.strip()]


alpha_mdm = load_names(HERE / "step0_alpha_mdm_tables.txt")
alpha_oms = load_names(HERE / "step0_alpha_oms_tables.txt")
alpha_edm = load_names(HERE / "step0_alpha_edm_tables.txt")
crims_mdm = load_names(HERE / "step0_crims_mdm_tables.txt")

fks: list[tuple[str, str, str, str]] = []
with (HERE / "step0_fk_inventory.tsv").open() as fh:
    for row in csv.reader(fh, delimiter="\t"):
        if len(row) >= 4:
            fks.append((row[0], row[1], row[2], row[3]))

# COUPLED_FABRIC derivation (mirror of gen_migration_plan.py):
#   (1) seed: every FABRIC table with at least one surviving inbound FK from a
#       non-DROP DATA table.
#   (2) closure: every FABRIC table that is FK-referenced by another FABRIC
#       that is already coupled (transitive over FABRIC inbound edges).
DROP_TABLES = set()
CLS = {}
with (HERE / "classification_draft.csv").open() as f:
    for row in csv.DictReader(f):
        if "DROP" in row["classification"]:
            DROP_TABLES.add(row["source_table"])
        CLS[(row["source_schema"], row["source_table"])] = row["classification"]

seed_coupled: set[str] = set()
ref_to: dict[str, set[str]] = {}
for _conname, src, tgt, _def in fks:
    src_table = src.split(".")[-1]
    if src_table in DROP_TABLES:
        continue
    tgt_table = tgt.split(".")[-1]
    src_cls = CLS.get(("mdm", src_table))
    tgt_cls = CLS.get(("mdm", tgt_table))
    if tgt_cls in ("FABRIC_REF", "FABRIC_RULE") and src_cls in ("DATA", "DATA_LINEAGE"):
        seed_coupled.add(tgt_table)
    if src_cls in ("FABRIC_REF", "FABRIC_RULE") and tgt_cls in ("FABRIC_REF", "FABRIC_RULE"):
        ref_to.setdefault(tgt_table, set()).add(src_table)

COUPLED_FABRIC: set[str] = set(seed_coupled)
stack = list(seed_coupled)
while stack:
    y = stack.pop()
    for x in ref_to.get(y, set()):
        if x not in COUPLED_FABRIC and CLS.get(("mdm", x)) in ("FABRIC_REF", "FABRIC_RULE"):
            COUPLED_FABRIC.add(x)
            stack.append(x)

def bare(qualified: str) -> str:
    return qualified.split(".")[-1]

def schema_of(qualified: str) -> str:
    return qualified.split(".")[0] if "." in qualified else "mdm"

# ---------------------------------------------------------------------------
# Classification rules for alpha.mdm (314)
# ---------------------------------------------------------------------------
# Homes after migration:
#   crims.mdm   — DATA (execution/instance rows)
#   alpha.public.ref_*  — FABRIC_REF (reference/lookup)
#   alpha.public.<rule> — FABRIC_RULE (generalized rules keyed by bo_key)
#   DROP        — obsolete/technical
#   alpha.oms   — METADATA (control plane, stays)
#   alpha.edm   — split: issuer_master -> crims.mdm, rest TBD

DROP_EXACT = {
    "candidates",             # superseded by per-domain *_match_candidate
    "mdm_candidates",         # superseded
    "match_rules",            # superseded by per-domain *_match_rule
    "merge_events",           # superseded by per-domain *_merge_log
    "schema_drift_notifications",
    "mdm_bo_ui_metadata",     # UI metadata, not data-plane
    "mdm_field_rules",        # superseded by per-domain *_dq_rule
    "mdm_funds",              # superseded by crims.mdm.mandate/portfolio family
    "us_equities_master",     # superseded by crims.orm.security
    "risk_rating",            # superseded by per-domain rating family
    "source_versions",        # technical versioning, not referenced by FKs
}

# FABRIC_RULE: per-domain rule tables that generalize to alpha.public keyed by bo_key
RULE_PATTERNS = [
    r"_match_rule$",
    r"_survivorship_rule$",
    r"_dq_rule$",
    r"_source_priority$",
    r"_type_mapping$",
    r"_field_mapping$",
    r"_feed_schedule$",
]

# FABRIC_REF: pure reference/lookup enumerations (tenant-independent codes)
REF_EXACT = {
    "domains", "sources", "source_systems",
    "time_zone", "rolling_convention", "business_day_definition",
    "agreement_type", "source_of_wealth_type",
    "fair_value_level", "fair_value_classification",
    "fatca_crs_status", "fatca_crs_classification",
    "kyc_status", "sanctions_list_source",
    "holiday_type", "holiday_rule_type",
    "curve_category", "curve_compounding_frequency", "curve_interpolation_method",
    "rating_agency", "rating_scale", "rating_outlook", "rating_watch",
    "rating_type", "rating_action_type",
    "price_source", "price_type", "price_observation_type", "price_quality_tier",
    "price_variance_threshold",
    "product_type", "product_status", "product_category", "product_sub_type",
    "product_registration_type", "product_document_type",
    "product_lifecycle_event_type", "product_distribution_channel",
    "product_target_market_type", "product_share_class_type",
    "counterparty_type", "counterparty_status",
    "party_type", "party_status", "party_segment", "party_sub_type",
    "party_document_type", "party_relationship_type", "party_role",
    "benchmark_type", "benchmark_provider", "benchmark_currency_variant",
    "benchmark_rebalance_frequency", "benchmark_return_variant",
    "benchmark_weighting_method", "benchmark_regulatory_regime",
    "benchmark_use_case",
    "ca_event_type", "ca_mandatory_type", "ca_payment_type", "ca_election_type",
    "calendar_type", "calendar_hierarchy_type", "calendar_classification",
    "settlement_calendar",  # calendar-family ref anchored on calendar_master
    "payment_frequency",
}

REF_SUFFIX = re.compile(
    r"_(type|status|category|classification|level|tier|outlook|watch|"
    r"convention|definition|channel|frequency|mode)$"
)

def classify_mdm(name: str) -> str:
    if name in DROP_EXACT:
        return "DROP"
    if name in REF_EXACT or REF_SUFFIX.search(name):
        return "FABRIC_REF"
    for pat in RULE_PATTERNS:
        if re.search(pat, name):
            return "FABRIC_RULE"
    # party/document/etc special: party_golden_* and logs are DATA
    return "DATA"


mdm_class = {t: classify_mdm(t) for t in alpha_mdm}

# alpha.oms: control-plane metadata stays; execution rows move to crims (orm already there)
oms_class: dict[str, str] = {}
for t in alpha_oms:
    if t in ("subtype_registry", "migration_log"):
        oms_class[t] = "METADATA"
    elif t in ("account", "position", "security", "trade_order", "settlement",
               "allocation", "execution", "order_event", "order_link",
               "order_slice", "orders", "position_lots"):
        # crims.orm already has account/allocation/execution/order/position/security/
        # migration_log equivalents — these alpha.oms STI wrappers are superseded
        oms_class[t] = "DROP_SUPERSEDED_BY_CRIMS_ORM"
    else:
        oms_class[t] = "TBD"

# alpha.edm: issuer_master -> crims.mdm; rest classified crudely
edm_class: dict[str, str] = {}
for t in alpha_edm:
    if t == "issuer_master":
        edm_class[t] = "DATA"  # -> crims.mdm.issuer_master
    elif t in ("async_jobs", "etl_run", "scheduled_jobs", "scheduled_job_runs",
               "job_items", "job_exports", "wasm_module_version",
               "performance_settings", "preference_versions",
               "bp_team_instance_roles", "effective_permissions_artifacts"):
        edm_class[t] = "INFRA_KEEP_ALPHA"
    elif t.endswith("_trace") or t.endswith("_lineage") or t == "gold_copy_lineage":
        edm_class[t] = "LINEAGE_TBD"
    else:
        edm_class[t] = "DATA"  # master/golden rows -> crims.mdm

# ---------------------------------------------------------------------------
# Home resolution: where does each alpha.mdm table end up?
# ---------------------------------------------------------------------------
def home(cls: str, name: str) -> str:
    # Architecture rule: data-plane -> crims; control-plane/metadata -> alpha.
    # FABRIC_REF + FABRIC_RULE live in alpha.public (control plane) by default,
    # but per the 2026-09-23 universal fabric rule, any FABRIC table with at least
    # one surviving inbound FK from a DATA table is execution-coupled and moves
    # to crims.mdm with its dependents.  Tables with zero such inbound stay
    # in alpha.public as pure fabric metadata (true config).
    if cls == "DROP":
        return "DROPPED"
    if cls in ("FABRIC_REF", "FABRIC_RULE"):
        if name in COUPLED_FABRIC:
            return "crims.mdm"  # coupled = execute-coupled, move with dependents
        return "alpha.public"
    if cls == "DATA":
        return "crims.mdm"
    return "unknown"

# ---------------------------------------------------------------------------
# FK breakage: for each FK, do from/to land in different homes?
# ---------------------------------------------------------------------------
cross: list[tuple[str, str, str, str, str]] = []  # conname, from, to, from_home, to_home
same = 0
dropped_fk = 0
for conname, frm, to, _def in fks:
    fs, ff = schema_of(frm), bare(frm)
    ts, tf = schema_of(to), bare(to)
    if fs != "mdm":
        continue  # from-side is what moves; all 394 are from mdm
    f_cls = mdm_class.get(ff, "UNKNOWN")
    if ts == "mdm":
        t_cls = mdm_class.get(tf, "UNKNOWN")
        if f_cls == "DROP" or t_cls == "DROP":
            dropped_fk += 1
            continue
        fh, th = home(f_cls, ff), home(t_cls, tf)
        if fh != th:
            cross.append((conname, ff, tf, fh, th))
        else:
            same += 1
    elif ts == "edm":
        # edm.issuer_master -> crims.mdm ; other edm tables stay alpha (TBD)
        if tf == "issuer_master":
            fh, th = home(f_cls, ff), "crims.mdm"
            if fh != th:
                cross.append((conname, ff, "edm." + tf, fh, th))
            else:
                same += 1
        else:
            cross.append((conname, ff, "edm." + tf, home(f_cls, ff), "alpha.edm"))
    elif ts == "public":
        # fund_hierarchy -> public.tenants : decided DROP
        dropped_fk += 1
    else:
        cross.append((conname, ff, to, home(f_cls, ff), ts))

# ---------------------------------------------------------------------------
# Inbound-FK pressure on FABRIC_REF / FABRIC_RULE targets
# ---------------------------------------------------------------------------
inbound: dict[str, list[str]] = defaultdict(list)
for conname, frm, to, _def in fks:
    if schema_of(to) == "mdm":
        inbound[bare(to)].append(bare(frm))

ref_breakage: list[tuple[str, int, list[str]]] = []
rule_breakage: list[tuple[str, int, list[str]]] = []
for t, c in sorted(mdm_class.items()):
    if c in ("FABRIC_REF", "FABRIC_RULE"):
        froms = inbound.get(t, [])
        # only count froms that survive (not DROP themselves)
        surviving = [f for f in froms if mdm_class.get(f) != "DROP"]
        if surviving:
            rec = (t, len(surviving), surviving[:6])
            (ref_breakage if c == "FABRIC_REF" else rule_breakage).append(rec)

# ---------------------------------------------------------------------------
# Report
# ---------------------------------------------------------------------------
def tally(d: dict[str, str]) -> dict[str, int]:
    out: dict[str, int] = defaultdict(int)
    for v in d.values():
        out[v] += 1
    return dict(sorted(out.items()))

print("=" * 72)
print("STEP 0.5 CLASSIFICATION ANALYSIS")
print("=" * 72)
print(f"alpha.mdm tables : {len(alpha_mdm)}")
print(f"alpha.oms tables : {len(alpha_oms)}")
print(f"alpha.edm tables : {len(alpha_edm)}")
print(f"crims.mdm tables : {len(crims_mdm)}")
print(f"FK inventory     : {len(fks)}")
print()
print("alpha.mdm classification tally:")
for k, v in tally(mdm_class).items():
    print(f"  {k:16s} {v:4d}")
print()
print("alpha.oms classification tally:")
for k, v in tally(oms_class).items():
    print(f"  {k:32s} {v:4d}")
print()
print("alpha.edm classification tally:")
for k, v in tally(edm_class).items():
    print(f"  {k:24s} {v:4d}")
print()
print("-" * 72)
print("FK OUTCOMES (from mdm side, 394 total)")
print(f"  same-home (preserved)     : {same}")
print(f"  dropped (DROP side or G18): {dropped_fk}")
print(f"  CROSS-HOME (break)        : {len(cross)}")
print("-" * 72)
if cross:
    by_pair: dict[tuple[str, str], int] = defaultdict(int)
    for _c, _f, _t, fh, th in cross:
        by_pair[(fh, th)] += 1
    print("cross-home breakdown by (from_home -> to_home):")
    for (fh, th), n in sorted(by_pair.items(), key=lambda x: -x[1]):
        print(f"  {fh:16s} -> {th:16s}  {n:4d}")
    print()
    print("cross-home FKs (conname | from | to | from_home | to_home):")
    for c, f, t, fh, th in cross:
        print(f"  {c:36s} {f:32s} {t:32s} {fh:14s} {th}")
print()
print("-" * 72)
print(f"FABRIC_REF targets with inbound FKs from surviving tables: {len(ref_breakage)}")
for t, n, froms in sorted(ref_breakage, key=lambda x: -x[1]):
    print(f"  {t:40s} {n:3d} inbound  e.g. {', '.join(froms)}")
print()
print(f"FABRIC_RULE targets with inbound FKs: {len(rule_breakage)}")
for t, n, froms in sorted(rule_breakage, key=lambda x: -x[1]):
    print(f"  {t:40s} {n:3d} inbound  e.g. {', '.join(froms)}")
print()
print("-" * 72)
print("DROP list:")
for t in sorted(t for t, c in mdm_class.items() if c == "DROP"):
    inb = len(inbound.get(t, []))
    print(f"  {t:40s} inbound_fks={inb}")
print()
print("oms classification detail:")
for t in sorted(oms_class):
    print(f"  {t:40s} {oms_class[t]}")
print()
print("edm non-DATA detail:")
for t in sorted(edm_class):
    if edm_class[t] != "DATA":
        print(f"  {t:40s} {edm_class[t]}")

# ---------------------------------------------------------------------------
# Write draft CSV
# ---------------------------------------------------------------------------
csv_path = HERE / "classification_draft.csv"
with csv_path.open("w", newline="") as fh:
    w = csv.writer(fh)
    w.writerow(["source_schema", "source_table", "classification", "home", "inbound_fks", "notes"])
    for t in sorted(alpha_mdm):
        c = mdm_class[t]
        w.writerow(["mdm", t, c, home(c, t), len(inbound.get(t, [])), ""])
    for t in sorted(alpha_oms):
        w.writerow(["oms", t, oms_class[t], "alpha.oms" if oms_class[t] == "METADATA" else "DROPPED" if "DROP" in oms_class[t] else "TBD", len(inbound.get(t, [])), ""])
    for t in sorted(alpha_edm):
        c = edm_class[t]
        tgt = "crims.mdm" if c == "DATA" else "alpha.edm" if c in ("INFRA_KEEP_ALPHA", "LINEAGE_TBD") else "TBD"
        if t == "issuer_master":
            tgt = "crims.mdm"
        w.writerow(["edm", t, c, tgt, len(inbound.get(t, [])), ""])
    for t in sorted(crims_mdm):
        note = "COLLISION" if t in ("party", "rating_scale") else "pre-existing"
        w.writerow(["crims.mdm(existing)", t, "EXISTING", "crims.mdm", 0, note])

print()
print(f"draft CSV written: {csv_path}")
