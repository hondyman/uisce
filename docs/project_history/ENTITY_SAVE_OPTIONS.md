# Entity Save Options Comparison

## Option 1: Auto-Save Individual Changes (Recommended for UX)

### How it works:
```
User adds Entity → Immediately POST /api/entity/{entity_id}
User adds Field → Immediately POST /api/entity/{entity_id}/fields
User adds Subtype → Immediately POST /api/entity/{entity_id}/subtypes
```

### Pros:
- ✅ No "SAVE & APPLY" button needed (more modern UX)
- ✅ Smaller payloads
- ✅ Changes persist immediately
- ✅ Better for collaborative scenarios

### Cons:
- ❌ More API calls
- ❌ Need to handle network errors per action
- ❌ Harder to undo multiple changes at once

### Implementation example:
```typescript
// In handleFinish for entity creation
if (type === 'entity') {
  const key = values.name.toLowerCase().replace(/\s+/g, '_');
  const newEntity = { name: values.name, entity_fields: [], subtypes: {} };
  
  // Save to backend immediately
  try {
    await saveEntity(key, newEntity);
    setEntities({ ...entities, [key]: newEntity });
    message.success(`Entity "${values.name}" created and saved!`);
  } catch (error) {
    message.error(`Failed to create entity: ${error.message}`);
  }
}
```

---

## Option 2: Track Changes and Send Only Deltas (Keep SAVE & APPLY)

### How it works:
```
User adds Entity → Local state updated
User adds Field → Local state updated
[User clicks SAVE & APPLY]
→ POST /api/entity-schema with: { changed: [entity1, entity2], deleted: [...] }
```

### Pros:
- ✅ Batch updates (single API call)
- ✅ All-or-nothing saves
- ✅ Easy to implement with existing UI

### Cons:
- ❌ Still saves full entity objects (not minimal delta)
- ❌ Still sends unused data
- ❌ Need to keep SAVE & APPLY button

### Implementation example:
```typescript
// Track what changed
const [initialEntities, setInitialEntities] = useState(initialData);
const [entities, setEntities] = useState(initialData);

// On save, compute delta
const saveAndApply = async () => {
  const changed = Object.keys(entities).filter(key => 
    JSON.stringify(entities[key]) !== JSON.stringify(initialEntities[key])
  );
  
  const payload = {
    changed: changed.map(key => ({ key, ...entities[key] })),
    deleted: Object.keys(initialEntities).filter(k => !entities[k])
  };
  
  await saveEntitySchema(payload);
  setInitialEntities(entities); // Reset baseline
};
```

---

## Option 3: True Minimal Delta (Full Optimization)

### How it works:
```
Tracks: which exact field/property changed in which entity
Sends only that diff:
{
  "trades": { "entity_fields": [{ key: "new_field", ... }] },
  "clients": { "subtypes": { "new_subtype": { ... } } }
}
```

### Pros:
- ✅ Smallest possible payload
- ✅ Efficient network usage
- ✅ Clear change history

### Cons:
- ❌ Complex to implement
- ❌ Requires deep diff logic
- ❌ Overkill for most use cases

---

## Recommendation

**Go with Option 1 (Auto-Save)** because:

1. **Better UX**: Users see changes persist immediately (no guessing)
2. **Aligns with agents.md**: Tenant-scoped operations should be atomic
3. **Simpler backend**: Each endpoint is focused on one resource type
4. **Modern pattern**: Similar to Google Docs, Figma, etc.

### Proposed API Design for Option 1:

```
POST /api/entity-schema/entities
  - Create new entity
  - Body: { key: "trades", name: "Trades", entity_fields: [], subtypes: {} }

POST /api/entity-schema/entities/{entity_id}/fields
  - Add field to entity
  - Body: { level: "entity", name: "Field Name", type: "date" }

POST /api/entity-schema/entities/{entity_id}/fields/{field_id}
  - Update field

DELETE /api/entity-schema/entities/{entity_id}/fields/{field_id}
  - Delete field

POST /api/entity-schema/entities/{entity_id}/subtypes
  - Create subtype

POST /api/entity-schema/entities/{entity_id}/subtypes/{subtype_id}/fields
  - Add field to subtype
```

---

## What You Have Now

**Option 2 variant** - Save & Apply sends entire schema, which works but is inefficient.

## What Do You Want?

Let me know which approach you prefer and I'll refactor it:
- [ ] Option 1: Auto-save individual changes
- [ ] Option 2: Track changes, send deltas with SAVE & APPLY
- [ ] Option 3: Full minimal delta tracking
