/**
 * RLS (Row-Level Security) Validation Test
 * Ensures tenant isolation under various attack scenarios
 */

import http from 'k6/http';
import { check } from 'k6';
import { Rate } from 'k6/metrics';

const BACKEND_URL = __ENV.BACKEND_URL || 'http://host.docker.internal:8081';
const HASURA_URL = __ENV.HASURA_URL || 'http://host.docker.internal:8080';

// Custom metrics
const rlsBypassRate = new Rate('rls_bypass_attempts');
const dataLeakRate = new Rate('data_leak_detected');
const authBypassRate = new Rate('auth_bypass_attempts');

// Test tenants
const tenants = [
  { id: '550e8400-e29b-41d4-a716-446655440001', userId: 'user-1', token: 'token-1' },
  { id: '550e8400-e29b-41d4-a716-446655440002', userId: 'user-2', token: 'token-2' },
  { id: '550e8400-e29b-41d4-a716-446655440003', userId: 'user-3', token: 'token-3' },
];

export const options = {
  vus: 10,
  duration: '5m',
  thresholds: {
    rls_bypass_attempts: ['rate==0'],    // NO bypass attempts should succeed
    data_leak_detected: ['rate==0'],     // NO data leaks allowed
    auth_bypass_attempts: ['rate==0'],   // NO auth bypasses
  },
};

export default function () {
  // Test 1: Cross-tenant data access attempts
  testCrossTenantAccess();
  
  // Test 2: SQL injection in tenant ID header
  testSQLInjectionInTenantHeader();
  
  // Test 3: Missing tenant header
  testMissingTenantHeader();
  
  // Test 4: Invalid tenant ID formats
  testInvalidTenantIDFormats();
  
  // Test 5: Privilege escalation attempts
  testPrivilegeEscalation();
  
  // Test 6: Concurrent cross-tenant requests
  testConcurrentCrossTenantRequests();
}

function testCrossTenantAccess() {
  const tenant1 = tenants[0];
  const tenant2 = tenants[1];
  
  const headers = {
    'Content-Type': 'application/json',
    'X-Hasura-Tenant-Id': tenant2.id,
    'X-Hasura-User-Id': tenant1.userId,
    'Authorization': `Bearer ${tenant1.token}`,
  };
  
  const res = http.get(`${BACKEND_URL}/api/v1/calendars`, { headers });
  
  const blocked = res.status === 403 || 
                  (res.status === 200 && JSON.parse(res.body).data?.length === 0);
  
  check(res, {
    'cross-tenant access blocked': () => blocked,
  });
  
  rlsBypassRate.add(!blocked);
  
  if (!blocked) {
    dataLeakRate.add(1);
    console.error(`DATA LEAK: Tenant ${tenant1.id} accessed Tenant ${tenant2.id} data!`);
  }
}

function testSQLInjectionInTenantHeader() {
  const tenant = tenants[0];
  
  const injectionAttempts = [
    "'; DROP TABLE calendars; --",
    "1' OR '1'='1",
    "1; SELECT * FROM users; --",
    "../../../etc/passwd",
    "<script>alert('xss')</script>",
  ];
  
  for (const injection of injectionAttempts) {
    const headers = {
      'Content-Type': 'application/json',
      'X-Hasura-Tenant-Id': injection,
      'X-Hasura-User-Id': tenant.userId,
    };
    
    const res = http.get(`${BACKEND_URL}/api/v1/calendars`, { headers });
    
    const blocked = res.status === 400 || res.status === 403 || res.status === 401;
    
    check(res, {
      [`SQL injection blocked: ${injection.substring(0, 20)}`]: () => blocked,
    });
    
    rlsBypassRate.add(!blocked);
  }
}

function testMissingTenantHeader() {
  const tenant = tenants[0];
  
  const headers = {
    'Content-Type': 'application/json',
    'X-Hasura-User-Id': tenant.userId,
  };
  
  const res = http.get(`${BACKEND_URL}/api/v1/calendars`, { headers });
  
  const blocked = res.status === 400 || res.status === 401;
  
  check(res, {
    'missing tenant header blocked': () => blocked,
  });
  
  rlsBypassRate.add(!blocked);
}

function testInvalidTenantIDFormats() {
  const tenant = tenants[0];
  
  const invalidIDs = [
    'not-a-uuid',
    '12345',
    '',
    'null',
    'undefined',
    '550e8400-e29b-41d4-a716-446655440001-EXTRA',
  ];
  
  for (const invalidID of invalidIDs) {
    const headers = {
      'Content-Type': 'application/json',
      'X-Hasura-Tenant-Id': invalidID,
      'X-Hasura-User-Id': tenant.userId,
    };
    
    const res = http.get(`${BACKEND_URL}/api/v1/calendars`, { headers });
    
    const blocked = res.status === 400 || res.status === 403;
    
    check(res, {
      [`invalid tenant ID blocked: ${invalidID}`]: () => blocked,
    });
    
    rlsBypassRate.add(!blocked);
  }
}

function testPrivilegeEscalation() {
  const tenant = tenants[0];
  
  const headers = {
    'Content-Type': 'application/json',
    'X-Hasura-Tenant-Id': tenant.id,
    'X-Hasura-User-Id': tenant.userId,
    'X-Hasura-Role': 'user',
  };
  
  const adminEndpoints = [
    '/api/v1/admin/stats',
    '/api/v1/admin/users',
    '/api/v1/admin/health',
  ];
  
  for (const endpoint of adminEndpoints) {
    const res = http.get(`${BACKEND_URL}${endpoint}`, { headers });
    
    const blocked = res.status === 403;
    
    check(res, {
      [`privilege escalation blocked: ${endpoint}`]: () => blocked,
    });
    
    authBypassRate.add(!blocked);
  }
}

function testConcurrentCrossTenantRequests() {
  for (let i = 0; i < 5; i++) {
    const tenant1 = tenants[i % tenants.length];
    const tenant2 = tenants[(i + 1) % tenants.length];
    
    const headers = {
      'Content-Type': 'application/json',
      'X-Hasura-Tenant-Id': tenant2.id,
      'X-Hasura-User-Id': tenant1.userId,
      'Authorization': `Bearer ${tenant1.token}`,
    };
    
    const res = http.get(`${BACKEND_URL}/api/v1/calendars`, { headers });
    
    const blocked = res.status === 403 || 
                    (res.status === 200 && JSON.parse(res.body).data?.length === 0);
    
    check(res, {
      [`concurrent cross-tenant blocked (iteration ${i})`]: () => blocked,
    });
    
    rlsBypassRate.add(!blocked);
  }
}
