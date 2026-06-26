# Condition Builder Examples

## Example 1: Text Field Condition - Company Name Must Contain "Inc"

**Scenario:** Validate that all customer companies registered as incorporated entities have "Inc" in their name.

**Steps:**
1. Create validation rule with Target Entity: "Customer"
2. In Step 4 (Conditions):
   - Field: "Company Name" (text type)
   - Operator: "Contains" (text-specific operator)
   - Value: "Inc"

**Result:**
```json
{
  "field": "company_name",
  "fieldType": "text",
  "operator": "contains",
  "value": "Inc",
  "fieldLabel": "Company Name"
}
```

**When Applied:** Rule validates each customer's company_name field contains "Inc"

---

## Example 2: Number Field Condition - Revenue Over Threshold

**Scenario:** Apply premium pricing to customers with annual revenue over $1,000,000.

**Steps:**
1. Create validation rule with Target Entity: "Customer"
2. In Step 4 (Conditions):
   - Field: "Annual Revenue" (number type)
   - Operator: "Greater Than" (number-specific operator)
   - Value: "1000000"

**Note:** Value input automatically becomes number type when "Annual Revenue" selected.

**Result:**
```json
{
  "field": "annual_revenue",
  "fieldType": "number",
  "operator": "greater_than",
  "value": "1000000",
  "fieldLabel": "Annual Revenue"
}
```

---

## Example 3: Date Field Condition - Recent Registration

**Scenario:** Identify newly registered customers (registered in last 90 days).

**Steps:**
1. Create validation rule with Target Entity: "Customer"
2. In Step 4 (Conditions):
   - Field: "Registration Date" (date type)
   - Operator: "After" (date-specific operator, labeled as "After")
   - Value: [Use calendar picker to select] "2024-09-15"

**Note:** Value input automatically becomes date picker when "Registration Date" selected.

**Result:**
```json
{
  "field": "registration_date",
  "fieldType": "date",
  "operator": "greater_than",
  "value": "2024-09-15",
  "fieldLabel": "Registration Date"
}
```

---

## Example 4: Boolean Field Condition - Active Status

**Scenario:** Validate that all processing only applies to active customers.

**Steps:**
1. Create validation rule with Target Entity: "Customer"
2. In Step 4 (Conditions):
   - Field: "Is Active" (boolean type)
   - Operator: "Equals" (only 2 operators for boolean)
   - Value: "True" (dropdown select)

**Note:** Value input automatically becomes boolean dropdown when "Is Active" selected.

**Result:**
```json
{
  "field": "is_active",
  "fieldType": "boolean",
  "operator": "equals",
  "value": "true",
  "fieldLabel": "Is Active"
}
```

---

## Example 5: Subtype-Specific Condition - VIP Customer

**Scenario:** For VIP customers specifically, ensure they have premium support assigned.

**Steps:**
1. Create validation rule
2. Step 2 - Configuration:
   - Target Entity: "Customer"
   - **Sub-Entity Type: "VIP Customer"** (NEW: Select subtype)
3. Step 4 - Conditions:
   - Field dropdown now shows:
     - All Customer entity fields (id, company_name, etc.)
     - Plus any VIP Customer specific fields
   - Field: "Premium Support Level" (VIP-specific field)
   - Operator: "Not Equals"
   - Value: "None"

**Result:**
```json
{
  "field": "premium_support_level",
  "fieldType": "text",
  "operator": "not_equals",
  "value": "None",
  "fieldLabel": "Premium Support Level"
}
```

**Applied Only To:** VIP Customers (due to sub_entity_type selection)

---

## Example 6: Multiple Conditions (Comma-Separated)

**Scenario:** Ensure all large enterprise customers have complete registration data.

**Steps:**
1. Create validation rule with Target Entity: "Customer"
2. Step 4 - Conditions: Click "+ Add Condition" for each rule
   
**Condition 1:**
- Field: "Company Name"
- Operator: "Is Not Empty"
- Value: (no value needed for "Is Not Empty")

**Condition 2:**
- Field: "Contact Email"
- Operator: "Is Not Empty"
- Value: (no value needed)

**Condition 3:**
- Field: "Annual Revenue"
- Operator: "Greater Than"
- Value: "500000"

**Result:**
```json
{
  "conditions": [
    {
      "field": "company_name",
      "fieldType": "text",
      "operator": "is_not_empty",
      "value": "",
      "fieldLabel": "Company Name"
    },
    {
      "field": "contact_email",
      "fieldType": "text",
      "operator": "is_not_empty",
      "value": "",
      "fieldLabel": "Contact Email"
    },
    {
      "field": "annual_revenue",
      "fieldType": "number",
      "operator": "greater_than",
      "value": "500000",
      "fieldLabel": "Annual Revenue"
    }
  ]
}
```

**Note:** Currently conditions are AND-ed together (all must be true). Future enhancement will add OR logic.

---

## Example 7: Empty/Not Empty Conditions

**Scenario:** Validate no duplicate emails in customer database.

**Steps:**
1. Create validation rule with Target Entity: "Customer"
2. Step 4 - Conditions:
   - Field: "Email"
   - Operator: "Is Not Empty"
   - Value: (empty - not needed for this operator)

**Result:**
```json
{
  "field": "email",
  "fieldType": "text",
  "operator": "is_not_empty",
  "value": "",
  "fieldLabel": "Email"
}
```

---

## Operator Behavior Reference

### Text Field Operators
| Operator | Description | Example |
|----------|-------------|---------|
| Equals | Exact match | "ABC Corporation" |
| Not Equals | Not exact match | "XYZ Inc" |
| Contains | Contains substring | "Corp" (matches "ABC Corporation") |
| Starts With | String starts with value | "ABC" (matches "ABC Corporation") |
| Ends With | String ends with value | "Inc" (matches "ABC Inc") |
| Is Empty | Field is null/empty | (no value) |
| Is Not Empty | Field has value | (no value) |

### Number Field Operators
| Operator | Description | Example |
|----------|-------------|---------|
| Equals | Exact value | 1000000 |
| Not Equals | Not exact value | 500000 |
| Greater Than | > | 1000000 |
| Less Than | < | 500000 |
| Is Empty | Field is null | (no value) |
| Is Not Empty | Field has value | (no value) |

### Date Field Operators
| Operator | Description | Example |
|----------|-------------|---------|
| Equals | Exact date match | 2024-01-15 |
| Not Equals | Not exact date | 2024-01-01 |
| After | After date (>) | 2024-01-15 |
| Before | Before date (<) | 2024-12-31 |
| Is Empty | Field is null | (no value) |
| Is Not Empty | Field has value | (no value) |

### Boolean Field Operators
| Operator | Description | Value |
|----------|-------------|-------|
| Equals | Exact boolean match | True or False |
| Not Equals | Opposite boolean | True or False |

---

## UI Behavior Notes

### Dynamic Operator Filtering
When you select a field:
- **Text field selected** → Operator dropdown changes to show text-specific operators
- **Number field selected** → Operator dropdown changes to show number-specific operators
- **Date field selected** → Operator dropdown changes to show date-specific operators
- **Boolean field selected** → Operator dropdown shows only Equals/Not Equals

**Smart Validation:** If you had "Greater Than" selected for a text field and change to another text field, the operator automatically resets to the first valid operator for text fields (Equals).

### Dynamic Value Input
When you select a field:
- **Text field selected** → Value shows text input
- **Number field selected** → Value shows number spinner input
- **Date field selected** → Value shows calendar date picker
- **Boolean field selected** → Value shows True/False dropdown

**Examples:**
```
Text field: [text input] → Type "ABC"
Number field: [number input with ↑↓] → Type 1000000
Date field: [calendar icon] → Pick date
Boolean field: [dropdown] → Select True or False
```

---

## Error Prevention

The condition builder prevents common errors:

### Prevented Error #1: Invalid Operator for Field Type
❌ **Before:** Could select "Greater Than" for text field
✅ **After:** Operator dropdown only shows valid operators for selected field type

### Prevented Error #2: Wrong Value Type
❌ **Before:** Could type "abc" in number field
✅ **After:** Number field only accepts numeric input

### Prevented Error #3: Date in Wrong Format
❌ **Before:** Could type "1/15/2024" or "15-01-2024" inconsistently
✅ **After:** Calendar picker ensures YYYY-MM-DD format

### Prevented Error #4: Invalid Boolean Value
❌ **Before:** Could type "yes", "1", "true" inconsistently
✅ **After:** Boolean dropdown ensures "true" or "false" values

---

## Integration with Backend

All conditions are sent to backend in this format:
```json
{
  "rule_name": "...",
  "rule_type": "...",
  "target_entity": "...",
  "severity": "...",
  "condition_json": {
    "conditions": [
      {
        "field": "company_name",
        "operator": "contains",
        "value": "Inc"
      },
      {
        "field": "annual_revenue",
        "operator": "greater_than",
        "value": "1000000"
      }
    ]
  }
}
```

**Note:** `fieldType` and `fieldLabel` are UI metadata and not included in backend request.

Backend processes the core fields: `field`, `operator`, `value`.

---

## Best Practices

1. **Be Specific:** Use specific conditions rather than "Is Not Empty" when possible
2. **Test with Data:** Manually verify conditions match intended business rules
3. **Document Logic:** Add clear rule names and descriptions
4. **Use Business Names:** Refer to business-friendly field names when discussing
5. **Review Regularly:** Check that conditions still match business requirements
6. **Version Control:** Keep track of rule changes for audit trail

---

## Troubleshooting Examples

### Issue: Field dropdown shows no options
**Solution:** Ensure entity is selected in Step 2

### Issue: Operator reverted after field change
**Solution:** This is intentional - invalid operators are auto-reset to first valid operator

### Issue: Value shows as text input but need date
**Solution:** Ensure correct field is selected (field type must be "date")

### Issue: Can't enter decimal in number field
**Solution:** Number field accepts both integers and decimals

### Issue: Date picker not opening
**Solution:** Try clicking directly on the date input, or using keyboard (Tab to focus, spacebar to open)
