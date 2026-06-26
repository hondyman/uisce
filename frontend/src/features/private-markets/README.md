# Private Markets Explorer

A multi-audience financial analytics platform for private markets with tailored dashboards for Limited Partners (LPs), General Partners (GPs), and Fund-of-Funds (FoF) managers.

## Features

### 🎯 Multi-Audience Dashboards
- **LP Dashboard**: IRR curves, J-curves, multiples analysis, benchmarking, and liquidity tracking
- **GP Dashboard**: Deployment pacing, IRR/NAV tracking, fee analysis, value attribution, exit analysis, and benchmarking
- **FoF Dashboard**: Multi-fund portfolio analytics and meta-analysis capabilities

### 📊 Analytics Modules
- **Fund Selection**: Interactive fund picker with filtering and search
- **Performance Charts**: IRR curves, J-curves, and multiple overlays
- **Benchmarking**: Peer group comparisons and market indices
- **Liquidity Analysis**: Cash flow projections and liquidity stress testing
- **Deployment Tracking**: Capital deployment pacing and timing analysis

### 🔧 Governance & Templates
- **Template Review Dashboard**: Governance workflow for template approval and versioning
- **Registry-Ready Bundles**: JSON-based metric bundles for each audience
- **Version Control**: Template versioning with change tracking
- **Steward Oversight**: Dedicated governance interface for data stewards

### 🏗️ Architecture
- **Config-Driven**: Dynamic module loading based on user role and bundle configuration
- **Context Management**: Centralized state management for user, bundle, and role context
- **Modular Components**: Reusable analytics components across all dashboards
- **Type Safety**: Full TypeScript support with proper interfaces

## Usage

### Basic Setup
```tsx
import { ExplorerProvider, PrivateMarketsExplorer } from './features/private-markets';

function App() {
  return (
    <ExplorerProvider>
      <PrivateMarketsExplorer />
    </ExplorerProvider>
  );
}
```

### URL Parameters
Control user role and bundle via URL parameters:
- `?role=lp` - Limited Partner view
- `?role=gp` - General Partner view
- `?role=fof` - Fund of Funds view
- `?role=steward` - Governance/Steward view

### Bundle Configuration
Bundles are defined in JSON files:
- `lp_private_markets_bundle.json` - LP-specific metrics and modules
- `gp_private_markets_bundle.json` - GP-specific metrics and modules
- `fof_private_markets_bundle.json` - FoF-specific metrics and modules

## File Structure

```
frontend/src/features/private-markets/
├── PrivateMarketsExplorer.tsx      # Main explorer shell
├── LPPrivateMarketsDashboard.tsx   # LP dashboard
├── GPPrivateMarketsDashboard.tsx   # GP dashboard
├── TemplateReviewDashboard.tsx     # Governance dashboard
├── ExplorerContext.tsx             # Context provider
├── components/                     # Shared components
│   ├── FundSelector.tsx
│   ├── IRRCurveChart.tsx
│   ├── JCurvePlot.tsx
│   ├── MultipleOverlayPanel.tsx
│   ├── BenchmarkComparison.tsx
│   ├── LiquidityPanel.tsx
│   └── DeploymentPacingChart.tsx
├── bundles/                        # Configuration bundles
│   ├── lp_private_markets_bundle.json
│   ├── gp_private_markets_bundle.json
│   └── fof_private_markets_bundle.json
└── index.ts                        # Main exports
```

## API Integration

The explorer expects the following API endpoints:
- `GET /api/user/{id}` - User profile and role information
- `GET /api/bundles?audience={role}` - Available bundles for user role
- `GET /api/funds` - Fund data for analytics
- `GET /api/metrics/{fundId}` - Performance metrics for specific fund

## Development

### Adding New Analytics Modules
1. Create component in `components/` directory
2. Export from `index.ts`
3. Add to bundle configuration JSON
4. Import and use in appropriate dashboard

### Customizing Bundles
Edit the JSON bundle files to:
- Add/remove modules
- Configure module parameters
- Update governance settings
- Modify SLA requirements

### Extending User Roles
1. Update `User` interface in `ExplorerContext.tsx`
2. Add role handling in `PrivateMarketsExplorer.tsx`
3. Create role-specific dashboard component
4. Add corresponding bundle configuration

## Bundle Schema

```json
{
  "id": "bundle-id",
  "name": "Bundle Name",
  "audience": "lp|gp|fof",
  "version": "1.0.0",
  "modules": [
    {
      "id": "module-id",
      "name": "Module Name",
      "type": "chart|table|panel",
      "config": {}
    }
  ],
  "metrics": [
    {
      "id": "metric-id",
      "name": "Metric Name",
      "type": "percentage|currency|ratio",
      "formula": "calculation_formula"
    }
  ],
  "governance": {
    "status": "active|draft|deprecated",
    "steward_group": "data-stewards",
    "schema_hash": "hash_value",
    "sla": {
      "refresh_frequency": "daily",
      "max_latency": "4h"
    }
  }
}
```

## Navigation

Access the Private Markets Explorer at `/private-markets` in your application. The explorer will automatically:
1. Detect user role from URL parameters
2. Load appropriate bundle configuration
3. Render role-specific dashboard
4. Provide navigation between views (Dashboard, Analytics, Governance)

## Future Enhancements

- [ ] Advanced analytics with machine learning insights
- [ ] Real-time data streaming and alerts
- [ ] Custom dashboard builder for end users
- [ ] Integration with external data providers
- [ ] Mobile-responsive design optimization
- [ ] Multi-tenant bundle management
