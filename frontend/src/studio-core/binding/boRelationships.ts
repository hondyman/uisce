/**
 * Shared shape of GET /api/business-objects/{boId}/relationships
 * (BusinessObjectService.GetBusinessObjectRelationships). This is the one
 * relationship source any studio (Page Studio, Report Studio, the
 * API/Query builder) may use — do not add a second API.
 *
 * Studio-agnostic: moved out of pages/page-studio so Report Studio can
 * import the same relationship-fence logic Page Studio already runs on,
 * per HANDOFF_REPORT_BUILDER_SPINE_PLAN.md Phase 1 (spine extraction).
 */
export interface JoinColumn {
  source: string;
  target: string;
}

export interface BORelationship {
  id?: string;
  relatedObjectName: string;
  targetObjectId: string;
  relationshipType: string;
  cardinality: string;
  joinCondition: string;
  joinColumns?: JoinColumn[];
  sourceDriverTable?: string;
  targetDriverTable?: string;
  /** 'bo' = another Business Object; 'relatedTable' = extra table of this BO (not a chip). */
  kind?: 'bo' | 'relatedTable';
  linkTable?: string;
}

/** Drag payload when an author drops a related Business Object onto the canvas. */
export interface RelatedObjectDragPayload {
  parentSourceId: string;
  parentBoId: string;
  targetObjectId: string;
  relatedObjectName: string;
  cardinality: string;
  joinCondition: string;
  joinColumns?: JoinColumn[];
  kind?: 'bo' | 'relatedTable';
  linkTable?: string;
}

/** Child-side column from a resolved "child.col = parent.col" joinCondition. */
export const fkColumnFromJoinCondition = (joinCondition: string): string => {
  const left = joinCondition.split('=')[0]?.trim() || '';
  const dot = left.lastIndexOf('.');
  return dot >= 0 ? left.slice(dot + 1) : left;
};

/**
 * Cardinality is source:target. A many-valued *target* is a related list
 * (Table); a one-valued target is a lookup/parent (Form). The graph decides;
 * callers must not hardcode "always Table".
 */
export const widgetTypeForCardinality = (cardinality: string): 'Table' | 'Form' => {
  const right = (cardinality.split(':')[1] || cardinality).toUpperCase();
  if (/[NM]/.test(right) || /MANY/.test(right)) return 'Table';
  return 'Form';
};
