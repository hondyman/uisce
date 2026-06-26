import * as yaml from 'js-yaml';
import { SemanticModelConfig, ColumnInfo, SemanticElement, CubeOptions } from './types';
import { devLog } from '../../../utils/devLogger';

export const toast = (options: { title: string; description: string; variant?: string }) => {
  devLog(`Toast: ${options.title} - ${options.description}`);
};

export const copyContent = (content: string, type: string, toast: (options: { title: string; description: string; variant?: string }) => void) => {
  navigator.clipboard.writeText(content);
  toast({
    title: `${type} Copied`,
    description: `The ${type.toLowerCase()} content has been copied to your clipboard.`,
  });
};

export const downloadContent = (content: string, filename: string, type: string, toast: (options: { title: string; description: string; variant?: string }) => void) => {
  const blob = new Blob([content], { type: type === 'JSON' ? 'application/json' : 'text/yaml' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
  
  toast({
    title: `${type} Downloaded`,
    description: `${filename} has been downloaded.`,
  });
};

export const isNumericType = (type: string) => {
  return type.includes('int') || type.includes('float') || type.includes('decimal') || type.includes('numeric');
};

export const getColumnType = (dbType: string) => {
  if (dbType.includes('int') || dbType.includes('serial')) return 'number';
  if (dbType.includes('float') || dbType.includes('decimal') || dbType.includes('numeric')) return 'number';
  if (dbType.includes('date') || dbType.includes('timestamp')) return 'time';
  if (dbType.includes('bool')) return 'boolean';
  return 'string';
};

export const addDimensionFromColumn = (
  table: string, 
  column: ColumnInfo, 
  isCore: boolean, 
  setConfig: React.Dispatch<React.SetStateAction<SemanticModelConfig>>, 
  toast: (options: { title: string; description: string; variant?: string }) => void
) => {
  const newDimension: SemanticElement = {
    id: `${table}_${column.column_name}_${Date.now()}`,
    name: `${table}_${column.column_name}`,
    title: column.column_name.replace(/_/g, ' ').replace(/\b\w/g, (l: string) => l.toUpperCase()),
    sourceTable: table,
    sourceColumn: column.column_name,
    type: getColumnType(column.data_type),
    sql: column.column_name,
    public: true,
    description: `Dimension from ${table}.${column.column_name}`
  };
  
  setConfig((prev: SemanticModelConfig) => ({
    ...prev,
    [isCore ? 'core' : 'custom']: {
      ...prev[isCore ? 'core' : 'custom'],
      dimensions: [...prev[isCore ? 'core' : 'custom'].dimensions, newDimension]
    }
  }));
  
  toast({
    title: "Dimension Added",
    description: `Added ${isCore ? 'core' : 'custom'} dimension: ${newDimension.title}`,
  });
};

export const addMeasureFromColumn = (
  table: string, 
  column: ColumnInfo, 
  isCore: boolean, 
  setConfig: React.Dispatch<React.SetStateAction<SemanticModelConfig>>, 
  toast: (options: { title: string; description: string; variant?: string }) => void
) => {
  if (!isNumericType(column.data_type)) {
    toast({
      title: "Invalid Column Type",
      description: "Measures can only be created from numeric columns.",
      variant: "destructive"
    });
    return;
  }

  const newMeasure: SemanticElement = {
    id: `${table}_${column.column_name}_sum_${Date.now()}`,
    name: `${table}_${column.column_name}_sum`,
    title: `Total ${column.column_name.replace(/_/g, ' ').replace(/\b\w/g, (l: string) => l.toUpperCase())}`,
    sourceTable: table,
    sourceColumn: column.column_name,
    type: 'sum',
    sql: column.column_name,
    public: true,
    description: `Sum of ${table}.${column.column_name}`
  };
  
  setConfig((prev: SemanticModelConfig) => ({
    ...prev,
    [isCore ? 'core' : 'custom']: {
      ...prev[isCore ? 'core' : 'custom'],
      measures: [...prev[isCore ? 'core' : 'custom'].measures, newMeasure]
    }
  }));
  
  toast({
    title: "Measure Added",
    description: `Added ${isCore ? 'core' : 'custom'} measure: ${newMeasure.title}`,
  });
};

export const generateCoreYAML = (config: SemanticModelConfig, modelName: string) => {
  // Helper: ensure a single primary key; synthesize if composite is specified
  const enforcePrimaryKey = (dims: SemanticElement[], options?: CubeOptions, tableName?: string): SemanticElement[] => {
  const out = dims.map(d => ({ ...d }));
    const pkDims = out.filter(d => d.primary_key);
    const pkFields = options?.governance?.pkFields || [];
    if (pkFields.length > 1) {
      // remove existing PK flags
      out.forEach(d => { if (d.primary_key) delete d.primary_key; });
      // synthesize composite PK dimension
      const synthName = 'pk';
      const synth: SemanticElement = {
        id: `${tableName || modelName}_pk`,
        name: synthName,
        title: 'Primary Key',
        sourceTable: tableName || modelName,
        sourceColumn: synthName,
        type: 'string',
        sql: `CONCAT_WS('-', ${pkFields.map(f => `{CUBE}.${f}`).join(', ')})`,
        public: false,
        description: 'Synthetic composite primary key',
        primary_key: true,
        meta: {
          composite: true,
          fields: pkFields,
          origin: options?.governance?.pkOrigin || {},
        }
      };
      out.unshift(synth);
      return out;
    }
    if (pkDims.length === 0 && pkFields.length === 1) {
      // set the single listed pk field as primary_key
      const idx = out.findIndex(d => d.name === pkFields[0]);
      if (idx >= 0) {
        out[idx] = { ...out[idx], primary_key: true, meta: { ...(out[idx].meta||{}), origin: options?.governance?.pkOrigin || {} } };
      }
    } else if (pkDims.length > 1) {
      // if multiple marked, keep the first and unset the rest for consistency
      pkDims.slice(1).forEach(d => { d.primary_key = false; });
    }
    return out;
  };

  const coreConfig: any = {
    name: `${modelName || "untitled_model"}_core`,
    sql_table: `public.${modelName?.toLowerCase() || 'table'}`,
    dimensions: enforcePrimaryKey(config.core.dimensions, config.core.options, modelName).map((dim: SemanticElement) => {
      const dimSql = (dim.sql.includes('{CUBE}') || /[()\s]/.test(dim.sql)) ? dim.sql : `{CUBE}.${dim.sql}`;
      return ({
        name: dim.name,
        type: dim.type,
        sql: dimSql,
        title: dim.title || dim.name,
        meta: { core: true },
        ...(dim.primary_key ? { primary_key: true } : {}),
        ...(dim.description && { description: dim.description })
      });
    }),
    measures: config.core.measures.map((measure: SemanticElement) => ({
      name: measure.name,
      type: measure.type,
      sql: measure.sql.includes('{CUBE}') ? measure.sql : `{CUBE}.${measure.sql}`,
      title: measure.title || measure.name,
      meta: { core: true },
      ...(measure.description && { description: measure.description })
    }))
  };
  // Pass-through optional cube-level options for core
  const coreOpts = config.core.options || {} as any;
  if (coreOpts.title) coreConfig.title = coreOpts.title;
  if (coreOpts.description) coreConfig.description = coreOpts.description;
  if (typeof coreOpts.public === 'boolean') coreConfig.public = coreOpts.public;
  if (coreOpts.meta) coreConfig.meta = coreOpts.meta;
  if (coreOpts.refresh_key) coreConfig.refresh_key = coreOpts.refresh_key;
  if (coreOpts.access_policy) coreConfig.access_policy = coreOpts.access_policy;
  if (coreOpts.segments && coreOpts.segments.length) coreConfig.segments = coreOpts.segments;
  if (coreOpts.hierarchies && coreOpts.hierarchies.length) coreConfig.hierarchies = coreOpts.hierarchies;
  if (coreOpts.sql_alias) coreConfig.sql_alias = coreOpts.sql_alias;
  if (coreOpts.data_source) coreConfig.data_source = coreOpts.data_source;
  if (coreOpts.pre_aggregations) coreConfig.pre_aggregations = coreOpts.pre_aggregations;
  if (coreOpts.extends) coreConfig.extends = coreOpts.extends;
  // Auto title/description if missing
  if (!coreConfig.title) coreConfig.title = `${(modelName || 'Model').replace(/_/g,' ')} Core`;
  if (!coreConfig.description) coreConfig.description = `Core dimensions and measures for ${modelName}`;
  // Default refresh_key hint if not set and updated_at exists
  if (!coreConfig.refresh_key && config.core.dimensions.some(d => d.name === 'updated_at' || d.sql === 'updated_at')) {
    coreConfig.refresh_key = { sql: `SELECT MAX(updated_at) FROM public.${modelName?.toLowerCase() || 'table'}` };
  }
  
  if (config.core.joins && config.core.joins.length > 0) {
    const tenantField = coreOpts?.governance?.tenantField;
    coreConfig.joins = config.core.joins.map((join: SemanticElement) => {
      let sql = join.sql;
      if (tenantField && !/tenant_id|\{CUBE\.[^}]*tenant/i.test(sql)) {
        sql = `${sql} AND {CUBE.${tenantField}} = {${join.name}.${tenantField}}`;
      }
      return ({
        name: join.name,
        sql,
        relationship: join.relationship,
        meta: { core: true, ...(join.meta||{}) }
      });
    });
  }
  
  // Ensure sql_table and sql fields use period as separator (schema.table)
  if (coreConfig.sql_table && typeof coreConfig.sql_table === 'string' && coreConfig.sql_table.includes('/')) {
    coreConfig.sql_table = coreConfig.sql_table.replace(/^\/?/, '').replace('/', '.');
  }
  if (coreConfig.sql && typeof coreConfig.sql === 'string' && coreConfig.sql.includes('/')) {
    coreConfig.sql = coreConfig.sql.replace(/^\/?/, '').replace('/', '.');
  }
  return yaml.dump(coreConfig, { 
    flowLevel: -1,
    styles: {
      '!!null': 'empty'
    }
  });
};

export const generateFinalYAML = (config: SemanticModelConfig, modelName: string) => {
  const allDimensions = [
    ...config.core.dimensions.map((d: SemanticElement) => ({ ...d, meta: { core: true } })),
    ...config.custom.dimensions.map((d: SemanticElement) => ({ ...d, meta: { custom: true } }))
  ];
  
  const allMeasures = [
    ...config.core.measures.map((m: SemanticElement) => ({ ...m, meta: { core: true } })),
    ...config.custom.measures.map((m: SemanticElement) => ({ ...m, meta: { custom: true } }))
  ];
  
  const allJoins = [
    ...(config.core.joins || []).map((j: SemanticElement) => ({ ...j, meta: { core: true } })),
    ...(config.custom.joins || []).map((j: SemanticElement) => ({ ...j, meta: { custom: true } }))
  ];

  const finalDimensions = allDimensions.map((dim: SemanticElement & { meta: any }) => {
    const override = config.custom.overrides?.dimensions?.[dim.name];
    return override ? { ...dim, ...override } : dim;
  });

  const finalMeasures = allMeasures.map((measure: SemanticElement & { meta: any }) => {
    const override = config.custom.overrides?.measures?.[measure.name];
    return override ? { ...measure, ...override } : measure;
  });

  // Ensure sql_table uses period as separator (schema.table)
  const schemaTable = (modelName && modelName.includes('/'))
    ? modelName.replace(/^\/?/, '').replace('/', '.')
    : `public.${modelName?.toLowerCase() || 'table'}`;
  const finalConfig: any = {
    name: modelName || "untitled_model",
    sql_table: schemaTable,
    extends: `${modelName || "untitled_model"}_core`,
    dimensions: finalDimensions.map((dim: SemanticElement & { meta: any }) => {
      const dimSql = (dim.sql.includes('{CUBE}') || /[()\s]/.test(dim.sql)) ? dim.sql : `{CUBE}.${dim.sql}`;
      return ({
        name: dim.name,
        type: dim.type,
        sql: dimSql,
        title: dim.title || dim.name,
        meta: dim.meta,
        ...(dim.primary_key ? { primary_key: true } : {}),
        ...(dim.description && { description: dim.description })
      });
    }),
    measures: finalMeasures.map((measure: SemanticElement & { meta: any }) => ({
      name: measure.name,
      type: measure.type,
      sql: measure.sql.includes('{CUBE}') ? measure.sql : `{CUBE}.${measure.sql}`,
      title: measure.title || measure.name,
      meta: measure.meta,
      ...(measure.description && { description: measure.description })
    }))
  };
  // Pass-through optional cube-level options for final cube (custom overrides)
  const finOpts = (config.custom.options || {}) as any;
  if (finOpts.title) finalConfig.title = finOpts.title;
  if (finOpts.description) finalConfig.description = finOpts.description;
  if (typeof finOpts.public === 'boolean') finalConfig.public = finOpts.public;
  if (finOpts.meta) finalConfig.meta = finOpts.meta;
  if (finOpts.refresh_key) finalConfig.refresh_key = finOpts.refresh_key;
  if (finOpts.access_policy) finalConfig.access_policy = finOpts.access_policy;
  if (finOpts.segments && finOpts.segments.length) finalConfig.segments = finOpts.segments;
  if (finOpts.hierarchies && finOpts.hierarchies.length) finalConfig.hierarchies = finOpts.hierarchies;
  if (finOpts.sql_alias) finalConfig.sql_alias = finOpts.sql_alias;
  if (finOpts.data_source) finalConfig.data_source = finOpts.data_source;
  if (finOpts.pre_aggregations) finalConfig.pre_aggregations = finOpts.pre_aggregations;
  if (finOpts.extends) finalConfig.extends = finOpts.extends;
  // Auto title/description if missing
  if (!finalConfig.title) finalConfig.title = (modelName || 'Model').replace(/_/g,' ');
  if (!finalConfig.description) finalConfig.description = `Final cube for ${modelName}`;
  // Default refresh_key hint if not set and updated_at exists
  if (!finalConfig.refresh_key && finalDimensions.some(d => d.name === 'updated_at' || d.sql === 'updated_at')) {
    finalConfig.refresh_key = { sql: `SELECT MAX(updated_at) FROM public.${modelName?.toLowerCase() || 'table'}` };
  }
  
  if (allJoins.length > 0) {
    const tenantField = finOpts?.governance?.tenantField || config.core.options?.governance?.tenantField;
    finalConfig.joins = allJoins.map((join: SemanticElement & { meta: any }) => {
      let sql = join.sql;
      if (tenantField && !/tenant_id|\{CUBE\.[^}]*tenant/i.test(sql)) {
        sql = `${sql} AND {CUBE.${tenantField}} = {${join.name}.${tenantField}}`;
      }
      return ({
        name: join.name,
        sql,
        relationship: join.relationship,
        meta: join.meta
      });
    });
  }
  
  return yaml.dump(finalConfig, { 
    flowLevel: -1,
    styles: {
      '!!null': 'empty'
    }
  });
};

// Optional TOML generator for multi-protocol readiness
export const toTOML = (obj: any, indent = 0): string => {
  const pad = '  '.repeat(indent);
  if (Array.isArray(obj)) {
    return obj.map(item => toTOML(item, indent)).join('\n');
  }
  if (obj && typeof obj === 'object') {
    let out = '';
    for (const [key, val] of Object.entries(obj)) {
      if (Array.isArray(val) && val.length && typeof val[0] === 'object') {
        for (const el of val as any[]) {
          out += `\n${pad}[[${key}]]\n` + toTOML(el, indent + 0);
        }
      } else if (val && typeof val === 'object') {
        out += `\n${pad}[${key}]\n` + toTOML(val, indent + 0);
      } else {
        const scalar = typeof val === 'string' ? `"${val}"` : val;
        out += `${pad}${key} = ${scalar}\n`;
      }
    }
    return out;
  }
  return String(obj);
};