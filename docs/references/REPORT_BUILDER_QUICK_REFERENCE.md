# Report Builder - Quick Reference Guide

## 📋 API Quick Start

### Type Inference

```go
// Determine data type from value
dataType := InferDataType(someValue)
// Returns: "object", "array", "number", "string", "boolean", "null", "mixed"

// Determine entity type from value
entityType := InferEntityType(someValue)
// Returns: "relationship", "collection", "measure", "attribute"
```

### Default Mappings

```go
// Get filter type for data type
filterType := GetDefaultFilterType("number") // Returns "range"
filterType := GetDefaultFilterType("string") // Returns "contains"

// Get aggregation for entity type
agg := GetDefaultAggregation("measure")    // Returns "sum"
agg := GetDefaultAggregation("attribute")  // Returns "count"
```

### Validation Functions

```go
// Validate UUID format
if err := ValidateUUID(templateID); err != nil {
    return fmt.Errorf("invalid template ID: %w", err)
}

// Sanitize and validate string
name, err := ValidateAndSanitizeString(input, "Rule Name", MaxEntityNameLength)
if err != nil {
    return fmt.Errorf("invalid name: %w", err)
}

// Validate drag-drop state
if err := ValidateDragDropState(&dropState); err != nil {
    return fmt.Errorf("invalid drop: %w", err)
}

// Find section by ID
sectionIndex, err := FindSectionByID(template.Sections, targetID)
if err != nil {
    return fmt.Errorf("section not found: %w", err)
}
```

---

## 🔄 Common Patterns

### Pattern 1: Creating a Report Section with Validation

```go
func CreateReportSection(ctx context.Context, templateID string, section *ReportSection) error {
    // Validate template ID
    if err := ValidateUUID(templateID); err != nil {
        return fmt.Errorf("invalid template ID: %w", err)
    }
    
    // Get template
    template, err := rb.GetReportTemplate(ctx, templateID)
    if err != nil {
        return fmt.Errorf("failed to get template: %w", err)
    }
    
    // Validate section
    if section == nil || section.Title == "" {
        return fmt.Errorf("section title is required")
    }
    
    // Add section
    template.Sections = append(template.Sections, *section)
    
    // Save with error handling
    if err := rb.SaveReportTemplate(ctx, template); err != nil {
        return fmt.Errorf("failed to save template: %w", err)
    }
    
    return nil
}
```

### Pattern 2: Handling Drop Actions

```go
func HandleDrop(ctx context.Context, templateID string, dropState DragDropState) error {
    // Validate inputs
    if err := ValidateDragDropState(&dropState); err != nil {
        return fmt.Errorf("invalid drop state: %w", err)
    }
    if err := ValidateUUID(templateID); err != nil {
        return fmt.Errorf("invalid template ID: %w", err)
    }
    
    // Get template
    template, err := rb.GetReportTemplate(ctx, templateID)
    if err != nil {
        return fmt.Errorf("failed to get template: %w", err)
    }
    
    // Find section
    sectionIndex, err := FindSectionByID(template.Sections, dropState.TargetSectionID)
    if err != nil {
        return fmt.Errorf("section lookup failed: %w", err)
    }
    
    // Route to handler
    switch dropState.Action {
    case "add_to_table":
        handler := &AddToTableHandler{}
        if err := handler.Handle(&template.Sections[sectionIndex], dropState.SourceEntity, dropState.TargetSectionID); err != nil {
            return fmt.Errorf("failed to handle drop: %w", err)
        }
    case "create_filter":
        filter := ReportFilter{
            ID:              uuid.New(),
            FilterType:      GetDefaultFilterType(dropState.SourceEntity.DataType),
            EntityID:        dropState.SourceEntity.EntityID,
            EntityName:      dropState.SourceEntity.EntityName,
            ApplyToSections: []string{dropState.TargetSectionID},
            DroppedFrom:     "drag_drop",
            Operator:        "and",
        }
        template.Filters = append(template.Filters, filter)
    }
    
    // Save
    if err := rb.SaveReportTemplate(ctx, template); err != nil {
        return fmt.Errorf("failed to save after drop: %w", err)
    }
    
    return nil
}
```

### Pattern 3: Creating Entities from Semantic Views

```go
func ProcessSemanticView(ctx context.Context, tenantID string) error {
    // Validate tenant ID
    if err := ValidateUUID(tenantID); err != nil {
        return fmt.Errorf("invalid tenant ID: %w", err)
    }
    
    // Get semantic views
    views, err := rb.GetSemanticViewsForReporting(ctx, tenantID)
    if err != nil {
        return fmt.Errorf("failed to get views: %w", err)
    }
    
    // Process each view
    for _, view := range views {
        // Views already have validated entities
        for _, entity := range view.DraggableEntities {
            // Use validated entity
            dataType := entity.DataType    // Already inferred
            entityType := entity.Type      // Already inferred
            filterType := GetDefaultFilterType(dataType)
            _ = filterType
        }
    }
    
    return nil
}
```

---

## 🚨 Error Handling Examples

### Handling JSON Errors

```go
// OLD (Bad)
json.Unmarshal([]byte(data), &target)
// Silent failure!

// NEW (Good)
if err := json.Unmarshal([]byte(data), &target); err != nil {
    return fmt.Errorf("failed to parse data: %w", err)
}
```

### Validating Drop State

```go
dropState := &DragDropState{
    SourceEntity:    entity,
    TargetSectionID: sectionID,
    Action:          "invalid_action",
}

if err := ValidateDragDropState(dropState); err != nil {
    // err: "invalid action: invalid_action"
}
```

### Finding Sections

```go
sectionIndex, err := FindSectionByID(template.Sections, "00000000-0000-0000-0000-000000000000")
if err != nil {
    // err: "section not found: 00000000-0000-0000-0000-000000000000"
    // Caller can decide how to handle (create, skip, error)
}
```

---

## 📊 Type Mappings Reference

### Data Type → Filter Type

| Data Type | Filter Type |
|-----------|-------------|
| number | range |
| string | contains |
| boolean | equals |
| date | between |
| (default) | equals |

### Entity Type → Aggregation

| Entity Type | Aggregation |
|-------------|-------------|
| measure | sum |
| attribute | count |
| (default) | count |

### Value Type → Entity/Data Type

| Value Type | Entity Type | Data Type |
|-----------|------------|-----------|
| map[string]interface{} | relationship | object |
| []interface{} | collection | array |
| float64 | measure | number |
| string | attribute | string |
| bool | attribute | boolean |
| nil | attribute | null |
| (other) | attribute | mixed |

---

## 🔒 Validation Constants

```go
const (
    MaxEntityNameLength     = 255      // Maximum length for entity names
    MaxDescriptionLength    = 2000     // Maximum length for descriptions
    MaxFilterValueLength    = 1000     // Maximum length for filter values
    MaxRuleExpressionLength = 5000     // Maximum length for rule expressions
)
```

---

## ✅ Unit Test Examples

### Testing Type Inference

```go
func TestInferDataType(t *testing.T) {
    tests := []struct {
        name     string
        value    interface{}
        expected string
    }{
        {
            name:     "map becomes object",
            value:    map[string]interface{}{"key": "value"},
            expected: "object",
        },
        {
            name:     "slice becomes array",
            value:    []interface{}{1, 2, 3},
            expected: "array",
        },
        {
            name:     "float64 becomes number",
            value:    42.0,
            expected: "number",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := InferDataType(tt.value)
            if got != tt.expected {
                t.Errorf("InferDataType(%v) = %s, want %s", tt.value, got, tt.expected)
            }
        })
    }
}
```

### Testing Validation

```go
func TestValidateUUID_InvalidFormat(t *testing.T) {
    err := ValidateUUID("not-a-uuid")
    if err == nil {
        t.Error("ValidateUUID should reject invalid format")
    }
    if !strings.Contains(err.Error(), "invalid UUID format") {
        t.Errorf("Expected UUID format error, got: %v", err)
    }
}

func TestValidateAndSanitizeString_TooLong(t *testing.T) {
    input := strings.Repeat("a", 300)
    _, err := ValidateAndSanitizeString(input, "TestField", 255)
    if err == nil {
        t.Error("ValidateAndSanitizeString should reject oversized input")
    }
}
```

---

## 🎯 Common Issues & Solutions

### Issue 1: "Invalid UUID format" Error

**Problem:** Template ID is not a valid UUID
```go
templateID := "not-a-valid-uuid"
err := ValidateUUID(templateID) // Error
```

**Solution:** Ensure ID comes from uuid.New() or is properly formatted
```go
templateID := uuid.New().String() // Valid
err := ValidateUUID(templateID)   // No error
```

### Issue 2: "Section not found" Error

**Problem:** Section ID doesn't exist in template
```go
sectionIndex, err := FindSectionByID(template.Sections, unknownID)
// Error: "section not found: ..."
```

**Solution:** Verify section ID before dropping
```go
sectionIndex, err := FindSectionByID(template.Sections, knownID)
if err != nil {
    // Handle missing section: create new, use first, etc.
}
```

### Issue 3: "Invalid drop state" Error

**Problem:** Drop action is not in allowed set
```go
dropState.Action = "invalid_action"
err := ValidateDragDropState(&dropState)
// Error: "invalid action: invalid_action"
```

**Solution:** Use one of the 4 allowed actions
```go
validActions := []string{"add_to_table", "create_filter", "create_aggregation", "create_rule"}
// Use one of these
```

### Issue 4: JSON Unmarshaling Fails

**Problem:** Corrupted or mismatched JSON in database
```go
template, err := rb.GetReportTemplate(ctx, templateID)
// Error: "failed to unmarshal sections: json.SyntaxError..."
```

**Solution:** Check database for corrupt data, regenerate templates if needed
```go
// Or fix at save time:
if err := rb.SaveReportTemplate(ctx, template); err != nil {
    // err: "failed to marshal sections: ..."
}
```

---

## 📈 Performance Tips

### Tip 1: Validate Early
```go
// Good: Fail fast
if err := ValidateUUID(templateID); err != nil {
    return err
}

// Bad: Do work then validate
template, err := rb.GetReportTemplate(ctx, templateID) // Slower if ID invalid
```

### Tip 2: Cache Type Mappings
```go
// The mappings are already constants (cached at compile time)
// No need to rebuild them in loops

// Good
var filters []ReportFilter
for _, entity := range entities {
    filterType := GetDefaultFilterType(entity.DataType) // Instant lookup
    filters = append(filters, ReportFilter{FilterType: filterType})
}
```

### Tip 3: Reuse Validation Functions
```go
// Good: Use helper
if err := ValidateDragDropState(&state); err != nil {
    return err
}

// Bad: Repeat validation
if state.SourceEntity.EntityID == "" {
    return fmt.Errorf("source entity ID cannot be empty")
}
if state.TargetSectionID == "" {
    return fmt.Errorf("target section ID cannot be empty")
}
// ... repeat for all fields
```

---

## 🔗 Related Files

- **Main Implementation:** `/services/ai-trade-reconciliation/backend/internal/reports/builder.go`
- **Helper Utilities:** `/services/ai-trade-reconciliation/backend/internal/reports/builder_helpers.go`
- **Type Definitions:** `/services/ai-trade-reconciliation/backend/internal/reports/models.go`
- **Report Engine:** `/services/ai-trade-reconciliation/backend/internal/reports/engine.go`
- **Documentation:** `/REPORT_BUILDER_IMPROVEMENTS.md`

---

## 📞 Support

For questions about specific functions, refer to the godoc comments in the source files:
- `builder_helpers.go` - All public function documentation
- `builder.go` - Modified methods documentation

---

**Version:** 1.0  
**Last Updated:** October 30, 2025  
**Status:** Production Ready ✅

