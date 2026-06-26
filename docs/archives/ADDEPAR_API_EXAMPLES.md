# Addepar-Competitive Wealth Platform - GraphQL & API Examples

**Status**: Ready for Hasura Integration  
**Database**: wealth_app  
**Date**: October 29, 2025

---

## 📖 Table of Contents

1. [GraphQL Queries](#graphql-queries)
2. [GraphQL Mutations](#graphql-mutations)
3. [SQL Queries](#sql-queries)
4. [REST API Examples](#rest-api-examples)
5. [Real-Time Subscriptions](#real-time-subscriptions)

---

## 🔍 GraphQL Queries

After tracking tables in Hasura, use these queries:

### 1. Get Portfolio Holdings

```graphql
query GetPortfolioHoldings($portfolioId: uuid!) {
  v_entity_holdings(
    where: { 
      portfolio_entity_id: { _eq: $portfolioId }
      status: { _eq: ACTIVE }
    }
    order_by: { current_market_value: desc }
  ) {
    position_id
    portfolio_name
    holding_name
    ticker
    holding_type
    shares
    current_price
    current_market_value
    cost_basis
    unrealized_gain_loss
    return_pct
    as_of_date
  }
}

# Variables:
{
  "portfolioId": "df6b2386-095c-4bd1-9739-67296fb54f25"
}
```

**Response Example**:
```json
{
  "data": {
    "v_entity_holdings": [
      {
        "position_id": "pos-123",
        "portfolio_name": "Growth Portfolio 2025",
        "holding_name": "SPDR S&P 500 ETF",
        "ticker": "SPY",
        "holding_type": "ETF",
        "shares": 1000,
        "current_price": 330,
        "current_market_value": 330000,
        "cost_basis": 300000,
        "unrealized_gain_loss": 30000,
        "return_pct": 10.0,
        "as_of_date": "2025-10-29"
      },
      {
        "position_id": "pos-456",
        "portfolio_name": "Growth Portfolio 2025",
        "holding_name": "Apple Inc",
        "ticker": "AAPL",
        "holding_type": "STOCK",
        "shares": 500,
        "current_price": 204,
        "current_market_value": 102000,
        "cost_basis": 90000,
        "unrealized_gain_loss": 12000,
        "return_pct": 13.33,
        "as_of_date": "2025-10-29"
      }
    ]
  }
}
```

### 2. Get Portfolio Summary

```graphql
query GetPortfolioSummary($portfolioId: uuid!) {
  v_entity_portfolio_summary(
    where: { portfolio_entity_id: { _eq: $portfolioId } }
  ) {
    portfolio_name
    portfolio_type
    total_positions
    total_market_value
    total_cost_basis
    total_unrealized_gain_loss
    portfolio_return_pct
    as_of_date
  }
}

# Variables:
{
  "portfolioId": "df6b2386-095c-4bd1-9739-67296fb54f25"
}
```

**Response Example**:
```json
{
  "data": {
    "v_entity_portfolio_summary": [
      {
        "portfolio_name": "Growth Portfolio 2025",
        "portfolio_type": "PORTFOLIO",
        "total_positions": 5,
        "total_market_value": 710000,
        "total_cost_basis": 641000,
        "total_unrealized_gain_loss": 69000,
        "portfolio_return_pct": 10.76,
        "as_of_date": "2025-10-29"
      }
    ]
  }
}
```

### 3. Get Entity Details

```graphql
query GetEntity($entityId: uuid!) {
  entities_by_pk(id: $entityId) {
    id
    model_type
    display_name
    ticker
    cusip
    isin
    status
    ownership_type
    created_at
    updated_at
    
    # Relationships
    attributes {
      attributes
      valid_from
      valid_to
    }
    
    positions_as_owner {
      id
      shares
      market_value
      cost_basis
    }
    
    positions_as_owned {
      id
      owner_id
      shares
    }
    
    market_data {
      current_price
      day_change_pct
      as_of_date
    }
  }
}

# Variables:
{
  "entityId": "df6b2386-095c-4bd1-9739-67296fb54f25"
}
```

### 4. Get Transaction History

```graphql
query GetTransactionHistory($positionId: uuid!, $limit: Int = 50) {
  position_transactions(
    where: { position_id: { _eq: $positionId } }
    order_by: { trade_date: desc }
    limit: $limit
  ) {
    id
    transaction_type
    trade_date
    settlement_date
    units
    price
    amount
    fees
    net_amount
    is_short_term
    tax_lot_id
    notes
    created_at
    
    # Related data
    entity {
      display_name
      ticker
    }
  }
}

# Variables:
{
  "positionId": "pos-123",
  "limit": 50
}
```

### 5. Search Entities

```graphql
query SearchEntities($search: String!, $modelType: String, $tenant: uuid!) {
  entities(
    where: {
      _and: [
        { tenant_id: { _eq: $tenant } }
        { _or: [
            { display_name: { _ilike: $search } }
            { ticker: { _ilike: $search } }
            { cusip: { _eq: $search } }
          ]
        }
        { model_type: { _eq: $modelType } }
        { status: { _eq: ACTIVE } }
      ]
    }
    limit: 20
  ) {
    id
    model_type
    display_name
    ticker
    cusip
    ownership_type
    
    market_data(limit: 1, order_by: { as_of_date: desc }) {
      current_price
      day_change_pct
    }
  }
}

# Variables:
{
  "search": "%AAPL%",
  "modelType": "STOCK",
  "tenant": "your-tenant-id"
}
```

---

## ✏️ GraphQL Mutations

### 1. Create Entity

```graphql
mutation CreateEntity(
  $modelType: String!
  $tenantId: uuid!
  $displayName: String!
  $ticker: String
  $ownershipType: ownership_type_enum!
) {
  insert_entities_one(object: {
    model_type: $modelType
    tenant_id: $tenantId
    original_name: $displayName
    display_name: $displayName
    ticker: $ticker
    ownership_type: $ownershipType
    status: ACTIVE
  }) {
    id
    display_name
    model_type
    created_at
  }
}

# Variables:
{
  "modelType": "STOCK",
  "tenantId": "your-tenant-id",
  "displayName": "Nvidia Corp",
  "ticker": "NVDA",
  "ownershipType": "SHARE_BASED"
}
```

### 2. Create Position

```graphql
mutation CreatePosition(
  $ownerId: uuid!
  $ownedId: uuid!
  $shares: numeric!
  $costBasis: numeric!
  $marketValue: numeric!
  $tenantId: uuid!
) {
  insert_positions_one(object: {
    owner_id: $ownerId
    owned_id: $ownedId
    shares: $shares
    cost_basis: $costBasis
    market_value: $marketValue
    as_of_date: "2025-10-29"
    status: ACTIVE
    tenant_id: $tenantId
  }) {
    id
    owner_id
    owned_id
    shares
    market_value
    cost_basis
  }
}

# Variables:
{
  "ownerId": "df6b2386-095c-4bd1-9739-67296fb54f25",
  "ownedId": "security-id-nvda",
  "shares": 100,
  "costBasis": 25000,
  "marketValue": 27000,
  "tenantId": "your-tenant-id"
}
```

### 3. Record Transaction

```graphql
mutation RecordTransaction(
  $positionId: uuid!
  $entityId: uuid!
  $type: transaction_type_enum!
  $tradeDate: date!
  $units: numeric!
  $price: numeric!
  $amount: numeric!
  $fees: numeric!
  $tenantId: uuid!
) {
  insert_position_transactions_one(object: {
    position_id: $positionId
    entity_id: $entityId
    transaction_type: $type
    trade_date: $tradeDate
    units: $units
    price: $price
    amount: $amount
    fees: $fees
    net_amount: $amount
    tenant_id: $tenantId
  }) {
    id
    transaction_type
    trade_date
    amount
    created_at
  }
}

# Variables:
{
  "positionId": "pos-123",
  "entityId": "entity-nvda",
  "type": "BUY",
  "tradeDate": "2025-10-29",
  "units": 100,
  "price": 270,
  "amount": 27000,
  "fees": 15,
  "tenantId": "your-tenant-id"
}
```

### 4. Update Market Data

```graphql
mutation UpdateMarketData(
  $entityId: uuid!
  $currentPrice: numeric!
  $dayChange: numeric!
  $dayChangePercent: numeric!
  $asOfDate: date!
) {
  insert_entity_market_data_one(
    object: {
      entity_id: $entityId
      current_price: $currentPrice
      day_change: $dayChange
      day_change_pct: $dayChangePercent
      as_of_date: $asOfDate
      as_of_time: "now()"
      source: "bloomberg"
    }
    on_conflict: {
      constraint: entity_market_data_unique
      update_columns: [current_price, day_change, day_change_pct, as_of_time]
    }
  ) {
    id
    entity_id
    current_price
    as_of_date
  }
}

# Variables:
{
  "entityId": "security-id-aapl",
  "currentPrice": 210,
  "dayChange": 2.5,
  "dayChangePercent": 1.20,
  "asOfDate": "2025-10-29"
}
```

### 5. Close Position

```graphql
mutation ClosePosition($positionId: uuid!) {
  update_positions_by_pk(
    pk_columns: { id: $positionId }
    _set: {
      status: CLOSED
      closing_date: "2025-10-29"
      is_active: false
    }
  ) {
    id
    status
    closing_date
    is_active
  }
}

# Variables:
{
  "positionId": "pos-123"
}
```

---

## 🗄️ SQL Queries

### 1. Portfolio Performance Over Time

```sql
-- Compare portfolio performance across dates
WITH portfolio_dates AS (
  SELECT DISTINCT as_of_date 
  FROM v_entity_holdings 
  WHERE portfolio_entity_id = 'df6b2386-095c-4bd1-9739-67296fb54f25'
  ORDER BY as_of_date DESC
  LIMIT 12  -- Last 12 reporting dates
)
SELECT 
  h.as_of_date,
  s.total_positions,
  s.total_market_value,
  s.total_cost_basis,
  s.total_unrealized_gain_loss,
  s.portfolio_return_pct,
  LAG(s.total_market_value) OVER (ORDER BY h.as_of_date) as prev_value,
  (s.total_market_value - LAG(s.total_market_value) OVER (ORDER BY h.as_of_date)) as daily_change
FROM portfolio_dates pd
JOIN v_entity_holdings h ON h.as_of_date = pd.as_of_date
JOIN v_entity_portfolio_summary s ON s.as_of_date = pd.as_of_date 
  AND s.portfolio_entity_id = h.portfolio_entity_id
GROUP BY h.as_of_date, s.*
ORDER BY h.as_of_date DESC;
```

### 2. Sector Allocation

```sql
-- Calculate portfolio allocation by sector
SELECT 
  COALESCE(a.attributes->>'sector', 'Other') as sector,
  COUNT(*) as num_positions,
  SUM(h.current_market_value) as sector_value,
  (SUM(h.current_market_value) / (SELECT SUM(current_market_value) FROM v_entity_holdings WHERE portfolio_entity_id = 'portfolio-id')::numeric) * 100 as sector_pct
FROM v_entity_holdings h
LEFT JOIN entity_attributes a ON h.holding_entity_id = a.entity_id AND a.valid_to IS NULL
WHERE h.portfolio_entity_id = 'portfolio-id'
  AND h.as_of_date = CURRENT_DATE
GROUP BY sector
ORDER BY sector_value DESC;
```

### 3. Top Performers

```sql
-- Find holdings with highest returns
SELECT 
  h.holding_name,
  h.ticker,
  h.holding_type,
  h.shares,
  h.current_price,
  h.cost_basis,
  h.current_market_value,
  h.unrealized_gain_loss,
  h.return_pct,
  RANK() OVER (ORDER BY h.return_pct DESC) as performance_rank
FROM v_entity_holdings h
WHERE h.portfolio_entity_id = 'portfolio-id'
  AND h.as_of_date = CURRENT_DATE
ORDER BY h.return_pct DESC
LIMIT 10;
```

### 4. Tax Loss Harvesting Opportunities

```sql
-- Identify positions with losses for tax harvesting
SELECT 
  h.holding_name,
  h.ticker,
  h.shares,
  h.cost_basis,
  h.current_market_value,
  h.unrealized_gain_loss,
  h.return_pct
FROM v_entity_holdings h
WHERE h.portfolio_entity_id = 'portfolio-id'
  AND h.as_of_date = CURRENT_DATE
  AND h.unrealized_gain_loss < 0  -- Only losses
ORDER BY h.unrealized_gain_loss ASC;
```

### 5. Dividend Income

```sql
-- Calculate total dividend income
SELECT 
  SUM(amount) as total_dividends,
  COUNT(*) as num_dividends,
  AVG(amount) as avg_dividend,
  STRING_AGG(DISTINCT e.display_name, ', ') as holding_names
FROM position_transactions pt
JOIN entities e ON pt.entity_id = e.id
WHERE pt.transaction_type = 'DIVIDEND'
  AND pt.trade_date >= CURRENT_DATE - INTERVAL '1 year'
  AND pt.position_id IN (
    SELECT id FROM positions 
    WHERE owner_id = 'portfolio-id'
  );
```

---

## 🌐 REST API Examples

(When using Hasura with REST endpoints enabled)

### 1. Get Portfolio Holdings

```bash
curl -X POST https://your-hasura-instance/api/rest/portfolio-holdings \
  -H "Content-Type: application/json" \
  -H "X-Hasura-Admin-Secret: your-secret" \
  -d '{
    "portfolio_id": "df6b2386-095c-4bd1-9739-67296fb54f25"
  }'
```

### 2. Create Entity

```bash
curl -X POST https://your-hasura-instance/api/rest/create-entity \
  -H "Content-Type: application/json" \
  -H "X-Hasura-Admin-Secret: your-secret" \
  -d '{
    "model_type": "STOCK",
    "tenant_id": "your-tenant-id",
    "display_name": "New Company Inc",
    "ticker": "NCI",
    "ownership_type": "SHARE_BASED"
  }'
```

### 3. Update Market Price

```bash
curl -X PATCH https://your-hasura-instance/api/rest/update-market-price \
  -H "Content-Type: application/json" \
  -H "X-Hasura-Admin-Secret: your-secret" \
  -d '{
    "entity_id": "entity-id",
    "current_price": 210.50,
    "day_change_pct": 2.5,
    "as_of_date": "2025-10-29"
  }'
```

---

## 📡 Real-Time Subscriptions

### 1. Watch Portfolio Changes

```graphql
subscription OnPortfolioUpdate($portfolioId: uuid!) {
  v_entity_holdings(
    where: { portfolio_entity_id: { _eq: $portfolioId } }
    order_by: { current_market_value: desc }
  ) {
    position_id
    holding_name
    ticker
    current_market_value
    unrealized_gain_loss
    return_pct
    as_of_date
  }
}
```

### 2. Watch Entity Market Data

```graphql
subscription OnPriceUpdate($entityId: uuid!) {
  entity_market_data(
    where: { entity_id: { _eq: $entityId } }
    order_by: { as_of_time: desc }
    limit: 1
  ) {
    current_price
    day_change
    day_change_pct
    bid_price
    ask_price
    volume
    as_of_time
  }
}
```

### 3. Watch New Transactions

```graphql
subscription OnNewTransaction($positionId: uuid!) {
  position_transactions(
    where: { position_id: { _eq: $positionId } }
    order_by: { trade_date: desc }
  ) {
    id
    transaction_type
    trade_date
    units
    price
    amount
    created_at
  }
}
```

---

## 🎯 Client Code Examples

### JavaScript/React with Apollo Client

```javascript
import { useSubscription, useQuery, useMutation, gql } from '@apollo/client';

const GET_PORTFOLIO_HOLDINGS = gql`
  query GetPortfolioHoldings($portfolioId: uuid!) {
    v_entity_holdings(
      where: { portfolio_entity_id: { _eq: $portfolioId } }
    ) {
      holding_name
      ticker
      shares
      current_market_value
      unrealized_gain_loss
      return_pct
    }
  }
`;

export function PortfolioHoldings({ portfolioId }) {
  const { data, loading, error } = useQuery(GET_PORTFOLIO_HOLDINGS, {
    variables: { portfolioId }
  });
  
  if (loading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;
  
  return (
    <table>
      <thead>
        <tr>
          <th>Security</th>
          <th>Ticker</th>
          <th>Shares</th>
          <th>Value</th>
          <th>Gain/Loss</th>
          <th>Return %</th>
        </tr>
      </thead>
      <tbody>
        {data.v_entity_holdings.map(holding => (
          <tr key={holding.ticker}>
            <td>{holding.holding_name}</td>
            <td>{holding.ticker}</td>
            <td>{holding.shares}</td>
            <td>${holding.current_market_value}</td>
            <td>${holding.unrealized_gain_loss}</td>
            <td>{holding.return_pct}%</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
```

### Python with GQL

```python
from gql import gql, Client
from gql.transport.requests import RequestsHTTPTransport

client = Client(
    transport=RequestsHTTPTransport(
        url="https://your-hasura-instance/v1/graphql",
        headers={"X-Hasura-Admin-Secret": "your-secret"}
    )
)

query = gql("""
    query GetPortfolioHoldings($portfolioId: uuid!) {
        v_entity_holdings(
            where: { portfolio_entity_id: { _eq: $portfolioId } }
        ) {
            holding_name
            ticker
            current_market_value
            return_pct
        }
    }
""")

result = client.execute(query, variable_values={
    "portfolioId": "df6b2386-095c-4bd1-9739-67296fb54f25"
})

for holding in result['v_entity_holdings']:
    print(f"{holding['holding_name']}: {holding['return_pct']}%")
```

---

## 🧪 Testing Commands

### Verify Hasura Setup
```bash
curl https://your-hasura-instance/v1/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ __typename }"}'
```

### Test Query
```bash
curl https://your-hasura-instance/v1/graphql \
  -H "Content-Type: application/json" \
  -H "X-Hasura-Admin-Secret: your-secret" \
  -d '{
    "query": "{ entities_aggregate { aggregate { count } } }"
  }'
```

---

**Status**: ✅ Ready to Deploy  
**Last Updated**: October 29, 2025
