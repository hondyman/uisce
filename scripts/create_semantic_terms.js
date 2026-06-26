#!/usr/bin/env node
/*
Script: create_semantic_terms.js
Purpose: For each qualified path in the `columns` array below, find the matching catalog_node
and create a semantic term node with properties suitable for cube.dev (dimension/measure/date),
then create an edge linking the semantic term (source) to the catalog column node (target).

Usage:
  TENANT_ID=<tenant> DATASOURCE_ID=<datasource> API_BASE_URL=http://localhost:8080 node scripts/create_semantic_terms.js

Required environment variables:
  - API_BASE_URL: Base URL for backend (e.g. http://localhost:8080)
  - TENANT_ID: Tenant UUID
  - DATASOURCE_ID: Datasource UUID
  - EDGE_TYPE_ID: (optional) ID of the edge_type to use for semantic mapping. If omitted, the script will try to create the edge using a relationship_type fallback.

Notes:
  - The script uses fetch (Node 18+). It is idempotent: it will skip creating a semantic term if a term with the same `node_name` and properties already exists.
  - It will attempt to locate catalog nodes by `qualified_path` by fetching `/api/catalog/nodes` and matching locally.
  - If your backend exposes admin endpoints for `catalog_edge_type`, query that table to find the correct `edge_type_id` for the mapping type you want (e.g. 'semantic_column_to_db_column' or similar).

Customize the `columns` array below with any list of qualified paths you want processed.
*/

const fetch = globalThis.fetch || require('node-fetch');
const API_BASE = process.env.API_BASE_URL || 'http://localhost:8080';
const TENANT_ID = process.env.TENANT_ID || process.env.SELECTED_TENANT_ID || '';
const DATASOURCE_ID = process.env.DATASOURCE_ID || process.env.SELECTED_DATASOURCE_ID || '';
const EDGE_TYPE_ID = process.env.EDGE_TYPE_ID || process.env.MAPPING_EDGE_TYPE_ID || '';

if (!API_BASE) {
  console.error('Missing API_BASE_URL environment variable');
  process.exit(1);
}
if (!TENANT_ID || !DATASOURCE_ID) {
  console.error('Missing TENANT_ID or DATASOURCE_ID. Set TENANT_ID and DATASOURCE_ID env vars.');
  process.exit(1);
}

// --- Customize the list of columns below (qualified_path strings) ---
const columns = [
"/hdb_catalog/hdb_action_log/status",
"/public/customer_demographics/customer_type_id",
"/public/customer_demographics/customer_desc",
"/public/customer_customer_demo/customer_id",
"/public/customer_customer_demo/customer_type_id",
"/public/customers/customer_id",
"/public/customers/company_name",
"/public/customers/contact_name",
"/public/customers/contact_title",
"/public/customers/address",
"/public/customers/city",
"/public/customers/region",
"/public/customers/postal_code",
"/public/customers/country",
"/public/customers/phone",
"/public/customers/fax",
"/public/dim_country_cd/cntry_cd",
"/public/dim_country_cd/cntry_name",
"/public/employee_territories/employee_id",
"/public/employee_territories/territory_id",
"/public/categories/category_id",
"/public/categories/category_name",
"/public/categories/description",
"/public/categories/picture",
"/public/ref_service_codes/service_code",
"/public/ref_service_codes/service_code_name",
"/public/shippers/shipper_id",
"/public/shippers/company_name",
"/public/shippers/phone",
"/public/region/region_id",
"/public/region/region_description",
"/public/us_states/state_id",
"/public/us_states/state_name",
"/public/us_states/state_abbr",
"/public/us_states/state_region",
"/public/territories/territory_id",
"/public/territories/territory_description",
"/public/territories/region_id",
"/public/orders/order_id",
"/public/orders/customer_id",
"/public/orders/employee_id",
"/public/orders/order_date",
"/public/orders/required_date",
"/public/orders/shipped_date",
"/public/orders/ship_via",
"/public/orders/freight",
"/public/orders/ship_name",
"/public/orders/ship_address",
"/public/orders/ship_city",
"/public/orders/ship_region",
"/public/orders/ship_postal_code",
"/public/orders/ship_country",
"/public/products/product_id",
"/public/products/product_name",
"/public/products/supplier_id",
"/public/products/category_id",
"/public/products/quantity_per_unit",
"/public/products/unit_price",
"/public/products/units_in_stock",
"/public/products/units_on_order",
"/public/products/reorder_level",
"/public/products/discontinued",
"/public/suppliers/supplier_id",
"/public/suppliers/company_name",
"/public/suppliers/contact_name",
"/public/suppliers/contact_title",
"/public/suppliers/address",
"/public/suppliers/city",
"/public/suppliers/region",
"/public/suppliers/postal_code",
"/public/suppliers/country",
"/public/suppliers/phone",
"/public/suppliers/fax",
"/public/suppliers/homepage",
"/public/employees/employee_id",
"/public/employees/last_name",
"/public/employees/first_name",
"/public/employees/title",
"/public/employees/title_of_courtesy",
"/public/employees/birth_date",
"/public/employees/hire_date",
"/public/employees/address",
"/public/employees/city",
"/public/employees/region",
"/public/employees/postal_code",
"/public/employees/country",
"/public/employees/home_phone",
"/public/employees/extension",
"/public/employees/photo",
"/public/employees/notes",
"/public/employees/reports_to",
"/public/employees/photo_path",
"/public/order_details/order_id",
"/public/order_details/product_id",
"/public/order_details/unit_price",
"/public/order_details/quantity",
"/public/order_details/discount",
"/hdb_catalog/hdb_version/hasura_uuid",
"/hdb_catalog/hdb_version/version",
"/hdb_catalog/hdb_version/upgraded_on",
"/hdb_catalog/hdb_version/cli_state",
"/hdb_catalog/hdb_version/console_state",
"/hdb_catalog/hdb_version/ee_client_id",
"/hdb_catalog/hdb_version/ee_client_secret",
"/hdb_catalog/hdb_action_log/status",
"/hdb_catalog/hdb_action_log/id",
"/hdb_catalog/hdb_action_log/action_name",
"/hdb_catalog/hdb_action_log/input_payload",
"/hdb_catalog/hdb_action_log/request_headers",
"/hdb_catalog/hdb_action_log/session_variables",
"/hdb_catalog/hdb_action_log/response_payload",
"/hdb_catalog/hdb_action_log/errors",
"/hdb_catalog/hdb_action_log/created_at",
"/hdb_catalog/hdb_action_log/response_received_at",
"/hdb_catalog/hdb_scheduled_events/id",
"/hdb_catalog/hdb_scheduled_events/webhook_conf",
"/hdb_catalog/hdb_scheduled_events/scheduled_time",
"/hdb_catalog/hdb_scheduled_events/retry_conf",
"/hdb_catalog/hdb_scheduled_events/payload",
"/hdb_catalog/hdb_scheduled_events/header_conf",
"/hdb_catalog/hdb_scheduled_events/status",
"/hdb_catalog/hdb_scheduled_events/tries",
"/hdb_catalog/hdb_scheduled_events/created_at",
"/hdb_catalog/hdb_scheduled_events/next_retry_at",
"/hdb_catalog/hdb_scheduled_events/comment",
"/hdb_catalog/hdb_scheduled_event_invocation_logs/id",
"/hdb_catalog/hdb_scheduled_event_invocation_logs/event_id",
"/hdb_catalog/hdb_scheduled_event_invocation_logs/status",
"/hdb_catalog/hdb_scheduled_event_invocation_logs/request",
"/hdb_catalog/hdb_scheduled_event_invocation_logs/response",
"/hdb_catalog/hdb_scheduled_event_invocation_logs/created_at",
"/hdb_catalog/hdb_metadata/id",
"/hdb_catalog/hdb_metadata/metadata",
"/hdb_catalog/hdb_metadata/resource_version",
"/hdb_catalog/hdb_schema_notifications/id",
"/hdb_catalog/hdb_schema_notifications/notification",
"/hdb_catalog/hdb_schema_notifications/resource_version",
"/hdb_catalog/hdb_schema_notifications/instance_id",
"/hdb_catalog/hdb_schema_notifications/updated_at",
"/hdb_catalog/hdb_cron_events/id",
"/hdb_catalog/hdb_cron_events/trigger_name",
"/hdb_catalog/hdb_cron_events/scheduled_time",
"/hdb_catalog/hdb_cron_events/status",
"/hdb_catalog/hdb_cron_events/tries",
"/hdb_catalog/hdb_cron_events/created_at",
"/hdb_catalog/hdb_cron_events/next_retry_at",
"/hdb_catalog/hdb_cron_event_invocation_logs/id",
"/hdb_catalog/hdb_cron_event_invocation_logs/event_id",
"/hdb_catalog/hdb_cron_event_invocation_logs/status",
"/hdb_catalog/hdb_cron_event_invocation_logs/request",
"/hdb_catalog/hdb_cron_event_invocation_logs/response",
"/hdb_catalog/hdb_cron_event_invocation_logs/created_at",
"/agg/agg_metadata/agg_name",
"/agg/agg_metadata/last_update",
"/agg/agg_metadata/definition_hash",
"/public/custom_categories/category_id",
"/public/custom_categories/category_name",
"/public/custom_categories/description",
"/public/custom_categories/picture",
"/aggregates/preaggregation_audit/id",
"/aggregates/preaggregation_audit/job_name",
"/aggregates/preaggregation_audit/metric_node_id",
"/aggregates/preaggregation_audit/grain",
"/aggregates/preaggregation_audit/records_processed",
"/aggregates/preaggregation_audit/execution_time_ms",
"/aggregates/preaggregation_audit/status",
"/aggregates/preaggregation_audit/error_message",
"/aggregates/preaggregation_audit/started_at",
"/aggregates/preaggregation_audit/completed_at",
"/aggregates/data_quality_monitoring/id",
"/aggregates/data_quality_monitoring/metric_id",
"/aggregates/data_quality_monitoring/check_type",
"/aggregates/data_quality_monitoring/check_value",
"/aggregates/data_quality_monitoring/threshold",
"/aggregates/data_quality_monitoring/status",
"/aggregates/data_quality_monitoring/checked_at",
"/aggregates/data_quality_monitoring/details",
"/aggregates/preaggregated_metrics/id",
"/aggregates/preaggregated_metrics/node_id",
"/aggregates/preaggregated_metrics/name",
"/aggregates/preaggregated_metrics/value",
"/aggregates/preaggregated_metrics/grain",
"/aggregates/preaggregated_metrics/grain_values",
"/aggregates/preaggregated_metrics/last_refresh",
"/aggregates/preaggregated_metrics/refresh_schedule",
"/aggregates/preaggregated_metrics/source_formula",
"/aggregates/preaggregated_metrics/data_quality",
"/aggregates/preaggregated_metrics/business_context",
"/aggregates/preaggregated_metrics/created_at",
"/aggregates/preaggregated_metrics/updated_at",
"/agg/agg_metadata/agg_name",
"/agg/agg_metadata/last_update",
"/agg/agg_metadata/definition_hash",
"/hdb_catalog/hdb_version/hasura_uuid",
"/hdb_catalog/hdb_version/version",
"/hdb_catalog/hdb_version/upgraded_on",
"/hdb_catalog/hdb_version/cli_state",
"/hdb_catalog/hdb_version/console_state",
"/hdb_catalog/hdb_version/ee_client_id",
"/hdb_catalog/hdb_version/ee_client_secret",
"/hdb_catalog/hdb_metadata/id",
"/hdb_catalog/hdb_metadata/metadata",
"/hdb_catalog/hdb_metadata/resource_version",
"/hdb_catalog/hdb_action_log/id",
"/hdb_catalog/hdb_action_log/action_name",
"/hdb_catalog/hdb_action_log/input_payload",
"/hdb_catalog/hdb_action_log/request_headers",
"/hdb_catalog/hdb_action_log/session_variables",
"/hdb_catalog/hdb_action_log/response_payload",
"/hdb_catalog/hdb_action_log/errors",
"/hdb_catalog/hdb_action_log/created_at",
"/hdb_catalog/hdb_action_log/response_received_at",
"/hdb_catalog/hdb_scheduled_events/id",
"/hdb_catalog/hdb_scheduled_events/webhook_conf",
"/hdb_catalog/hdb_scheduled_events/scheduled_time",
"/hdb_catalog/hdb_scheduled_events/retry_conf",
"/hdb_catalog/hdb_scheduled_events/payload",
"/hdb_catalog/hdb_scheduled_events/header_conf",
"/hdb_catalog/hub_scheduled_events/status",
"/hdb_catalog/hdb_scheduled_events/tries",
"/hdb_catalog/hdb_scheduled_events/created_at",
"/hdb_catalog/hdb_scheduled_events/next_retry_at",
];

// ------------------------------------------------------------------

async function fetchAllCatalogNodes() {
  const params = new URLSearchParams();
  params.append('tenant_id', TENANT_ID);
  params.append('datasource_id', DATASOURCE_ID);

  const url = `${API_BASE}/api/catalog/nodes?${params.toString()}`;
  console.log('Fetching catalog nodes from', url);
  const resp = await fetch(url, { headers: {
    'X-Tenant-ID': TENANT_ID,
    'X-Tenant-Datasource-ID': DATASOURCE_ID,
    'Content-Type': 'application/json'
  }, credentials: 'include' });

  if (!resp.ok) {
    const txt = await resp.text();
    throw new Error(`Failed to fetch catalog nodes: ${resp.status} ${txt}`);
  }

  const json = await resp.json();
  // API may return { catalog_node: [...] } or raw array
  if (json == null) return [];
  if (Array.isArray(json)) return json;
  if (Array.isArray(json.catalog_node)) return json.catalog_node;
  // fallback: try to find arrays inside
  for (const k of Object.keys(json)) {
    if (Array.isArray(json[k])) return json[k];
  }
  throw new Error('Unexpected catalog/nodes response format');
}

function inferPropertiesFromPath(qualifiedPath) {
  const name = qualifiedPath.split('/').pop() || qualifiedPath;
  const lower = name.toLowerCase();

  // Basic inference
  const isId = /(^id$|_id$|id$|^id_|\bid\b)/i.test(lower);
  const isDate = /date|time|created_at|updated_at|scheduled_time|response_received_at/.test(lower);
  const isBool = /is_|has_|flag|discontinued/.test(lower);
  const isAmount = /price|amount|freight|cost|total|unit_price|quantity|discount/.test(lower);
  const isText = /name|desc|address|city|region|country|phone|fax|title|notes|company_name|contact_name/.test(lower);

  let data_type = 'text';
  let semanticKind = 'dimension';
  const properties = {};

  if (isDate) {
    data_type = 'date';
    semanticKind = 'time';
    properties.grain = 'day';
    properties.is_time_dimension = true;
  } else if (isAmount) {
    data_type = 'number';
    semanticKind = 'measure';
    properties.aggregation = 'sum';
    properties.format = 'decimal';
  } else if (isBool) {
    data_type = 'boolean';
    semanticKind = 'dimension';
  } else if (isId) {
    data_type = 'number';
    semanticKind = 'dimension';
    properties.is_foreign_key = true;
  } else if (isText) {
    data_type = 'text';
    semanticKind = 'dimension';
  } else {
    // fallback
    data_type = 'text';
    semanticKind = 'dimension';
  }

  // Cube.dev-like properties (examples)
  const cubeProps = {
    data_type,
    semantic_kind: semanticKind, // 'dimension' | 'measure' | 'time'
    searchable: isText || semanticKind === 'dimension',
    groupable: semanticKind === 'dimension',
    aggregatable: semanticKind === 'measure',
    ...properties
  };

  return { node_name: name, properties: cubeProps };
}

async function createSemanticTerm(nodeName, properties) {
  const params = new URLSearchParams();
  params.append('tenant_id', TENANT_ID);
  params.append('datasource_id', DATASOURCE_ID);

  const url = `${API_BASE}/api/glossary/terms?${params.toString()}`;
  const body = {
    node_name: nodeName,
    description: `Auto-generated semantic term for ${nodeName}`,
    properties,
    catalog_type: 'semantic_term'
  };

  const resp = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Tenant-ID': TENANT_ID,
      'X-Tenant-Datasource-ID': DATASOURCE_ID,
    },
    body: JSON.stringify(body),
    credentials: 'include'
  });

  if (!resp.ok) {
    const txt = await resp.text();
    throw new Error(`Failed to create semantic term: ${resp.status} ${txt}`);
  }
  return resp.json();
}

async function createEdge(subjectNodeId, objectNodeId) {
  const params = new URLSearchParams();
  params.append('tenant_id', TENANT_ID);
  params.append('datasource_id', DATASOURCE_ID);

  const url = `${API_BASE}/api/glossary/edges?${params.toString()}`;
  const body = {
    subject_node_id: subjectNodeId,
    object_node_id: objectNodeId,
    edge_type_id: EDGE_TYPE_ID || null
  };

  const resp = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Tenant-ID': TENANT_ID,
      'X-Tenant-Datasource-ID': DATASOURCE_ID,
    },
    body: JSON.stringify(body),
    credentials: 'include'
  });

  if (!resp.ok) {
    const txt = await resp.text();
    throw new Error(`Failed to create edge: ${resp.status} ${txt}`);
  }
  return resp.json();
}

(async function main() {
  try {
    const allNodes = await fetchAllCatalogNodes();

    // Build map by qualified_path and by node_name
    const byPath = new Map();
    const byName = new Map();
    for (const n of allNodes) {
      if (n.qualified_path) byPath.set(n.qualified_path, n);
      if (n.node_name) byName.set(n.node_name, n);
    }

    const results = [];

    for (const q of columns) {
      console.log('\nProcessing', q);
      const colNode = byPath.get(q) || null;
      if (!colNode) {
        console.warn('  - catalog_node not found for qualified_path:', q);
      }

      // infer properties
      const inferred = inferPropertiesFromPath(q);
      const semanticName = inferred.node_name;
      const semanticProps = inferred.properties;

      // Check for existing semantic term with same name
      const possibleExisting = allNodes.find(n => (n.node_name === semanticName && (n.catalog_type === 'semantic_term' || n.catalog_type_name === 'semantic_term')));
      if (possibleExisting) {
        console.log('  - Semantic term already exists, skipping create:', possibleExisting.id);
        if (colNode) {
          // Optionally create edge if not present
          try {
            const edgeRes = await createEdge(possibleExisting.id, colNode.id);
            console.log('  - Created/ensured edge linking to column:', edgeRes.id || '(edge created)');
            results.push({ qualified_path: q, status: 'linked_existing', semantic: possibleExisting.id });
          } catch (e) {
            console.warn('  - Failed to create edge for existing semantic term:', e.message);
            results.push({ qualified_path: q, status: 'linked_existing_edge_failed', error: String(e) });
          }
        } else {
          results.push({ qualified_path: q, status: 'exists_unlinked', semantic: possibleExisting.id });
        }
        continue;
      }

      // create semantic term (even if the catalog column is missing)
      try {
        const created = await createSemanticTerm(semanticName, semanticProps);
        console.log('  - Created semantic term:', created.id || created);
        if (colNode) {
          // create edge linking semantic term -> column
          try {
            const edge = await createEdge(created.id || created.ID || created.Id, colNode.id);
            console.log('  - Created edge linking semantic term to column:', edge.id || edge);
            results.push({ qualified_path: q, status: 'created_and_linked', semantic: created.id || created });
          } catch (e) {
            console.warn('  - Semantic term created but failed to create edge:', e.message);
            results.push({ qualified_path: q, status: 'created_edge_failed', semantic: created.id || created, error: String(e) });
          }
        } else {
          results.push({ qualified_path: q, status: 'created_unlinked', semantic: created.id || created });
        }
      } catch (e) {
        console.error('  - Failed to create semantic term for', q, e.message);
        results.push({ qualified_path: q, status: 'create_failed', error: String(e) });
      }
    }

    console.log('\nSummary:');
    console.table(results);
  } catch (err) {
    console.error('Fatal error:', err);
    process.exit(2);
  }
})();
