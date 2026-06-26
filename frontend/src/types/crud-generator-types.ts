/**
 * CRUD UI Generator Types
 * 
 * These types define the configuration schema for dynamically generating
 * CRUD (Create, Read, Update, Delete) pages from Business Object definitions
 * and their relationships.
 */

// ============================================================================
// Field Configuration
// ============================================================================

/** Supported data types for fields */
export type FieldDataType =
    | 'string'
    | 'number'
    | 'integer'
    | 'boolean'
    | 'date'
    | 'datetime'
    | 'time'
    | 'enum'
    | 'json'
    | 'array'
    | 'uuid';

/** Widget types for field rendering */
export type FieldWidget =
    | 'text'
    | 'textarea'
    | 'number'
    | 'date'
    | 'datetime'
    | 'time'
    | 'select'
    | 'multiselect'
    | 'checkbox'
    | 'switch'
    | 'radio'
    | 'autocomplete'
    | 'lookup'
    | 'json-editor'
    | 'rich-text'
    | 'file-upload'
    | 'hidden';

/** Configuration for a lookup/reference field */
export interface LookupSource {
    /** Name of the related Business Object */
    boName: string;
    /** Field to use as the stored value (typically the primary key) */
    valueField: string;
    /** Field(s) to display to the user */
    labelField: string;
    /** Additional fields to include in the dropdown display */
    additionalFields?: string[];
    /** API endpoint for fetching lookup values */
    endpoint?: string;
    /** Filter to apply when fetching lookup values */
    filter?: Record<string, unknown>;
    /** Whether to allow searching/filtering */
    searchable?: boolean;
    /** Minimum characters before search is triggered */
    minSearchLength?: number;
}

/** Validation rules for a field */
export interface FieldValidation {
    required?: boolean;
    minLength?: number;
    maxLength?: number;
    min?: number;
    max?: number;
    pattern?: string;
    patternMessage?: string;
    custom?: string; // Custom validation function name
}

/** Complete configuration for a form field */
export interface FieldConfig {
    /** Unique field identifier */
    name: string;
    /** Display label */
    label: string;
    /** Data type */
    type: FieldDataType;
    /** UI widget to render */
    widget: FieldWidget;
    /** Is this field required? */
    required: boolean;
    /** Is this field read-only? */
    readOnly: boolean;
    /** Is this field disabled? */
    disabled?: boolean;
    /** Is this field visible? */
    visible?: boolean;
    /** Default value */
    defaultValue?: unknown;
    /** Placeholder text */
    placeholder?: string;
    /** Help text displayed below the field */
    helpText?: string;
    /** For enum types, the list of options */
    options?: Array<{ value: string | number; label: string }>;
    /** For lookup fields, the source configuration */
    lookupSource?: LookupSource;
    /** Validation rules */
    validation?: FieldValidation;
    /** Grid column span (1-12) */
    colSpan?: number;
    /** Custom styling */
    className?: string;
    /** Conditional visibility expression */
    visibleWhen?: string;
    /** Conditional enabled expression */
    enabledWhen?: string;
}

// ============================================================================
// Relationship Configuration
// ============================================================================

/** Types of relationships */
export type RelationshipType = '1:1' | '1:M' | 'M:1' | 'M:M';

/** UI roles for relationships */
export type UIRole = 'lookup' | 'detail' | 'child_collection' | 'association';

/** Configuration for a relationship */
export interface RelationshipConfig {
    /** Unique identifier for this relationship */
    id: string;
    /** Name of the target Business Object */
    targetBO: string;
    /** Display name for the relationship */
    displayName: string;
    /** Type of relationship (cardinality) */
    relationshipType: RelationshipType;
    /** How this relationship should be rendered in the UI */
    uiRole: UIRole;
    /** Is this a lookup/reference relationship? */
    isLookup: boolean;
    /** Fields to display from the related object */
    displayFields?: string[];
    /** Fields that can be inline-edited */
    editableFields?: string[];
    /** Whether users can add new related records inline */
    canAddNew?: boolean;
    /** Whether users can remove the relationship */
    canRemove?: boolean;
    /** Filter to apply when loading related records */
    filter?: Record<string, unknown>;
    /** Default sort order for related records */
    sortBy?: string;
    /** Sort direction */
    sortDirection?: 'asc' | 'desc';
    /** Maximum records to display inline (for child collections) */
    maxDisplay?: number;
}

// ============================================================================
// Layout Configuration
// ============================================================================

/** Section within a page layout */
export interface LayoutSection {
    /** Section identifier */
    id: string;
    /** Section title */
    title: string;
    /** Section description */
    description?: string;
    /** Type of section */
    type: 'fields' | 'relationship' | 'custom';
    /** Fields in this section (for type='fields') */
    fields?: string[];
    /** Relationship ID (for type='relationship') */
    relationshipId?: string;
    /** Custom component name (for type='custom') */
    component?: string;
    /** Is this section collapsible? */
    collapsible?: boolean;
    /** Is this section initially collapsed? */
    collapsed?: boolean;
    /** Grid column configuration */
    columns?: number;
}

/** Tab within a tabbed layout */
export interface LayoutTab {
    /** Tab identifier */
    id: string;
    /** Tab label */
    label: string;
    /** Icon for the tab */
    icon?: string;
    /** Sections within this tab */
    sections: LayoutSection[];
}

/** Overall page layout configuration */
export interface LayoutConfig {
    /** Layout type */
    type: 'single' | 'tabs' | 'wizard';
    /** For single layout: sections to display */
    sections?: LayoutSection[];
    /** For tabs layout: tab configuration */
    tabs?: LayoutTab[];
    /** For wizard layout: steps */
    steps?: LayoutSection[];
    /** Fields to display in the header/summary area */
    headerFields?: string[];
    /** Side panel configuration */
    sidePanel?: {
        enabled: boolean;
        width: number | string;
        sections: LayoutSection[];
    };
}

// ============================================================================
// Action Configuration
// ============================================================================

/** Action that can be performed on the page */
export interface ActionConfig {
    /** Action identifier */
    id: string;
    /** Display label */
    label: string;
    /** Icon */
    icon?: string;
    /** Action type */
    type: 'submit' | 'cancel' | 'delete' | 'custom';
    /** Variant (for styling) */
    variant?: 'primary' | 'secondary' | 'danger' | 'text';
    /** Confirmation message */
    confirmMessage?: string;
    /** API endpoint for custom actions */
    endpoint?: string;
    /** HTTP method for custom actions */
    method?: 'GET' | 'POST' | 'PUT' | 'DELETE';
    /** Visibility condition */
    visibleWhen?: string;
    /** Enabled condition */
    enabledWhen?: string;
}

// ============================================================================
// Complete Page Configuration
// ============================================================================

/** Complete configuration for a CRUD page */
export interface CRUDPageConfig {
    /** Business Object name */
    boName: string;
    /** Display name for the page */
    displayName: string;
    /** Description */
    description?: string;
    /** Icon */
    icon?: string;
    /** Field configurations */
    fields: FieldConfig[];
    /** Relationship configurations */
    relationships: RelationshipConfig[];
    /** Layout configuration */
    layout: LayoutConfig;
    /** Available actions */
    actions?: ActionConfig[];
    /** API configuration */
    api?: {
        baseUrl: string;
        listEndpoint?: string;
        getEndpoint?: string;
        createEndpoint?: string;
        updateEndpoint?: string;
        deleteEndpoint?: string;
    };
    /** Permissions */
    permissions?: {
        canCreate?: boolean;
        canRead?: boolean;
        canUpdate?: boolean;
        canDelete?: boolean;
    };
}

// ============================================================================
// Generator Input Types
// ============================================================================

/** Business Object definition used as input for the generator */
export interface BODefinition {
    id: string;
    name: string;
    displayName: string;
    description?: string;
    icon?: string;
    drivingTableId?: string;
    fields: Array<{
        name: string;
        type: string;
        label?: string;
        required?: boolean;
        semanticTermId?: string;
    }>;
    relationships?: Array<{
        targetBOId: string;
        targetBOName: string;
        relationshipType: RelationshipType;
        uiRole?: UIRole;
        isLookup?: boolean;
        joinPath?: Array<{
            table: string;
            column: string;
        }>;
    }>;
}

/** Options for the CRUD page generator */
export interface CRUDGeneratorOptions {
    /** Include all fields by default */
    includeAllFields?: boolean;
    /** Fields to exclude */
    excludeFields?: string[];
    /** Default widget mappings by type */
    widgetMappings?: Partial<Record<string, FieldWidget>>;
    /** Generate side panel for lookups */
    generateSidePanel?: boolean;
    /** Use tabs for relationships */
    useTabsForRelationships?: boolean;
    /** Include audit fields */
    includeAuditFields?: boolean;
}
