# Quick Start - MDM Material-UI Integration

## 5-Minute Setup

### 1. Install Dependencies
```bash
npm install @mui/material @mui/icons-material @mui/system @emotion/react @emotion/styled
```

### 2. Basic App Shell
Create `src/App.tsx`:

```typescript
import React, { useState } from 'react';
import { Box, Button, Stack, Container } from '@mui/material';
import { createTheme, ThemeProvider } from '@mui/material/styles';
import CssBaseline from '@mui/material/CssBaseline';

// Import all dashboard components
import { SemanticRuleBuilderDashboard } from './components/MDM_MUI_Dashboard';
import { RuleImpactComparison } from './components/MDM_MUI_RuleComparison';
import { ImpactAnalysisDashboard } from './components/MDM_MUI_ImpactAnalysis';
import { RealtimeNotificationsDashboard } from './components/MDM_MUI_RealtimeNotifications';

// Create custom theme
const theme = createTheme({
  palette: {
    primary: {
      main: '#137fec',
    },
    background: {
      default: '#f6f7f8',
    },
  },
  typography: {
    fontFamily: '"Inter", -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
  },
});

type PageType = 'dashboard' | 'comparison' | 'impact' | 'realtime';

export default function App() {
  const [currentPage, setCurrentPage] = useState<PageType>('dashboard');

  const pages: Record<PageType, React.ReactNode> = {
    dashboard: <SemanticRuleBuilderDashboard />,
    comparison: <RuleImpactComparison />,
    impact: <ImpactAnalysisDashboard />,
    realtime: <RealtimeNotificationsDashboard />,
  };

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <Box sx={{ minHeight: '100vh' }}>
        {/* Optional: Navigation Bar */}
        <Box sx={{ display: 'flex', gap: 1, p: 2, bgcolor: '#f3f4f6' }}>
          <Button
            variant={currentPage === 'dashboard' ? 'contained' : 'outlined'}
            onClick={() => setCurrentPage('dashboard')}
          >
            Dashboard
          </Button>
          <Button
            variant={currentPage === 'comparison' ? 'contained' : 'outlined'}
            onClick={() => setCurrentPage('comparison')}
          >
            Comparison
          </Button>
          <Button
            variant={currentPage === 'impact' ? 'contained' : 'outlined'}
            onClick={() => setCurrentPage('impact')}
          >
            Impact
          </Button>
          <Button
            variant={currentPage === 'realtime' ? 'contained' : 'outlined'}
            onClick={() => setCurrentPage('realtime')}
          >
            Real-time
          </Button>
        </Box>

        {/* Render current page */}
        {pages[currentPage]}
      </Box>
    </ThemeProvider>
  );
}
```

### 3. Run
```bash
npm start
```

---

## Component Import Patterns

### Single Component Usage
```typescript
import { SemanticRuleBuilderDashboard } from './MDM_MUI_Dashboard';

export default function Page1() {
  return <SemanticRuleBuilderDashboard />;
}
```

### With Custom Theme Override
```typescript
import { createTheme, ThemeProvider } from '@mui/material/styles';
import { RuleImpactComparison } from './MDM_MUI_RuleComparison';

const customTheme = createTheme({
  palette: {
    primary: { main: '#YOUR_COLOR' },
  },
});

export default function App() {
  return (
    <ThemeProvider theme={customTheme}>
      <RuleImpactComparison />
    </ThemeProvider>
  );
}
```

### Multiple Components in Router
```typescript
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { SemanticRuleBuilderDashboard } from './MDM_MUI_Dashboard';
import { RuleImpactComparison } from './MDM_MUI_RuleComparison';
import { ImpactAnalysisDashboard } from './MDM_MUI_ImpactAnalysis';
import { RealtimeNotificationsDashboard } from './MDM_MUI_RealtimeNotifications';

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/dashboard" element={<SemanticRuleBuilderDashboard />} />
        <Route path="/comparison" element={<RuleImpactComparison />} />
        <Route path="/impact" element={<ImpactAnalysisDashboard />} />
        <Route path="/realtime" element={<RealtimeNotificationsDashboard />} />
      </Routes>
    </BrowserRouter>
  );
}
```

---

## Component Capabilities

### Page 1: Semantic Rule Builder Dashboard
**Use When**: Designing/editing semantic rules
- ✅ 3-column responsive layout
- ✅ Drag-drop semantic terms (UI ready)
- ✅ Rule editor with IF/THEN/ELSE blocks
- ✅ Confidence scoring interface
- ✅ Save/Publish actions
- ✅ Test dataset selector
- ✅ Live simulation results

**Data Integration Points**:
```typescript
// Connect to your backend
const [rules, setRules] = useState([]);
const [results, setResults] = useState([]);

// On Save
const handleSave = async (ruleData) => {
  const response = await fetch('/api/v1/rules', {
    method: 'POST',
    body: JSON.stringify(ruleData)
  });
  // Update UI
};

// On Run Simulation
const handleSimulate = async (testData) => {
  const response = await fetch('/api/v1/simulate', {
    method: 'POST',
    body: JSON.stringify(testData)
  });
  setResults(await response.json());
};
```

---

### Page 2: Rule Impact & Version Comparison
**Use When**: Comparing production vs draft rules
- ✅ Side-by-side code diff
- ✅ Confidence shift visualization
- ✅ Sample result table
- ✅ Approval workflow tracking
- ✅ Change justification form
- ✅ Risk level indicator

**Data Integration Points**:
```typescript
// Fetch versions
const [v21, setV21] = useState(null);
const [v22, setV22] = useState(null);

useEffect(() => {
  const fetchVersions = async () => {
    const prod = await fetch('/api/v1/rules/rule-092/versions/2.1');
    const draft = await fetch('/api/v1/rules/rule-092/versions/2.2');
    setV21(await prod.json());
    setV22(await draft.json());
  };
  fetchVersions();
}, []);

// Submit for review
const handleSubmit = async (justification) => {
  await fetch('/api/v1/rules/rule-092/submit-review', {
    method: 'POST',
    body: JSON.stringify({ justification, version: '2.2' })
  });
};
```

---

### Page 3: Impact Analysis Dashboard
**Use When**: Analyzing rule business impact
- ✅ KPI cards with metrics
- ✅ Business unit breakdown
- ✅ Process distribution charts
- ✅ 7-day trend analysis
- ✅ Multi-level filtering
- ✅ Export/share reports

**Data Integration Points**:
```typescript
// Fetch impact metrics
const [metrics, setMetrics] = useState({
  totalImpacted: 0,
  accuracy: 0,
  conflicts: 0,
  remediationTime: 0,
});

useEffect(() => {
  const fetchMetrics = async () => {
    const response = await fetch('/api/v1/impact-analysis', {
      params: {
        businessUnit: selectedUnit,
        dateRange: dateRange
      }
    });
    setMetrics(await response.json());
  };
  fetchMetrics();
}, [selectedUnit, dateRange]);
```

---

### Page 4: Real-time Operations Monitor
**Use When**: Monitoring jobs and system health
- ✅ Live event streaming
- ✅ Performance metrics dashboard
- ✅ Notification settings
- ✅ Event filtering by source
- ✅ Detailed event inspection
- ✅ Multi-transport configuration

**Data Integration Points**:
```typescript
// Server-Sent Events connection
useEffect(() => {
  const eventSource = new EventSource('/api/v1/events/stream');

  eventSource.onmessage = (event) => {
    const newEvent = JSON.parse(event.data);
    setStreamEvents(prev => [newEvent, ...prev]);
  };

  eventSource.onerror = () => {
    eventSource.close();
  };

  return () => eventSource.close();
}, []);

// Or WebSocket alternative
useEffect(() => {
  const socket = new WebSocket('wss://localhost:8080/events');

  socket.onmessage = (event) => {
    const message = JSON.parse(event.data);
    setStreamEvents(prev => [message, ...prev]);
  };

  return () => socket.close();
}, []);
```

---

## Common Customization Tasks

### Change Brand Colors
```typescript
const theme = createTheme({
  palette: {
    primary: { main: '#004D99' },    // Change from #137fec
    success: { main: '#00AA66' },
    warning: { main: '#FF8833' },
    error: { main: '#DD3333' },
  },
});
```

### Update Typography
```typescript
const theme = createTheme({
  typography: {
    fontFamily: '"Roboto", sans-serif',  // Change from Inter
    h4: { 
      fontSize: '2rem', 
      fontWeight: 600,  // Change from 900
    },
    body2: {
      fontSize: '0.95rem',  // Slightly larger
    },
  },
});
```

### Add Dark Mode
```typescript
const theme = createTheme({
  palette: {
    mode: 'dark',
    primary: { main: '#137fec' },
    background: {
      default: '#121212',
      paper: '#1e1e1e',
    },
  },
});
```

### Responsive Adjustments
```typescript
// In individual components
<Box
  sx={{
    p: { xs: 1, sm: 2, md: 3 },           // Padding
    fontSize: { xs: '0.875rem', md: '1rem' },  // Font size
    display: { xs: 'none', md: 'block' }  // Hide on mobile
  }}
>
  Content
</Box>
```

---

## Integration with Backend

### Example: Connecting to Feature 4 Services

**Export Service Integration**:
```typescript
// In MDM_MUI_Dashboard.tsx
const handleExport = async () => {
  const response = await fetch('/api/v1/jobs/my-job/exports', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      format: 'csv',  // or 'json', 'parquet'
      ruleId: 'rule-092'
    })
  });
  const { exportId } = await response.json();
  // Download when ready
};
```

**Scheduler Integration**:
```typescript
// In MDM_MUI_RealtimeNotifications.tsx
const handleSchedule = async (schedule) => {
  const response = await fetch('/api/v1/schedules', {
    method: 'POST',
    body: JSON.stringify({
      rule_id: 'rule-092',
      schedule_type: 'daily',
      schedule_value: '08:00:00',
      timezone: 'UTC'
    })
  });
};
```

---

## Styling Examples

### Custom Card Styling
```typescript
<Card
  sx={{
    background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
    borderRadius: 2,
    boxShadow: '0 10px 30px rgba(0,0,0,0.2)',
    color: 'white',
    '&:hover': {
      transform: 'scale(1.02)',
      boxShadow: '0 15px 40px rgba(0,0,0,0.3)',
    },
  }}
>
  Custom Styled Card
</Card>
```

### Custom Button Styles
```typescript
<Button
  variant="contained"
  sx={{
    background: 'linear-gradient(90deg, #137fec 0%, #0b5ed7 100%)',
    textTransform: 'none',
    fontSize: '1rem',
    fontWeight: 700,
    py: 1.5,
    px: 4,
    borderRadius: 1.5,
    '&:hover': {
      background: 'linear-gradient(90deg, #0b5ed7 0%, #0951ba 100%)',
    },
  }}
>
  Custom Button
</Button>
```

### Data Table Styling
```typescript
<Table
  sx={{
    '& thead': {
      '& th': {
        backgroundColor: '#f3f4f6',
        fontWeight: 700,
        fontSize: '0.75rem',
        textTransform: 'uppercase',
      },
    },
    '& tbody': {
      '& tr': {
        '&:hover': {
          backgroundColor: '#f9fafb',
        },
      },
    },
  }}
>
  {/* Table content */}
</Table>
```

---

## Performance Tips

### 1. Memoize List Components
```typescript
const EventCard = React.memo(({ event }: { event: StreamEvent }) => (
  <Card>{event.title}</Card>
));
```

### 2. Use useCallback for Event Handlers
```typescript
const handleSearch = useCallback(
  debounce((term: string) => {
    // Expensive search operation
  }, 300),
  []
);
```

### 3. Virtualize Long Lists
```typescript
import { FixedSizeList } from 'react-window';

<FixedSizeList
  height={600}
  itemCount={streamEvents.length}
  itemSize={120}
  width="100%"
>
  {({ index, style }) => (
    <Box style={style}>
      <EventCard event={streamEvents[index]} />
    </Box>
  )}
</FixedSizeList>
```

### 4. Lazy Load Images/Charts
```typescript
const ChartComponent = React.lazy(() => import('./Chart'));

<Suspense fallback={<Skeleton />}>
  <ChartComponent data={data} />
</Suspense>
```

---

## Troubleshooting

### Issue: "Cannot find module '@mui/material'"
```bash
npm install @mui/material @mui/icons-material @emotion/react @emotion/styled
```

### Issue: TypeScript errors in components
- Ensure `tsconfig.json` has `"jsx": "react-jsx"`
- Check `@types/react`, `@types/react-dom` are installed

### Issue: Theme colors not applying
- Ensure `ThemeProvider` wraps all components
- Use `sx={{ color: 'primary.main' }}` or `sx={{ color: theme.palette.primary.main }}`

### Issue: Responsive layout breaking on mobile
- Check that Grid items have `xs` breakpoint: `<Grid item xs={12} md={6}>`
- Test with Chrome DevTools mobile view

### Issue: Events not streaming in real-time
- Check CORS configuration: `Access-Control-Allow-Origin: *`
- Verify EventSource/WebSocket path is correct
- Check browser console for connection errors

---

## Next Steps

1. **Copy components** to your project's `src/components/` folder
2. **Install Material-UI dependencies** (see step 1 above)
3. **Create theme** using provided palette colors
4. **Wrap app** with `ThemeProvider`
5. **Import dashboards** and add to routes
6. **Connect to backend** API endpoints
7. **Test** on all target browsers
8. **Deploy** to production

---

## Resources

- **Material-UI Docs**: https://mui.com
- **Icons Reference**: https://fonts.google.com/icons
- **Theme Creator**: https://zenoo.github.io/mui-theme-creator/
- **Component API**: https://mui.com/api/
- **Styling Guide**: https://mui.com/material-ui/customize-components/

---

## Support

For questions or issues:
1. Check Material-UI documentation
2. Review component source code for patterns
3. Check TypeScript types in IDE IntelliSense
4. Test in isolated Storybook story
5. Check browser console for errors

