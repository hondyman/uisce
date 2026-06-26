import { renderHook, waitFor } from '@testing-library/react-hooks';
import { useReportBuilder } from '../useReportBuilder';

// Mock fetch
global.fetch = jest.fn();

describe('useReportBuilder', () => {
  const tenantId = 'tenant-123';
  const datasourceId = 'ds-456';

  beforeEach(() => {
    (global.fetch as jest.Mock).mockClear();
  });

  describe('generateSQL', () => {
    it('should generate SQL successfully', async () => {
      const mockSQL = 'SELECT * FROM customers JOIN orders';
      (global.fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ query: mockSQL }),
      });

      const { result } = renderHook(() =>
        useReportBuilder(tenantId, datasourceId)
      );

      const sql = await result.current.generateSQL({
        baseEntityId: 'customers',
        relatedEntities: ['orders'],
        metrics: [],
        dimensions: [],
        filters: [],
      });

      expect(sql).toBe(mockSQL);
      expect(global.fetch).toHaveBeenCalledWith(
        '/api/reports/generate',
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            'X-Tenant-ID': tenantId,
            'X-Tenant-Datasource-ID': datasourceId,
          }),
        })
      );
    });

    it('should handle generation errors', async () => {
      (global.fetch as jest.Mock).mockResolvedValueOnce({
        ok: false,
        status: 400,
        json: async () => ({ error: 'Invalid config' }),
      });

      const { result } = renderHook(() =>
        useReportBuilder(tenantId, datasourceId)
      );

      const sql = await result.current.generateSQL({
        baseEntityId: '',
        relatedEntities: [],
        metrics: [],
        dimensions: [],
        filters: [],
      });

      expect(sql).toBeNull();
      expect(result.current.error).not.toBeNull();
    });
  });

  describe('executeReport', () => {
    it('should execute report and return results', async () => {
      const mockResults = {
        query: 'SELECT * FROM customers',
        results: [{ id: 1, name: 'John' }],
        rowCount: 1,
      };

      (global.fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => mockResults,
      });

      const { result } = renderHook(() =>
        useReportBuilder(tenantId, datasourceId)
      );

      const response = await result.current.executeReport({
        baseEntityId: 'customers',
        relatedEntities: [],
        metrics: [],
        dimensions: [],
        filters: [],
      });

      expect(response).toEqual(mockResults);
      expect(global.fetch).toHaveBeenCalledWith(
        '/api/reports/preview',
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            'X-Tenant-ID': tenantId,
            'X-Tenant-Instance-ID': datasourceId,
          }),
        })
      );
    });

    it('should set loading state during execution', async () => {
      (global.fetch as jest.Mock).mockImplementation(
        () =>
          new Promise((resolve) =>
            setTimeout(
              () =>
                resolve({
                  ok: true,
                  json: async () => ({
                    query: '',
                    results: [],
                    rowCount: 0,
                  }),
                }),
              100
            )
          )
      );

      const { result } = renderHook(() =>
        useReportBuilder(tenantId, datasourceId)
      );

      expect(result.current.loading).toBe(false);

      result.current.executeReport({
        baseEntityId: 'customers',
        relatedEntities: [],
        metrics: [],
        dimensions: [],
        filters: [],
      });

      expect(result.current.loading).toBe(true);

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });
    });
  });

  describe('exportReport', () => {
    it('should export report as CSV', async () => {
      const mockBlob = new Blob(['data'], { type: 'text/csv' });
      (global.fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        blob: async () => mockBlob,
      });

      const { result } = renderHook(() =>
        useReportBuilder(tenantId, datasourceId)
      );

      const url = await result.current.exportReport(
        {
          baseEntityId: 'customers',
          relatedEntities: [],
          metrics: [],
          dimensions: [],
          filters: [],
        },
        'csv'
      );

      expect(url).not.toBeNull();
      expect(global.fetch).toHaveBeenCalledWith(
        '/api/reports/export?format=csv',
        expect.any(Object)
      );
    });
  });
});
