# GraphQL Query Reference - Addepar Business Entities

## Quick Start

### Endpoint
- **URL:** `POST /graphql`
- **Playground:** `GET /graphql/playground`
- **Headers:** 
  - `X-Tenant-ID: {uuid}` (required)
  - `Content-Type: application/json`

### Example Request
```bash
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  -d '{"query":"query { entities(limit: 10) { id displayName } }"}'
```

---

## Query Examples

### 1. Get Single Entity

```graphql
query GetEntity($id: UUID!) {
  entity(id: $id) {
    id
    modelType
    displayName
    originalName
    ownershipType
    status
    isActive
    createdAt
    updatedAt
  }
}
```

**Variables:**
```json
{
  "id": "12345678-1234-1234-1234-123456789abc"
}
```

---

### 2. List All Entities with Filtering

```graphql
query ListEntities($modelType: String, $limit: Int = 20, $offset: Int = 0) {
  entities(modelType: $modelType, limit: $limit, offset: $offset) {
    id
    modelType
    displayName
    createdAt
  }
}
```

**Variables:**
```json
{
  "modelType": "STOCK",
  "limit": 50,
  "offset": 0
}
```

---

### 3. Get Complete Ownership Tree (Main Feature)

```graphql
query GetOwnershipTree($rootId: UUID!, $depth: Int = 3, $asOf: Date) {
  ownershipTree(rootId: $rootId, depth: $depth, asOf: $asOf) {
    entity {
      id
      modelType
      displayName
      ownershipType
    }
    position {
      id
      ownershipPercentage
      shares
      units
      marketValue
      inceptingDate
      closingDate
    }
    children {
      entity {
        id
        displayName
        modelType
      }
      position {
        ownershipPercentage
      }
      children {
        entity {
          id
          displayName
        }
        children {
          entity {
            id
            displayName
          }
        }
      }
    }
  }
}
```

**Variables:**
```json
{
  "rootId": "household-00000000-0000-0000-0000-000000000000",
  "depth": 3,
  "asOf": "2025-12-31"
}
```

**Response Example:**
```json
{
  "data": {
    "ownershipTree": {
      "entity": {
        "id": "household-uuid",
        "modelType": "HOUSEHOLD",
        "displayName": "Growth Portfolio 2025",
        "ownershipType": "PERCENT_BASED"
      },
      "children": [
        {
          "entity": {
            "id": "person-uuid",
            "modelType": "PERSON_NODE",
            "displayName": "Client A"
          },
          "position": {
            "ownershipPercentage": 100
          },
          "children": [
            {
              "entity": {
                "id": "account-uuid",
                "modelType": "FINANCIAL_ACCOUNT",
                "displayName": "Schwab IRA"
              },
              "position": {
                "ownershipPercentage": 100
              },
              "children": [
                {
                  "entity": {
                    "id": "stock-uuid",
                    "modelType": "STOCK",
                    "displayName": "AAPL"
                  },
                  "position": {
                    "ownershipPercentage": 50
                  }
                }
              ]
            }
          ]
        }
      ]
    }
  }
}
```

---

### 4. Get All Containers (Households, Accounts, etc.)

```graphql
query GetAllContainers {
  entities(
    modelType: "HOUSEHOLD"
    limit: 100
  ) {
    id
    displayName
    createdAt
  }
}
```

---

### 5. Get All Assets (Stocks, Bonds, etc.)

```graphql
query GetAllStocks {
  entities(
    modelType: "STOCK"
    limit: 1000
  ) {
    id
    displayName
    originalName
  }
}
```

---

### 6. Get Entity with Attributes

```graphql
query GetEntityWithAttributes($id: UUID!) {
  entity(id: $id) {
    id
    modelType
    displayName
    attributes {
      id
      key
      value
    }
  }
}
```

**Response Example:**
```json
{
  "entity": {
    "id": "stock-uuid",
    "modelType": "STOCK",
    "displayName": "AAPL",
    "attributes": [
      {
        "key": "ticker",
        "value": "AAPL"
      },
      {
        "key": "sector",
        "value": "Technology"
      },
      {
        "key": "market_cap",
        "value": 3000000000000
      }
    ]
  }
}
```

---

## Mutation Examples

### 1. Create New Entity

```graphql
mutation CreateStock($displayName: String!, $attributes: JSON!) {
  createEntity(
    modelType: "STOCK"
    displayName: $displayName
    attributes: $attributes
  ) {
    id
    modelType
    displayName
  }
}
```

**Variables:**
```json
{
  "displayName": "Apple Inc",
  "attributes": {
    "ticker": "AAPL",
    "sector": "Technology",
    "exchange": "NASDAQ"
  }
}
```

---

### 2. Create Ownership Position

```graphql
mutation CreateOwnership(
  $ownerID: UUID!
  $ownedID: UUID!
  $percentage: Float
) {
  createPosition(
    ownerID: $ownerID
    ownedID: $ownedID
    ownershipPercentage: $percentage
  ) {
    id
    ownershipPercentage
  }
}
```

**Variables:**
```json
{
  "ownerID": "household-uuid",
  "ownedID": "stock-uuid",
  "percentage": 50
}
```

---

### 3. Create Complete Portfolio Structure

```graphql
mutation CreatePortfolio(
  $householdName: String!
  $clientName: String!
) {
  step1: createEntity(
    modelType: "HOUSEHOLD"
    displayName: $householdName
    attributes: {}
  ) {
    id
  }
  
  step2: createEntity(
    modelType: "PERSON_NODE"
    displayName: $clientName
    attributes: {}
  ) {
    id
  }
  
  step3: createEntity(
    modelType: "FINANCIAL_ACCOUNT"
    displayName: "Primary Account"
    attributes: { "custodian": "Schwab" }
  ) {
    id
  }
}
```

---

## Advanced Queries

### 1. Portfolio Summary (All Holdings)

```graphql
query PortfolioSummary($householdId: UUID!) {
  ownershipTree(rootId: $householdId, depth: 5) {
    entity {
      id
      displayName
      modelType
    }
    children {
      entity { id displayName modelType }
      position { ownershipPercentage marketValue }
      children {
        entity { id displayName modelType }
        position { ownershipPercentage marketValue }
        children {
          entity { id displayName modelType }
          position { ownershipPercentage marketValue }
          children {
            entity { id displayName modelType }
            position { ownershipPercentage marketValue }
          }
        }
      }
    }
  }
}
```

---

### 2. Filter by Model Type and Date

```graphql
query HistoricalPortfolio($householdId: UUID!, $asOfDate: Date!) {
  ownershipTree(
    rootId: $householdId
    depth: 3
    asOf: $asOfDate
  ) {
    entity {
      id
      displayName
    }
    position {
      ownershipPercentage
      inceptingDate
      closingDate
    }
    children {
      entity { id displayName modelType }
      position { ownershipPercentage }
    }
  }
}
```

---

### 3. Flat List of All Holdings

```graphql
query AllHoldings {
  entities(modelType: "STOCK", limit: 1000) {
    id
    displayName
    originalName
  }
}
```

---

### 4. Hierarchical Path to Entity

```graphql
query EntityPath($entityId: UUID!) {
  entity(id: $entityId) {
    id
    displayName
    modelType
    createdAt
  }
}
```

Note: GraphQL doesn't have native parent traversal, but you can:
1. Get the household
2. Walk the tree until you find this entity
3. Or query the database directly for ancestors

---

## Common Use Cases

### Use Case 1: Display Portfolio Tree in UI

```graphql
query {
  ownershipTree(rootId: "household-uuid", depth: 2) {
    entity { id displayName modelType }
    children {
      entity { id displayName modelType }
      position { ownershipPercentage }
    }
  }
}
```

### Use Case 2: Get All Accounts for a Client

```graphql
query {
  entities(modelType: "FINANCIAL_ACCOUNT", limit: 100) {
    id
    displayName
  }
}
```

### Use Case 3: Calculate Portfolio Value

```graphql
query {
  ownershipTree(rootId: "household-uuid", depth: 5) {
    entity { displayName }
    position { marketValue }
    children {
      entity { displayName }
      position { marketValue }
    }
  }
}
```

(Client-side: sum all `marketValue` fields recursively)

### Use Case 4: Check Permissions on Entity

```graphql
query {
  entity(id: "entity-uuid") {
    id
    modelType
    displayName
  }
}
```

(If query fails with "forbidden", user lacks read permission)

### Use Case 5: Create Multi-Level Position

```graphql
mutation {
  # Create sleeve under household
  sleeve: createPosition(
    ownerID: "household-uuid"
    ownedID: "new-sleeve-uuid"
    ownershipPercentage: 80
  ) { id }
  
  # Add stocks to sleeve
  stock1: createPosition(
    ownerID: "new-sleeve-uuid"
    ownedID: "aapl-uuid"
    ownershipPercentage: 50
  ) { id }
  
  stock2: createPosition(
    ownerID: "new-sleeve-uuid"
    ownedID: "msft-uuid"
    ownershipPercentage: 50
  ) { id }
}
```

---

## Error Responses

### 401: Unauthorized
```json
{
  "errors": [
    {
      "message": "X-Tenant-ID header missing"
    }
  ]
}
```

### 403: Forbidden
```json
{
  "errors": [
    {
      "message": "forbidden: insufficient permissions to read entity"
    }
  ]
}
```

### 400: Invalid Input
```json
{
  "errors": [
    {
      "message": "hierarchy rule violated: STOCK cannot own BOND"
    }
  ]
}
```

### 500: Server Error
```json
{
  "errors": [
    {
      "message": "internal server error",
      "extensions": {
        "trace_id": "abc123"
      }
    }
  ]
}
```

---

## Performance Tips

### ✅ DO

- ✅ Limit depth: `ownershipTree(depth: 3)` not 10
- ✅ Use pagination: `limit: 50, offset: 0`
- ✅ Cache results: Trees don't change often
- ✅ Request only needed fields (less payload)
- ✅ Use aliases for multiple queries

### ❌ DON'T

- ❌ Fetch entire tree with depth 10
- ❌ Request all 1000+ fields
- ❌ Make sequential queries (use aliases)
- ❌ Re-fetch same data repeatedly

---

## Testing

### Test Single Query
```bash
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  -d '{"query":"{ entities(limit: 5) { id displayName } }"}'
```

### Test with Apollo Client (JavaScript)
```javascript
import ApolloClient from 'apollo-client';
import { InMemoryCache } from 'apollo-cache-inmemory';
import { HttpLink } from 'apollo-link-http';

const client = new ApolloClient({
  cache: new InMemoryCache(),
  link: new HttpLink({
    uri: 'http://localhost:8080/graphql',
    headers: {
      'X-Tenant-ID': '00000000-0000-0000-0000-000000000000'
    }
  })
});

client.query({
  query: gql`query { entities(limit: 10) { id displayName } }`
}).then(result => console.log(result));
```

### Test with Insomnia / Postman
1. Set URL: `http://localhost:8080/graphql`
2. Set Method: `POST`
3. Set Header: `X-Tenant-ID: 00000000-0000-0000-0000-000000000000`
4. Paste query in Body (GraphQL tab)
5. Send

---

## Additional Resources

- **Schema:** `/backend/internal/graphql/schema/addepar_ownership.graphqls`
- **Resolvers:** `/backend/internal/graphql/addepar_ownership_resolvers.go`
- **Integration Guide:** `GRAPHQL_WIRING_GUIDE.md`
- **Checklist:** `GRAPHQL_INTEGRATION_CHECKLIST.md`

---

**Status: 🟢 READY TO USE**

Start querying your portfolio hierarchy now!
