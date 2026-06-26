# WealthStream with Generative UI

## Overview
Metadata-first wealth management platform with AI-powered generative UI, inspired by Workday OMS.

## What Was Built

### Frontend GenUI Engine
- ✅ **Zod Schemas** (`genui/schema.ts`) - Type-safe component definitions
- ✅ **Dynamic Renderer** (`genui/Renderer.tsx`) - Renders AI-generated layouts
- ✅ **6 Widget Types**:
  - `ChartWidget` - Line, area, bar charts (Recharts)
  - `GridWidget` - Data tables (AG-Grid)
  - `CardWidget` - KPI/stat cards
  - `FormWidget` - Dynamic forms with validation
  - `TimelineWidget` - Audit/workflow events
  - `DisclosureBanner` - Compliance notices

### Backend Metadata Model
- ✅ **Core Domain** (`pkg/meta/model.go`)
  - `BusinessObjectDefinition` - Metadata-driven business objects
  - `FieldDefinition` - Dynamic fields with validation
  - `RelationshipDefinition` - Object relationships
  - `PolicyDefinition` - CEL-based governance

## Architecture

```
┌─────────────────────────────────────────────────┐
│         WealthStream GenUI Platform             │
├─────────────────────────────────────────────────┤
│                                                 │
│  NL Query → Intent API → Layout JSON → Render  │
│                                                 │
│  ┌──────────┐  ┌───────────┐  ┌──────────┐    │
│  │ User     │→ │  GenUI    │→ │ React    │    │
│  │ Intent   │  │  Backend  │  │ Renderer │    │
│  └──────────┘  └───────────┘  └──────────┘    │
│       ↓              ↓              ↓          │
│  ┌──────────────────────────────────────┐     │
│  │   Hasura GraphQL + Postgres          │     │
│  │   (Metadata Tables + Tenant Data)    │     │
│  └──────────────────────────────────────┘     │
│       ↓                                        │
│  ┌──────────────────────────────────────┐     │
│  │   StarRocks + Iceberg/Nessie         │     │
│  │   (Unified Lakehouse Analytics)      │     │
│  └──────────────────────────────────────┘     │
│                                                 │
└─────────────────────────────────────────────────┘
```

## Sample Layout JSON

```json
{
  "version": 1,
  "title": "Portfolio Dashboard",
  "components": [
    {
      "id": "nav_chart",
      "type": "chart",
      "chartType": "line",
      "title": "Portfolio NAV",
      "binding": {
        "gql": "query { portfolioNav { date nav } }",
        "dataPath": "data.portfolioNav"
      },
      "xField": "date",
      "yFields": ["nav"]
    },
    {
      "id": "holdings_grid",
      "type": "grid",
      "title": "Top Holdings",
      "columns": [
        {"field": "symbol", "headerName": "Symbol"},
        {"field": "shares", "headerName": "Shares"},
        {"field": "value", "headerName": "Value", "type": "currency"}
      ]
    }
  ]
}
```

## Usage

```tsx
import { GenUIRenderer } from "@/genui/Renderer";

// Fetch layout from backend
const layoutJson = await fetch("/genui/intent", {
  method: "POST",
  body: JSON.stringify({ query: "Show my portfolio performance" })
}).then(r => r.json());

// Render dynamically
<GenUIRenderer layoutJson={layoutJson} />
```

## Next Steps

1. **Backend GenUI API** - Implement layout generation with LLM
2. **CEL Visibility Rules** - Evaluate expressions for component visibility
3. **Hasura Integration** - Auto-generate GraphQL from metadata
4. **Agent Workflows** - Temporal workflows for portfolio operations
5. **Metadata Admin UI** - Configure business objects and policies

## Benefits

- ✅ **Metadata-Driven**: Add features via configuration, not code
- ✅ **AI-Powered**: Natural language → working dashboards
- ✅ **Type-Safe**: Zod validation + TypeScript
- ✅ **Multi-Tenant**: Isolated per client with RLS
- ✅ **Audit-Ready**: Every interaction tracked
- ✅ **Extensible**: Register new component types easily
