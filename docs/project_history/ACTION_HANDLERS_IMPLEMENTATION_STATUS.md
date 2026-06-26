# Action Handlers Implementation Status

## Summary
Hasura action handlers for `search_business_terms`, `validate_business_term`, and `get_semantic_lineage` have been implemented in the api-gateway to proxy requests to the backend service instead of attempting circular GraphQL calls.

## Changes Made

### 1. Hasura Metadata Updates (`hasura/metadata/actions.yaml`)
- **search_business_terms**: Added explicit `arguments` section with query, tenant_id, limit, offset parameters and `output_type: SearchBusinessTermsResponse`
- Applied metadata successfully: `hasura metadata apply` confirmed INFO

### 2. API Gateway Configuration (`api-gateway/main.go`)

#### Config Struct (lines ~32)
- Added `BackendURL string` field to Config struct

#### Initialization (lines ~640)
- Set `config.BackendURL = getEnv("BACKEND_URL", "http://localhost:8080")` during config creation in main()

#### Handler Implementations

**handleBusinessTermSearch** (lines ~1519):
```go
func handleBusinessTermSearch(c *gin.Context, config Config) {
    // Binds JSON from Hasura request
    var req BusinessTermSearchRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    // Sets defaults
    if req.Limit == 0 {
        req.Limit = 20
    }

    // Proxies to backend /business-terms endpoint
    backendURL := fmt.Sprintf("%s/business-terms?query=%s&limit=%d&offset=%d",
        config.BackendURL,
        url.QueryEscape(req.Query),
        req.Limit,
        req.Offset,
    )

    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Get(backendURL)
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to reach backend service: " + err.Error()})
        return
    }

    // Forwards backend response directly to Hasura
    body, _ := io.ReadAll(resp.Body)
    c.Data(resp.StatusCode, "application/json", body)
}
```

**handleBusinessTermValidation** (lines ~1600):
```go
func handleBusinessTermValidation(c *gin.Context, config Config) {
    var req BusinessTermValidationRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    // Placeholder response - returns valid:true for all inputs
    // Production: would call backend validation service
    c.JSON(200, gin.H{
        "valid":    true,
        "errors":   []string{},
        "warnings": []string{},
    })
}
```

**handleSemanticLineage** (lines ~1620):
```go
func handleSemanticLineage(c *gin.Context, config Config) {
    nodeID := c.Query("node_id")
    tenantID := c.Query("tenant_id")
    depth := 2

    // Placeholder response - returns node with no edges
    // Production: would call backend lineage service
    c.JSON(200, gin.H{
        "nodes": []gin.H{{"id": nodeID, "name": nodeID, "type": "unknown"}},
        "edges": []gin.H{},
    })
}
```

### 3. Docker Configuration (`docker-compose.yml`)
- `API_GATEWAY_AUTH_TOKEN=Bearer test-token-12345` set in both Hasura and api-gateway services
- `BACKEND_URL=http://backend:8080` set in api-gateway service
- Bearer token bypass implemented in JWT middleware (lines ~229-237 of main.go)

## Status

### ✅ Completed
- Hasura metadata structure fixed and applied
- `config.BackendURL` properly initialized from `BACKEND_URL` env var
- `handleBusinessTermSearch` refactored to proxy backend
- `handleBusinessTermValidation` updated with placeholder response
- `handleSemanticLineage` updated with placeholder response
- Bearer token authentication working (Hasura → API Gateway path verified)
- Network connectivity validated (Hasura and api-gateway communicating)

### ⏳ Pending
- **Docker rebuild verification**: Confirm new handler code is actually deployed (Docker caching issues encountered)
- **End-to-end test**: Test `search_business_terms` action through Hasura GraphQL with real backend data
- **Remaining handlers**: dynamic_insert/update/delete, getRelatedObjects, getRelationshipSuggestions, applyRelationship, dismissRelationshipSuggestion
- **Production handlers**: Replace placeholder responses in validation/lineage handlers with real backend calls once backend endpoints verified

## Next Steps

1. **Rebuild Docker image from scratch** (ensure no caching):
   ```bash
   docker rmi semlayer-api-gateway:latest
   docker compose build --no-cache api-gateway
   docker compose restart api-gateway
   ```

2. **Test search_business_terms action through Hasura**:
   ```bash
   curl -X POST http://localhost:8083/v1/graphql \
     -H "Content-Type: application/json" \
     -H "x-hasura-admin-secret: newadminsecretkey" \
     -d '{"query":"mutation {search_business_terms(query: \"revenue\", limit: 5, offset: 0) {results {id display_name} total}}"}'
   ```

3. **Verify backend response** is properly returned from handler

4. **Implement remaining handlers** using same pattern (proxy to backend or stub responses)

5. **Remove debug logging**:
   - Remove `[JWT]` debug log from JWTMiddleware (lines ~119-120)
   - Remove any handler-level debug logs

6. **Final smoke test** with all actions working end-to-end

## Technical Notes

- **Config Capture**: The config struct is captured by closure in the route handlers defined at line 920-926, so all route handlers have access to the initialized config values
- **Backend URL Path**: Full URL is constructed in handler as `config.BackendURL + "/business-terms?query=..."`, so `config.BackendURL` must NOT have a trailing slash
- **JSON Binding**: Hasura sends request body with arguments at top level (not wrapped), so `BusinessTermSearchRequest` struct fields must match Hasura action argument names
- **Environment**: `BACKEND_URL` env var is set in docker-compose.yml and available during main() execution
