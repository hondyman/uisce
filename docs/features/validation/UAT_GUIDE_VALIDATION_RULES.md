# User Acceptance Testing (UAT) Guide

**Project**: Advanced Validation Rules System  
**Version**: 2.0 (Complete)  
**Date**: October 20, 2025  
**Target Users**: Business Analysts, Data Stewards, Rule Administrators  

---

## 📋 Quick Start: Access the System

### 1. Prerequisites
- ✅ Backend running on `http://localhost:8080`
- ✅ Frontend running on `http://localhost:5173` or `http://localhost:5174`
- ✅ Database configured and seeded
- ✅ Tenant and datasource selected in Fabric Builder

### 2. Launch the Validation Rules Editor
1. Navigate to Fabric Builder UI
2. Ensure you have selected:
   - ✅ Tenant (from tenant picker)
   - ✅ Product (from product picker)
   - ✅ Datasource (from datasource picker)
3. Go to Validation Rules section
4. Click "Add Rule" button

### 3. Verify Initial State
- [ ] Dialog opens with "Create New Rule" title
- [ ] 4 tabs visible: "📋 Templates", "⚙️ Configure", "▶️ Test", "📊 Impact"
- [ ] Only Tab 0 (Templates) is enabled initially
- [ ] Tabs 1-3 are disabled (grayed out)

---

## 🧪 Test Scenarios

### SCENARIO 1: Create Rule from Template

**Goal**: Test template selection and form auto-population

**Steps**:
1. [ ] Click "Add Rule" to open dialog
2. [ ] In **Tab 0 (Templates)**:
   - [ ] Scroll through 8 template categories
   - [ ] Select "Email Validation" template
   - [ ] Verify form shows: "Uses template: Email Validation - Validates email format and pattern"
3. [ ] Verify form is pre-populated:
   - [ ] Rule Name: "Email Validation"
   - [ ] Business Process: "Customer"
   - [ ] Field: "email"
   - [ ] Condition shows email regex pattern
4. [ ] Click "Next" to go to **Tab 1 (Configure)**
   - [ ] Verify all fields are populated with template values
5. [ ] Click "Next" to go to **Tab 2 (Test)**
   - [ ] See "Generate Sample Data" section
   - [ ] Enter `20` for record count
   - [ ] Check "Include Edge Cases" checkbox
   - [ ] Click Generate button
   - [ ] Verify data appears in preview table
   - [ ] Test data includes nulls and special characters
6. [ ] Click "Next" to go to **Tab 3 (Impact)**
   - [ ] See risk assessment
   - [ ] Verify estimated affected records display
   - [ ] Check department breakdown chart
7. [ ] Click "Create" to save rule
   - [ ] Dialog closes
   - [ ] Rule appears in main rules table
   - [ ] Success message visible (if applicable)

**Expected Results**:
- ✅ All 8 templates load without errors
- ✅ Form pre-fills correctly from template
- ✅ Test data generates with realistic values
- ✅ Impact analysis shows estimated affected records
- ✅ Rule saves successfully

**Acceptance Criteria**:
- ✅ User can complete workflow from template to creation in <2 minutes
- ✅ No errors in browser console (F12)
- ✅ Sample data contains proper edge cases (nulls, empty strings)
- ✅ Impact shows reasonable estimates for test data

---

### SCENARIO 2: Clone Existing Rule

**Goal**: Test rule cloning with conflict detection

**Prerequisites**:
- At least 2 validation rules exist in the system

**Steps**:
1. [ ] Click "Add Rule" to open dialog
2. [ ] In **Tab 0 (Templates)**, scroll down to "OR Clone Existing Rule" section
3. [ ] Verify existing rules list displays:
   - [ ] Rule names visible
   - [ ] Severity badges (red/orange/blue) visible
4. [ ] Click a rule to clone (e.g., "Email Validation")
5. [ ] Verify form populates with:
   - [ ] Rule Name: "{original} (Copy)"
   - [ ] All other fields match original
6. [ ] **Conflict Detection**:
   - [ ] System shows warning: "90% match with original rule"
   - [ ] Shows "Consider making this more specific"
7. [ ] In **Tab 1 (Configure)**:
   - [ ] Modify rule name to make unique: "Email Validation - Strict"
   - [ ] Click "Browse" button next to Field input
   - [ ] Verify field selector opens

**Field Selector Test**:
8. [ ] In Advanced Field Selector dialog:
   - [ ] See entity cards (Employee, Department, Company, Customer, Country)
   - [ ] Click on "Customer" card
   - [ ] Fields list shows: id, email, phone, status, created_at
   - [ ] Click "email" field
   - [ ] Field path shows: "email" (copied to parent form)
   - [ ] Dialog closes automatically
9. [ ] Back in Configure tab:
   - [ ] Field input now shows "email"

**Continue Workflow**:
10. [ ] Proceed through Tab 2 (Test) and Tab 3 (Impact)
11. [ ] Click "Create" to save
12. [ ] Verify new rule appears in list with different name

**Expected Results**:
- ✅ Cloning functionality works
- ✅ Conflict detection identifies similarity
- ✅ Field selector shows entities and relationships
- ✅ Field paths work correctly
- ✅ New rule saved successfully

**Acceptance Criteria**:
- ✅ Clone operation completes in <30 seconds
- ✅ Conflict detection is accurate (>70% similarity detected)
- ✅ Field selector is intuitive and responsive
- ✅ No errors with dot notation paths

---

### SCENARIO 3: Field Selection with Dot Notation

**Goal**: Test advanced field selector with entity relationships

**Steps**:
1. [ ] Click "Add Rule" → Tab 0 → Skip templates (no selection)
2. [ ] Advance to **Tab 1 (Configure)**
3. [ ] Enter entity: "Employee"
4. [ ] Click "Browse" button
5. [ ] In **Advanced Field Selector**:

**Navigation Test**:
   - [ ] Employee entity card selected
   - [ ] Fields visible: id, email, first_name, last_name, department_id, salary, hire_date, is_active
   - [ ] Click "department_id" field
   - [ ] See "RelatedEntity: Department"
   - [ ] See relationship path at top: "Employee > department"
   - [ ] Verify "Navigate →" option appears
   - [ ] Click navigate arrow
   
**Relationship Traversal**:
   - [ ] Now showing Department entity
   - [ ] Fields visible: id, name, company_id, budget
   - [ ] Breadcrumb shows: "Employee > Department"
   - [ ] Can click "company_id" to navigate to Company
   - [ ] Click navigate arrow
   - [ ] Now showing Company entity
   - [ ] Fields visible: id, name, country_id, founded_year, revenue
   - [ ] Breadcrumb shows: "Employee > Department > Company"
   - [ ] Can click "country_id" to navigate to Country
   - [ ] Click navigate arrow
   - [ ] Now showing Country entity
   - [ ] Fields visible: id, name, region
   - [ ] Breadcrumb shows: "Employee > Department > Company > Country"
   - [ ] Click "name" field
   - [ ] Dialog closes with field path: "employee.department.company.country.name"

**Result**:
6. [ ] Back in Configure tab, field shows: "employee.department.company.country.name"

**Expected Results**:
- ✅ Entity relationship navigation smooth
- ✅ Dot notation paths generated correctly
- ✅ Breadcrumb trail accurate
- ✅ Can navigate up and down relationship chains

**Acceptance Criteria**:
- ✅ Supports 4+ levels of relationship nesting
- ✅ Dot notation paths are readable and follow standard conventions
- ✅ No performance lag when navigating
- ✅ Field metadata displays correctly (type, nullable, format)

---

### SCENARIO 4: Sample Data Generation

**Goal**: Test realistic test data generation for rule preview

**Steps**:
1. [ ] Click "Add Rule" → Configure simple rule
2. [ ] Go to **Tab 2 (Test)**
3. [ ] In "Generate Sample Data":
   - [ ] Set record count: "50"
   - [ ] Check "Include Edge Cases"
   - [ ] Leave format as "JSON"
   - [ ] Click "Generate" button
4. [ ] Verify preview table shows:
   - [ ] First 5 records displayed
   - [ ] Realistic data values (names, emails, dates)
   - [ ] Some records marked with "NULL" or "(empty)"

**Edge Cases Verification**:
5. [ ] Scroll through preview, look for:
   - [ ] NULL values in nullable fields
   - [ ] Empty strings ""
   - [ ] Special characters (!, @, #, etc.)
   - [ ] Boundary values (year 1900, year 2099)

**Export Test**:
6. [ ] Click "Download JSON"
   - [ ] File downloads successfully
   - [ ] Verify file contains valid JSON (open in text editor)
7. [ ] Generate new data set, click "Copy to Clipboard"
   - [ ] Can paste into text editor (verifies copy worked)
8. [ ] Change format to "CSV"
   - [ ] Click "Download CSV"
   - [ ] File downloads successfully
   - [ ] Can open in Excel or text editor
   - [ ] Verify proper CSV formatting (quotes, commas)

**Testing Against Rule**:
9. [ ] In **Live Preview** section below:
   - [ ] See rule test results
   - [ ] Some records should "PASS" (valid email)
   - [ ] Some records should "FAIL" (NULL, invalid format)
   - [ ] Edge cases clearly marked

**Expected Results**:
- ✅ Sample data generation completes in <2 seconds
- ✅ Data is realistic for field types
- ✅ Edge cases included when checked
- ✅ Export works in both JSON and CSV formats
- ✅ Preview shows test execution results

**Acceptance Criteria**:
- ✅ Supports 1-1000 records
- ✅ Edge cases are truly edge cases (not just random)
- ✅ Data generation performance acceptable (<3 seconds for 1000 records)
- ✅ CSV export handles special characters correctly

---

### SCENARIO 5: Impact Analysis

**Goal**: Test rule impact assessment before deployment

**Steps**:
1. [ ] Create or edit any rule, go to **Tab 3 (Impact)**
2. [ ] Verify display shows:
   - [ ] "Estimated Affected Records" with number
   - [ ] Risk level indicator (Low/Medium/High/Critical) with color
   - [ ] Progress bar showing risk percentage
3. [ ] Look for additional details:
   - [ ] Department breakdown (pie or bar chart)
   - [ ] Suggested actions/recommendations
   - [ ] Severity summary

**Risk Assessment**:
4. [ ] Try creating rules with different severities:
   - [ ] "error" rule: Should show "High" or "Critical" risk
   - [ ] "warning" rule: Should show "Medium" or "High" risk
   - [ ] "info" rule: Should show "Low" or "Medium" risk
5. [ ] Verify recommendations change based on:
   - [ ] Number of affected records
   - [ ] Rule severity
   - [ ] Related existing rules

**Expected Results**:
- ✅ Impact analysis displays for all rule types
- ✅ Estimated record counts are reasonable
- ✅ Risk colors are consistent (red=high, yellow=medium, green=low)
- ✅ Recommendations are helpful

**Acceptance Criteria**:
- ✅ Impact calculation completes in <1 second
- ✅ Risk assessment is accurate (can spot-check a few)
- ✅ Recommendations prevent user from creating harmful rules
- ✅ Department breakdown provides useful insights

---

### SCENARIO 6: Conflict Detection & Prevention

**Goal**: Test conflict detection prevents duplicate/conflicting rules

**Steps**:
1. [ ] Navigate to ValidationRuleEditor
2. [ ] Identify an existing rule (e.g., "Email Validation")
3. [ ] Click "Add Rule"
4. [ ] Try to create a nearly identical rule:
   - [ ] Same entity: "Customer"
   - [ ] Same field: "email"
   - [ ] Similar condition to existing rule
5. [ ] In Tab 0 (Templates):
   - [ ] Verify clone section shows the similar rule
   - [ ] Warning displays: "Rule similar to existing rule"
6. [ ] In Tab 1 (Configure):
   - [ ] Just before saving, system should show conflict warning
   - [ ] Suggests: "Consider modifying this rule to be more specific"
7. [ ] Modify condition to be different
8. [ ] Save rule
   - [ ] Rule saves successfully
   - [ ] No duplicates created

**Expected Results**:
- ✅ Conflict detection works for >70% similar rules
- ✅ Warnings don't prevent saving but alert user
- ✅ Users can choose to override if intentional
- ✅ No duplicate rules created unintentionally

**Acceptance Criteria**:
- ✅ Conflict detection accuracy >95%
- ✅ False negatives are rare
- ✅ False positives are acceptable (user can choose to save anyway)
- ✅ Performance impact on form is minimal

---

### SCENARIO 7: End-to-End Workflow

**Goal**: Complete workflow from creation to deployment

**Steps**:

**Phase 1: Selection**
1. [ ] Click "Add Rule"
2. [ ] In Tab 0: Select "Date Range Validation" template
3. [ ] Form auto-fills with template values

**Phase 2: Configuration**
4. [ ] In Tab 1: 
   - [ ] Modify name: "Valid Hire Date"
   - [ ] Entity: "Employee"
   - [ ] Field: Click Browse → select "hire_date"
   - [ ] Condition: Already set to date range check
   - [ ] Set priority: 75
   - [ ] Enable: Check the enabled toggle

**Phase 3: Testing**
5. [ ] In Tab 2:
   - [ ] Generate 100 sample records with edge cases
   - [ ] Preview shows: Recent hires, old hires, NULL dates
   - [ ] Live preview shows which pass/fail
   - [ ] Verify results make sense

**Phase 4: Impact Review**
6. [ ] In Tab 3:
   - [ ] Check estimated affected records (should be reasonable)
   - [ ] Review risk level
   - [ ] Read recommendations
   - [ ] Decide if safe to deploy

**Phase 5: Save**
7. [ ] Click "Create" button
8. [ ] Dialog closes
9. [ ] New rule appears in main table
10. [ ] Can edit or delete from main table

**Expected Results**:
- ✅ All workflow steps complete successfully
- ✅ Transitions between tabs smooth
- ✅ Data persists correctly
- ✅ Final rule appears in list

**Acceptance Criteria**:
- ✅ Entire workflow takes <5 minutes for experienced user
- ✅ No data is lost during process
- ✅ Rule can be edited after creation
- ✅ Rule can be deleted without issues

---

## 🐛 Bug Report Template

When you encounter issues, please use this template:

**Bug Title**: [Brief description]

**Severity**: 
- [ ] Critical (system breaks)
- [ ] High (feature unusable)
- [ ] Medium (feature degraded)
- [ ] Low (minor issue)

**Steps to Reproduce**:
1. 
2. 
3. 

**Expected Behavior**:


**Actual Behavior**:


**Screenshot/Video**:
[Attach if applicable]

**Browser & Environment**:
- Browser: 
- OS: 
- Resolution:

**Console Errors** (Press F12, check Console tab):
```
[Paste any errors here]
```

---

## ✅ Test Completion Checklist

### Quick Check (15 minutes)
- [ ] Can add new rule from template
- [ ] Can clone existing rule
- [ ] Can navigate through all 4 tabs
- [ ] Can generate sample data
- [ ] Can save rule

### Comprehensive Check (1 hour)
- [ ] Scenario 1: Template Creation ✅
- [ ] Scenario 2: Rule Cloning ✅
- [ ] Scenario 3: Dot Notation ✅
- [ ] Scenario 4: Sample Data ✅
- [ ] Scenario 5: Impact Analysis ✅
- [ ] Scenario 6: Conflict Detection ✅
- [ ] Scenario 7: End-to-End ✅

### Performance Check
- [ ] Template load: <500ms
- [ ] Field selector open: <300ms
- [ ] Sample data gen (100 records): <2s
- [ ] All 4 tabs load smoothly
- [ ] No lag when switching tabs

### Data Quality Check
- [ ] Generated sample data is realistic
- [ ] Edge cases properly included
- [ ] Dot notation paths correct
- [ ] Entity relationships accurate
- [ ] No duplicate rules created

### Browser Compatibility
- [ ] Chrome latest
- [ ] Firefox latest
- [ ] Safari latest
- [ ] Edge latest

---

## 📊 Success Metrics

**User Experience**:
- [ ] Rule creation time: <5 minutes (vs. 15 min before)
- [ ] Error rate: <5% (vs. 20% before with duplicates)
- [ ] User confidence: High (validated by impact analysis)

**System Performance**:
- [ ] Page load time: <2 seconds
- [ ] Tab switching: <300ms
- [ ] Data generation: <2s per 100 records
- [ ] API response time: <500ms

**Data Quality**:
- [ ] Conflict detection: >95% accuracy
- [ ] Sample data coverage: 100% of field types
- [ ] Impact analysis accuracy: >90%

---

## 🎓 User Training Topics

**For Business Analysts**:
1. Creating rules from templates
2. Understanding impact analysis
3. When to use different rule types
4. Reading generated test data

**For Data Stewards**:
1. Complete workflow start-to-finish
2. Reviewing rules for conflicts
3. Managing rule versions
4. Documentation best practices

**For IT/Admin**:
1. Backend API endpoints
2. Deploying to production
3. Monitoring rule execution
4. Troubleshooting common issues

---

## 🚀 Sign-Off

**UAT Coordinator**: _________________

**Date**: _________________

**Status**: 
- [ ] PASS - Ready for production
- [ ] PASS with Issues - Document issues and decide on fix timeline
- [ ] FAIL - Blockers found, needs additional development

**Comments**:
___________________________________________________
___________________________________________________

---

*Advanced Validation Rules System - UAT Complete*  
*User Acceptance Testing Guide v2.0*
