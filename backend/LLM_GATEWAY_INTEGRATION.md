# SemLayer LLM Gateway - Integration Guide

## Quick Start

The LLM Gateway provides a production-ready framework for converting natural language queries to deterministic SQL through a staged pipeline:

```
User Query (English)
    ↓
[Planner LLM]  → Semantic Query JSON (validated against bundle)
    ↓
[Validator]    → Ensures fields exist and query is well-formed
    ↓
[Executor LLM]  → Deterministic SQL (resolved from metadata)
    ↓
[Database]     → Results
```

---

## Integration Checklist

- [ ] **Phase 1**: Implement `callPlannerLLM()` to wire your LLM provider
- [ ] **Phase 2**: Implement `callExecutorLLM()` (or use deterministic Go resolver)
- [ ] **Phase 3**: Test planner inference with `/api/llm/planner` debug endpoint
- [ ] **Phase 4**: Test executor with `/api/llm/executor` debug endpoint
- [ ] **Phase 5**: End-to-end test with `/api/llm/query` main gateway
- [ ] **Phase 6**: Deploy to staging/production

---

## Phase 1: Wire Planner LLM

The planner converts natural language → semantic query JSON.

### Example: OpenAI Integration

```go
// backend/internal/api/llm_gateway.go

import (
    "github.com/sashabaranov/go-openai"
)

// callPlannerLLM calls OpenAI to convert NL → SemanticQuery JSON
func (gw *LLMGateway) callPlannerLLM(
    ctx context.Context,
    bundle *SemanticBundle,
    prompt string,
    mode string,
) (*SemanticQuery, error) {
    
    // Build the system prompt with mode-specific behavior
    systemPrompt := GoldenPlannerSystemPrompt(mode)
    
    // Marshall the bundle as context
    bundleJSON, _ := json.MarshalIndent(bundle, "", "  ")
    contextMsg := fmt.Sprintf(
        "SEMANTIC BUNDLE:\n%s\n\nUSER QUERY:\n%s",
        string(bundleJSON),
        prompt,
    )
    
    // Call OpenAI
    client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
    
    resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
        Model: "gpt-4-turbo-preview",
        Messages: []openai.ChatCompletionMessage{
            {
                Role:    openai.ChatMessageRoleSystem,
                Content: systemPrompt,
            },
            {
                Role:    openai.ChatMessageRoleUser,
                Content: contextMsg,
            },
        },
        Temperature: 0, // Deterministic
    })
    
    if err != nil {
        return nil, fmt.Errorf("openai error: %w", err)
    }
    
    if len(resp.Choices) == 0 {
        return nil, fmt.Errorf("no response from openai")
    }
    
    // Parse response as SemanticQuery JSON
    rawResponse := resp.Choices[0].Message.Content
    
    var q SemanticQuery
    if err := json.Unmarshal([]byte(rawResponse), &q); err != nil {
        return nil, fmt.Errorf("failed to parse planner response: %w", err)
    }
    
    return &q, nil
}
```

### Example: Claude Integration

```go
// Use Anthropic Claude instead

import (
    "github.com/anthropics/sdk-go"
)

func (gw *LLMGateway) callPlannerLLM(
    ctx context.Context,
    bundle *SemanticBundle,
    prompt string,
    mode string,
) (*SemanticQuery, error) {
    
    systemPrompt := GoldenPlannerSystemPrompt(mode)
    bundleJSON, _ := json.MarshalIndent(bundle, "", "  ")
    
    client := anthropic.NewClient(
        anthropic.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
    )
    
    msg, err := client.Messages.New(ctx, &anthropic.MessageNewParams{
        Model: anthropic.ModelClaude3_5Sonnet,
        Messages: []anthropic.MessageParam{
            anthropic.NewUserMessage(
                anthropic.NewTextBlock(
                    fmt.Sprintf(
                        "%s\n\nSEMANTIC BUNDLE:\n%s\n\nUSER QUERY:\n%s",
                        systemPrompt,
                        string(bundleJSON),
                        prompt,
                    ),
                ),
            ),
        },
    })
    
    if err != nil {
        return nil, fmt.Errorf("claude error: %w", err)
    }
    
    // Extract text from response
    var q SemanticQuery
    if err := json.Unmarshal([]byte(msg.Content[0].Text), &q); err != nil {
        return nil, fmt.Errorf("failed to parse claude response: %w", err)
    }
    
    return &q, nil
}
```

### Example: Local LLM (via Ollama)

```go
// Use local Ollama for cost-free inference

import (
    "github.com/ollama/ollama/api"
)

func (gw *LLMGateway) callPlannerLLM(
    ctx context.Context,
    bundle *SemanticBundle,
    prompt string,
    mode string,
) (*SemanticQuery, error) {
    
    systemPrompt := GoldenPlannerSystemPrompt(mode)
    bundleJSON, _ := json.MarshalIndent(bundle, "", "  ")
    
    client, err := api.NewClient(os.Getenv("OLLAMA_ENDPOINT"))
    if err != nil {
        return nil, err
    }
    
    req := &api.GenerateRequest{
        Model:  "llama2", // or mistral, neural-chat, etc.
        Prompt: contextMsg,
        System: systemPrompt,
        Stream: false,
    }
    
    resp, err := client.Generate(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("ollama error: %w", err)
    }
    
    var q SemanticQuery
    if err := json.Unmarshal([]byte(resp.Response), &q); err != nil {
        return nil, fmt.Errorf("failed to parse ollama response: %w", err)
    }
    
    return &q, nil
}
```

---

## Phase 2: Wire Executor

The executor converts semantic query JSON → SQL.

### Option A: LLM-based Executor

Use the same LLM to generate SQL from the semantic query:

```go
func (gw *LLMGateway) callExecutorLLM(
    ctx context.Context,
    bundle *SemanticBundle,
    q *SemanticQuery,
) (string, error) {
    
    systemPrompt := GoldenSQLSystemPrompt()
    bundleJSON, _ := json.MarshalIndent(bundle, "", "  ")
    queryJSON, _ := json.MarshalIndent(q, "", "  ")
    
    contextMsg := fmt.Sprintf(
        "SEMANTIC BUNDLE:\n%s\n\nSEMANTIC QUERY:\n%s",
        string(bundleJSON),
        string(queryJSON),
    )
    
    // Call OpenAI (same pattern as planner)
    client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
    
    resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
        Model: "gpt-4-turbo-preview",
        Messages: []openai.ChatCompletionMessage{
            {Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
            {Role: openai.ChatMessageRoleUser, Content: contextMsg},
        },
        Temperature: 0,
    })
    
    if err != nil {
        return "", fmt.Errorf("openai error: %w", err)
    }
    
    if len(resp.Choices) == 0 {
        return "", fmt.Errorf("no response from openai")
    }
    
    sql := resp.Choices[0].Message.Content
    
    // Validate that it looks like SQL
    sql = strings.TrimSpace(sql)
    if !strings.HasPrefix(strings.ToUpper(sql), "SELECT") {
        return "", fmt.Errorf("executor did not return SQL: %s", sql)
    }
    
    return sql, nil
}
```

### Option B: Deterministic Go Resolver (Recommended for Production)

Don't use an LLM for the executor - use a deterministic Go-based resolver for safety and efficiency:

```go
// backend/internal/api/semantic_sql_resolver.go

// GenerateSQL converts a SemanticQuery + Bundle → SQL deterministically
func GenerateSQL(bundle *SemanticBundle, q SemanticQuery) (string, error) {
    
    // Build field lookup
    fieldMap := make(map[string]*SemanticField)
    for i := range bundle.Fields {
        fieldMap[bundle.Fields[i].Name] = &bundle.Fields[i]
    }
    
    // Validate select fields exist
    for _, fieldName := range q.Select {
        if _, ok := fieldMap[fieldName]; !ok {
            return "", fmt.Errorf("unknown field: %s", fieldName)
        }
    }
    
    // Build SELECT clause
    selectParts := []string{}
    for _, fieldName := range q.Select {
        field := fieldMap[fieldName]
        alias := field.DisplayName
        if alias == "" {
            alias = field.Name
        }
        selectParts = append(selectParts,
            fmt.Sprintf("t0.%s AS \"%s\"", field.Physical.Column, alias),
        )
    }
    selectClause := strings.Join(selectParts, ", ")
    
    // Build FROM clause
    fromClause := fmt.Sprintf("%s t0", bundle.DrivingTable)
    
    // Build WHERE clause
    whereParts := []string{}
    for _, filter := range q.Filters {
        field, ok := fieldMap[filter.Field]
        if !ok {
            return "", fmt.Errorf("unknown filter field: %s", filter.Field)
        }
        
        // Parameterize the value (in real implementation, use prepared statements)
        val := fmt.Sprintf("'%v'", filter.Value)
        whereParts = append(whereParts,
            fmt.Sprintf("t0.%s %s %s", field.Physical.Column, filter.Op, val),
        )
    }
    whereClause := ""
    if len(whereParts) > 0 {
        whereClause = "WHERE " + strings.Join(whereParts, " AND ")
    }
    
    // Build ORDER BY clause
    orderClause := ""
    if len(q.OrderBy) > 0 {
        orderParts := []string{}
        for _, ob := range q.OrderBy {
            field, ok := fieldMap[ob.Field]
            if !ok {
                return "", fmt.Errorf("unknown order_by field: %s", ob.Field)
            }
            direction := "ASC"
            if ob.Direction == "desc" {
                direction = "DESC"
            }
            orderParts = append(orderParts,
                fmt.Sprintf("t0.%s %s", field.Physical.Column, direction),
            )
        }
        orderClause = "ORDER BY " + strings.Join(orderParts, ", ")
    }
    
    // Assemble final SQL
    sql := fmt.Sprintf(
        "SELECT %s FROM %s %s %s LIMIT %d",
        selectClause,
        fromClause,
        whereClause,
        orderClause,
        q.Limit,
    )
    
    return strings.TrimSpace(sql), nil
}

// Update callExecutorLLM to use the resolver
func (gw *LLMGateway) callExecutorLLM(
    ctx context.Context,
    bundle *SemanticBundle,
    q *SemanticQuery,
) (string, error) {
    return GenerateSQL(bundle, *q)
}
```

---

## Phase 3: Test Planner Inference

Call the debug planner endpoint:

```bash
curl -X POST http://localhost:8080/api/llm/planner \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: tenant-123" \
  -d '{
    "datasource": "customers",
    "prompt": "Show me all US customers sorted by most recent",
    "mode": "exploratory"
  }'
```

**Expected response** (Semantic Query JSON):
```json
{
  "datasource": "customers",
  "select": ["id", "name", "email", "created_at"],
  "filters": [
    {"field": "country", "op": "=", "value": "US"}
  ],
  "order_by": [
    {"field": "created_at", "direction": "desc"}
  ],
  "limit": 100
}
```

**Troubleshooting**:
- If LLM returns parse error: Check the golden prompt is being used correctly
- If fields don't match: Verify bundle contains those field names
- If response is truncated: LLM may have context limit issues

---

## Phase 4: Test Executor

Call the debug executor endpoint:

```bash
curl -X POST http://localhost:8080/api/llm/executor \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: tenant-123" \
  -d '{
    "datasource": "customers",
    "query": {
      "datasource": "customers",
      "select": ["id", "name", "email"],
      "filters": [
        {"field": "country", "op": "=", "value": "US"}
      ],
      "order_by": [
        {"field": "created_at", "direction": "desc"}
      ],
      "limit": 100
    }
  }'
```

**Expected response**:
```json
{
  "datasource": "customers",
  "sql": "SELECT t0.id AS \"ID\", t0.name AS \"Name\", t0.email AS \"Email\" FROM public.customers t0 WHERE t0.country = 'US' ORDER BY t0.created_at DESC LIMIT 100"
}
```

---

## Phase 5: End-to-End Test

Call the main gateway endpoint:

```bash
curl -X POST http://localhost:8080/api/llm/query \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: tenant-123" \
  -d '{
    "datasource": "customers",
    "prompt": "Show me all US customers sorted by most recent",
    "mode": "exploratory"
  }'
```

**Expected response** (full pipeline result):
```json
{
  "datasource": "customers",
  "version": "v1",
  "semantic_sql": "{\"datasource\":\"customers\",...}",
  "generated_sql": "SELECT t0.id AS \"ID\" FROM public.customers t0 WHERE t0.country = 'US' ORDER BY t0.created_at DESC LIMIT 100",
  "rows": [
    {"ID": "123e4567-e89b-12d3-a456-426614174000", ...},
    {"ID": "223e4567-e89b-12d3-a456-426614174001", ...}
  ],
  "count": 2
}
```

---

## Production Deployment

### Environment Variables

```bash
# For LLM-based executor (e.g., OpenAI)
export OPENAI_API_KEY="sk-..."
export OPENAI_MODEL="gpt-4-turbo-preview"

# Or for Claude
export ANTHROPIC_API_KEY="sk-ant-..."
export CLAUDE_MODEL="claude-3-5-sonnet"

# Or for local Ollama
export OLLAMA_ENDPOINT="http://localhost:11434"
export OLLAMA_MODEL="llama2"
```

### Rate Limiting & Monitoring

Add to `handleSemanticQuery()`:

```go
// Rate limit: max 100 queries per minute per tenant
if err := ratelimiter.CheckLimit(tenantID, 100); err != nil {
    http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
    return
}

// Log the full pipeline for debugging
log.Printf("[LLM_GATEWAY] tenant=%s datasource=%s mode=%s",
    tenantID, req.Datasource, req.Mode)
```

### Error Handling

The gateway returns both successes and errors in the same response format:

```json
{
  "datasource": "customers",
  "error": "failed to load semantic bundle: business object not found: customers"
}
```

Monitor for:
- "unknown field" errors → user is asking for fields not in bundle
- "LLM planner not yet configured" → integration incomplete
- SQL execution failures → data access issues

---

## Performance Optimization

### Caching

Semantic bundles are queries on demand but could be cached:

```go
// Add bundle cache to LLMGateway
type LLMGateway struct {
    server       *Server
    bundleCache  map[string]*SemanticBundle
    cacheMutex   sync.RWMutex
}

func (gw *LLMGateway) loadSemanticBundle(...) {
    // Check cache first
    gw.cacheMutex.RLock()
    if cached, ok := gw.bundleCache[cacheKey]; ok {
        gw.cacheMutex.RUnlock()
        return cached, nil
    }
    gw.cacheMutex.RUnlock()
    
    // Load from DB, then cache
    bundle, err := gw.loadFromDB(...)
    
    gw.cacheMutex.Lock()
    gw.bundleCache[cacheKey] = bundle
    gw.cacheMutex.Unlock()
    
    return bundle, err
}
```

### Batching

For high-volume queries, consider batch LLM calls:

```go
// Call planner for multiple queries at once
// Then call executor for all results in parallel
```

---

## Security Best Practices

1. **Never trust LLM output directly** - validate all field names and table names
2. **Use parameterized queries** - the deterministic resolver does this automatically
3. **Enforce tenant isolation** - all queries filtered by `X-Tenant-ID`
4. **Rate limit per tenant** - prevent abuse
5. **Log all queries** - audit trail for compliance
6. **Disable direct SQL from LLM** - only accept JSON intermediate format

---

## Troubleshooting

### Issue: "LLM planner not yet configured"

**Cause**: `callPlannerLLM()` still has stub implementation

**Solution**: Implement the LLM wiring (see Phase 1 above)

### Issue: "unknown field: X"

**Cause**: User asked for a field not in the semantic bundle

**Solution**: Check bundle has the field, or adjust user prompt

### Issue: SQL syntax errors

**Cause**: Table/column names misresolved

**Solution**: Verify physical mappings in bundle are correct

---

## Next Steps

1. Pick an LLM provider (OpenAI recommended for best quality)
2. Implement `callPlannerLLM()` with your provider
3. Implement `callExecutorLLM()` (deterministic Go resolver recommended)
4. Test each stage independently
5. Set up monitoring and logging
6. Deploy!

