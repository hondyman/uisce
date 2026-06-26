# Calculation Engine: Testing, Extensions & External Service Integration

## Part 1: Testing Calculations Directly from Cards

### Current State
Currently, the Calculations Library page **does not have a direct "Test" or "Run" button** on the calculation cards. The available actions are:
- **Edit** - Modify calculation definition
- **View** - See calculation details

### Workaround for Testing Calculations

#### Option 1: Use the Backend API Directly
```bash
# Test IRR calculation with sample data
curl -X POST http://localhost:8082/api/calc/run \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: your-tenant-id" \
  -H "X-Tenant-Datasource-ID: your-datasource-id" \
  -d '{
    "financial": {
      "type": "xirr",
      "formula": "xirr(ARRAY_AGG(cash_flow), ARRAY_AGG(transaction_date))",
      "arguments": {
        "cash_flows": [100, -50, 75, -120],
        "dates": ["2023-01-01", "2023-06-01", "2024-01-01", "2024-06-01"]
      }
    }
  }'
```

#### Option 2: Add a "Test" Button (Enhancement Needed)
To enable direct testing from the card UI, we need to add a Test button to `CalculationsLibraryPage.tsx`:

**Location:** `frontend/src/features/fabric/pages/CalculationsLibraryPage.tsx`

**What to add:**
```tsx
// In CardActions, add alongside Edit button:
<Button
  size="small"
  startIcon={<PlayArrowIcon />}
  onClick={() => handleTestCalculation(calculation)}
  color="success"
>
  Test
</Button>

// Add handler function:
const handleTestCalculation = async (calculation: CalculationOption) => {
  // Open a test dialog with sample data input
  setTestDialogOpen(true);
  setSelectedCalculation(calculation);
};
```

---

## Part 2: Adding Custom Functions & External Service Integration

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Calculation Execution Flow                │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. Calculation Library (Frontend)                           │
│     └─ Edit calculation config                              │
│        └─ Add external service integration                  │
│                                                              │
│  2. Custom Components System (Fabric Menu)                   │
│     └─ Configure external APIs                             │
│        └─ Map data sources                                 │
│        └─ Test connectivity                                │
│                                                              │
│  3. Backend API Dispatcher                                  │
│     └─ /api/calc/run                                       │
│        └─ Fetch external data                             │
│        └─ Transform & cache                               │
│        └─ Execute calculation                             │
│        └─ Return results                                  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Solution: Custom Components for External Data Fetching

The system already has a **Custom Components** framework that supports API integrations:

**Location:** `Fabric → Custom Components` or `http://localhost:5173/fabric/custom-components`

#### Step 1: Create a Custom API Integration Component

1. Navigate to: **Fabric Menu → Custom Components**
2. Click **"Add Component"** button
3. Select **"API Integration"** type

#### Step 2: Configure External Service

Fill in the following configuration:

```json
{
  "name": "Current Price Fetcher",
  "type": "api_integration",
  "config": {
    "apiEndpoint": "https://api.example.com/v1/prices",
    "refreshInterval": 300,
    "width": "100%",
    "height": "400px",
    "method": "GET",
    "headers": {
      "Authorization": "Bearer YOUR_API_KEY",
      "Content-Type": "application/json"
    },
    "queryParams": {
      "ticker": "AAPL",
      "format": "json"
    }
  },
  "events": [
    {
      "eventName": "onDataFetched",
      "action": "custom",
      "customScript": "window.PriceCache = response.data; window.emitEvent('price-updated', response.data);"
    }
  ]
}
```

#### Step 3: Test Connectivity

The Custom Components page includes a **"Test API"** button:

```bash
# Backend endpoint that tests the connection
POST /api/custom-components/test-api
  ?tenant_id=your-tenant
  &datasource_id=your-datasource

Body:
{
  "url": "https://api.example.com/v1/prices",
  "method": "GET",
  "headers": {
    "Authorization": "Bearer YOUR_API_KEY"
  },
  "body": {}
}

Response:
{
  "status_code": 200,
  "status": "200 OK",
  "success": true,
  "body": "{...price data...}"
}
```

---

## Part 3: Integrating External Data into Calculations

### Approach 1: Pre-Aggregation with External Data

Create a pre-aggregation template that includes external data fetching:

```sql
-- Pre-aggregation template with external service integration
CREATE TABLE preagg_irr_with_prices AS
SELECT 
    investment_id,
    cash_flow,
    transaction_date,
    -- External API call via lateral join
    (
        SELECT DISTINCT price 
        FROM LATERAL fetch_current_price(stock_ticker) AS prices
        WHERE prices.effective_date <= transaction_date
        ORDER BY prices.effective_date DESC 
        LIMIT 1
    ) as historical_price,
    historical_price * quantity as position_value
FROM investments
WHERE transaction_date >= NOW() - INTERVAL '5 years';
```

### Approach 2: Custom Transformation Function

Define a custom function in the calculation configuration:

**In CalculationsLibraryPage.tsx, extend CalculationOption:**

```typescript
interface CalculationOption {
  // ... existing fields ...
  
  // NEW: External data transformations
  transformations?: Array<{
    name: string;
    type: 'api_fetch' | 'sql_transform' | 'custom_code';
    config: {
      // For api_fetch:
      endpoint?: string;
      method?: string;
      headers?: Record<string, string>;
      cacheStrategy?: 'no-cache' | 'short' | 'long';
      
      // For sql_transform:
      query?: string;
      
      // For custom_code:
      code?: string;
    };
    inputMapping?: Record<string, string>;
    outputMapping?: Record<string, string>;
  }>;
}
```

### Approach 3: Execute External Service During Calculation

Modify the backend `/api/calc/run` handler to support external service calls:

**In `backend/internal/api/api.go`:**

```go
type CalculationRequest struct {
    Financial FinancialCalc `json:"financial"`
    
    // NEW: External service integrations
    ExternalServices []ExternalServiceCall `json:"external_services,omitempty"`
    
    // NEW: Data context
    DataContext map[string]interface{} `json:"data_context,omitempty"`
}

type ExternalServiceCall struct {
    Name        string                 `json:"name"`
    URL         string                 `json:"url"`
    Method      string                 `json:"method"`
    Headers     map[string]string      `json:"headers"`
    Body        map[string]interface{} `json:"body"`
    CacheTTL    int                    `json:"cache_ttl_seconds,omitempty"` // 0 = no cache
    RetryPolicy RetryPolicy            `json:"retry_policy,omitempty"`
    Timeout     int                    `json:"timeout_seconds"` // default 30
}

type RetryPolicy struct {
    MaxRetries int `json:"max_retries"`
    BackoffMs  int `json:"backoff_ms"`
}
```

---

## Part 4: Practical Example: IRR with Current Market Prices

### Setup: Fetch current stock prices from external service

#### 1. Create Custom Component for Price Service

**UI Path:** Fabric → Custom Components → Add Component

```json
{
  "name": "Market Data Fetcher",
  "type": "api_integration",
  "config": {
    "apiEndpoint": "https://api.marketdata.example.com/v1/quotes",
    "refreshInterval": 60,
    "method": "POST",
    "headers": {
      "Authorization": "Bearer market-api-key",
      "Content-Type": "application/json"
    }
  }
}
```

#### 2. Update Calculation with External Data

**Edit Calculation:** Investment XIRR

**Original Formula:**
```
{{ xirr(ARRAY_AGG(${pre_agg_name}.cash_flow), 
        ARRAY_AGG(${pre_agg_name}.transaction_date)) }}
```

**Enhanced Formula with External Prices:**
```json
{
  "name": "investment_xirr_with_prices",
  "title": "Investment XIRR with Market Prices",
  "financial_calc": {
    "type": "xirr",
    "formula": "xirr(ARRAY_AGG(adjusted_cash_flow), ARRAY_AGG(transaction_date))"
  },
  "transformations": [
    {
      "name": "fetch_current_prices",
      "type": "api_fetch",
      "config": {
        "endpoint": "https://api.marketdata.example.com/v1/quotes",
        "method": "POST",
        "headers": {
          "Authorization": "Bearer market-api-key"
        },
        "cacheStrategy": "short"
      },
      "inputMapping": {
        "tickers": "investments.stock_ticker"
      },
      "outputMapping": {
        "current_price": "prices.price"
      }
    },
    {
      "name": "adjust_cash_flows",
      "type": "sql_transform",
      "query": "SELECT cash_flow * current_price as adjusted_cash_flow FROM investments JOIN prices USING (ticker)"
    }
  ]
}
```

#### 3. Execute Calculation with External Data

```bash
curl -X POST http://localhost:8082/api/calc/run \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: tenant-123" \
  -H "X-Tenant-Datasource-ID: ds-456" \
  -d '{
    "financial": {
      "type": "xirr",
      "formula": "xirr(ARRAY_AGG(adjusted_cash_flow), ARRAY_AGG(transaction_date))"
    },
    "external_services": [
      {
        "name": "market_prices",
        "url": "https://api.marketdata.example.com/v1/quotes",
        "method": "POST",
        "headers": {
          "Authorization": "Bearer market-api-key"
        },
        "body": {
          "tickers": ["AAPL", "MSFT", "GOOGL"]
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
      "portfolio_id": "portfolio-789",
      "as_of_date": "2024-11-02"
    }
  }'
```

#### 4. Expected Response

```json
{
  "result": {
    "type": "percentage",
    "value": 0.1345,
    "display": "13.45%",
    "metadata": {
      "calculation_type": "xirr_with_prices",
      "calculation_time_ms": 1250,
      "row_count": 156,
      "external_services_called": [
        {
          "name": "market_prices",
          "status": "success",
          "response_time_ms": 245,
          "data_points": 3,
          "cached": false
        }
      ],
      "transformations_applied": [
        "fetch_current_prices",
        "adjust_cash_flows"
      ]
    }
  }
}
```

---

## Part 5: Where to Set Up External Service Integrations

### Location 1: Custom Components (Main Entry Point)
- **URL:** `http://localhost:5173/fabric/custom-components`
- **Menu:** Fabric → Custom Components
- **Supported Types:**
  - ✅ API Integration (HTTP/HTTPS)
  - ✅ Web Component (Embedded)
  - ✅ Custom Code (JavaScript)
  - ✅ iFrame (External Apps)
  - ✅ Custom Widget (D3.js, Chart.js)

### Location 2: Calculation Configuration
- **URL:** `http://localhost:5173/fabric/calculations`
- **File:** `frontend/src/features/fabric/pages/CalculationsLibraryPage.tsx`
- **Edit Calculation** to add:
  - External service references
  - Transformation steps
  - Data mapping rules

### Location 3: Backend Services
- **File:** `backend/internal/api/api.go`
- **Function:** `runPreview()` at line 4618
- **Dispatcher:** `backend/internal/services/calculation_dispatcher.go`

### Location 4: Pre-Aggregation Templates
- **File:** `frontend/src/components/UnifiedSemanticBuilder/financialCalculations.ts`
- **Structure:** `preAggregationTemplate` field
- **For:** Caching external data before calculation

---

## Part 6: Implementation Roadmap

### Phase 1: Add Direct Test Button (Easy)
**Time:** 1-2 hours
**Changes:**
- Add "Test" button to calculation cards
- Create test data input dialog
- Call `/api/calc/run` with sample data

**Files:**
- `frontend/src/features/fabric/pages/CalculationsLibraryPage.tsx`

### Phase 2: Enhance External Service Support (Medium)
**Time:** 4-6 hours
**Changes:**
- Extend `CalculationOption` to support transformations
- Add UI for mapping external services
- Create service call orchestration

**Files:**
- `frontend/src/features/fabric/pages/CalculationsLibraryPage.tsx`
- `backend/internal/api/api.go`
- `backend/internal/services/calculation_dispatcher.go`

### Phase 3: Implement Caching Layer (Medium)
**Time:** 4-6 hours
**Changes:**
- Add Redis caching for external API calls
- Implement cache invalidation strategies
- Add cache TTL configuration

**Files:**
- `backend/internal/services/cache_manager.go` (new)
- `backend/internal/api/api.go`

### Phase 4: Add Risk Factors Service (Advanced)
**Time:** 6-8 hours
**Changes:**
- Create dedicated risk factors API
- Integrate with calculation engine
- Add real-time factor updates

**Files:**
- `backend/internal/handlers/risk_factors_handler.go` (new)
- `backend/internal/services/risk_factors_service.go` (new)
- `frontend/src/components/CustomComponentManager/` (update)

---

## Part 7: API Endpoints Reference

### Current Calculation Endpoints

```
POST /api/calc/run
  Execute a single calculation
  Input: FinancialCalc object
  Output: { result: {...} }

POST /api/calc/vectorized
  Execute batch calculations
  Input: { metrics: [...], entities: [...] }
  Output: { results: [...], batch_info: {...} }
```

### Custom Components Endpoints

```
GET  /api/custom-components
POST /api/custom-components
GET  /api/custom-components/{id}
PUT  /api/custom-components/{id}
DELETE /api/custom-components/{id}

POST /api/custom-components/test-api
  Test external API connectivity
  Input: { url, method, headers, body }
  Output: { status_code, headers, body, success }

GET  /api/custom-components/export
POST /api/custom-components/import
```

### New Endpoints (To Implement)

```
POST /api/calc/run-with-context
  Execute calculation with external services
  Input: CalculationRequest with ExternalServiceCall[]
  Output: { result: {...}, metadata: {...} }

GET  /api/calc/test
  Quick test endpoint for calculation validation
  Input: CalculationOption
  Output: { valid: boolean, errors: [...] }

POST /api/services/cache/invalidate
  Clear cached external service data
  Input: { service_name: string, ttl_seconds: number }
  Output: { invalidated: number, timestamp: ... }
```

---

## Summary Table

| Feature | Current State | Enhancement Needed | Implementation Level |
|---------|---------------|-------------------|----------------------|
| Test calculations from UI | ❌ No | ✅ Add Test button | Easy |
| External API integration | ✅ Custom Components | ✅ Link to calculations | Medium |
| Data transformation | ✅ SQL templates | ✅ Generalize framework | Medium |
| Caching | ⚠️ Partial | ✅ Full Redis layer | Medium |
| Risk factors service | ❌ No | ✅ Build dedicated API | Hard |
| Real-time data sync | ❌ No | ✅ WebSocket support | Hard |

