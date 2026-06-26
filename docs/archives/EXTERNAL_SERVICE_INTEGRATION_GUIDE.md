# External Service Integration: Complete Workflow Guide

## Overview

This guide shows how to integrate external services (current prices, risk factors, market data, etc.) into your calculation engine.

---

## Example Scenario: IRR with Real-Time Market Prices

You want to calculate IRR **adjusted for current market prices** by fetching live stock prices from an external API.

### Workflow Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                      User Flow                                      │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  1. Navigate to Custom Components                                  │
│     Fabric → Custom Components                                     │
│     URL: http://localhost:5173/fabric/custom-components           │
│                  ↓                                                  │
│  2. Create API Integration Component                               │
│     Name: "Market Price Fetcher"                                   │
│     Type: API Integration                                          │
│     Endpoint: https://api.example.com/v1/prices                   │
│                  ↓                                                  │
│  3. Configure & Test Connection                                    │
│     Click "Test API" button                                        │
│     Verify successful connection                                   │
│                  ↓                                                  │
│  4. Use in Calculation                                             │
│     Fabric → Calculations Library                                  │
│     Edit calculation to reference the service                      │
│                  ↓                                                  │
│  5. Execute with External Data                                     │
│     Call /api/calc/run with external_services config               │
│     System fetches data → Transforms → Calculates                  │
│                  ↓                                                  │
│  6. View Results                                                   │
│     See calculated value with metadata showing external calls       │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## PART 1: Set Up External Service Integration (Custom Components)

### Step 1: Navigate to Custom Components

**URL:** `http://localhost:5173/fabric/custom-components`
**Menu:** Fabric → Custom Components

### Step 2: Add New API Integration Component

Click **"Add Component"** or **"Create New"** button

### Step 3: Fill in Configuration

#### Form Field: Name
```
Market Price Service Integration
```

#### Form Field: Type
```
Select: "API Integration" (green icon with database symbol)
```

#### Configuration Details Panel

**API Endpoint:**
```
https://api.marketdata.example.com/v1/prices
```

**Refresh Interval (seconds):**
```
300
```
(Refresh prices every 5 minutes)

**Additional Config (JSON):**
```json
{
  "method": "POST",
  "headers": {
    "Authorization": "Bearer YOUR_API_KEY_HERE",
    "Content-Type": "application/json"
  },
  "retryPolicy": {
    "maxRetries": 3,
    "backoffMs": 1000
  },
  "timeout": 10
}
```

### Step 4: Configure Events

Events allow you to trigger actions when data is fetched.

**Click:** "Events" tab

**Add Event:**
- Event Name: `onPricesFetched`
- Action: `custom`
- Custom Script:
```javascript
// Cache the prices in memory for calculations
window.MarketPrices = response.data;
console.log('Prices updated:', response.data);
// Emit event for other components to listen
window.emitEvent('prices-ready', response.data);
```

### Step 5: Configure Data Filters

Filters determine which data to fetch based on context.

**Click:** "Filters" tab

**Add Filter:**
- Field: `tickers`
- Operator: `in`
- Listen To Component: (leave empty for now)

### Step 6: Test the Connection

**Click:** "Test API" button at bottom of component config

#### Test Dialog Opens

Fill in test parameters:

**Test URL:**
```
https://api.marketdata.example.com/v1/prices
```

**Test Method:**
```
POST
```

**Test Headers:**
```json
{
  "Authorization": "Bearer YOUR_API_KEY_HERE",
  "Content-Type": "application/json"
}
```

**Test Body:**
```json
{
  "tickers": ["AAPL", "MSFT", "GOOGL"],
  "fields": ["price", "currency", "timestamp"]
}
```

**Click:** "Test Connection"

#### Expected Success Response

```json
{
  "status_code": 200,
  "status": "200 OK",
  "success": true,
  "headers": {
    "Content-Type": "application/json",
    "Cache-Control": "max-age=300"
  },
  "body": {
    "data": [
      {
        "ticker": "AAPL",
        "price": 228.45,
        "currency": "USD",
        "timestamp": "2024-11-02T14:30:00Z"
      },
      {
        "ticker": "MSFT",
        "price": 416.89,
        "currency": "USD",
        "timestamp": "2024-11-02T14:30:00Z"
      },
      {
        "ticker": "GOOGL",
        "price": 178.23,
        "currency": "USD",
        "timestamp": "2024-11-02T14:30:00Z"
      }
    ]
  }
}
```

### Step 7: Save Component

**Click:** "Save Component" button

**Confirmation:** Component saved successfully with ID: `comp-market-price-service`

---

## PART 2: Reference Service in Calculation

### Step 1: Navigate to Calculations

**URL:** `http://localhost:5173/fabric/calculations`
**Menu:** Fabric → Calculations Library

### Step 2: Find or Create Calculation

For this example: Click "Edit" on **"Investment XIRR"** card

Or create new:
- Name: `investment_xirr_with_prices`
- Title: `Investment XIRR with Market Prices`
- Category: `Performance`
- Subcategory: `IRR`

### Step 3: Update Calculation Configuration

**Original SQL:**
```sql
{{ xirr(ARRAY_AGG(${pre_agg_name}.cash_flow), 
        ARRAY_AGG(${pre_agg_name}.transaction_date)) }}
```

**Updated SQL with External Data Mapping:**
```sql
{{ xirr(
  ARRAY_AGG(
    ${investments}.cash_flow * 
    COALESCE(${market_prices}.current_price, 1.0)
  ), 
  ARRAY_AGG(${investments}.transaction_date)
) }}
```

**Add Configuration Fields:**

**External Services Section:** (New)
```json
{
  "external_services": [
    {
      "id": "market_prices",
      "name": "Market Price Fetcher",
      "component_id": "comp-market-price-service",
      "input_mapping": {
        "tickers": "investments.stock_ticker"
      },
      "output_mapping": {
        "current_price": "price",
        "ticker": "ticker"
      },
      "cache_ttl": 300
    }
  ]
}
```

### Step 4: Save Calculation

**Click:** "Save Changes"

Confirmation: Calculation updated with external service reference

---

## PART 3: Execute Calculation with External Services

### Option A: Use Test Button (Frontend)

1. Go to Calculations Library
2. Find your calculation
3. Click "Test" button
4. Enter sample investment data
5. System automatically:
   - Extracts tickers from your data
   - Calls Market Price Fetcher
   - Transforms data
   - Executes IRR calculation
   - Returns result

### Option B: Direct API Call (Backend)

```bash
curl -X POST http://localhost:8082/api/calc/run \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: tenant-123" \
  -H "X-Tenant-Datasource-ID: ds-456" \
  -d '{
    "financial": {
      "type": "xirr",
      "formula": "xirr(ARRAY_AGG(adjusted_cf), ARRAY_AGG(tx_date))"
    },
    "external_services": [
      {
        "name": "market_prices",
        "url": "https://api.marketdata.example.com/v1/prices",
        "method": "POST",
        "headers": {
          "Authorization": "Bearer YOUR_API_KEY_HERE",
          "Content-Type": "application/json"
        },
        "body": {
          "tickers": ["AAPL", "MSFT", "GOOGL"],
          "fields": ["price", "currency", "timestamp"]
        },
        "cache_ttl_seconds": 300,
        "timeout_seconds": 10,
        "retry_policy": {
          "max_retries": 3,
          "backoff_ms": 1000
        }
      }
    ],
    "data_context": {
      "portfolio_id": "port-789",
      "as_of_date": "2024-11-02"
    }
  }'
```

### Expected Response

```json
{
  "result": {
    "type": "percentage",
    "value": 0.1456,
    "display": "14.56%",
    "calculation_details": {
      "cash_flows_adjusted": [
        {"date": "2023-01-15", "original": 100, "price": 185.50, "adjusted": 18550},
        {"date": "2023-07-20", "original": -50, "price": 195.25, "adjusted": -9762.5},
        {"date": "2024-01-10", "original": 75, "price": 210.00, "adjusted": 15750},
        {"date": "2024-06-30", "original": -120, "price": 228.45, "adjusted": -27414}
      ]
    },
    "metadata": {
      "calculation_type": "xirr_with_prices",
      "calculation_time_ms": 1450,
      "row_count": 156,
      "external_services": [
        {
          "name": "market_prices",
          "status": "success",
          "response_time_ms": 342,
          "data_points_fetched": 3,
          "cache_status": "miss",
          "timestamp": "2024-11-02T14:35:22Z"
        }
      ],
      "transformations_applied": [
        "join_market_data",
        "apply_price_adjustment",
        "aggregate_cash_flows"
      ]
    }
  }
}
```

---

## PART 4: Common External Services Setup

### Example 1: Stock Price API (Alpha Vantage)

**Provider:** Alpha Vantage
**URL:** https://www.alphavantage.co/

**Custom Component Config:**
```json
{
  "name": "Alpha Vantage Stock Prices",
  "type": "api_integration",
  "config": {
    "apiEndpoint": "https://www.alphavantage.co/query",
    "method": "GET",
    "headers": {
      "User-Agent": "semlayer-calc-engine"
    },
    "queryParams": {
      "apikey": "YOUR_ALPHAVANTAGE_KEY",
      "function": "GLOBAL_QUOTE",
      "outputsize": "compact"
    },
    "refreshInterval": 300
  }
}
```

**Test Request:**
```bash
curl "https://www.alphavantage.co/query?function=GLOBAL_QUOTE&symbol=AAPL&apikey=YOUR_KEY"
```

### Example 2: Risk Factors Service (Internal)

**Provider:** Your Internal Risk Service
**URL:** https://risk-api.yourcompany.com/

**Custom Component Config:**
```json
{
  "name": "Internal Risk Factors",
  "type": "api_integration",
  "config": {
    "apiEndpoint": "https://risk-api.yourcompany.com/v1/factors",
    "method": "POST",
    "headers": {
      "Authorization": "Bearer internal-service-token",
      "Content-Type": "application/json"
    },
    "refreshInterval": 600,
    "cacheStrategy": "long"
  }
}
```

**Test Request:**
```bash
curl -X POST https://risk-api.yourcompany.com/v1/factors \
  -H "Authorization: Bearer internal-service-token" \
  -H "Content-Type: application/json" \
  -d '{"securities": ["AAPL", "MSFT"], "as_of_date": "2024-11-02"}'
```

### Example 3: Exchange Rate Service

**Provider:** Open Exchange Rates
**URL:** https://openexchangerates.org/

**Custom Component Config:**
```json
{
  "name": "Currency Exchange Rates",
  "type": "api_integration",
  "config": {
    "apiEndpoint": "https://openexchangerates.org/api/latest.json",
    "method": "GET",
    "headers": {},
    "queryParams": {
      "app_id": "YOUR_OPENEXCHANGERATES_ID",
      "base": "USD"
    },
    "refreshInterval": 3600
  }
}
```

---

## PART 5: Monitoring & Troubleshooting

### View External Service Call History

**Backend Logs:**
```bash
docker logs semlayer-backend-1 | grep "external_service"
```

### Check Cache Status

**API Endpoint to add:**
```
GET /api/calc/cache/status?service_name=market_prices
```

**Response:**
```json
{
  "service": "market_prices",
  "cache_size_bytes": 2048,
  "entries_count": 12,
  "ttl_remaining_seconds": 245,
  "hit_rate": 0.87,
  "last_refresh": "2024-11-02T14:35:22Z"
}
```

### Common Errors & Solutions

| Error | Cause | Solution |
|-------|-------|----------|
| `Connection timeout` | API unreachable | Check URL, network, firewall |
| `401 Unauthorized` | Invalid API key | Verify authorization header |
| `429 Too Many Requests` | Rate limit exceeded | Increase refresh interval, add caching |
| `500 Internal Server Error` | API error | Check external service logs |
| `Data mapping failed` | Wrong field names | Verify output_mapping configuration |
| `Calculation failed` | Invalid transformed data | Check data types and transformation logic |

### Enable Debug Logging

**Set in backend config:**
```yaml
logging:
  level: debug
  include_external_services: true
```

---

## PART 6: Performance Optimization

### Caching Strategy

```json
{
  "cache_ttl_seconds": 300,
  "cache_strategy": "short"  // Options: no-cache, short, long
}
```

### Parallel External Calls

The system automatically parallelizes multiple external service calls:

```bash
# Before (Sequential): ~1000ms
# Service A: 400ms
# Service B: 300ms
# Service C: 300ms
# Total: 1000ms

# After (Parallel): ~400ms
# Services A, B, C: 400ms (max)
```

### Batch Processing

For bulk calculations:

```bash
curl -X POST http://localhost:8082/api/calc/vectorized \
  -H "Content-Type: application/json" \
  -d '{
    "metrics": ["investment_xirr_with_prices", "nav_calc"],
    "entities": ["portfolio-1", "portfolio-2", "portfolio-3"],
    "external_services": [...]
  }'
```

---

## PART 7: Security Considerations

### API Key Management

**DO NOT** hardcode API keys in configuration. Use environment variables:

```bash
# .env file (not in git)
MARKET_API_KEY=sk_live_xxxxxxxxxxxxx
RISK_API_KEY=internal_token_xxxxxxxx
```

**Reference in component:**
```json
{
  "headers": {
    "Authorization": "Bearer ${MARKET_API_KEY}"
  }
}
```

### Network Security

1. **Use HTTPS only** - Never HTTP for external APIs
2. **Certificate validation** - Enable in production
3. **Rate limiting** - Implement on backend
4. **Data encryption** - Encrypt sensitive data in transit
5. **Access control** - Restrict to authorized tenants only

### Data Privacy

- Don't send sensitive data (PII, account numbers) to external services
- Cache responses appropriately
- Comply with data retention policies
- Audit external service access

---

## Summary: Full Flow in 5 Steps

| Step | Component | Action |
|------|-----------|--------|
| 1 | Custom Components | Create API Integration with external service |
| 2 | Custom Components | Test connectivity to external service |
| 3 | Calculations Library | Edit calculation to reference external service |
| 4 | Calculation Test | Run test with sample data |
| 5 | Backend API | Execute with live data, fetch external data, calculate result |

---

## Next Actions

Choose your path:

### Path A: Minimal Setup (30 mins)
✅ Set up custom component for stock prices
✅ Test basic connectivity
✅ Reference in one calculation

### Path B: Full Integration (4 hours)
✅ Set up multiple external services
✅ Configure caching layer
✅ Add error handling
✅ Performance optimization

### Path C: Advanced (1 week)
✅ Build dedicated risk factors service
✅ Implement real-time data sync
✅ Add webhooks for data updates
✅ Create monitoring dashboards

