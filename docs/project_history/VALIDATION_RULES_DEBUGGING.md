# Debugging Validation Rules Visibility

## Issue
Validation rules created for an entity (e.g., "Employee") are not appearing in the entity's "Validations" tab.

## Root Cause Candidates

The filtering logic in `filterValidationRulesForEntity()` expects the rule's `target_entity` field to match one of these values in the match set:
- Entity key (e.g., "employee")
- Entity name (e.g., "Employee")
- Entity businessName (e.g., "Employee")
- Entity technicalName (e.g., "employee")

## Debugging Steps

### Step 1: Check Raw Rule Data
1. Open browser Developer Console (F12)
2. Navigate to the Validation Rules page and create a rule for "Employee"
3. Navigate to Entity Details page for "Employee"
4. Click on the "⚡ Validations" tab to trigger the fetch
5. In the console, look for logs like:
   ```
   EntityDetailsPage: All validation rules (detailed): [
     {id: "...", rule_name: "...", target_entity: "...", ...}
   ]
   ```

**Check:** Does the rule appear in the list? What is the `target_entity` value?

### Step 2: Check Match Set
Look for logs like:
```
buildMatchSet for entity employee : ["employee", "Employee", "employee", ...]
```

**Check:** Are all expected entity names in this list? Is it lowercase?

### Step 3: Check Matching Logic
Look for logs like:
```
Rule matching result: {
  ruleName: "...",
  targetEntities: [],
  targetEntity: "Employee",
  hasGlobal: false,
  hasEntitySpecific: false,  // <-- This is the problem if false
  isEntitySpecific: false,
  assignmentType: "direct"
}
```

**Check:** Is `isEntitySpecific` false? If so, the rule won't be displayed.

### Step 4: Identify the Issue

**If `target_entity` is present but `isEntitySpecific` is false:**
- The value in `target_entity` doesn't match any value in the match set
- Most likely: case sensitivity issue (e.g., "EMPLOYEE" vs "employee")
- Or: entity name format mismatch (e.g., "Employee" vs "employee_info")

**If `target_entity` is empty/null:**
- The backend is not storing the `target_entity` value correctly
- Check the backend API endpoint `/api/validation-rules` to see if it's returning the field

## Possible Fixes

### Fix 1: Normalize Entity Names in Backend
Ensure the backend stores `target_entity` in a consistent format (lowercase or exact match with entity schema)

### Fix 2: Case-Insensitive Matching (Already Implemented)
The matching already converts to lowercase, so this shouldn't be the issue unless the value is null/empty

### Fix 3: Add `target_entities` Array
The backend might need to populate the `target_entities` array instead of just `target_entity` (newer field format)

## Expected Log Output

When everything works correctly, you should see:
```
Rule matching result: {
  ruleName: "My Employee Rule",
  targetEntities: [],
  targetEntity: "Employee",
  hasGlobal: false,
  hasEntitySpecific: true,      // <-- This should be true
  isEntitySpecific: true,       // <-- This should be true
  assignmentType: "direct"
}

Filtered rules for entity employee : {
  total: 1,                     // <-- Rules found
  byType: { global: 0, direct: 1, mixed: 0 }
}
```

## Next Steps

1. **Collect the debug logs** from your browser console by following the steps above
2. **Share the logs** showing what `target_entity` value is actually stored
3. **Check the backend** to verify how it's storing and returning the `target_entity` field
4. **Compare with entity names** to identify any mismatch

## Backend API Endpoint to Inspect

```
GET /api/validation-rules?tenant_id=<TENANT_ID>&datasource_id=<DATASOURCE_ID>
Headers: 
  X-Tenant-ID: <TENANT_ID>
  X-Tenant-Datasource-ID: <DATASOURCE_ID>
```

This should return rules with all required fields populated.
