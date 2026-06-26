import { useEffect, useRef } from 'react';
import { devWarn, devError } from '../utils/devLogger';
import { SemanticModel } from '../components/UnifiedSemanticBuilder/types';

const asRecord = (v: unknown): Record<string, unknown> | null => {
  if (v && typeof v === 'object' && !Array.isArray(v)) return v as Record<string, unknown>;
  return null;
};

export interface UseCoreModelBuilderProps {
  selectedModel: {
    is_core?: boolean;
    is_custom?: boolean;
    model_key?: string;
    display_name?: string;
    // When present, this should already be in the builder's SemanticModel shape
    // or a close variant; we'll normalize below.
    resolved_config?: unknown;
  } | null;
  datasourceId: string;
  setSemanticModel: React.Dispatch<React.SetStateAction<SemanticModel>>;
  setLastLoadedModelKey: React.Dispatch<React.SetStateAction<string | null>>;
  showNotification: (message: string, type: 'success' | 'error') => void;
}

export function useCoreModelBuilder({
  selectedModel,
  datasourceId,
  setSemanticModel,
  setLastLoadedModelKey,
  showNotification,
}: UseCoreModelBuilderProps) {
  // Track the last core model_key we processed to avoid repeating work on re-renders
  const lastProcessedKeyRef = useRef<string | null>(null);
  // Track the kind (core/custom) that produced the lastProcessedKey so switching from
  // a custom view back to a core model with the same key will still refresh the canvas
  // and forms. This addresses the issue where auto-selecting the FIRST item on a tab
  // switch didn't immediately update the UI when the model key matched a previously
  // processed core selection but the current view was custom-only.
  const lastProcessedKindRef = useRef<'core' | 'custom' | null>(null);
  useEffect(() => {
    const loadCoreModel = async () => {
      // Only act on existing, non-custom core models
      if (!selectedModel || selectedModel.is_custom || !selectedModel.is_core) {
        return;
      }

      const safeModelKey = selectedModel.model_key || '';
      const safeDisplayName = selectedModel.display_name || '';

      // Guard against re-processing the same core model repeatedly on re-renders.
      // However, if the last processed kind was 'custom', we DO want to rebuild
      // the core model even if the keys match, to ensure the canvas reflects core.
      if (lastProcessedKeyRef.current === safeModelKey && lastProcessedKindRef.current === 'core') {
        return;
      }

      // 1) Prefer resolved_config from the catalog if available
      try {
        if (selectedModel.resolved_config) {
          const cfg = selectedModel.resolved_config as unknown;
          const now = Date.now();

          // Normalize either array or object-map into an array, and add builder fields
          const normEntries = (val: unknown, kind: 'dimension' | 'measure' | 'filter' | 'join') => {
            const arr: unknown[] = Array.isArray(val)
              ? (val as unknown[])
              : val && typeof val === 'object'
              ? Object.entries(asRecord(val) || {}).map(([name, v]: [string, unknown]) => ({ name, ...(v as Record<string, unknown> || {}) }))
              : [];
            return arr.map((e: unknown, idx: number) => {
              const rec = asRecord(e) || {};
              const baseId = `${kind}_${String(rec.name ?? idx)}_${now}`;
              return {
                  ...rec,
                  id: rec.id ?? baseId,
                  title: rec.title ?? rec.name ?? `${kind} ${idx}`,
                  sourceTable: rec.sourceTable ?? rec.table ?? '',
                  sourceColumn: rec.sourceColumn ?? rec.column ?? '',
                  is_custom: rec.is_custom ?? false,
                  isEditing: false,
                } as Record<string, unknown>;
            });
          };

          // Some configs come nested under a top-level "cubes" or a single "cube" structure
          const pickCube = (raw: unknown) => {
            if (!raw) return null;
            const r = asRecord(raw) || {};
            if (r.cubes && Array.isArray(r.cubes)) {
              const matchName = r.name ?? safeModelKey ?? safeDisplayName;
              const cubes = r.cubes as unknown[];
              const found = cubes.find((c: unknown) => asRecord(c)?.name === matchName) || cubes[0];
              return found ?? null;
            }
            if (r.cube && typeof r.cube === 'object') return raw;
            return raw;
          };

          const chosen: unknown = pickCube(cfg);
          const getField = (key: string) => {
            const ch = asRecord(chosen) || {};
            if (ch.cube && (ch as Record<string, unknown>)[key] === undefined) {
              return (ch as Record<string, unknown>)[key] ?? (asRecord(ch.cube) || {})[key];
            }
            return (ch as Record<string, unknown>)[key] ?? (asRecord(cfg) || {})[key];
          };

          const rawName = String((asRecord(cfg)?.name ?? asRecord(chosen)?.name ?? safeDisplayName ?? safeModelKey ?? 'semantic_model'));
          // Accept both filters and segments
          const rawFilters = getField('filters') ?? getField('segments');

          const normalized: SemanticModel = {
            name: rawName,
            // Narrowing at this boundary: cast to the SemanticModel element arrays explicitly
            dimensions: normEntries(getField('dimensions'), 'dimension') as unknown as SemanticModel['dimensions'],
            measures: normEntries(getField('measures'), 'measure') as unknown as SemanticModel['measures'],
            filters: normEntries(rawFilters, 'filter') as unknown as SemanticModel['filters'],
            joins: normEntries(getField('joins'), 'join') as unknown as SemanticModel['joins'],
            is_custom: false,
          };

          // If everything came back empty, log for visibility but still set so UI reflects name
          try {
            const d = normalized.dimensions?.length || 0;
            const m = normalized.measures?.length || 0;
            const f = normalized.filters?.length || 0;
            const j = normalized.joins?.length || 0;
            if (d + m + f + j === 0) {
              devWarn('[useCoreModelBuilder] resolved_config normalized to empty arrays; original keys:', Object.keys(cfg || {}));
            }
          } catch {}

          setSemanticModel(normalized);
          setLastLoadedModelKey(safeModelKey);
          lastProcessedKeyRef.current = safeModelKey;
          lastProcessedKindRef.current = 'core';
          return;
        }
      } catch (err) {
        // If normalization fails, fall through to metadata-based generation.
        devWarn('Failed to load core model from resolved_config; falling back to metadata.', err);
      }

      // 2) Fallback: synthesize a basic core model from datasource metadata
      try {
        const metadataResponse = await fetch(`/api/metadata/${datasourceId}`);
        if (!metadataResponse.ok) {
          throw new Error('Failed to fetch database metadata');
        }
        const metadata = await metadataResponse.json();

        const cols = Array.isArray((metadata as Record<string, unknown>)?.columns)
          ? ((metadata as Record<string, unknown>).columns as unknown[])
          : [];

        const dimensions = cols
          .map((col: unknown, index: number) => {
            const c = asRecord(col) || {};
            const colType = String(c.type ?? '');
            return { c, colType, index };
          })
          .filter(({ colType }) => colType === 'string' || colType === 'date')
          .map(({ c, index }) => {
            const name = String(c.name ?? `col_${index}`);
            const table = String(c.table ?? '');
            return {
              id: `dim_${name}_${Date.now()}`,
              name,
              title: name,
              type: String(c.type ?? 'string'),
              sql: `${table}.${name}`,
              sourceTable: table,
              sourceColumn: name,
              description: String(c.description ?? ''),
              isEditing: false,
              is_custom: false,
              isCore: true,
            };
          });

        const measures = cols
          .map((col: unknown, index: number) => {
            const c = asRecord(col) || {};
            const colType = String(c.type ?? '');
            return { c, colType, index };
          })
          .filter(({ colType }) => colType === 'number')
          .map(({ c, index }) => {
            const name = String(c.name ?? `col_${index}`);
            const table = String(c.table ?? '');
            return {
              id: `meas_${name}_${Date.now()}`,
              name,
              title: name,
              type: 'number',
              sql: `SUM(${table}.${name})`,
              sourceTable: table,
              sourceColumn: name,
              description: String(c.description ?? ''),
              isEditing: false,
              is_custom: false,
              isCore: true,
            };
          });

        setSemanticModel({
          name: safeDisplayName || safeModelKey || 'semantic_model',
          dimensions,
          measures,
          filters: [],
          joins: [],
          is_custom: false,
        });

  setLastLoadedModelKey(safeModelKey);
  lastProcessedKeyRef.current = safeModelKey;
  lastProcessedKindRef.current = 'core';
      } catch (error) {
        devError('Error building core model:', error);
        showNotification('Failed to build core model from database metadata', 'error');
      }
    };

    loadCoreModel();
  }, [
    selectedModel?.model_key,
    selectedModel?.is_core,
    selectedModel?.is_custom,
    // Trigger when a resolved config appears/disappears
    Boolean(selectedModel?.resolved_config),
    datasourceId,
  ]);
}
