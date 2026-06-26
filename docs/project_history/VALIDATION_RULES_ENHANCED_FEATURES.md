# Validation Rules Enhanced Features Guide

## Overview

The Validation Rules system has been enhanced with three powerful features designed to accelerate rule creation, build user confidence, and prevent deployment mistakes:

1. **Rule Templates** - Pre-built validation patterns for common use cases
2. **Live Preview** - Real-time testing with sample data before deployment
3. **Impact Analysis** - Understand scope and risk before deploying rules

---

## 🚀 Quick Start: 4-Tab Workflow

When creating a new validation rule, users follow an intuitive 4-tab workflow:

```
Templates → Configure → Test → Impact → Save
   ↓          ↓          ↓       ↓
 Pick      Fill in    Try it  Assess   Create
template    details   out    risk
```

### Tab 1: Templates (Optional but Recommended)
- Browse 8 pre-built rule templates organized by category
- Search by keyword, category, or use case
- Click a template to see preview
- Templates auto-populate the configuration form

### Tab 2: Configure (Required)
- Customize rule details (name, entity, field, severity)
- Adjust rule condition using the ConditionBuilder
- Set actions on success/failure
- Shows which template was selected (if any)

### Tab 3: Test (Optional but Valuable)
- Upload or paste sample data (JSON/CSV format)
- Test the rule against sample data
- See which records pass/fail the validation
- Refine the rule if needed
- Build confidence before deployment

### Tab 4: Impact (Optional but Important)
- Analyze how many records will be affected
- See severity breakdown (warnings vs. errors vs. info)
- View recommendations based on risk level
- See which departments/areas are impacted
- Make informed deployment decisions

---

## 📋 Rule Templates

### Location
- **Module**: `frontend/src/data/ruleTemplates.ts`
- **Selector Component**: `frontend/src/components/validation/RuleTemplatesSelector.tsx`

### Available Templates

#### Data Quality Category (🚫)
1. **Not Null Check** - Ensure a field is never empty
   - Perfect for: Customer IDs, dates, amounts
   - Validates that field IS NOT NULL

2. **Uniqueness Check** (🔑) - Ensure field values are unique
   - Perfect for: Email addresses, SSN, account numbers
   - Checks COUNT(*) = 1 for each value

3. **Duplicate Detection** (🔍) - Find duplicate records
   - Perfect for: Finding duplicate customers, transactions
   - Identifies records with same key fields

#### Business Logic Category (📊)
4. **Range/Bounds Check** - Verify numeric field is within acceptable range
   - Perfect for: Age, salary, interest rates
   - Validates value between min and max

5. **Pattern/Format Match** (✓) - Match string patterns
   - Perfect for: Email, phone, postal code
   - Uses regex patterns for validation

6. **Cross-Field Comparison** (⚖️) - Compare multiple fields
   - Perfect for: End date > Start date, Amount <= Limit
   - Custom business logic

#### Referential Integrity Category (🔗)
7. **Lookup/Referential Integrity** - Verify lookup references exist
   - Perfect for: Department exists, Product is valid
   - Checks values against reference tables

8. **Custom Business Rule** (🎯) - Define your own rule
   - Perfect for: Complex multi-field logic
   - Full flexibility for advanced use cases

### Using Templates

#### Programmatic Access
```typescript
import { RULE_TEMPLATES, getTemplatesByCategory, searchTemplates } from '@/data/ruleTemplates';

// Get all templates in a category
const dataQualityRules = getTemplatesByCategory('data-quality');

// Search templates by keyword
const results = searchTemplates('email');

// Get specific template by ID
const template = RULE_TEMPLATES.find(t => t.id === 'not-null');
```

#### In the UI
```tsx
<RuleTemplatesSelector
  onTemplateSelected={(template, rule) => {
    // Template selected - rule object has been populated
    // with template's baseRule properties
  }}
  targetEntity="Customer" // Optional: pre-filter templates for this entity
/>
```

### Template Structure
```typescript
{
  id: string;                          // Unique identifier
  name: string;                        // Display name
  description: string;                 // One-line description
  category: 'data-quality' | 'business-logic' | 'referential-integrity' | 'format-validation';
  icon: string;                        // Emoji icon
  baseRule: Partial<ValidationRule>;  // Pre-configured rule properties
  helpText: string;                    // Extended help text
  commonUse: string[];                 // Example use cases
}
```

---

## ▶️ Live Preview Component

### Location
- **Component**: `frontend/src/components/validation/LivePreview.tsx`
- **Integration**: Tab 3 in ValidationRuleEditor

### Features

#### Sample Data Input
- **JSON Format**: Paste raw JSON array
  ```json
  [
    { "customer_id": "C001", "email": "john@example.com", "age": 28 },
    { "customer_id": "C002", "email": "jane@example.com", "age": 17 }
  ]
  ```

- **CSV Format**: Paste CSV data (auto-parsed)
  ```
  customer_id,email,age
  C001,john@example.com,28
  C002,jane@example.com,17
  ```

#### Test Results Display
- **Summary Tab**
  - Total records tested
  - Pass count and percentage
  - Fail count and percentage
  - Warning count and percentage

- **Results Tab**
  - Table showing each record's validation status
  - Detailed error/warning messages
  - Timestamp of test run
  - Easy scanning with status icons

#### Mock Rule Evaluation
The component includes intelligent mock evaluation logic:

```typescript
// NOT NULL validation
field IS NOT NULL → checks if field exists and not empty

// RANGE validation  
10 < value < 100 → checks numeric boundaries

// PATTERN validation
value MATCHES /regex/ → validates string patterns

// LOOKUP validation
value IN (ref_table) → checks if value exists in reference set
```

### Usage Example

```tsx
<LivePreview
  rule={{
    target_entity: 'Customer',
    field_name: 'email',
    rule_condition: 'field MATCHES /^[^@]+@[^@]+\\.[^@]+$/',
    severity: 'error'
  }}
  onTestResults={(results) => {
    console.log(`Tested ${results.length} records`);
    const failures = results.filter(r => r.status === 'fail');
    console.log(`${failures.length} records failed`);
  }}
/>
```

### Benefits
✅ **Immediate Feedback** - See what will break before deploying  
✅ **Confidence Building** - Users know the rule works on real-like data  
✅ **Edge Case Testing** - Test with unusual values upfront  
✅ **Time Saving** - Catch issues before production impact  

---

## 📊 Impact Analysis Component

### Location
- **Component**: `frontend/src/components/validation/ImpactAnalysis.tsx`
- **Integration**: Tab 4 in ValidationRuleEditor

### Features

#### Risk Assessment
Automatically calculates risk level based on impact percentage:

```
< 1%     → GREEN (Low Risk)     - Safe to deploy immediately
1-5%     → YELLOW (Medium Risk) - Review recommendations
5-10%    → ORANGE (High Risk)   - Consider staged deployment
> 10%    → RED (Critical Risk)  - Requires approval
```

#### Impact Metrics
- **Total Records**: How many records the rule targets
- **Affected Records**: How many fail the validation
- **Percentage**: (Affected / Total) * 100%
- **By Severity**:
  - Error records (🔴)
  - Warning records (⚠️)
  - Info records (ℹ️)

#### Department Breakdown
Shows which departments/areas are affected:

```
Customer Relations: 12 records (45%)
Operations: 8 records (30%)
Finance: 6 records (22%)
IT: 1 record (3%)
```

#### Auto-Generated Recommendations

**Low Risk (< 1%)**
```
"Great! This rule affects less than 1% of records.
You can deploy immediately."
```

**Medium Risk (1-5%)**
```
"Consider testing with affected users first.
The rule impacts 3% of records."
```

**High Risk (5-10%)**
```
"Consider a staged deployment:
1. Deploy to 10% of records first
2. Monitor for 24-48 hours
3. Roll out to remaining records"
```

**Critical Risk (> 10%)**
```
"This rule requires executive approval before deployment.
It affects 15% of your records across multiple departments.
Consider phasing it in over time."
```

#### Sample Affected Records
Shows 3-5 example records that will be affected, helping users understand the impact tangibly.

### Usage Example

```tsx
<ImpactAnalysis
  rule={{
    target_entity: 'Customer',
    field_name: 'email',
    rule_condition: 'field MATCHES /^[^@]+@[^@]+\\.[^@]+$/',
    severity: 'error',
    is_enabled: true
  }}
  tenantId={tenantId}
  datasourceId={datasourceId}
/>
```

### Benefits
✅ **Risk Awareness** - Understand scope before deployment  
✅ **Smart Recommendations** - Get guidance based on actual impact  
✅ **Department Visibility** - See cross-functional impact  
✅ **Executive Ready** - Data to justify deployment decisions  
✅ **Mistake Prevention** - Catch high-impact rules early  

---

## 🔧 Integration: ValidationRuleEditor

### Location
`frontend/src/components/validation/ValidationRuleEditor.tsx`

### Enhanced Dialog Structure

```tsx
<Dialog open={openDialog} maxWidth="md" fullWidth>
  <DialogTitle>Create New Rule</DialogTitle>
  
  {/* Tab Navigation */}
  <Tabs value={dialogTab}>
    <Tab label="📋 Templates" />      {/* Tab 0 */}
    <Tab label="⚙️ Configure" />     {/* Tab 1 */}
    <Tab label="▶️ Test" />          {/* Tab 2 */}
    <Tab label="📊 Impact" />        {/* Tab 3 */}
  </Tabs>
  
  <DialogContent>
    {dialogTab === 0 && <RuleTemplatesSelector ... />}
    {dialogTab === 1 && <RuleConfigForm ... />}
    {dialogTab === 2 && <LivePreview ... />}
    {dialogTab === 3 && <ImpactAnalysis ... />}
  </DialogContent>
  
  <DialogActions>
    <Button onClick={previousTab}>Back</Button>
    <Button onClick={nextTab}>Next</Button>
    <Button onClick={saveRule}>Create</Button>
  </DialogActions>
</Dialog>
```

### Key Implementation Details

#### Tab State Management
```typescript
const [dialogTab, setDialogTab] = useState(0);
const [selectedTemplate, setSelectedTemplate] = useState<RuleTemplate | null>(null);
```

#### Template Selection Handler
```typescript
const handleTemplateSelected = (template, rule) => {
  // Populate form with template data
  setFormData({ ... });
  // Move to configuration tab
  setDialogTab(1);
};
```

#### Form Data Flow
1. User selects template → Auto-populated in form
2. User customizes form → Data stored in formData state
3. User tests in LivePreview → Passes/fails against sample data
4. User views ImpactAnalysis → Sees scope and recommendations
5. User clicks Create → Rule saved with all customizations

---

## 📱 User Experience Flow

### Creating a Simple Rule (Email Validation)

1. **Start**
   - Click "Add Rule" button

2. **Templates (Tab 0)**
   - Search "email" in templates
   - See "Pattern/Format Match" template
   - Click it to preview
   - Click "Use This Template"

3. **Configure (Tab 1)**
   - Name: "Validate Email Format"
   - Entity: "Customer"
   - Field: "email"
   - Condition auto-filled from template
   - Click "Next"

4. **Test (Tab 2)**
   - Paste sample customer data (10 records)
   - Click "Test Rule"
   - See results: 8 pass, 2 fail (invalid emails)
   - Review failed records
   - Feel confident about the rule
   - Click "Next"

5. **Impact (Tab 3)**
   - System analyzes all customer records
   - Shows 245 affected records out of 5000 (4.9%)
   - Risk level: Medium
   - Recommendations: Test with affected users first
   - Department breakdown shows impact spread
   - Click "Create"

6. **Rule Deployed**
   - Rule created and ready to enforce

### Editing an Existing Rule

- Click "Edit" button on rule
- Opens dialog directly in Configure tab (skips templates)
- Can modify any field
- Click "Update" to save changes

---

## 🧪 Testing the Features

### Test Scenario 1: Try a Template

1. Navigate to Validation Rules page
2. Click "Add Rule"
3. In Templates tab, click "Not Null Check"
4. See preview dialog with template details
5. Click "Use This Template"
6. Verify form is populated with template data

### Test Scenario 2: Live Preview with Data

1. In Configure tab, set up a rule
2. Move to Test tab
3. Paste this sample data:
   ```json
   [
     {"customer_id": "C001", "age": 25},
     {"customer_id": "C002", "age": null},
     {"customer_id": "C003", "age": 17}
   ]
   ```
4. Click "Test Rule"
5. Verify results show one null failure

### Test Scenario 3: Impact Analysis

1. In Impact tab, observe:
   - Risk level indicator
   - Affected records percentage
   - Department breakdown
   - Generated recommendations
2. Verify risk level matches the impact percentage

---

## 🔌 Backend Integration

### Current State
All three components use **mock data** for immediate functionality.

### Backend Endpoints Needed

#### LivePreview Testing
```
POST /api/validations/test-rule
{
  "rule_id": "string (optional)",
  "rule_condition": "string",
  "sample_data": { records: [...] },
  "tenant_id": "uuid",
  "datasource_id": "uuid"
}

Response: {
  "results": [
    { "row_id": "1", "status": "pass", "message": "" },
    { "row_id": "2", "status": "fail", "message": "..." }
  ],
  "summary": {
    "total": 10,
    "passed": 9,
    "failed": 1,
    "warnings": 0
  }
}
```

#### ImpactAnalysis Evaluation
```
POST /api/validations/analyze-impact
{
  "rule_id": "string (optional)",
  "rule_condition": "string",
  "target_entity": "string",
  "tenant_id": "uuid",
  "datasource_id": "uuid"
}

Response: {
  "total_records": 5000,
  "affected_records": 245,
  "percentage": 4.9,
  "severity_breakdown": {
    "error": 150,
    "warning": 95,
    "info": 0
  },
  "department_breakdown": [
    { "name": "Customer Relations", "count": 112 },
    { "name": "Operations", "count": 89 }
  ],
  "sample_records": [ ... ],
  "risk_level": "medium"
}
```

### Mock Data Hook Locations
- **LivePreview**: Mock evaluation in `handleTestRule()` function (line ~120)
- **ImpactAnalysis**: Mock analysis in `useMemo()` hook (line ~80)

Search for `// TODO: Replace with actual API call` comments to find mock implementation points.

---

## 📚 API Reference

### RuleTemplatesSelector Props
```typescript
interface RuleTemplatesSelectorProps {
  onTemplateSelected: (template: RuleTemplate, rule: Partial<ValidationRule>) => void;
  targetEntity?: string; // Optional: pre-filter for entity
}
```

### LivePreview Props
```typescript
interface LivePreviewProps {
  rule: {
    target_entity: string;
    field_name: string;
    rule_condition: string;
    severity: 'error' | 'warning' | 'info';
  };
  onTestResults?: (results: TestResult[]) => void;
}

interface TestResult {
  row_id: string | number;
  status: 'pass' | 'fail' | 'warning';
  message: string;
  timestamp: string;
}
```

### ImpactAnalysis Props
```typescript
interface ImpactAnalysisProps {
  rule: ValidationRule;
  tenantId: string;
  datasourceId: string;
}

interface ValidationRule {
  id?: string;
  name: string;
  description: string;
  target_entity: string;
  field_name: string;
  rule_type: 'null-check' | 'range' | 'pattern' | 'lookup' | 'comparison' | 'custom';
  rule_condition: string;
  severity: 'error' | 'warning' | 'info';
  is_enabled: boolean;
}
```

---

## 🎯 Best Practices

### For Rule Creators
1. **Always use a template** - They're built from real-world patterns
2. **Test with realistic data** - Use samples close to actual production data
3. **Review the impact analysis** - Don't ignore risk warnings
4. **Start conservative** - You can always relax rules later
5. **Document your rules** - Use clear names and descriptions

### For Administrators
1. **Set severity appropriately** - Errors block, warnings notify
2. **Review high-impact rules** - Anything > 5% needs approval
3. **Monitor newly deployed rules** - Check validation logs first 24 hours
4. **Archive old templates** - Keep template library clean and current
5. **Collect feedback** - Users will find edge cases

### For Developers
1. **Connect to real APIs** - Replace mock implementations
2. **Add caching** - Live preview runs frequently
3. **Optimize queries** - ImpactAnalysis must scan all records
4. **Add audit logging** - Track rule creation and deployment
5. **Extend templates** - Add industry-specific templates over time

---

## 🚀 Future Enhancements

- [ ] Custom template creation by users
- [ ] Template versioning and deprecation
- [ ] A/B testing rules before full deployment
- [ ] Scheduled rule deployments
- [ ] Rule dependency management
- [ ] Performance benchmarking for rules
- [ ] Machine learning suggestions for new rules
- [ ] Integration with data profiling tools

---

## 📞 Support

For questions or issues with the enhanced validation rules features, check:

1. **Component Documentation**: Comments in source files
2. **Mock Data Examples**: In `LivePreview.tsx` and `ImpactAnalysis.tsx`
3. **Template Definitions**: In `ruleTemplates.ts`
4. **This Guide**: Sections above

---

*Last Updated: 2024 - Validation Rules Enhanced Features v1.0*
