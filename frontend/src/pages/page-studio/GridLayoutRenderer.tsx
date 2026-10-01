import React, { useEffect, useState, useMemo } from 'react';
import { Box, Paper, Typography, IconButton, Stack } from '@mui/material';
import DragIndicatorIcon from '@mui/icons-material/DragIndicator';
import AspectRatioIcon from '@mui/icons-material/AspectRatio';
import type { ComponentDefinition, PageGridLayout, GridLayoutItem, ReportParameterBarConfig } from '../../types/pageStudio';
import { batchExecuteSavedQueries, BatchExecuteQueryItem, BatchExecuteItemResult, RuntimeFilter } from '../../features/query-builder/services/savedQueryApi';
import { useCrossFilterBus, useActiveCrossFilters } from '../../features/query-builder/utils/crossFilterBus';
import ReportParameterBar from './ReportParameterBar';

export interface GridLayoutRendererProps {
  layout: PageGridLayout & { parameterBar?: ReportParameterBarConfig };
  components: Record<string, ComponentDefinition>;
  mode?: 'design' | 'preview';
  tenantId?: string;
  variables?: Record<string, any>;
  onUpdateVariable?: (name: string, value: any) => void;
  onBulkBindTiles?: (paramVarName: string, updatedComponents: Record<string, ComponentDefinition>) => void;
  searchParams?: URLSearchParams;
  onUpdateSearchParams?: (params: Record<string, string>) => void;
  onUpdateLayoutItem?: (item: GridLayoutItem) => void;
  renderComponent: (comp: ComponentDefinition, data?: BatchExecuteItemResult) => React.ReactNode;
}

export const GridLayoutRenderer: React.FC<GridLayoutRendererProps> = ({
  layout,
  components,
  mode = 'preview',
  tenantId = 'default',
  variables = {},
  onUpdateVariable,
  onBulkBindTiles,
  searchParams,
  onUpdateSearchParams,
  onUpdateLayoutItem,
  renderComponent,
}) => {
  const bus = useCrossFilterBus();
  const activeFilters = useActiveCrossFilters(bus);
  const [batchResults, setBatchResults] = useState<Record<string, BatchExecuteItemResult>>({});
  const [loading, setLoading] = useState(false);

  // Extract all queries from placed components
  const queriesToBatch = useMemo(() => {
    const items: BatchExecuteQueryItem[] = [];
    for (const item of layout.items) {
      const comp = components[item.i];
      if (!comp) continue;

      const savedQueryId = (comp.props?.config as any)?.savedQueryId || (comp.props as any)?.savedQueryId;
      if (savedQueryId) {
        // Resolve parameter bindings against page variables or literal defaults
        let resolvedParams: Record<string, any> = { ...((comp.props as any)?.params || {}) };
        const savedQueryParams = comp.props?.savedQueryParams as Record<string, any> | undefined;
        if (savedQueryParams && variables) {
          for (const [paramName, binding] of Object.entries(savedQueryParams)) {
            if (binding?.mode === 'pageVar' && binding?.varName) {
              resolvedParams[paramName] = variables[binding.varName];
            } else if (binding?.mode === 'literal') {
              resolvedParams[paramName] = binding.value;
            }
          }
        }

        // Map active cross filters to runtime filters
        const runtimeFilters: RuntimeFilter[] = activeFilters.map((f) => ({
          termNodeId: f.termNodeId,
          operator: f.operator || (Array.isArray(f.value) ? 'in' : 'eq'),
          value: f.value,
        }));

        items.push({
          id: item.i,
          savedQueryId,
          params: Object.keys(resolvedParams).length > 0 ? resolvedParams : undefined,
          runtimeFilters,
          limit: (comp.props as any)?.limit || 1000,
        });
      }
    }
    return items;
  }, [layout.items, components, activeFilters, variables]);

  const queryPayloadKey = useMemo(() => JSON.stringify(queriesToBatch), [queriesToBatch]);

  // Execute batch load when queries change
  useEffect(() => {
    let cancelled = false;
    if (!queriesToBatch.length) return;

    setLoading(true);
    batchExecuteSavedQueries(queriesToBatch)
      .then((resp) => {
        if (cancelled) return;
        const mapped: Record<string, BatchExecuteItemResult> = {};
        for (const r of resp.results) {
          mapped[r.id] = r;
        }
        setBatchResults(mapped);
      })
      .catch((err) => {
        console.error('Failed to batch execute queries for grid layout:', err);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [queryPayloadKey]);

  return (
    <Box sx={{ width: '100%' }}>
      {layout.parameterBar?.enabled && (
        <ReportParameterBar
          config={layout.parameterBar}
          components={components}
          variables={variables}
          onUpdateVariable={onUpdateVariable || (() => {})}
          onBulkBindTiles={onBulkBindTiles}
          mode={mode}
          searchParams={searchParams}
          onUpdateSearchParams={onUpdateSearchParams}
        />
      )}
      <Box
        sx={{
          display: 'grid',
          gridTemplateColumns: 'repeat(12, 1fr)',
          gap: 2,
          width: '100%',
          minHeight: 400,
          p: 1,
        }}
      >
        {layout.items.map((item) => {
          const comp = components[item.i];
          if (!comp) return null;

          const result = batchResults[item.i];

          return (
            <Box
              key={item.i}
              sx={{
                gridColumn: `${item.x + 1} / span ${item.w}`,
                gridRow: `${item.y + 1} / span ${item.h}`,
                minHeight: item.h * 60,
                position: 'relative',
                borderRadius: 1,
              }}
            >
              {mode === 'design' && (
                <Stack
                  direction="row"
                  spacing={0.5}
                  sx={{
                    position: 'absolute',
                    top: 4,
                    right: 4,
                    zIndex: 2,
                    bgcolor: 'background.paper',
                    borderRadius: 1,
                    boxShadow: 1,
                    p: 0.25,
                  }}
                >
                  <IconButton size="small" title="Move tile (Drag handle)">
                    <DragIndicatorIcon fontSize="small" />
                  </IconButton>
                  <IconButton
                    size="small"
                    title="Resize tile"
                    onClick={() => {
                      if (onUpdateLayoutItem) {
                        const nextW = item.w === 12 ? 6 : item.w === 6 ? 3 : 12;
                        onUpdateLayoutItem({ ...item, w: nextW });
                      }
                    }}
                  >
                    <AspectRatioIcon fontSize="small" />
                  </IconButton>
                </Stack>
              )}

              {renderComponent(comp, result)}
            </Box>
          );
        })}
      </Box>
    </Box>
  );
};

export default GridLayoutRenderer;
