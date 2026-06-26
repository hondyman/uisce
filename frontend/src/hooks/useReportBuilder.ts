import { useState, useCallback } from 'react';

export interface Metric {
  field: string;
  aggregation: 'SUM' | 'AVG' | 'COUNT' | 'MIN' | 'MAX';
  alias: string;
}

export interface Filter {
  field: string;
  operator: '=' | '>' | '<' | 'LIKE' | 'IN';
  value: string;
}

export interface ReportQueryConfig {
  baseEntityId: string;
  baseEntityName?: string;
  relatedEntities: string[];
  metrics: Metric[];
  dimensions: string[];
  filters: Filter[];
}

export interface GenerateSQLRequest extends ReportQueryConfig {}

export interface GenerateSQLResponse {
  query: string;
}

export interface ExecuteReportRequest extends ReportQueryConfig {
  limit?: number;
}

export interface ExecuteReportResponse {
  query: string;
  results: any[];
  rowCount: number;
}

export const useReportBuilder = (tenantId: string, datasourceId: string) => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const generateSQL = useCallback(
    async (config: GenerateSQLRequest): Promise<string | null> => {
      setLoading(true);
      setError(null);

      try {
        const response = await fetch('/api/reports/generate', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'X-Tenant-ID': tenantId,
            'X-Tenant-Datasource-ID': datasourceId,
          },
          body: JSON.stringify(config),
        });

        if (!response.ok) {
          const errorData = await response.json().catch(() => ({}));
          throw new Error(
            errorData.error || `Failed to generate SQL: ${response.statusText}`
          );
        }

        const data: GenerateSQLResponse = await response.json();
        return data.query;
      } catch (err) {
        const errorMsg = err instanceof Error ? err.message : 'Failed to generate SQL';
        setError(errorMsg);
        return null;
      } finally {
        setLoading(false);
      }
    },
    [tenantId, datasourceId]
  );

  const executeReport = useCallback(
    async (config: ExecuteReportRequest): Promise<ExecuteReportResponse | null> => {
      setLoading(true);
      setError(null);

      try {
        const response = await fetch('/api/reports/preview', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'X-Tenant-ID': tenantId,
            'X-Tenant-Datasource-ID': datasourceId,
          },
          body: JSON.stringify({
            ...config,
            limit: config.limit || 100,
          }),
        });

        if (!response.ok) {
          const errorData = await response.json().catch(() => ({}));
          throw new Error(
            errorData.error || `Failed to execute report: ${response.statusText}`
          );
        }

        const data: ExecuteReportResponse = await response.json();
        return data;
      } catch (err) {
        const errorMsg = err instanceof Error ? err.message : 'Failed to execute report';
        setError(errorMsg);
        return null;
      } finally {
        setLoading(false);
      }
    },
    [tenantId, datasourceId]
  );

  const exportReport = useCallback(
    async (
      config: ExecuteReportRequest,
      format: 'csv' | 'json' = 'csv'
    ): Promise<string | null> => {
      setLoading(true);
      setError(null);

      try {
        const response = await fetch(`/api/reports/export?format=${format}`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'X-Tenant-ID': tenantId,
            'X-Tenant-Datasource-ID': datasourceId,
          },
          body: JSON.stringify(config),
        });

        if (!response.ok) {
          const errorData = await response.json().catch(() => ({}));
          throw new Error(
            errorData.error || `Failed to export report: ${response.statusText}`
          );
        }

        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        return url;
      } catch (err) {
        const errorMsg = err instanceof Error ? err.message : 'Failed to export report';
        setError(errorMsg);
        return null;
      } finally {
        setLoading(false);
      }
    },
    [tenantId, datasourceId]
  );

  return {
    generateSQL,
    executeReport,
    exportReport,
    loading,
    error,
  };
};
