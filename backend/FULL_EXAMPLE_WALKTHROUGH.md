# SemLayer LLM Gateway - Full End-to-End Walkthrough

> Note: All runtime API requests are region-scoped and MUST include the `X-Tenant-Region` header (e.g. `X-Tenant-Region: eu-west`). This file provides examples; ensure you add the header to curl snippets and client SDKs.

## Scenario: Exploratory Analytics

You're a business analyst asking: **"Show me the 20 most recent retail customers in the US with their id, name, email, and loyalty points."**

---

## Step 1: User Sends Natural Language Query

```bash
curl -X POST http://localhost:8080/api/llm/query \
  -H "X-Tenant-ID: acme-corp" \
  -H "Content-Type: application/json" \
  -d '{
    "datasource": "customers",
    "version": "v1",
    "prompt": "Show me the 20 most recent retail customers in the US with their id, name, email, and loyalty points.",
    "mode": "exploratory"
  }'
```

---

## Step 2: Gateway Loads Semantic Bundle

The gateway queries the database and retrieves the complete semantic bundle for `customers`:

```json
{
  "business_object_id": "be7b9e37-5b9b-41fe-ac6e-58465387eb7c",
  "business_object_name": "customers",
  "datasource_id": "postgres-main",
  "driving_table": "customers",
  "version": "v1",
  "discriminator": {
    "column_name": "customer_type",
    "subtypes": [
      {
        "id": "RETAIL",
        "label": "Retail Customer",
        "discriminator_value": "RETAIL",
        "fields": ["loyalty_points"],
        "required_fields": ["loyalty_points"]
      }
    ]
  },
  "fields": [
    {
      "field_id": "uuid-id",
      "name": "id",
      "display_name": "Customer ID",
      "physical": { "table": "customers", "column": "id" }
    },
    {
      "field_id": "uuid-name",
      "name": "name",
      "display_name": "Customer Name",
      "physical": { "table": "customers", "column": "name" }
    },
    {
      "field_id": "uuid-email",
      "name": "email",
      "display_name": "Email Address",
      "physical": { "table": "customers", "column": "email" }
    },
    {
      "field_id": "uuid-country",
      "name": "country",
      "display_name": "Country",
      "physical": { "table": "customers", "column": "country" }
    },
    {
      "field_id": "uuid-created",
      "name": "created_at",
      "display_name": "Created At",
      "physical": { "table": "customers", "column": "created_at" }
    },
    {
      "field_id": "uuid-loyalty",
      "name": "loyalty_points",
      "display_name": "Loyalty Points",
      "subtype": "RETAIL",
      "physical": { "table": "customers", "column": "loyalty_points" }
    }
  ]
}
```

**Key observations:**
- Field `loyalty_points` has `"subtype": "RETAIL"` - only available for retail customers
- Discriminator column `customer_type` distinguishes subtypes
- All fields map to physical columns

---

## Step 3: Planner LLM Converts NL → Semantic Query

**System Prompt** (Exploratory Mode):
```
You are a semantic query planner for the SemLayer platform.
...
MODE: EXPLORATORY
- If the user is vague, choose a small, sensible default set of fields.
- Always include a LIMIT (default 100).
...
```

**Input to Gemini:**
```
SEMANTIC BUNDLE:
{bundle JSON above}

USER QUERY:
Show me the 20 most recent retail customers in the US with their id, name, email, and loyalty points.
```

**Gemini analyzes:**
- User explicitly asks for: `id`, `name`, `email`, `loyalty_points`
- User explicitly says: "retail customers" → infer subtype RETAIL
- User explicitly says: "in the US" → filter country = "US"
- User explicitly says: "20 most recent" → limit 20, order by created_at DESC
- Mode is exploratory → perform all inferences

**Gemini Returns** (Semantic Query JSON):
```json
{
  "datasource": "customers",
  "version": "v1",
  "select": [
    "id",
    "name",
    "email",
    "loyalty_points"
  ],
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
  "limit": 20
}
```

---

## Step 4: Gateway Validates Semantic Query

The validator checks:

- ✅ Datasource "customers" matches bundle
- ✅ Field "id" exists in bundle
- ✅ Field "name" exists in bundle
- ✅ Field "email" exists in bundle
- ✅ Field "loyalty_points" exists in bundle (and has subtype RETAIL)
- ✅ Filter field "country" exists in bundle
- ✅ Order-by field "created_at" exists in bundle
- ✅ Limit 20 is within bounds (1-100000)

**Result:** ✅ All validations pass

---

## Step 5: Executor Converts Semantic Query + Bundle → SQL

**System Prompt** (SQL Generator):
```
You are an expert semantic SQL generator for the SemLayer platform.
You NEVER guess table names, column names, joins, or filters.
You ONLY use the metadata provided in the Semantic Bundle.
...
RESOLUTION RULES:
1. Resolve field names to physical columns
2. Enforce subtype constraints automatically
3. Use display_name for column aliases
4. Apply filters in deterministic order
...
```

**Input to Gemini:**
```
SEMANTIC BUNDLE:
{bundle JSON}

SEMANTIC QUERY:
{
  "datasource": "customers",
  "select": ["id", "name", "email", "loyalty_points"],
  "filters": [{"field": "country", "op": "=", "value": "US"}],
  "order_by": [{"field": "created_at", "direction": "desc"}],
  "limit": 20
}
```

**Gemini resolves:**
- `id` field → column `id` with display name "Customer ID"
- `name` field → column `name` with display name "Customer Name"
- `email` field → column `email` with display name "Email Address"
- `loyalty_points` field → column `loyalty_points` with display name "Loyalty Points"
  - **BUT** this field has subtype RETAIL
  - Therefore, **enforce subtype filter**: `customer_type = 'RETAIL'`
- `country` filter → column `country` with operator `=` and value `'US'`
- `created_at` order → column `created_at` with direction DESC
- limit 20

**Gemini Returns** (SQL):
```sql
SELECT
  t0.id AS "Customer ID",
  t0.name AS "Customer Name",
  t0.email AS "Email Address",
  t0.loyalty_points AS "Loyalty Points"
FROM customers AS t0
WHERE
  t0.customer_type = 'RETAIL'
  AND t0.country = 'US'
ORDER BY t0.created_at DESC
LIMIT 20
```

**Key Points:**
- Subtype filter `customer_type = 'RETAIL'` added automatically because `loyalty_points` requires it
- User only said "US", but executor enforces subtype correctness
- Display names used for aliases (user-friendly column headers)
- All physical mappings resolved from bundle

---

## Step 6: Gateway Executes SQL Against Database

```sql
SELECT
  t0.id AS "Customer ID",
  t0.name AS "Customer Name",
  t0.email AS "Email Address",
  t0.loyalty_points AS "Loyalty Points"
FROM customers AS t0
WHERE
  t0.customer_type = 'RETAIL'
  AND t0.country = 'US'
ORDER BY t0.created_at DESC
LIMIT 20
```

Database returns 20 rows matching the criteria.

---

## Step 7: Gateway Returns Complete Response

```json
{
  "datasource": "customers",
  "version": "v1",
  "semantic_sql": "{\"datasource\":\"customers\",\"version\":\"v1\",\"select\":[\"id\",\"name\",\"email\",\"loyalty_points\"],\"filters\":[{\"field\":\"country\",\"op\":\"=\",\"value\":\"US\"}],\"order_by\":[{\"field\":\"created_at\",\"direction\":\"desc\"}],\"limit\":20}",
  "generated_sql": "SELECT t0.id AS \"Customer ID\", t0.name AS \"Customer Name\", t0.email AS \"Email Address\", t0.loyalty_points AS \"Loyalty Points\" FROM customers AS t0 WHERE t0.customer_type = 'RETAIL' AND t0.country = 'US' ORDER BY t0.created_at DESC LIMIT 20",
  "rows": [
    {
      "Customer ID": "cust-00901",
      "Customer Name": "Alice Carter",
      "Email Address": "alice.carter@example.com",
      "Loyalty Points": 3450
    },
    {
      "Customer ID": "cust-00887",
      "Customer Name": "Bob Martinez",
      "Email Address": "bob.martinez@example.com",
      "Loyalty Points": 2890
    },
    {
      "Customer ID": "cust-00875",
      "Customer Name": "Carol Davidson",
      "Email Address": "carol.d@example.com",
      "Loyalty Points": 2450
    },
    {
      "Customer ID": "cust-00863",
      "Customer Name": "Diego Rodriguez",
      "Email Address": "diego.r@example.com",
      "Loyalty Points": 1890
    },
    {
      "Customer ID": "cust-00851",
      "Customer Name": "Elena Sanchez",
      "Email Address": "elena.s@example.com",
      "Loyalty Points": 1670
    }
  ],
  "count": 5
}
```

---

## 🔍 What Just Happened

### The Platform Automatically:

1. **Loaded metadata** without user thinking about schema
2. **Validated** all field references before SQL generation
3. **Inferred subtype constraint** (RETAIL) from field selection
4. **Enforced brand correctness** (customer_type filter)
5. **Generated deterministic SQL** from bundle + semantic query
6. **Returned user-friendly column headers** (display names)
7. **Ensured multi-tenant isolation** (tenant ID from header)

### The Result:

- ✅ Natural language query → deterministic SQL → accurate results
- ✅ No guessing about schema
- ✅ No SQL injection vulnerabilities
- ✅ Reproducible across runs
- ✅ Auditable (see both semantic query and SQL)

---

## Comparison: Strict Reporting Mode

If the user had used `"mode": "strict_reporting"`:

```bash
curl -X POST http://localhost:8080/api/llm/query \
  -H "X-Tenant-ID: acme-corp" \
  -H "Content-Type: application/json" \
  -d '{
    "datasource": "customers",
    "prompt": "Give me id, name, email, loyalty_points for customers",
    "mode": "strict_reporting"
  }'
```

**Planner would return:**
```json
{
  "datasource": "customers",
  "select": ["id", "name", "email", "loyalty_points"],
  "filters": [],
  "order_by": [],
  "limit": 100
}
```

**No inferred filters or ordering!**

**Executor would still add:**
```sql
WHERE t0.customer_type = 'RETAIL'
```

Why? Because selecting `loyalty_points` requires enforcing the subtype for **correctness**, not user preference. This is the key distinction:

- **Planner (exploratory):** Infers user intent
- **Planner (strict):** No inference, takes what user says literally
- **Executor:** Enforces domain correctness (e.g., subtype constraints)

---

## The Architecture in Action

```
"Show me recent retail US customers..."
        ↓
    [Planner LLM + Exploratory Mode]
        ↓
  Semantic Query JSON (validated)
        ↓
    [Executor + Bundle]
        ↓
  SQL with subtypes enforced
        ↓
    [Database]
        ↓
  Results with friendly headers
```

Each stage is:
- ✅ Deterministic
- ✅ Validated
- ✅ Traceable
- ✅ Safe

---

## Running This Yourself

1. **Requirements:**
   - Backend running with Gemini configured
   - `GEMINI_API_KEY` environment variable set
   - Database with customers table + metadata

2. **Test the full pipeline:**
   ```bash
   curl -X POST http://localhost:8080/api/llm/query \
     -H "X-Tenant-ID: acme-corp" \
     -H "Content-Type: application/json" \
     -d '{
       "datasource": "customers",
       "version": "v1",
       "prompt": "Show me the 20 most recent retail customers in the US with their id, name, email, and loyalty points.",
       "mode": "exploratory"
     }'
   ```

3. **Inspect the pipeline stages:**
   - `/api/llm/planner` - See semantic query inference
   - `/api/llm/executor` - See SQL generation
   - `/api/llm/query` - Full end-to-end

4. **Debug if needed:**
   - Check `semantic_sql` field for planner output
   - Check `generated_sql` field for executor output
   - Look at `error` field for diagnostic messages

---

## Next: Try Strict Mode

Try the same query with `"mode": "strict_reporting"` and see how it differs!

