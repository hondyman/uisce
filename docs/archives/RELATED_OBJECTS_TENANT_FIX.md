# Related Objects Tenant Scope Fix

## Problem
The Related Objects feature was failing with error:
```
Error loading related objects: environment variable 'TENANT_ID' not set
```

This occurred because Hasura GraphQL actions were configured to read tenant scope from environment variables (`TENANT_ID`, `DATASOURCE_ID`) which are not set in the environment. The frontend was correctly passing `tenantId` and `datasourceId` as GraphQL query parameters, but Hasura was not converting these to the backend API's expected format.

## Root Cause
The Hasura action configuration files had incorrect header mappings:

**Before (❌ BROKEN):**
```yaml
- name: getRelatedObjects
  definition:
    handler: http://api-gateway:8000/api/relationships/objects
    headers:
      - name: X-Tenant-ID
        value_from_env: TENANT_ID  # ❌ Environment variable not set
      - name: X-Tenant-Datasource-ID
        value_from_env: DATASOURCE_ID  # ❌ Environment variable not set
```

The backend API (`api.go:5905`) correctly expects query parameters:
```go
tenantID := r.URL.Query().Get("tenant_id")
datasourceID := r.URL.Query().Get("datasource_id")
```

But Hasura wasn't passing the GraphQL arguments as query parameters.

## Solution
Updated Hasura action configurations to use `request_transform` with `Kriti` template engine to:
1. Extract `tenantId` and `datasourceId` from GraphQL input arguments
2. Convert them to URL query parameters that the backend API expects
3. Remove environment variable dependencies

**After (✅ FIXED):**
```yaml
- name: getRelatedObjects
  definition:
    handler: http://api-gateway:8000/api/relationships/objects
    forward_client_headers: true
    request_transform:
      version: 2
      template_engine: Kriti
      url:
        value: 'http://api-gateway:8000/api/relationships/objects?tenant_id={{$body.input.tenantId}}&datasource_id={{$body.input.datasourceId}}&entity={{$body.input.entity}}'
      body:
        action: remove  # No body needed, parameters in URL
    headers:
      - name: Content-Type
        value: application/json
      - name: Authorization
        value_from_env: API_GATEWAY_AUTH_TOKEN
```

## Files Changed

### 1. `/metadata/actions.yaml`
Updated 4 relationship-related GraphQL actions:
- `getRelatedObjects` - Added URL template with query parameters
- `getRelationshipSuggestions` - Added URL template with query parameters  
- `applyRelationship` - Added URL template + body transformation
- `dismissRelationshipSuggestion` - Added URL template + body transformation

All now use `request_transform` to pass arguments as query parameters to backend.

### 2. `/hasura/metadata/actions.yaml`
Applied identical fixes to the backup Hasura metadata configuration.

## How It Works

### Frontend (Already Correct)
```typescript
// RelatedObjectsPanel.tsx
const { data, loading, error } = useQuery(GET_RELATED_OBJECTS, {
  variables: { tenantId, datasourceId, entity },  // ✅ Passed to GraphQL
  fetchPolicy: "cache-and-network",
});
```

### GraphQL Query
```graphql
query GetRelatedObjects($tenantId: ID!, $datasourceId: ID!, $entity: String!) {
  getRelatedObjects(tenantId: $tenantId, datasourceId: $datasourceId, entity: $entity) {
    # ... fields
  }
}
```

### Hasura Transformation (Now Fixed)
Kriti template extracts arguments from `$body.input`:
```
$body.input.tenantId  → ?tenant_id=<value>
$body.input.datasourceId → &datasource_id=<value>
$body.input.entity → &entity=<value>
```

### Backend API (Already Correct)
```go
func (s *Server) getRelatedObjects(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")      // ✅ Reads from query params
	datasourceID := r.URL.Query().Get("datasource_id")  // ✅ Reads from query params
	entity := r.URL.Query().Get("entity")            // ✅ Reads from query params
	// ... rest of handler
}
```

## Testing
After deploying these changes:

1. ✅ Select a tenant and datasource in the Fabric Builder UI
2. ✅ Navigate to Entity Manager
3. ✅ Click on "🔗 Relationships" tab
4. ✅ Select an entity from the dropdown
5. ✅ Related Objects should load without errors
6. ✅ AI Suggest button should work for relationship suggestions
7. ✅ Apply/Dismiss relationship actions should work

## Configuration Details

### `request_transform` Structure
```yaml
request_transform:
  version: 2  # Hasura v2 format
  template_engine: Kriti  # Template language
  url:
    value: 'http://...?param={{$body.input.argName}}'  # URL template
  body:
    action: remove | transform  # What to do with request body
    template: '...'  # (Optional) Body template if transform
```

### Query Parameter Syntax
- `{{$body.input.tenantId}}` - Extracts `tenantId` argument from GraphQL input
- `{{$body.input.datasourceId}}` - Extracts `datasourceId` argument
- `{{$body.input.entity}}` - Extracts `entity` argument
- URL encoding is handled automatically by Hasura

### Mutation Body Transformation
For mutations like `applyRelationship`, the body is reconstructed to match backend expectations:
```json
{
  "sourceEntity": "value",
  "targetEntity": "value",
  "edgeType": "REFERENCE",
  "cardinality": "1:N",
  "fkColumn": "user_id",
  "confidence": 0.95
}
```

## Deployment Steps

1. Deploy updated `metadata/actions.yaml` to Hasura
2. Deploy updated `hasura/metadata/actions.yaml` backup
3. Hasura will automatically reload GraphQL actions
4. Test relationship feature in UI

No backend code changes required - the Go API handlers already work correctly once they receive the proper query parameters.

## Notes

- **Tenant Scope**: The tenant context is provided by the user selecting it in the Fabric Builder UI, not from environment
- **Headers**: `X-Tenant-ID` and `X-Tenant-Datasource-ID` are now passed via `forward_client_headers: true` instead of environment
- **Authorization**: API Gateway token still comes from environment variable (appropriate for service-to-service communication)
- **Backward Compatibility**: These changes only affect the GraphQL action layer; backend API handlers remain unchanged
