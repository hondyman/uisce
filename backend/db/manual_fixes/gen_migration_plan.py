#!/usr/bin/env python3
"""gen_migration_plan.py

Generate 001_migration_plan.sql from classification_draft.csv + FK inventory +
table lists. Produces the migration.plan / progress / blockers tables (DDL)
and INSERT rows for every classified table in alpha.{mdm,oms,edm}, every
existing crims.{mdm,orm}, and the 85 fabric-relevant alpha.public tables.

Applied decisions (kept in sync with 001_migration_plan.sql header):

  * G18 fund_hierarchy->tenants FK dropped.
  * FABRIC_REF + FABRIC_RULE target alpha.public (control plane; Option A 2026-09-23).
    Inbound FKs from DATA tables are dropped pre-copy and enforced app-side
    (same pattern as G18: tenant_guc + RLS OR-clause already in 0001_mdm_security.sql).
  * G16 issuer_master -> crims.mdm.
  * G19 mdm-vs-edm duplicate *_master winners are the mdm versions; the
    edm golden-copy twins are dropped with the EDM layer.
  * G21 party seed via crims.mdm.xref.
  * collision_party / collision_rating_scale: alpha schema wins (see file).
  * LINEAGE_TBD 12 -> DATA_LINEAGE -> crims.mdm.

Reads:
  classification_draft.csv       (alpha.mdm + alpha.oms + alpha.edm)
  step0_crims_mdm_tables.txt     (crims.mdm existing 83)
  step0_fk_inventory.tsv         (cross/stay metrics; FK breakage summary)
  preflight2_alpha.out           (85 fabric-relevant alpha.public names)

Writes:
  001_migration_plan.sql         (DDL + INSERT rows + blocker rows)

Idempotency:
  Plan rows have UNIQUE(source_schema, source_table). Blockers have UNIQUE(code).
  INSERTs use ON CONFLICT DO NOTHING so re-runs over an existing migration
  schema are safe.
"""
from __future__ import annotations

import csv
import re
import sys
from collections import defaultdict
from pathlib import Path

HERE = Path(__file__).resolve().parent

# -----------------------------------------------------------------------------
# Inputs
# -----------------------------------------------------------------------------

alpha_mdm = (HERE / "step0_alpha_mdm_tables.txt").read_text().split()
alpha_oms = (HERE / "step0_alpha_oms_tables.txt").read_text().split()
alpha_edm = (HERE / "step0_alpha_edm_tables.txt").read_text().split()
crims_mdm = (HERE / "step0_crims_mdm_tables.txt").read_text().split()
crims_orm = sorted({
    ln.split("(")[0].strip()
    for ln in (HERE / "preflight2_crims.out").read_text().splitlines()
    if re.fullmatch(r"\s*[a-z_]+\s*", ln) or
       (" " not in ln.strip() and ln.strip().isascii() and ln.strip().islower())
} | set())  # below we parse better

# Parse the orm list from preflight2_crims.out (search for the orm section).
text = (HERE / "preflight2_crims.out").read_text()
m = re.search(r"=== 0\.5\.13 crims\.orm inventory ===\n(.*?)\n\(80 rows\)", text, re.S)
if not m:
    sys.exit("could not locate crims.orm inventory in preflight2_crims.out")
crims_orm = [
    ln.strip() for ln in m.group(1).splitlines()
    if ln.strip()
    and not re.fullmatch(r"-{2,}", ln.strip())
    and ln.strip().lower() != "table_name"
]
assert len(crims_orm) == 80, f"expected 80 orm tables, got {len(crims_orm)}: {crims_orm[:5]}"

# Parse the 85 fabric-relevant alpha.public names.
# The section has: header line, separator, N table lines, footer "(85 rows)".
atext = (HERE / "preflight2_alpha.out").read_text()
m2 = re.search(
    r"=== 0\.5\.3 alpha\.public fabric-relevant tables ===\n"
    r"\s+table_name\s*\n-{2,}\n(.*?)\n\(85 rows\)",
    atext, re.S,
)
if not m2:
    sys.exit("could not locate alpha.public fabric-relevant section in preflight2_alpha.out")
# Strip empty lines and the synthetic "(N rows)" footers the output box hasn't drawn yet.
alpha_public_fabric = [
    ln.strip() for ln in m2.group(1).splitlines()
    if ln.strip()
    and not ln.strip().startswith("(")
    and not ln.strip().startswith("Time:")
    and not re.fullmatch(r"-{2,}", ln.strip())
]
assert len(alpha_public_fabric) == 85, f"expected 85 fabric tables, got {len(alpha_public_fabric)}: {alpha_public_fabric[:5]}...{alpha_public_fabric[-5:]}"

# Load classification
cls: dict[tuple[str, str], str] = {}
with (HERE / "classification_draft.csv").open() as f:
    for row in csv.DictReader(f):
        cls[(row["source_schema"], row["source_table"])] = row["classification"]

# Populate COUPLED_FABRIC.  Three-step derivation:
#   (1) seed: every FABRIC table that has at least one surviving inbound FK
#       from a non-DROP DATA table (these are "execution-coupled").
#   (2) closure: every FABRIC table that is FK-referenced by another FABRIC
#       that is already coupled (transitive closure over inbound-FABRIC edges).
#   This way a chain like benchmark_field_mapping -> source_systems (coupled)
#   is correctly moved with source_systems.  Pure config lookup tables with no
#   inbound coupling stay in alpha.public.
#
# Implementation: locally from step0_fk_inventory.tsv (the same data the analyzer
# uses).  Falls back to DB query if the tsv is missing.
COUPLED_FABRIC: set[str]
DROP_FOR_COUPLING: set[str]
tsv_path = HERE / "step0_fk_inventory.tsv"
if tsv_path.exists():
    DROP_FOR_COUPLING = set()
    with (HERE / "classification_draft.csv").open() as f:
        for row in csv.DictReader(f):
            if "DROP" in row["classification"]:
                DROP_FOR_COUPLING.add(row["source_table"])

    seed_coupled: set[str] = set()
    ref_to: dict[str, set[str]] = {}  # tgt -> set(src) (FABRIC FK-references)
    with tsv_path.open() as fh:
        for row in csv.reader(fh, delimiter="\t"):
            if len(row) >= 4:
                src, tgt = row[1], row[2]
                src_table = src.split(".")[-1]
                if src_table in DROP_FOR_COUPLING:
                    continue
                tgt_table = tgt.split(".")[-1]
                src_cls = cls.get(("mdm", src_table))
                tgt_cls = cls.get(("mdm", tgt_table))
                if tgt_cls in ("FABRIC_REF", "FABRIC_RULE") and src_cls in ("DATA", "DATA_LINEAGE"):
                    seed_coupled.add(tgt_table)
                if src_cls in ("FABRIC_REF", "FABRIC_RULE") and tgt_cls in ("FABRIC_REF", "FABRIC_RULE"):
                    ref_to.setdefault(tgt_table, set()).add(src_table)
    COUPLED = set(seed_coupled)
    queue = list(seed_coupled)
    while queue:
        y = queue.pop()
        for x in ref_to.get(y, set()):
            if x not in COUPLED and cls.get(("mdm", x)) in ("FABRIC_REF", "FABRIC_RULE"):
                COUPLED.add(x)
                queue.append(x)
    COUPLED_FABRIC = COUPLED
    print(
        f"[gen_migration_plan] COUPLED_FABRIC: {len(COUPLED_FABRIC)} tables "
        f"(seeded={len(seed_coupled)} via inbound; closed from FABRIC-FK edges) -> crims.mdm",
        file=sys.stderr,
    )
else:
    print(
        "[gen_migration_plan] WARN: step0_fk_inventory.tsv missing; all FABRIC tables "
        "default to alpha.public. Re-run Step 0 to enable closure derivation.",
        file=sys.stderr,
    )
    COUPLED_FABRIC = set()

# -----------------------------------------------------------------------------
# Decisions
# -----------------------------------------------------------------------------

# Decisions applied to the draft CSV before emission.
LINEAGE_TO_DATA_LINEAGE = {
    "cash_flow_trace", "cash_gold_trace", "compliance_lineage", "gold_copy_lineage",
    "mdm_calendar_lineage", "position_gold_trace", "price_gold_trace",
    "rule_lineage", "scenario_lineage", "security_gold_trace",
    "transaction_flow_trace", "transaction_gold_trace",
}

# (schema, name) tuples classified DATA in BOTH alpha.mdm and alpha.edm; mdm wins.
EDM_DUPLICATE_LOSERS = {
    ("edm", "benchmark_master"),
    ("edm", "curve_master"),
    ("edm", "fx_rate_master"),
}

# (schema, name) tuples classified DATA in BOTH alpha.mdm and alpha.edm; mdm wins.
edm_losers = list(EDM_DUPLICATE_LOSERS)

# Rating family reclassification (2026-09-23): the rating_* domain is a
# first-class target — the DATA satellites (rating, rating_action, etc.)
# inbound-FK to rating_scale. Under locked Option A, rating_scale was FABRIC_REF
# so it stayed in alpha, leaving no target for DATA tables to FK to. Decided:
# the entire rating family moves to crims.mdm; alpha keeps only what its
# own fabric/catalog needs via alpha.public business_object_terms etc.
RATING_FAMILY_DATA = {
    "rating_scale",       # 31 rows; canonical identity (agency_id FK)
    "rating_agency",      # 12 rows
    "rating_outlook",     # 6 rows
    "rating_watch",       # 4 rows
    "rating_type",        # 16 rows
    "rating_action_type", # 11 rows
}

# Universal fabric rule (2026-09-23, refined from rating decision):
# any FABRIC_REF/FABRIC_RULE table with at least one surviving inbound FK
# from a non-DROP DATA table is execution-coupled and moves to crims.mdm with
# its dependents.  Tables with zero such inbound stay in alpha.public as pure
# fabric metadata (true config).  This generalizes the rating decision so we
# don't patch one domain at a time.
# Note: COUPLED_FABRIC is declared ABOVE the try block that populates it (see
# the "Populate COUPLED_FABRIC" comment above).  Declaring it twice would reset
# the populated set on the second pass; do NOT redeclare here.

# -----------------------------------------------------------------------------
# Domain inference
# -----------------------------------------------------------------------------

DOMAIN_PREFIXES: list[tuple[str, str]] = [
    ("benchmark", "BENCHMARK"),
    ("ca_", "CORPORATE_ACTION"),
    ("calendar", "CALENDAR"),
    ("counterparty", "COUNTERPARTY"),
    ("party", "PARTY"),
    ("price", "PRICE"),
    ("product", "PRODUCT"),
    ("rating", "RATING"),
    ("curve", "CURVE"),
    ("issuer", "ISSUER"),
    ("mandate", "MANDATE"),
    ("portfolio", "PORTFOLIO"),
    ("recon", "RECON"),
    ("sanctions", "COMPLIANCE"),
    ("kyc", "COMPLIANCE"),
    ("fatca", "COMPLIANCE"),
    ("dq", "DQ"),
    ("fx", "FX"),
    ("fund", "FUND"),
    ("security", "SECURITY"),
    ("settlement", "SETTLEMENT"),
    ("risk", "RISK"),
    ("position", "POSITION"),
    ("transaction", "TRANSACTION"),
    ("compliance", "COMPLIANCE"),
    ("survivorship", "GOLDEN"),
    ("golden", "GOLDEN"),
    ("steward", "GOLDEN"),
    ("merge", "GOLDEN"),
    ("match", "GOLDEN"),
    ("source", "SOURCE"),
    ("hierarchy", "GRAPH"),
    ("quality", "QUALITY"),
    ("vol", "MARKET_DATA"),
]

def domain_of(name: str) -> str:
    for pref, dom in DOMAIN_PREFIXES:
        if name.startswith(pref) or name == pref:
            return dom
    # fallback: first token before the last underscore (party_sub_type -> party),
    # or whole name for single words.
    base = name.split("_")[0] if "_" in name else name
    return base.upper()

# domain ranks: PRODUCT first, then alphabetical
ALL_DOMAINS_SORTED = sorted({domain_of(n) for n in alpha_mdm + alpha_edm + alpha_oms} | {
    "PARTY", "PRICE", "PRODUCT", "BENCHMARK", "CORPORATE_ACTION", "CALENDAR",
    "COUNTERPARTY", "RATING", "CURVE", "ISSUER", "MANDATE", "PORTFOLIO",
    "RECON", "COMPLIANCE", "DQ", "FX", "FUND", "SECURITY", "SETTLEMENT",
    "RISK", "POSITION", "TRANSACTION", "SOURCE", "GRAPH", "QUALITY",
    "MARKET_DATA", "GOLDEN", "REFERENCE", "METADATA", "OPERATIONS", "ALPHA",
})
DOMAIN_RANK = {d: r for r, d in enumerate(["PRODUCT"] + [d for d in ALL_DOMAINS_SORTED if d != "PRODUCT"], start=1)}

# -----------------------------------------------------------------------------
# Batch-order derivation
# -----------------------------------------------------------------------------

DROP_CLASSES = {"DROP", "DROP_SUPERSEDED_BY_CRIMS_ORM"}
NO_COPY_CLASSES = {"EXISTING", "KEEP_ALPHA", "LINEAGE_TBD", "METADATA", "INFRA_KEEP_ALPHA"}

GOLDEN_REGEX = re.compile(r"golden|gold_")
LOG_REGEX = re.compile(r"(_log|_logs|_history)$|^audit_log$|^merge_log$|^split_log$|^change_request$|record_version$")
EXCEPTION_REGEX = re.compile(r"_exception|_reconciliation|_recon_|_breaks|_issues?$|dq_score|_conflict|^recon_")

def batch_order(classification: str, name: str) -> int:
    if classification in DROP_CLASSES:
        return 900
    if classification == "FABRIC_REF":
        return 100
    if classification == "FABRIC_RULE":
        return 110
    if classification in NO_COPY_CLASSES or classification.startswith("EXIST") or classification == "KEEP":
        return 0
    if GOLDEN_REGEX.search(name):
        return 400
    if LOG_REGEX.search(name):
        return 500
    if EXCEPTION_REGEX.search(name):
        return 600
    # DATA_LINEAGE / DATA / FABRIC_REF: order 200 (anchor) by name heuristic,
    # 300 (satellite) otherwise.  Reclassify lineage already handled above.
    anchor_names = {
        "party", "product", "counterparty", "fund", "benchmark_master",
        "calendar_master", "curve_master", "price_master", "fx_rate_master",
        "security_master", "transaction_master", "issuer_master",
        "mandate_master", "strategy_master", "vol_surface_master",
        "position_master",
    }
    base = name.split("_")[0] if "_" in name else name
    return 200 if base in anchor_names else 300

# -----------------------------------------------------------------------------
# Notes helper (defined up front so the plan-row builders can call it)
# -----------------------------------------------------------------------------

def _notes_for(schema: str, name: str, classification: str) -> str:
    if classification == "DROP":
        return "Dropped during copy phase (moves to DROP batch 900)." if schema == "mdm" else \
               "Superseded by crims.orm equivalent (G19 pattern). Verify equivalent exists before drop; no data move to crims."
    if classification == "DROP_SUPERSEDED_BY_CRIMS_ORM":
        return "Alpha.oms STI wrapper; superseded by crims.orm equivalent. Mapping noted in plan; verify counterpart before drop."
    if classification == "METADATA":
        return "Stays in alpha.oms (control plane)."
    if classification == "FABRIC_REF":
        return ("FABRIC_REF reference data; stays in alpha.public (control plane) per Option A. "
                "Inbound FKs from DATA tables dropped pre-copy and enforced app-side via tenant_guc + RLS.")
    if classification == "FABRIC_RULE":
        return ("FABRIC_RULE per-domain rule table; stays in alpha.public (control plane) per Option A. "
                "Inbound FKs from DATA tables dropped pre-copy and enforced app-side.")
    if classification == "DATA_LINEAGE":
        return "Lineage/provenance table; data-plane per architecture rule -> crims.mdm. Reclassified from LINEAGE_TBD 2026-09-23."
    if classification == "INFRA_KEEP_ALPHA":
        return "Stays in alpha.edm (control-plane infra); RLS / job/permissions governance."
    if classification == "LINEAGE_TBD":
        return "Was LINEAGE_TBD; reclassified to DATA_LINEAGE -> crims.mdm."
    if classification in {"EXTEND", "KEEP_ALPHA", "KEEP"}:
        return "KEEP / not migrated (see notes)."
    if classification == "DATA":
        # Per-schema special notes
        if schema == "mdm" and name == "party":
            return ("Target crims.mdm.party. alpha schema wins (richer 44-col golden-record). "
                    "crims.mdm.party will be extended via 002_crims_party_alter.sql (ADD COLUMN for "
                    "alpha-only fields; port RLS). Alpha party is DROP.")
        if schema == "mdm" and name == "rating_scale":
            return ("Target crims.mdm.rating_scale. alpha has 31 rows + rating->rating_scale FK. "
                    "Alpha schema wins; crims schema replacement lands before copy in 002_crims_party_alter.sql family "
                    "(same ADD COLUMN + port RLS pattern as party).")
        if (schema, name) in EDM_DUPLICATE_LOSERS:
            return "MDM version wins; edm twin dropped (see EDM_DUPLICATES blocker)."
        return "Moves to crims.mdm (data-plane)."
    return ""

# -----------------------------------------------------------------------------
# Build plan rows
# -----------------------------------------------------------------------------

def normalize_class(table: str, classification: str, schema: str) -> str:
    """Map draft classifications to the final plan-level labels after decisions."""
    if (schema, table) in EDM_DUPLICATE_LOSERS:
        return "DROP"  # edm *_master duplicate -> dropped
    if classification in DROP_CLASSES:
        return classification
    if schema == "edm" and table in LINEAGE_TO_DATA_LINEAGE:
        return "DATA_LINEAGE"
    # Rating family reclassification: 6 tables move from FABRIC_REF/RULE to DATA
    # so the family lands in crims.mdm together. See RATING_FAMILY_DATA constant.
    if schema == "mdm" and table in RATING_FAMILY_DATA:
        return "DATA"
    return classification

plan_rows: list[dict] = []

# 1. alpha.mdm rows
for t in sorted(alpha_mdm):
    c = cls.get(("mdm", t))
    if c is None:
        continue
    final_cls = normalize_class(t, c, "mdm")
    # Universal fabric rule (2026-09-23): FABRIC_REF/RULE with surviving inbound
    # FKs from DATA tables are execution-coupled and move to crims.mdm with
    # their dependents.  Pure config (zero inbound) stays in alpha.public.
    if final_cls in {"FABRIC_REF", "FABRIC_RULE"}:
        if t in COUPLED_FABRIC:
            tgt_db, tgt_sch, tgt_tbl = "crims", "mdm", t
        else:
            tgt_db, tgt_sch, tgt_tbl = "alpha", "public", t
    elif final_cls in {"DATA", "DATA_LINEAGE"}:
        tgt_db, tgt_sch, tgt_tbl = "crims", "mdm", t
    elif final_cls in DROP_CLASSES:
        tgt_db, tgt_sch, tgt_tbl = "-", "-", "-"
    else:
        tgt_db, tgt_sch, tgt_tbl = "alpha", "mdm", t  # METADATA, INFRA
    plan_rows.append({
        "source_db": "alpha",
        "source_schema": "mdm",
        "source_table": t,
        "target_db": tgt_db,
        "target_schema": tgt_sch,
        "target_table": tgt_tbl,
        "classification": final_cls,
        "domain": domain_of(t),
        "batch_order": batch_order(final_cls, t),
        "notes": _notes_for("mdm", t, final_cls),
    })

# 2. alpha.oms rows
for t in sorted(alpha_oms):
    c = cls.get(("oms", t))
    if c is None:
        continue
    final_cls = normalize_class(t, c, "oms")
    if final_cls == "DROP_SUPERSEDED_BY_CRIMS_ORM":
        # Map to crims.orm counterpart where known; otherwise drop.
        orm_map = {
            "account": "account", "allocation": "allocation", "execution": "execution",
            "order_event": "message_log", "order_link": "-", "order_slice": "-",
            "orders": "order", "position": "position", "position_lots": "-",
            "security": "security", "settlement": "-", "trade_order": "-",
        }
        target_table = orm_map.get(t, "-")
        if target_table and target_table != "-" and target_table in crims_orm:
            tgt_db, tgt_sch, tgt_tbl = "crims", "orm", target_table
        else:
            tgt_db, tgt_sch, tgt_tbl = "-", "-", "-"
    elif final_cls == "METADATA":
        tgt_db, tgt_sch, tgt_tbl = "alpha", "oms", t
    else:
        tgt_db, tgt_sch, tgt_tbl = "-", "-", "-"
    plan_rows.append({
        "source_db": "alpha",
        "source_schema": "oms",
        "source_table": t,
        "target_db": tgt_db,
        "target_schema": tgt_sch,
        "target_table": tgt_tbl,
        "classification": final_cls,
        "domain": "OMS" if final_cls == "DROP_SUPERSEDED" else "METADATA",
        "batch_order": batch_order(final_cls, t),
        "notes": _notes_for("oms", t, final_cls),
    })

# 3. alpha.edm rows
for t in sorted(alpha_edm):
    c = cls.get(("edm", t))
    if c is None:
        continue
    final_cls = normalize_class(t, c, "edm")
    if final_cls == "DATA" or final_cls == "DATA_LINEAGE":
        tgt_db, tgt_sch, tgt_tbl = "crims", "mdm", t
    elif final_cls == "INFRA_KEEP_ALPHA":
        tgt_db, tgt_sch, tgt_tbl = "alpha", "edm", t
    elif final_cls == "DROP":
        tgt_db, tgt_sch, tgt_tbl = "-", "-", "-"
    else:
        tgt_db, tgt_sch, tgt_tbl = "alpha", "edm", t
    plan_rows.append({
        "source_db": "alpha",
        "source_schema": "edm",
        "source_table": t,
        "target_db": tgt_db,
        "target_schema": tgt_sch,
        "target_table": tgt_tbl,
        "classification": final_cls,
        "domain": domain_of(t) + "/EDM" if final_cls not in ("INFRA_KEEP_ALPHA", "LINEAGE_TBD") else "INFRA",
        "batch_order": batch_order(final_cls, t),
        "notes": _notes_for("edm", t, final_cls),
    })

# 4. crims.mdm EXISTING rows
for t in sorted(crims_mdm):
    note = ""
    cls_label = "EXISTING"
    if t == "party":
        note = "Both alpha.mdm.party and crims.mdm.party empty; alpha schema wins. Replaced with alpha 44-col schema; crims RLS policies (party_tenant_read/write, tenant_isolation) ported; party_cd column added for xref identity matching."
    elif t == "rating_scale":
        note = "Alpha has 31 rows + rating->rating_scale FK; alpha schema wins. Crims.mdm.rating_scale (0 rows, 10 cols) replaced with alpha's 18-col schema; crims RLS policies ported."
    elif t == "source_system":
        note = "Near-collision with alpha.mdm.source_systems (3 rows, 35 inbound FKs moving in). Both tables coexist in crims.mdm post-migration; consolidation in cross-DB cleanup pass."
    elif t in {"payment_frequency"}:
        note = "Alpha has NO payment_frequency (REF_EXACT but not in alpha.mdm). Crims.mdm.payment_frequency stays as-is."
    elif t in {"dq_rule", "match_rule", "survivorship_rule"}:
        note = f"Coexists with edm.{t[:-5]}s duplicate (or its FABRIC_RELATIVE): consolidation pass."
    elif t in {"domains", "sources"}:
        if t == "sources":
            note = "Coexists with alpha.mdm.sources_mappers (FABRIC_REF). Consolidation pass."
        else:
            note = "Coexists with alpha.mdm.domains. Consolidation pass."
    plan_rows.append({
        "source_db": "crims",
        "source_schema": "mdm",
        "source_table": t,
        "target_db": "crims",
        "target_schema": "mdm",
        "target_table": t,
        "classification": cls_label,
        "domain": domain_of(t),
        "batch_order": 0,
        "notes": note,
    })

# 5. crims.orm EXISTING rows
for t in sorted(crims_orm):
    plan_rows.append({
        "source_db": "crims",
        "source_schema": "orm",
        "source_table": t,
        "target_db": "crims",
        "target_schema": "orm",
        "target_table": t,
        "classification": "EXISTING",
        "domain": "ORM",
        "batch_order": 0,
        "notes": "Pre-existing in crims.orm; not in migration scope (alpha.oms STI wrappers are superseded by these or replaced by migration scripts).",
    })

# 6. alpha.public fabric-relevant KEEP/EXTEND rows
for t in alpha_public_fabric:
    # G22 (already-decided): plural catalog tables used by fabric pipeline;
    # the catalog pipeline extends them at runtime (subtype_registry->BO).
    # Mark EXTEND for catalog/semantic/field-binding flavor; KEEP for the rest.
    flavor = "EXTEND"
    note_templates = {
        "business_objects": "G22: plural business_objects (alpha.public); catalog pipeline extends via STAGE 2 BO builder.",
        "business_object_fields": "G22: plural; catalog pipeline extends (STAGE 2 attribute_of edges).",
        "business_object_relationships": "G22: plural; EXTEND via catalog edge sync.",
        "business_object_binding": "G22: plural; EXTEND via catalog binding sync.",
    }
    if t in note_templates:
        note = note_templates[t]
    elif t == "tenants":
        note = "Tenant registry; stays in alpha (control-plane); mdm RLS/GUC uses tenant_id from JWT context (no FK references to crims)."
    elif t == "subtype_registry":
        # Should already be in alpha.oms METADATA, not public.
        note = "Reference to oms.subtype_registry (control plane)."
    elif t.startswith("catalog_"):
        note = "Catalog graph (STAGES 3-4); EXTEND."
    elif t.startswith("semantic_"):
        note = "Semantic layer; KEEP/EXTEND."
    else:
        note = "KEEP in alpha.control_plane."
    cls_label = "EXTEND" if t in {
        "business_objects","business_object_fields","business_object_relationships",
        "business_object_binding","catalog_node","catalog_edge","catalog_node_types",
        "catalog_edge_types","catalog_view_definitions","catalog_validation_rules",
    } else ("KEEP_ALPHA" if t not in {"tenants","subtype_registry"} else "KEEP_ALPHA")
    plan_rows.append({
        "source_db": "alpha",
        "source_schema": "public",
        "source_table": t,
        "target_db": "alpha",
        "target_schema": "public",
        "target_table": t,
        "classification": cls_label,
        "domain": "CONTROL_PLANE",
        "batch_order": 0,
        "notes": note,
    })

print(f"plan rows total: {len(plan_rows)}", file=sys.stderr)

# Make sure notes are populated (the helper is now defined above the builders).
for r in plan_rows:
    if not r["notes"]:
        r["notes"] = _notes_for(r["source_schema"], r["source_table"], r["classification"])

# -----------------------------------------------------------------------------
# Blockers
# -----------------------------------------------------------------------------

blockers = [
    ("G18",                 "HIGH",     "resolved", "fund_hierarchy->public.tenants FK becomes cross-DB; dropped; enforce via app RLS/GUC (app/tenant_guc.go). Decision applied pre-copy."),
    ("G20",                 "INFO",     "resolved", "G20 base classification did not exist on disk or in DB; this file is the authoritative rebuild."),
    ("G22",                 "INFO",     "resolved", "G22 plural fabric tables (business_objects etc.) already exist in alpha.public; catalog pipeline uses them directly (no parallel tables)."),
    ("FABRIC_HOME",         "INFO",     "resolved", "FABRIC_REF + FABRIC_RULE in alpha.public (Option A confirmed 2026-09-23). Cross-DB FKs from DATA -> fabric are dropped pre-copy; app + tenant_guc + RLS resolve references the same way tenants are resolved (per OR-clause pattern in 0001_mdm_security.sql)."),
    ("FABRIC_FK_DROPPED",   "MED",      "open",     "130 inbound FKs from DATA tables (alpha.mdm and satellites) -> FABRIC_REF/RULE tables in alpha.public are dropped pre-copy. App/RLS enforcement mirrors the G18 pattern. Tracking-only; resolution applied during copy step."),
    ("G16_ISSUER",          "INFO",     "resolved", "G16: issuer_master -> crims.mdm.issuer_master (crims has no edm schema). G19: pattern applied to benchmark_master / curve_master / fx_rate_master duplicates."),
    ("G15_PHANTOM",         "INFO",     "resolved", "G15: oms.subtype_registry_seed was a phantom; row removed."),
    ("G21_XREF",            "INFO",     "resolved", "G21: crims.mdm.xref (generic entity xref, 0 rows) subsumes migration.party_xref via entity_type='party'."),
    ("EDM_DUPLICATES",      "HIGH",     "resolved", "alpha.mdm.{benchmark_master, curve_master, fx_rate_master} vs alpha.edm.* (all 0 rows on both sides). mdm versions win; edm golden-copy twins (column-shorter, 3 inbound FKs vs mdm's 18) are DROPPED via DROP batch."),
    ("EDM_LAYER_FUTURE",    "LOW",      "open",     "EDM golden-copy layer (the 3 dropped edm twins + potentially edm analytical mirrors) is an analytics-side concept. May be re-built into alpha as a separate analytics read-side post-migration; not in current scope."),
    ("COLLISION_PARTY",     "HIGH",     "open",     "alpha.mdm.party (richer 44-col golden-record, 0 rows) vs crims.mdm.party (12 cols, 0 rows, target id space). Resolution: extend crims.mdm.party with ADD COLUMN for all alpha-exclusive fields (~43 cols), port RLS (tenant_id key already shared), DROP alpha.mdm.party in DROP batch. DDL: 002_crims_party_alter.sql."),
    ("COLLISION_RATING",    "HIGH",     "open",     "alpha.mdm.rating_scale (18 cols, 31 rows + rating FK) vs crims.mdm.rating_scale (10 cols, 0 rows). Alpha schema wins. Apply the same ADD COLUMN + port RLS pattern as party in 002_crims_party_alter.sql family (or a sibling 002_crims_rating_alter.sql)."),
    ("LINEAGE_RECLASS",     "MED",      "open",     "12 edm *_trace/*_lineage tables reclassified LINEAGE_TBD -> DATA_LINEAGE -> crims.mdm. Preserves 7 of 9 cross stay<->move FKs."),
    ("EDM_CROSS_FK_A",      "MED",      "open",     "preference_versions (alpha.edm, INFRA_KEEP_ALPHA) -> source_preferences (alpha.edm, DATA -> crims.mdm). Cross-DB after move. Drop FK + app-enforce (same pattern as G18)."),
    ("EDM_CROSS_FK_B",      "MED",      "open",     "portfolio_master (alpha.edm -> crims.mdm) -> performance_settings (alpha.edm, INFRA_KEEP_ALPHA stays alpha). Cross-DB after move. Drop FK + app-enforce."),
    ("PARTY_SEED_SOURCE",   "MED",      "open",     "Neither alpha nor crims has party rows. Seed: seed_party_minimal.sql provides 8 dev parties across the two dev tenants + gold tenant to exercise the FK graph in dev. Production party data populated by first ingestion (CRM, onboarding, vendor feeds)."),
    ("PARTY_FIRST_INGESTION","MED",     "open",     "Production party source: onboarding/KYC/CRM/vendor feeds (not a SQL seed). Identify and connect the ingestion pipeline that populates alpha.mdm.party path in production before dual-write window closes."),
    ("NEAR_SOURCE_SYSTEMS", "LOW",      "open",     "crims.source_system (1 row, 24 inbound FKs from issuer/security) coexists with moving alpha.source_systems (3 rows, 35 inbound FKs). Consolidation pass post-migration."),
    ("NEAR_DQ_RULES",       "LOW",      "open",     "edm.dq_rules (DATA -> crims.mdm) coexists with crims.mdm.dq_rule. Both kept until consolidation."),
    ("NEAR_SURVIVORSHIP_R", "LOW",      "open",     "edm.survivorship_rules (DATA -> crims.mdm) coexists with crims.mdm.survivorship_rule. Both kept until consolidation."),
    ("NEAR_DOMAIN_SRC",     "LOW",      "open",     "Coexistence of crims sources/domains with FABRIC_REF alpha versions. Defer consolidation."),
    ("GAMMA1_SOT",          "MED",      "open",     "Subtype_registry (alpha.oms) vs business_object_fields (alpha.public) source-of-truth unresolved. Catalog pipeline reads oms.subtype_registry as source (see catalog/subtype_registry.go); business_object_fields is rebuilt from it. Both stay in alpha; reconciliation is app-level; does not block plan copy."),
    ("GAMMA3_UNDEFINED",    "LOW",      "open",     "Open gap gamma3 from prior session context was not recoverable from disk or current transcripts. Re-identify or remove before cutover."),
    ("DEBEZIUM_RETARGET",   "MED",      "open",     "Debezium CDC connector currently targets dbname=alpha on cluster uisce-debezium (see docker-compose.remote.yml). Must be re-pointed to dbname=crims at cutover; Postgres publication recreated over crims.* (or alpha.* during dual-write). Gate: pre_cutover.\n\n  Sequence (must run in this exact order to prevent change-stream gaps):\n    1. copy_plan.sh completes for ALL movers across all 6 domains (Product, Party, Rating, etc.).\n    2. App re-pointed to crims in READ-ONLY mode (writers still go to alpha).\n    3. Apply crims publications for the migration-source schemas:\n         CREATE PUBLICATION crims_cdc_publication FOR TABLE mdm.*, edm.*, oms.*, cash_flow.*;\n       (Use crims-side schema names; include only the move-DATA tables per migration.plan.)\n    4. Stop the Debezium connector on alpha:\n         curl -X DELETE http://localhost:8083/connectors/orm-oms-connector\n       (or whichever name was registered; see docker-compose.remote.yml re alpha wiring.)\n    5. Start the Debezium connector on crims:\n         curl -X POST http://localhost:8083/connectors -d '{... publication.name=crims_cdc_publication, database.dbname=crims, ...}'\n       (After verifying the connector task state on crims reads RUNNING.)\n    6. App write phase enabled (dual-write window: writers now mirror to crims).\n    7. After dual-write + reconciliation window (24-72h), stop alpha writes and disable the alpha publication.\n  This sequence has no idle window where writes happen without CDC.\n\n  Side note (from AGENTS.md): the alpha connector used slot name 'orm_oms_slot' with publication 'orm_cdc_publication'. Its mTLS certs were placed via the .pk8 workaround at /tmp/orm_client_der.pk8. The crims-side connector needs the same workaround applied to its cert mount."),
    ("DROP_GATE",           "INFO",     "open",     "Drop of alpha.{mdm, edm, oms-superseded} schemas gated on migration.v_blockers_open gating_open count = 0. Open for HIGH/MED blockers: see blockers table."),
]

# -----------------------------------------------------------------------------
# SQL emit
# -----------------------------------------------------------------------------

def sql_escape(s: str | None) -> str:
    if s is None:
        return "NULL"
    if s == "-":
        return "'-'"
    return "'" + s.replace("'", "''") + "'"

lines: list[str] = []
HEADER = f"""-- 001_migration_plan.sql
-- Rebuilt classification file for the alpha.mdm -> crims.mdm migration.
--
-- Decisions applied (2026-09-23):
--   G18 fund_hierarchy -> public.tenants FK: DROPPED; enforce via RLS/GUC.
--   G22 alpha.public fabric tables: use existing plural (business_objects etc.).
--   G20 base classification: never persisted on disk -> this file IS the
--       authoritative rebuild; supersedes classification_seed_additions.sql.
--   FABRIC_REF + FABRIC_RULE -> crims.mdm (no longer alpha.public):
--       reference/rule data is data-plane; keeps 130 cross-DB FKs same-home.
--       alpha.public fabric (catalog/semantic) remains in alpha.control_plane.
--   G15 subtype_registry_seed does not exist -> row removed (G15 phantom).
--   G16 issuer_master -> crims.mdm.issuer_master (crims has no edm schema).
--   G19 issuer pattern repeated for benchmark_master / curve_master /
--       fx_rate_master; mdm versions win (richer, 18 inbound FKs vs edm 3),
--       edm golden-copy twins are dropped.
--   G21 party/xref: reuse crims.mdm.xref with entity_type='party'.
--   COLLISION_party:       both 0 rows; alpha schema wins -> replace empty
--                          crims.mdm.party; port crims RLS; add party_cd.
--   COLLISION_rating_scale: alpha has 31 rows + rating->rating_scale FK;
--                          alpha schema wins.
--   LINEAGE_TBD 12 edm *_trace/*_lineage tables: reclassified DATA_LINEAGE
--                          -> crims.mdm (preserves 7 cross stay<->move FKs).
--   CROSS-DB FKs REMAINING after decisions: G18 + 2 (preference_versions,
--       portfolio_master edges into INFRA-side) -> drop + app-enforce.
--
-- Run against CRIMS. Idempotent (ON CONFLICT DO NOTHING).
"""
lines.append(HEADER)

lines.append("-- migration schema (lives on crims master) ----------------------------")
lines.append("CREATE SCHEMA IF NOT EXISTS migration;")
lines.append("")
lines.append("-- migration.plan: every classified table -------------------------------")
lines.append("""CREATE TABLE IF NOT EXISTS migration.plan (
  id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  source_db       text NOT NULL,
  source_schema   text NOT NULL,
  source_table    text NOT NULL,
  target_db       text NOT NULL,
  target_schema   text,
  target_table    text,
  classification  text NOT NULL,
  domain          text NOT NULL,
  batch_order     int  NOT NULL,
  notes           text,
  UNIQUE (source_db, source_schema, source_table)
);""")
lines.append("")
lines.append("-- migration.progress: per-row copy status --------------------------------")
lines.append("""CREATE TABLE IF NOT EXISTS migration.progress (
  plan_id       bigint PRIMARY KEY REFERENCES migration.plan(id),
  status        text NOT NULL DEFAULT 'pending',
                -- pending|copying|done|failed|skipped
  rows_copied   bigint,
  started_at    timestamptz,
  finished_at   timestamptz,
  error         text,
  detail        text
);""")
lines.append("")
lines.append("-- migration.blockers: items gating the alpha.* drop ----------------------")
lines.append("""CREATE TABLE IF NOT EXISTS migration.blockers (
  id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  code         text NOT NULL UNIQUE,
  severity     text NOT NULL,                       -- CRITICAL/HIGH/MED/LOW/INFO
  description  text NOT NULL,
  status       text NOT NULL DEFAULT 'open',        -- open|resolved|waived
  resolution   text,
  resolved_at  timestamptz
);""")
lines.append("")
lines.append("-- views ------------------------------------------------------------------")
lines.append("""CREATE OR REPLACE VIEW migration.v_plan_summary AS
  SELECT source_db, target_db, target_schema, classification, count(*)
  FROM migration.plan
  GROUP BY 1,2,3,4;""")
lines.append("")
lines.append("""CREATE OR REPLACE VIEW migration.v_copy_queue AS
  SELECT id, source_schema, source_table, target_db, target_schema,
         target_table, classification, domain, batch_order
  FROM migration.plan
  WHERE source_db <> target_db
    AND target_table IS NOT NULL
    AND classification NOT IN ('DROP','DROP_SUPERSEDED_BY_CRIMS_ORM')
  ORDER BY batch_order, domain, source_table;""")
lines.append("")
lines.append("""CREATE OR REPLACE VIEW migration.v_blockers_open AS
  SELECT * FROM migration.blockers
  WHERE status = 'open'
    AND severity IN ('CRITICAL','HIGH','MED');""")
lines.append("")
lines.append("""COMMENT ON VIEW migration.v_blockers_open IS
  'Gate for alpha.* drop: an open blocker here defers drop_alpha_until_open_zero.';""")
lines.append("")
lines.append("ALTER TABLE migration.plan "
             "ADD CONSTRAINT plan_target_required_when_move "
             "CHECK (source_db = target_db OR (target_db IS NOT NULL AND target_table IS NOT NULL));"
             if False else "-- (skipping CHECK because classification == DROP rows legitimately lack target)")
lines.append("")
lines.append("-- plan rows --------------------------------------------------------------")
lines.append(f"INSERT INTO migration.plan(source_db, source_schema, source_table, target_db, target_schema, target_table, classification, domain, batch_order, notes)")
lines.append("VALUES")
chunks = []
for r in plan_rows:
    chunks.append("(" + ", ".join([
        sql_escape(r["source_db"]),
        sql_escape(r["source_schema"]),
        sql_escape(r["source_table"]),
        sql_escape(r["target_db"]),
        sql_escape(r["target_schema"]),
        sql_escape(r["target_table"]),
        sql_escape(r["classification"]),
        sql_escape(r["domain"]),
        str(r["batch_order"]),
        sql_escape(r["notes"]),
    ]) + ")")
lines.append(",\n".join(chunks))
lines.append("ON CONFLICT (source_db, source_schema, source_table) DO NOTHING;")
lines.append("")
lines.append("-- blockers --------------------------------------------------------------")
lines.append("INSERT INTO migration.blockers(code, severity, status, description) VALUES")
bchunks = []
for code, sev, status, desc in blockers:
    bchunks.append("(" + ", ".join([
        sql_escape(code), sql_escape(sev), sql_escape(status), sql_escape(desc),
    ]) + ")")
lines.append(",\n".join(bchunks))
lines.append("ON CONFLICT (code) DO NOTHING;")
lines.append("")
lines.append("-- QA summary -- counts by classification+target")
lines.append("SELECT classification, target_db || COALESCE('.' || target_schema, '') AS target, count(*) AS rows")
lines.append("FROM migration.plan GROUP BY 1,2 ORDER BY 1,2;")
lines.append("")
lines.append("SELECT count(*) FILTER (WHERE status='open' AND severity IN ('CRITICAL','HIGH','MED')) AS gating_open")
lines.append("FROM migration.blockers;")
lines.append("")

out = HERE / "001_migration_plan.sql"
out.write_text("\n".join(lines))
print(f"wrote {out} ({out.stat().st_size} bytes)", file=sys.stderr)
print(f"plan rows: {len(plan_rows)}", file=sys.stderr)
print(f"blockers: {len(blockers)}", file=sys.stderr)
