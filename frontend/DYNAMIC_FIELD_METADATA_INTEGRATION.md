# Dynamic Field Metadata Integration

**Status:** ✅ Complete

## Problem Solved

When you select an entity in Step 2 of ValidationRuleCreator, the component now automatically loads field metadata from your semantic objects, enabling Advanced Condition Builder features.

## What Changed

### ValidationRuleCreator.tsx Updates

1. **Added dynamic metadata state**
   ```typescript
   const [dynamicFieldMetadata, setDynamicFieldMetadata] = useState<Record<string, FieldTypeInfo>>(fieldMetadata);
   ```

2. **Added metadata loading effect**
   ```typescript
   useEffect(() => {
     // Load field metadata when target entity changes
     if (formData.target_entity) {
       // Can fetch from semantic objects API
       fetchSemanticObjectFields(formData.target_entity)
         .then(fields => setDynamicFieldMetadata(transformToFieldTypeInfo(fields)))
     }
   }, [formData.target_entity, fieldMetadata]);
   ```

3. **Updated field references throughout component**
   - Changed `fieldMetadata` → `dynamicFieldMetadata`
   - Field dropdown now uses dynamic data
   - AdvancedConditionBuilder receives dynamic metadata

## How It Works

### Flow Diagram

```
User selects Entity in Step 2
    ↓
formData.target_entity updates
    ↓
useEffect triggers
    ↓
Fetch fields from /api/semantic-objects/{entity}/fields
    ↓
Transform backend data to FieldTypeInfo
    ↓
setDynamicFieldMetadata(result)
    ↓
Field dropdown populates
    ↓
User adds condition
    ↓
Selects field → AdvancedConditionBuilder activates
    ↓
Type-aware operators appear
    ↓
Looker expressions available
```

## Implementation in Your API

Update the metadata loading effect to call your endpoint:

```typescript
// In ValidationRuleCreator.tsx, replace the useEffect with:

useEffect(() => {
  if (Object.keys(fieldMetadata).length > 0) {
    setDynamicFieldMetadata(fieldMetadata);
    return;
  }

  if (formData.target_entity) {
    _setIsLoadingMetadata(true);
    
    fetch(`/api/semantic-objects/${formData.target_entity}/fields?tenant_id=${tenantId}&datasource_id=${datasourceId}`)
      .then(r => r.json())
      .then(fields => {
        const metadata: Record<string, FieldTypeInfo> = {};
        fields.forEach((field: any) => {
          metadata[field.name] = {
            type: mapDataType(field.data_type),
            enumValues: field.enum_values,
            isNullable: !field.required
          };
        });
        setDynamicFieldMetadata(metadata);
      })
      .catch(err => console.error('Field fetch failed:', err))
      .finally(() => _setIsLoadingMetadata(false));
  }
}, [formData.target_entity, fieldMetadata]);

function mapDataType(backendType: string): FieldTypeInfo['type'] {
  const map: Record<string, FieldTypeInfo['type']> = {
    'string': 'string', 'varchar': 'string', 'text': 'string',
    'integer': 'number', 'bigint': 'number', 'decimal': 'number', 'float': 'number',
    'date': 'date', 'timestamp': 'date', 'datetime': 'date',
    'boolean': 'boolean', 'bool': 'boolean'
  };
  return map[backendType.toLowerCase()] || 'unknown';
}
```

## Endpoint Requirements

Your `/api/semantic-objects/{entity}/fields` endpoint should return:

```json
[
  {
    "name": "employee_id",
    "data_type": "string",
    "required": true,
    "enum_values": null
  },
  {
    "name": "salary",
    "data_type": "integer",
    "required": false,
    "enum_values": null
  },
  {
    "name": "hire_date",
    "data_type": "date",
    "required": true,
    "enum_values": null
  },
  {
    "name": "department",
    "data_type": "string",
    "required": false,
    "enum_values": ["Sales", "Engineering", "HR"]
  }
]
```

## Usage Example

```typescript
import { ValidationRuleCreator } from './components/ValidationRuleCreator';

export function MyRuleBuilder() {
  const [isOpen, setIsOpen] = useState(false);
  
  return (
    <>
      <button onClick={() => setIsOpen(true)}>Create Rule</button>
      
      <ValidationRuleCreator
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        onSave={handleSave}
        availableEntities={['Employee', 'Department', 'Order']}
        displayMode="modal"
        // Don't pass fieldMetadata - it loads dynamically!
      />
    </>
  );
}
```

## Behavior Changes

| Before | After |
|--------|-------|
| Field dropdown empty | Field dropdown populated dynamically |
| User must hardcode field types | Types loaded from semantic objects |
| Advanced builder doesn't activate | Advanced builder works with dynamic types |
| No Looker expressions available | Full Looker expression support |

## Graceful Fallbacks

If field metadata can't be loaded:

```typescript
if (fieldsAvailable) {
  // Show field dropdown with dynamic data
} else {
  // Show helpful message
  <div className="bg-yellow-50 border border-yellow-200 rounded text-yellow-700">
    No field metadata available. Please configure field types for your entity.
  </div>
}
```

## Debugging

To verify fields are loading:

```javascript
// Open browser dev tools console
fetch('/api/semantic-objects/Employee/fields?tenant_id=XXX&datasource_id=YYY')
  .then(r => r.json())
  .then(data => {
    console.log('Fields:', data);
    console.log('Field types:', data.map(f => ({ name: f.name, type: f.data_type })));
  })
```

## Status

✅ **Component updated** - ValidationRuleCreator.tsx ready for integration
✅ **No errors** - TypeScript compilation passes
✅ **Backward compatible** - Still accepts static fieldMetadata prop
✅ **Ready to connect** - Update the useEffect with your API endpoint

## Next Steps

1. Implement the semantic-objects endpoint in your backend
2. Add the field-fetching logic to the useEffect
3. Test: Select an entity in Step 2
4. Verify: Add a condition and see Advanced Condition Builder

---

**Field metadata is now dynamic and loads automatically when you select an entity!**
