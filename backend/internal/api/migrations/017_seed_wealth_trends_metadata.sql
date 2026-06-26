-- Seed Wealth Management Trends Metadata
-- Implements the "Metadata-First" strategy for 8 key wealth trends

-- 1. Hyper-Personalization: Client Profile Extensions
-- Note: We extend the existing 'bo_client' or create a new 'bo_client_profile' linked to it.
-- Let's create a separate profile object for modularity.
INSERT INTO meta_objects (id, name, version_major, version_minor, version_patch, status, payload)
VALUES ('bo_client_profile', 'Client Profile', 1, 0, 0, 'active', '{
  "meta": {
    "id": "bo_client_profile",
    "name": "Client Profile",
    "version": {"major": 1, "minor": 0, "patch": 0},
    "status": "active",
    "valid_from": "2025-01-01T00:00:00Z"
  },
  "attributes": [
    {"name": "esg_preference_score", "type": "decimal", "required": false, "description": "0-100 score for ESG focus"},
    {"name": "risk_capacity", "type": "string", "required": true, "default": "MODERATE"},
    {"name": "liquidity_needs", "type": "string", "required": false},
    {"name": "tax_bracket", "type": "string", "required": false}
  ],
  "rels": [
    {"name": "client", "target_bo": "bo_client", "cardinality": "one", "on_delete": "cascade"}
  ],
  "lifecycle": ["ACTIVE", "ARCHIVED"],
  "policies": ["VIEW_SENSITIVE_DATA"]
}');

-- 2. Alternative Investments: Private Asset
INSERT INTO meta_objects (id, name, version_major, version_minor, version_patch, status, payload)
VALUES ('bo_private_asset', 'Private Asset', 1, 0, 0, 'active', '{
  "meta": {
    "id": "bo_private_asset",
    "name": "Private Asset",
    "version": {"major": 1, "minor": 0, "patch": 0},
    "status": "active",
    "valid_from": "2025-01-01T00:00:00Z"
  },
  "attributes": [
    {"name": "name", "type": "string", "required": true},
    {"name": "asset_class", "type": "string", "required": true, "default": "PRIVATE_EQUITY"},
    {"name": "valuation", "type": "decimal", "required": true},
    {"name": "valuation_date", "type": "date", "required": true},
    {"name": "lock_up_period_months", "type": "integer", "required": false},
    {"name": "vintage_year", "type": "integer", "required": false}
  ],
  "rels": [],
  "lifecycle": ["ACTIVE", "LIQUIDATED", "WRITTEN_OFF"],
  "policies": ["VIEW_PRIVATE_MARKETS"]
}');

-- 3. Automated Rebalancing: Rebalancing Schedule
INSERT INTO meta_objects (id, name, version_major, version_minor, version_patch, status, payload)
VALUES ('bo_rebalancing_schedule', 'Rebalancing Schedule', 1, 0, 0, 'active', '{
  "meta": {
    "id": "bo_rebalancing_schedule",
    "name": "Rebalancing Schedule",
    "version": {"major": 1, "minor": 0, "patch": 0},
    "status": "active",
    "valid_from": "2025-01-01T00:00:00Z"
  },
  "attributes": [
    {"name": "frequency", "type": "string", "required": true, "default": "QUARTERLY"},
    {"name": "drift_threshold_percent", "type": "decimal", "required": true, "default": 5.0},
    {"name": "last_run_date", "type": "date", "required": false},
    {"name": "next_run_date", "type": "date", "required": true},
    {"name": "auto_execute", "type": "boolean", "required": true, "default": false}
  ],
  "rels": [
    {"name": "portfolio", "target_bo": "bo_portfolio", "cardinality": "one", "on_delete": "cascade"}
  ],
  "lifecycle": ["ACTIVE", "PAUSED", "TERMINATED"],
  "policies": ["MANAGE_REBALANCING"]
}');

-- 4. Subscription Services: Subscription Plan
INSERT INTO meta_objects (id, name, version_major, version_minor, version_patch, status, payload)
VALUES ('bo_subscription_plan', 'Subscription Plan', 1, 0, 0, 'active', '{
  "meta": {
    "id": "bo_subscription_plan",
    "name": "Subscription Plan",
    "version": {"major": 1, "minor": 0, "patch": 0},
    "status": "active",
    "valid_from": "2025-01-01T00:00:00Z"
  },
  "attributes": [
    {"name": "plan_name", "type": "string", "required": true},
    {"name": "monthly_fee", "type": "decimal", "required": true},
    {"name": "annual_fee", "type": "decimal", "required": true},
    {"name": "features", "type": "jsonb", "required": false},
    {"name": "is_active", "type": "boolean", "required": true, "default": true}
  ],
  "rels": [],
  "lifecycle": ["DRAFT", "ACTIVE", "RETIRED"],
  "policies": ["MANAGE_SUBSCRIPTIONS"]
}');

-- 5. Metrics: Net P&L (Example Metric Definition)
INSERT INTO meta_metrics (id, name, version_major, version_minor, version_patch, status, payload)
VALUES ('metric_net_pnl', 'Net Profit & Loss', 1, 0, 0, 'active', '{
  "meta": {
    "id": "metric_net_pnl",
    "name": "Net Profit & Loss",
    "version": {"major": 1, "minor": 0, "patch": 0},
    "status": "active",
    "valid_from": "2025-01-01T00:00:00Z"
  },
  "definition": {
    "formula": "realized_pnl + unrealized_pnl - fees",
    "grain": ["portfolio_id", "date"],
    "format": "currency",
    "unit": "USD"
  },
  "source": {
    "type": "derived",
    "inputs": ["metric_realized_pnl", "metric_unrealized_pnl", "metric_fees"]
  },
  "policies": ["VIEW_PERFORMANCE"]
}');
