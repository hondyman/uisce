# Hasura Configuration Guide: Consolidated Metrics & DAX Functions

**Date:** November 3, 2025  
**Purpose:** Configure Hasura to expose consolidated `public` schema tables instead of domain-specific ones

---

## 🎯 Overview

This guide shows how to update Hasura metadata to use the consolidated metrics_registry and dax_functions tables from the public schema.

---

## 📋 Prerequisites

- Hasura v2+ installed
- `public.metrics_registry` and `public.dax_functions` tables created (from migration)
- Hasura CLI configured
- `HASURA_GRAPHQL_ADMIN_SECRET` set

---

## 🔧 Option 1: Hasura Metadata via YAML Files

### Step 1: Create Table Metadata

Create `hasura/metadata/databases/alpha/tables/public_metrics_registry.yaml`:

```yaml
table:
  schema: public
  name: metrics_registry

object_relationships: []

array_relationships:
  - name: by_schema_domain_functions
    using:
      manual_configuration:
        remote_table:
          schema: public
          name: dax_functions
        column_mapping:
          schema_domain: schema_domain

select_permissions:
  - role: user
    permission:
      columns:
        - id
        - node_id
        - schema_domain
        - category
        - description
        - formula_type
        - formula
        - arguments
        - badge
        - function_class
        - functions_used
        - governance_status
        - audience
        - tags
        - created_at
        - updated_at
      filter:
        schema_domain:
          _eq: X-Hasura-Domain
      allow_aggregations: true

  - role: admin
    permission:
      columns: "*"
      filter: {}
      allow_aggregations: true

insert_permissions:
  - role: admin
    permission:
      check:
        schema_domain:
          _eq: X-Hasura-Domain
      columns:
        - node_id
        - schema_domain
        - category
        - description
        - formula_type
        - formula
        - arguments
        - badge
        - function_class
        - functions_used
        - governance_status
        - audience
        - tags

update_permissions:
  - role: admin
    permission:
      columns:
        - category
        - description
        - governance_status
        - audience
        - tags
        - updated_at
      filter:
        schema_domain:
          _eq: X-Hasura-Domain
      check: null

delete_permissions:
  - role: admin
    permission:
      filter:
        schema_domain:
          _eq: X-Hasura-Domain

event_triggers: []
```

Create `hasura/metadata/databases/alpha/tables/public_dax_functions.yaml`:

```yaml
table:
  schema: public
  name: dax_functions

object_relationships: []

array_relationships: []

select_permissions:
  - role: user
    permission:
      columns:
        - id
        - name
        - schema_domain
        - class
        - badge
        - description
        - created_at
      filter:
        schema_domain:
          _eq: X-Hasura-Domain
      allow_aggregations: true

  - role: admin
    permission:
      columns: "*"
      filter: {}
      allow_aggregations: true

insert_permissions:
  - role: admin
    permission:
      check:
        schema_domain:
          _eq: X-Hasura-Domain
      columns:
        - name
        - schema_domain
        - class
        - badge
        - description

update_permissions:
  - role: admin
    permission:
      columns:
        - badge
        - description
      filter:
        schema_domain:
          _eq: X-Hasura-Domain
      check: null

delete_permissions:
  - role: admin
    permission:
      filter:
        schema_domain:
          _eq: X-Hasura-Domain

event_triggers: []
```

### Step 2: Deploy Metadata

```bash
# Navigate to your Hasura project directory
cd hasura

# Apply metadata
hasura metadata apply

# Check status
hasura metadata inconsistency status
```

---

## 🔧 Option 2: Hasura Metadata via CLI Commands

### Add Table

```bash
hasura metadata add table --source alpha public metrics_registry

hasura metadata add table --source alpha public dax_functions
```

### Create Relationships

```bash
# Relationship between metrics and dax functions (by schema_domain)
hasura metadata create-relationship \
  --source alpha \
  --table public_metrics_registry \
  --name by_domain_dax_functions \
  --remote-table public_dax_functions \
  --column schema_domain \
  --remote-column schema_domain
```

---

## 🔧 Option 3: GraphQL Queries via Console

### Step 1: Access Hasura Console

```bash
hasura console --skip-update-check
# Opens http://localhost:9695
```

### Step 2: Navigate to API Tab

1. Click **API** tab
2. In GraphQL Explorer, write queries:

```graphql
# Query metrics by domain
query GetMetricsByDomain($domain: String!) {
  public_metrics_registry(
    where: { schema_domain: { _eq: $domain } }
    order_by: { node_id: asc }
    limit: 100
  ) {
    id
    node_id
    schema_domain
    category
    description
    formula_type
    formula
    created_at
  }
}

# Query DAX functions by domain
query GetDAXFunctionsByDomain($domain: String!) {
  public_dax_functions(
    where: { schema_domain: { _eq: $domain } }
    order_by: { name: asc }
  ) {
    id
    name
    schema_domain
    class
    badge
    description
  }
}

# Query metrics across multiple domains
query GetMetricsMultipleDomains($domains: [String!]!) {
  public_metrics_registry(
    where: { schema_domain: { _in: $domains } }
    order_by: [{ schema_domain: asc }, { node_id: asc }]
  ) {
    node_id
    schema_domain
    category
    description
  }
}

# Aggregate queries
query MetricsStats($domain: String!) {
  public_metrics_registry_aggregate(
    where: { schema_domain: { _eq: $domain } }
  ) {
    aggregate {
      count
      group_by {
        category
        count: count
      }
    }
  }
}
```

---

## 🔐 Permissions & Row-Level Security

### User Role (Domain-Scoped)

```yaml
select_permissions:
  - role: user
    permission:
      columns:
        - id
        - node_id
        - schema_domain
        - category
        - description
        - formula_type
        - formula
        - created_at
      filter:
        schema_domain:
          _eq: X-Hasura-Domain  # Session variable
```

This uses the `X-Hasura-Domain` session variable to automatically filter results.

**When calling Hasura, include:**

```bash
curl -H "X-Hasura-Domain: banking" \
     -H "X-Hasura-Admin-Secret: $HASURA_GRAPHQL_ADMIN_SECRET" \
     -X POST \
     -d '{"query":"query { public_metrics_registry { node_id } }"}' \
     http://localhost:8080/v1/graphql
```

### Admin Role (Full Access)

```yaml
select_permissions:
  - role: admin
    permission:
      columns: "*"
      filter: {}  # No filter - sees all domains
```

---

## 🔄 Backward Compatibility Views (Optional)

If you still have code querying old domain schemas, create views:

```sql
-- Create view for backward compatibility
CREATE VIEW banking.metrics_registry AS
SELECT 
  id, node_id, category, description, formula_type, formula,
  arguments, badge, function_class, functions_used,
  governance_status, audience, tags, created_at, updated_at
FROM public.metrics_registry
WHERE schema_domain = 'banking';

CREATE VIEW banking.dax_functions AS
SELECT id, name, class, badge, description, created_at
FROM public.dax_functions
WHERE schema_domain = 'banking';

-- Repeat for each domain...
```

Then add these views to Hasura metadata:

```bash
hasura metadata add table --source alpha banking metrics_registry
hasura metadata add table --source alpha banking dax_functions
```

---

## 🧪 Testing Hasura Queries

### Test Query Execution

```bash
#!/bin/bash
# test_hasura_queries.sh

HASURA_URL="http://localhost:8080"
ADMIN_SECRET="your-admin-secret"
DOMAIN="banking"

# Test 1: Get metrics for domain
curl -X POST "$HASURA_URL/v1/graphql" \
  -H "Content-Type: application/json" \
  -H "X-Hasura-Admin-Secret: $ADMIN_SECRET" \
  -d '{
    "query": "query { public_metrics_registry(where: {schema_domain: {_eq: \"'$DOMAIN'\"}}) { node_id category } }"
  }'

# Test 2: Get DAX functions
curl -X POST "$HASURA_URL/v1/graphql" \
  -H "Content-Type: application/json" \
  -H "X-Hasura-Admin-Secret: $ADMIN_SECRET" \
  -d '{
    "query": "query { public_dax_functions(where: {schema_domain: {_eq: \"'$DOMAIN'\"}}) { name class } }"
  }'

# Test 3: Aggregate query
curl -X POST "$HASURA_URL/v1/graphql" \
  -H "Content-Type: application/json" \
  -H "X-Hasura-Admin-Secret: $ADMIN_SECRET" \
  -d '{
    "query": "query { public_metrics_registry_aggregate(where: {schema_domain: {_eq: \"'$DOMAIN'\"}}) { aggregate { count } } }"
  }'
```

---

## 📊 Index Configuration

Ensure Hasura can efficiently query with indexes:

```sql
-- Check that indexes exist
SELECT indexname FROM pg_indexes 
WHERE tablename = 'metrics_registry' AND schemaname = 'public';

-- Expected output:
-- idx_metrics_registry_schema_domain
-- idx_metrics_registry_node_id
-- idx_metrics_registry_category

-- If missing, create them:
CREATE INDEX idx_metrics_registry_schema_domain ON public.metrics_registry(schema_domain);
CREATE INDEX idx_metrics_registry_node_id ON public.metrics_registry(node_id);
CREATE INDEX idx_dax_functions_schema_domain ON public.dax_functions(schema_domain);
```

---

## 🔄 GraphQL Schema Stitching

### Custom Types

If you need to aggregate data, create custom types:

```yaml
# hasura/metadata/custom_types.yaml

objects:
  - name: MetricsAggregate
    fields:
      - name: domain
        type: String!
      - name: metric_count
        type: Int!
      - name: categories
        type: '[String!]!'
      - name: last_updated
        type: DateTime!

  - name: DAXFunctionSummary
    fields:
      - name: domain
        type: String!
      - name: function_count
        type: Int!
      - name: classes
        type: '[String!]!'
```

### Custom Query Resolvers

```yaml
# hasura/metadata/databases/alpha/query_collections/consolidated_queries.yaml

queries:
  - name: MetricsSummary
    definition:
      name: metrics_summary
      arguments:
        domain:
          type: String!
      return_type: MetricsAggregate!
```

---

## 🚀 Deployment

### Production Deployment Steps

```bash
# 1. Backup current metadata
hasura metadata export
git add .
git commit -m "Backup Hasura metadata before consolidation"

# 2. Update metadata files (as shown above)

# 3. Apply metadata to staging
export HASURA_GRAPHQL_ENDPOINT=http://staging-hasura:8080
hasura metadata apply

# 4. Test queries on staging
bash test_hasura_queries.sh

# 5. If successful, apply to production
export HASURA_GRAPHQL_ENDPOINT=http://prod-hasura:8080
hasura metadata apply

# 6. Verify production
bash test_hasura_queries.sh
```

### Rollback

```bash
# Revert to previous metadata
git checkout HEAD^ -- hasura/metadata/
hasura metadata apply
```

---

## 📈 Performance Tuning

### Enable Query Analysis

```bash
# Check slow queries
hasura metadata analyze

# View query execution plans
HASURA_LOG_LEVEL=debug hasura console --skip-update-check
```

### Optimize Queries

```graphql
# ✅ GOOD: Paginated with limit
query {
  public_metrics_registry(
    where: { schema_domain: { _eq: "banking" } }
    limit: 100
    offset: 0
    order_by: { created_at: desc }
  ) {
    node_id
    category
  }
}

# ❌ BAD: Fetching all at once
query {
  public_metrics_registry(
    where: { schema_domain: { _eq: "banking" } }
  ) {
    node_id
    category
  }
}
```

---

## 🔗 Integration Examples

### Frontend React/Apollo Client

```typescript
import { gql, useQuery } from '@apollo/client';

const GET_METRICS = gql`
  query GetMetrics($domain: String!) {
    public_metrics_registry(
      where: { schema_domain: { _eq: $domain } }
      order_by: { node_id: asc }
    ) {
      id
      node_id
      category
      description
    }
  }
`;

export function MetricsList({ domain }: { domain: string }) {
  const { data, loading, error } = useQuery(GET_METRICS, {
    variables: { domain },
    context: {
      headers: {
        'X-Hasura-Domain': domain,
      },
    },
  });

  if (loading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return (
    <ul>
      {data?.public_metrics_registry.map((metric: any) => (
        <li key={metric.id}>{metric.node_id}</li>
      ))}
    </ul>
  );
}
```

### Backend Express.js

```typescript
import { request } from 'graphql-request';

const query = gql`
  query GetMetrics($domain: String!) {
    public_metrics_registry(
      where: { schema_domain: { _eq: $domain } }
    ) {
      node_id
      category
    }
  }
`;

async function getMetrics(domain: string) {
  const data = await request(
    process.env.HASURA_GRAPHQL_ENDPOINT,
    query,
    { domain },
    {
      'X-Hasura-Admin-Secret': process.env.HASURA_GRAPHQL_ADMIN_SECRET,
    }
  );
  return data.public_metrics_registry;
}
```

---

## ✅ Verification Checklist

- [ ] Tables added to Hasura
- [ ] Relationships created
- [ ] Permissions configured
- [ ] GraphQL queries tested
- [ ] Pagination works
- [ ] Aggregations work
- [ ] Row-level security working
- [ ] Indexes in place
- [ ] Performance acceptable
- [ ] Frontend integration working
- [ ] Backend integration working
- [ ] Ready for production

---

## 📞 Troubleshooting

### Query Returns Empty

```bash
# Check if data exists in consolidated table
psql -h localhost -U postgres -d alpha << 'EOF'
SELECT COUNT(*) FROM public.metrics_registry WHERE schema_domain = 'banking';
EOF

# Check permissions
hasura metadata inconsistency status
```

### Permission Denied

```bash
# Verify role configuration
hasura metadata show --set databases.alpha.tables[].select_permissions

# Test with admin secret
curl -H "X-Hasura-Admin-Secret: $HASURA_SECRET" ...
```

### Slow Queries

```bash
# Analyze query performance
EXPLAIN ANALYZE SELECT * FROM public.metrics_registry WHERE schema_domain = 'banking';

# Verify index usage
SELECT * FROM pg_stat_user_indexes WHERE relname = 'metrics_registry';
```

---

## 📚 References

- [Hasura Documentation](https://hasura.io/docs/)
- [GraphQL Best Practices](https://hasura.io/docs/latest/graphql/core/guides/performance.html)
- [Row Level Security](https://hasura.io/docs/latest/graphql/core/auth/authorization/index.html)
