import os
import hashlib
import json
import time
from datetime import datetime, timezone

# Salt for cryptographic PII hashing
SALT = os.getenv("ANONYMIZATION_SALT", "uisce_secret_salt_2026")

def hash_pii(user_id: str) -> str:
    """Hashes PII user_id using SHA-256 with salt"""
    return hashlib.sha256(f"{SALT}:{user_id}".encode('utf-8')).hexdigest()[:16]

def map_tenant_to_vertical(tenant_id: str) -> str:
    """Maps tenant ID to an industry vertical"""
    verticals = ["finance", "healthcare", "tech", "asset_management"]
    idx = sum(ord(c) for c in tenant_id) % len(verticals)
    return verticals[idx]

def run_anonymization_etl(raw_records: list) -> list:
    """Flow 1: Strips PII, hashes IDs, and generalizes timestamps for Repo 4"""
    scrubbed = []
    for rec in raw_records:
        scrubbed.append({
            "hashed_user_id": hash_pii(rec.get("user_id", "unknown")),
            "industry_vertical": map_tenant_to_vertical(rec.get("tenant_id", "default")),
            "action": rec.get("action", "UPDATE"),
            "entity_type": rec.get("entity_type", "bp_teams"),
            "event_date": datetime.now(timezone.utc).strftime("%Y-%m-%d"),
        })
    print(f"[Repo 4 ETL] Scrubbed and anonymized {len(scrubbed)} records.")
    return scrubbed

def run_ai_sentiment_analysis(text: str) -> dict:
    """Flow 2: Simulates LLM sentiment and risk scoring"""
    text_lower = text.lower()
    if "unauthorized" in text_lower or "violation" in text_lower or "hacked" in text_lower:
        return {"sentiment_score": -0.85, "risk_level": "CRITICAL", "keywords": ["unauthorized", "violation"]}
    elif "update" in text_lower or "modify" in text_lower:
        return {"sentiment_score": 0.10, "risk_level": "LOW", "keywords": ["routine_update"]}
    else:
        return {"sentiment_score": 0.50, "risk_level": "LOW", "keywords": ["normal_operation"]}

def run_statistical_anomaly_detection(access_counts: list) -> list:
    """Flow 3: Calculates Z-scores on event frequencies to flag anomalies (Z > 3.0)"""
    if not access_counts:
        return []
    mean = sum(access_counts) / len(access_counts)
    std_dev = (sum((x - mean) ** 2 for x in access_counts) / len(access_counts)) ** 0.5
    if std_dev == 0:
        std_dev = 1.0

    alerts = []
    for count in access_counts:
        z_score = (count - mean) / std_dev
        if z_score > 3.0:
            alerts.append({
                "anomaly_type": "RBAC_ACCESS_SURGE",
                "z_score": round(z_score, 2),
                "summary": f"Event count {count} exceeded industry mean {mean:.1f} with Z-score {z_score:.2f}."
            })
    return alerts

if __name__ == "__main__":
    print("=================================================================")
    print(" Uisce Repo 4 Intelligence & AI Engine Initialized")
    print("=================================================================")
    
    sample_records = [
        {"user_id": "john.doe@bank.com", "tenant_id": "99e99e99-99e9-49e9-89e9-99e99e99e999", "action": "UPDATE", "entity_type": "bp_teams"},
        {"user_id": "alice.smith@hedge.com", "tenant_id": "11111111-2222-3333-4444-555555555555", "action": "DELETE", "entity_type": "bp_roles"},
    ]
    scrubbed = run_anonymization_etl(sample_records)
    sentiment = run_ai_sentiment_analysis("Unauthorized access to portfolio detected")
    alerts = run_statistical_anomaly_detection([10, 12, 11, 14, 13, 500])
    
    print(f"[AI Sentiment] Result: {json.dumps(sentiment)}")
    print(f"[AI Anomaly] Flagged Alerts: {json.dumps(alerts)}")
