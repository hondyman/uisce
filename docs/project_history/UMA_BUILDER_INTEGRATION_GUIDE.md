# UMA Builder React Component - Integration Guide

## Overview

The **UMA Builder** is a production-ready React component for visualizing and managing Unified Managed Accounts (UMA) portfolios. It provides:

- **Visual sleeve management** using ReactFlow for drag-and-drop portfolio design
- **Real-time drift detection** with color-coded warnings
- **Rebalance plan generation** with AI-suggested trades
- **Approval workflows** for compliance-gated execution
- **Full ABAC enforcement** via tenant context and headers
- **Responsive design** for desktop and mobile

## Features

### 1. Visual Portfolio Layout
- Sleeves displayed as interconnected nodes in ReactFlow
- Color-coded by drift status (green = healthy, red = rebalancing needed)
- Tooltips show detailed sleeve metrics
- Drag-and-drop reorganization (future enhancement)

### 2. Sleeve Management
- Edit target allocations and drift thresholds
- View current vs. target allocations per sleeve
- Real-time validation against business rules
- Audit trail for all changes

### 3. Rebalance Workflow
- **Suggest Rebalance** button triggers API call to workflow
- Displays rebalance plan with suggested trades
- Shows tax harvesting opportunities
- Approval workflow with signal support
- Real-time status polling via Temporal

### 4. Compliance & Security
- Multi-tenant isolation via X-Tenant-ID headers
- ABAC enforcement at component level
- Datasource-scoped queries
- Audit logging of all rebalance actions

## Installation

### 1. Add Dependencies

```bash
npm install reactflow @tanstack/react-query @mui/material @mui/icons-material
```

### 2. Import Component

```tsx
import { UMABuilder } from '@/components/UMABuilder';
```

### 3. Basic Usage

```tsx
// Simple case: render builder for a specific UMA
<UMABuilder 
  umaId="uma-123"
  onRebalanceTriggered={(workflowId) => console.log('Workflow:', workflowId)}
/>

// Read-only mode (for advisors viewing only)
<UMABuilder 
  umaId="uma-123"
  readOnly={true}
/>
```

## Props

```typescript
interface UMABuilderProps {
  umaId?: string;                           // UMA account ID to load
  onRebalanceTriggered?: (workflowId: string) => void;  // Callback when rebalance starts
  readOnly?: boolean;                       // Disable editing (default: false)
}
```

## Integration Example

### Page Component

```tsx
import React, { useState } from 'react';
import { UMABuilder } from '@/components/UMABuilder';
import { Box, Alert } from '@mui/material';

export const UMAManagementPage: React.FC = () => {
  const [currentWorkflow, setCurrentWorkflow] = useState<string | null>(null);

  const handleRebalanceTriggered = (workflowId: string) => {
    setCurrentWorkflow(workflowId);
    // Optionally navigate to workflow status page
    // navigate(`/workflows/${workflowId}`);
  };

  return (
    <Box sx={{ p: 3 }}>
      <h1>UMA Portfolio Management</h1>
      
      {currentWorkflow && (
        <Alert severity="info" sx={{ mb: 2 }}>
          Rebalance workflow started: {currentWorkflow}
        </Alert>
      )}

      <UMABuilder 
        umaId="uma-123"
        onRebalanceTriggered={handleRebalanceTriggered}
      />
    </Box>
  );
};
```

### Dashboard Integration

```tsx
import { UMABuilder } from '@/components/UMABuilder';
import { useState } from 'react';

export const PortfolioDashboard = () => {
  const [selectedUMA, setSelectedUMA] = useState('uma-001');

  return (
    <div style={{ display: 'grid', gridTemplateColumns: '200px 1fr', gap: 16, height: '100vh' }}>
      {/* Sidebar: UMA List */}
      <div>
        <h3>Accounts</h3>
        {/* UMA selection UI */}
      </div>

      {/* Main: UMA Builder */}
      <UMABuilder umaId={selectedUMA} />
    </div>
  );
};
```

## API Integration

The component calls these backend endpoints (all tenant-scoped):

### 1. Load UMA Account
```
GET /api/uma/:id
  ?tenant_id=<ID>&datasource_id=<ID>
  Headers: X-Tenant-ID, X-Tenant-Datasource-ID
```

**Response:**
```json
{
  "id": "uma-123",
  "name": "John Smith Portfolio",
  "aum": 5000000,
  "status": "active",
  "sleeves": [
    {
      "id": "sleeve-1",
      "model": "Growth",
      "sleeveType": "equities",
      "targetAllocation": 0.60,
      "currentAllocation": 0.62,
      "drift": 0.02,
      "minDriftThreshold": 0.05,
      "status": "active"
    }
  ]
}
```

### 2. Update Sleeve
```
PUT /api/uma/sleeves/:id
  ?tenant_id=<ID>&datasource_id=<ID>
  Headers: X-Tenant-ID, X-Tenant-Datasource-ID
  Body: {
    "targetAllocation": 0.65,
    "minDriftThreshold": 0.05
  }
```

### 3. Trigger Rebalance
```
POST /api/uma/rebalance/request
  ?tenant_id=<ID>&datasource_id=<ID>
  Headers: X-Tenant-ID, X-Tenant-Datasource-ID
  Body: {
    "uma_account_id": "uma-123",
    "request_type": "manual"
  }
```

**Response:**
```json
{
  "workflow_id": "workflow-abc123",
  "plan": {
    "id": "plan-123",
    "driftSignal": 0.02,
    "trades": [
      {
        "symbol": "VTSAX",
        "side": "buy",
        "quantity": 500,
        "estimatedPrice": 145.67,
        "estimatedValue": 72835.00,
        "reason": "Rebalance to target allocation"
      }
    ],
    "approvalStatus": "pending_approval"
  }
}
```

### 4. Approve Rebalance
```
POST /api/uma/rebalance/:planId/approve
  ?tenant_id=<ID>&datasource_id=<ID>
  Headers: X-Tenant-ID, X-Tenant-Datasource-ID
  Body: {
    "approval_signal": "approved_by_compliance"
  }
```

## UI Features

### Header Section
- Account name and AUM display
- Status indicator
- "Suggest Rebalance" button (red when drift detected, blue otherwise)

### Summary Cards
- Total current allocation (% of target)
- Number of sleeves
- AUM display
- Status indicator

### Drift Alert
Shows warning when any sleeve exceeds its drift threshold with recommendation to rebalance.

### ReactFlow Canvas
- Visualizes sleeves as connected nodes
- Shows drift status via color coding
- Tooltips with detailed metrics
- Responsive layout

### Sleeves Table
- Tabular view of all sleeves
- Columns:
  - Model, Type
  - Target %, Current %, Drift %
  - Threshold %
  - Status
  - Actions (Edit)

### Edit Sleeve Dialog
- Modify target allocation
- Update drift threshold
- Read-only fields: Model, Type
- Persist changes to backend

### Rebalance Plan Dialog
- Shows suggested trades
- Displays trade details: Symbol, Side, Qty, Price, Value, Reason
- Provides approval notes field
- Approve & Execute button (if pending approval)

## State Management

Uses React Query for:
- Data fetching and caching
- Mutation handling (PUT, POST requests)
- Error handling and retry logic
- Background polling for workflow status

Example:
```tsx
const { data: umaAccount, isLoading, error } = useQuery({
  queryKey: ['uma', umaId],
  queryFn: async () => {
    // Fetch UMA data...
  },
  enabled: !!umaId,
});
```

## Error Handling

The component gracefully handles:
- Missing/invalid UMA ID → Alert: "UMA Account not found"
- Network errors → Retry with exponential backoff
- Permission errors → Disable actions, show message
- Validation errors → Show inline field errors

## Tenant Scoping

All API calls automatically include:
```tsx
// Query parameters
?tenant_id=${localStorage.getItem('selected_tenant')}
&datasource_id=${localStorage.getItem('selected_datasource')}

// Headers
X-Tenant-ID: <from localStorage>
X-Tenant-Datasource-ID: <from localStorage>
```

⚠️ **Important**: Ensure tenant context is set in localStorage before rendering component:
```tsx
localStorage.setItem('selected_tenant', 'tenant-123');
localStorage.setItem('selected_datasource', 'ds-456');
```

## Accessibility

- Keyboard navigation support (Tab, Arrow keys in ReactFlow)
- ARIA labels on buttons and form fields
- Color contrast meets WCAG AA standards
- Responsive design for mobile/tablet

## Performance Optimization

1. **React Query Caching**: UMA data cached with 5-minute stale time
2. **Lazy Loading**: ReactFlow canvas only rendered when data available
3. **Memoization**: Node components memoized to prevent re-renders
4. **Debounced Updates**: Form input updates debounced 300ms

## Customization

### Custom Node Renderer
```tsx
// Override sleeve node appearance
const CustomSleeveNode = ({ data }: NodeProps) => (
  <div style={{ /* your styles */ }}>
    {data.model}
  </div>
);

// Pass to ReactFlow
<ReactFlow nodes={nodes.map(n => ({ ...n, type: 'custom' }))} />
```

### Custom Styling
Override CSS variables:
```css
:root {
  --uma-primary-color: #1976d2;
  --uma-drift-threshold-color: #d32f2f;
  --uma-healthy-color: #4caf50;
}
```

## Troubleshooting

### Component Not Loading
- **Check**: Tenant context set in localStorage
- **Check**: UMA ID is valid
- **Check**: User has ABAC permission for read:uma

### Rebalance Not Triggering
- **Check**: Network tab for /api/uma/rebalance/request POST
- **Check**: Temporal server running and healthy
- **Check**: ABAC engine permits rebalance:create action

### Trades Not Showing
- **Check**: Rebalance workflow completed successfully
- **Check**: Tax harvesting simulation succeeded
- **Check**: Trade execution constraints satisfied

## Testing

```tsx
import { render, screen } from '@testing-library/react';
import { UMABuilder } from '@/components/UMABuilder';

describe('UMABuilder', () => {
  it('should render UMA account name', async () => {
    render(<UMABuilder umaId="uma-123" />);
    
    // Component fetches data, then renders
    expect(await screen.findByText(/John Smith/)).toBeInTheDocument();
  });

  it('should show drift warning when threshold exceeded', async () => {
    render(<UMABuilder umaId="uma-123" />);
    
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'One or more sleeves have exceeded drift threshold'
    );
  });

  it('should trigger rebalance on button click', async () => {
    const onRebalanceTriggered = jest.fn();
    
    render(
      <UMABuilder 
        umaId="uma-123" 
        onRebalanceTriggered={onRebalanceTriggered}
      />
    );
    
    const button = await screen.findByRole('button', { name: /Suggest Rebalance/ });
    fireEvent.click(button);
    
    expect(onRebalanceTriggered).toHaveBeenCalled();
  });
});
```

## Performance Targets

- **Initial Load**: < 2s (UMA data fetch + canvas render)
- **Plan Generation**: < 5s (rules evaluation + tax sim)
- **Approval Workflow**: < 1s (signal via Temporal)
- **UI Response**: < 100ms (sleeves table sort, filters)

## Security Notes

1. **Multi-Tenant Safety**: All queries filtered by tenant_id
2. **ABAC Enforcement**: Component respects read:uma, write:uma permissions
3. **Data Privacy**: No sensitive data logged to console
4. **XSS Protection**: All user input sanitized via React

## Future Enhancements

- [ ] Drag-and-drop sleeve reordering
- [ ] AI-suggested rebalancing (xAI integration)
- [ ] What-if scenario modeling
- [ ] Tax impact visualization
- [ ] Restricted list integration
- [ ] Multi-custodian support
- [ ] Real-time price streaming
- [ ] Rebalance analytics dashboard

## Support & Contribution

For issues or feature requests:
1. Check existing GitHub issues
2. Provide reproduction steps
3. Include browser/OS version
4. Attach network logs if API-related

---

**Status**: Production Ready ✅
**Last Updated**: October 28, 2025
**Maintenance**: Actively maintained
