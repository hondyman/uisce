-- ============================================================================
-- WORKDAY-STYLE METADATA-DRIVEN UI SCHEMA
-- Enables dynamic form generation, validation, and business process orchestration
-- ============================================================================

-- ============================================================================
-- BUSINESS OBJECT DEFINITIONS (The Core Data Model)
-- ============================================================================

CREATE TABLE IF NOT EXISTS business_objects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    
    -- Identity
    bo_name VARCHAR(100) NOT NULL UNIQUE,
    bo_description TEXT,
    entity_type VARCHAR(50), -- employee|customer|order|department|etc
    
    -- Configuration
    is_system_bo BOOLEAN DEFAULT FALSE, -- System-provided vs custom
    is_active BOOLEAN DEFAULT TRUE,
    
    -- Extensibility
    allow_custom_fields BOOLEAN DEFAULT TRUE,
    allow_field_deletion BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_bo_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_bo_tenant ON business_objects(tenant_id);
CREATE INDEX IF NOT EXISTS idx_bo_name ON business_objects(bo_name);

-- ============================================================================
-- BUSINESS OBJECT FIELDS (Define Data Structure)
-- ============================================================================

CREATE TABLE IF NOT EXISTS bo_fields (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_object_id UUID NOT NULL,
    
    -- Field identity
    field_name VARCHAR(100) NOT NULL, -- snake_case: employee_id, first_name
    field_type VARCHAR(20) NOT NULL, -- string|number|date|boolean|reference|picklist|decimal|datetime
    
    -- Display configuration
    display_label VARCHAR(200) NOT NULL, -- "Employee ID", "First Name"
    display_order INT NOT NULL, -- For default ordering
    section_name VARCHAR(100), -- "Basic Info", "Contact", "Employment"
    help_text TEXT,
    placeholder_text VARCHAR(200),
    
    -- Validation configuration
    is_required BOOLEAN DEFAULT FALSE,
    is_readonly BOOLEAN DEFAULT FALSE,
    is_searchable BOOLEAN DEFAULT TRUE,
    is_sortable BOOLEAN DEFAULT TRUE,
    
    -- Type-specific constraints
    max_length INT, -- For string fields
    min_value DECIMAL, -- For numeric fields
    max_value DECIMAL,
    decimal_places INT, -- For decimal fields
    date_format VARCHAR(20), -- "YYYY-MM-DD", "MM/DD/YYYY"
    time_zone VARCHAR(50), -- For datetime fields
    
    -- Reference configuration (for reference type)
    reference_bo_id UUID REFERENCES business_objects(id), -- Points to another BO
    reference_display_field VARCHAR(100), -- Which field to show in dropdown
    reference_search_fields TEXT[], -- Which fields to search when looking up
    
    -- Picklist configuration (for picklist type)
    picklist_values TEXT[], -- {"Active", "Inactive", "Pending"}
    picklist_source VARCHAR(50), -- inline|database|external_api
    picklist_source_table VARCHAR(100), -- If database source
    
    -- Default values
    default_value TEXT,
    default_value_type VARCHAR(20), -- static|formula|current_user|current_datetime
    
    -- Relationships
    validation_rule_ids UUID[] DEFAULT '{}', -- References to validation_rules
    
    -- Metadata
    is_system_field BOOLEAN DEFAULT FALSE,
    is_custom_field BOOLEAN DEFAULT FALSE,
    custom_properties JSONB, -- For storing field-specific metadata
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_field_business_object FOREIGN KEY (business_object_id) REFERENCES business_objects(id) ON DELETE CASCADE,
    CONSTRAINT unique_field_per_business_object UNIQUE(business_object_id, field_name)
);

CREATE INDEX IF NOT EXISTS idx_bo_fields_business_object ON bo_fields(business_object_id);
CREATE INDEX IF NOT EXISTS idx_bo_fields_ref_bo ON bo_fields(reference_bo_id);
CREATE INDEX IF NOT EXISTS idx_bo_fields_section ON bo_fields(section_name);

-- ============================================================================
-- VALIDATION RULES (The Rule Engine)
-- ============================================================================

CREATE TABLE IF NOT EXISTS validation_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    
    -- Rule identity
    rule_name VARCHAR(100) NOT NULL,
    rule_description TEXT,
    rule_category VARCHAR(50), -- format|range|uniqueness|cross_field|custom
    
    -- Severity
    severity VARCHAR(20) DEFAULT 'error', -- error|warning|info
    
    -- Error messaging
    error_message TEXT NOT NULL, -- "Employee ID must be in format EMP followed by 6 digits"
    help_message TEXT, -- Displayed in help tooltip
    
    -- Rule condition (stored as JSON for flexibility)
    condition_type VARCHAR(50), -- regex|compare|unique_check|range|custom_function|script
    condition_json JSONB NOT NULL, -- Stores rule parameters
    
    -- Execution configuration
    execute_client_side BOOLEAN DEFAULT TRUE, -- Can run in browser
    execute_server_side BOOLEAN DEFAULT TRUE, -- Should run on server
    run_on_blur BOOLEAN DEFAULT FALSE, -- Validate field on blur event
    run_on_change BOOLEAN DEFAULT FALSE, -- Validate field on change event
    run_on_submit BOOLEAN DEFAULT TRUE, -- Always validate on submit
    
    -- Performance
    requires_database_call BOOLEAN DEFAULT FALSE, -- For uniqueness checks
    cache_results BOOLEAN DEFAULT TRUE,
    cache_ttl_seconds INT DEFAULT 3600,
    
    -- Activation
    is_active BOOLEAN DEFAULT TRUE,
    applies_to_new_records BOOLEAN DEFAULT TRUE,
    applies_to_updates BOOLEAN DEFAULT TRUE,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_rule_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_validation_rules_tenant ON validation_rules(tenant_id);
CREATE INDEX IF NOT EXISTS idx_validation_rules_category ON validation_rules(rule_category);

-- ============================================================================
-- PAGE LAYOUTS (The UI Configuration)
-- ============================================================================

CREATE TABLE IF NOT EXISTS page_layouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    business_object_id UUID NOT NULL,
    
    -- Layout identity
    layout_name VARCHAR(100) NOT NULL, -- "Employee Entry Form", "Manager Approval"
    layout_type VARCHAR(50) NOT NULL, -- form|grid|detail|wizard
    layout_description TEXT,
    
    -- Responsive design
    default_columns INT DEFAULT 2, -- 1|2|3 for responsive grid
    mobile_layout VARCHAR(50), -- single_column|hide_sections|drawer
    
    -- Configuration
    is_default_layout BOOLEAN DEFAULT FALSE, -- Used when no layout specified
    is_system_layout BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    
    -- Metadata
    custom_css JSONB, -- Custom styling rules
    custom_js JSONB, -- Custom JavaScript logic
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_layout_business_object FOREIGN KEY (business_object_id) REFERENCES business_objects(id) ON DELETE CASCADE,
    CONSTRAINT fk_layout_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT unique_layout_name_per_business_object UNIQUE(tenant_id, business_object_id, layout_name)
);

CREATE INDEX IF NOT EXISTS idx_layouts_business_object ON page_layouts(business_object_id);
CREATE INDEX IF NOT EXISTS idx_layouts_tenant ON page_layouts(tenant_id);

-- ============================================================================
-- LAYOUT SECTIONS (Grouping Fields in Forms)
-- ============================================================================

CREATE TABLE IF NOT EXISTS layout_sections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    layout_id UUID NOT NULL,
    
    -- Section metadata
    section_order INT NOT NULL, -- Display order (1, 2, 3, etc.)
    section_title VARCHAR(200) NOT NULL, -- "Basic Information", "Contact Info"
    section_description TEXT,
    section_columns INT DEFAULT 2, -- Grid columns for this section
    
    -- Appearance
    is_collapsible BOOLEAN DEFAULT FALSE,
    is_initially_collapsed BOOLEAN DEFAULT FALSE,
    has_border BOOLEAN DEFAULT TRUE,
    background_color VARCHAR(7), -- Hex color
    
    -- Visibility & accessibility
    is_visible BOOLEAN DEFAULT TRUE,
    visibility_condition JSONB, -- When to show/hide section
    help_text TEXT,
    
    -- Field references
    field_ids UUID[] NOT NULL DEFAULT '{}', -- Which fields are in this section
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_section_layout FOREIGN KEY (layout_id) REFERENCES page_layouts(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_sections_layout ON layout_sections(layout_id);

-- ============================================================================
-- LAYOUT ACTIONS (Buttons & Form Actions)
-- ============================================================================

CREATE TABLE IF NOT EXISTS layout_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    layout_id UUID NOT NULL,
    
    -- Action metadata
    action_order INT NOT NULL, -- Button display order
    action_label VARCHAR(100) NOT NULL, -- "Save", "Submit for Approval", "Cancel"
    action_type VARCHAR(50) NOT NULL, -- save|submit|cancel|draft|delete|custom
    action_icon VARCHAR(50), -- Material Design icon name
    
    -- Validation
    requires_validation BOOLEAN DEFAULT FALSE, -- Must validate before action
    requires_confirmation BOOLEAN DEFAULT FALSE,
    confirmation_message TEXT,
    
    -- Business Logic
    triggers_bp_id UUID, -- Business Process to trigger
    triggers_webhooks JSONB DEFAULT '[]'::jsonb, -- External webhooks to call
    
    -- Conditions
    is_visible BOOLEAN DEFAULT TRUE,
    visibility_condition JSONB, -- When button should appear
    is_enabled BOOLEAN DEFAULT TRUE,
    enable_condition JSONB, -- When button should be clickable
    
    -- Button styling
    button_style VARCHAR(50), -- primary|secondary|danger|warning|success
    button_size VARCHAR(50), -- small|medium|large
    
    -- Success/Error handling
    success_message TEXT, -- Message shown on success
    error_message TEXT,
    redirect_on_success VARCHAR(500), -- URL to redirect after success
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_action_layout FOREIGN KEY (layout_id) REFERENCES page_layouts(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_actions_layout ON layout_actions(layout_id);

-- ============================================================================
-- FIELD-VALIDATION LINKING (Many-to-Many Relationship)
-- ============================================================================

CREATE TABLE IF NOT EXISTS field_validation_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    field_id UUID NOT NULL,
    validation_rule_id UUID NOT NULL,
    
    -- Execution order
    rule_order INT NOT NULL DEFAULT 1,
    
    -- Override for this specific field
    override_message TEXT, -- Field-specific error message
    override_severity VARCHAR(20), -- Override rule severity for this field
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_field FOREIGN KEY (field_id) REFERENCES bo_fields(id) ON DELETE CASCADE,
    CONSTRAINT fk_validation FOREIGN KEY (validation_rule_id) REFERENCES validation_rules(id) ON DELETE CASCADE,
    CONSTRAINT unique_field_rule UNIQUE(field_id, validation_rule_id)
);

CREATE INDEX IF NOT EXISTS idx_field_validation_field ON field_validation_rules(field_id);
CREATE INDEX IF NOT EXISTS idx_field_validation_rule ON field_validation_rules(validation_rule_id);

-- ============================================================================
-- FORM INSTANCE TRACKING (For analytics and audit)
-- ============================================================================

CREATE TABLE IF NOT EXISTS form_submissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    business_object_id UUID NOT NULL,
    layout_id UUID NOT NULL,
    
    -- Submission metadata
    submission_id VARCHAR(255) NOT NULL UNIQUE,
    submitted_by UUID NOT NULL, -- User ID who submitted
    submitted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Form data
    form_data JSONB NOT NULL, -- Complete form submission data
    form_data_hash VARCHAR(256), -- SHA-256 hash for integrity checking
    
    -- Validation results
    validation_passed BOOLEAN NOT NULL,
    validation_errors JSONB DEFAULT '[]'::jsonb, -- Error details
    validation_warnings JSONB DEFAULT '[]'::jsonb, -- Warning details
    validation_timestamp TIMESTAMP,
    
    -- Processing
    status VARCHAR(50) DEFAULT 'pending', -- pending|approved|rejected|processed|error
    status_reason TEXT,
    processed_at TIMESTAMP,
    processed_by UUID, -- User who approved/rejected
    
    -- Audit trail
    ip_address INET,
    user_agent TEXT,
    api_version VARCHAR(20),
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_submission_business_object FOREIGN KEY (business_object_id) REFERENCES business_objects(id),
    CONSTRAINT fk_submission_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_submissions_tenant ON form_submissions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_submissions_business_object ON form_submissions(business_object_id);
CREATE INDEX IF NOT EXISTS idx_submissions_status ON form_submissions(status);
CREATE INDEX IF NOT EXISTS idx_submissions_submitted_by ON form_submissions(submitted_by);
CREATE INDEX IF NOT EXISTS idx_submissions_timestamp ON form_submissions(submitted_at DESC);

-- ============================================================================
-- FIELD DEPENDENCIES (For conditional visibility and validation)
-- ============================================================================

CREATE TABLE IF NOT EXISTS field_dependencies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    field_id UUID NOT NULL,
    dependent_on_field_id UUID NOT NULL,
    
    -- Dependency type
    dependency_type VARCHAR(50) NOT NULL, -- visibility|validation|required|disable|populate
    
    -- Condition for dependency
    condition_operator VARCHAR(20), -- equals|not_equals|contains|gt|lt|gte|lte|regex_match|in_list
    condition_value JSONB, -- The value to compare against
    
    -- Action to take
    action_type VARCHAR(50), -- show|hide|require|optional|validate|populate
    action_value JSONB, -- Optional value for populate action
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_field FOREIGN KEY (field_id) REFERENCES bo_fields(id) ON DELETE CASCADE,
    CONSTRAINT fk_dependent_field FOREIGN KEY (dependent_on_field_id) REFERENCES bo_fields(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_dependencies_field ON field_dependencies(field_id);
CREATE INDEX IF NOT EXISTS idx_dependencies_dependent ON field_dependencies(dependent_on_field_id);

-- ============================================================================
-- CONDITIONAL VISIBILITY RULES (For dynamic UI)
-- ============================================================================

CREATE TABLE IF NOT EXISTS visibility_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    
    -- Rule identity
    rule_name VARCHAR(100),
    
    -- Target (what to show/hide)
    target_type VARCHAR(50) NOT NULL, -- field|section|action
    target_id UUID NOT NULL,
    
    -- Condition (when to show/hide)
    condition_json JSONB NOT NULL, -- {field_id: "...", operator: "equals", value: "..."}
    
    -- Logic
    logic_type VARCHAR(20) DEFAULT 'and', -- and|or
    
    -- Action
    action VARCHAR(20) NOT NULL, -- show|hide
    
    is_active BOOLEAN DEFAULT TRUE,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_visibility_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_visibility_tenant ON visibility_rules(tenant_id);
CREATE INDEX IF NOT EXISTS idx_visibility_target ON visibility_rules(target_type, target_id);

-- ============================================================================
-- FORM CUSTOMIZATIONS PER TENANT/USER
-- ============================================================================

CREATE TABLE IF NOT EXISTS layout_customizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    layout_id UUID NOT NULL,
    user_id UUID, -- If NULL, applies to all users in tenant
    
    -- Customization type
    customization_type VARCHAR(50), -- hide_field|show_field|reorder|rename|change_label|custom_css
    
    -- What to customize
    target_id UUID, -- Field or section ID
    
    -- Customization details
    customization_json JSONB, -- Flexible for different customization types
    
    is_active BOOLEAN DEFAULT TRUE,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_custom_layout FOREIGN KEY (layout_id) REFERENCES page_layouts(id) ON DELETE CASCADE,
    CONSTRAINT fk_custom_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_customizations_layout ON layout_customizations(layout_id);
CREATE INDEX IF NOT EXISTS idx_customizations_tenant_user ON layout_customizations(tenant_id, user_id);

-- ============================================================================
-- GRANTS
-- ============================================================================

GRANT SELECT, INSERT, UPDATE, DELETE ON business_objects TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON bo_fields TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON validation_rules TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON page_layouts TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON layout_sections TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON layout_actions TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON field_validation_rules TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON form_submissions TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON field_dependencies TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON visibility_rules TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON layout_customizations TO app_user;

-- ============================================================================
-- INITIAL DATA: EXAMPLE BUSINESS OBJECTS
-- ============================================================================

-- Note: Replace with actual tenant IDs in your system
-- This is a template showing the structure

/*
-- Employee Business Object
INSERT INTO business_objects (tenant_id, bo_name, entity_type, bo_description)
VALUES (
    '00000000-0000-0000-0000-000000000000', -- Your tenant ID
    'Employee',
    'employee',
    'Core employee information and employment details'
);

-- Get the BO ID for further inserts
-- SELECT id INTO @employee_bo_id FROM business_objects WHERE bo_name = 'Employee';

-- Sample Fields for Employee BO
INSERT INTO bo_fields (bo_id, field_name, field_type, display_label, display_order, section_name, is_required)
VALUES
    (@employee_bo_id, 'employee_id', 'string', 'Employee ID', 1, 'Basic Info', true),
    (@employee_bo_id, 'first_name', 'string', 'First Name', 2, 'Basic Info', true),
    (@employee_bo_id, 'last_name', 'string', 'Last Name', 3, 'Basic Info', true),
    (@employee_bo_id, 'email', 'string', 'Email Address', 4, 'Contact Info', true),
    (@employee_bo_id, 'phone', 'string', 'Phone Number', 5, 'Contact Info', false),
    (@employee_bo_id, 'hire_date', 'date', 'Hire Date', 6, 'Employment Details', true),
    (@employee_bo_id, 'salary', 'decimal', 'Annual Salary', 7, 'Compensation', true),
    (@employee_bo_id, 'department', 'reference', 'Department', 8, 'Employment Details', true),
    (@employee_bo_id, 'status', 'picklist', 'Employment Status', 9, 'Employment Details', true);

-- Picklist values for status field
UPDATE bo_fields 
SET picklist_values = ARRAY['Active', 'Inactive', 'On Leave', 'Terminated']
WHERE bo_id = @employee_bo_id AND field_name = 'status';
*/
