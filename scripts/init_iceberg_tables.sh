#!/bin/bash
set -e

ICEBERG_REST_URL="${ICEBERG_REST_URL:-http://localhost:8181}"

echo "Initializing Iceberg default namespace and uisce_global_audit table..."

# 1. Create default namespace
curl -s -X POST -H "Content-Type: application/json" "${ICEBERG_REST_URL}/v1/namespaces" \
  -d '{"namespace": ["default"]}' || true

# 2. Pre-create uisce_global_audit schema in Iceberg REST catalog
curl -s -X POST -H "Content-Type: application/json" "${ICEBERG_REST_URL}/v1/namespaces/default/tables" \
  -d '{
    "name": "uisce_global_audit",
    "schema": {
      "type": "struct",
      "fields": [
        {"id": 1, "name": "event_id", "type": "string", "required": true},
        {"id": 2, "name": "tenant_id", "type": "string", "required": true},
        {"id": 3, "name": "tenant_instance_id", "type": "string", "required": false},
        {"id": 4, "name": "action", "type": "string", "required": true},
        {"id": 5, "name": "entity_type", "type": "string", "required": true},
        {"id": 6, "name": "entity_id", "type": "string", "required": true},
        {"id": 7, "name": "before_state", "type": "string", "required": false},
        {"id": 8, "name": "after_state", "type": "string", "required": false},
        {"id": 9, "name": "user_id", "type": "string", "required": true},
        {"id": 10, "name": "timestamp", "type": "timestamp", "required": true}
      ]
    }
  }' || true

echo "Iceberg catalog tables initialized successfully!"
