

export interface Field {
  key: string;
  name: string;
  businessName: string; // Display name (e.g., "Legal Name") - FROM SEMANTIC TERM
  technicalName: string; // Lowercase_with_underscores (e.g., "legal_name") - FROM SEMANTIC TERM
  type: 'text' | 'number' | 'date' | 'boolean' | 'json' | 'array'; // FROM SEMANTIC TERM METADATA
  isCore?: boolean; // true if from core BO, false if custom
  inheritedFrom?: string; // entity key this was inherited from
  inheritedFromKey?: string; // Technical name of where field was inherited
  semanticTermId?: string; // Link to semantic term in catalog (optional for backward compat)
  semanticTermName?: string; // Display name of linked semantic term (optional for backward compat)
  semanticId?: string; // Alternative ID for compatibility
  semanticTerms?: string[]; // Multiple semantic term links
  description?: string; // From semantic term properties
  sequence?: number; // Order in which to display (0, 1, 2, ...)
  validation?: 'valid' | 'warning' | 'error';
  validationMessage?: string;
  group?: string;
  lastModifiedAt?: string; // ISO timestamp when field was added/edited
  createdBy?: string; // User who created the field
}

export interface Subtype {
  key?: string; // Technical name (lowercase_with_underscores)
  name: string; // Display name
  businessName?: string; // Business-friendly name
  technicalName?: string; // Lowercase_with_underscores version
  entity_fields?: Field[]; // Inherited fields from parent entity
  subtype_fields: Field[];
  isCore?: boolean; // true if cloned from core
  basedOnEntity?: string; // entity key if cloned from core
}

export interface Entity {
  id?: string; // UUID from database
  key?: string; // Technical name (lowercase_with_underscores)
  name: string; // Display name (e.g., "Client Investor")
  businessName?: string; // Business-friendly name
  technicalName?: string; // Lowercase_with_underscores (e.g., "client_investor")
  description?: string; // Entity description for UI
  entity_fields: Field[];
  subtypes: Record<string, Subtype>;
  isCore?: boolean; // true if this is a core BO
  coreFields?: Field[]; // explicitly separate core fields
  customFields?: Field[]; // explicitly separate custom fields
  clonesFrom?: string; // if cloned, which entity it came from (e.g., "client_investor")
  clonesFromKey?: string; // Technical key of parent (e.g., "client_investor")
  cloneParentName?: string; // Display name of parent for UI tracking
}

export interface HierarchyNode {
  id: string;
  name: string;
  displayName?: string;
  icon: string;
  children?: HierarchyNode[];
  fields?: Field[];
}

export interface Entities {
  [key: string]: Entity;
}

