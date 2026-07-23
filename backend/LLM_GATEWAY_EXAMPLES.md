# LLM Gateway - End-to-End Examples

This document provides concrete examples of requests and responses through the full LLM gateway pipeline.

---

## Scenario 1: Exploratory Analytics - "Recent US Customers"

### 1a. Step 1 - User Sends Natural Language

```bash
curl -X POST http://localhost:8080/api/llm/query \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: acme-corp" \
  -d '{
    "datasource": "customers",
    "prompt": "Show me recent customers in the US, sorted by most recent",
    "mode": "exploratory"
  }'
```

### 1b. Step 2 - Planner LLM (NL → Semantic Query)

**System Prompt** (with MODE: EXPLORATORY):
```
You are a semantic query planner for the SemLayer platform.
...
MODE: EXPLORATORY
- If the user is vague, choose a small, sensible default set of fields.
- Always include a LIMIT (default 100).
...
```

**LLM Internal Processing**:
- User says "recent customers in the US"
- Planner infers:
  - "recent" → `created_at` field (likely present)
  - "customers in the US" → `country == "US"` filter
  - "sorted by most recent" → `ORDER BY created_at DESC`
  - Exploratory mode → auto-select reasonable fields

**LLM Response** (returned as JSON):
```json
{
  "datasource": "customers",
  "version": null,
  "select": ["id", "name", "email", "country", "created_at"],
  "filters": [
    {
      "field": "country",
      "op": "=",
      "value": "US"
    }
  ],
  "order_by": [
    {
      "field": "created_at",
      "direction": "desc"
    }
  ],
  "limit": 100
}
```

### 1c. Step 3 - Validation

The gateway validates this semantic query against the bundle:

**Semantic Bundle** (abbreviated):
```json
{
  "business_object_id": "bo-cust-001",
  "business_object_name": "customers",
  "fields": [
    {"field_id": "f-id", "name": "id", "display_name": "Customer ID", "physical": {"column": "customer_id"}},
    {"field_id": "f-email", "name": "email", "display_name": "Email", "physical": {"column": "email_address"}},
    {"field_id": "f-country", "name": "country", "display_name": "Country", "physical": {"column": "country_code"}},
    {"field_id": "f-created", "name": "created_at", "display_name": "Created", "physical": {"column": "created_at"}},
    {"field_id": "f-name", "name": "name", "display_name": "Customer Name", "physical": {"column": "full_name"}}
  ]
}
```

**Validation Checks**:
- ✅ `id`, `name`, `email`, `country`, `created_at` all exist in bundle
- ✅ `country` filter field exists
- ✅ `created_at` order_by field exists
- ✅ Limit is within bounds (100 <= 100k)

**Result**: Validation passes ✅

### 1d. Step 4 - Executor (Semantic Query → SQL)

**System Prompt**:
```
You are an expert semantic SQL generator for the SemLayer platform.
You NEVER guess table names, column names, joins, or filters.
You ONLY use the metadata provided in the Semantic Bundle.
...
```

**LLM Receives**:
1. System prompt (the golden SQL generator)
2. Bundle metadata (table names, column names, physical mappings)
3. Semantic query (what to select, filter, order by)

**LLM Processes**:
- Resolve `id` → `t0.customer_id` (from bundle's physical mapping)
- Resolve `name` → `t0.full_name`
- Resolve `email` → `t0.email_address`
- Resolve `country` → `t0.country_code`
- Resolve `created_at` → `t0.created_at`
- Resolve filter: `country = "US"` → `t0.country_code = 'US'`
- Resolve order: `created_at DESC` → `t0.created_at DESC`
- Resolve table: `public.customers`

**LLM Response** (SQL string):
```sql
SELECT 
  t0.customer_id AS "Customer ID",
  t0.full_name AS "Customer Name",
  t0.email_address AS "Email",
  t0.country_code AS "Country",
  t0.created_at AS "Created"
FROM public.customers t0
WHERE t0.country_code = 'US'
ORDER BY t0.created_at DESC
LIMIT 100
```

### 1e. Step 5 - Execute Query

Database executes the SQL and returns rows.

### 1f. Step 6 - Full Response

```json
{
  "datasource": "customers",
  "version": "v1",
  "semantic_sql": "{\"datasource\":\"customers\",\"version\":null,\"select\":[\"id\",\"name\",\"email\",\"country\",\"created_at\"],\"filters\":[{\"field\":\"country\",\"op\":\"=\",\"value\":\"US\"}],\"order_by\":[{\"field\":\"created_at\",\"direction\":\"desc\"}],\"limit\":100}",
  "generated_sql": "SELECT t0.customer_id AS \"Customer ID\", t0.full_name AS \"Customer Name\", t0.email_address AS \"Email\", t0.country_code AS \"Country\", t0.created_at AS \"Created\" FROM public.customers t0 WHERE t0.country_code = 'US' ORDER BY t0.created_at DESC LIMIT 100",
  "rows": [
    {
      "Customer ID": "cust-00001",
      "Customer Name": "Alice Corporation",
      "Email": "alice@example.com",
      "Country": "US",
      "Created": "2025-12-15T10:30:00Z"
    },
    {
      "Customer ID": "cust-00002",
      "Customer Name": "Bob Industries",
      "Email": "bob@example.com",
      "Country": "US",
      "Created": "2025-12-14T14:22:00Z"
    },
    {
      "Customer ID": "cust-00003",
      "Customer Name": "Carol Ventures",
      "Email": "carol@example.com",
      "Country": "US",
      "Created": "2025-12-13T09:15:00Z"
    }
  ],
  "count": 3
}
```

---

## Scenario 2: Strict Reporting - "Q4 Revenue by Region"

### 2a. Request (Strict Mode)

```bash
curl -X POST http://localhost:8080/api/llm/query \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: acme-corp" \
  -d '{
    "datasource": "sales",
    "prompt": "Q4 2025 total revenue by region",
    "mode": "strict"
  }'
```

### 2b. Planner Output

With `MODE: STRICT_REPORTING`, the planner is conservative:

```json
{
  "datasource": "sales",
  "version": null,
  "select": ["revenue", "region"],
  "filters": [
    {
      "field": "quarter",
      "op": "=",
      "value": "Q4"
    },
    {
      "field": "year",
      "op": "=",
      "value": 2025
    }
  ],
  "order_by": [
    {
      "field": "region",
      "direction": "asc"
    }
  ],
  "limit": 100
}
```

**Note**: Strict mode only includes explicit fields and filters. If the planner can't confidently map "total" and "by region", it omits aggregation.

### 2c. Executor Output

```sql
SELECT 
  t0.region AS "Region",
  t0.revenue AS "Revenue"
FROM public.sales t0
WHERE t0.quarter = 'Q4' AND t0.year = 2025
ORDER BY t0.region ASC
LIMIT 100
```

### 2d. Full Response

```json
{
  "datasource": "sales",
  "version": "v1",
  "semantic_sql": "{...}",
  "generated_sql": "SELECT...",
  "rows": [
    {"Region": "APAC", "Revenue": 2500000},
    {"Region": "EMEA", "Revenue": 1800000},
    {"Region": "LATAM", "Revenue": 900000},
    {"Region": "NORTH_AMERICA", "Revenue": 5200000}
  ],
  "count": 4
}
```

---

## Scenario 3: Debug Mode - Isolating Each Stage

### 3a. Test Planner Only

```bash
curl -X POST http://localhost:8080/api/llm/planner \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: acme-corp" \
  -d '{
    "datasource": "transactions",
    "prompt": "large transactions last month",
    "mode": "exploratory"
  }'
```

**Response** (just the semantic query):
```json
{
  "datasource": "transactions",
  "version": null,
  "select": ["id", "amount", "date", "status"],
  "filters": [
    {
      "field": "amount",
      "op": ">",
      "value": 10000
    },
    {
      "field": "date",
      "op": ">=",
      "value": "2026-01-05"
    }
  ],
  "order_by": [
    {
      "field": "amount",
      "direction": "desc"
    }
  ],
  "limit": 100
}
```

### 3b. Test Executor Only

```bash
curl -X POST http://localhost:8080/api/llm/executor \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: acme-corp" \
  -d '{
    "datasource": "transactions",
    "query": {
      "datasource": "transactions",
      "select": ["id", "amount", "date"],
      "filters": [
        {"field": "amount", "op": ">", "value": 10000}
      ],
      "order_by": [
        {"field": "amount", "direction": "desc"}
      ],
      "limit": 100
    }
  }'
```

**Response** (just the SQL):
```json
{
  "datasource": "transactions",
  "sql": "SELECT t0.transaction_id AS \"ID\", t0.transaction_amount AS \"Amount\", t0.transaction_date AS \"Date\" FROM public.transactions t0 WHERE t0.transaction_amount > 10000 ORDER BY t0.transaction_amount DESC LIMIT 100"
}
```

---

## Scenario 4: Error Cases

### 4a. Unknown Field Error

```bash
curl -X POST http://localhost:8080/api/llm/query \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: acme-corp" \
  -d '{
    "datasource": "customers",
    "prompt": "show me customer loyalty_points",
    "mode": "strict"
  }'
```

**Planner infers**:
```json
{
  "datasource": "customers",
  "select": ["id", "loyalty_points"],
  "filters": [],
  "order_by": [],
  "limit": 100
}
```

**Validation fails** (loyalty_points not in bundle):
```json
{
  "datasource": "customers",
  "error": "unknown select field: loyalty_points"
}
```

### 4b. Missing Tenant Header

```bash
curl -X POST http://localhost:8080/api/llm/query \
  -H "Content-Type: application/json" \
  -d '{
    "datasource": "customers",
    "prompt": "recent customers"
  }'
```

**Response**:
```json
HTTP/1.1 400 Bad Request

{
  "datasource": "customers",
  "error": "X-Tenant-ID header is required"
}
```

### 4c. LLM Not Configured

```bash
curl -X POST http://localhost:8080/api/llm/query \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: acme-corp" \
  -d '{
    "datasource": "customers",
    "prompt": "recent customers"
  }'
```

**Response** (if LLM integration not yet wired):
```json
{
  "datasource": "customers",
  "error": "LLM planner not yet configured; configure LLM_API_ENDPOINT and credentials"
}
```

---

## Scenario 5: Diagnostic Endpoints

### 5a. Get Available Prompts

```bash
curl -X GET http://localhost:8080/api/llm/prompts
```

**Response** (golden prompts and documentation):
```json
{
  "planner_prompt": {
    "exploratory": "You are a semantic query planner...",
    "strict": "MODE: STRICT_REPORTING...",
    "crud": "MODE: CRUD_VALIDATION..."
  },
  "executor_prompt": "You are an expert semantic SQL generator...",
  "documentation": "https://github.com/yourusername/semlayer/wiki/LLM-Prompts"
}
```

### 5b. Get Supported Modes

```bash
curl -X GET http://localhost:8080/api/llm/modes
```

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

## Performance Notes

### Latency Breakdown

For a typical query:
- **Planner LLM**: 500-2000ms (OpenAI, Claude; depends on model + network)
- **Validation**: 1-5ms
- **Executor LLM**: 300-1500ms (or <1ms if using deterministic Go resolver)
- **SQL Execution**: 10-500ms (depends on query complexity + DB load)
- **Marshaling**: 1-10ms

**Total end-to-end**: ~1-4 seconds (with LLM) or ~50-600ms (with Go resolver)

### Recommendations

- **Use deterministic Go resolver for executor** to cut latency in half
- **Cache semantic bundles** to avoid DB queries on every request
- **Implement request batching** for high-volume scenarios
- **Add LLM response caching** for identical prompts

---

## Testing Checklist

- [ ] Planner correctly interprets natural language
- [ ] Validator catches unknown fields
- [ ] Executor generates valid SQL
- [ ] Database returns correct rows
- [ ] Full pipeline latency is acceptable
- [ ] Error messages are clear
- [ ] Tenant isolation is enforced
- [ ] Rate limiting works

