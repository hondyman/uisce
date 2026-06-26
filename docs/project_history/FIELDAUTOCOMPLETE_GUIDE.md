# FieldAutocomplete Component Guide

## Overview

The `FieldAutocomplete` component is a feature-rich, context-aware autocomplete field selector designed for the Fabric Builder stack. It provides intelligent filtering, keyboard navigation, and recently-used field memory to enhance user experience.

## Features

✨ **Smart Context-Aware Search**
- Searches both field names and descriptions
- Helps users find fields even if they don't know the exact technical name
- Real-time filtering as users type

🎯 **Recently Used Memory**
- Remembers the last 5 fields selected for each entity
- Shows recently used fields first for faster access
- Persists across browser sessions using sessionStorage

⌨️ **Full Keyboard Navigation**
- **Arrow Up/Down**: Navigate through field options
- **Enter**: Select the highlighted field
- **Escape**: Close the dropdown
- **Mouse Move**: Synchronizes with keyboard highlight

🎨 **Rich Information Display**
- Field type with emoji indicators (🔑 for uuid, 📝 for text, etc.)
- Colored type badges (purple for uuid, blue for text, green for integers, etc.)
- Nullability indicator
- Related entity references
- Field descriptions

📱 **Accessible & Responsive**
- Material-UI integration
- Full keyboard support for accessibility
- Mobile-friendly dropdown positioning
- Error state handling

## Installation & Usage

### Basic Usage

```tsx
import FieldAutocomplete from '@/components/common/FieldAutocomplete';

export function MyComponent() {
  const [selectedField, setSelectedField] = useState('');

  return (
    <FieldAutocomplete
      value={selectedField}
      onChange={(value) => setSelectedField(value)}
      entityName="Employee"
      label="Select Field to Validate"
      placeholder="Search for a field..."
      required={true}
    />
  );
}
```

### Integration in Forms

```tsx
<Grid container spacing={2}>
  <Grid item xs={12} sm={6}>
    <FieldAutocomplete
      value={formData.field_name}
      onChange={(value) => setFormData({ ...formData, field_name: value })}
      entityName={formData.target_entity}
      label="Field to Validate"
      placeholder="Search for a field..."
      error={errors.field_name}
      required
      showRecentFields={true}
    />
  </Grid>
  <Grid item xs={12} sm={6}>
    <TextField
      label="Validation Rule"
      value={formData.rule}
      onChange={(e) => setFormData({ ...formData, rule: e.target.value })}
      fullWidth
    />
  </Grid>
</Grid>
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `value` | `string` | required | The currently selected field name |
| `onChange` | `(value: string) => void` | required | Callback when field is selected |
| `entityName` | `string` | required | Entity name to filter fields (e.g., "Employee", "Department") |
| `label` | `string` | undefined | Label displayed above the input |
| `placeholder` | `string` | "Search for a field..." | Input placeholder text |
| `error` | `string` | undefined | Error message to display (if undefined, no error state) |
| `required` | `boolean` | false | Shows red asterisk (*) if true |
| `showRecentFields` | `boolean` | true | Show recently used fields first |
| `disabled` | `boolean` | false | Disable the input |

## Customizing Entity Schemas

The component uses `ENTITY_SCHEMAS` to define available fields for each entity. Update or extend this in `FieldAutocomplete.tsx`:

```tsx
export const ENTITY_SCHEMAS: Record<string, Field[]> = {
  Employee: [
    {
      name: 'employee_id',
      type: 'uuid',
      description: 'Unique employee identifier',
      nullable: false,
    },
    {
      name: 'first_name',
      type: 'text',
      description: 'Employee first name',
      nullable: false,
    },
    // ... more fields
  ],
  Department: [
    // Department fields
  ],
  // Add more entities as needed
};
```

### Field Type Indicators

The component automatically maps field types to emoji indicators and color badges:

| Type | Icon | Badge Color |
|------|------|------------|
| uuid | 🔑 | Purple |
| text/varchar | 📝 | Blue |
| integer/int/bigint | #️⃣ | Green |
| decimal/numeric | 💰 | Amber |
| boolean/bool | ✓ | Red |
| timestamp/date/time | ⏰/📅/🕐 | Indigo |
| json/jsonb | {} | Slate |
| array | [] | Slate |

## Keyboard Navigation Behavior

### When Dropdown is Closed
- **Arrow Down**: Opens dropdown and highlights first item

### When Dropdown is Open
- **Arrow Down**: Moves highlight to next item (cycles to first when at end)
- **Arrow Up**: Moves highlight to previous item (cycles to last when at start)
- **Enter**: Selects the highlighted field and closes dropdown
- **Escape**: Closes dropdown without selecting
- **Mouse Move**: Updates highlight position to match mouse location

## Recently Used Fields

The component automatically tracks recently used fields:

```tsx
// Stored in sessionStorage as:
localStorage.setItem(
  `recent_fields_${entityName}`,
  JSON.stringify(fieldNames) // max 5 fields
);

// Example:
// recent_fields_Employee = ["employee_id", "email", "first_name"]
```

Recently used fields:
- Persist during the browser session
- Are cleared when the browser is closed
- Show at the top of the dropdown when the input is empty
- Are limited to the most recent 5 fields

## Styling & Customization

The component uses Material-UI `TextField` and `Paper` for styling. To customize colors or spacing:

```tsx
// Example: Custom styling
<Box sx={{ maxWidth: 400 }}>
  <FieldAutocomplete
    value={selectedField}
    onChange={setSelectedField}
    entityName="Employee"
    label="Custom Styled Field"
  />
</Box>
```

### Dropdown Styling

The dropdown appearance can be customized by modifying the `Paper` component in the component code:

```tsx
<Paper
  sx={{
    position: 'absolute',
    top: '100%',
    left: 0,
    right: 0,
    zIndex: 1300,
    mt: 1,
    maxHeight: 320,  // Adjust max height
    overflowY: 'auto',
    boxShadow: 1,
  }}
>
```

## Accessibility Features

✅ Keyboard navigation support for all interactions
✅ Error states with accessible error messages
✅ Clear field type information for non-technical users
✅ Descriptive field descriptions
✅ Recent fields support for power users
✅ Mouse and keyboard support

## Example: ValidationResultsPanel Integration

```tsx
import FieldAutocomplete from '@/components/common/FieldAutocomplete';

export function ValidationResultsPanel() {
  const [filterBP, setFilterBP] = useState('');
  
  return (
    <Grid container spacing={2}>
      <Grid item xs={12} sm={6}>
        <FieldAutocomplete
          value={filterBP}
          onChange={(value) => setFilterBP(value)}
          entityName="BusinessProcess"
          label="Filter by Business Process"
          placeholder="Search for a business process..."
        />
      </Grid>
      {/* Rest of form */}
    </Grid>
  );
}
```

## Common Use Cases

### 1. Validation Rule Creator
```tsx
<FieldAutocomplete
  value={validationRule.field}
  onChange={(field) => setValidationRule({ ...validationRule, field })}
  entityName={validationRule.entity}
  label="Field to Validate"
  error={errors.field}
  required
/>
```

### 2. Data Quality Check Setup
```tsx
<FieldAutocomplete
  value={qualityCheck.compareField}
  onChange={setCompareField}
  entityName="Transaction"
  label="Compare Field"
  showRecentFields={true}
/>
```

### 3. Business Process Filter
```tsx
<FieldAutocomplete
  value={filterBP}
  onChange={setFilterBP}
  entityName="BusinessProcess"
  label="Business Process"
  placeholder="Search processes..."
  disabled={!tenantSelected}
/>
```

## Troubleshooting

### Fields Not Appearing
- Verify the entity name matches exactly (case-sensitive)
- Check that `ENTITY_SCHEMAS` includes the entity definition
- Ensure fields are properly configured with name, type, and description

### Keyboard Navigation Not Working
- Click the input field first to focus it
- Arrow Down should always open the dropdown
- Ensure dropdown isn't hidden behind other elements (check z-index)

### Recently Used Fields Not Persisting
- Check that sessionStorage is not disabled
- Verify the browser allows sessionStorage
- Recently used fields are cleared when the session ends

### Search Not Finding Fields
- Try searching by field description instead of just name
- Check that the description text matches your search query
- Search is case-insensitive but requires at least partial match

## Performance Considerations

- Component filters fields client-side (fast)
- Supports up to 100+ fields per entity efficiently
- Uses React.useMemo for performance optimization
- Dropdown auto-scrolling uses efficient scrollIntoView API

## Future Enhancements

Potential improvements for future versions:
- [ ] Async field loading from API for large schemas
- [ ] Keyboard shortcuts customization
- [ ] Multi-select mode
- [ ] Custom rendering functions
- [ ] Field grouping/categories
- [ ] Search highlighting in results
- [ ] Keyboard hint tooltips
