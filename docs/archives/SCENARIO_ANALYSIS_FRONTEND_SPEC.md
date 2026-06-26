# Scenario Analysis Feature - Complete Frontend Implementation

## Overview
This document provides comprehensive details for the Scenario Analysis Feature frontend implementation with four main screens, complete with design specifications, component architecture, and integration points.

---

## 1. Main Scenario Analysis Screen (`ScenarioAnalysisPro.tsx`)

### Purpose
Primary interface for portfolio scenario analysis with side-by-side configuration and results panels.

### Layout
- **Left Panel (33% width)**: Configuration Panel
  - Portfolio selector dropdown
  - Scenario selector with predefined options + custom/AI proposal options
  - "Run Analysis" button with loading state
  - Analysis History with clickable items

- **Right Panel (67% width)**: Results Display
  - Dynamic title showing selected scenario
  - Two-column grid for Base Case and Scenario Case cards
  - Comparison Analysis section below

### Components

#### Configuration Panel
```typescript
- Title: "Scenario Analysis"
- Subtitle: "Analyze portfolio performance under various market conditions"
- Portfolio Selector
  - Label: "Select Portfolio"
  - Format: "ID | $AUM M"
  - Events: onChange triggers portfolio selection
  
- Scenario Selector
  - Predefined options:
    * Market Crash (-20%)
    * Interest Rate Hike (+2%)
    * High Inflation (+5%)
    * Tech Bubble Burst (-30% on tech stocks)
    * Geopolitical Crisis
  - Special options:
    * Create Custom Scenario...
    * AI Scenario Proposal...
  
- Run Analysis Button
  - Disabled until both portfolio and scenario selected
  - Shows "Running Analysis..." during loading
  - Triggers POST to /api/portfolio/:id/scenario
  
- Analysis History
  - Scrollable list of previous analyses
  - Format: Scenario name + timestamp
  - Clicking loads previous result
  - Highlight active selection
```

#### Results Display - Base Case Card
```typescript
Status Badge: "Optimized" (green)
Metrics:
  - Current AUM: Large bold text
  - Sharpe Ratio: Gauge chart (green-yellow scale, max 3.0)
  - Risk Score: Gauge chart (yellow-red scale, max 100)
Asset Allocation:
  - Horizontal progress bars
  - Show % for each asset class
  - Format: [Asset] [=====>] 35%
```

#### Results Display - Scenario Case Card
```typescript
Status Badge: Dynamic ("At Risk" | "Strong") based on AUM change
Metrics:
  - Projected AUM: Large bold text with % change indicator
  - Sharpe Ratio: Gauge + delta (e.g., -1.0)
  - Risk Score: Gauge + delta (e.g., +37)
Asset Allocation:
  - Same format as Base Case
  - Different color scheme (red bars for risk)
```

#### Comparison Analysis Card
```typescript
Three metric columns:
  - AUM Change: dollar amount + percentage
  - Sharpe Change: delta value
  - Risk Change: delta value
Color coding:
  - Negative = red (#DE350B)
  - Positive = green (#00875A)
```

### Data Flow
```
User selects portfolio + scenario
    ↓
Click "Run Analysis"
    ↓
POST /api/portfolio/:id/scenario { scenario: "..." }
    ↓
Receive AnalysisResult object
    ↓
Update analysisResult state
    ↓
Add to analysisHistory
    ↓
Render results in right panel
```

### State Management
```typescript
interface ScenarioAnalysisPro {
  selectedPortfolio: string          // Portfolio ID
  selectedScenario: string           // Scenario name
  analysisResult: AnalysisResult | null
  loadingAnalysis: boolean
  analysisHistory: AnalysisHistoryItem[]
}
```

### Styling
- Background: Gradient from slate-50 to slate-100 (light mode)
- Cards: White bg with slate-200 borders
- Text: slate-900 headings, slate-600 labels
- Accents: Blue (#137fec) for primary actions
- Gauges: Green for good, yellow for warning, red for danger

---

## 2. AI Scenario Proposal Modal (`AIScenarioProposal.tsx`)

### Purpose
Modal overlay for AI-generated market scenarios based on current data.

### Trigger
When user selects "AI Scenario Proposal..." from scenario dropdown.

### Sections

#### Header
```typescript
Title: "AI Proposed Scenarios"
Subtitle: "Leverage AI to identify and propose new scenarios based on current and historical market data"
Close Button: X icon (top right)
```

#### Market Snapshot Section
```typescript
Title: "AI-Powered Market Snapshot"
Description: Narrative about current market conditions
Three metric cards:
  1. S&P 500
     - Value: 4,510.50
     - Change: +0.5% (green)
  
  2. VIX
     - Value: 15.80
     - Change: -1.2% (green, inverse)
  
  3. 10-Yr Treasury Yield
     - Value: 4.25%
     - Change: +0.02% (red for rate increase)
```

#### AI-Generated Scenarios List
```typescript
For each scenario card:
  - Title: Scenario name
  - Category badge: Blue (#137fec)
  - Impact badge: Red/Yellow/Green based on severity
  - Description: 1-2 sentence explanation
  - Confidence score: Top right (92%, 78%, etc.)
  - Buttons:
    * "Run Analysis" - Blue button
    * "View Details" - Gray button
    
Card layout: Full width, stacked vertically
Hover effect: Border highlights on hover
```

#### Footer
```typescript
Left: "Refresh" button with reload icon
Right: "Cancel" button (text only)
```

### Modals Triggered

#### Scenario Details Sub-Modal
When user clicks "View Details":

```typescript
Overlays the main modal with higher z-index
Title: "Scenario Details: [Scenario Name]"
Close button in header

Content Sections:
1. AI Rationale
   - Main explanation paragraph
   - Key Drivers table (3 rows):
     * Key Driver 1 | Description
     * Key Driver 2 | Description
     * Key Driver 3 | Description

2. Projected Impact
   - 3 columns:
     * Projected Alpha: +1.5%
     * Risk Profile: Moderate-High
     * Key Sector Exposure: Tech, Healthcare

3. Supporting Data
   - Tab navigation:
     * Market Trends (default)
     * Economic Indicators
     * Backtested Performance
   - Placeholder for interactive chart/data table

Footer:
  - "Close" button
  - "Use this Scenario" button (blue, primary)
```

### Data Structure
```typescript
interface AIProposedScenario {
  id: string
  title: string
  description: string
  confidence: number          // 0-100
  impact: 'High' | 'Medium' | 'Low'
  category: string
  marketSnapshot?: string
  keyDrivers?: string[]
  projectedAlpha?: number
  riskProfile?: string
}

interface MarketData {
  sp500: number
  sp500Change: number
  vix: number
  vixChange: number
  treasuryYield: number
  treasuryYieldChange: number
}
```

### API Endpoints
```
GET /api/ai/scenario-proposals
Response: {
  scenarios: AIProposedScenario[]
  marketData: MarketData
}
```

### User Flow
```
1. User in ScenarioAnalysisPro selects "AI Scenario Proposal..." from dropdown
2. AIScenarioProposal modal opens
3. Displays market snapshot + 3-5 AI-generated scenarios
4. User can:
   a. Click "Run Analysis" → closes modal, runs analysis with that scenario
   b. Click "View Details" → opens sub-modal with full details
   c. From sub-modal: "Use this Scenario" → same as (4a)
   d. Click "Refresh" → refetch scenarios
   e. Click "Cancel" → close modal
```

---

## 3. Custom Scenario Builder (Future Enhancement)

### Purpose
Allow users to define custom market conditions.

### Trigger
When user selects "Create Custom Scenario..." from scenario dropdown.

### UI Pattern
Modal similar to AI Proposal with form:
```typescript
Fields:
  - Scenario Name (text input)
  - Market Shock Type (dropdown):
    * Market Correction
    * Rate Change
    * Sector Rotation
    * Inflation Scenario
    * Custom
  
  - Severity (slider): 1-100
  - Asset Class Adjustments:
    * Equities: -50% to +50%
    * Bonds: -50% to +50%
    * Commodities: -50% to +50%
  
Buttons:
  - "Save Scenario" (blue)
  - "Cancel" (gray)
  - "Preview" (see results before saving)
```

---

## 4. Gauge Component (`Gauge.tsx`)

### Purpose
Reusable SVG gauge chart for visual metrics display.

### Props
```typescript
interface GaugeProps {
  value: number                      // Current value
  max?: number                       // Maximum value (default: 100)
  color?: string                     // Hex color (#00875A, #FFAB00, #DE350B)
  size?: 'small' | 'medium' | 'large'
  label?: string                     // Optional label above gauge
  showDelta?: boolean                // Show change from baseline
  deltaValue?: number
}
```

### Rendering
- SVG circle chart
- Filled arc represents percentage of max
- Rotated -90 degrees so fill starts at top
- Center text shows exact value
- Stroke color indicates performance

### Color Schemes
```
Green (#00875A): Good performance (Sharpe > 1.5)
Yellow (#FFAB00): Warning (Sharpe 0.8-1.5)
Red (#DE350B): Risk (Sharpe < 0.8)
```

---

## Integration Points

### 1. Route Integration
```typescript
// In router configuration
import ScenarioAnalysisPro from './components/ScenarioAnalysisPro'
import AIScenarioProposal from './components/AIScenarioProposal'
import Gauge from './components/Gauge'

<Route path="/scenario-analysis" component={ScenarioAnalysisPro} />
```

### 2. Main Navigation
Add to sidebar/top nav:
```
- Dashboard
- Portfolios
- **Scenario Analysis** ← new route
- Reporting
- Settings
```

### 3. Apollo Client Subscriptions
```graphql
subscription {
  portfolios {
    id
    aum
    sharpe
    risk
    status
    assetAllocation {
      asset
      percentage
    }
  }
}
```

### 4. API Endpoints Required

Backend must implement:

```
POST /api/portfolio/:id/scenario
  Body: { scenario: "Market Crash (-20%)" }
  Response: {
    baseCase: { aum, sharpe, risk, status, assetAllocation }
    scenarioCase: { aum, aumChange, sharpe, sharpeChange, risk, riskChange, status, assetAllocation }
    comparison: { aumDifference, sharpeDifference, riskDifference }
  }

GET /api/ai/scenario-proposals
  Response: {
    scenarios: AIProposedScenario[]
    marketData: MarketData
  }
```

### 5. Temporal Workflow Integration
```typescript
// Call existing workflow
POST /api/portfolio/:id/scenario
  → ExecuteWorkflow(..., ScenarioAnalysis, portfolioID, scenario)
  → Uses xAI for projections
  → Returns analysis results
```

---

## Design Specifications

### Color Palette
```
Primary: #137fec (Blue)
Success: #00875A (Green)
Warning: #FFAB00 (Amber)
Danger: #DE350B (Red)

Backgrounds:
  Light: #f6f7f8
  Dark: #101922
  Card Light: #ffffff
  Card Dark: #1a2531

Text:
  Light: #172B4D
  Dark: #ffffff
  Muted Light: #6b7280
  Muted Dark: #9dabb9
```

### Typography
```
Font: Inter
Sizes: 12px, 14px, 16px, 18px, 20px, 24px, 32px
Weights: 400 (normal), 500 (medium), 600 (semibold), 700 (bold), 900 (black)
```

### Spacing
```
xs: 4px
sm: 8px
md: 12px
lg: 16px
xl: 24px
2xl: 32px
```

### Shadows
```
sm: 0 1px 2px 0 rgba(0, 0, 0, 0.05)
md: 0 4px 6px -1px rgba(0, 0, 0, 0.1)
lg: 0 10px 15px -3px rgba(0, 0, 0, 0.1)
xl: 0 20px 25px -5px rgba(0, 0, 0, 0.1)
2xl: 0 25px 50px -12px rgba(0, 0, 0, 0.25)
```

### Border Radius
```
DEFAULT: 4px
lg: 8px
xl: 12px
full: 9999px
```

---

## Accessibility Considerations

1. **ARIA Labels**: All buttons and interactive elements have discernible text
2. **Keyboard Navigation**: Full support for Tab, Enter, Escape
3. **Color Contrast**: All text meets WCAG AA standards
4. **Focus Management**: Modals trap focus, return focus on close
5. **Screen Reader**: Proper semantic HTML and ARIA attributes

---

## Performance Optimizations

1. **Lazy Loading**: AI Proposal modal loads only when triggered
2. **Memoization**: Gauge components memoized to prevent re-renders
3. **Debouncing**: Scenario selection debounced to prevent rapid requests
4. **Caching**: Analysis history stored in component state
5. **Virtual Scrolling**: History list uses virtual scrolling if > 50 items

---

## Responsive Design

### Breakpoints
- **sm**: 640px
- **md**: 768px
- **lg**: 1024px
- **xl**: 1280px

### Adjustments
- **sm & below**: Stack left panel above right panel (100% width each)
- **md**: 40/60 split
- **lg+**: 33/67 split (current design)

---

## Testing Checklist

- [ ] Portfolio selector populates correctly
- [ ] Scenario dropdown includes all options
- [ ] Run Analysis button disabled until both selections made
- [ ] API call succeeds and returns correct data structure
- [ ] Results render in correct panels
- [ ] Gauge charts display with correct percentages
- [ ] History list persists within session
- [ ] Clicking history item loads correct result
- [ ] AI Proposal modal opens and fetches scenarios
- [ ] Scenario details sub-modal opens correctly
- [ ] Refresh button refetches scenarios
- [ ] Dark mode styling applies consistently
- [ ] All accessibility requirements met
- [ ] Responsive breakpoints work on mobile/tablet

---

## Future Enhancements

1. **Export Analysis**: Download PDF report of scenario analysis
2. **Scenario Comparison**: Compare multiple scenarios side-by-side
3. **Automation**: Schedule recurring scenario analyses
4. **Alerts**: Set thresholds for analysis results
5. **Collaboration**: Share scenarios with team members
6. **Advanced Analytics**: Sensitivity analysis, Monte Carlo simulations
7. **Machine Learning**: Pattern recognition in scenario outcomes
8. **Real-Time Updates**: WebSocket for live market data in snapshots

---

## Component Hierarchy

```
ScenarioAnalysisPro
├── Configuration Panel
│   ├── Portfolio Selector
│   ├── Scenario Selector
│   ├── Run Analysis Button
│   └── Analysis History
│       └── History Items
└── Results Display
    ├── Base Case Card
    │   ├── Metrics
    │   ├── Gauge (Sharpe)
    │   ├── Gauge (Risk)
    │   └── Asset Allocation Bars
    ├── Scenario Case Card
    │   ├── Metrics
    │   ├── Gauge (Sharpe with delta)
    │   ├── Gauge (Risk with delta)
    │   └── Asset Allocation Bars
    └── Comparison Analysis
        └── Metric Columns

AIScenarioProposal (Modal)
├── Header
├── Market Snapshot
│   └── Metric Cards (S&P 500, VIX, Treasury)
├── Scenarios List
│   └── Scenario Cards
│       └── Action Buttons
├── Footer
└── ScenarioDetailsModal (Sub-modal)
    ├── AI Rationale
    ├── Projected Impact
    ├── Supporting Data (Tabs)
    └── Action Buttons
```

---

## File Structure

```
frontend/src/components/
├── ScenarioAnalysisPro.tsx        (Main screen)
├── AIScenarioProposal.tsx         (AI proposal modal + details)
├── Gauge.tsx                       (Reusable gauge chart)
├── styles/
│   └── scenarioAnalysis.css       (All styling)
└── hooks/
    └── useScenarioAnalysis.ts     (Custom hook for logic)
```

---

This comprehensive guide provides all specifications needed to build a world-class, production-ready Scenario Analysis feature rivaling industry leaders like Addepar and Aladdin.
