# Semantic Term Property Inference - Examples & Reference

This document provides concrete examples of how the semantic term wizard intelligently infers properties based on column characteristics.

## Property Inference Quick Reference

The wizard automatically detects and sets the following properties:

| Property | Detection Logic | Example |
|----------|---|---|
| `data_type` | From column data type | "Dimension", "Measure", "Time" |
| `foreign_key` | Column ends with `_ID`, `ID`, or starts with `FK_` | `USER_ID` → true |
| `nullable` | ID/Key columns → false; others → true (temporal → false) | `EMAIL` → true |
| `temporal` | Contains `_DATE`, `_AT`, `TIMESTAMP`, `CREATED`, `UPDATED`, `DELETED` | `CREATED_AT` → true |
| `status_flag` | Ends with `_STATUS`, `_STATE`, `_FLAG` or contains `IS_`, `HAS_` | `IS_ACTIVE` → true |
| `cardinality` | Actual column cardinality from database analysis | 150000 |
| `frequent_values` | Most common values in column | ["1", "2", "3"] |
| `inferred_patterns` | Data patterns detected | ["numeric_id"] |
| `schema` | Source schema name | "public" |
| `table` | Source table name | "users" |
| `source_column` | Original column name | "USER_ID" |
| `sql` | Cube.js compatible SQL reference (backend only) | "{CUBE}.USER_ID" |

---

## Detailed Examples

### Example 1: User ID (Foreign Key)

**Column Definition**:
```sql
CREATE TABLE orders (
    ORDER_ID BIGINT PRIMARY KEY,
    USER_ID BIGINT NOT NULL,  -- <- This column
    ORDER_AMOUNT DECIMAL(10,2),
    CREATED_AT TIMESTAMP NOT NULL
);
```

**Column Properties Detected**:
```json
{
    "data_type": "Dimension",
    "foreign_key": true,
    "nullable": false,
    "schema": "public",
    "table": "orders",
    "source_column": "USER_ID",
    "sql": "{CUBE}.USER_ID"
}
```

**Why**:
- Ends with `_ID` → `foreign_key: true`
- Not nullable (column constraint) AND matches _ID pattern → `nullable: false`
- `data_type: "Dimension"` (numeric ID)

---

### Example 2: Customer Name (Regular Attribute)

**Column Definition**:
```sql
CREATE TABLE customers (
    CUSTOMER_ID BIGINT PRIMARY KEY,
    CUSTOMER_NAME VARCHAR(255),  -- <- This column
    EMAIL VARCHAR(255) UNIQUE,
    PHONE VARCHAR(20)
);
```

**Column Properties Detected**:
```json
{
    "data_type": "Dimension",
    "foreign_key": false,
    "nullable": true,
    "schema": "public",
    "table": "customers",
    "source_column": "CUSTOMER_NAME",
    "sql": "{CUBE}.CUSTOMER_NAME"
}
```

**Why**:
- No ID/FK patterns → `foreign_key: false`
- Not a key column → `nullable: true`
- VARCHAR → `data_type: "Dimension"`

---

### Example 3: Created At (Temporal Audit Field)

**Column Definition**:
```sql
CREATE TABLE events (
    EVENT_ID BIGINT PRIMARY KEY,
    EVENT_NAME VARCHAR(255),
    CREATED_AT TIMESTAMP,  -- <- This column
    UPDATED_AT TIMESTAMP,
    DELETED_AT TIMESTAMP NULL
);
```

**Column Properties Detected**:
```json
{
    "data_type": "Time",
    "foreign_key": false,
    "nullable": false,
    "temporal": true,
    "schema": "public",
    "table": "events",
    "source_column": "CREATED_AT",
    "sql": "{CUBE}.CREATED_AT"
}
```

**Why**:
- Contains `CREATED` pattern → `temporal: true`
- Temporal columns default to NOT nullable → `nullable: false`
- TIMESTAMP → `data_type: "Time"`

---

### Example 4: Order Status (Status Flag)

**Column Definition**:
```sql
CREATE TABLE orders (
    ORDER_ID BIGINT PRIMARY KEY,
    ORDER_STATUS VARCHAR(20),  -- <- This column
    IS_FULFILLED BOOLEAN DEFAULT FALSE,
    IS_SHIPPED BOOLEAN DEFAULT FALSE
);
```

**Column Properties Detected**:
```json
{
    "data_type": "Dimension",
    "foreign_key": false,
    "nullable": true,
    "status_flag": true,
    "schema": "public",
    "table": "orders",
    "source_column": "ORDER_STATUS",
    "sql": "{CUBE}.ORDER_STATUS"
}
```

**Why**:
- Ends with `_STATUS` → `status_flag: true`
- Not a key column → `nullable: true`
- VARCHAR → `data_type: "Dimension"`

---

### Example 5: Is Active (Boolean Flag)

**Column Definition**:
```sql
CREATE TABLE users (
    USER_ID BIGINT PRIMARY KEY,
    USERNAME VARCHAR(255) UNIQUE,
    IS_ACTIVE BOOLEAN DEFAULT TRUE,  -- <- This column
    IS_VERIFIED BOOLEAN DEFAULT FALSE
);
```

**Column Properties Detected**:
```json
{
    "data_type": "Dimension",
    "foreign_key": false,
    "nullable": true,
    "status_flag": true,
    "schema": "public",
    "table": "users",
    "source_column": "IS_ACTIVE",
    "sql": "{CUBE}.IS_ACTIVE"
}
```

**Why**:
- Contains `IS_` prefix → `status_flag: true`
- Boolean columns can be nullable → `nullable: true`
- Boolean → `data_type: "Dimension"`

---

### Example 6: Sales Amount (Measure)

**Column Definition**:
```sql
CREATE TABLE sales (
    SALE_ID BIGINT PRIMARY KEY,
    PRODUCT_ID BIGINT NOT NULL,
    SALES_AMOUNT DECIMAL(12,2),  -- <- This column
    QUANTITY INT,
    TRANSACTION_DATE DATE
);
```

**Column Properties Detected**:
```json
{
    "data_type": "Measure",
    "foreign_key": false,
    "nullable": true,
    "cardinality": 1500000,
    "schema": "public",
    "table": "sales",
    "source_column": "SALES_AMOUNT",
    "sql": "{CUBE}.SALES_AMOUNT"
}
```

**Why**:
- No FK patterns → `foreign_key: false`
- Can be nullable → `nullable: true`
- DECIMAL numeric type → `data_type: "Measure"`
- Large cardinality indicates measure

---

### Example 7: High-Cardinality Foreign Key

**Column Definition**:
```sql
CREATE TABLE transactions (
    TRANSACTION_ID BIGINT PRIMARY KEY,
    USER_ID BIGINT NOT NULL,  -- <- This column
    PRODUCT_ID BIGINT NOT NULL,
    TRANSACTION_DATE TIMESTAMP NOT NULL
);
```

**Column Properties Detected (with Cardinality)**:
```json
{
    "data_type": "Dimension",
    "foreign_key": true,
    "nullable": false,
    "cardinality": 150000,
    "frequent_values": ["1", "2", "5", "10", "3"],
    "inferred_patterns": ["numeric_sequential"],
    "schema": "public",
    "table": "transactions",
    "source_column": "USER_ID",
    "sql": "{CUBE}.USER_ID"
}
```

**Why**:
- Ends with `_ID` → `foreign_key: true`
- Not nullable → `nullable: false`
- Database analysis found 150,000 distinct values → `cardinality: 150000`
- Top 5 values extracted → `frequent_values`
- Pattern analysis detected sequence → `inferred_patterns`

---

## Detection Pattern Reference

### Foreign Key Detection
Triggers when column name matches ANY of:
- Ends with `_ID` → `user_id`, `product_id`, `fk_account_id`
- Ends with `ID` (exactly) → `id`, `ID`, `userId`
- Starts with `FK_` → `fk_user`, `FK_CUSTOMER_ID`
- Contains `_FK_` → `user_fk_mapping`, `product_fk_ref`

**Pattern Tests**:
```
USER_ID ✓         ORDER_ID ✓        CUSTOMER_FK_ID ✓
FK_USER ✓         ID ✓              FK_REGION_CODE ✓
USERID ✗          USER_NAME ✗       FK_ATTR ✓
```

### Temporal Field Detection
Triggers when column name matches ANY of:
- Ends with `_DATE` → `order_date`, `transaction_date`
- Ends with `_AT` → `created_at`, `updated_at`
- Ends with `_TIME` → `event_time`, `start_time`
- Contains `TIMESTAMP` → `created_timestamp`, `event_at_timestamp`
- Contains `CREATED` → `date_created`, `created_on`
- Contains `UPDATED` → `last_updated`, `updated_at`
- Contains `DELETED` → `deleted_at`, `soft_delete_date`

**Pattern Tests**:
```
CREATED_AT ✓      DATE_CREATED ✓     UPDATED_TIMESTAMP ✓
ORDER_DATE ✓      DELETED_AT ✓       END_TIME ✓
CREATION_DATE ✓   EFFECTIVE_DATE ✓   CREATED ✓
CREATED_BY ✗      DATE_MODIFIED ✗    DELETION_ID ✗
```

### Status/Flag Detection
Triggers when column name matches ANY of:
- Ends with `_STATUS` → `order_status`, `account_status`
- Ends with `_STATE` → `payment_state`, `job_state`
- Ends with `_FLAG` → `is_archived_flag`, `deleted_flag`
- Contains `IS_` → `is_active`, `is_verified`
- Contains `HAS_` → `has_permission`, `has_children`

**Pattern Tests**:
```
ORDER_STATUS ✓    IS_ACTIVE ✓        HAS_PERMISSION ✓
PAYMENT_STATE ✓   IS_VERIFIED ✓      DELETED_FLAG ✓
STATUS_CODE ✗     ACTIVE ✗           HAS_ID ✗
IS_ORDER ✗        HAS_VALUE ✗        STATUS_ID ✗
```

### Nullability Rules (in order of precedence)
1. **Temporal columns**: Always NOT nullable (temporal fields are system-maintained)
2. **Key columns**: NOT nullable if ends with `_ID`, `_KEY`, `PK_`, or exactly `ID`
3. **Everything else**: Nullable by default

---

## How the Wizard Uses These Properties

### 1. In the Database Catalog
Properties are stored as JSONB in the `catalog_node.properties` field:
```sql
INSERT INTO catalog_node (properties) VALUES ('{
  "data_type":"Dimension",
  "foreign_key":true,
  "nullable":false,
  "temporal":false,
  "status_flag":false,
  "schema":"public",
  "table":"users",
  "source_column":"USER_ID"
}'::jsonb);
```

### 2. In Semantic Term Creation
When ApplyEnrichment is called, it automatically populates semantic term properties:
```go
req := &ApplyEnrichmentRequest{
    Proposal: &EnrichmentProposal{
        SemanticTermName: "USER_ID",
        SemanticTermType: "Dimension",
        // ...
    },
    Column: &DatabaseColumn{
        Column: "USER_ID",
        Schema: "public",
        Table: "users",
        // ...
    },
}
// Properties are inferred: {foreign_key: true, nullable: false, ...}
```

### 3. In Cube.js Dimensions (Backend)
The SQL property enables proper Cube.js configuration:
```javascript
// Generated from properties["sql"] = "{CUBE}.USER_ID"
dimensions: {
  userId: {
    sql: `${CUBE}.USER_ID`,
    type: "number"
  }
}
```

---

## API Usage

### Create Semantic Term with Property Inference

**Request**:
```json
POST /api/semantic-mapping/enrich/apply
{
    "proposal": {
        "semantic_term_name": "USER_ID",
        "semantic_term_type": "Dimension",
        "business_term_name": "User Identifier",
        "domain_hierarchy": ["CRM", "User"],
        "confidence": 0.95
    },
    "column_id": "col-uuid-123",
    "tenant_id": "tenant-uuid",
    "datasource_id": "ds-uuid",
    "column": {
        "node_id": "col-node-uuid",
        "schema": "public",
        "table": "users",
        "column": "USER_ID",
        "cardinality": 150000
    }
}
```

**Response**:
```json
{
    "semantic_term_id": "term-uuid-123",
    "business_term_id": "biz-uuid-456"
}
```

**Stored Properties** (in catalog_node):
```json
{
    "data_type": "Dimension",
    "foreign_key": true,
    "nullable": false,
    "schema": "public",
    "table": "users",
    "source_column": "USER_ID",
    "cardinality": 150000,
    "sql": "{CUBE}.USER_ID"
}
```

### Auto-Enrichment with Threshold

**Request**:
```json
POST /api/semantic-mapping/enrich/auto
{
    "tenant_id": "tenant-uuid",
    "datasource_id": "ds-uuid",
    "threshold": 0.85
}
```

**Process**:
1. Fetches all columns in datasource
2. Calls SuggestEnrichment for each column
3. For columns with confidence ≥ 0.85:
   - Calls inferSemanticTermProperties (automatically)
   - Creates semantic term with full property set
4. Returns statistics

---

## Testing & Validation

### Running Tests
```bash
# Run property inference tests
go test ./internal/analytics -v -run TestInferSemanticTermProperties

# Run with cardinality
go test ./internal/analytics -v -run TestInferSemanticTermPropertiesCardinality

# Run all analytics tests
go test ./internal/analytics -v
```

### Test Coverage
- ✅ Foreign key columns (USER_ID, FK_USER, etc.)
- ✅ Regular attributes (CUSTOMER_NAME)
- ✅ Temporal fields (CREATED_AT, UPDATED_DATE)
- ✅ Status flags (ORDER_STATUS, IS_ACTIVE)
- ✅ Cardinality and data patterns
- ✅ Null column handling
- ✅ Primary key columns (ID, PK_ID)

---

## Future Enhancements

### Enhanced Label Generation
Could automatically generate human-readable labels:
```json
{
    "label": "User ID",  // From USER_ID
    "label_long": "Unique User Identifier",
    "order": 1
}
```

### Input Type Recommendations
Could suggest UI input types:
```json
{
    "input_type": "number",  // Numeric foreign key
    "input_format": "integer",
    "min_value": 1,
    "max_value": 999999
}
```

### LLM-Enhanced Descriptions
Could use LLM to generate business descriptions:
```json
{
    "description": "Unique identifier linking to the users table. Used to track which user placed the order.",
    "category": "identifier",
    "business_context": "CRM"
}
```

---

## Troubleshooting

### Property Not Detected?
Check if column name matches the detection patterns:

1. **Foreign Key Not Detected**
   - Ensure column name ends with `_ID`, `ID`, starts with `FK_`, or contains `_FK_`
   - Example: `user_key` won't match; use `user_id` instead

2. **Temporal Not Detected**
   - Use standard suffixes: `_DATE`, `_AT`, `_TIME`
   - Or include patterns: `CREATED`, `UPDATED`, `DELETED`
   - Example: `trans_time` won't match; use `transaction_at` or `transaction_time`

3. **Status Flag Not Detected**
   - Use standard suffixes: `_STATUS`, `_STATE`, `_FLAG`
   - Or use prefix patterns: `IS_`, `HAS_`
   - Example: `active` won't match; use `is_active` or `active_flag`

### Override Property Inference
If automatic detection is incorrect, manually set properties in the semantic term after creation:

```sql
UPDATE catalog_node 
SET properties = properties || '{"nullable": false}'::jsonb
WHERE id = 'term-uuid'
AND node_type_id = 'semantic-term-type-id';
```

