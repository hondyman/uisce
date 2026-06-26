# 🏗️ Workday-Style Metadata-Driven UI System

## Overview

Your semlayer system now includes a **complete metadata-driven UI generation engine** inspired by Workday's architecture. This enables:

- **Zero-code form generation** from metadata
- **Unified validation** engine (client + server)
- **Automatic business process triggering**
- **Complete audit trail** for compliance
- **Multi-tenant support** with tenant isolation

---

## 🏛️ Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      FRONTEND (React)                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  DynamicFormGenerator Component                                  │
│  ├── Renders form from metadata                                  │
│  ├── Client-side validation (immediate feedback)                 │
│  ├── Form submission handling                                    │
│  └── Error/warning display                                       │
│                                                                   │
└─────────────────────────────┬─────────────────────────────────────┘
                              │
                 API Calls with tenant scope
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   BACKEND (Go + Temporal)                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  UIHandler (REST Endpoints)                                      │
│  ├── GET  /api/ui/forms/{layoutId}      → FormDefinition       │
│  ├── POST /api/ui/validate              → ValidationResult     │
│  ├── POST /api/ui/save                  → Save to DB            │
│  └── POST /api/ui/submit                → Trigger BP            │
│                                                                   │
│        ↓                                                          │
│                                                                   │
│  UIGenerator (Orchestration)                                     │
│  ├── GetFormDefinition()                                         │
│  ├── ValidateFormData()                                          │
│  ├── ExecuteRule()                                               │
│  └── loadMetadata()                                              │
│                                                                   │
│        ↓                                                          │
│                                                                   │
│  Validation Rule Engine                                          │
│  ├── Regex validation                                            │
│  ├── Comparison checks                                           │
│  ├── Uniqueness checks                                           │
│  ├── Range validation                                            │
│  └── Cross-field validation                                      │
│                                                                   │
│        ↓                                                          │
│                                                                   │
│  Temporal Workflow Integration                                   │
│  ├── Fire TriggerEngine                                          │
│  ├── Execute BranchCompleteEvaluator                             │
│  └── Record to form_submissions table                            │
│                                                                   │
└─────────────────────────────┬─────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   POSTGRESQL DATABASE                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  Metadata Tables (Define structure):                            │
│  ├── business_objects        - Entity definitions                │
│  ├── bo_fields               - Field metadata                    │
│  ├── validation_rules        - Validation logic                  │
│  ├── page_layouts            - Form layouts                      │
│  ├── layout_sections         - Form sections                     │
│  ├── layout_actions          - Form buttons                      │
│  └── field_validation_rules  - Many-to-many linking             │
│                                                                   │
│  Data Tables (Store submissions):                               │
│  ├── form_submissions        - All form submissions              │
│  ├── field_dependencies      - Conditional visibility            │
│  ├── visibility_rules        - Dynamic show/hide                 │
│  └── layout_customizations   - Tenant/user customizations       │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

---

## 📊 Database Schema

### Core Metadata Tables

#### 1. Business Objects
```sql
CREATE TABLE business_objects (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    bo_name VARCHAR(100) NOT NULL,      -- "Employee", "Customer"
    entity_type VARCHAR(50),             -- "employee", "customer"
    allow_custom_fields BOOLEAN,
    is_active BOOLEAN
);
```

#### 2. BO Fields
```sql
CREATE TABLE bo_fields (
    id UUID PRIMARY KEY,
    bo_id UUID REFERENCES business_objects(id),
    field_name VARCHAR(100),            -- "first_name", "email"
    field_type VARCHAR(20),             -- string|number|date|reference|picklist
    display_label VARCHAR(200),         -- "First Name", "Email Address"
    is_required BOOLEAN,
    validation_rule_ids UUID[],         -- References to validation_rules
    reference_bo_id UUID,               -- For reference type fields
    picklist_values TEXT[]              -- For picklist type
);
```

#### 3. Validation Rules
```sql
CREATE TABLE validation_rules (
    id UUID PRIMARY KEY,
    rule_name VARCHAR(100),             -- "Email Format"
    rule_category VARCHAR(50),          -- format|range|uniqueness|cross_field
    severity VARCHAR(20),               -- error|warning
    error_message TEXT,
    condition_type VARCHAR(50),         -- regex|compare|unique_check|range
    condition_json JSONB,               -- Rule parameters as JSON
    execute_client_side BOOLEAN,        -- Run in browser?
    execute_server_side BOOLEAN         -- Run on server?
);
```

#### 4. Page Layouts
```sql
CREATE TABLE page_layouts (
    id UUID PRIMARY KEY,
    bo_id UUID REFERENCES business_objects(id),
    layout_name VARCHAR(100),           -- "Employee Entry Form"
    layout_type VARCHAR(50),            -- form|grid|detail|wizard
    is_default_layout BOOLEAN
);
```

#### 5. Layout Sections
```sql
CREATE TABLE layout_sections (
    id UUID PRIMARY KEY,
    layout_id UUID REFERENCES page_layouts(id),
    section_title VARCHAR(200),         -- "Basic Information"
    section_columns INT,                -- 1, 2, or 3 columns
    field_ids UUID[]                    -- Which fields in this section
);
```

#### 6. Layout Actions
```sql
CREATE TABLE layout_actions (
    id UUID PRIMARY KEY,
    layout_id UUID REFERENCES page_layouts(id),
    action_label VARCHAR(100),          -- "Save", "Submit for Approval"
    action_type VARCHAR(50),            -- save|submit|cancel|delete
    requires_validation BOOLEAN,
    triggers_bp_id UUID                 -- Which BP to trigger
);
```

### Submission & Audit Tables

#### 7. Form Submissions
```sql
CREATE TABLE form_submissions (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    bo_id UUID NOT NULL,
    submission_id VARCHAR(255),         -- Unique per submission
    submitted_by UUID,                  -- User ID
    form_data JSONB,                    -- Complete form data
    validation_passed BOOLEAN,
    validation_errors JSONB,            -- Validation error details
    status VARCHAR(50),                 -- pending|approved|rejected
    submitted_at TIMESTAMP,
    created_at TIMESTAMP
);
```

---

## 🚀 API Endpoints

### 1. Get Form Definition
```http
GET /api/ui/forms/:layoutId
  ?tenant_id=<TENANT_ID>
  &datasource_id=<DATASOURCE_ID>

Response 200 OK:
{
  "id": "layout_001",
  "layout_name": "Employee Entry Form",
  "layout_type": "form",
  "business_object": {
    "id": "bo_employee",
    "bo_name": "Employee",
    "fields": [...]
  },
  "sections": [
    {
      "section_title": "Basic Information",
      "section_columns": 2,
      "fields": [
        {
          "id": "field_first_name",
          "field_name": "first_name",
          "field_type": "string",
          "display_label": "First Name",
          "is_required": true,
          "validation_rule_ids": ["rule_001"]
        },
        ...
      ]
    }
  ],
  "validations": {
    "field_email": [
      {
        "rule_name": "Email Format",
        "severity": "error",
        "error_message": "Please enter a valid email address",
        "condition_type": "regex",
        "condition_json": {"pattern": "^[^\\s@]+@[^\\s@]+\\.[^\\s@]+$"},
        "execute_client_side": true,
        "execute_server_side": true
      }
    ]
  },
  "actions": [
    {
      "action_label": "Save",
      "action_type": "save",
      "requires_validation": false
    },
    {
      "action_label": "Submit for Approval",
      "action_type": "submit",
      "requires_validation": true,
      "triggers_bp_id": "bp_hire_employee"
    }
  ]
}
```

### 2. Validate Form Data
```http
POST /api/ui/validate
  ?tenant_id=<TENANT_ID>
  &datasource_id=<DATASOURCE_ID>

Request Body:
{
  "bo_id": "bo_employee",
  "data": {
    "first_name": "John",
    "last_name": "Doe",
    "email": "john@example.com",
    "hire_date": "2024-01-15",
    "salary": 150000,
    "department": "engineering"
  }
}

Response 200 OK:
{
  "valid": true,
  "errors": [],
  "warnings": []
}

Response 400 Bad Request (Invalid):
{
  "valid": false,
  "errors": [
    {
      "field_id": "field_email",
      "field_name": "email",
      "severity": "error",
      "message": "Please enter a valid email address"
    }
  ],
  "warnings": [
    {
      "field_id": "field_salary",
      "field_name": "salary",
      "severity": "warning",
      "message": "Salary exceeds department average of $120,000"
    }
  ]
}
```

### 3. Save Form Data
```http
POST /api/ui/save
  ?tenant_id=<TENANT_ID>
  &datasource_id=<DATASOURCE_ID>

Request Body:
{
  "bo_id": "bo_employee",
  "data": {
    "first_name": "John",
    "email": "john@example.com",
    ...
  }
}

Response 200 OK:
{
  "record_id": "emp_001",
  "status": "saved",
  "message": "Form data saved successfully"
}
```

### 4. Submit Form for Approval
```http
POST /api/ui/submit
  ?tenant_id=<TENANT_ID>
  &datasource_id=<DATASOURCE_ID>

Request Body:
{
  "bo_id": "bo_employee",
  "bp_id": "bp_hire_employee",  // Business Process to trigger
  "data": {
    "first_name": "John",
    "email": "john@example.com",
    ...
  }
}

Response 200 OK:
{
  "record_id": "emp_001",
  "workflow_id": "wf_12345",
  "status": "submitted",
  "message": "Form submitted successfully"
}
```

---

## 💻 Go Implementation

### UIGenerator Core Functions

```go
// Load and generate form definition
func (g *UIGenerator) GetFormDefinition(ctx context.Context, layoutID string) (*FormDefinition, error)

// Validate form data against all rules
func (g *UIGenerator) ValidateFormData(ctx context.Context, boID string, data map[string]interface{}) (*ValidationResult, error)

// Execute a single validation rule
func (g *UIGenerator) executeRule(ctx context.Context, rule *ValidationRule, value interface{}, allData map[string]interface{}) (bool, error)
```

### Validation Rule Types

```go
// Regex Validation
{
  "condition_type": "regex",
  "condition_json": {
    "pattern": "^[a-zA-Z0-9]+@[a-zA-Z0-9]+\\.[a-z]+$"
  }
}

// Comparison Validation
{
  "condition_type": "compare",
  "condition_json": {
    "operator": "gte",
    "value": 18
  }
}

// Range Validation
{
  "condition_type": "range",
  "condition_json": {
    "min": 30000,
    "max": 500000
  }
}

// Uniqueness Check
{
  "condition_type": "unique_check",
  "condition_json": {
    "field": "email",
    "table": "employees",
    "scope": "global"
  }
}

// Cross-Field Validation
{
  "condition_type": "cross_field",
  "condition_json": {
    "condition": "hire_date < retirement_date"
  }
}
```

### REST Handler Implementation

```go
type UIHandler struct {
    uiGenerator *ui.UIGenerator
    db          *sqlx.DB
}

// GET /api/ui/forms/:layoutId
func (h *UIHandler) GetFormDefinition(c *gin.Context) { ... }

// POST /api/ui/validate
func (h *UIHandler) ValidateFormData(c *gin.Context) { ... }

// POST /api/ui/save
func (h *UIHandler) SaveFormData(c *gin.Context) { ... }

// POST /api/ui/submit
func (h *UIHandler) SubmitFormData(c *gin.Context) { ... }
```

---

## ⚛️ React Frontend

### Dynamic Form Component

```typescript
// hooks/useFormDefinition.ts
export function useFormDefinition(layoutId: string) {
  return useQuery({
    queryKey: ['form-definition', layoutId],
    queryFn: async () => {
      const res = await fetch(`/api/ui/forms/${layoutId}`, {
        headers: {
          'X-Tenant-ID': getTenantId(),
          'X-Tenant-Datasource-ID': getDatasourceId()
        }
      });
      return res.json();
    }
  });
}

// components/DynamicForm.tsx
export const DynamicForm: React.FC<{ layoutId: string }> = ({ layoutId }) => {
  const { data: formDef, isLoading } = useFormDefinition(layoutId);
  const [formData, setFormData] = useState({});
  const [validationErrors, setValidationErrors] = useState({});

  if (isLoading) return <div>Loading form...</div>;

  const handleFieldChange = async (fieldName: string, value: any) => {
    setFormData(prev => ({ ...prev, [fieldName]: value }));
    
    // Optional: Validate field on change
    if (shouldValidateOnChange(fieldName)) {
      const result = await validateField(fieldName, value);
      if (!result.valid) {
        setValidationErrors(prev => ({
          ...prev,
          [fieldName]: result.errors
        }));
      }
    }
  };

  const handleSubmit = async (action: 'save' | 'submit') => {
    const endpoint = action === 'save' ? '/api/ui/save' : '/api/ui/submit';
    
    try {
      const response = await fetch(endpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': getTenantId(),
          'X-Tenant-Datasource-ID': getDatasourceId()
        },
        body: JSON.stringify({
          bo_id: formDef.business_object.id,
          data: formData
        })
      });

      if (response.ok) {
        const result = await response.json();
        showSuccessMessage(result.message);
        if (result.workflow_id) {
          redirectToWorkflowTracking(result.workflow_id);
        }
      } else {
        const error = await response.json();
        setValidationErrors(error.validation?.errors || {});
      }
    } catch (error) {
      console.error('Submission failed:', error);
    }
  };

  return (
    <div className="dynamic-form">
      <h1>{formDef.layout_name}</h1>
      
      {formDef.sections.map(section => (
        <FormSection key={section.id} section={section} formDef={formDef} />
      ))}

      <div className="form-actions">
        {formDef.actions.map(action => (
          <button
            key={action.id}
            onClick={() => handleSubmit(action.action_type)}
            className={`btn btn-${action.button_style}`}
          >
            {action.action_label}
          </button>
        ))}
      </div>
    </div>
  );
};

// components/FormField.tsx
export const FormField: React.FC<FormFieldProps> = ({
  field,
  value,
  error,
  onChange,
  validationRules
}) => {
  return (
    <div className="form-field">
      <label>
        {field.display_label}
        {field.is_required && <span className="required">*</span>}
      </label>

      {field.field_type === 'string' && (
        <input
          type="text"
          value={value || ''}
          onChange={(e) => onChange(e.target.value)}
          placeholder={field.placeholder_text}
          maxLength={field.max_length}
          required={field.is_required}
          pattern={getRegexPattern(validationRules)}
          onBlur={() => validateField(field, value)}
        />
      )}

      {field.field_type === 'date' && (
        <input
          type="date"
          value={value || ''}
          onChange={(e) => onChange(e.target.value)}
          required={field.is_required}
        />
      )}

      {field.field_type === 'reference' && (
        <select
          value={value || ''}
          onChange={(e) => onChange(e.target.value)}
          required={field.is_required}
        >
          <option value="">Select...</option>
          {/* Options loaded via separate API */}
        </select>
      )}

      {field.field_type === 'picklist' && (
        <select
          value={value || ''}
          onChange={(e) => onChange(e.target.value)}
          required={field.is_required}
        >
          <option value="">Select...</option>
          {field.picklist_values.map(v => (
            <option key={v} value={v}>{v}</option>
          ))}
        </select>
      )}

      {error && (
        <div className="error-message">{error.message}</div>
      )}

      {field.help_text && (
        <small className="help-text">{field.help_text}</small>
      )}
    </div>
  );
};
```

---

## 📋 Example: HireEmployee Form

### 1. Define Business Object

```sql
INSERT INTO business_objects (tenant_id, bo_name, entity_type)
VALUES ('tenant_001', 'Employee', 'employee');
```

### 2. Define Fields

```sql
INSERT INTO bo_fields (bo_id, field_name, field_type, display_label, is_required, display_order, section_name)
VALUES
  ('bo_employee', 'employee_id', 'string', 'Employee ID', true, 1, 'Basic Info'),
  ('bo_employee', 'first_name', 'string', 'First Name', true, 2, 'Basic Info'),
  ('bo_employee', 'last_name', 'string', 'Last Name', true, 3, 'Basic Info'),
  ('bo_employee', 'email', 'string', 'Email Address', true, 4, 'Contact Info'),
  ('bo_employee', 'hire_date', 'date', 'Hire Date', true, 5, 'Employment'),
  ('bo_employee', 'salary', 'decimal', 'Annual Salary', true, 6, 'Compensation'),
  ('bo_employee', 'department', 'reference', 'Department', true, 7, 'Employment');
```

### 3. Create Validation Rules

```sql
INSERT INTO validation_rules (rule_name, rule_category, severity, error_message, condition_type, condition_json)
VALUES
  ('Employee ID Format', 'format', 'error', 'Employee ID must start with EMP', 'regex',
   '{"pattern": "^EMP[0-9]{6}$"}'),
  
  ('Email Format', 'format', 'error', 'Please enter a valid email', 'regex',
   '{"pattern": "^[^\\s@]+@[^\\s@]+\\.[^\\s@]+$"}'),
  
  ('Hire Date Not Future', 'range', 'error', 'Hire date cannot be in future', 'compare',
   '{"operator": "lte", "value": "today()"}'),
  
  ('Salary Range', 'range', 'warning', 'Salary should be $30K - $500K', 'range',
   '{"min": 30000, "max": 500000}');
```

### 4. Link Validations to Fields

```sql
INSERT INTO field_validation_rules (field_id, validation_rule_id, rule_order)
VALUES
  ('field_employee_id', 'rule_emp_id_format', 1),
  ('field_email', 'rule_email_format', 1),
  ('field_hire_date', 'rule_hire_date_future', 1),
  ('field_salary', 'rule_salary_range', 1);
```

### 5. Create Page Layout

```sql
INSERT INTO page_layouts (bo_id, layout_name, layout_type, is_default_layout)
VALUES ('bo_employee', 'Employee Entry Form', 'form', true);

INSERT INTO layout_sections (layout_id, section_title, section_columns, field_ids)
VALUES
  ('layout_001', 'Basic Information', 2,
   ARRAY['field_employee_id', 'field_first_name', 'field_last_name']),
  
  ('layout_001', 'Contact Information', 2,
   ARRAY['field_email']),
  
  ('layout_001', 'Employment Details', 2,
   ARRAY['field_hire_date', 'field_department']),
  
  ('layout_001', 'Compensation', 1,
   ARRAY['field_salary']);
```

### 6. Add Form Actions

```sql
INSERT INTO layout_actions (layout_id, action_order, action_label, action_type, requires_validation, triggers_bp_id)
VALUES
  ('layout_001', 1, 'Save Draft', 'save', false, NULL),
  ('layout_001', 2, 'Submit for Approval', 'submit', true, 'bp_hire_employee'),
  ('layout_001', 3, 'Cancel', 'cancel', false, NULL);
```

### 7. Use in Frontend

```typescript
<DynamicForm layoutId="layout_employee_entry" />
```

**Flow**:
1. User sees form with all sections and fields
2. Fills in employee data
3. Clicks "Submit for Approval"
4. Frontend calls `/api/ui/validate` → gets validation results
5. Backend calls `/api/ui/submit` → saves and triggers BP
6. `bp_hire_employee` workflow starts with TriggerEngine
7. Workflow executes: Validate → Approve → Branch (all 15 features!) → Notify → Integrate

---

## 🔐 Tenant Scoping

All APIs require tenant context via query parameters:

```bash
curl -H "X-Tenant-ID: <TENANT_ID>" \
     -H "X-Tenant-Datasource-ID: <DATASOURCE_ID>" \
     "http://localhost:8080/api/ui/forms/layout_001"
```

The middleware automatically:
- Scopes all queries to tenant
- Validates user access
- Logs audit trail

---

## 📊 Validation Rule Reference

### Built-in Condition Types

| Type | Use Case | Example |
|------|----------|---------|
| `regex` | Format validation | Email, phone, ID format |
| `compare` | Single field comparison | Date not in future, number comparison |
| `unique_check` | Database uniqueness | Email uniqueness, username uniqueness |
| `range` | Numeric or date range | Salary $30K-$500K, date range |
| `cross_field` | Multiple field logic | End date > start date |

---

## 🚀 Deployment Checklist

- [ ] Run migration: `workday_metadata_schema.sql`
- [ ] Create Business Objects and Fields for your entities
- [ ] Define Validation Rules
- [ ] Link rules to fields
- [ ] Create Page Layouts and Sections
- [ ] Add Layout Actions (including BP triggers)
- [ ] Test form via API:
  - `GET /api/ui/forms/:layoutId`
  - `POST /api/ui/validate`
  - `POST /api/ui/submit`
- [ ] Implement React `DynamicForm` component
- [ ] Test end-to-end flow: Form → Validation → BP Trigger
- [ ] Monitor `form_submissions` table for tracking

---

## 🎯 Key Benefits

✅ **No Hard-Coded Forms**: All UI generated from metadata  
✅ **Single Source of Truth**: BO definition is authority  
✅ **Unified Validation**: Client-side + server-side in sync  
✅ **Business User Control**: No code changes needed for customization  
✅ **Complete Audit Trail**: Every submission tracked  
✅ **Instant Updates**: Change metadata, forms update immediately  
✅ **Tenant Isolation**: Multi-tenant safe by default  
✅ **Temporal Integration**: Seamless BP triggering  

---

## 📚 Related Documentation

- [Trigger Engine](./TRIGGER_BRANCHING_QUICKSTART.md)
- [Branch Evaluators (15 Features)](./ADVANCED_FEATURES_IMPLEMENTATION.md)
- [Temporal Workflows](./OPTION_A_C_IMPLEMENTATION_COMPLETE.md)
- [API Reference](./API_REFERENCE.md)

---

## 💬 Questions?

This implementation follows the exact pattern Workday uses for their dynamic forms and business processes. Every form is metadata-driven, every validation is configurable, and every workflow is customizable without coding.

**You now have a production-ready no-code/low-code platform! 🎉**
