import { useCallback } from 'react';
import yaml from 'js-yaml';

const asRecord = (v: unknown): Record<string, unknown> | null => {
  if (v && typeof v === 'object' && !Array.isArray(v)) return v as Record<string, unknown>;
  return null;
};

// Exported pure helper for computing the custom diff vs. core
// Inputs:
// - selectedModel: expects { is_custom: boolean, parent_model_key?: string }
// - parentResolvedConfig: the resolved core config object (Cube.js-like shape)
// - currentConfig: the current in-editor semantic model object (normalized arrays)
// Output: minimal object containing only changed/new fields and an `extends` key
export function computeCustomModelDiff(params: {
  selectedModel: { is_custom?: boolean; parent_model_key?: string } | null | undefined;
  parentResolvedConfig: unknown;
  currentConfig: unknown;
}) {
  const { selectedModel, parentResolvedConfig, currentConfig } = params;
  if (!selectedModel || !selectedModel.is_custom || !selectedModel.parent_model_key) return null;

  const parentConfigRaw = parentResolvedConfig;

  // Helpers: canonicalize objects (sorted keys), strip UI-only fields, and deep-compare
  const stripUiFields = (obj: unknown) => {
    if (!obj || typeof obj !== 'object') return obj;
    const rec = asRecord(obj) || {};
    const { id, isEditing, is_custom, drillMembers, metadata, can_edit, core_model_exists, custom_model_exists, ...rest } = rec;
    return rest;
  };
  const sortKeys = (val: any): any => {
    if (Array.isArray(val)) return val.map(sortKeys);
    if (val && typeof val === 'object') {
      const out: any = {};
      for (const k of Object.keys(val).sort()) out[k] = sortKeys(val[k]);
      return out;
    }
    return val;
  };
  const canonicalize = (val: unknown) => sortKeys(stripUiFields(val));
  const deepEqual = (a: unknown, b: unknown) => {
    try { return JSON.stringify(canonicalize(a)) === JSON.stringify(canonicalize(b)); } catch { return false; }
  };

  // Minimal allowed keys per type to avoid leaking UI/derived fields
  const pickAllowed = (obj: unknown, type: string) => {
    const common = ['name', 'title', 'description'];
    const baseSrc = ['sourceTable', 'sourceColumn', 'sql'];
    const byType: Record<string, string[]> = {
      dimension: [...common, ...baseSrc, 'type', 'format', 'public', 'drillMembers'],
      measure: [...common, ...baseSrc, 'type', 'format', 'public', 'aggregationType', 'filters', 'drillMembers'],
      filter: [...common, ...baseSrc, 'type', 'values', 'operator'],
      join: ['name', 'leftTable', 'rightTable', 'sql', 'relationship', 'joinType', 'description'],
      pre_aggregation: ['name', 'type', 'measures', 'dimensions', 'timeDimension', 'granularity', 'partitionGranularity', 'scheduledRefresh', 'indexes', 'refreshKey', 'sql', 'description'],
    };
    const rec = asRecord(obj) || {};
    const allowed = byType[type] || Object.keys(rec);
    const out: Record<string, unknown> = {};
    for (const k of allowed) {
      if (rec && (rec as Record<string, unknown>)[k] !== undefined) out[k] = (rec as Record<string, unknown>)[k];
    }
    return out;
  };

  // Normalize possible shapes: arrays, object-maps, nested cube(s)
  const pickCube = (raw: unknown) => {
    if (!raw) return null;
    const r = asRecord(raw) || {};
    if (r.cubes && Array.isArray(r.cubes)) {
      const matchName = r.name || selectedModel.parent_model_key;
      const cubes = r.cubes as unknown[];
      const found = cubes.find((c: unknown) => asRecord(c)?.name === matchName);
      return found ?? cubes[0] ?? raw;
    }
    if (r.cube && typeof r.cube === 'object') return raw;
    return raw;
  };
  const parentChosen: unknown = pickCube(parentConfigRaw) || {};
  const getParentField = (key: string) => {
    const pc = asRecord(parentChosen) || {};
    if (pc.cube && (pc as Record<string, unknown>)[key] === undefined) {
      return (pc as Record<string, unknown>)[key] ?? (asRecord(pc.cube) || {})[key];
    }
    return (pc as Record<string, unknown>)[key] ?? (asRecord(parentConfigRaw) || {})[key];
  };

  const toArray = (val: unknown) => {
    if (Array.isArray(val)) return val as unknown[];
    if (val && typeof val === 'object') return Object.entries(asRecord(val) || {}).map(([name, v]: [string, unknown]) => ({ name, ...((v as Record<string, unknown>) || {}) }));
    return [] as unknown[];
  };

  const mkDeltaList = (currList: unknown[], parentList: unknown[], type: string) => {
    const pList = parentList || [];
    const pMap = new Map<string, unknown>(pList.map((p: unknown) => [String(asRecord(p)?.name ?? ''), p]));
    const out: Array<Record<string, unknown>> = [];
    for (const curr of currList || []) {
      const cleanCurr = stripUiFields(curr);
      const name = String(asRecord(curr)?.name ?? '');
      const parent = pMap.get(name);
      if (!parent) {
        out.push(pickAllowed(cleanCurr, type) as Record<string, unknown>);
        continue;
      }
      const allowedCurr = pickAllowed(cleanCurr, type);
      const allowedParent = pickAllowed(stripUiFields(parent), type);
      const diffItem: Record<string, unknown> = { name };
      let changed = false;
      const keys = Array.from(new Set([...Object.keys(allowedCurr), ...Object.keys(allowedParent)]));
      for (const k of keys) {
        if (k === 'name') continue;
        const vCurr = (allowedCurr as Record<string, unknown>)[k];
        const vParent = (allowedParent as Record<string, unknown>)[k];
        if (!deepEqual(vCurr, vParent)) { diffItem[k] = vCurr; changed = true; }
      }
      if (changed) out.push(diffItem);
    }
    return out;
  };

  const diff: any = {
    extends: selectedModel.parent_model_key,
  };

  const parentDimensionsArr = toArray(getParentField('dimensions'));
  const customDimensions = mkDeltaList(asRecord(currentConfig)?.dimensions as unknown[] || [], parentDimensionsArr, 'dimension');
  if (customDimensions.length > 0) diff.dimensions = customDimensions;

  const parentMeasuresArr = toArray(getParentField('measures'));
  const customMeasures = mkDeltaList(asRecord(currentConfig)?.measures as unknown[] || [], parentMeasuresArr, 'measure');
  // above line keeps behavior but avoids broad any by favoring record access; if no measures found, fallback to currentConfig?.measures via any
  if (customMeasures.length > 0) diff.measures = customMeasures;

  const parentFiltersArr = toArray(getParentField('filters') ?? getParentField('segments'));
  const customFilters = mkDeltaList(asRecord(currentConfig)?.filters as unknown[] || [], parentFiltersArr, 'filter');
  if (customFilters.length > 0) diff.filters = customFilters;

  const parentJoinsArr = toArray(getParentField('joins'));
  const customJoins = mkDeltaList(asRecord(currentConfig)?.joins as unknown[] || [], parentJoinsArr, 'join');
  if (customJoins.length > 0) diff.joins = customJoins;

  const parentPreAggArr = toArray(getParentField('pre_aggregations'));
  const customPreAgg = mkDeltaList(asRecord(currentConfig)?.pre_aggregations as unknown[] || [], parentPreAggArr, 'pre_aggregation');
  if (customPreAgg.length > 0) (diff as Record<string, unknown>).pre_aggregations = customPreAgg;

  return diff;
}

// Helper hook that returns generator functions for the builder.
export const useBuilderGenerators = (params: {
  selectedModel: { is_custom?: boolean; parent_model_key?: string } | null;
  catalogModels: Array<{ model_key?: string; resolved_config?: unknown }>;
  semanticModel: unknown;
  rawGenerateJSON: () => string;
  rawGenerateYAML: () => string;
}) => {
  const { selectedModel, catalogModels, semanticModel, rawGenerateJSON, rawGenerateYAML } = params;

  const generateCustomModelObject = useCallback(() => {
    if (!selectedModel || !selectedModel.is_custom || !selectedModel.parent_model_key) return null;
    const parentModel = catalogModels.find((m: any) => m.model_key === selectedModel.parent_model_key);
    if (!parentModel || !parentModel.resolved_config) return null;

    return computeCustomModelDiff({
      selectedModel,
      parentResolvedConfig: parentModel.resolved_config,
      currentConfig: semanticModel,
    });
  }, [selectedModel, catalogModels, semanticModel]);

  const generateMergedModelObject = useCallback(() => {
    if (!selectedModel || !selectedModel.is_custom || !selectedModel.parent_model_key) return semanticModel;
    const parentModel = catalogModels.find((m: any) => m.model_key === selectedModel.parent_model_key);
  if (!parentModel || !parentModel.resolved_config) return semanticModel;

  const parentConfigRaw = parentModel.resolved_config;
    const pickCube = (raw: any) => {
      if (!raw) return null;
      if (raw.cubes && Array.isArray(raw.cubes)) {
        const matchName = raw.name || selectedModel.parent_model_key;
        return raw.cubes.find((c: any) => c?.name === matchName) || raw.cubes[0] || raw;
      }
      if (raw.cube && typeof raw.cube === 'object') return raw;
      return raw;
    };
    const parentChosen: unknown = pickCube(parentConfigRaw) || {};
    const getParentField = (key: string) => {
      const pc = asRecord(parentChosen) || {};
      if (pc.cube && (pc as Record<string, unknown>)[key] === undefined) {
        return (pc as Record<string, unknown>)[key] ?? (asRecord(pc.cube) || {})[key];
      }
      return (pc as Record<string, unknown>)[key] ?? (asRecord(parentConfigRaw) || {})[key];
    };
    const toArray = (val: unknown) => {
      if (Array.isArray(val)) return val as unknown[];
      if (val && typeof val === 'object') return Object.entries(asRecord(val) || {}).map(([name, v]: [string, unknown]) => ({ name, ...((v as Record<string, unknown>) || {}) }));
      return [] as unknown[];
    };

    const mergeByName = (parentList: unknown[] = [], currList: unknown[] = []) => {
      const map = new Map<string, Record<string, unknown>>();
      for (const p of parentList || []) {
        const rec = asRecord(p) || {};
        map.set(String(rec.name ?? ''), rec);
      }
      for (const c of currList || []) {
        const rec = asRecord(c) || {};
        const key = String(rec.name ?? '');
        map.set(key, { ...(map.get(key) || {}), ...rec });
      }
      return Array.from(map.values());
    };

    return {
      name: asRecord(semanticModel)?.name,
      dimensions: mergeByName(toArray(getParentField('dimensions')), asRecord(semanticModel)?.dimensions as unknown[] || []),
      measures: mergeByName(toArray(getParentField('measures')), asRecord(semanticModel)?.measures as unknown[] || []),
      filters: mergeByName(toArray(getParentField('filters') ?? getParentField('segments')), asRecord(semanticModel)?.filters as unknown[] || []),
      joins: mergeByName(toArray(getParentField('joins')), asRecord(semanticModel)?.joins as unknown[] || []),
      ...(asRecord(semanticModel)?.pre_aggregations ? { pre_aggregations: mergeByName(toArray(getParentField('pre_aggregations')), asRecord(semanticModel)?.pre_aggregations as unknown[] || []) } : {}),
    };
  }, [selectedModel, catalogModels, semanticModel]);

  const generateJSON = useCallback(() => {
    try { return rawGenerateJSON(); } catch { return '{}'; }
  }, [rawGenerateJSON]);

  const generateYAML = useCallback(() => {
    try { return rawGenerateYAML(); } catch { return ''; }
  }, [rawGenerateYAML]);

  const generateCoreJSON = useCallback(() => {
    if (!selectedModel || !selectedModel.parent_model_key) return generateJSON();
    const parent = catalogModels.find((m: any) => m.model_key === selectedModel.parent_model_key);
    return parent && parent.resolved_config ? JSON.stringify(parent.resolved_config, null, 2) : generateJSON();
  }, [selectedModel, catalogModels, generateJSON]);

  const generateCoreYAML = useCallback(() => {
    if (!selectedModel || !selectedModel.parent_model_key) return generateYAML();
    const parent = catalogModels.find((m: any) => m.model_key === selectedModel.parent_model_key);
    return parent && parent.resolved_config ? yaml.dump(parent.resolved_config) : generateYAML();
  }, [selectedModel, catalogModels, generateYAML]);

  const generateCustomJSON = useCallback(() => {
    const obj = generateCustomModelObject();
    return obj ? JSON.stringify(obj, null, 2) : '{}';
  }, [generateCustomModelObject]);

  const generateCustomYAML = useCallback(() => {
    const obj = generateCustomModelObject();
    return obj ? yaml.dump(obj) : '';
  }, [generateCustomModelObject]);

  return {
    generateCustomModelObject,
    generateMergedModelObject,
    generateJSON,
    generateYAML,
    generateCoreJSON,
    generateCoreYAML,
    generateCustomJSON,
    generateCustomYAML,
  } as const;
};
