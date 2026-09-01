import type { FieldDefinition } from './subtypeFieldResolver';

export interface BORelationship {
  id: string;
  key: string;
  name: string;
  sourceTable: string;
  satelliteJoinCondition?: string | null;
  targetBoKey: string;
  targetSubtypeKey?: string | null;
  targetTable: string;
  joinType: 'INNER' | 'LEFT' | 'RIGHT';
  joinCondition: string;
  scopedSubtypeKey?: string | null;
}

export interface SubtypeDefinition {
  id: string;
  key: string;
  name: string;
  displayName: string;
  technicalName: string;
  subtypeFields: FieldDefinition[];
  isCore?: boolean;
  basedOnEntity?: string;
  relationshipAllowlist?: string[];
}

export interface BusinessObject {
  id: string;
  key: string;
  name: string;
  displayName: string;
  coreFields?: FieldDefinition[];
  customFields?: FieldDefinition[];
  config?: {
    fields?: FieldDefinition[];
  };
  subtypes?: Record<string, SubtypeDefinition>;
  relationships?: BORelationship[];
}

export interface EffectiveQueryScope {
  boKey: string;
  selectedSubtypeKey: string | null;
  discriminatorClause: string | null;
  fields: FieldDefinition[];
  relationships: BORelationship[];
}

function extractRootFields(bo: BusinessObject): FieldDefinition[] {
  if (bo.customFields && bo.customFields.length > 0) return bo.customFields;
  if (bo.config?.fields && bo.config.fields.length > 0) return bo.config.fields;
  return [...(bo.coreFields || []), ...(bo.customFields || [])];
}

function deduplicateFields(fields: FieldDefinition[]): FieldDefinition[] {
  const seen = new Set<string>();
  const out: FieldDefinition[] = [];
  for (const f of fields) {
    const id = f.key || f.technicalName || f.id;
    if (!id || seen.has(id)) continue;
    seen.add(id);
    out.push({ ...f, scope: 'ASSIGNED' });
  }
  return out;
}

export function resolveEffectiveQueryScope(
  bo: BusinessObject,
  selectedSubtypeKey: string | null
): EffectiveQueryScope {
  const allRelationships = bo.relationships || [];
  const rootFields = deduplicateFields(extractRootFields(bo));

  if (!selectedSubtypeKey || !bo.subtypes?.[selectedSubtypeKey]) {
    return {
      boKey: bo.key,
      selectedSubtypeKey: null,
      discriminatorClause: null,
      fields: rootFields.map((f) => ({ ...f, scope: 'ASSIGNED' })),
      relationships: allRelationships.filter((rel) => !rel.scopedSubtypeKey),
    };
  }

  const activeSubtype = bo.subtypes![selectedSubtypeKey];
  const assignedFields = (activeSubtype.subtypeFields || []).map((f) => ({
    ...f,
    scope: 'ASSIGNED' as const,
  }));

  const seenKeys = new Set<string>(
    assignedFields.map((f) => f.key || f.technicalName || f.id).filter(Boolean)
  );
  const inheritedFields = rootFields
    .filter((f) => {
      const id = f.key || f.technicalName || f.id;
      return id && !seenKeys.has(id);
    })
    .map((f) => ({ ...f, scope: 'INHERITED' as const }));

  const allowedRelKeys = new Set<string>(activeSubtype.relationshipAllowlist || []);
  const reachableRelationships = allRelationships.filter((rel) => {
    if (!rel.scopedSubtypeKey) return true;
    return (
      rel.scopedSubtypeKey === selectedSubtypeKey &&
      allowedRelKeys.has(rel.key)
    );
  });

  return {
    boKey: bo.key,
    selectedSubtypeKey,
    discriminatorClause: `t0.subtype_code = '${selectedSubtypeKey}'`,
    fields: [...assignedFields, ...inheritedFields],
    relationships: reachableRelationships,
  };
}
