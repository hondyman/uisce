import { gql } from '@apollo/client';

// ============================================================================
// Integrity Check Queries
// ============================================================================

export const GET_INTEGRITY_CHECK_HISTORY = gql`
  query GetIntegrityCheckHistory($tenant_instance_id: uuid!, $limit: Int!) {
    datasource_integrity_checks(
      where: { tenant_instance_id: { _eq: $tenant_instance_id } }
      order_by: { started_at: desc }
      limit: $limit
    ) {
      id
      check_type
      status
      postgres_row_count
      ignite_row_count
      starrocks_row_count
      row_count_delta
      row_count_delta_percent
      schema_changes
      checksum_valid
      executed_by
      started_at
      completed_at
      duration_ms
      error_message
      recommendations
    }
  }
`;

export const GET_LATEST_INTEGRITY_CHECK = gql`
  query GetLatestIntegrityCheck($tenant_instance_id: uuid!) {
    datasource_integrity_checks(
      where: { tenant_instance_id: { _eq: $tenant_instance_id } }
      order_by: { started_at: desc }
      limit: 1
    ) {
      id
      check_type
      status
      row_count_delta
      schema_changes
      error_message
      completed_at
    }
  }
`;

// ============================================================================
// Schema Snapshot Queries
// ============================================================================

export const GET_SCHEMA_SNAPSHOTS = gql`
  query GetSchemaSnapshots($tenant_instance_id: uuid!, $limit: Int!) {
    datasource_schema_snapshots(
      where: { tenant_instance_id: { _eq: $tenant_instance_id } }
      order_by: { captured_at: desc }
      limit: $limit
    ) {
      id
      table_count
      column_count
      captured_at
      captured_by
      is_baseline
      notes
      change_summary
    }
  }
`;

export const GET_SCHEMA_BASELINE = gql`
  query GetSchemaBaseline($tenant_instance_id: uuid!) {
    datasource_schema_snapshots(
      where: {
        tenant_instance_id: { _eq: $tenant_instance_id }
        is_baseline: { _eq: true }
      }
      order_by: { captured_at: desc }
      limit: 1
    ) {
      id
      snapshot_data
      table_count
      column_count
      captured_at
      captured_by
    }
  }
`;

// ============================================================================
// Health Check Queries
// ============================================================================

export const GET_HEALTH_CHECK_HISTORY = gql`
  query GetHealthCheckHistory($tenant_instance_id: uuid!, $limit: Int!) {
    datasource_health_checks(
      where: { tenant_instance_id: { _eq: $tenant_instance_id } }
      order_by: { checked_at: desc }
      limit: $limit
    ) {
      id
      status
      response_time_ms
      error_message
      connection_pool_size
      active_connections
      idle_connections
      checked_at
    }
  }
`;

// ============================================================================
// Data Classification Templates
// ============================================================================

export const GET_DATA_CLASSIFICATION_TEMPLATES = gql`
  query GetDataClassificationTemplates {
    data_classification_templates(order_by: { name: asc }) {
      id
      name
      display_name
      description
      config
      is_system
    }
  }
`;

// ============================================================================
// Datasource Health Summary (Aggregates)
// ============================================================================

export const GET_DATASOURCE_HEALTH_SUMMARY = gql`
  query GetDatasourceHealthSummary($tenant_product_id: uuid!) {
    healthy: tenant_product_datasource_aggregate(
      where: { tenant_product_id: { _eq: $tenant_product_id }, health_status: { _eq: "healthy" } }
    ) { aggregate { count } }
    degraded: tenant_product_datasource_aggregate(
      where: { tenant_product_id: { _eq: $tenant_product_id }, health_status: { _eq: "degraded" } }
    ) { aggregate { count } }
    unhealthy: tenant_product_datasource_aggregate(
      where: { tenant_product_id: { _eq: $tenant_product_id }, health_status: { _eq: "unhealthy" } }
    ) { aggregate { count } }
    unknown: tenant_product_datasource_aggregate(
      where: { tenant_product_id: { _eq: $tenant_product_id }, health_status: { _eq: "unknown" } }
    ) { aggregate { count } }
    total: tenant_product_datasource_aggregate(
      where: { tenant_product_id: { _eq: $tenant_product_id } }
    ) { aggregate { count } }
  }
`;

export const GET_DATASOURCE_INTEGRITY_SUMMARY = gql`
  query GetDatasourceIntegritySummary($tenant_product_id: uuid!) {
    valid: tenant_product_datasource_aggregate(
      where: { tenant_product_id: { _eq: $tenant_product_id }, integrity_status: { _eq: "valid" } }
    ) { aggregate { count } }
    warning: tenant_product_datasource_aggregate(
      where: { tenant_product_id: { _eq: $tenant_product_id }, integrity_status: { _eq: "warning" } }
    ) { aggregate { count } }
    invalid: tenant_product_datasource_aggregate(
      where: { tenant_product_id: { _eq: $tenant_product_id }, integrity_status: { _eq: "invalid" } }
    ) { aggregate { count } }
    unknown: tenant_product_datasource_aggregate(
      where: { tenant_product_id: { _eq: $tenant_product_id }, integrity_status: { _eq: "unknown" } }
    ) { aggregate { count } }
  }
`;
