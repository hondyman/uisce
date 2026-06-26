import yaml from 'js-yaml';

// Parse JSON/JSONC/YAML into a normalized "custom-only" semantic model.
// Uses jsonc-parser when available for lenient JSONC parsing and robust AST
// handling. Returns null on parse failure.
export const parseCodeToCustomModel = (text: string, format: 'json' | 'yaml' | 'jsonc' | null, modelName?: string) => {
  try {
    let obj: any = null;
    if (format === 'yaml') {
      obj = yaml.load(text);
    } else {
      // Try jsonc-parser if available (supports comments/trailing commas)
      try {
        // eslint-disable-next-line @typescript-eslint/no-var-requires
        const { parse } = require('jsonc-parser');
        obj = parse(text);
      } catch (e) {
        obj = JSON.parse(text);
      }
    }
    if (!obj || typeof obj !== 'object') return null;

    // Prefer explicit cube if present
    const pickCube = (raw: any) => {
      if (!raw) return null;
      if (Array.isArray(raw.cubes) && raw.cubes.length) return raw.cubes[0];
      if (raw.cube) return raw.cube;
      // If top-level looks like a cube (has measures/dimensions) return raw
      if (raw.measures || raw.dimensions || raw.filters || raw.joins) return raw;
      return null;
    };

    const chosen = pickCube(obj) || {};

    const normEntries = (items: any) => {
      if (!items) return [];
      // accept array or object map
  if (Array.isArray(items)) return items.map((it: any, _idx: number) => ({ id: it.id || `imported_${_idx}_${Date.now()}`, is_custom: true, ...it }));
  if (typeof items === 'object') return Object.entries(items).map(([k, v]: any, _idx: number) => ({ id: (v && v.id) || `imported_${k}_${Date.now()}`, is_custom: true, name: (v && v.name) || k, ...(v || {}) }));
      return [];
    };

    const measures = normEntries(chosen.measures || obj.measures);
    const dimensions = normEntries(chosen.dimensions || obj.dimensions);
    const filters = normEntries(chosen.filters || obj.filters || obj.segments);
    const joins = normEntries(chosen.joins || obj.joins);

    const customModel: any = {
      name: chosen.name || obj.name || modelName || 'semantic_model',
      measures,
      dimensions,
      filters,
      joins,
      is_custom: true,
    };
    return customModel;
  } catch (e) {
    return null;
  }
};

export default parseCodeToCustomModel;
