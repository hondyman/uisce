package graphql

// Datasource GraphQL Operations - All CRUD operations for datasource entities

// ============================================================================
// Datasource Queries
// ============================================================================

// GetDatasourceByID fetches a single datasource by ID
const GetDatasourceByID = `
query GetDatasourceByID($id: uuid!) {
  tenant_product_datasource_by_pk(id: $id) {
    id
    tenant_product_id
    alpha_datasource_id
    source_name
    is_active
    config
    environment
    tags
    description
    read_only
    pool_config
    scan_schedule
    health_config
    integrity_checks
    sla_config
    data_classification
    last_heartbeat_at
    health_status
    health_message
    last_integrity_check_at
    integrity_status
    integrity_message
    last_scan_at
    last_scan_status
    connection_id
    created_at
    updated_at
    created_by
    updated_by
  }
}
`

// GetDatasourcesByTenantProduct fetches all datasources for a tenant product
const GetDatasourcesByTenantProduct = `
query GetDatasourcesByTenantProduct($tenant_product_id: uuid!) {
  tenant_product_datasource(
    where: { tenant_product_id: { _eq: $tenant_product_id } }
    order_by: { source_name: asc }
  ) {
    id
    source_name
    environment
    is_active
    health_status
    integrity_status
    last_scan_at
    last_scan_status
    tags
    read_only
    alpha_datasource {
      datasource_code
      display_name
    }
  }
}
`

// GetDatasourcesWithFilters fetches datasources with optional filters
const GetDatasourcesWithFilters = `
query GetDatasourcesWithFilters(
  $tenant_product_id: uuid!
  $environment: String
  $health_status: String
  $integrity_status: String
  $is_active: Boolean
) {
  tenant_product_datasource(
    where: {
      tenant_product_id: { _eq: $tenant_product_id }
      environment: { _eq: $environment }
      health_status: { _eq: $health_status }
      integrity_status: { _eq: $integrity_status }
      is_active: { _eq: $is_active }
    }
    order_by: { source_name: asc }
  ) {
    id
    source_name
    environment
    is_active
    health_status
    integrity_status
    last_scan_at
    tags
  }
}
`

// ============================================================================
// Datasource Mutations
// ============================================================================

// InsertDatasource creates a new datasource
const InsertDatasource = `
mutation InsertDatasource($object: tenant_product_datasource_insert_input!) {
  insert_tenant_product_datasource_one(object: $object) {
    id
    source_name
    environment
    created_at
  }
}
`

// UpdateDatasource updates an existing datasource
const UpdateDatasource = `
mutation UpdateDatasource($id: uuid!, $changes: tenant_product_datasource_set_input!) {
  update_tenant_product_datasource_by_pk(
    pk_columns: { id: $id }
    _set: $changes
  ) {
    id
    source_name
    updated_at
  }
}
`

// UpdateDatasourceHealth updates health status
const UpdateDatasourceHealth = `
mutation UpdateDatasourceHealth($id: uuid!, $status: String!, $message: String, $heartbeat_at: timestamptz) {
  update_tenant_product_datasource_by_pk(
    pk_columns: { id: $id }
    _set: {
      health_status: $status
      health_message: $message
      last_heartbeat_at: $heartbeat_at
    }
  ) {
    id
    health_status
  }
}
`

// UpdateDatasourceIntegrity updates integrity status
const UpdateDatasourceIntegrity = `
mutation UpdateDatasourceIntegrity($id: uuid!, $status: String!, $message: String, $check_at: timestamptz) {
  update_tenant_product_datasource_by_pk(
    pk_columns: { id: $id }
    _set: {
      integrity_status: $status
      integrity_message: $message
      last_integrity_check_at: $check_at
    }
  ) {
    id
    integrity_status
  }
}
`

// UpdateDatasourceScanStatus updates scan status
const UpdateDatasourceScanStatus = `
mutation UpdateDatasourceScanStatus($id: uuid!, $status: String!, $scan_at: timestamptz, $error: String) {
  update_tenant_product_datasource_by_pk(
    pk_columns: { id: $id }
    _set: {
      last_scan_status: $status
      last_scan_at: $scan_at
      scan_error_message: $error
    }
  ) {
    id
    last_scan_status
  }
}
`

// DeleteDatasource deletes a datasource
const DeleteDatasource = `
mutation DeleteDatasource($id: uuid!) {
  delete_tenant_product_datasource_by_pk(id: $id) {
    id
  }
}
`

// ============================================================================
// Integrity Check Operations
// ============================================================================

// InsertIntegrityCheck records an integrity check result
const InsertIntegrityCheck = `
mutation InsertIntegrityCheck($object: datasource_integrity_checks_insert_input!) {
  insert_datasource_integrity_checks_one(object: $object) {
    id
    status
    check_type
    started_at
  }
}
`

// GetIntegrityCheckHistory fetches recent integrity checks
const GetIntegrityCheckHistory = `
query GetIntegrityCheckHistory($datasource_id: uuid!, $limit: Int!) {
  datasource_integrity_checks(
    where: { datasource_id: { _eq: $datasource_id } }
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
`

// UpdateIntegrityCheckComplete marks an integrity check as complete
const UpdateIntegrityCheckComplete = `
mutation UpdateIntegrityCheckComplete(
  $id: uuid!
  $status: String!
  $completed_at: timestamptz!
  $duration_ms: Int
  $postgres_row_count: bigint
  $ignite_row_count: bigint
  $starrocks_row_count: bigint
  $row_count_delta: bigint
  $row_count_delta_percent: numeric
  $schema_changes: jsonb
  $checksum_valid: Boolean
  $error_message: String
  $recommendations: jsonb
) {
  update_datasource_integrity_checks_by_pk(
    pk_columns: { id: $id }
    _set: {
      status: $status
      completed_at: $completed_at
      duration_ms: $duration_ms
      postgres_row_count: $postgres_row_count
      ignite_row_count: $ignite_row_count
      starrocks_row_count: $starrocks_row_count
      row_count_delta: $row_count_delta
      row_count_delta_percent: $row_count_delta_percent
      schema_changes: $schema_changes
      checksum_valid: $checksum_valid
      error_message: $error_message
      recommendations: $recommendations
    }
  ) {
    id
    status
    completed_at
  }
}
`

// ============================================================================
// Schema Snapshot Operations
// ============================================================================

// InsertSchemaSnapshot records a schema snapshot
const InsertSchemaSnapshot = `
mutation InsertSchemaSnapshot($object: datasource_schema_snapshots_insert_input!) {
  insert_datasource_schema_snapshots_one(object: $object) {
    id
    captured_at
    is_baseline
  }
}
`

// GetLatestSchemaBaseline fetches the most recent baseline snapshot
const GetLatestSchemaBaseline = `
query GetLatestSchemaBaseline($datasource_id: uuid!) {
  datasource_schema_snapshots(
    where: {
      datasource_id: { _eq: $datasource_id }
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
`

// ClearPreviousBaselines unsets baseline flag on previous snapshots
const ClearPreviousBaselines = `
mutation ClearPreviousBaselines($datasource_id: uuid!) {
  update_datasource_schema_snapshots(
    where: {
      datasource_id: { _eq: $datasource_id }
      is_baseline: { _eq: true }
    }
    _set: { is_baseline: false }
  ) {
    affected_rows
  }
}
`

// ============================================================================
// Health Check Operations
// ============================================================================

// InsertHealthCheck records a health check result
const InsertHealthCheck = `
mutation InsertHealthCheck($object: datasource_health_checks_insert_input!) {
  insert_datasource_health_checks_one(object: $object) {
    id
    status
    checked_at
  }
}
`

// GetHealthCheckHistory fetches recent health checks
const GetHealthCheckHistory = `
query GetHealthCheckHistory($datasource_id: uuid!, $limit: Int!) {
  datasource_health_checks(
    where: { datasource_id: { _eq: $datasource_id } }
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
`

// ============================================================================
// Connection Operations
// ============================================================================

// GetConnectionByID fetches a connection by ID
const GetConnectionByID = `
query GetConnectionByID($id: uuid!) {
  connections_by_pk(id: $id) {
    id
    tenant_id
    name
    type
    host
    port
    database
    schema
    username
    metadata
    is_active
    last_tested_at
    last_test_status
  }
}
`

// InsertConnection creates a new connection
const InsertConnection = `
mutation InsertConnection($object: connections_insert_input!) {
  insert_connections_one(object: $object) {
    id
    name
    created_at
  }
}
`

// UpdateConnection updates an existing connection
const UpdateConnection = `
mutation UpdateConnection($id: uuid!, $changes: connections_set_input!) {
  update_connections_by_pk(
    pk_columns: { id: $id }
    _set: $changes
  ) {
    id
    updated_at
  }
}
`

// ============================================================================
// Data Classification Templates
// ============================================================================

// GetDataClassificationTemplates fetches all classification templates
const GetDataClassificationTemplates = `
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
`

// ============================================================================
// Aggregations & Statistics
// ============================================================================

// GetDatasourceHealthSummary gets health status counts
const GetDatasourceHealthSummary = `
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
`
