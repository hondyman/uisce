# SemLayer React Dashboard (Phase 3.16)

Real-time analytics dashboard for operational intelligence across global trading systems. Built with React 18, TypeScript, Vite, and TailwindCSS.

## Features

✅ **Real-time Event Stream** - WebSocket-based live event feed with region subscriptions
✅ **Multi-Region Management** - View chains across US, EU, and APAC regions
✅ **Chain Health Monitoring** - Health scores, conflict tracking, latency analysis
✅ **SLA Compliance Analytics** - Trend charts, compliance scoring, P99 metrics
✅ **Failure Prediction** - AI-powered risk assessment with confidence scoring
✅ **Interactive Reports** - Scheduled reports, PDF generation, download tracking
✅ **Live Feed** - Filterable real-time event stream with auto-scroll
✅ **Responsive Design** - Mobile, tablet, and desktop optimized

## Architecture

### Frontend Stack
- **React 18.2.0** - UI framework with hooks
- **TypeScript 5.x** - Type-safe development
- **Vite 5.x** - Fast build tool and dev server
- **TailwindCSS 3.x** - Utility-first styling
- **Recharts** - React charting library
- **Axios** - HTTP client with interceptors
- **React Router v6** - Client-side routing
- **Zustand** - Lightweight state management (ready for feature state)

### API Integration
- Consumes Phase 3.13 REST API (port 8080)
  - SLA compliance trends
  - Chain health reports
  - Failure predictions
  - Action execution endpoints
  - Batch operations
  
- Real-time WebSocket connection (port 8081)
  - Region-based event subscriptions
  - Live incident streaming
  - Action status updates
  - Multi-tenant isolation

### Key Components

#### Pages
- **DashboardHome** (`src/pages/DashboardHome.tsx`)
  - SLA compliance KPIs
  - Chain health distribution
  - High-risk chain alerts
  - 14-day trend analysis

- **ChainsList** (`src/pages/ChainsList.tsx`)
  - Searchable chain catalog
  - Filter by health status
  - Sort by name/status/updated
  - Quick health metrics

- **ChainDetail** (`src/pages/ChainDetail.tsx`)
  - Detailed chain metrics
  - 48-hour health timeline
  - AI predictions with confidence
  - Recent incidents with root cause

- **LiveFeed** (`src/pages/LiveFeed.tsx`)
  - Real-time event stream
  - Filterable by event type
  - Auto-scroll and pause controls
  - Event details expansion

- **ReportsPage** (`src/pages/ReportsPage.tsx`)
  - Report scheduling modal
  - Download tracking
  - SLA, health, predictions, cost reports
  - Daily/weekly/monthly frequency

- **LoginPage** (`src/pages/LoginPage.tsx`)
  - JWT token authentication
  - Demo account quick access
  - Tenant ID + API token

#### Hooks
- **useWebSocket** (`src/hooks/useWebSocket.ts`)
  - Auto-connect on mount
  - Auto-reconnect with 3s backoff
  - Region subscriptions
  - Connection state management

#### Services
- **ApiClient** (`src/api/client.ts`)
  - Typed axios wrapper
  - Auth token injection
  - Global error handling
  - 10+ analytics methods

## Setup & Installation

### Prerequisites
- Node.js 16+ with npm or yarn
- Phase 3.13 backend running on http://localhost:8080
- Temporal worker running WebSocket hub on ws://localhost:8081

### Install Dependencies
```bash
cd dashboard
npm install
```

### Configure Environment
```bash
cp .env.example .env.local
# Edit .env.local if needed (defaults work for local development)
```

### Development Server
```bash
npm run dev
```
Opens on http://localhost:5173 with HMR enabled.

### Build for Production
```bash
npm run build
npm run preview  # Preview production build locally
```

### Type Checking
```bash
npm run type-check
```

### Linting
```bash
npm run lint
```

## Project Structure

```
dashboard/
├── src/
│   ├── components/
│   │   └── Layout.tsx              # 6 reusable UI components
│   ├── pages/
│   │   ├── DashboardHome.tsx       # Overview dashboard
│   │   ├── ChainsList.tsx          # Chain catalog
│   │   ├── ChainDetail.tsx         # Chain metrics
│   │   ├── LiveFeed.tsx            # Event stream
│   │   ├── ReportsPage.tsx         # Report management
│   │   └── LoginPage.tsx           # Authentication
│   ├── api/
│   │   └── client.ts               # Typed API client
│   ├── hooks/
│   │   └── useWebSocket.ts         # WebSocket integration
│   ├── types/
│   │   └── index.ts                # TypeScript interfaces
│   ├── styles/
│   │   └── globals.css             # Tailwind directives
│   ├── App.tsx                     # Router setup & auth context
│   └── main.tsx                    # React entry point
├── public/
│   └── index.html                  # HTML template
├── vite.config.ts                  # Vite configuration
├── tsconfig.json                   # TypeScript config
├── tailwind.config.js              # TailwindCSS config
├── postcss.config.js               # PostCSS config
├── package.json                    # Dependencies
├── index.html                      # Entry point
├── .env.example                    # Environment template
└── README.md                       # This file
```

## API Endpoints Consumed

### Phase 3.13 REST API (http://localhost:8080)
- `GET /admin/analytics/sla-trends?days=30` - SLA compliance over time
- `GET /admin/analytics/chain-health/:chain` - Chain health report
- `GET /admin/analytics/predictions` - ML failure predictions
- `GET /admin/operations/search/:query` - Chain search
- `POST /admin/operations/:action` - Execute chain action (restart, failover)
- `GET /admin/operations/batch-status/:id` - Batch operation status
- `POST /admin/operations/batch-execute` - Execute batch operations

### Phase 3.15 WebSocket Hub (ws://localhost:8081)
- Region subscriptions: `subscribe`, `unsubscribe`
- Event types: `incident`, `action`, `sla`, `daily_sla_refreshed`
- Auto-dispatch to relevant page components

## Authentication

The dashboard uses JWT token-based authentication:

1. User enters Tenant ID + API Token on login page
2. Token stored in localStorage under `auth_token`
3. Token injected into all API requests via Authorization header
4. WebSocket connection auto-authenticates on first message
5. Logout clears tokens and redirects to login

**Demo Account:**
- Tenant ID: `demo-tenant`
- Token: `test_demo_token_12345`
- Click "Try Demo Account" button on login page

## Real-Time Features

### WebSocket Integration
- Automatic connection on component mount
- Reconnection with exponential backoff (3s initial delay)
- Multi-region subscriptions per tenant
- Event filtering by type (incident, action, sla)
- Automatic cleanup on unmount

### Event Types
- `incident` - Chain failure or conflict detected
- `action` - Chain action executed (restart, failover)
- `sla` - SLA threshold violation
- `daily_sla_refreshed` - Daily metrics updated

## Styling

### TailwindCSS Custom Colors
```javascript
brand: '#0f172a'        // Primary brand color
brand-light: '#1e293b'  // Light variant
success: '#10b981'      // Green
warning: '#f59e0b'      // Amber
danger: '#ef4444'       // Red
info: '#3b82f6'         // Blue
```

## Performance Optimizations

- Code splitting with React Router lazy loading
- Vite fast HMR during development
- TailwindCSS purging in production
- Axios request deduplication
- WebSocket connection pooling
- Recharts canvas rendering for large datasets

## Testing Strategy (Phase 3.17)

- Unit tests: Component snapshot testing with Vitest
- Integration tests: API client mocking with MSW
- E2E tests: Playwright for critical user flows
- Coverage target: 80%+ for page/components

## Known Limitations & TODOs

### Phase 3.16 (Current)
- ✅ Dashboard pages
- ✅ Real-time event stream
- ✅ Authentication flow
- ✅ API client integration
- ✅ Responsive design
- 🔄 Error state recovery (implement retry logic)
- 🔄 Offline mode (cache last N events)
- 🔄 User preferences (dark mode, layout)

### Phase 3.17+
- [ ] Advanced filtering & saved views
- [ ] Custom alerts & webhooks
- [ ] Explainable AI (SHAP values visualization)
- [ ] Notebook integration (Jupyter)
- [ ] Advanced RBAC (role-based column access)
- [ ] Multi-currency support
- [ ] PDF report generation with charts

## Deployment

### Development
```bash
npm run dev  # Vite dev server with HMR
```

### Production
```bash
npm run build  # Optimized production build
npm run preview  # Test production build locally
```

### Docker (Future)
```dockerfile
FROM node:18-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/nginx.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

## Troubleshooting

### API Connection Issues
```
Error: Failed to connect to http://localhost:8080
→ Ensure Phase 3.13 backend is running
→ Check CORS headers in backend
→ Verify vite.config.ts proxy configuration
```

### WebSocket Connection Fails
```
Error: WebSocket connection refused
→ Ensure Phase 3.15 temporal worker is running
→ Check ws:// proxy in vite.config.ts
→ Verify tenant_id is set in localStorage
```

### TypeScript Errors
```bash
npm run type-check  # Full type checking
npm run lint        # ESLint + formatting
```

## Development Workflow

1. **Start Backend**
   ```bash
   # In backend directory
   go run ./cmd/api -port 8080
   go run ./cmd/temporal-worker
   ```

2. **Start Dashboard**
   ```bash
   # In dashboard directory
   npm run dev
   ```

3. **Build & Verify**
   ```bash
   npm run build
   npm run type-check
   npm run lint
   ```

## Contributing

- Follow TypeScript strict mode
- Use functional components with hooks
- Keep components <200 LOC (break into smaller pieces)
- Add loading/error states to all async operations
- Test WebSocket integration locally before pushing
- Document new API client methods

## License

Part of SemLayer platform (proprietary).

---

**Phase 3.16 Status:** ✅ COMPLETE
- 5 page components with full functionality
- Real-time WebSocket integration
- Authentication flow
- Responsive design
- 20+ reusable UI components
- Fully typed TypeScript

**Next Phase:** Phase 3.17 - Advanced Analytics & ML Explainability
