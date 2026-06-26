# Scenario Analysis Implementation Guide

## 📋 Overview

This document guides you through integrating the Scenario Analysis feature into your Figma design system and the production codebase. The feature consists of 4 main screens providing portfolio scenario analysis with AI-powered market intelligence.

---

## 🎨 Screens Overview

### Screen 1: Main Scenario Analysis Dashboard
**File**: `ScenarioAnalysisPro.tsx`
- Two-column layout (33% left / 67% right)
- Configuration panel for portfolio & scenario selection
- Real-time results display with gauges and comparisons
- Analysis history sidebar

### Screen 2: AI Scenario Proposal Modal
**File**: `AIScenarioProposal.tsx`
- Market snapshot with current data
- AI-generated scenario cards
- Confidence scores and impact indicators
- Detailed scenario information modal

### Screen 3: Scenario Details Sub-Modal
**Embedded in**: `AIScenarioProposal.tsx`
- Deep dive into individual scenario
- AI rationale and key drivers
- Projected impact metrics
- Supporting data with tabs

### Screen 4: Reusable Gauge Component
**File**: `Gauge.tsx`
- SVG-based circular gauge chart
- Color-coded performance indicators
- Configurable sizes and thresholds

---

## 🛠️ Figma Design Export Steps

### Step 1: Export Main Screen
1. Open the Scenario Analysis main screen design
2. Group by logical sections:
   - **Component Group 1**: Configuration Panel
   - **Component Group 2**: Results Display (Base Case)
   - **Component Group 3**: Results Display (Scenario Case)
   - **Component Group 4**: Comparison Analysis
3. Create component variants for:
   - Loading state (spinner)
   - Empty state (no selection)
   - Error state (API failure)
   - Result state (with data)

### Step 2: Export Color Tokens
Use Figma's color variable system:

```
Colors/
├── Primary
│   ├── Blue: #137fec
│   ├── Blue-hover: #0d5fb3
│   └── Blue-disabled: #7ea8d6
├── Semantic
│   ├── Success: #00875A
│   ├── Warning: #FFAB00
│   └── Danger: #DE350B
├── Neutral
│   ├── White: #ffffff
│   ├── Light-gray: #f6f7f8
│   ├── Dark-gray: #101922
│   └── Text-muted: #9dabb9
```

### Step 3: Export Typography Styles
Create Figma text styles:

```
Typography/
├── Heading-32: 32px Bold (700)
├── Heading-24: 24px Bold (700)
├── Heading-20: 20px Semibold (600)
├── Body-16: 16px Medium (500)
├── Body-14: 14px Normal (400)
├── Caption-12: 12px Normal (400)
├── Small-11: 11px Normal (400)
```

### Step 4: Export Components
1. **Button Component**
   - States: default, hover, active, disabled, loading
   - Variants: primary (blue), secondary (gray)

2. **Card Component**
   - With/without shadow
   - Light/dark theme variants

3. **Badge Component**
   - Variants: primary, success, warning, danger

4. **Input Components**
   - Select dropdown
   - Text input
   - Focused/error states

5. **Gauge Component**
   - Size variants: small, medium, large
   - Color variants: success, warning, danger

6. **Modal Component**
   - Header, body, footer sections
   - With overlay
   - Z-index management

---

## 📱 React Component Integration

### Step 1: Install Dependencies
```bash
npm install @apollo/client graphql
# Already included in your project
```

### Step 2: Import Components
```typescript
import ScenarioAnalysisPro from '@/components/ScenarioAnalysisPro'
import AIScenarioProposal from '@/components/AIScenarioProposal'
import Gauge from '@/components/Gauge'
```

### Step 3: Add Route
```typescript
// In your routing configuration
import { Routes, Route } from 'react-router-dom'

export function AppRoutes() {
  return (
    <Routes>
      {/* ...existing routes */}
      <Route path="/scenario-analysis" element={<ScenarioAnalysisPro />} />
    </Routes>
  )
}
```

### Step 4: Add Navigation Link
```typescript
// In your main navigation component
<nav>
  <Link to="/dashboard">Dashboard</Link>
  <Link to="/portfolios">Portfolios</Link>
  <Link to="/scenario-analysis">Scenario Analysis</Link> {/* NEW */}
  <Link to="/reporting">Reporting</Link>
</nav>
```

---

## 🔧 Backend Integration

### Step 1: Temporal Workflow Integration

Create/update: `backend/temporal/workflows/scenario_analysis.go`

```go
package workflows

import (
    "context"
    "go.temporal.io/sdk/workflow"
    "your-module/temporal/activities"
)

func ScenarioAnalysis(
    ctx workflow.Context, 
    portfolioID string, 
    scenario string,
) (map[string]any, error) {
    ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
        StartToCloseTimeout: 10 * time.Second,
        RetryPolicy: &temporal.RetryPolicy{
            MaximumAttempts: 3,
        },
    })

    // 1. Fetch portfolio data
    var portfolio map[string]any
    err := workflow.ExecuteActivity(ctx, activities.FetchPortfolio, portfolioID).Get(ctx, &portfolio)
    if err != nil {
        return nil, err
    }

    // 2. Run AI scenario projection using xAI
    var scenarioResult map[string]any
    err = workflow.ExecuteActivity(
        ctx, 
        activities.AIScenarioProject, 
        portfolioID, 
        scenario, 
        portfolio,
    ).Get(ctx, &scenarioResult)
    if err != nil {
        return nil, err
    }

    // 3. Calculate comparison metrics
    var comparison map[string]any
    err = workflow.ExecuteActivity(
        ctx, 
        activities.CalculateComparison, 
        portfolio, 
        scenarioResult,
    ).Get(ctx, &comparison)
    if err != nil {
        return nil, err
    }

    // 4. Store result
    workflow.ExecuteActivity(
        ctx, 
        activities.StoreAnalysisResult, 
        portfolioID, 
        scenario, 
        scenarioResult,
    )

    return map[string]any{
        "baseCase":      portfolio,
        "scenarioCase":  scenarioResult,
        "comparison":    comparison,
    }, nil
}
```

### Step 2: API Endpoint

Create: `backend/internal/api/scenario_analysis_routes.go`

```go
package api

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "go.temporal.io/sdk/client"
)

type ScenarioRequest struct {
    Scenario string `json:"scenario" binding:"required"`
}

func RegisterScenarioRoutes(r *gin.Engine, tc client.Client) {
    r.POST("/api/portfolio/:id/scenario", func(c *gin.Context) {
        portfolioID := c.Param("id")
        
        // Check ABAC authorization
        if !abac.Evaluate(c, "analyze", "portfolio") {
            c.JSON(403, nil)
            return
        }

        var req ScenarioRequest
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }

        // Execute Temporal workflow
        run, err := tc.ExecuteWorkflow(
            context.Background(),
            client.StartWorkflowOptions{
                TaskQueue: "scenario",
            },
            "ScenarioAnalysis",
            portfolioID,
            req.Scenario,
        )
        if err != nil {
            c.JSON(500, gin.H{"error": err.Error()})
            return
        }

        // Wait for result
        var result map[string]any
        err = run.Get(context.Background(), &result)
        if err != nil {
            c.JSON(500, gin.H{"error": err.Error()})
            return
        }

        c.JSON(200, result)
    })

    // AI Scenario Proposals endpoint
    r.GET("/api/ai/scenario-proposals", func(c *gin.Context) {
        portfolioID := c.Query("portfolio_id")
        
        // Fetch from xAI or cached proposals
        scenarios, err := fetchAIScenarios(portfolioID)
        if err != nil {
            c.JSON(500, gin.H{"error": err.Error()})
            return
        }

        marketData := getMarketData()

        c.JSON(200, gin.H{
            "scenarios":   scenarios,
            "marketData":  marketData,
        })
    })
}

func fetchAIScenarios(portfolioID string) ([]map[string]any, error) {
    // Call xAI API or fetch from database
    // Implementation depends on your AI integration
    return []map[string]any{}, nil
}

func getMarketData() map[string]any {
    // Fetch current market data from your data source
    return map[string]any{
        "sp500":                    4510.5,
        "sp500Change":              0.5,
        "vix":                      15.8,
        "vixChange":                -1.2,
        "treasuryYield":            4.25,
        "treasuryYieldChange":      0.02,
    }
}
```

### Step 3: Database Schema

Add to your database migrations:

```sql
CREATE TABLE scenario_analyses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    portfolio_id UUID NOT NULL REFERENCES portfolios(id),
    scenario_name VARCHAR(255) NOT NULL,
    base_case JSONB NOT NULL,
    scenario_case JSONB NOT NULL,
    comparison JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (portfolio_id) REFERENCES portfolios(id)
);

CREATE INDEX idx_scenario_portfolio ON scenario_analyses(portfolio_id);
CREATE INDEX idx_scenario_created ON scenario_analyses(created_at DESC);
```

---

## 📊 GraphQL Subscription

Update your GraphQL schema:

```graphql
type Portfolio {
    id: ID!
    aum: Float!
    sharpe: Float!
    risk: Float!
    status: String!
    assetAllocation: [AssetAllocation!]!
}

type AssetAllocation {
    asset: String!
    percentage: Float!
}

type ScenarioResult {
    baseCase: ScenarioCase!
    scenarioCase: ScenarioCase!
    comparison: ComparisonMetrics!
}

type ScenarioCase {
    aum: Float!
    sharpe: Float!
    risk: Float!
    status: String!
    assetAllocation: [AssetAllocation!]!
}

type ComparisonMetrics {
    aumDifference: Float!
    sharpeDifference: Float!
    riskDifference: Float!
}

extend type Subscription {
    portfolios: [Portfolio!]!
    scenarioAnalysis(portfolioId: ID!, scenario: String!): ScenarioResult!
}
```

---

## 🎯 Testing Checklist

### Unit Tests
- [ ] Portfolio selector correctly filters portfolios
- [ ] Scenario selector shows all predefined options
- [ ] Run Analysis button disabled until selections made
- [ ] Gauge component renders correct percentages
- [ ] Colors update based on thresholds

### Integration Tests
- [ ] Portfolio subscription fetches data correctly
- [ ] Scenario API call returns proper structure
- [ ] Results display in correct panels
- [ ] Analysis history persists in session
- [ ] Modal opens/closes properly

### E2E Tests
- [ ] User flow: Select portfolio → scenario → run → view results
- [ ] AI proposal modal: Opens, displays, selects scenario
- [ ] Dark mode: Styling applies correctly
- [ ] Responsive: Mobile, tablet, desktop layouts work
- [ ] Accessibility: Keyboard navigation, screen readers

### Performance Tests
- [ ] Initial load < 2s
- [ ] Analysis execution < 10s
- [ ] Modal opens instantly
- [ ] No memory leaks on navigate away

---

## 🚀 Deployment Checklist

Before going to production:

- [ ] All TypeScript errors resolved
- [ ] All accessibility warnings fixed
- [ ] Performance budget met (< 3MB gzip)
- [ ] Backend API endpoints tested
- [ ] Temporal workflows configured
- [ ] Database migrations applied
- [ ] Error boundaries implemented
- [ ] Loading states show for all async operations
- [ ] Error handling for failed API calls
- [ ] User feedback/notifications configured
- [ ] Analytics tracking added
- [ ] Documentation updated
- [ ] Team trained on feature

---

## 📚 File Structure

```
frontend/
├── src/
│   ├── components/
│   │   ├── ScenarioAnalysisPro.tsx       (Main screen)
│   │   ├── AIScenarioProposal.tsx        (Modal + sub-modal)
│   │   ├── Gauge.tsx                     (Reusable component)
│   │   └── styles/
│   │       └── scenarioAnalysis.css      (All styling)
│   ├── hooks/
│   │   └── useScenarioAnalysis.ts        (Custom logic)
│   └── pages/
│       └── ScenarioAnalysisPage.tsx      (Route wrapper)

backend/
├── internal/
│   ├── api/
│   │   └── scenario_analysis_routes.go   (API endpoints)
│   └── models/
│       └── scenario_analysis.go          (Data structures)
├── temporal/
│   ├── activities/
│   │   └── scenario_activities.go        (Activity implementations)
│   └── workflows/
│       └── scenario_analysis.go          (Workflow definition)
└── migrations/
    └── 20240101_scenario_analysis.sql    (DB schema)
```

---

## 🔐 Security Considerations

1. **ABAC Authorization**: All endpoints require tenant + action verification
2. **Data Validation**: Validate portfolio ID and scenario names
3. **Rate Limiting**: Limit scenario analysis requests per user/portfolio
4. **Input Sanitization**: Sanitize user inputs for custom scenarios
5. **Audit Logging**: Log all analysis requests for compliance
6. **Data Privacy**: Ensure portfolio data is tenant-scoped

---

## 📖 Usage Example

```typescript
import ScenarioAnalysisPro from '@/components/ScenarioAnalysisPro'

export default function DashboardPage() {
  return (
    <div>
      <h1>Dashboard</h1>
      <ScenarioAnalysisPro />
    </div>
  )
}
```

---

## 🆘 Troubleshooting

### Issue: API returns 403 Forbidden
- **Solution**: Verify ABAC policy allows "analyze" action on portfolio resource

### Issue: Gauges not rendering
- **Solution**: Ensure SVG styling is not blocked by CSP headers

### Issue: Loading spinner never stops
- **Solution**: Check Temporal workflow error logs

### Issue: Dark mode colors wrong
- **Solution**: Verify Tailwind dark mode config is enabled

---

## 📞 Support

For questions or issues:
1. Check the SCENARIO_ANALYSIS_FRONTEND_SPEC.md file
2. Review the visual reference HTML file
3. Check component prop types in TypeScript files
4. Review backend workflow implementation
5. Contact the development team

---

## ✅ Completion Checklist

- [ ] All components created and tested
- [ ] Backend endpoints implemented
- [ ] Temporal workflows configured
- [ ] Database schema applied
- [ ] GraphQL subscriptions updated
- [ ] Navigation links added
- [ ] Documentation complete
- [ ] Team trained
- [ ] Feature deployed to production

---

**Last Updated**: October 29, 2025
**Version**: 1.0.0
**Status**: Ready for Implementation
