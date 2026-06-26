# 🚀 Workday-Style Dynamic UI System - Deployment Guide

## Current Status

✅ **Backend Complete**:
- `ui_generator.go` - Zero compilation errors
- `ui_handler.go` - Zero compilation errors
- Database schema ready to deploy
- Example configuration ready

⏳ **Pending**:
- Database initialization
- Example data loading
- API endpoint testing
- React frontend implementation

---

## 🔧 Step 1: Deploy Database Schema

### Issue: Why the exit code 1?

The last command that failed was likely trying to deploy the schema but encountered an issue. Let's diagnose and fix it.

### Check Database Connection

```bash
# First, verify PostgreSQL is running
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable -c "SELECT version();"
```

**Expected output**: PostgreSQL version info

### Deploy Schema Step-by-Step

```bash
# Step 1.1: Navigate to project
cd /Users/eganpj/GitHub/semlayer

# Step 1.2: Find the schema file
ls -la backend/db/migrations/workday_metadata_schema.sql

# Step 1.3: Run the schema deployment
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable \
  -f backend/db/migrations/workday_metadata_schema.sql

# Expected output:
# CREATE TABLE
# CREATE INDEX
# ... (multiple times)
# GRANT
# ... (multiple times)
```

### Verify Schema Created

```bash
# List all tables
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable -c "\dt"

# Expected output should show:
# business_objects
# bo_fields
# validation_rules
# page_layouts
# layout_sections
# layout_actions
# field_validation_rules
# form_submissions
# field_dependencies
# visibility_rules
# layout_customizations

# Count tables
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable -c "\dt" | wc -l

# Should output: 15 (11 tables + 3 lines of headers + 1 blank)
```

### Common Issues & Solutions

#### ❌ Issue: "psql: command not found"

**Solution**: Install psql via Homebrew
```bash
brew install postgresql
```

#### ❌ Issue: "could not connect to server"

**Solution**: PostgreSQL is not running. Start it:
```bash
# If you have docker-compose
docker-compose up -d postgres

# Or if PostgreSQL is installed locally
brew services start postgresql

# Check if running
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable -c "SELECT 1;"
```

#### ❌ Issue: "FATAL: database 'alpha' does not exist"

**Solution**: Create the database first
```bash
psql postgres://postgres:postgres@localhost:5432 -c "CREATE DATABASE alpha;"
```

#### ❌ Issue: "ERROR: relation 'business_objects' already exists"

**Solution**: Schema already deployed. Skip to Step 2.

#### ❌ Issue: "ERROR: type 'jsonb' does not exist"

**Solution**: PostgreSQL version too old (need 9.4+). Check version:
```bash
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable -c "SELECT version();"
```

#### ❌ Issue: Permission errors on GRANT statements

**Solution**: The schema tries to grant permissions to app_user. Run this first:
```bash
# Create app_user if it doesn't exist
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable -c "
  DO \$\$ BEGIN
    CREATE USER app_user WITH PASSWORD 'app_password';
  EXCEPTION WHEN DUPLICATE_OBJECT THEN
    ALTER USER app_user WITH PASSWORD 'app_password';
  END
  \$\$;
"

# Then run the schema
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable \
  -f backend/db/migrations/workday_metadata_schema.sql
```

---

## 🔧 Step 2: Load Example Data

Once schema is deployed, populate with HireEmployee example:

```bash
# Run example setup
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable \
  -f backend/db/migrations/example_hire_employee_setup.sql

# Expected output:
# INSERT 0 1
# INSERT 0 1
# ... (multiple times)
```

### Verify Example Data

```bash
# Check Business Objects
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable -c "
  SELECT bo_name, entity_type, is_active FROM business_objects;
"

# Expected: Employee | employee | t

# Check Fields
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable -c "
  SELECT bo_id, field_name, field_type, is_required 
  FROM bo_fields 
  ORDER BY display_order;
"

# Expected: 9 rows (employee_id, first_name, last_name, email, phone, hire_date, department, salary, employment_status)

# Check Validation Rules
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable -c "
  SELECT rule_name, condition_type, severity 
  FROM validation_rules;
"

# Expected: 5 rows (Employee ID Format, Email Format, Email Uniqueness, Hire Date Not Future, Salary Range)

# Check Page Layouts
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable -c "
  SELECT layout_name, layout_type, bo_id 
  FROM page_layouts;
"

# Expected: Employee Onboarding Form | form | <bo_id>

# Check Sections
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable -c "
  SELECT section_title, columns, array_length(field_ids, 1) as field_count 
  FROM layout_sections 
  ORDER BY section_order;
"

# Expected: 4 rows with section titles and field counts
```

---

## 🔧 Step 3: Start Backend API

Before testing endpoints, ensure the backend is running:

```bash
# From project root
cd /Users/eganpj/GitHub/semlayer

# Build backend
go build -o ./bin/api ./cmd/api

# Start backend (runs on :8080)
./bin/api

# Expected output:
# Starting Semlayer API...
# Listening on :8080
```

### In another terminal, verify it's running:

```bash
curl http://localhost:8080/health

# Expected: {"status":"ok"}
```

---

## 🧪 Step 4: Test Form Definition Endpoint

Get the layout ID first:

```bash
# Get layout ID
LAYOUT_ID=$(psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable \
  -t -c "SELECT id FROM page_layouts WHERE layout_name = 'Employee Onboarding Form' LIMIT 1;")

echo "Layout ID: $LAYOUT_ID"
```

### Test endpoint:

```bash
# Get the tenant ID and datasource ID from the database
TENANT_ID="00000000-0000-0000-0000-000000000001"  # Use a valid tenant
DATASOURCE_ID="11111111-1111-1111-1111-111111111111"  # Use a valid datasource

# Call endpoint
curl -v \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
  "http://localhost:8080/api/ui/forms/$LAYOUT_ID"

# Expected response (200 OK):
{
  "id": "<layout_id>",
  "business_object": {
    "id": "<bo_id>",
    "bo_name": "Employee",
    "bo_description": "Core employee information",
    "fields": [
      {
        "id": "<field_id>",
        "field_name": "employee_id",
        "field_type": "string",
        "display_label": "Employee ID",
        "is_required": true,
        "validation_rule_ids": ["<rule_id>"]
      },
      ... (8 more fields)
    ]
  },
  "sections": [
    {
      "id": "<section_id>",
      "section_title": "Basic Information",
      "columns": 2,
      "field_ids": ["<field_id>", "<field_id>"]
    },
    ... (3 more sections)
  ],
  "actions": [
    {
      "action_label": "Save Draft",
      "action_type": "save",
      "requires_validation": false
    },
    {
      "action_label": "Submit for Approval",
      "action_type": "submit",
      "requires_validation": true,
      "triggers_bp_id": "bp_hire_employee"
    },
    {
      "action_label": "Cancel",
      "action_type": "cancel"
    }
  ]
}
```

### Save response for next tests:

```bash
# Save the layout ID for reuse
export LAYOUT_ID="<from-above>"
export TENANT_ID="00000000-0000-0000-0000-000000000001"
export DATASOURCE_ID="11111111-1111-1111-1111-111111111111"
```

---

## 🧪 Step 5: Test Validation Endpoint

### Test Valid Data:

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
  -d '{
    "bo_id": "<bo_id-from-previous-response>",
    "data": {
      "employee_id": "EMP123456",
      "first_name": "John",
      "last_name": "Doe",
      "email": "john.doe@company.com",
      "phone": "555-1234",
      "hire_date": "2024-01-15",
      "department": "Engineering",
      "employment_status": "Full-Time",
      "salary": 150000
    }
  }' \
  http://localhost:8080/api/ui/validate

# Expected response (200 OK):
{
  "valid": true,
  "errors": [],
  "warnings": [
    {
      "field_id": "salary_field_id",
      "field_name": "salary",
      "severity": "warning",
      "message": "Salary should be between $30,000 and $500,000"
    }
  ]
}

# Note: Salary is in range so no warning, but if it were $600K, there would be a warning
```

### Test Invalid Data (Employee ID format):

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
  -d '{
    "bo_id": "<bo_id>",
    "data": {
      "employee_id": "INVALID",
      "first_name": "John",
      "last_name": "Doe",
      "email": "john.doe@company.com",
      "phone": "555-1234",
      "hire_date": "2024-01-15",
      "department": "Engineering",
      "employment_status": "Full-Time",
      "salary": 150000
    }
  }' \
  http://localhost:8080/api/ui/validate

# Expected response (200 OK, but valid=false):
{
  "valid": false,
  "errors": [
    {
      "field_id": "employee_id_field_id",
      "field_name": "employee_id",
      "severity": "error",
      "message": "Employee ID must be in format EMP followed by 6 digits"
    }
  ],
  "warnings": []
}
```

### Test Invalid Email Format:

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
  -d '{
    "bo_id": "<bo_id>",
    "data": {
      "employee_id": "EMP123456",
      "first_name": "John",
      "last_name": "Doe",
      "email": "not-an-email",
      "phone": "555-1234",
      "hire_date": "2024-01-15",
      "department": "Engineering",
      "employment_status": "Full-Time",
      "salary": 150000
    }
  }' \
  http://localhost:8080/api/ui/validate

# Expected response (200 OK, but valid=false):
{
  "valid": false,
  "errors": [
    {
      "field_id": "email_field_id",
      "field_name": "email",
      "severity": "error",
      "message": "Please enter a valid email address"
    }
  ],
  "warnings": []
}
```

### Test Required Field Missing:

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
  -d '{
    "bo_id": "<bo_id>",
    "data": {
      "first_name": "John",
      "last_name": "Doe",
      "email": "john.doe@company.com"
    }
  }' \
  http://localhost:8080/api/ui/validate

# Expected response (200 OK, but valid=false):
{
  "valid": false,
  "errors": [
    {
      "field_id": "employee_id_field_id",
      "field_name": "employee_id",
      "severity": "error",
      "message": "employee_id is required"
    },
    {
      "field_id": "hire_date_field_id",
      "field_name": "hire_date",
      "severity": "error",
      "message": "hire_date is required"
    },
    ... (more required fields)
  ],
  "warnings": []
}
```

---

## 🧪 Step 6: Test Form Submission (Save)

Save form data without triggering BP:

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
  -d '{
    "bo_id": "<bo_id>",
    "data": {
      "employee_id": "EMP789012",
      "first_name": "Jane",
      "last_name": "Smith",
      "email": "jane.smith@company.com",
      "phone": "555-5678",
      "hire_date": "2024-02-01",
      "department": "Marketing",
      "employment_status": "Full-Time",
      "salary": 120000
    }
  }' \
  http://localhost:8080/api/ui/save

# Expected response (200 OK):
{
  "record_id": "emp_abc123xyz",
  "status": "saved",
  "message": "Form data saved successfully"
}
```

### Verify Data Saved:

```bash
# Check form_submissions table
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable -c "
  SELECT submission_id, bo_id, validation_passed, status, submitted_at 
  FROM form_submissions 
  ORDER BY submitted_at DESC 
  LIMIT 1;
"

# Expected: One row with status='saved'
```

---

## 🧪 Step 7: Test Form Submission (With BP Trigger)

Submit form and trigger Business Process:

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
  -d '{
    "bo_id": "<bo_id>",
    "bp_id": "bp_hire_employee",
    "data": {
      "employee_id": "EMP345678",
      "first_name": "Bob",
      "last_name": "Johnson",
      "email": "bob.johnson@company.com",
      "phone": "555-9999",
      "hire_date": "2024-03-15",
      "department": "Finance",
      "employment_status": "Full-Time",
      "salary": 95000
    }
  }' \
  http://localhost:8080/api/ui/submit

# Expected response (200 OK):
{
  "record_id": "emp_def456ghi",
  "workflow_id": "bp_hire_employee_def456ghi",
  "status": "submitted",
  "message": "Form submitted for approval. Temporal workflow started."
}
```

### Check Temporal Workflow:

```bash
# If you have Temporal UI running (usually http://localhost:8088)
open http://localhost:8088

# Look for workflow with ID: bp_hire_employee_def456ghi
# Status should be: Running
```

### Check Form Submission Record:

```bash
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable -c "
  SELECT submission_id, workflow_id, validation_passed, status, submitted_at 
  FROM form_submissions 
  WHERE submission_id = 'emp_def456ghi';
"

# Expected: status='submitted', workflow_id='bp_hire_employee_def456ghi'
```

---

## 🎨 Step 8: Build React Frontend

Once backend is tested, build the React component:

### Create Form Hook

```typescript
// frontend/src/hooks/useFormDefinition.ts
import { useQuery, useMutation } from '@tanstack/react-query';

export function useFormDefinition(layoutId: string) {
  return useQuery({
    queryKey: ['form-definition', layoutId],
    queryFn: async () => {
      const response = await fetch(
        `/api/ui/forms/${layoutId}?tenant_id=${getTenantId()}&datasource_id=${getDatasourceId()}`,
        {
          headers: {
            'X-Tenant-ID': getTenantId(),
            'X-Tenant-Datasource-ID': getDatasourceId(),
          }
        }
      );
      if (!response.ok) throw new Error('Failed to load form');
      return response.json();
    }
  });
}

export function useFormValidation(boId: string) {
  return useMutation({
    mutationFn: async (data: any) => {
      const response = await fetch('/api/ui/validate', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': getTenantId(),
          'X-Tenant-Datasource-ID': getDatasourceId(),
        },
        body: JSON.stringify({ bo_id: boId, data })
      });
      if (!response.ok) throw new Error('Validation failed');
      return response.json();
    }
  });
}

export function useFormSubmit() {
  return useMutation({
    mutationFn: async (data: any) => {
      const response = await fetch('/api/ui/submit', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': getTenantId(),
          'X-Tenant-Datasource-ID': getDatasourceId(),
        },
        body: JSON.stringify(data)
      });
      if (!response.ok) throw new Error('Submission failed');
      return response.json();
    }
  });
}

function getTenantId(): string {
  return localStorage.getItem('selected_tenant_id') || '';
}

function getDatasourceId(): string {
  return localStorage.getItem('selected_datasource_id') || '';
}
```

### Create Form Component

See `/WORKDAY_UI_IMPLEMENTATION_COMPLETE.md` for React component examples.

---

## 📋 Deployment Checklist

- [ ] PostgreSQL running
- [ ] Database `alpha` created
- [ ] User `app_user` created
- [ ] Schema deployed (11 tables)
- [ ] Example data loaded
- [ ] Backend compiled and running
- [ ] GET /api/ui/forms/:layoutId returns FormDefinition
- [ ] POST /api/ui/validate validates data correctly
- [ ] POST /api/ui/save saves to form_submissions
- [ ] POST /api/ui/submit triggers Temporal workflow
- [ ] React frontend component built
- [ ] Frontend hooks wire to backend
- [ ] Multi-tenant scoping verified (X-Tenant-ID headers)
- [ ] Audit trail recording (form_submissions populated)

---

## 🐛 Troubleshooting

### Backend won't compile

```bash
# Check for syntax errors
cd /Users/eganpj/GitHub/semlayer
go build ./cmd/api

# Check for missing imports
go mod tidy
go mod download
```

### Endpoints return 500 error

```bash
# Check backend logs
tail -f backend.log

# Common issues:
# - Database connection failed
# - Missing tenant_id header
# - BO/layout not found
```

### Validation not working

```bash
# Verify validation rules exist
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable -c "
  SELECT rule_name, condition_type, condition_json 
  FROM validation_rules;
"

# Check condition_json format (should be valid JSON)
```

### Form submission not triggering workflow

```bash
# Verify Temporal is running
curl http://localhost:7233/health

# Check Temporal logs for errors
# Ensure bp_id matches a real business process
```

---

## 📚 Next Steps

1. ✅ Deploy database schema
2. ✅ Load example data
3. ✅ Test all API endpoints
4. 🔄 Build React frontend (see Step 8 above)
5. 🔄 Integration testing
6. 🔄 Performance testing
7. 🔄 Security audit
8. 🔄 Production deployment

**Happy deploying!** 🚀
