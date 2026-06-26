# Frontend Code Update Guide: Consolidated Metrics & DAX Functions

**Date:** November 3, 2025  
**Purpose:** Update React/TypeScript frontend to work with consolidated public schema tables

---

## 🎯 Overview

The frontend consumes metrics and DAX function data through REST API endpoints. When the backend switches to querying consolidated tables with the `schema_domain` filter, the frontend needs minimal changes since the API contract can remain the same.

**Key Point:** If your API layer abstracts the schema selection, frontend components don't need to change!

---

## 📋 Prerequisites

- React 16.8+ (with hooks)
- TypeScript 4.0+
- API layer in place (Node.js/Express backend)
- Axios or Fetch API for HTTP calls
- Understanding of how your API routes bundle/metric endpoints

---

## 🔧 Step 1: Understand Current Data Flow

### Before Consolidation

```typescript
// Frontend component fetches from domain-specific endpoint
const MetricsDisplay = () => {
  const [metrics, setMetrics] = useState([]);

  useEffect(() => {
    // API calls domain-specific backend
    fetch('/api/metrics/banking')  // Uses banking schema
      .then(res => res.json())
      .then(data => setMetrics(data));
  }, []);

  return <div>{/* Display metrics */}</div>;
};
```

### After Consolidation (No Change Needed!)

```typescript
// If API abstraction is good, this stays THE SAME
const MetricsDisplay = () => {
  const [metrics, setMetrics] = useState([]);

  useEffect(() => {
    // API still looks the same to frontend!
    fetch('/api/metrics/banking')  // Queries public schema behind the scenes
      .then(res => res.json())
      .then(data => setMetrics(data));
  }, []);

  return <div>{/* Display metrics */}</div>;
};
```

---

## 🔄 Step 2: Update API Service Layer (if needed)

### Create Consolidated Metrics Service

```typescript
// frontend/src/services/metricsService.ts

interface Metric {
  id: number;
  nodeId: string;
  schemaDomain: string;  // New: track which domain
  category: string;
  description: string;
  formulaType: string;
  formula: string;
  arguments?: Record<string, any>;
  badge?: string;
  functionClass?: string;
  functionsUsed?: string[];
  governanceStatus?: string;
  audience?: string[];
  tags?: string[];
  createdAt: string;
  updatedAt: string;
}

interface DAXFunction {
  id: number;
  name: string;
  schemaDomain: string;  // New: track which domain
  class: string;
  badge?: string;
  description?: string;
  createdAt: string;
}

export class MetricsService {
  private baseURL = (import.meta.env.VITE_API_BASE_URL as string) || 'http://localhost:8080';

  /**
   * Get metrics for a specific domain
   * @param domain - Domain name (banking, retail, etc.)
   */
  async getMetricsByDomain(domain: string): Promise<Metric[]> {
    const response = await fetch(
      `${this.baseURL}/api/metrics?domain=${domain}`
    );
    if (!response.ok) {
      throw new Error(`Failed to fetch metrics: ${response.statusText}`);
    }
    return response.json();
  }

  /**
   * Get metrics for multiple domains
   * @param domains - Array of domain names
   */
  async getMetricsByDomains(domains: string[]): Promise<Metric[]> {
    const params = new URLSearchParams();
    domains.forEach(d => params.append('domains', d));
    
    const response = await fetch(
      `${this.baseURL}/api/metrics?${params.toString()}`
    );
    if (!response.ok) {
      throw new Error(`Failed to fetch metrics: ${response.statusText}`);
    }
    return response.json();
  }

  /**
   * Get a specific metric by node ID and domain
   */
  async getMetricByNodeID(
    domain: string,
    nodeId: string
  ): Promise<Metric | null> {
    const response = await fetch(
      `${this.baseURL}/api/metrics/${nodeId}?domain=${domain}`
    );
    if (response.status === 404) {
      return null;
    }
    if (!response.ok) {
      throw new Error(`Failed to fetch metric: ${response.statusText}`);
    }
    return response.json();
  }

  /**
   * Get metrics by category within a domain
   */
  async getMetricsByCategory(
    domain: string,
    category: string
  ): Promise<Metric[]> {
    const response = await fetch(
      `${this.baseURL}/api/metrics?domain=${domain}&category=${category}`
    );
    if (!response.ok) {
      throw new Error(`Failed to fetch metrics: ${response.statusText}`);
    }
    return response.json();
  }

  /**
   * Get all DAX functions for a domain
   */
  async getDAXFunctionsByDomain(domain: string): Promise<DAXFunction[]> {
    const response = await fetch(
      `${this.baseURL}/api/dax-functions?domain=${domain}`
    );
    if (!response.ok) {
      throw new Error(`Failed to fetch DAX functions: ${response.statusText}`);
    }
    return response.json();
  }

  /**
   * Get DAX functions for multiple domains
   */
  async getDAXFunctionsByDomains(domains: string[]): Promise<DAXFunction[]> {
    const params = new URLSearchParams();
    domains.forEach(d => params.append('domains', d));
    
    const response = await fetch(
      `${this.baseURL}/api/dax-functions?${params.toString()}`
    );
    if (!response.ok) {
      throw new Error(`Failed to fetch DAX functions: ${response.statusText}`);
    }
    return response.json();
  }

  /**
   * Get a specific DAX function
   */
  async getDAXFunctionByName(
    domain: string,
    name: string
  ): Promise<DAXFunction | null> {
    const response = await fetch(
      `${this.baseURL}/api/dax-functions/${name}?domain=${domain}`
    );
    if (response.status === 404) {
      return null;
    }
    if (!response.ok) {
      throw new Error(`Failed to fetch DAX function: ${response.statusText}`);
    }
    return response.json();
  }
}

export const metricsService = new MetricsService();
```

---

## 🎣 Step 3: Update React Hooks

### Create Custom Hook for Metrics

```typescript
// frontend/src/hooks/useMetrics.ts

import { useState, useEffect, useCallback } from 'react';
import { metricsService, type Metric } from '../services/metricsService';

interface UseMetricsResult {
  metrics: Metric[];
  loading: boolean;
  error: Error | null;
  refetch: () => Promise<void>;
}

export function useMetrics(domain: string): UseMetricsResult {
  const [metrics, setMetrics] = useState<Metric[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const fetchMetrics = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await metricsService.getMetricsByDomain(domain);
      setMetrics(data);
    } catch (err) {
      setError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setLoading(false);
    }
  }, [domain]);

  useEffect(() => {
    fetchMetrics();
  }, [fetchMetrics]);

  return { metrics, loading, error, refetch: fetchMetrics };
}

export function useMetricsMultipleDomains(
  domains: string[]
): UseMetricsResult {
  const [metrics, setMetrics] = useState<Metric[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const fetchMetrics = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      if (domains.length === 0) {
        setMetrics([]);
        return;
      }
      const data = await metricsService.getMetricsByDomains(domains);
      setMetrics(data);
    } catch (err) {
      setError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setLoading(false);
    }
  }, [domains]);

  useEffect(() => {
    fetchMetrics();
  }, [fetchMetrics]);

  return { metrics, loading, error, refetch: fetchMetrics };
}
```

---

## 📊 Step 4: Update Components

### Example: Metrics List Component

```typescript
// frontend/src/components/MetricsList.tsx

import React from 'react';
import { useMetrics } from '../hooks/useMetrics';
import { Metric } from '../services/metricsService';

interface MetricsListProps {
  domain: string;
  onSelectMetric?: (metric: Metric) => void;
}

export const MetricsList: React.FC<MetricsListProps> = ({
  domain,
  onSelectMetric,
}) => {
  const { metrics, loading, error, refetch } = useMetrics(domain);

  if (loading) {
    return <div>Loading metrics...</div>;
  }

  if (error) {
    return (
      <div style={{ color: 'red' }}>
        Error loading metrics: {error.message}
        <button onClick={refetch}>Retry</button>
      </div>
    );
  }

  if (metrics.length === 0) {
    return <div>No metrics found for domain: {domain}</div>;
  }

  return (
    <div>
      <h2>Metrics for {domain}</h2>
      <table>
        <thead>
          <tr>
            <th>Node ID</th>
            <th>Category</th>
            <th>Description</th>
            <th>Formula Type</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {metrics.map((metric) => (
            <tr key={`${metric.schemaDomain}-${metric.nodeId}`}>
              <td>{metric.nodeId}</td>
              <td>{metric.category}</td>
              <td>{metric.description}</td>
              <td>{metric.formulaType}</td>
              <td>
                {onSelectMetric && (
                  <button onClick={() => onSelectMetric(metric)}>
                    View Details
                  </button>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};
```

### Example: Multi-Domain Metrics Component

```typescript
// frontend/src/components/MultiDomainMetrics.tsx

import React, { useState } from 'react';
import { useMetricsMultipleDomains } from '../hooks/useMetrics';
import { Metric } from '../services/metricsService';

interface MultiDomainMetricsProps {
  domains: string[];
}

export const MultiDomainMetrics: React.FC<MultiDomainMetricsProps> = ({
  domains,
}) => {
  const [selectedCategory, setSelectedCategory] = useState<string | null>(null);
  const { metrics, loading, error } = useMetricsMultipleDomains(domains);

  if (loading) {
    return <div>Loading metrics from {domains.length} domains...</div>;
  }

  if (error) {
    return <div style={{ color: 'red' }}>Error: {error.message}</div>;
  }

  // Group by domain
  const groupedByDomain = metrics.reduce(
    (acc, metric) => {
      if (!acc[metric.schemaDomain]) {
        acc[metric.schemaDomain] = [];
      }
      acc[metric.schemaDomain].push(metric);
      return acc;
    },
    {} as Record<string, Metric[]>
  );

  // Group by category if selected
  const filteredMetrics = selectedCategory
    ? metrics.filter((m) => m.category === selectedCategory)
    : metrics;

  const categories = Array.from(new Set(metrics.map((m) => m.category)));

  return (
    <div>
      <h2>Metrics from {domains.join(', ')}</h2>
      
      <div>
        <label>Filter by category: </label>
        <select
          value={selectedCategory || ''}
          onChange={(e) => setSelectedCategory(e.target.value || null)}
        >
          <option value="">All Categories</option>
          {categories.map((cat) => (
            <option key={cat} value={cat}>
              {cat}
            </option>
          ))}
        </select>
      </div>

      {Object.entries(groupedByDomain).map(([domain, domainMetrics]) => (
        <div key={domain} style={{ marginTop: '20px' }}>
          <h3>{domain} ({domainMetrics.length} metrics)</h3>
          <ul>
            {domainMetrics
              .filter((m) =>
                selectedCategory ? m.category === selectedCategory : true
              )
              .map((metric) => (
                <li key={metric.id}>
                  <strong>{metric.nodeId}</strong> ({metric.category})
                  <br />
                  <small>{metric.description}</small>
                </li>
              ))}
          </ul>
        </div>
      ))}

      <p>Total metrics: {filteredMetrics.length}</p>
    </div>
  );
};
```

### Example: DAX Functions Component

```typescript
// frontend/src/components/DAXFunctionsList.tsx

import React from 'react';
import { metricsService } from '../services/metricsService';

interface DAXFunctionsListProps {
  domain: string;
}

export const DAXFunctionsList: React.FC<DAXFunctionsListProps> = ({
  domain,
}) => {
  const [functions, setFunctions] = React.useState([]);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<Error | null>(null);

  React.useEffect(() => {
    const fetchFunctions = async () => {
      try {
        setLoading(true);
        const data = await metricsService.getDAXFunctionsByDomain(domain);
        setFunctions(data);
      } catch (err) {
        setError(err instanceof Error ? err : new Error(String(err)));
      } finally {
        setLoading(false);
      }
    };

    fetchFunctions();
  }, [domain]);

  if (loading) {
    return <div>Loading DAX functions...</div>;
  }

  if (error) {
    return <div style={{ color: 'red' }}>Error: {error.message}</div>;
  }

  return (
    <div>
      <h2>DAX Functions for {domain}</h2>
      {functions.length === 0 ? (
        <p>No DAX functions found</p>
      ) : (
        <table>
          <thead>
            <tr>
              <th>Name</th>
              <th>Class</th>
              <th>Badge</th>
              <th>Description</th>
            </tr>
          </thead>
          <tbody>
            {functions.map((fn: any) => (
              <tr key={fn.id}>
                <td>{fn.name}</td>
                <td>{fn.class}</td>
                <td>{fn.badge || '-'}</td>
                <td>{fn.description || '-'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
};
```

---

## 🧪 Step 5: Update Tests

### Jest + React Testing Library

```typescript
// frontend/src/components/__tests__/MetricsList.test.tsx

import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { MetricsList } from '../MetricsList';
import * as metricsService from '../../services/metricsService';

jest.mock('../../services/metricsService');

describe('MetricsList', () => {
  it('displays loading state initially', () => {
    (metricsService.metricsService.getMetricsByDomain as jest.Mock).mockImplementation(
      () => new Promise(() => {}) // Never resolves
    );

    render(<MetricsList domain="banking" />);
    expect(screen.getByText(/loading metrics/i)).toBeInTheDocument();
  });

  it('displays metrics after loading', async () => {
    const mockMetrics = [
      {
        id: 1,
        nodeId: 'METRIC_001',
        schemaDomain: 'banking',
        category: 'performance',
        description: 'Return on Assets',
        formulaType: 'dax',
        formula: 'DIVIDE(...)',
        createdAt: '2025-01-01',
        updatedAt: '2025-01-01',
      },
      {
        id: 2,
        nodeId: 'METRIC_002',
        schemaDomain: 'banking',
        category: 'risk',
        description: 'Value at Risk',
        formulaType: 'sql',
        formula: 'SELECT ...',
        createdAt: '2025-01-01',
        updatedAt: '2025-01-01',
      },
    ];

    (metricsService.metricsService.getMetricsByDomain as jest.Mock).mockResolvedValue(
      mockMetrics
    );

    render(<MetricsList domain="banking" />);

    await waitFor(() => {
      expect(screen.getByText('METRIC_001')).toBeInTheDocument();
      expect(screen.getByText('METRIC_002')).toBeInTheDocument();
    });
  });

  it('displays error message on fetch failure', async () => {
    const error = new Error('API Error');
    (metricsService.metricsService.getMetricsByDomain as jest.Mock).mockRejectedValue(
      error
    );

    render(<MetricsList domain="banking" />);

    await waitFor(() => {
      expect(screen.getByText(/error loading metrics/i)).toBeInTheDocument();
    });
  });

  it('calls onSelectMetric when button clicked', async () => {
    const mockMetrics = [
      {
        id: 1,
        nodeId: 'METRIC_001',
        schemaDomain: 'banking',
        category: 'performance',
        description: 'Return on Assets',
        formulaType: 'dax',
        formula: 'DIVIDE(...)',
        createdAt: '2025-01-01',
        updatedAt: '2025-01-01',
      },
    ];

    (metricsService.metricsService.getMetricsByDomain as jest.Mock).mockResolvedValue(
      mockMetrics
    );

    const onSelectMetric = jest.fn();
    render(
      <MetricsList domain="banking" onSelectMetric={onSelectMetric} />
    );

    await waitFor(() => {
      const button = screen.getByText('View Details');
      button.click();
      expect(onSelectMetric).toHaveBeenCalledWith(mockMetrics[0]);
    });
  });
});
```

---

## 🔗 Step 6: Integrate with Apollo Client (if using GraphQL)

### If using Hasura GraphQL

```typescript
// frontend/src/apolloClient.ts

import { ApolloClient, InMemoryCache, HttpLink } from '@apollo/client';

const httpLink = new HttpLink({
  uri: (import.meta.env.VITE_HASURA_ENDPOINT as string) || 'http://localhost:8080/v1/graphql',
  credentials: 'include',
  headers: {
    'X-Hasura-Admin-Secret': (import.meta.env.VITE_HASURA_SECRET as string),
  },
});

export const client = new ApolloClient({
  link: httpLink,
  cache: new InMemoryCache(),
});
```

### GraphQL Query

```typescript
// frontend/src/graphql/queries.ts

import { gql } from '@apollo/client';

export const GET_METRICS_BY_DOMAIN = gql`
  query GetMetricsByDomain($domain: String!) {
    public_metrics_registry(
      where: { schema_domain: { _eq: $domain } }
      order_by: { node_id: asc }
    ) {
      id
      node_id
      schema_domain
      category
      description
      formula_type
      formula
      arguments
      badge
      function_class
      functions_used
      governance_status
      audience
      tags
      created_at
      updated_at
    }
  }
`;

export const GET_DAX_FUNCTIONS_BY_DOMAIN = gql`
  query GetDAXFunctionsByDomain($domain: String!) {
    public_dax_functions(
      where: { schema_domain: { _eq: $domain } }
      order_by: { name: asc }
    ) {
      id
      name
      schema_domain
      class
      badge
      description
      created_at
    }
  }
`;
```

### Apollo Hook

```typescript
// frontend/src/hooks/useMetricsGraphQL.ts

import { useQuery } from '@apollo/client';
import { GET_METRICS_BY_DOMAIN } from '../graphql/queries';

export function useMetricsGraphQL(domain: string) {
  const { data, loading, error, refetch } = useQuery(
    GET_METRICS_BY_DOMAIN,
    {
      variables: { domain },
    }
  );

  return {
    metrics: data?.public_metrics_registry || [],
    loading,
    error,
    refetch,
  };
}
```

---

## 📋 Migration Checklist

### Update Services
- [ ] Create consolidated metrics service (`metricsService.ts`)
- [ ] Update API endpoints to handle `domain` parameter
- [ ] Add type definitions for new `schemaDomain` field
- [ ] Update error handling

### Update Hooks
- [ ] Create `useMetrics` hook for single domain
- [ ] Create `useMetricsMultipleDomains` hook for multiple domains
- [ ] Create `useDAXFunctions` hook
- [ ] Add error handling to hooks

### Update Components
- [ ] Update MetricsList component
- [ ] Update MetricsViewer component
- [ ] Update DAXFunctionReference component
- [ ] Update BundleExplorer component
- [ ] Update any dashboard components

### Update GraphQL (if applicable)
- [ ] Update GraphQL queries to filter by `schema_domain`
- [ ] Update Apollo cache configuration
- [ ] Test mutations with new schema

### Tests
- [ ] Update unit tests for services
- [ ] Update component tests
- [ ] Update integration tests
- [ ] Add tests for multi-domain queries
- [ ] Add tests for error scenarios

### Documentation
- [ ] Update API documentation
- [ ] Update component prop types
- [ ] Update examples in storybook
- [ ] Update README with new patterns

---

## 🚀 Deployment

### Before Deployment
1. Ensure backend API layer is ready and tested
2. Verify all API endpoints accept and filter by `domain` parameter
3. Confirm Hasura GraphQL is updated (if applicable)

### Deployment Steps
1. Deploy backend changes (database migration + code)
2. Wait for backend to stabilize
3. Deploy frontend changes
4. Test in staging environment
5. Monitor error logs in production

### Rollback
1. Revert frontend to previous version
2. Old API should still work with old schema structure
3. If needed, restore database from backup

---

## 📊 Performance Considerations

### Caching Strategy

```typescript
// Use React Query or SWR for better caching
import useSWR from 'swr';

const fetcher = (url: string) => fetch(url).then(r => r.json());

export function useMetricsWithCache(domain: string) {
  const { data, error, isLoading, mutate } = useSWR(
    `/api/metrics?domain=${domain}`,
    fetcher,
    {
      revalidateOnFocus: false,
      revalidateOnReconnect: false,
      dedupingInterval: 60000, // 1 minute
    }
  );

  return {
    metrics: data || [],
    loading: isLoading,
    error,
    refetch: mutate,
  };
}
```

### Lazy Loading

```typescript
import { lazy, Suspense } from 'react';

const MetricsTable = lazy(() => 
  import('./MetricsTable').then(m => ({ default: m.MetricsTable }))
);

export function LazyMetricsDisplay() {
  return (
    <Suspense fallback={<div>Loading...</div>}>
      <MetricsTable domain="banking" />
    </Suspense>
  );
}
```

---

## 🆘 Troubleshooting

### Issue: "404 Not Found" for metrics

**Solution:** Verify `domain` parameter is being passed to API:
```typescript
// Correct
fetch('/api/metrics?domain=banking')

// Incorrect (will fail)
fetch('/api/metrics')
```

### Issue: "Schema domain mismatch"

**Solution:** Ensure `schemaDomain` in frontend matches backend domain value:
```typescript
// Backend returns this
{
  node_id: 'METRIC_001',
  schema_domain: 'banking'  // Exact match required
}

// Frontend must use exact value
const metrics = metrics.filter(m => m.schemaDomain === 'banking');
```

### Issue: "Data not displaying after consolidation"

**Solution:** Check that:
1. Backend returned data with `schema_domain` field
2. Frontend is displaying `schema_domain` in tables/lists
3. API is filtering correctly

---

## 📝 Example: Complete Integration

```typescript
// frontend/src/pages/MetricsPage.tsx

import React, { useState } from 'react';
import { MetricsList } from '../components/MetricsList';
import { DAXFunctionsList } from '../components/DAXFunctionsList';
import { MultiDomainMetrics } from '../components/MultiDomainMetrics';

export const MetricsPage: React.FC = () => {
  const [selectedDomain, setSelectedDomain] = useState('banking');
  const [viewMode, setViewMode] = useState<'single' | 'multi'>('single');

  const domains = [
    'banking',
    'retail',
    'capital_markets',
    'financial_services',
    'wealth_management',
  ];

  return (
    <div style={{ padding: '20px' }}>
      <h1>Metrics & DAX Functions</h1>

      <div style={{ marginBottom: '20px' }}>
        <label>
          View Mode:
          <select value={viewMode} onChange={(e) => setViewMode(e.target.value as any)}>
            <option value="single">Single Domain</option>
            <option value="multi">Multiple Domains</option>
          </select>
        </label>
      </div>

      {viewMode === 'single' ? (
        <>
          <label>
            Select Domain:
            <select value={selectedDomain} onChange={(e) => setSelectedDomain(e.target.value)}>
              {domains.map((d) => (
                <option key={d} value={d}>
                  {d}
                </option>
              ))}
            </select>
          </label>
          <MetricsList domain={selectedDomain} />
          <DAXFunctionsList domain={selectedDomain} />
        </>
      ) : (
        <MultiDomainMetrics domains={domains} />
      )}
    </div>
  );
};
```

---

## ✅ Verification Checklist

After updating frontend code:

- [ ] All services compiled without errors
- [ ] All components render without errors
- [ ] Metrics display correctly for single domain
- [ ] Metrics display correctly for multiple domains
- [ ] DAX functions display correctly
- [ ] Filtering by category works
- [ ] Error handling works (mock API failures)
- [ ] Loading states display correctly
- [ ] API calls include `domain` parameter
- [ ] Tests pass
- [ ] No console errors in browser DevTools

---

## 📞 Support

For issues:

1. Check BACKEND_REFACTORING_GUIDE.md for backend changes
2. Check CODE_MIGRATION_GUIDE.md for integration patterns
3. Check CONSOLIDATION_PLAN.md for overall architecture
4. Review test examples above for debugging patterns

