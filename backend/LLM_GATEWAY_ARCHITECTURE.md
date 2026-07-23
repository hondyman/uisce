# SemLayer LLM Gateway - Unified Natural Language to SQL Pipeline

## Overview

The LLM Gateway provides a **unified, deterministic pipeline** for converting natural language queries into SQL:

```
Natural Language Query
    ↓
Planner LLM (NL → Semantic Query JSON)
    ↓
Validation (Semantic Query ↔ Bundle)
    ↓
Executor (Semantic Query + Bundle → SQL)
    ↓
SQL Execution (Query → Database Rows)
    ↓
Response (Rows + Metadata)
```

This architecture ensures that:
- **LLMs never directly access the database** - they only emit structured JSON and SQL
- **All field references are validated** against the semantic bundle before execution
- **UUID-first architecture** prevents name collisions and supports field renames
- **Reproducibility** - same query produces same SQL across runs

---

## API Endpoints

### 1. Main Gateway: `POST /api/llm/query`

**Purpose**: Full end-to-end pipeline: NL → Planner → Semantic Query → Executor → SQL → DB

**Request**:
```json
{
  "datasource": "customers",
  "version": "v1",
  "prompt": "Show me all US-based retail customers created since January, ordered by most recent",
  "mode": "exploratory"
}
```

**Response**:
```json
{
  "datasource": "customers",
  "version": "v1",
  "semantic_sql": "{\"datasource\":\"customers\",\"select\":[\"id\",\"name\",\"email\"],\"filters\":[{\"field\":\"country\",\"op\":\"=\",\"value\":\"US\"},{\"field\":\"customer_type\",\"op\":\"=\",\"value\":\"RETAIL\"}],\"order_by\":[{\"field\":\"created_at\",\"direction\":\"desc\"}],\"limit\":100}",
  "generated_sql": "SELECT t0.id AS \"ID\", t0.name AS \"Name\", t0.email AS \"Email\" FROM customers t0 WHERE t0.country = 'US' AND t0.customer_type = 'RETAIL' ORDER BY t0.created_at DESC LIMIT 100",
  "rows": [
    {"ID": "123e4567-e89b-12d3-a456-426614174000", "Name": "Alice Corp", "Email": "alice@example.com"},
    {"ID": "223e4567-e89b-12d3-a456-426614174001", "Name": "Bob Inc", "Email": "bob@example.com"}
  ],
  "count": 2
}
```

**Modes**:
- `exploratory` (default): Loose interpretation; infers "interesting" fields if user is vague; default limit 100
- `strict`: Strict validation; only explicit fields; errors if nothing maps
- `crud`: Parse as create/update/delete operation

---

### 2. Debug: `POST /api/llm/planner`

**Purpose**: Test the planner stage in isolation (NL → Semantic Query JSON)

**Request**:
```json
{
  "datasource": "customers",
  "prompt": "US retail customers since January",
  "mode": "exploratory"
}
```

**Response** (Semantic Query JSON):
```json
{
  "datasource": "customers",
  "version": null,
  "select": ["id", "name", "email", "created_at"],
  "filters": [
    {"field": "country", "op": "=", "value": "US"},
    {"field": "customer_type", "op": "=", "value": "RETAIL"},
    {"field": "created_at", "op": ">=", "value": "2026-01-01"}
  ],
  "order_by": [{"field": "created_at", "direction": "desc"}],
  "limit": 100
}
```

**Use this to**:
- Verify the planner correctly interprets natural language
- Validate field names and filter logic before full execution
- Debug mode-specific behavior

---

### 3. Debug: `POST /api/llm/executor`

**Purpose**: Test the executor stage in isolation (Semantic Query + Bundle → SQL)

**Request**:
```json
{
  "datasource": "customers",
  "query": {
    "datasource": "customers",
    "select": ["id", "name", "email"],
    "filters": [
      {"field": "country", "op": "=", "value": "US"}
    ],
    "order_by": [{"field": "created_at", "direction": "desc"}],
    "limit": 100
  }
}
```

**Response**:
```json
{
  "datasource": "customers",
  "sql": "SELECT t0.id AS \"ID\", t0.name AS \"Name\", t0.email AS \"Email\" FROM customers t0 WHERE t0.country = 'US' ORDER BY t0.created_at DESC LIMIT 100"
}
```

**Use this to**:
- Verify SQL generation from semantic queries
- Validate table/column name resolution
- Test join logic and subtype handling
- Debug physical mappings

---

### 4. Diagnostic: `GET /api/llm/prompts`

**Purpose**: Retrieve the golden system prompts for planner and executor

**Response**:
```json
{
  "planner_prompt": {
    "exploratory": "You are a semantic query planner...",
    "strict": "MODE: STRICT_REPORTING...",
    "crud": "MODE: CRUD_VALIDATION..."
  },
  "executor_prompt": "You are an expert semantic SQL generator...",
  "documentation": "See https://github.com/yourusername/semlayer/wiki/LLM-Prompts"
}
```

**Use this to**:
- Copy exact prompts to your LLM (OpenAI, Claude, etc.)
- Validate prompt versions
- Document the system contract

---

### 5. Diagnostic: `GET /api/llm/modes`

**Purpose**: Information about available query modes

**Response**:
```json
{
  "modes": {
    "exploratory": "Loose defaults; auto-select interesting fields; default limit 100",
    "strict": "No assumptions; only explicit fields; error if nothing maps",
    "crud": "Parse as create/update/delete operation; emit CRUDOperation JSON"
  },
  "default_mode": "exploratory",
  "example_request": {
    "datasource": "customers",
    "version": "v1",
    "prompt": "Show me all US-based retail customers created since January, ordered by most recent",
    "mode": "exploratory"
  }
}
```

---

## Architecture

### Core Types

```go
// SemanticQuery: Intermediate format between NL and SQL
type SemanticQuery struct {
    Datasource  string      // e.g., "customers"
    Version     *string     // e.g., "v1"
    Select      []string    // Field names: ["id", "name", "email"]
    Filters     []Filter    // WHERE conditions
    OrderBy     []OrderBy   // Sort instructions
    Limit       int         // Row limit (default 100, max 100000)
}

// Filter: A single WHERE condition
type Filter struct {
    Field string      // Field name (validated against bundle)
    Op    string      // Operator: "=", ">", "IN", "LIKE", etc.
    Value interface{} // Literal value (string, number, array, etc.)
}

// OrderBy: A single sort instruction
type OrderBy struct {
    Field     string // Field name
    Direction string // "asc" or "desc"
}
```

### Gateway Flow (Pseudocode)

```go
func (gw *LLMGateway) ProcessQuery(ctx, req) (*SemanticQueryResponse, error) {
    // 1. Load semantic bundle (cached)
    bundle := loadBundle(req.Datasource, req.Version)
    
    // 2. Call Planner LLM: NL → SemanticQuery JSON
    semQuery := callPlannerLLM(bundle, req.Prompt, req.Mode)
    
    // 3. Validate semantic query against bundle
    validateSemanticQuery(bundle, semQuery)
    
    // 4. Call Executor: SemanticQuery + Bundle → SQL
    sql := callExecutorLLM(bundle, semQuery)
    
    // 5. Execute SQL
    rows := executeSQL(sql)
    
    // 6. Return response
    return SemanticQueryResponse{
        Datasource:   req.Datasource,
        SemanticSQL:  semQuery,
        GeneratedSQL: sql,
        Rows:         rows,
    }
}
```

---

## Integration Guide

### Step 1: Configure LLM Service

Update `llm_gateway.go` to wire in your LLM provider:

```go
func (gw *LLMGateway) callPlannerLLM(ctx, bundle, prompt, mode) (*SemanticQuery, error) {
    systemPrompt := GoldenPlannerSystemPrompt(mode)
    
    // Call your LLM service (OpenAI, Claude, etc.)
    response := yourLLMClient.ChatCompletion(ctx, &ChatRequest{
        System:   systemPrompt,
        Context:  marshalBundleForLLM(bundle),
        UserMsg:  prompt,
    })
    
    // Parse response as SemanticQuery
    var q SemanticQuery
    json.Unmarshal(response.Content, &q)
    return &q, nil
}

func (gw *LLMGateway) callExecutorLLM(ctx, bundle, q) (string, error) {
    systemPrompt := GoldenSQLSystemPrompt()
    
    response := yourLLMClient.ChatCompletion(ctx, &ChatRequest{
        System:   systemPrompt,
        Context:  marshalBundleForLLM(bundle),
        UserMsg:  marshalSemanticQueryForLLM(q),
    })
    
    return response.Content, nil
}
```

### Step 2: Test Each Stage

**1. Test planner inference**:
```bash
curl -X POST http://localhost:8080/api/llm/planner \
  -H "Content-Type: application/json" \
  -d '{
    "datasource": "customers",
    "prompt": "show me recent customers in california",
    "mode": "exploratory"
  }'
```

**2. Test executor generation**:
```bash
curl -X POST http://localhost:8080/api/llm/executor \
  -H "Content-Type: application/json" \
  -d '{
    "datasource": "customers",
    "query": {
      "datasource": "customers",
      "select": ["id", "name"],
      "filters": [{"field": "state", "op": "=", "value": "CA"}],
      "limit": 100
    }
  }'
```

**3. Test full pipeline**:
```bash
curl -X POST http://localhost:8080/api/llm/query \
  -H "Content-Type: application/json" \
  -d '{
    "datasource": "customers",
    "prompt": "show me recent customers in california",
    "mode": "exploratory"
  }'
```

---

## Semantic Bundle Contract

The **Semantic Bundle** is the single source of truth for the LLM. It includes:

```json
{
  "business_object_id": "123e4567-e89b-12d3-a456-426614174000",
  "business_object_name": "customers",
  "datasource_id": "pg-prod-01",
  "driving_table": "public.customers",
  "version": "v1",
  "fields": [
    {
      "field_id": "abcd-1234",
      "name": "id",
      "display_name": "Customer ID",
      "semantic_term": "customer_identifier",
      "physical": {
        "datasource_id": "pg-prod-01",
        "table": "public.customers",
        "column": "id"
      }
    },
    {
      "field_id": "efgh-5678",
      "name": "country",
      "display_name": "Country",
      "semantic_term": "geo_country",
      "physical": {
        "datasource_id": "pg-prod-01",
        "table": "public.customers",
        "column": "country_code"
      }
    }
  ],
  "relationships": [
    {
      "relationship_id": "rel-001",
      "target_business_object": "orders",
      "join_type": "INNER",
      "source_column": "id",
      "target_column": "customer_id"
    }
  ]
}
```

---

## Error Handling

### Validation Errors

If the semantic query is invalid:
```json
{
  "datasource": "customers",
  "error": "unknown field: loyalty_points"
}
```

### SQL Execution Errors

If SQL generation or execution fails:
```json
{
  "datasource": "customers",
  "error": "SQL execution failed: connection timeout"
}
```

### LLM Errors

If the LLM is not configured:
```json
{
  "datasource": "customers",
  "error": "LLM planner not yet configured; configure LLM_API_ENDPOINT and credentials"
}
```

---

## Performance & Safety

### Caching
- Semantic bundles are cached at load time
- Name resolver uses pre-loaded O(1) cache
- Queries are parameterized to prevent SQL injection

### Limits
- Default limit: 100 rows
- Maximum limit: 100,000 rows
- Query timeout: Inherits from database connection pool

### Validation Layers
1. **HTTP layer**: Request struct validation
2. **Semantic layer**: Field existence checks
3. **Database layer**: SQL parameterization, RLS policies
4. **LLM layer**: Deterministic prompts, bounded outputs

---

## Example Use Cases

### 1. Business Analyst: Natural Language Analytics
```
User: "How many new customers did we acquire last month, by region?"

Pipeline:
- Planner infers: SELECT customer_id, created_at, region FROM customers WHERE created_at >= '2025-01-01' GROUP BY region
- Executor validates fields and generates SQL
- Database returns aggregated results
```

### 2. Data Engineer: Strict Report Definition
```
User request (mode: "strict"):
{
  "datasource": "sales",
  "prompt": "Q3 revenue by product_id and sales_region",
  "mode": "strict"
}

Pipeline:
- Planner requires EXACT field names
- Executor validates strict schema
- Returns deterministic, reproducible report
```

### 3. Data Scientist: Exploratory Analysis
```
User request (mode: "exploratory"):
{
  "datasource": "transactions",
  "prompt": "anomalies in recent transactions",
  "mode": "exploratory"
}

Pipeline:
- Planner auto-selects "interesting" numeric and categorical fields
- Executor finds reasonable defaults
- Returns sample data for exploration
```

---

## Security Considerations

### Never Trust LLM Output Directly
- All field names validated against bundle
- All table/column names resolved from metadata
- SQL is parameterized before execution
- No direct SQL from LLM ever executes

### Multi-Tenant Isolation
- All queries filtered by `tenant_id` at database layer
- RLS policies prevent cross-tenant data leaks
- LLM responses cannot bypass tenant boundaries

### Rate Limiting
- Consider adding rate limits per datasource/mode
- Monitor LLM API usage costs
- Implement query complexity scoring

---

## Next Steps

1. **Wire LLM Service**: Implement `callPlannerLLM()` and `callExecutorLLM()`
2. **Test with Real Data**: Run full pipeline against staging database
3. **Add Monitoring**: Log planner/executor outputs for debugging
4. **Optimize Prompts**: Tune golden prompts for your data model
5. **Deploy to Production**: Add to CI/CD pipeline

---

## References

- **Golden Prompts**: Available at `GET /api/llm/prompts`
- **Semantic Bundle Spec**: See [SemanticBundle struct](./api.go#L224)
- **Supported Operators**: `=`, `!=`, `>`, `<`, `>=`, `<=`, `IN`, `LIKE`, `ILIKE`, `IS NULL`
- **LLM Provider Docs**: OpenAI, Claude, LLaMA, etc.
