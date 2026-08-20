import type { EdgeType, EdgeProperty } from '../../types/edgeTypes';

export type EdgeCardinality = '1:1' | '1:N' | 'N:1' | 'N:M';

export interface CatalogNodeTypeInfo {
  id: string;
  catalog_type_name: string;
  description?: string;
  type?: 'core' | 'custom';
  properties?: any[];
  config?: any;
}

export interface CatalogNodeItem {
  id: string;
  node_name: string;
  node_type_id?: string;
  catalog_type?: string;
  catalog_type_name?: string;
  description?: string;
  qualified_path?: string;
  parent_id?: string;
  parent_name?: string;
  datasource?: string;
  datasource_id?: string;
  schema?: string;
  table?: string;
  properties?: Record<string, any>;
  is_active?: boolean;
  type?: 'core' | 'custom';
}

export interface TargetNodeDraft {
  isNew: boolean;
  node_name: string;
  description?: string;
  catalog_type: string;
  node_type_id: string;
  properties?: Record<string, any>; // JSON attributes for target node
}

export interface EdgePropertiesDraft {
  transformation?: string;
  confidence?: number;
  mapping_notes?: string;
  source_column?: string;
  match_strategy?: string;
  properties?: Record<string, any>; // Additional JSON attributes for the edge
}

export type GovernanceTier = 'draft' | 'custom' | 'gold_certified';

export interface UniversalValueType {
  name: string;
  category: string;
  standard: string;
  description: string;
  subProperties?: string[];
  validationRule?: string;
  isPii?: boolean;
}

export interface CompositeClusterMember {
  sourceColumn: string;
  sourceNodeId: string;
  subProperty: string;
  suggestedTermName: string;
  confidence: number;
}

export interface CompositeCluster {
  clusterId: string;
  clusterType: 'Address' | 'PersonName' | 'ContactCommunication' | 'FinancialAmount' | 'AuditTimestamp';
  entityName: string;
  tableName: string;
  universalParent: string;
  standard: string;
  compositeTermName: string;
  members: CompositeClusterMember[];
  isMapped?: boolean;
}

export interface RelatedItemCandidate {
  id: string;
  node_name: string;
  catalog_type?: string;
  relation_type: 'see_also' | 'sibling' | 'equivalent' | 'parent';
  description?: string;
  isAlreadyLinked?: boolean;
}

export interface VendorAlignment {
  vendor: 'BLOOMBERG' | 'FACTSET' | 'LSEG' | 'SP_CAPITAL_IQ' | 'MSCI' | 'MORNINGSTAR';
  mnemonic: string;
  canonicalTermName: string;
  category: string;
  description: string;
  feedType?: string;
}

export interface HierarchicalEdgeDraft {
  edge_type_name: string; // e.g. 'SPECIALIZES', 'BELONGS_TO', 'LICENSED_BY', 'SYNONYM_OF', 'COMPOSITE_MEMBER_OF'
  target_node_name: string;
  target_catalog_type: string;
  properties?: Record<string, any>;
}

export interface MatchSuggestion {
  targetNode: CatalogNodeItem;
  targetDraft?: TargetNodeDraft; // If target node needs to be created
  edgeDraft?: EdgePropertiesDraft; // JSON attributes for edge
  confidence: number; // 0 to 100
  matchReason: string;
  matchType: 'exact_normalized' | 'abbreviation' | 'token_overlap' | 'fuzzy' | 'description' | 'gemini_ai' | 'contextual_disambiguated' | 'composite_cluster' | 'vendor_aligned' | 'manual';
  relatedItems?: RelatedItemCandidate[]; // "See Also" discoveries
  isGenericCollision?: boolean;
  contextualEntity?: string;
  suggestedContextualTerm?: string;
  isContextualDisambiguated?: boolean;
  universalParentName?: string; // e.g. "Address" for "VendorAddress"
  universalStandard?: string; // e.g. "ISO 19160"
  governanceTier?: GovernanceTier;
  hierarchicalEdgesToCreate?: HierarchicalEdgeDraft[]; // Automatically links SPECIALIZES / BELONGS_TO / LICENSED_BY edges
  compositeCluster?: CompositeCluster;
  vendorAlignment?: VendorAlignment;
}

export interface GraphMappingRow {
  id: string;
  sourceNode: CatalogNodeItem;
  edgeType: EdgeType;
  cardinality: EdgeCardinality;
  currentTargetNode: CatalogNodeItem | null;
  existingEdgeId?: string;
  suggestion: MatchSuggestion | null;
  alternativeSuggestions: MatchSuggestion[];
  selectedTargetNode: CatalogNodeItem | null; // Currently chosen target
  selectedTargetDraft?: TargetNodeDraft | null; // If user selects a newly suggested node
  edgeProperties?: Record<string, any>;
  isMapped: boolean;
  isModified: boolean;
  isSaving?: boolean;
  isRejecting?: boolean;
  relatedItems: RelatedItemCandidate[]; // Discovered related nodes ("See Also")
  cardinalityConflict?: string; // Reason if mapping would violate 1:1 or 1:N
  isGenericCollision?: boolean;
  contextualEntity?: string;
  suggestedContextualTerm?: string;
  universalParentName?: string;
  governanceTier?: GovernanceTier;
  compositeCluster?: CompositeCluster;
  vendorAlignment?: VendorAlignment;
}

export interface MapperFilterOptions {
  search: string;
  filterTab: 'all' | 'unmapped' | 'high_confidence' | 'composite_clusters' | 'vendor_aligned' | 'generic_collisions' | 'needs_review' | 'has_see_also' | 'mapped';
  datasourceFilter: string;
  minConfidence: number;
}
