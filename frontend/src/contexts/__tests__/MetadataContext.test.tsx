import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { act } from 'react-dom/test-utils';
import React, { useEffect } from 'react';
import { MetadataProvider, useLayout, useSchema, useMetadataContext } from '../MetadataContext';

// Mock fetch
const mockFetch = vi.fn();
global.fetch = mockFetch;

// Mock localStorage
const mockLocalStorage: Record<string, string> = {
  'selected_tenant': JSON.stringify({ id: 'tenant-123' }),
  'selected_datasource': JSON.stringify({ id: 'ds-456' }),
};

beforeEach(() => {
  mockFetch.mockClear();
  vi.spyOn(Storage.prototype, 'getItem').mockImplementation((key) => mockLocalStorage[key] || null);
});

afterEach(() => {
  vi.restoreAllMocks();
});

// Test component that uses useLayout hook
function LayoutConsumer({ layoutKey, testId }: { layoutKey: string | null; testId: string }) {
  const { layout, loading, error } = useLayout(layoutKey);
  return (
    <div data-testid={testId}>
      <span data-testid={`${testId}-loading`}>{loading ? 'loading' : 'done'}</span>
      <span data-testid={`${testId}-name`}>{layout?.name || 'none'}</span>
      <span data-testid={`${testId}-error`}>{error?.message || 'no-error'}</span>
    </div>
  );
}

// Test component that exposes context methods
function ContextConsumer({ onReady }: { onReady: (ctx: ReturnType<typeof useMetadataContext>) => void }) {
  const ctx = useMetadataContext();
  useEffect(() => {
    onReady(ctx);
  }, [ctx, onReady]);
  return null;
}

describe('MetadataProvider', () => {
  describe('useLayout hook', () => {
    it('should return null layout when key is null', () => {
      render(
        <MetadataProvider>
          <LayoutConsumer layoutKey={null} testId="test" />
        </MetadataProvider>
      );
      
      expect(screen.getByTestId('test-loading').textContent).toBe('done');
      expect(screen.getByTestId('test-name').textContent).toBe('none');
      expect(mockFetch).not.toHaveBeenCalled();
    });

    it('should fetch layout and update state', async () => {
      const mockLayout = { id: '1', name: 'Test Layout', layout: { fields: [] } };
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockLayout,
      });

      render(
        <MetadataProvider>
          <LayoutConsumer layoutKey="test-layout" testId="test" />
        </MetadataProvider>
      );

      expect(screen.getByTestId('test-loading').textContent).toBe('loading');

      await waitFor(() => {
        expect(screen.getByTestId('test-loading').textContent).toBe('done');
      });

      expect(screen.getByTestId('test-name').textContent).toBe('Test Layout');
      expect(mockFetch).toHaveBeenCalledTimes(1);
      expect(mockFetch).toHaveBeenCalledWith('/api/layouts/test-layout', expect.any(Object));
    });

    it('should include tenant headers in requests', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ id: '1', name: 'Test', layout: {} }),
      });

      render(
        <MetadataProvider>
          <LayoutConsumer layoutKey="test-layout" testId="test" />
        </MetadataProvider>
      );

      await waitFor(() => {
        expect(mockFetch).toHaveBeenCalled();
      });

      const [, options] = mockFetch.mock.calls[0];
      const headers = options.headers as Headers;
      expect(headers.get('X-Tenant-ID')).toBe('tenant-123');
      expect(headers.get('X-Tenant-Datasource-ID')).toBe('ds-456');
    });

    it('should handle 404 responses gracefully', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 404,
      });

      render(
        <MetadataProvider>
          <LayoutConsumer layoutKey="nonexistent" testId="test" />
        </MetadataProvider>
      );

      await waitFor(() => {
        expect(screen.getByTestId('test-loading').textContent).toBe('done');
      });

      expect(screen.getByTestId('test-name').textContent).toBe('none');
    });
  });

  describe('request deduplication', () => {
    it('should deduplicate concurrent requests for the same layout', async () => {
      const mockLayout = { id: '1', name: 'Shared Layout', layout: {} };
      mockFetch.mockImplementation(() => 
        new Promise(resolve => 
          setTimeout(() => resolve({
            ok: true,
            json: async () => mockLayout,
          }), 50)
        )
      );

      // Simulate multiple components requesting the same layout
      render(
        <MetadataProvider>
          <LayoutConsumer layoutKey="shared-layout" testId="consumer1" />
          <LayoutConsumer layoutKey="shared-layout" testId="consumer2" />
          <LayoutConsumer layoutKey="shared-layout" testId="consumer3" />
        </MetadataProvider>
      );

      // Wait for all to resolve
      await waitFor(() => {
        expect(screen.getByTestId('consumer1-loading').textContent).toBe('done');
        expect(screen.getByTestId('consumer2-loading').textContent).toBe('done');
        expect(screen.getByTestId('consumer3-loading').textContent).toBe('done');
      });

      // Only ONE fetch should have been made despite multiple consumers
      expect(mockFetch).toHaveBeenCalledTimes(1);

      // All consumers should have the same data
      expect(screen.getByTestId('consumer1-name').textContent).toBe('Shared Layout');
      expect(screen.getByTestId('consumer2-name').textContent).toBe('Shared Layout');
      expect(screen.getByTestId('consumer3-name').textContent).toBe('Shared Layout');
    });
  });

  describe('cache invalidation', () => {
    it('should invalidate all cached data', async () => {
      let contextRef: ReturnType<typeof useMetadataContext> | null = null;
      
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ id: '1', name: 'Test', layout: {} }),
      });

      render(
        <MetadataProvider>
          <ContextConsumer onReady={(ctx) => { contextRef = ctx; }} />
        </MetadataProvider>
      );

      // Wait for context to be available
      await waitFor(() => {
        expect(contextRef).not.toBeNull();
      });

      // Cache an item
      await act(async () => {
        await contextRef!.fetchLayout('layout-1');
      });

      expect(contextRef!.getLayout('layout-1')).not.toBeNull();

      // Invalidate all
      act(() => {
        contextRef!.invalidateAll();
      });

      expect(contextRef!.getLayout('layout-1')).toBeNull();
    });
  });

  describe('preloading', () => {
    it('should preload multiple layouts in parallel', async () => {
      let contextRef: ReturnType<typeof useMetadataContext> | null = null;
      
      mockFetch.mockImplementation((url: string) => 
        Promise.resolve({
          ok: true,
          json: async () => ({ id: url, name: 'Preloaded', layout: {} }),
        })
      );

      render(
        <MetadataProvider>
          <ContextConsumer onReady={(ctx) => { contextRef = ctx; }} />
        </MetadataProvider>
      );

      await waitFor(() => {
        expect(contextRef).not.toBeNull();
      });

      act(() => {
        contextRef!.preloadLayouts(['preload-1', 'preload-2', 'preload-3']);
      });

      await waitFor(() => {
        expect(contextRef!.getLayout('preload-1')).not.toBeNull();
        expect(contextRef!.getLayout('preload-2')).not.toBeNull();
        expect(contextRef!.getLayout('preload-3')).not.toBeNull();
      });

      expect(mockFetch).toHaveBeenCalledTimes(3);
    });
  });
});

// Test component for schema
function SchemaConsumer({ schemaKey, testId }: { schemaKey: string | null; testId: string }) {
  const { schema, loading } = useSchema(schemaKey);
  return (
    <div data-testid={testId}>
      <span data-testid={`${testId}-loading`}>{loading ? 'loading' : 'done'}</span>
      <span data-testid={`${testId}-slug`}>{schema?.slug || 'none'}</span>
    </div>
  );
}

describe('useSchema hook', () => {
  it('should fetch schema with deduplication', async () => {
    const mockSchema = { 
      id: '1', 
      slug: 'test-schema',
      fields: [{ name: 'id', type: 'string' }]
    };
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => mockSchema,
    });

    render(
      <MetadataProvider>
        <SchemaConsumer schemaKey="test-schema" testId="test" />
      </MetadataProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId('test-loading').textContent).toBe('done');
    });

    expect(screen.getByTestId('test-slug').textContent).toBe('test-schema');
    expect(mockFetch).toHaveBeenCalledWith('/api/schemas/test-schema', expect.any(Object));
  });
});
