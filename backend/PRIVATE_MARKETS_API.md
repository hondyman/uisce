# Private Markets Explorer - Backend API

## Overview
The backend has been extended with API endpoints to support the Private Markets Explorer frontend application. The implementation includes mock data for development and testing purposes.

## New API Endpoints

### User Management
- `GET /api/user/{id}` - Get user information by ID
  - Query parameters: `role` (lp, gp, fof, steward)
  - Returns user profile with role, organization, and permissions

### Bundle Management
- `GET /api/bundles` - Get available bundles for user audience
  - Query parameters: `audience` (lp, gp, fof)
  - Returns bundle configuration with modules, metrics, and governance

### Fund Data
- `GET /api/funds` - Get list of available private markets funds
  - Returns fund details including vintage, manager, strategy, geography

### Fund Metrics
- `GET /api/metrics/{fundId}` - Get performance metrics for specific fund
  - Returns TVPI, RVPI, IRR, XIRR, PME, and other key metrics

## Data Structures

### User
```go
type User struct {
    ID           string   `json:"id"`
    Name         string   `json:"name"`
    Role         string   `json:"role"` // lp, gp, fof, steward
    Organization string   `json:"organization"`
    Permissions  []string `json:"permissions"`
}
```

### Bundle
```go
type Bundle struct {
    ID       string            `json:"id"`
    Name     string            `json:"name"`
    Audience string            `json:"audience"`
    Version  string            `json:"version"`
    Modules  []BundleModule    `json:"modules"`
    Metrics  []BundleMetric    `json:"metrics"`
    Governance BundleGovernance `json:"governance"`
}
```

### Fund
```go
type Fund struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    Vintage   int       `json:"vintage"`
    Manager   string    `json:"manager"`
    Strategy  string    `json:"strategy"`
    Geography string    `json:"geography"`
    Status    string    `json:"status"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

### FundMetrics
```go
type FundMetrics struct {
    FundID        string    `json:"fund_id"`
    TVPI          float64   `json:"tvpi"`
    RVPI          float64   `json:"rvpi"`
    IRR           float64   `json:"irr"`
    XIRR          float64   `json:"xirr"`
    PME           float64   `json:"pme"`
    PaidInCapital float64   `json:"paid_in_capital"`
    Distributions float64   `json:"distributions"`
    ResidualValue float64   `json:"residual_value"`
    AsOfDate      time.Time `json:"as_of_date"`
}
```

## Bundle Configurations

### LP Bundle (Limited Partner)
- **Modules**: Fund Selector, IRR Curve Chart, J-Curve Plot, Benchmark Comparison, Liquidity Panel
- **Metrics**: TVPI, IRR, PME calculations
- **Governance**: Daily refresh, 4h max latency

### GP Bundle (General Partner)
- **Modules**: Deployment Pacing, IRR/NAV Tracking, Fee Analysis, Value Attribution, Exit Analysis
- **Metrics**: DPI, RVPI, TVPI calculations
- **Governance**: Weekly refresh, 24h max latency

### FoF Bundle (Fund of Funds)
- **Modules**: Portfolio Overview, Manager Performance, Allocation Analysis, Risk Attribution
- **Metrics**: Portfolio IRR, Diversification Score, Alpha calculations
- **Governance**: Monthly refresh, 48h max latency

## Sample Data

### Funds
1. **Tech Growth Fund III** (2020) - Venture Capital, North America
2. **Infrastructure Partners II** (2019) - Infrastructure, Europe
3. **Real Estate Fund IV** (2021) - Real Estate, Asia Pacific
4. **Healthcare Innovation Fund** (2022) - Healthcare, Global

### Metrics (Fund-specific)
Each fund has customized performance metrics:
- Tech Fund: TVPI 1.85, IRR 15.6%, PME 1.12
- Infrastructure: TVPI 1.65, IRR 12.3%, PME 1.08
- Real Estate: TVPI 1.92, IRR 14.5%, PME 1.15
- Healthcare: TVPI 2.05, IRR 17.8%, PME 1.22

## Usage Examples

### Get User with LP Role
```bash
GET /api/user/current?role=lp
```

### Get GP Bundles
```bash
GET /api/bundles?audience=gp
```

### Get Fund List
```bash
GET /api/funds
```

### Get Fund Metrics
```bash
GET /api/metrics/fund-1
```

## Integration with Frontend

The backend endpoints are designed to work seamlessly with the Private Markets Explorer frontend:

1. **ExplorerContext** uses `/api/user/{id}` to initialize user state
2. **Bundle loading** uses `/api/bundles` to get configuration
3. **Fund selection** uses `/api/funds` to populate dropdowns
4. **Analytics modules** use `/api/metrics/{fundId}` for data

## Future Enhancements

- Database integration for persistent data
- Authentication and authorization
- Real-time data streaming
- Advanced filtering and search
- Historical data and time-series analysis
- Custom bundle creation and management

## Testing

To test the API endpoints:

1. Start the backend server: `go run ./cmd/server`
2. Use curl or a REST client to test endpoints
3. Frontend integration available at `/private-markets`

The implementation provides a solid foundation for the Private Markets Explorer with proper API contracts and mock data for development.
