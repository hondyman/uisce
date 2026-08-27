export interface FieldDefinition {
  id: string;
  key: string;
  technicalName: string;
  displayName: string;
  dataType: string;
  isCore?: boolean;
  scope?: 'ASSIGNED' | 'INHERITED';
  bindingRequirement?: 'REQUIRED' | 'OPTIONAL' | 'CALCULATED' | 'INTERNAL';
  bindingStatus?: 'RESOLVED' | 'PARTIALLY_RESOLVED' | 'UNRESOLVED' | 'NOT_APPLICABLE';
  validationRules?: Record<string, any>;
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
}

/**
 * Resolves the effective field set for a Business Object view.
 * - When root is active: returns deduplicated core + custom attributes.
 * - When a subtype is active and showInherited is false: returns strictly assigned subtype fields.
 * - When a subtype is active and showInherited is true: returns union of assigned + inherited root fields.
 */
export function resolveSubtypeFields(
  bo: BusinessObject | null | undefined,
  selectedSubtypeKey: string | null,
  showInheritedFields: boolean
): FieldDefinition[] {
  if (!bo) return [];

  // Extract root baseline fields
  const rootFields: FieldDefinition[] = (() => {
    if (bo.customFields && bo.customFields.length > 0) return bo.customFields;
    if (bo.config?.fields && bo.config.fields.length > 0) return bo.config.fields;
    return [...(bo.coreFields || []), ...(bo.customFields || [])];
  })();

  // Deduplicate root baseline by stable technical identifier
  const seenRootKeys = new Set<string>();
  const deduplicatedRoot: FieldDefinition[] = [];

  for (const field of rootFields) {
    const identifier = field.key || field.technicalName || field.id;
    if (!identifier || seenRootKeys.has(identifier)) continue;
    seenRootKeys.add(identifier);
    deduplicatedRoot.push({
      ...field,
      scope: 'ASSIGNED',
    });
  }

  // 1. Root Object Active View
  if (!selectedSubtypeKey || !bo.subtypes || !bo.subtypes[selectedSubtypeKey]) {
    return deduplicatedRoot;
  }

  // 2. Subtype Object Active View
  const activeSubtype = bo.subtypes[selectedSubtypeKey];
  const assignedFields: FieldDefinition[] = (activeSubtype.subtypeFields || []).map((f) => ({
    ...f,
    scope: 'ASSIGNED',
  }));

  if (!showInheritedFields) {
    return assignedFields;
  }

  // 3. Union Mode (Assigned + Inherited Baseline)
  const seenSubtypeKeys = new Set<string>(
    assignedFields.map((f) => f.key || f.technicalName || f.id).filter(Boolean)
  );

  const inheritedFields: FieldDefinition[] = [];
  for (const rootField of deduplicatedRoot) {
    const identifier = rootField.key || rootField.technicalName || rootField.id;
    if (identifier && !seenSubtypeKeys.has(identifier)) {
      inheritedFields.push({
        ...rootField,
        scope: 'INHERITED',
      });
    }
  }

  return [...assignedFields, ...inheritedFields];
}
