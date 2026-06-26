# Quick Fix: Track Changes Only (Option 2 Implementation)

This is the minimal refactor to make the existing SAVE & APPLY button only send changed entities instead of the entire schema.

## Changes Needed

### 1. Add change tracking to EntityConfigPage.tsx

Replace the useState for entities with:

```typescript
const [initialEntities, setInitialEntities] = useState<Entities>(initialData);
const [entities, setEntities] = useState<Entities>(initialData);

// Compute what changed
const computeChanges = () => {
  const changed: string[] = [];
  const deleted: string[] = [];

  // Find changed entities
  for (const key of Object.keys(entities)) {
    if (!(key in initialEntities)) {
      // New entity
      changed.push(key);
    } else if (JSON.stringify(entities[key]) !== JSON.stringify(initialEntities[key])) {
      // Modified entity
      changed.push(key);
    }
  }

  // Find deleted entities
  for (const key of Object.keys(initialEntities)) {
    if (!(key in entities)) {
      deleted.push(key);
    }
  }

  return { changed, deleted };
};

const { changed, deleted } = useMemo(computeChanges, [entities, initialEntities]);
```

### 2. Update saveAndApply to send only changes

```typescript
const saveAndApply = async () => {
  devLog('[EntityConfigPage.saveAndApply] Starting save...');
  
  if (!hasTenantScope()) {
    devLog('[EntityConfigPage.saveAndApply] ERROR: No tenant scope!');
    message.error('Please select a tenant and datasource first');
    return;
  }

  const { changed, deleted } = computeChanges();

  if (changed.length === 0 && deleted.length === 0) {
    message.info('No changes to save');
    return;
  }

  try {
    const scope = getRequiredTenantScope();
    devLog('[EntityConfigPage.saveAndApply] Tenant scope confirmed:', { scope });
    devLog('[EntityConfigPage.saveAndApply] Changes:', { 
      changed: changed.length, 
      deleted: deleted.length,
      changedEntities: changed.map(k => ({ [k]: entities[k] }))
    });
  } catch (err) {
    devLog('[EntityConfigPage.saveAndApply] ERROR reading tenant scope:', { err });
    message.error('Tenant scope error - please reload and select again');
    return;
  }
  
  setIsSaving(true);
  try {
    const payload = {
      changed: Object.fromEntries(
        changed.map(key => [key, entities[key]])
      ),
      deleted: deleted,
    };
    
    devLog('[EntityConfigPage.saveAndApply] Calling saveEntitySchema with changes...');
    await saveEntitySchema(payload);
    
    // Update baseline after successful save
    setInitialEntities(entities);
    
    devLog('[EntityConfigPage.saveAndApply] Success!');
    message.success(`Saved ${changed.length} entities${deleted.length > 0 ? ` and deleted ${deleted.length}` : ''}!`);
  } catch (error) {
    devLog('[EntityConfigPage.saveAndApply] Failed:', { error });
    message.error(`Failed to save schema: ${error instanceof Error ? error.message : String(error)}`);
  } finally {
    setIsSaving(false);
  }
};
```

### 3. Update entitySchema.ts to handle change payload

```typescript
export interface EntitySchemaPayload {
  changed?: Record<string, Entity>;
  deleted?: string[];
}

export function saveEntitySchema(payload: EntitySchemaPayload | Entities): Promise<void> {
  devLog('[saveEntitySchema] Saving schema:', { payload });
  
  const body = JSON.stringify(payload);
  devLog('[saveEntitySchema] Request body size:', { size: body.length });
  
  return fetchAPI('/entity-schema', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body,
  }).then((result) => {
    devLog('[saveEntitySchema] Save successful:', { result });
  }).catch((error) => {
    devLog('[saveEntitySchema] Save failed:', { error });
    throw error;
  });
}
```

### 4. Update Backend to Handle Changes

In `backend/internal/api/api.go`, modify the `/entity-schema` endpoint:

```go
r.Post("/entity-schema", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")

    tenantID := r.Header.Get("X-Tenant-ID")
    tenantDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")

    if tenantID == "" || tenantDatasourceID == "" {
        http.Error(w, "X-Tenant-ID and X-Tenant-Datasource-ID headers are required", http.StatusBadRequest)
        return
    }

    var payload map[string]interface{}
    if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Check if this is a delta (changed/deleted) or full schema
    changed, hasChanged := payload["changed"]
    deleted, hasDeleted := payload["deleted"]

    var schemaData map[string]interface{}

    if hasChanged || hasDeleted {
        // Delta update: fetch existing schema and merge
        var existingData map[string]interface{}
        err := srv.DB.QueryRowContext(r.Context(), `
            SELECT schema_data FROM public.entity_schema 
            WHERE tenant_id = $1 AND tenant_datasource_id = $2
        `, tenantID, tenantDatasourceID).Scan(pq.JSON(&existingData))

        if err != nil && err != sql.ErrNoRows {
            http.Error(w, fmt.Sprintf("Failed to fetch existing schema: %v", err), http.StatusInternalServerError)
            return
        }

        if existingData == nil {
            existingData = make(map[string]interface{})
        }

        // Apply changes
        if changedMap, ok := changed.(map[string]interface{}); ok {
            for k, v := range changedMap {
                existingData[k] = v
            }
        }

        // Apply deletions
        if deletedList, ok := deleted.([]interface{}); ok {
            for _, d := range deletedList {
                if key, ok := d.(string); ok {
                    delete(existingData, key)
                }
            }
        }

        schemaData = existingData
    } else {
        // Full schema replace
        schemaData = payload
    }

    // Convert to JSON for storage
    schemaJSON, err := json.Marshal(schemaData)
    if err != nil {
        http.Error(w, fmt.Sprintf("Failed to marshal schema data: %v", err), http.StatusInternalServerError)
        return
    }

    // Upsert the entity schema
    _, err = srv.DB.ExecContext(r.Context(), `
        INSERT INTO public.entity_schema (tenant_id, tenant_datasource_id, schema_data, updated_at)
        VALUES ($1, $2, $3, NOW())
        ON CONFLICT (tenant_id, tenant_datasource_id)
        DO UPDATE SET schema_data = EXCLUDED.schema_data, updated_at = NOW()
    `, tenantID, tenantDatasourceID, schemaJSON)

    if err != nil {
        http.Error(w, fmt.Sprintf("Failed to save entity schema: %v", err), http.StatusInternalServerError)
        return
    }

    result := map[string]interface{}{
        "success": true,
        "message": "Entity schema saved successfully",
    }
    json.NewEncoder(w).Encode(result)
})
```

## Result

### Before (Current)
```json
POST /api/entity-schema
{
  "trades": { "name": "Trades", "entity_fields": [...], "subtypes": {...} },
  "clients": { "name": "Clients", ... },
  "portfolios": { "name": "Portfolios", ... },
  "hhhhh": { "name": "hhhhh", ... }  // All entities sent every time
}
```

### After (Option 2)
```json
POST /api/entity-schema
{
  "changed": {
    "hhhhh": { "name": "hhhhh", "entity_fields": [], "subtypes": {} }  // Only new/changed
  },
  "deleted": []
}
```

## Benefits

✅ Smaller payload (only changed entities)
✅ Backend knows what was deleted
✅ Can track who changed what for audit logs
✅ Minimal changes to existing code
✅ Backward compatible (full schema still works)

---

Want me to implement this, or would you prefer the auto-save approach (Option 1)?
