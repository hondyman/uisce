# Portfolio Dashboard Integration Guide

This guide explains how to integrate the Portfolio Dashboard React component with the Portfolio Management System backend.

## 📦 Component Overview

The `PortfolioAnalysisDashboard` component provides:
- Real-time portfolio value tracking
- Allocation visualization (pie chart)
- Current vs Target allocation comparison
- Performance charting (30-day history)
- Holdings table with detailed metrics
- Rebalancing alerts and execution
- Fully dark-themed UI

## 🔗 Integration Steps

### Step 1: Copy Component File

The component is located at: `frontend/src/components/PortfolioAnalysisDashboard.tsx`

```bash
# Component already exists in the provided code
# Location: frontend/src/components/PortfolioAnalysisDashboard.tsx
```

### Step 2: Install Dependencies

The component requires:
```bash
npm install recharts lucide-react
```

**Dependencies**:
- `recharts` - Charts and graphs
- `lucide-react` - Icons
- `react` - Already installed

### Step 3: Update GraphQL Endpoint

In your `.env` file:
```env
VITE_API_BASE_URL=http://localhost:8080/v1/graphql
VITE_HASURA_ADMIN_SECRET=admin_secret_key
```

### Step 4: Setup Apollo Client

Create `src/lib/apolloClient.ts`:

```typescript
import {
  ApolloClient,
  InMemoryCache,
  HttpLink,
  ApolloLink,
  split,
  Observable,
} from '@apollo/client';
import { getMainDefinition } from '@apollo/client/utilities';
import { GraphQLWsLink } from '@apollo/client/link/subscriptions';
import { createClient } from 'graphql-ws';

// HTTP Link
const httpLink = new HttpLink({
  uri: process.env.VITE_API_BASE_URL || 'http://localhost:8080/v1/graphql',
  credentials: 'include',
  headers: {
    'X-Hasura-Admin-Secret': 
      process.env.VITE_HASURA_ADMIN_SECRET || 'admin_secret_key',
  },
});

// WebSocket Link for subscriptions
const wsLink = new GraphQLWsLink(
  createClient({
    url: (process.env.VITE_API_BASE_URL || 'http://localhost:8080/v1/graphql')
      .replace('http', 'ws'),
    connectionParams: {
      headers: {
        'X-Hasura-Admin-Secret': 
          process.env.VITE_HASURA_ADMIN_SECRET || 'admin_secret_key',
      },
    },
  })
);

// Split link based on operation type
const splitLink = split(
  ({ query }) => {
    const definition = getMainDefinition(query);
    return (
      definition.kind === 'OperationDefinition' &&
      definition.operation === 'subscription'
    );
  },
  wsLink,
  httpLink
);

// Apollo Client
export const client = new ApolloClient({
  link: splitLink,
  cache: new InMemoryCache(),
  defaultOptions: {
    watchQuery: {
      fetchPolicy: 'network-only',
    },
  },
});
```

### Step 5: Wrap App with Apollo Provider

In `src/App.tsx` or `src/index.tsx`:

```typescript
import { ApolloProvider } from '@apollo/client';
import { client } from './lib/apolloClient';
import PortfolioAnalysisDashboard from './components/PortfolioAnalysisDashboard';

function App() {
  return (
    <ApolloProvider client={client}>
      <PortfolioAnalysisDashboard />
    </ApolloProvider>
  );
}

export default App;
```

### Step 6: Update Component to Use GraphQL

Enhance the component to fetch data from Hasura:

```typescript
import React, { useState, useEffect } from 'react';
import { useQuery, useSubscription, gql } from '@apollo/client';
import { LineChart, Line, BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, PieChart, Pie, Cell } from 'recharts';
import { TrendingUp, RefreshCw, AlertCircle, DollarSign, Target } from 'lucide-react';

// GraphQL Query
const GET_PORTFOLIO = gql`
  query GetPortfolio($portfolioId: uuid!) {
    portfolios_by_pk(id: $portfolioId) {
      id
      name
      total_value
      target_allocation
      current_allocation
      holdings {
        id
        ticker
        shares
        current_price
        allocation_pct
        target_pct
      }
      recommendations(where: { status: { _eq: "pending" } }) {
        id
        type
        priority
        title
        expected_benefit
        recommended_actions
      }
    }
  }
`;

// GraphQL Subscription for real-time updates
const PORTFOLIO_UPDATES = gql`
  subscription PortfolioUpdates($portfolioId: uuid!) {
    portfolios_by_pk(id: $portfolioId) {
      id
      total_value
      current_allocation
      holdings {
        id
        ticker
        shares
        current_price
        allocation_pct
      }
    }
  }
`;

export default function PortfolioManager() {
  const portfolioId = 'portfolio-1'; // Get from route params or context
  
  const { data, loading, error } = useQuery(GET_PORTFOLIO, {
    variables: { portfolioId },
  });

  // Subscribe to real-time updates
  const { data: updateData } = useSubscription(PORTFOLIO_UPDATES, {
    variables: { portfolioId },
  });

  const portfolio = updateData?.portfolios_by_pk || data?.portfolios_by_pk;

  if (loading) return <div className="p-8 text-center text-gray-400">Loading...</div>;
  if (error) return <div className="p-8 text-center text-red-400">Error: {error.message}</div>;
  if (!portfolio) return <div className="p-8 text-center text-gray-400">No portfolio found</div>;

  // ... rest of component
}
```

## 🔌 API Integration

### Query Portfolios with Recommendations

```typescript
const GET_PORTFOLIOS_WITH_RECOMMENDATIONS = gql`
  query GetPortfolios {
    portfolios {
      id
      name
      total_value
      target_allocation
      current_allocation
      holdings_aggregate {
        aggregate {
          count
        }
      }
      recommendations(where: { status: { _eq: "pending" } }) {
        id
        type
        priority
        title
      }
    }
  }
`;
```

### Execute Recommendation

```typescript
const EXECUTE_RECOMMENDATION = gql`
  mutation ExecuteRecommendation($recommendationId: uuid!) {
    executeRecommendation(recommendation_id: $recommendationId) {
      order_id
      status
      tax_savings
      execution_time_ms
    }
  }
`;
```

### Subscribe to Real-time Notifications

```typescript
const NOTIFICATION_SUBSCRIPTION = gql`
  subscription NotificationUpdates($userId: uuid!) {
    notifications(
      where: { user_id: { _eq: $userId } }
      order_by: { created_at: desc }
      limit: 10
    ) {
      id
      subject
      message
      priority
      type
      created_at
    }
  }
`;
```

## 📱 Component Props

The component doesn't require props in its current form, but you can enhance it to accept:

```typescript
interface PortfolioManagerProps {
  portfolioId: string;
  userId: string;
  onRebalanceComplete?: (result: ExecutionResult) => void;
  onRecommendationExecuted?: (recommendation: Recommendation) => void;
}
```

## 🔄 Data Flow

```
┌─────────────────────────────────┐
│   React Component Mounts        │
└────────────┬────────────────────┘
             │
             ▼
┌─────────────────────────────────┐
│   useQuery(GET_PORTFOLIO)       │
│   - Fetch portfolio data        │
│   - Get holdings & metrics      │
│   - Get recommendations         │
└────────────┬────────────────────┘
             │
             ▼
┌─────────────────────────────────┐
│   useSubscription(UPDATES)      │
│   - Listen for real-time updates│
│   - Portfolio value changes     │
│   - New holdings/allocations    │
└────────────┬────────────────────┘
             │
             ▼
┌─────────────────────────────────┐
│   Component Renders              │
│   - Charts & tables             │
│   - Real-time updates           │
│   - User interactions           │
└────────────┬────────────────────┘
             │
             ▼
┌─────────────────────────────────┐
│   User Actions                   │
│   - Click "Execute Rebalance"   │
│   - Select recommendation       │
└────────────┬────────────────────┘
             │
             ▼
┌─────────────────────────────────┐
│   useMutation(EXECUTE)          │
│   - Send to backend             │
│   - Trigger workflow            │
│   - Update state                │
└────────────┬────────────────────┘
             │
             ▼
┌─────────────────────────────────┐
│   Workflow Execution             │
│   - Market data fetch           │
│   - Calculate rebalance         │
│   - Execute orders              │
│   - Update portfolio            │
└─────────────────────────────────┘
```

## 🎨 Styling Notes

The component uses Tailwind CSS with:
- Dark theme: `bg-gray-900`, `text-white`
- Gradients: `bg-gradient-to-r`
- Backdrop blur: `backdrop-blur`
- Hover states for interactivity

Make sure Tailwind CSS is configured in your project:

```bash
npm install -D tailwindcss postcss autoprefixer
npx tailwindcss init -p
```

## 🧪 Testing

Test the integration:

```bash
# 1. Start backend services
cd portfolio-management/docker
docker-compose up -d

# 2. Verify services are running
docker-compose ps

# 3. Test GraphQL endpoint
curl -X POST http://localhost:8080/v1/graphql \
  -H "Content-Type: application/json" \
  -H "X-Hasura-Admin-Secret: admin_secret_key" \
  -d '{"query": "{ portfolios { id name total_value } }"}'

# 4. Start frontend
npm start

# 5. Test component loads
# Navigate to http://localhost:3000
```

## 🐛 Troubleshooting

### GraphQL Connection Failed
```
Error: "Failed to fetch from GraphQL endpoint"
```

**Solution**:
- Verify Hasura is running: `docker-compose logs hasura`
- Check `VITE_API_BASE_URL` environment variable
- Ensure admin secret is correct
- Check CORS settings in Hasura

### Subscriptions not updating
```
No real-time updates appearing
```

**Solution**:
- Verify WebSocket URL is correct (change `http` to `ws`)
- Check browser console for WebSocket errors
- Ensure Hasura has WebSocket transport enabled
- Check firewall rules

### Data not loading
```
Query returns empty results
```

**Solution**:
- Verify database is initialized
- Check sample data exists: `docker-compose exec postgres psql -U portfolio portfolio_db -c "SELECT * FROM portfolios"`
- Verify Hasura has correct database credentials
- Check Hasura permissions

## 📚 References

- [Apollo Client Docs](https://www.apollographql.com/docs/react/)
- [GraphQL Subscriptions](https://www.apollographql.com/docs/react/data/subscriptions)
- [Hasura Subscriptions](https://hasura.io/docs/latest/subscriptions/postgres/index/)
- [Tailwind CSS](https://tailwindcss.com)
- [Recharts Documentation](https://recharts.org)

## ✅ Integration Checklist

- [ ] Component file copied to project
- [ ] Dependencies installed (`recharts`, `lucide-react`)
- [ ] Apollo Client configured
- [ ] Apollo Provider wraps app
- [ ] Environment variables set
- [ ] Backend services running
- [ ] GraphQL endpoint accessible
- [ ] Component renders without errors
- [ ] Data fetches successfully
- [ ] Real-time subscriptions working
- [ ] User interactions functional
- [ ] Tests passing

---

**Location**: `/Users/eganpj/GitHub/semlayer/portfolio-management/INTEGRATION_GUIDE.md`
**Version**: 1.0.0
