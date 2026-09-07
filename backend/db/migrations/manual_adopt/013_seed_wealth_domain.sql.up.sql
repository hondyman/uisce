-- Seed Wealth Management Business Objects into Metadata Registry

-- 1. Client Object
INSERT INTO meta_objects (id, name, version_major, version_minor, version_patch, status, payload)
VALUES ('bo_client', 'Client', 1, 0, 0, 'active', '{
  "meta": {
    "id": "bo_client",
    "name": "Client",
    "version": {"major": 1, "minor": 0, "patch": 0},
    "status": "active",
    "valid_from": "2025-01-01T00:00:00Z"
  },
  "attributes": [
    {"name": "client_code", "type": "string", "required": true},
    {"name": "first_name", "type": "string", "required": true},
    {"name": "last_name", "type": "string", "required": true},
    {"name": "date_of_birth", "type": "date", "required": false},
    {"name": "risk_tolerance", "type": "string", "required": true, "default": "MODERATE"},
    {"name": "net_worth", "type": "decimal", "required": false},
    {"name": "kyc_status", "type": "string", "required": true, "default": "PENDING_REVIEW"}
  ],
  "rels": [
    {"name": "advisor", "target_bo": "bo_user", "cardinality": "one", "on_delete": "set_null"}
  ],
  "lifecycle": ["ACTIVE", "INACTIVE", "SUSPENDED"],
  "policies": ["VIEW_ALL_CLIENTS", "MANAGE_CLIENTS"]
}');

-- 2. Portfolio Object
INSERT INTO meta_objects (id, name, version_major, version_minor, version_patch, status, payload)
VALUES ('bo_portfolio', 'Portfolio', 1, 0, 0, 'active', '{
  "meta": {
    "id": "bo_portfolio",
    "name": "Portfolio",
    "version": {"major": 1, "minor": 0, "patch": 0},
    "status": "active",
    "valid_from": "2025-01-01T00:00:00Z"
  },
  "attributes": [
    {"name": "name", "type": "string", "required": true},
    {"name": "description", "type": "string", "required": false},
    {"name": "portfolio_type", "type": "string", "required": true, "default": "INVESTMENT"},
    {"name": "base_currency", "type": "string", "required": true, "default": "USD"},
    {"name": "inception_date", "type": "date", "required": true}
  ],
  "rels": [
    {"name": "client", "target_bo": "bo_client", "cardinality": "one", "on_delete": "cascade"}
  ],
  "lifecycle": ["ACTIVE", "CLOSED"],
  "policies": ["VIEW_PORTFOLIOS", "MANAGE_PORTFOLIOS"]
}');

-- 3. Asset Object
INSERT INTO meta_objects (id, name, version_major, version_minor, version_patch, status, payload)
VALUES ('bo_asset', 'Asset', 1, 0, 0, 'active', '{
  "meta": {
    "id": "bo_asset",
    "name": "Asset",
    "version": {"major": 1, "minor": 0, "patch": 0},
    "status": "active",
    "valid_from": "2025-01-01T00:00:00Z"
  },
  "attributes": [
    {"name": "symbol", "type": "string", "required": true},
    {"name": "name", "type": "string", "required": true},
    {"name": "asset_type", "type": "string", "required": true},
    {"name": "exchange", "type": "string", "required": false},
    {"name": "isin", "type": "string", "required": false}
  ],
  "rels": [],
  "lifecycle": ["TRADEABLE", "SUSPENDED", "DELISTED"],
  "policies": ["MANAGE_MARKET_DATA"]
}');

-- 4. Transaction Object
INSERT INTO meta_objects (id, name, version_major, version_minor, version_patch, status, payload)
VALUES ('bo_transaction', 'Transaction', 1, 0, 0, 'active', '{
  "meta": {
    "id": "bo_transaction",
    "name": "Transaction",
    "version": {"major": 1, "minor": 0, "patch": 0},
    "status": "active",
    "valid_from": "2025-01-01T00:00:00Z"
  },
  "attributes": [
    {"name": "transaction_type", "type": "string", "required": true},
    {"name": "quantity", "type": "decimal", "required": true},
    {"name": "price_per_unit", "type": "decimal", "required": true},
    {"name": "total_amount", "type": "decimal", "required": true},
    {"name": "transaction_date", "type": "date", "required": true},
    {"name": "status", "type": "string", "required": true, "default": "PENDING"}
  ],
  "rels": [
    {"name": "portfolio", "target_bo": "bo_portfolio", "cardinality": "one", "on_delete": "cascade"},
    {"name": "asset", "target_bo": "bo_asset", "cardinality": "one", "on_delete": "restrict"}
  ],
  "lifecycle": ["PENDING", "COMPLETED", "FAILED", "CANCELLED"],
  "policies": ["VIEW_TRADES", "EXECUTE_TRADES"]
}');

-- 5. Order Object
INSERT INTO meta_objects (id, name, version_major, version_minor, version_patch, status, payload)
VALUES ('bo_order', 'Order', 1, 0, 0, 'active', '{
  "meta": {
    "id": "bo_order",
    "name": "Order",
    "version": {"major": 1, "minor": 0, "patch": 0},
    "status": "active",
    "valid_from": "2025-01-01T00:00:00Z"
  },
  "attributes": [
    {"name": "order_type", "type": "string", "required": true},
    {"name": "side", "type": "string", "required": true},
    {"name": "quantity", "type": "decimal", "required": true},
    {"name": "price", "type": "decimal", "required": false},
    {"name": "status", "type": "string", "required": true, "default": "PENDING"}
  ],
  "rels": [
    {"name": "portfolio", "target_bo": "bo_portfolio", "cardinality": "one", "on_delete": "cascade"},
    {"name": "asset", "target_bo": "bo_asset", "cardinality": "one", "on_delete": "restrict"}
  ],
  "lifecycle": ["PENDING", "FILLED", "CANCELLED", "REJECTED"],
  "policies": ["EXECUTE_TRADES"]
}');
