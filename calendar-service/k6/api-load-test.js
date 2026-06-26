/**
 * Calendar Sync Platform - API Load Test
 * Tests multi-tenant API performance under load
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

// Custom metrics
const errorRate = new Rate('errors');
const apiLatency = new Trend('api_latency');
const rlsViolationRate = new Rate('rls_violations');
const cacheHitRate = new Rate('cache_hits');
const syncSuccessRate = new Rate('sync_success');

// Test configuration from environment
const BACKEND_URL = __ENV.BACKEND_URL || 'http://host.docker.internal:8081';
const HASURA_URL = __ENV.HASURA_URL || 'http://host.docker.internal:8080';
const TENANT_COUNT = parseInt(__ENV.TENANT_COUNT) || 10;

// Test tenants (simulated)
const tenants = [];
for (let i = 0; i < TENANT_COUNT; i++) {
  tenants.push({
    id: `550e8400-e29b-41d4-a716-44665544000${i}`,
    userId: `user-${i}-${uuidv4()}`,
    token: `test-token-${i}`,
  });
}

export const options = {
  stages: [
    { duration: '1m', target: 10 },   // Ramp up to 10 users
    { duration: '2m', target: 50 },   // Ramp up to 50 users
    { duration: '3m', target: 100 },  // Ramp up to 100 users (peak)
    { duration: '2m', target: 100 },  // Stay at peak
    { duration: '1m', target: 0 },    // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'],  // 95% of requests < 500ms
    errors: ['rate<0.01'],             // Error rate < 1%
    rls_violations: ['rate==0'],       // NO RLS violations allowed
    api_latency: ['p(95)<300'],        // API latency p95 < 300ms
  },
};

export default function () {
  // Select random tenant for this iteration
  const tenant = tenants[Math.floor(Math.random() * tenants.length)];
  
  // Test 1: Health check
  testHealthCheck();
  
  // Test 2: Authenticated API calls
  testAuthenticatedAPIs(tenant);
  
  // Test 3: Calendar sync operations
  testCalendarSync(tenant);
  
  // Test 4: RLS isolation validation
  testRLSIsolation(tenant);
  
  // Test 5: Cache performance
  testCachePerformance(tenant);
  
  sleep(1);
}

function testHealthCheck() {
  const res = http.get(`${BACKEND_URL}/health`);
  
  check(res, {
    'health check status is 200': (r) => r.status === 200,
    'health check latency < 100ms': (r) => r.timings.duration < 100,
  });
  
  errorRate.add(res.status !== 200);
  apiLatency.add(res.timings.duration);
}

function testAuthenticatedAPIs(tenant) {
  const headers = {
    'Content-Type': 'application/json',
    'X-Hasura-Tenant-Id': tenant.id,
    'X-Hasura-User-Id': tenant.userId,
    'Authorization': `Bearer ${tenant.token}`,
  };
  
  // Test calendars endpoint
  const calendarsRes = http.get(
    `${BACKEND_URL}/api/v1/calendars`,
    { headers }
  );
  
  check(calendarsRes, {
    'calendars status is 200': (r) => r.status === 200,
    'calendars latency < 300ms': (r) => r.timings.duration < 300,
  });
  
  errorRate.add(calendarsRes.status !== 200);
  
  // Test events endpoint
  const eventsRes = http.get(
    `${BACKEND_URL}/api/v1/events?limit=50`,
    { headers }
  );
  
  check(eventsRes, {
    'events status is 200': (r) => r.status === 200,
    'events latency < 400ms': (r) => r.timings.duration < 400,
  });
  
  errorRate.add(eventsRes.status !== 200);
  
  // Test analytics endpoint
  const analyticsRes = http.get(
    `${BACKEND_URL}/api/v1/analytics/sync?start_date=2026-01-01&end_date=2026-02-28`,
    { headers }
  );
  
  check(analyticsRes, {
    'analytics status is 200': (r) => r.status === 200,
    'analytics latency < 500ms': (r) => r.timings.duration < 500,
  });
  
  errorRate.add(analyticsRes.status !== 200);
}

function testCalendarSync(tenant) {
  const headers = {
    'Content-Type': 'application/json',
    'X-Hasura-Tenant-Id': tenant.id,
    'X-Hasura-User-Id': tenant.userId,
    'Authorization': `Bearer ${tenant.token}`,
  };
  
  // Trigger sync
  const syncRes = http.post(
    `${BACKEND_URL}/api/v1/sync/google/sync`,
    JSON.stringify({
      google_calendar_id: 'primary',
      internal_calendar_id: `cal-${tenant.userId}`,
      time_range: {
        start: '2026-02-01T00:00:00Z',
        end: '2026-03-01T00:00:00Z',
      },
    }),
    { headers }
  );
  
  check(syncRes, {
    'sync trigger status is 202': (r) => r.status === 202,
    'sync trigger latency < 500ms': (r) => r.timings.duration < 500,
  });
  
  syncSuccessRate.add(syncRes.status === 202);
  errorRate.add(syncRes.status !== 202);
}

function testRLSIsolation(tenant) {
  const headers = {
    'Content-Type': 'application/json',
    'X-Hasura-Tenant-Id': tenant.id,
    'X-Hasura-User-Id': tenant.userId,
    'Authorization': `Bearer ${tenant.token}`,
  };
  
  // Try to access another tenant's data (should fail)
  const otherTenantId = tenants.find(t => t.id !== tenant.id)?.id || '00000000-0000-0000-0000-000000000000';
  
  const crossTenantRes = http.get(
    `${BACKEND_URL}/api/v1/calendars`,
    {
      headers: {
        ...headers,
        'X-Hasura-Tenant-Id': otherTenantId, // Different tenant
      },
    }
  );
  
  // Should return 403 or empty data (RLS working)
  const rlsWorking = crossTenantRes.status === 403 || 
                     (crossTenantRes.status === 200 && 
                      JSON.parse(crossTenantRes.body).data?.length === 0);
  
  check(crossTenantRes, {
    'RLS prevents cross-tenant access': () => rlsWorking,
  });
  
  rlsViolationRate.add(!rlsWorking);
  
  if (!rlsWorking) {
    console.error(`RLS VIOLATION: Tenant ${tenant.id} accessed tenant ${otherTenantId} data!`);
  }
}

function testCachePerformance(tenant) {
  const headers = {
    'Content-Type': 'application/json',
    'X-Hasura-Tenant-Id': tenant.id,
    'X-Hasura-User-Id': tenant.userId,
  };
  
  // Make same request twice (second should be cached)
  const res1 = http.get(`${BACKEND_URL}/api/v1/calendars`, { headers });
  sleep(0.1);
  const res2 = http.get(`${BACKEND_URL}/api/v1/calendars`, { headers });
  
  // Second request should be faster (cache hit)
  const cacheHit = res2.timings.duration < res1.timings.duration * 0.5;
  
  cacheHitRate.add(cacheHit);
  
  check(res2, {
    'cache hit improves latency': () => cacheHit,
  });
}

export function handleSummary(data) {
  // Generate HTML report
  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    './k6/results/load-test-report.html': htmlReport(data),
  };
}

function textSummary(data, options) {
  const metrics = data.metrics;
  
  return `
┌─────────────────────────────────────────────────────────────────┐
│                    LOAD TEST SUMMARY                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  HTTP Requests:     ${metrics.http_reqs?.values?.count || 0}                                  │
│  Request Duration:  p(95)=${(metrics.http_req_duration?.values?.['p(95)'] || 0).toFixed(0)}ms, p(99)=${(metrics.http_req_duration?.values?.['p(99)'] || 0).toFixed(0)}ms        │
│  Error Rate:        ${(metrics.errors?.values?.rate || 0) * 100}%                                  │
│  RLS Violations:    ${(metrics.rls_violations?.values?.rate || 0) * 100}%                                  │
│  Cache Hit Rate:    ${(metrics.cache_hits?.values?.rate || 0) * 100}%                                  │
│  Sync Success Rate: ${(metrics.sync_success?.values?.rate || 0) * 100}%                                  │
│                                                                  │
├─────────────────────────────────────────────────────────────────┤
│  THRESHOLD CHECKS:                                               │
│  ${metrics.http_req_duration?.values?.['p(95)'] < 500 ? '✅' : '❌'} p95 Latency < 500ms                                     │
│  ${metrics.errors?.values?.rate < 0.01 ? '✅' : '❌'} Error Rate < 1%                                          │
│  ${metrics.rls_violations?.values?.rate === 0 ? '✅' : '❌'} RLS Violations = 0%                                       │
│  ${metrics.api_latency?.values?.['p(95)'] < 300 ? '✅' : '❌'} API Latency p95 < 300ms                                 │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
  `;
}

function htmlReport(data) {
  // Simple HTML report template
  return `<!DOCTYPE html>
<html>
<head><title>Load Test Report</title></head>
<body>
  <h1>Load Test Report</h1>
  <p>Generated: ${new Date().toISOString()}</p>
  <pre>${JSON.stringify(data.metrics, null, 2)}</pre>
</body>
</html>`;
}
