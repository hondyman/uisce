import { SemanticModelConfig, SemanticElement, CubeOptions } from './types';

export interface ValidationIssue {
  level: 'error' | 'warning';
  code: string;
  message: string;
  details?: any;
}

export interface PKFKRegistry {
  // cube -> primary key field(s)
  primaryKeys: Record<string, string[]>;
}

export const buildRegistry = (config: SemanticModelConfig, modelName: string): PKFKRegistry => {
  const corePKs = extractPKs(config.core.dimensions, config.core.options);
  const customPKs = extractPKs(config.custom.dimensions, config.custom.options);
  // final cube name is modelName
  const reg: PKFKRegistry = { primaryKeys: {} };
  reg.primaryKeys[`${modelName}_core`] = corePKs;
  reg.primaryKeys[modelName] = customPKs.length ? customPKs : corePKs;
  return reg;
};

const extractPKs = (dims: SemanticElement[], options?: CubeOptions): string[] => {
  const explicit = dims.filter(d => d.primary_key).map(d => d.name);
  if (explicit.length) return explicit;
  const listed = options?.governance?.pkFields || [];
  return listed;
};

export const validateCubeConfig = (
  config: SemanticModelConfig,
  modelName: string,
  registry?: PKFKRegistry
): ValidationIssue[] => {
  const issues: ValidationIssue[] = [];
  const reg = registry || buildRegistry(config, modelName);
  const finalPKs = reg.primaryKeys[modelName] || [];
  if (!finalPKs.length) {
    issues.push({ level: 'error', code: 'PK_MISSING', message: `Primary key not defined for cube '${modelName}'.` });
  }
  // Validate joins declare semantic relationship and look like PK->FK
  const allJoins: SemanticElement[] = [
    ...(config.core.joins || []),
    ...(config.custom.joins || [])
  ];
  const allDimensionNames = new Set<string>([...config.core.dimensions, ...config.custom.dimensions].map(d => d.name));
  for (const j of allJoins) {
    if (!j.relationship) {
      issues.push({ level: 'warning', code: 'JOIN_RELATIONSHIP_MISSING', message: `Join '${j.name}' missing relationship; expected many_to_one for fact->dimension.` });
    }
    // basic check: sql contains equality between fields
    if (!j.sql || !/=/.test(j.sql)) {
      issues.push({ level: 'warning', code: 'JOIN_SQL_WEAK', message: `Join '${j.name}' SQL may be incomplete.` });
    }
    // FK field existence: try to detect CUBE field on left side like {CUBE.some_id}
    const match = j.sql?.match(/\{CUBE\.([a-zA-Z0-9_]+)\}/);
    if (match) {
      const fkField = match[1];
      if (!allDimensionNames.has(fkField)) {
        issues.push({ level: 'error', code: 'FK_FIELD_MISSING', message: `Join '${j.name}' references field '${fkField}' not present in dimensions.` });
      }
      // Naming consistency lint: prefer *_id for foreign keys
      if (!/_id$/.test(fkField)) {
        issues.push({ level: 'warning', code: 'NAMING_INCONSISTENT', message: `Field '${fkField}' in join '${j.name}' should end with '_id' for consistency.` });
      }
    }
    // If we know PKs of the target, warn if FK name does not align to any target PK hint
    const targetPKs = reg.primaryKeys[j.name] || [];
    if (targetPKs.length && match) {
      const fkField = match[1];
      const normalized = targetPKs.some(pk => pk.replace(/_id$/, '') === fkField.replace(/_id$/, ''));
      if (!normalized) {
        issues.push({ level: 'warning', code: 'PK_FK_MISMATCH', message: `FK '${fkField}' in join '${j.name}' does not align with target PK(s) [${targetPKs.join(', ')}].` });
      }
    }
  }
  return issues;
};

export const generateCompanionGovernanceJSON = (config: SemanticModelConfig, modelName: string): string => {
  const coreGov = config.core.options?.governance || {};
  const customGov = config.custom.options?.governance || {};
  const data = {
    cube: modelName,
    core: {
      pkFields: config.core.options?.governance?.pkFields || [],
      pkOrigin: coreGov.pkOrigin || {},
      steward: coreGov.steward,
      pii: coreGov.pii,
      lineage: coreGov.lineage,
      audit_fields: coreGov.audit_fields || [],
    },
    final: {
      pkFields: config.custom.options?.governance?.pkFields || config.core.options?.governance?.pkFields || [],
      pkOrigin: customGov.pkOrigin || coreGov.pkOrigin || {},
      steward: customGov.steward || coreGov.steward,
      pii: typeof customGov.pii === 'boolean' ? customGov.pii : coreGov.pii,
      lineage: customGov.lineage || coreGov.lineage,
      audit_fields: customGov.audit_fields || coreGov.audit_fields || [],
    },
    joins: [
      ...(config.core.joins || []).map(j => ({ cube: `${modelName}_core`, name: j.name, sql: j.sql, relationship: j.relationship, meta: j.meta || {} })),
      ...(config.custom.joins || []).map(j => ({ cube: modelName, name: j.name, sql: j.sql, relationship: j.relationship, meta: j.meta || {} })),
    ]
  };
  return JSON.stringify(data, null, 2);
};
