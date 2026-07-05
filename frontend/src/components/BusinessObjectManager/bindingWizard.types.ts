/**
 * Types for the binding-first Business Object create wizard.
 *
 * These types model the new binding/field-binding design while remaining
 * serializable to the current backend's business-object shape. When the
 * backend gains explicit business_object_binding / field_binding tables,
 * these types can be extended without changing the UI contract.
 */

export type EligibilitySource =
  | 'DIRECT'
  | 'RELATED'
  | 'CALCULATED'
  | 'MANUAL'
  | 'INHERITED'
  | 'OVERRIDE';

export type BindingRequirement =
  | 'REQUIRED'
  | 'OPTIONAL'
  | 'BACKEND_SPECIFIC'
  | 'CALCULATED'
  | 'INTERNAL';

export type BindingStatus = 'RESOLVED' | 'PARTIAL' | 'UNRESOLVED';

export type BoStatus = 'draft' | 'active' | 'deprecated';

/** A catalog column that a semantic term maps to for a given binding. */
export interface TermColumnMapping {
  columnNodeId: string;
  columnName: string;
  tableNodeId: string;
  tableName: string;
  isPrimarySource: boolean;
}

/** A semantic term presented in the wizard, with its known column mappings. */
export interface WizardSemanticTerm {
  termNodeId: string;
  termKey: string;
  termName: string;
  description?: string;
  dataType?: string;
  role?: string;
  eligibilitySource: EligibilitySource;
  mappings: TermColumnMapping[];
}

/** A field that has been added to the Business Object. */
export interface WizardField {
  /** Stable key derived from the term or manually entered. */
  key: string;
  /** Human-readable field name. */
  name: string;
  /** Business/display name. */
  displayName: string;
  description?: string;
  semanticTermId: string;
  semanticTermName: string;
  dataType?: string;
  role: string;
  bindingRequirement: BindingRequirement;
  eligibilitySource: EligibilitySource;
  bindingStatus: BindingStatus;
  /** The column mapping chosen for this field (undefined = unresolved). */
  selectedMapping?: TermColumnMapping;
  /** Sequence order in the BO. */
  sequence: number;
  /** True when the local tenant has overridden an inherited field. */
  hasLocalOverride?: boolean;
  has_local_override?: boolean;
  coreReferenceFieldId?: string;
  core_reference_field_id?: string;
  /** Local formula/expression override for calculated measures. */
  localExpressionOverride?: string;
  /** Audit justification for an override or custom field. */
  overrideReason?: string;
}

/** One binding context for the BO (today: one per BO; future: many). */
export interface WizardBinding {
  /** Local-only id for UI state; backend will eventually supply a binding id. */
  localId: string;
  /** Backend / datasource id. For now this is the current datasource. */
  backendId: string;
  backendName: string;
  /** The catalog table node that drives this binding. */
  drivingTableId?: string;
  drivingTableName?: string;
  drivingTableQualifiedPath?: string;
  isDefault: boolean;
  /** Fields scoped to this binding. */
  fields: WizardField[];
}

/** Top-level wizard form state. */
export interface WizardBusinessObject {
  name: string;
  key: string;
  displayName: string;
  description: string;
  status: BoStatus;
  enableHistory: boolean;
  historyMode: 'EXPLICIT_RANGE' | 'EVENT_LOG';
  bindings: WizardBinding[];
}

/** Catalog table node returned by /api/catalog/nodes?type=table. */
export interface CatalogTableNode {
  node_id: string;
  node_name: string;
  qualified_path: string;
  node_type?: string;
  description?: string;
}

/** Payload sent to the existing /api/business-objects endpoint. */
export interface CreateBusinessObjectPayload {
  name: string;
  display_name: string;
  description?: string;
  status: BoStatus;
  enable_history?: boolean;
  history_mode?: 'EXPLICIT_RANGE' | 'EVENT_LOG';
  driver_table_id?: string;
  driver_table_name?: string;
  config: {
    is_active: boolean;
    fields: CreateBusinessObjectFieldPayload[];
  };
}

export interface CreateBusinessObjectFieldPayload {
  key: string;
  name: string;
  businessName: string;
  displayName: string;
  technicalName: string;
  description?: string;
  type: string;
  role: string;
  semanticTermId: string;
  semanticTermName: string;
  sequence: number;
  isCore: boolean;
  bindingRequirement: BindingRequirement;
  eligibilitySource: EligibilitySource;
  bindingStatus: BindingStatus;
  sourceColumnNodeId?: string;
  sourceColumnName?: string;
  sourceTableNodeId?: string;
  sourceTableName?: string;
  coreReferenceFieldId?: string;
  core_reference_field_id?: string;
  hasLocalOverride?: boolean;
  has_local_override?: boolean;
}
