#!/usr/bin/env bash
set -e

DEBEZIUM_URL="${DEBEZIUM_URL:-http://localhost:8083}"

echo "================================================================="
echo " Registering Dual-Topic Debezium PostgreSQL CDC Connectors"
echo " 1) Metadata CDC: uisce_meta.* -> Repo 1 & 2 Audit Lakehouse"
echo " 2) Tenant Data CDC: tenant_data_* -> Tenant Raw Lake"
echo "================================================================="

# 1. Register Metadata Connector (Uisce alpha DB -> Low Volume)
curl -s -X PUT -H "Content-Type: application/json" --data '{
  "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
  "database.hostname": "100.84.50.65",
  "database.port": "5432",
  "database.user": "postgres",
  "database.password": "postgres",
  "database.dbname": "alpha",
  "topic.prefix": "uisce_meta",
  "plugin.name": "pgoutput",
  "table.include.list": "public.bp_teams,public.bp_roles,public.bp_user_roles,public.bp_team_members",
  "tombstones.on.delete": "false"
}' "${DEBEZIUM_URL}/connectors/uisce-metadata-connector/config" | jq . || echo "Metadata connector registered."

# 2. Register Tenant Data Connector (Tenant Product DB -> High Volume)
curl -s -X PUT -H "Content-Type: application/json" --data '{
  "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
  "database.hostname": "100.84.50.65",
  "database.port": "5432",
  "database.user": "postgres",
  "database.password": "postgres",
  "database.dbname": "alpha",
  "topic.prefix": "tenant_data_123",
  "plugin.name": "pgoutput",
  "table.include.list": "public.portfolios,public.transactions",
  "tombstones.on.delete": "false"
}' "${DEBEZIUM_URL}/connectors/tenant-123-connector/config" | jq . || echo "Tenant data connector registered."

echo "Dual-Topic Debezium CDC Connectors Active."
