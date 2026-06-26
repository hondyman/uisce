import { useCallback } from 'react';
import type { Dispatch, SetStateAction } from 'react';
import { replacePreAggregationPlaceholders, isPreAggregationNeeded, cleanupUnusedPreAggregations } from '../utils/semanticUtils';
import { getTableIdFromVal } from '../utils/tableHelpers';

const asRecord = (v: unknown): Record<string, unknown> | null => {
  if (v && typeof v === 'object' && !Array.isArray(v)) return v as Record<string, unknown>;
  return null;
};

const useElementCreator = ({ setSemanticModel, selectedColumn, ensureCustomAndApply }: { setSemanticModel: Dispatch<SetStateAction<any>>; selectedColumn: unknown | null; ensureCustomAndApply: (fn: (...args: unknown[]) => unknown, ...args: unknown[]) => unknown }) => {
  const createInner = useCallback((params: { mode: 'override' | 'custom'; kind: string; coreName?: string; values: unknown }, semanticModel: unknown) => {
    const { mode, kind, coreName: baseName, values } = params;
    const vals = asRecord(values) || {};
    const overrideTitle = mode === 'override' ? String(vals.title ?? '') : undefined;

    const id = `${kind}_${String(vals.name ?? '')}_${Date.now()}`;
    let newElement: Record<string, unknown>;
    let newPreAggregation: unknown = null;

    if (isPreAggregationNeeded(values, semanticModel)) {
      const { preAggregation, sqlPlaceholder } = replacePreAggregationPlaceholders(values, semanticModel, selectedColumn);
      newPreAggregation = preAggregation;
      // mutate vals for sql placeholder
      (vals as Record<string, unknown>).sql = sqlPlaceholder;
    }

    const serializedSourceTable = getTableIdFromVal((vals as Record<string, unknown>).sourceTable) || (newPreAggregation ? '<pre-aggregated>' : '');
    if (baseName) {
      newElement = { id, name: String(vals.name ?? ''), title: overrideTitle, description: String(vals.description ?? `Override for core ${kind}`), type: String(vals.type ?? (kind === 'measure' ? 'number' : 'string')), sql: String(vals.sql ?? ''), sourceTable: serializedSourceTable, sourceColumn: String(vals.sourceColumn ?? ''), format: vals.format, aggregationType: vals.aggregationType, defaultValue: vals.defaultValue, is_custom: true, isEditing: true, isNew: true, baseName };
    } else {
      newElement = { id, name: String(vals.name ?? ''), title: String(vals.title ?? vals.name), description: String(vals.description ?? `Custom ${kind}`), type: String(vals.type ?? (kind === 'measure' ? 'number' : 'string')), sql: String(vals.sql ?? ''), sourceTable: serializedSourceTable || '', sourceColumn: String(vals.sourceColumn ?? ''), format: vals.format, aggregationType: vals.aggregationType, defaultValue: vals.defaultValue, isEditing: true, isCore: false, isNew: true };
    }

    setSemanticModel((prev: unknown) => {
      const prevRec = asRecord(prev) || {};
      const newModel: Record<string, unknown> = { ...prevRec };
      const listKey = `${kind}s`;
      const prevList = Array.isArray(prevRec[listKey]) ? (prevRec[listKey] as unknown[]) : [];
      newModel[listKey] = [...prevList, newElement];
      if (newPreAggregation) {
        const prevPre = Array.isArray(prevRec.pre_aggregations) ? (prevRec.pre_aggregations as unknown[]) : [];
        newModel.pre_aggregations = [...prevPre, newPreAggregation];
      }
  return cleanupUnusedPreAggregations(newModel);
    });
  }, [setSemanticModel, selectedColumn]);

  const handleCreateElement = useCallback((params: { mode: 'override' | 'custom'; kind: string; coreName?: string; values: unknown }, semanticModel?: unknown) => {
    // Wrap creation with ensureCustomAndApply so consumers don't need to handle custom-model gating
    return ensureCustomAndApply(() => createInner(params, semanticModel), params);
  }, [ensureCustomAndApply, createInner]);

  return { handleCreateElement } as const;
};

export default useElementCreator;
