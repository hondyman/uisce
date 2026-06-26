# Phase 3.16 Implementation Guide

## Overview

Phase 3.16 implements a complete React dashboard consuming all Phase 3.13-3.15 backend APIs. The implementation spans:

- 5 page components (Home, Chains, ChainDetail, LiveFeed, Reports)
- 6 reusable UI components
- 1 custom WebSocket hook with auto-reconnection
- 1 typed API client with auth
- Complete routing & authentication

## File Summary

### Configuration Files (9 total)
1. **package.json** - Dependencies, scripts, metadata
2. **tsconfig.json** - TypeScript strict mode config
3. **vite.config.ts** - Build tool config with API proxies
4. **tailwind.config.js** - Custom TailwindCSS theme
5. **postcss.config.js** - PostCSS with TailwindCSS plugin
6. **.npmrc** - npm configuration (legacy peer deps)
7. **.gitignore** - Git exclusions
8. **.env.example** - Environment variables template
9. **index.html** - HTML entry point

### Core Application (9 files)
1. **src/main.tsx** - React entry point (40 lines)
2. **src/App.tsx** - Router & auth context (60 lines)
3. **src/styles/globals.css** - Tailwind directives (15 lines)
4. **src/types/index.ts** - TypeScript interfaces (110 lines)
5. **src/api/client.ts** - Typed axios wrapper (150 lines)
6. **src/hooks/useWebSocket.ts** - WebSocket integration (160 lines)
7. **src/components/Layout.tsx** - UI components (210 lines)

### Pages (6 components, 1,000+ lines)
1. **src/pages/DashboardHome.tsx** - Dashboard overview (280 lines)
2. **src/pages/ChainsList.tsx** - Chain catalog (320 lines)
3. **src/pages/ChainDetail.tsx** - Chain metrics (280 lines)
4. **src/pages/LiveFeed.tsx** - Event stream (240 lines)
5. **src/pages/ReportsPage.tsx** - Report management (320 lines)
6. **src/pages/LoginPage.tsx** - Authentication (180 lines)

### Documentation
- **README.md** - Project overview and setup guide
- **DEVELOPMENT.md** - This file

## Component Hierarchy

```
App (Router + AuthContext)
├── LoginPage (standalone)
├── Layout (wrapper component)
│   ├── Sidebar (navigation)
│   └── Header (status bar)
├── DashboardHome (/)
│   ├── SLA Trend Chart (Recharts LineChart)
│   ├── Health Distribution (Recharts PieChart)
│   └── MetricCards (4x KPI cards)
├── ChainsList (/chains)
│   └── ChainCard Grid
│       ├── Card (wrapper)
│       ├── Badge (status)
│       └── Health Metrics
├── ChainDetail (/chains/:chainId)
│   ├── Health Timeline (Recharts AreaChart)
│   ├── Prediction Card
│   ├── Incident List
│   └── Action Buttons
├── LiveFeed (/feed)
│   ├── Filter Controls
│   ├── Event Stream (scrollable)
│   └── EventRow Components
└── ReportsPage (/reports)
    ├── Statistics (KPI cards)
    ├── Report List
    ├── Filter Tabs
    └── ScheduleModal
```

## Data Flow

### API Integration
```
User Action
    ↓
API Client Method
    ↓
Axios Interceptor (adds auth token)
    ↓
HTTP Request → Backend (:8080)
    ↓
Parse Response
    ↓
Update Component State (useState/setData)
    ↓
Render with new data
```

### WebSocket Integration
```
Component Mount (useEffect)
    ↓
Connect to WebSocket (:8081)
    ↓
Subscribe to regions (us-east-1, eu-west-1, apac-1)
    ↓
Listen for events
    ↓
onEvent callback fires
    ↓
Update event array in state
    ↓
Re-render LiveFeed component
    ↓
Cleanup on unmount (unsubscribe)
```

## Key Design Patterns

### 1. Protected Routes
```typescript
function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = React.useContext(AuthContext)
  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }
  return <>{children}</>
}
```

### 2. API Client Singleton
```typescript
// Create once, import everywhere
export const apiClient = new ApiClient(
  process.env.VITE_API_BASE_URL || 'http://localhost:8080'
)
```

### 3. Custom Hook Pattern
```typescript
const { isConnected, subscribe, unsubscribe } = useWebSocket({
  tenantId,
  regions: ['us-east-1', 'eu-west-1'],
  onEvent: (event) => {
    // Handle event
  }
})
```

### 4. Type-Safe Props
```typescript
interface MetricCardProps {
  label: string
  value: string | number
  change?: number
  trend?: 'up' | 'down' | 'neutral'
  color?: 'success' | 'warning' | 'danger' | 'info'
}
```

## State Management

### Local Component State
- Page components use `useState` for data fetching
- Forms use `useState` for input values
- Modals use `useState` for visibility

### Global Auth State
- `AuthContext` with `useContext` hook
- Persisted in localStorage
- Used by route guards

### Real-Time State
- `LiveFeed` maintains event array
- New events prepended to array
- Max 500 events kept in memory
- Can be paused/cleared by user

*Note: A more complex app would use Zustand store for global state.*

## Error Handling Strategy

### API Errors
```typescript
catch (err: unknown) {
  if (err instanceof AxiosError) {
    setError(err.response?.data?.message || err.message)
  } else {
    setError('An unexpected error occurred')
  }
}
```

### WebSocket Errors
```typescript
// Auto-reconnection with 3s backoff
if (ws.readyState === WebSocket.CLOSED) {
  setTimeout(() => connect(), 3000)
}
```

### Display Errors
```tsx
if (error) return <ErrorMessage message={error} />
```

## Performance Optimizations

1. **Code Splitting** - React Router lazy loading (future)
2. **Memoization** - useMemo/useCallback for expensive operations
3. **Virtual Scrolling** - For large event lists (future)
4. **Progressive Loading** - Skeleton screens (future)
5. **Recharts Optimization** - Single canvas rendering

## Testing Approach

### Component Testing (Vitest + React Testing Library)
```typescript
describe('DashboardHome', () => {
  it('should load SLA data on mount', async () => {
    // Mock API
    // Render component
    // Assert data loaded
  })
})
```

### Integration Testing (MSW)
```typescript
server.use(
  http.get('/admin/analytics/sla-trends', () => {
    return HttpResponse.json([...])
  })
)
```

### E2E Testing (Playwright)
```typescript
test('should login and view dashboard', async ({ page }) => {
  await page.goto('/login')
  await page.fill('[name="tenantId"]', 'demo')
  await page.click('button:has-text("Try Demo")')
  await expect(page).toHaveURL('/')
})
```

## Common Development Tasks

### Adding a New Page

1. Create file in `src/pages/NewPage.tsx`
2. Export function component
3. Add route in `App.tsx`:
   ```typescript
   <Route path="/new" element={<ProtectedRoute><NewPage /></ProtectedRoute>} />
   ```
4. Add to sidebar navigation

### Adding a New API Method

1. Add to `ApiClient` class:
   ```typescript
   async getNewData(param: string): Promise<NewType[]> {
     return this.get(`/admin/api/new-endpoint/${param}`)
   }
   ```
2. Add response type to `src/types/index.ts`
3. Import and use in component:
   ```typescript
   const data = await apiClient.getNewData('value')
   ```

### Adding WebSocket Event Handler

1. Handle in `onEvent` callback:
   ```typescript
   const { isConnected } = useWebSocket({
     onEvent: (event) => {
       if (event.type === 'new_event_type') {
         // Handle
       }
     }
   })
   ```

### Styling a Component

1. Use TailwindCSS utility classes:
   ```tsx
   <div className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700">
     Button
   </div>
   ```
2. Or create CSS module for scoped styles
3. Use custom colors defined in `tailwind.config.js`

## Debugging

### Browser DevTools
- React DevTools (props/state inspection)
- Network tab (API requests)
- Console (errors/logs)
- Application tab (localStorage)

### VS Code Debugging
```json
{
  "type": "chrome",
  "request": "launch",
  "name": "Launch Chrome against localhost:5173",
  "url": "http://localhost:5173",
  "webRoot": "${workspaceFolder}/dashboard"
}
```

### WebSocket Debugging
1. Browser DevTools → Network tab
2. Filter by WebSocket connections
3. Messages tab shows frames

### API Debugging
```typescript
// Log all requests
apiClient.axios.interceptors.request.use(config => {
  console.log('Request:', config.url, config.data)
  return config
})
```

## Deployment Checklist

- [ ] `npm run type-check` passes
- [ ] `npm run lint` passes
- [ ] `npm run build` succeeds
- [ ] Build output is minified & sourcemap-free
- [ ] `.env` file configured for target environment
- [ ] CORS enabled in backend
- [ ] WebSocket endpoint accessible
- [ ] Auth tokens validated
- [ ] Error pages tested

## Monitoring & Observability

Future additions:
- Error logging (Sentry)
- Performance monitoring (Web Vitals)
- User analytics (Mixpanel)
- Feature flags (LaunchDarkly)

## Architecture Decisions

### Why React 18?
- Modern hooks API with Suspense support
- Automatic batching
- Concurrent rendering
- Proven in production

### Why Vite?
- 10x faster than Webpack
- Near-instant HMR
- Optimized production builds
- Minimal configuration

### Why TailwindCSS?
- Rapid UI development
- Consistent design system
- Purges unused CSS
- Works well with Component-driven design

### Why Recharts?
- React-native chart library
- Responsive containers
- Composable components
- Good TypeScript support

### Why WebSocket vs REST polling?
- Real-time updates without latency
- Reduced server load
- Two-way communication
- Better for live feeds

## Migration Paths

### From Demo Data to Real API
1. Remove mock data from components
2. Ensure API client responds correctly
3. Handle actual API exceptions
4. Cache responses appropriately

### Adding State Management (Zustand)
```typescript
import create from 'zustand'

const useStore = create((set) => ({
  slaData: [],
  setSLAData: (data) => set({ slaData: data }),
}))
```

### Converting to Next.js
- Move `src/pages` to `app/`
- Setup API routes in `app/api/`
- Use Server Components where possible
- Built-in optimization

## Future Enhancements

**Phase 3.17:**
- ML model explainability (SHAP values)
- Advanced filtering & saved views
- Custom alerts and webhooks
- PDF report generation

**Phase 3.18:**
- Role-based access control
- Dark mode toggle
- Mobile app (React Native)
- Offline support

**Phase 3.19:**
- Notebook integration
- Custom widgets
- Multi-language support
- Advanced charting (3D, maps)

---

**Document Version:** 1.0  
**Last Updated:** Phase 3.16 Complete  
**Maintainer:** SemLayer Team
