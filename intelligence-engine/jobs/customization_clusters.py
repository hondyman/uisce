"""
Uisce Customization Intelligence ETL
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
Reads CREATE events from the Postgres audit_logs table, clusters similar custom
roles and policies using TF-IDF + MiniBatchKMeans, and upserts recommendations
to the fact_customization_telemetry table.

Usage:
    python -m jobs.customization_clusters [--lookback-days 90] [--min-tenants 5] [--k 20]
    python -m jobs.customization_clusters --dsn "postgresql://user:pass@host/db"

Environment:
    DATABASE_URL   – primary Postgres DSN (defaults to localhost)
"""

from __future__ import annotations

import argparse
import hashlib
import logging
import os
import sys
from datetime import datetime, timedelta, timezone
from typing import Any

import numpy as np
import pandas as pd
from sklearn.cluster import MiniBatchKMeans
from sklearn.feature_extraction.text import TfidfVectorizer

try:
    import psycopg
    import psycopg.rows
except ImportError:
    sys.exit("psycopg[binary] is required: pip install psycopg[binary]")

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
)
log = logging.getLogger("customization_clusters")

# ── SQL ─────────────────────────────────────────────────────────────────────────

FETCH_AUDIT_SQL = """
SELECT
    al.tenant_id,
    al.entity_id,
    al.action,
    al.entity_type,
    COALESCE(al.details->>'role_name', al.details->>'policy_name', 'Unknown') AS name,
    COALESCE(al.details->>'description', '') AS description,
    COALESCE(al.details->>'permissions', '[]') AS permissions,
    al.created_at
FROM audit_logs al
WHERE
    -- Normalise action casing with .upper() for safety
    upper(al.action) = 'CREATE'
    AND al.entity_type IN ('bp_roles', 'bp_dynamic_policies')
    AND al.created_at >= NOW() - INTERVAL '%d days'
ORDER BY al.created_at DESC;
"""

UPSERT_TELEMETRY_SQL = """
INSERT INTO fact_customization_telemetry
    (id, cluster_id, pattern_hash, entity_type, sample_name,
     tenant_count, recommended_for_core, confidence_score, detected_at)
VALUES
    (%s, %s, %s, %s, %s, %s, %s, %s, %s)
ON CONFLICT (cluster_id, pattern_hash) DO UPDATE SET
    sample_name          = EXCLUDED.sample_name,
    tenant_count        = EXCLUDED.tenant_count,
    confidence_score    = EXCLUDED.confidence_score,
    detected_at          = EXCLUDED.detected_at,
    recommended_for_core = EXCLUDED.recommended_for_core;
"""

# ── Helpers ─────────────────────────────────────────────────────────────────────

def get_dsn() -> str:
    return os.environ.get(
        "DATABASE_URL",
        "postgresql://postgres:postgres@localhost:5432/uisce",
    )


def fetch_audit_logs(dsn: str, lookback_days: int) -> pd.DataFrame:
    log.info("Fetching audit logs (lookback=%d days) …", lookback_days)
    with psycopg.connect(dsn, row_factory=psycopg.rows.dict_row) as conn:
        df = pd.read_sql_query(
            FETCH_AUDIT_SQL % lookback_days,
            conn,
            params={"lookback_days": lookback_days},
        )
    log.info("Fetched %d audit rows", len(df))
    return df


def normalize_text_fields(df: pd.DataFrame) -> pd.DataFrame:
    df = df.copy()
    df["name"] = df["name"].fillna("").str.strip().str.lower()
    df["description"] = df["description"].fillna("").str.strip().str.lower()
    # Permissions are stored as JSON string; extract a normalised text blob.
    df["permissions_text"] = df["permissions"].apply(_permissions_to_text)
    df["combined_text"] = (
        df["name"] + " " + df["description"] + " " + df["permissions_text"]
    )
    return df


def _permissions_to_text(val: Any) -> str:
    try:
        import json
        perms = json.loads(val) if isinstance(val, str) else val
        return " ".join(sorted(p.lower() for p in perms if isinstance(p, str)))
    except Exception:
        return ""


def build_clusters(df: pd.DataFrame, k: int) -> pd.DataFrame:
    if len(df) < 2:
        log.warning("Too few rows to cluster; returning empty result")
        return pd.DataFrame(
            columns=["cluster_id", "pattern_hash", "entity_type",
                     "sample_name", "tenant_count", "recommended_for_core",
                     "confidence_score"]
        )

    vectorizer = TfidfVectorizer(
        ngram_range=(1, 2),
        lowercase=True,
        sublinear_tf=True,
        max_features=512,
        min_df=1,
    )
    X = vectorizer.fit_transform(df["combined_text"])

    actual_k = min(k, len(df))
    model = MiniBatchKMeans(n_clusters=actual_k, random_state=42, batch_size=1024)
    df = df.copy()
    df["cluster_id"] = model.fit_predict(X).astype(str)

    # Name each cluster by its most-frequent role name token (first word).
    cluster_names: dict[str, str] = {}
    for cid, grp in df.groupby("cluster_id"):
        top = (
            grp["name"]
            .str.split()
            .explode()
            .value_counts()
            .idxmax()
        )
        cluster_names[str(cid)] = top or f"cluster_{cid}"

    df["cluster_name"] = df["cluster_id"].map(cluster_names)
    return df


def compute_pattern_hash(name: str, description: str, permissions_text: str) -> str:
    payload = f"{name}|{description}|{permissions_text}"
    return hashlib.sha256(payload.encode()).hexdigest()[:32]


def build_recommendations(
    df: pd.DataFrame,
    min_tenants: int,
) -> pd.DataFrame:
    log.info("Building recommendations (min_tenants=%d) …", min_tenants)

    # Aggregate per cluster per tenant to avoid double-counting one tenant's roles.
    tenant_clusters = (
        df.groupby(["cluster_id", "cluster_name", "tenant_id"])
        .agg(
            entity_type=("entity_type", "first"),
            name=("name", "first"),
            description=("description", "first"),
            permissions_text=("permissions_text", "first"),
        )
        .reset_index()
    )

    # Count distinct tenants per cluster.
    cluster_counts = (
        tenant_clusters.groupby(["cluster_id", "cluster_name"])
        .agg(
            tenant_count=("tenant_id", "nunique"),
            entity_type=("entity_type", "first"),
            sample_name=("name", lambda x: x.value_counts().idxmax()),
            pattern_hash=(
                ["name", "description", "permissions_text"],
                lambda col: compute_pattern_hash(
                    col.iloc[0], col.iloc[1], col.iloc[2]
                ),
            ),
        )
        .reset_index()
    )

    recs = cluster_counts[cluster_counts["tenant_count"] >= min_tenants].copy()
    recs["recommended_for_core"] = True
    # Confidence: simple heuristic — higher tenant count → higher confidence, capped at 0.99.
    recs["confidence_score"] = (recs["tenant_count"] / (recs["tenant_count"] + 5)).round(2)
    recs["pattern_hash"] = recs["pattern_hash"].astype(str)
    recs["entity_type"] = recs["entity_type"].astype(str)
    recs["sample_name"] = recs["sample_name"].astype(str)
    recs["detected_at"] = datetime.now(timezone.utc)

    log.info(
        "Generated %d recommendations (from %d total clusters)",
        len(recs),
        len(cluster_counts),
    )
    return recs


def upsert_telemetry(dsn: str, recs: pd.DataFrame) -> int:
    if recs.empty:
        log.info("No recommendations to upsert")
        return 0

    log.info("Upserting %d rows to fact_customization_telemetry …", len(recs))
    with psycopg.connect(dsn, row_factory=psycopg.rows.dict_row) as conn:
        with conn.cursor() as cur:
            for _, row in recs.iterrows():
                cur.execute(
                    UPSERT_TELEMETRY_SQL,
                    (
                        str(row["cluster_id"]),
                        str(row["cluster_id"]),
                        row["pattern_hash"],
                        row["entity_type"],
                        row["sample_name"],
                        int(row["tenant_count"]),
                        row["recommended_for_core"],
                        float(row["confidence_score"]),
                        row["detected_at"],
                    ),
                )
    log.info("Upsert complete")
    return len(recs)


# ── Main ──────────────────────────────────────────────────────────────────────

def run(
    *,
    dsn: str | None = None,
    lookback_days: int = 90,
    min_tenants: int = 5,
    k: int = 20,
) -> int:
    dsn = dsn or get_dsn()
    log.info("Customization Intelligence ETL starting")
    log.info("  lookback_days=%d  min_tenants=%d  k=%d", lookback_days, min_tenants, k)

    try:
        df = fetch_audit_logs(dsn, lookback_days)
    except Exception as exc:
        log.error("Failed to fetch audit logs: %s", exc)
        return 1

    if df.empty:
        log.info("No CREATE events found; nothing to do")
        return 0

    df = normalize_text_fields(df)
    df = build_clusters(df, k=k)
    recs = build_recommendations(df, min_tenants=min_tenants)
    count = upsert_telemetry(dsn, recs)

    log.info(
        "ETL complete. %d recommendations written to fact_customization_telemetry", count
    )
    return 0


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Uisce Customization Intelligence ETL"
    )
    parser.add_argument(
        "--dsn",
        dest="dsn",
        metavar="DSN",
        default=None,
        help="Postgres DSN (default: from DATABASE_URL env var)",
    )
    parser.add_argument(
        "--lookback-days",
        dest="lookback_days",
        type=int,
        default=90,
        metavar="N",
        help="How many days of audit history to analyse (default: 90)",
    )
    parser.add_argument(
        "--min-tenants",
        dest="min_tenants",
        type=int,
        default=5,
        metavar="N",
        help="Minimum distinct tenants for a cluster to be recommended (default: 5)",
    )
    parser.add_argument(
        "--k",
        type=int,
        default=20,
        metavar="N",
        help="Maximum number of clusters (default: 20)",
    )
    args = parser.parse_args()
    code = run(
        dsn=args.dsn,
        lookback_days=args.lookback_days,
        min_tenants=args.min_tenants,
        k=args.k,
    )
    sys.exit(code)


if __name__ == "__main__":
    main()
