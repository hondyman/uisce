import { renderHook, waitFor } from '@testing-library/react-hooks';
import { useRelationshipDiscovery } from '../useRelationshipDiscovery';

// Mock fetch
global.fetch = jest.fn();

describe('useRelationshipDiscovery', () => {
  const tenantId = 'tenant-123';
  const datasourceId = 'ds-456';

  beforeEach(() => {
    (global.fetch as jest.Mock).mockClear();
  });

  describe('discoverRelationships', () => {
    it('should discover relationships successfully', async () => {
      const mockResponse = {
        directRelationships: [
          {
            relatedEntityId: 'entity-789',
            relatedEntityName: 'Orders',
            linkType: 'DIRECT_FK' as const,
            confidence: 0.95,
            cardinality: '1:N' as const,
            foreignKeyPath: [],
            columnMapping: [],
          },
        ],
        multiHopPaths: [],
      };

      (global.fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      });

      const { result } = renderHook(() =>
        useRelationshipDiscovery(tenantId, datasourceId)
      );

      const response = await result.current.discoverRelationships({
        entityId: 'entity-123',
      });

      expect(response).toEqual(mockResponse);
      expect(global.fetch).toHaveBeenCalledWith(
        '/api/relationships/discover',
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            'X-Tenant-ID': tenantId,
            'X-Tenant-Datasource-ID': datasourceId,
          }),
        })
      );
    });

    it('should handle errors gracefully', async () => {
      (global.fetch as jest.Mock).mockResolvedValueOnce({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
        json: async () => ({}),
      });

      const { result } = renderHook(() =>
        useRelationshipDiscovery(tenantId, datasourceId)
      );

      const response = await result.current.discoverRelationships({
        entityId: 'entity-123',
      });

      expect(response).toBeNull();
      expect(result.current.error).not.toBeNull();
    });

    it('should set loading state correctly', async () => {
      (global.fetch as jest.Mock).mockImplementation(
        () =>
          new Promise((resolve) =>
            setTimeout(
              () =>
                resolve({
                  ok: true,
                  json: async () => ({
                    directRelationships: [],
                    multiHopPaths: [],
                  }),
                }),
              100
            )
          )
      );

      const { result } = renderHook(() =>
        useRelationshipDiscovery(tenantId, datasourceId)
      );

      expect(result.current.loading).toBe(false);

      result.current.discoverRelationships({ entityId: 'entity-123' });

      expect(result.current.loading).toBe(true);

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });
    });
  });

  describe('applyRelationship', () => {
    it('should apply a relationship successfully', async () => {
      (global.fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ success: true }),
      });

      const { result } = renderHook(() =>
        useRelationshipDiscovery(tenantId, datasourceId)
      );

      const success = await result.current.applyRelationship({
        sourceEntityId: 'entity-123',
        targetEntityId: 'entity-789',
        linkType: 'DIRECT_FK',
        confidence: 0.95,
        cardinality: '1:N',
        foreignKeyPath: [],
        columnMapping: [],
      });

      expect(success).toBe(true);
      expect(global.fetch).toHaveBeenCalledWith(
        '/api/relationships/apply',
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            'X-Tenant-ID': tenantId,
            'X-Tenant-Instance-ID': datasourceId,
          }),
        })
      );
    });

    it('should handle apply errors', async () => {
      (global.fetch as jest.Mock).mockResolvedValueOnce({
        ok: false,
        status: 400,
        json: async () => ({ error: 'Invalid relationship' }),
      });

      const { result } = renderHook(() =>
        useRelationshipDiscovery(tenantId, datasourceId)
      );

      const success = await result.current.applyRelationship({
        sourceEntityId: 'entity-123',
        targetEntityId: 'entity-789',
        linkType: 'DIRECT_FK',
        confidence: 0.95,
        cardinality: '1:N',
        foreignKeyPath: [],
        columnMapping: [],
      });

      expect(success).toBe(false);
      expect(result.current.error).not.toBeNull();
    });
  });
});
