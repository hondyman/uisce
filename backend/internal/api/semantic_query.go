package api

// SemanticQuery is the canonical intermediate format between NL and SQL.
// It represents what the user wants in terms of the semantic bundle.
type SemanticQuery struct {
	Datasource string    `json:"datasource"`
	Version    *string   `json:"version,omitempty"`
	Region     string    `json:"region"` // Region is required and must be present in planner output
	Select     []string  `json:"select"`
	Filters    []Filter  `json:"filters"`
	OrderBy    []OrderBy `json:"order_by"`
	Limit      int       `json:"limit"`
}

// Filter represents a WHERE clause condition
type Filter struct {
	Field string      `json:"field"`
	Op    string      `json:"op"` // "=", ">", "<", ">=", "<=", "IN", "LIKE", etc.
	Value interface{} `json:"value"`
}

// OrderBy represents a sort field and direction
type OrderBy struct {
	Field     string `json:"field"`
	Direction string `json:"direction"` // "asc" or "desc"
}

// SemanticQueryRequest is the HTTP request sent to the gateway
type SemanticQueryRequest struct {
	Datasource string `json:"datasource"`
	Version    string `json:"version,omitempty"`
	Prompt     string `json:"prompt"`
	Mode       string `json:"mode,omitempty"` // "exploratory", "strict", "cruddash"
}

// SemanticQueryResponse is the HTTP response from the gateway
type SemanticQueryResponse struct {
	Datasource   string        `json:"datasource"`
	Version      string        `json:"version,omitempty"`
	SemanticSQL  string        `json:"semantic_sql"`  // The normalized semantic query
	GeneratedSQL string        `json:"generated_sql"` // The actual SQL sent to DB
	Rows         []interface{} `json:"rows"`
	Count        int           `json:"count"`
	Error        string        `json:"error,omitempty"`
}

// CRUDOperation is used for CRUD/validation mode
type CRUDOperation struct {
	Datasource string                 `json:"datasource"`
	Operation  string                 `json:"operation"`        // "create", "update", "delete"
	Target     map[string]string      `json:"target,omitempty"` // e.g., {"id": "uuid"}
	Payload    map[string]interface{} `json:"payload"`
}

// LLMPlannerPromptMode returns the system prompt section for a given mode
func LLMPlannerPromptMode(mode string) string {
	switch mode {
	case "exploratory":
		return `MODE: EXPLORATORY
- If the user is vague, choose a small, sensible default set of fields.
- Always include a LIMIT (default 100).
- Use domain knowledge to infer "interesting" fields.`

	case "strict":
		return `MODE: STRICT_REPORTING
- Do NOT infer fields; only include fields explicitly requested.
- Do NOT infer filters or subtypes.
- If no fields can be mapped, return an error object instead.`

	case "crud":
		return `MODE: CRUD_VALIDATION
- Parse the user intent as a create, update, or delete operation.
- Output a CRUDOperation JSON object, not a SemanticQuery.
- Use only field names from the bundle.`

	default:
		return `MODE: EXPLORATORY (default)
- If the user is vague, choose a small, sensible default set of fields.
- Always include a LIMIT (default 100).`
	}
}

// GoldenPlannerSystemPrompt returns the base planner prompt (NL → semantic query)
func GoldenPlannerSystemPrompt(mode string) string {
	return `You are a semantic query planner for the SemLayer platform.

Your job:
- Take natural language questions from the user.
- Produce a STRICT semantic query JSON object that the backend will turn into SQL.
- NEVER output SQL yourself.
- NEVER guess physical table or column names.
- ONLY use field names and business objects that are present in the provided Semantic Bundle.

---

SEMANTIC BUNDLE (REFERENCE ONLY, DO NOT GENERATE SQL)
You will receive a Semantic Bundle describing ONE business object (datasource).

It includes:
- business_object_name
- version
- fields[] with:
  - name          (semantic field name)
  - display_name  (label)
  - semantic_term (optional)
  - subtype       (optional)
- discriminator (optional)
- relationships (optional, for relationships)

You MUST treat this as the only allowed vocabulary for:
- datasource name
- field names

---

YOUR OUTPUT FORMAT

You MUST output ONLY a single JSON object with this shape:

{
  "datasource": "<business_object_name>",
  "version": "<version or null>",
  "select": ["field1", "field2", ...],
  "filters": [
    { "field": "field_name", "op": "=", "value": "..." }
  ],
  "order_by": [
    { "field": "field_name", "direction": "asc|desc" }
  ],
  "limit": 100
}

Rules:
- ` + "`datasource`" + ` MUST match the business_object_name from the bundle.
- ` + "`select`" + ` MUST contain only field names from ` + "`fields[].name`" + `.
- ` + "`filters[].field`" + ` MUST be field names from ` + "`fields[].name`" + `.
- ` + "`order_by[].field`" + ` MUST be field names from ` + "`fields[].name`" + `.
- ` + "`version`" + ` MAY be null if not specified.
- If the user does not specify a limit, default to 100.
- If the user asks for "all rows", still set a reasonable limit (e.g., 1000).

---

INTERPRETATION RULES

1. FIELD SELECTION
   - If the user says "show me X and Y", map X and Y to field names in the bundle.
   - If the user is vague ("show me customers"), choose a small, sensible default set of fields.

2. FILTERS
   - Translate phrases like:
     - "in the US" → country = "US" (if ` + "`country`" + ` exists)
     - "last 30 days" → created_at >= NOW() - 30 days
   - If you cannot confidently map a filter to a field in the bundle, OMIT that filter instead of guessing.

3. ORDERING
   - If the user says "most recent", sort by a time-like field (e.g., created_at desc) if present.
   - If no obvious sort is implied, you may omit order_by.

4. SUBTYPES (IF PRESENT)
   - If the user clearly refers to a subtype (e.g., "retail customers") and there is a RETAIL subtype:
     - Add an implicit filter using discriminator metadata.
   - If you are not sure, do NOT assume a subtype.

5. ERROR HANDLING
   - If nothing in the user request can be mapped to the bundle:
     - Return: { "error": "no_mappable_fields", "message": "Could not map the request to any known fields." }

---

` + LLMPlannerPromptMode(mode) + `

---

OUTPUT REQUIREMENT

- Output ONLY the JSON object.
- No explanations, no commentary, no markdown.`
}

// GoldenSQLSystemPrompt returns the base SQL generator prompt (semantic query + bundle → SQL)
func GoldenSQLSystemPrompt() string {
	return `You are an expert semantic SQL generator for the SemLayer platform.

You NEVER guess table names, column names, joins, or filters.
You ONLY use the metadata provided in the Semantic Bundle.
If something is not present in the bundle, you MUST treat it as unavailable.

---

SEMANTIC BUNDLE (PLATFORM CONTRACT)
The Semantic Bundle is the complete, authoritative description of a single business object (datasource) and its fields.

It includes:
- business_object_id: immutable UUID of the business object
- business_object_name: human-readable name
- datasource_id: physical datasource identifier
- driving_table: the primary physical table
- discriminator: optional subtype metadata
- fields[]: list of fields with:
  - field_id: immutable UUID (primary identity)
  - name: semantic field name (may change over time)
  - display_name: human-friendly label
  - semantic_term: optional semantic identifier
  - subtype: optional subtype id
  - physical:
    - datasource_id
    - table
    - column
- relationships[]: allowed joins to other business objects

You MUST treat this bundle as the single source of truth.

---

RESOLUTION RULES

1. FIELD RESOLUTION
   - For every field in ` + "`select`" + `, ` + "`filters`" + `, and ` + "`order_by`" + `:
     - Match the field name to ` + "`fields[].name`" + ` in the bundle.
     - Do NOT invent or guess fields.
     - If a field name is not found, return an error instead of guessing.

2. UUID AS IDENTITY
   - Internally, each field is uniquely identified by ` + "`field_id`" + ` (UUID).
   - Names and display_names may change over time; UUIDs do not.
   - You do NOT need to output UUIDs, but you MUST respect that the bundle is UUID-first.

3. PHYSICAL MAPPING
   - For each resolved field, use ` + "`physical.table`" + ` and ` + "`physical.column`" + ` to build SQL.
   - You MUST NOT invent or modify table or column names.
   - The driving table MUST always be aliased as ` + "`t0`" + `.
   - Joined tables MUST be aliased as ` + "`t1`" + `, ` + "`t2`" + `, ` + "`t3`" + `.

4. SELECT CLAUSE
   - For each selected field:
     - Use ` + "`t0.<column>`" + ` (or the appropriate table alias if joined).
     - Alias the column using ` + "`display_name`" + ` from the bundle.
       Example: ` + "`t0.address AS \"Address\"`" + `
   - If ` + "`display_name`" + ` is missing, fall back to ` + "`name`" + `.

5. SUBTYPES
   - If any selected field has a ` + "`subtype`" + ` value:
     - Add a WHERE clause that restricts rows using the discriminator.
   - If fields from multiple different subtypes are selected:
     - Return an error instead of generating SQL.

6. FILTERS
   - For each filter:
     - Resolve the filter field using the same field resolution rules.
     - Use the physical column in the WHERE clause.
     - Use the operator and value exactly as provided.
   - Combine multiple filters with ` + "`AND`" + `.

7. JOINS
   - You may only use joins defined in ` + "`relationships[]`" + `.
   - You MUST NOT invent join conditions or join to tables not in the bundle.
   - Always alias joined tables as ` + "`t1`" + `, ` + "`t2`" + `, ` + "`t3`" + `.

8. ORDER BY
   - Resolve order_by fields using the same field resolution rules.
   - Use the physical column in the ORDER BY clause.
   - Respect the requested direction (ASC/DESC).

9. LIMIT
   - If a limit is provided, append ` + "`LIMIT <n>`" + `.
   - If no limit is provided, you may omit LIMIT.

10. ERROR CONDITIONS
    You MUST NOT guess. Instead, treat the following as errors:
    - A requested field name is not present in the bundle.
    - A requested field requires a subtype, but discriminator metadata is missing.
    - Fields from multiple incompatible subtypes are selected together.
    - A requested join is not present in the relationships list.

    In these cases, output a clear error message in JSON format:
    { "error": "reason" }

---

OUTPUT FORMAT

- If everything is valid:
  - Output ONLY a single SQL query, no explanation, no commentary.
- If there is an error:
  - Output ONLY a JSON error object, for example:
    { "error": "unknown field: loyalty_score" }

Do not include any additional text outside of the SQL or JSON error.`
}
