-- ============================================================================
-- EXAMPLE: HireEmployee Business Object with Complete Metadata
-- ============================================================================
-- This script demonstrates the complete setup for a "Hire Employee" form
-- including Business Object, Fields, Validation Rules, and Page Layout

-- **WARNING**: Replace tenant_id with your actual tenant UUID!
-- ============================================================================

-- Step 1: Create the Employee Business Object
-- ============================================================================

INSERT INTO business_objects (
    tenant_id, bo_name, entity_type, bo_description, allow_custom_fields, 
    allow_field_deletion, is_system_bo, is_active
) VALUES (
    '00000000-0000-0000-0000-000000000001'::UUID,  -- YOUR TENANT ID HERE
    'Employee',
    'employee',
    'Core employee information for hiring and employment management',
    true,
    false,
    false,
    true
);

-- Get the BO ID for subsequent inserts
-- SELECT id INTO @bo_employee FROM business_objects WHERE bo_name = 'Employee' LIMIT 1;

-- For this example, we'll hardcode it:
-- @bo_employee = '550e8400-e29b-41d4-a716-446655440001'::UUID

-- ============================================================================
-- Step 2: Create Validation Rules
-- ============================================================================

-- Rule 1: Employee ID Format
INSERT INTO validation_rules (
    tenant_id, rule_name, rule_description, rule_category, severity,
    error_message, help_message, condition_type, condition_json,
    execute_client_side, execute_server_side, run_on_blur, run_on_submit, is_active
) VALUES (
    '00000000-0000-0000-0000-000000000001'::UUID,
    'Employee ID Format Validation',
    'Validates that Employee ID matches company format (EMP followed by 6 digits)',
    'format',
    'error',
    'Employee ID must be in format EMP followed by 6 digits (example: EMP123456)',
    'Your employee ID is assigned by HR',
    'regex',
    '{"pattern": "^EMP[0-9]{6}$"}'::jsonb,
    true,
    true,
    true,
    true,
    true
);

-- Rule 2: Email Format
INSERT INTO validation_rules (
    tenant_id, rule_name, rule_description, rule_category, severity,
    error_message, help_message, condition_type, condition_json,
    execute_client_side, execute_server_side, run_on_blur, run_on_submit, is_active
) VALUES (
    '00000000-0000-0000-0000-000000000001'::UUID,
    'Email Format Validation',
    'Validates email address format',
    'format',
    'error',
    'Please enter a valid email address (example: john.doe@company.com)',
    'Must be a valid corporate email address',
    'regex',
    '{"pattern": "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"}'::jsonb,
    true,
    true,
    true,
    true,
    true
);

-- Rule 3: Email Uniqueness
INSERT INTO validation_rules (
    tenant_id, rule_name, rule_description, rule_category, severity,
    error_message, help_message, condition_type, condition_json,
    execute_client_side, execute_server_side, requires_database_call, 
    run_on_submit, is_active
) VALUES (
    '00000000-0000-0000-0000-000000000001'::UUID,
    'Email Uniqueness Check',
    'Ensures email address is not already in use',
    'uniqueness',
    'error',
    'This email address is already in use. Please enter a different email.',
    'Each employee must have a unique email address',
    'unique_check',
    '{"field": "email", "table": "employees", "scope": "global"}'::jsonb,
    false,
    true,
    true,
    true,
    true
);

-- Rule 4: Hire Date Not in Future
INSERT INTO validation_rules (
    tenant_id, rule_name, rule_description, rule_category, severity,
    error_message, help_message, condition_type, condition_json,
    execute_client_side, execute_server_side, run_on_blur, run_on_submit, is_active
) VALUES (
    '00000000-0000-0000-0000-000000000001'::UUID,
    'Hire Date Not Future',
    'Validates that hire date is not in the future',
    'range',
    'error',
    'Hire date cannot be in the future',
    'The hire date should be today or a past date',
    'compare',
    '{"operator": "lte", "comparison_type": "date", "value": "TODAY()"}'::jsonb,
    true,
    true,
    true,
    true,
    true
);

-- Rule 5: Salary Range Warning
INSERT INTO validation_rules (
    tenant_id, rule_name, rule_description, rule_category, severity,
    error_message, help_message, condition_type, condition_json,
    execute_client_side, execute_server_side, run_on_blur, run_on_submit, is_active
) VALUES (
    '00000000-0000-0000-0000-000000000001'::UUID,
    'Salary Range Check',
    'Validates salary is within acceptable range for the company',
    'range',
    'warning',
    'Salary should be between $30,000 and $500,000 annually',
    'For unusual salary ranges, additional approval may be required',
    'range',
    '{"min": 30000, "max": 500000, "field_type": "decimal"}'::jsonb,
    true,
    true,
    false,
    true,
    true
);

-- ============================================================================
-- Step 3: Create BO Fields
-- ============================================================================

-- Field 1: Employee ID
INSERT INTO bo_fields (
    bo_id, field_name, field_type, display_label, display_order, section_name,
    help_text, is_required, is_readonly, max_length, validation_rule_ids
) VALUES (
    '550e8400-e29b-41d4-a716-446655440001'::UUID,
    'employee_id',
    'string',
    'Employee ID',
    1,
    'Basic Information',
    'Your unique employee identifier assigned by HR',
    true,
    false,
    20,
    ARRAY[
        (SELECT id FROM validation_rules WHERE rule_name = 'Employee ID Format Validation' LIMIT 1)
    ]
);

-- Field 2: First Name
INSERT INTO bo_fields (
    bo_id, field_name, field_type, display_label, display_order, section_name,
    help_text, is_required, max_length
) VALUES (
    '550e8400-e29b-41d4-a716-446655440001'::UUID,
    'first_name',
    'string',
    'First Name',
    2,
    'Basic Information',
    'Employee''s first name',
    true,
    100
);

-- Field 3: Last Name
INSERT INTO bo_fields (
    bo_id, field_name, field_type, display_label, display_order, section_name,
    help_text, is_required, max_length
) VALUES (
    '550e8400-e29b-41d4-a716-446655440001'::UUID,
    'last_name',
    'string',
    'Last Name',
    3,
    'Basic Information',
    'Employee''s last name',
    true,
    100
);

-- Field 4: Email
INSERT INTO bo_fields (
    bo_id, field_name, field_type, display_label, display_order, section_name,
    help_text, is_required, is_searchable, validation_rule_ids
) VALUES (
    '550e8400-e29b-41d4-a716-446655440001'::UUID,
    'email',
    'string',
    'Email Address',
    4,
    'Contact Information',
    'Corporate email address for official communication',
    true,
    true,
    ARRAY[
        (SELECT id FROM validation_rules WHERE rule_name = 'Email Format Validation' LIMIT 1),
        (SELECT id FROM validation_rules WHERE rule_name = 'Email Uniqueness Check' LIMIT 1)
    ]
);

-- Field 5: Phone
INSERT INTO bo_fields (
    bo_id, field_name, field_type, display_label, display_order, section_name,
    help_text, is_required, max_length
) VALUES (
    '550e8400-e29b-41d4-a716-446655440001'::UUID,
    'phone',
    'string',
    'Phone Number',
    5,
    'Contact Information',
    'Office phone number',
    false,
    20
);

-- Field 6: Hire Date
INSERT INTO bo_fields (
    bo_id, field_name, field_type, display_label, display_order, section_name,
    help_text, is_required, date_format, validation_rule_ids
) VALUES (
    '550e8400-e29b-41d4-a716-446655440001'::UUID,
    'hire_date',
    'date',
    'Hire Date',
    6,
    'Employment Details',
    'Date the employee starts work',
    true,
    'YYYY-MM-DD',
    ARRAY[
        (SELECT id FROM validation_rules WHERE rule_name = 'Hire Date Not Future' LIMIT 1)
    ]
);

-- Field 7: Department
INSERT INTO bo_fields (
    bo_id, field_name, field_type, display_label, display_order, section_name,
    help_text, is_required, reference_bo_id, reference_display_field
) VALUES (
    '550e8400-e29b-41d4-a716-446655440001'::UUID,
    'department',
    'reference',
    'Department',
    7,
    'Employment Details',
    'The department the employee will work in',
    true,
    (SELECT id FROM business_objects WHERE bo_name = 'Department' LIMIT 1),
    'department_name'
);

-- Field 8: Salary
INSERT INTO bo_fields (
    bo_id, field_name, field_type, display_label, display_order, section_name,
    help_text, is_required, decimal_places, min_value, max_value, validation_rule_ids
) VALUES (
    '550e8400-e29b-41d4-a716-446655440001'::UUID,
    'salary',
    'decimal',
    'Annual Salary',
    8,
    'Compensation',
    'Annual base salary (in USD)',
    true,
    2,
    '30000',
    '500000',
    ARRAY[
        (SELECT id FROM validation_rules WHERE rule_name = 'Salary Range Check' LIMIT 1)
    ]
);

-- Field 9: Employment Status
INSERT INTO bo_fields (
    bo_id, field_name, field_type, display_label, display_order, section_name,
    help_text, is_required, picklist_values, default_value
) VALUES (
    '550e8400-e29b-41d4-a716-446655440001'::UUID,
    'employment_status',
    'picklist',
    'Employment Status',
    9,
    'Employment Details',
    'Current employment status',
    true,
    ARRAY['Full-Time', 'Part-Time', 'Contract', 'Temporary'],
    'Full-Time'
);

-- ============================================================================
-- Step 4: Create Page Layout
-- ============================================================================

INSERT INTO page_layouts (
    tenant_id, bo_id, layout_name, layout_type, layout_description,
    default_columns, is_default_layout, is_active
) VALUES (
    '00000000-0000-0000-0000-000000000001'::UUID,
    '550e8400-e29b-41d4-a716-446655440001'::UUID,
    'Employee Onboarding Form',
    'form',
    'Standard form for hiring new employees',
    2,
    true,
    true
);

-- Get layout ID:
-- @layout_id = (SELECT id FROM page_layouts WHERE layout_name = 'Employee Onboarding Form' LIMIT 1);

-- ============================================================================
-- Step 5: Create Layout Sections
-- ============================================================================

-- Section 1: Basic Information
INSERT INTO layout_sections (
    layout_id, section_order, section_title, section_description, section_columns,
    is_collapsible, has_border, is_visible, help_text,
    field_ids
) VALUES (
    '660e8400-e29b-41d4-a716-446655440001'::UUID,
    1,
    'Basic Information',
    'Enter the employee''s basic personal information',
    2,
    false,
    true,
    true,
    'These fields are required to create the employee record',
    ARRAY[
        (SELECT id FROM bo_fields WHERE field_name = 'employee_id' LIMIT 1),
        (SELECT id FROM bo_fields WHERE field_name = 'first_name' LIMIT 1),
        (SELECT id FROM bo_fields WHERE field_name = 'last_name' LIMIT 1)
    ]
);

-- Section 2: Contact Information
INSERT INTO layout_sections (
    layout_id, section_order, section_title, section_description, section_columns,
    is_collapsible, has_border, is_visible, help_text,
    field_ids
) VALUES (
    '660e8400-e29b-41d4-a716-446655440001'::UUID,
    2,
    'Contact Information',
    'Employee contact details',
    2,
    false,
    true,
    true,
    'Email is required for corporate communication',
    ARRAY[
        (SELECT id FROM bo_fields WHERE field_name = 'email' LIMIT 1),
        (SELECT id FROM bo_fields WHERE field_name = 'phone' LIMIT 1)
    ]
);

-- Section 3: Employment Details
INSERT INTO layout_sections (
    layout_id, section_order, section_title, section_description, section_columns,
    is_collapsible, has_border, is_visible, help_text,
    field_ids
) VALUES (
    '660e8400-e29b-41d4-a716-446655440001'::UUID,
    3,
    'Employment Details',
    'Employment-related information',
    2,
    false,
    true,
    true,
    'Select the department and employment status',
    ARRAY[
        (SELECT id FROM bo_fields WHERE field_name = 'hire_date' LIMIT 1),
        (SELECT id FROM bo_fields WHERE field_name = 'department' LIMIT 1),
        (SELECT id FROM bo_fields WHERE field_name = 'employment_status' LIMIT 1)
    ]
);

-- Section 4: Compensation
INSERT INTO layout_sections (
    layout_id, section_order, section_title, section_description, section_columns,
    is_collapsible, is_initially_collapsed, has_border, is_visible,
    help_text, background_color,
    field_ids
) VALUES (
    '660e8400-e29b-41d4-a716-446655440001'::UUID,
    4,
    'Compensation',
    'Salary and compensation details',
    1,
    true,
    false,
    true,
    true,
    'Salary information is sensitive and collapsible by default',
    '#F5F5F5',
    ARRAY[
        (SELECT id FROM bo_fields WHERE field_name = 'salary' LIMIT 1)
    ]
);

-- ============================================================================
-- Step 6: Create Layout Actions
-- ============================================================================

-- Action 1: Save Draft
INSERT INTO layout_actions (
    layout_id, action_order, action_label, action_type, action_icon,
    requires_validation, button_style, button_size,
    success_message, error_message
) VALUES (
    '660e8400-e29b-41d4-a716-446655440001'::UUID,
    1,
    'Save Draft',
    'save',
    'save',
    false,
    'secondary',
    'medium',
    'Draft saved successfully',
    'Failed to save draft'
);

-- Action 2: Submit for Manager Approval
INSERT INTO layout_actions (
    layout_id, action_order, action_label, action_type, action_icon,
    requires_validation, requires_confirmation, confirmation_message,
    triggers_bp_id, button_style, button_size,
    success_message, error_message, redirect_on_success
) VALUES (
    '660e8400-e29b-41d4-a716-446655440001'::UUID,
    2,
    'Submit for Approval',
    'submit',
    'send',
    true,
    true,
    'Are you sure? This will route to the manager for approval.',
    '770e8400-e29b-41d4-a716-446655440001'::UUID,  -- bp_hire_employee ID
    'primary',
    'medium',
    'Submitted successfully! You will receive an email when approved.',
    'Submission failed. Please fix validation errors.',
    '/workflows'
);

-- Action 3: Cancel
INSERT INTO layout_actions (
    layout_id, action_order, action_label, action_type, action_icon,
    requires_validation, button_style, button_size
) VALUES (
    '660e8400-e29b-41d4-a716-446655440001'::UUID,
    3,
    'Cancel',
    'cancel',
    'close',
    false,
    'danger',
    'medium'
);

-- ============================================================================
-- SUMMARY
-- ============================================================================
-- You now have a complete HireEmployee form with:
--
-- ✅ 1 Business Object (Employee)
-- ✅ 9 Fields (ID, names, contact, employment, compensation)
-- ✅ 5 Validation Rules (format, uniqueness, range)
-- ✅ 1 Page Layout
-- ✅ 4 Layout Sections
-- ✅ 3 Form Actions (Save, Submit, Cancel)
--
-- To use this form:
--
-- 1. Update tenant_id with your actual tenant UUID
-- 2. Ensure Department BO exists (referenced by department field)
-- 3. Create bp_hire_employee Business Process
-- 4. Frontend calls: GET /api/ui/forms/layout_id
-- 5. Form renders with all validation rules
-- 6. User submits → triggers BP workflow
-- 7. Workflow executes: Validate → Approve → Branch → Notify → Integrate
--
-- The entire flow is metadata-driven, zero-code!
-- ============================================================================
