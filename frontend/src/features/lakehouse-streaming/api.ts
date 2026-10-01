/**
 * Lakehouse & CDC Streaming API Client
 */

export interface StreamLoaderInfo {
  topic: string;
  target_table: string;
  business_object: string;
  status: 'HEALTHY' | 'RUNNING' | 'WARNING' | 'ERROR';
  assigned_tenant_id?: string;
  validation_enabled: boolean;
  total_consumed: number;
  total_loaded: number;
  blocked_violations: number;
  tenant_mismatches: number;
  last_event_time?: string;
}

export interface LakehouseOverview {
  lakekeeper_status: string;
  catalog_uri: string;
  warehouse_bucket: string;
  s3_endpoint: string;
  loaders: StreamLoaderInfo[];
  total_loaded_today: number;
  total_dlq_blocked: number;
  active_partitions: number;
  iceberg_tables: number;
}

export interface IcebergColumn {
  name: string;
  type: string;
  required: boolean;
  doc?: string;
}

export interface IcebergTableDef {
  namespace: string;
  table_name: string;
  storage_tier: string;
  file_format: string;
  location: string;
  partition_fields: string[];
  columns: IcebergColumn[];
  properties?: Record<string, string>;
  row_count: number;
  size_bytes: number;
  last_updated: string;
}

export interface DLQViolationRecord {
  id: string;
  tenant_id: string;
  rule_key: string;
  rule_name: string;
  severity: string;
  business_object: string;
  record_id: string;
  field_name: string;
  invalid_value: string;
  error_message: string;
  write_blocked: boolean;
  source_subsystem: string;
  created_at: string;
  details?: Record<string, unknown>;
}

export const lakehouseApi = {
  async getOverview(): Promise<LakehouseOverview> {
    const res = await fetch('/api/lakehouse-streaming/overview');
    if (!res.ok) {
      throw new Error(`Failed to fetch lakehouse overview: ${res.statusText}`);
    }
    return res.json();
  },

  async listTables(): Promise<IcebergTableDef[]> {
    const res = await fetch('/api/lakehouse-streaming/tables');
    if (!res.ok) {
      throw new Error(`Failed to fetch iceberg tables: ${res.statusText}`);
    }
    return res.json();
  },

  async provisionTable(req: { namespace: string; table_name: string; partition_fields: string[]; business_object?: string }): Promise<any> {
    const res = await fetch('/api/lakehouse-streaming/provision', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(req),
    });
    if (!res.ok) {
      throw new Error(`Failed to provision table: ${res.statusText}`);
    }
    return res.json();
  },

  async listDLQ(limit = 50): Promise<DLQViolationRecord[]> {
    const res = await fetch(`/api/lakehouse-streaming/dlq?limit=${limit}`);
    if (!res.ok) {
      throw new Error(`Failed to list DLQ records: ${res.statusText}`);
    }
    return res.json();
  },

  async replayDLQ(violationIds: string[], action: 'replay' | 'dismiss'): Promise<any> {
    const res = await fetch('/api/lakehouse-streaming/dlq/replay', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ violation_ids: violationIds, action }),
    });
    if (!res.ok) {
      throw new Error(`Failed to replay DLQ: ${res.statusText}`);
    }
    return res.json();
  },
};
