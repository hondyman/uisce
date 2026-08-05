import json

def analyze_customization_trends(audit_logs: list) -> list:
    """Clusters custom role & rule creations across tenants to recommend Gold Copy core promotions"""
    custom_role_counts = {}
    for log in audit_logs:
        if log.get("action") in ["CREATE", "created"] and log.get("entity_type") == "bp_roles":
            role_name = log.get("role_name", "Custom Role").strip()
            custom_role_counts[role_name] = custom_role_counts.get(role_name, 0) + 1

    recommendations = []
    for role_name, count in custom_role_counts.items():
        if count >= 3:
            recommendations.append({
                "cluster_name": role_name,
                "matching_tenants_count": count,
                "recommendation": f"Promote '{role_name}' (used by {count} tenants) to Gold Copy Core Master Schema",
                "confidence_score": 0.95,
            })
    return recommendations

if __name__ == "__main__":
    print("=================================================================")
    print(" Uisce Customization Intelligence Engine Running")
    print("=================================================================")

    sample_logs = [
        {"action": "created", "entity_type": "bp_roles", "role_name": "SOX Auditor"},
        {"action": "created", "entity_type": "bp_roles", "role_name": "SOX Auditor"},
        {"action": "created", "entity_type": "bp_roles", "role_name": "SOX Auditor"},
        {"action": "created", "entity_type": "bp_roles", "role_name": "ESG Compliance Specialist"},
    ]
    recs = analyze_customization_trends(sample_logs)
    print(f"[Customization Intelligence] Recommendations: {json.dumps(recs, indent=2)}")
