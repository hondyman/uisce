import http from 'k6/http';
import { check, group } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const duration = new Trend('duration');
const successCounter = new Counter('successes');

// Configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const JWT_TOKEN = __ENV.JWT_TOKEN || 'test-jwt-token';
const TENANT_ID = __ENV.TENANT_ID || '550e8400-e29b-41d4-a716-446655440000';

export const options = {
  // Ramp up to 50 virtual users over 5 minutes, stay for 10 minutes, ramp down over 5 minutes
  stages: [
    { duration: '5m', target: 50 },   // Ramp up
    { duration: '10m', target: 50 },  // Stay
    { duration: '5m', target: 0 },    // Ramp down
  ],
  thresholds: {
    // Thresholds for performance SLOs
    'http_req_duration': ['p(95)<500', 'p(99)<1000'], // 95% under 500ms, 99% under 1s
    'http_req_duration{url:"http://localhost:8080/api/v1/health"}': ['p(95)<100'], // Health check fast
    'errors': ['rate<0.1'],  // Error rate below 10%
  },
  // Setup phase - runs once at the start
  setupTimeout: '30s',
  // Teardown phase - runs once at the end
  teardownTimeout: '30s',
};

// Setup function - runs once
export function setup() {
  console.log('Setting up load test...');
  
  // Create test data (profile, calendar, etc.)
  const setupRes = http.post(`${BASE_URL}/api/v1/calendars`, JSON.stringify({
    name: `LoadTest-Calendar-${Date.now()}`,
    description: 'Load test calendar',
    region: 'US',
  }), {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${JWT_TOKEN}`,
      'X-Hasura-Tenant-Id': TENANT_ID,
    },
  });

  const calendarId = setupRes.json('id');
  console.log(`Created calendar: ${calendarId}`);

  return { calendarId };
}

// Teardown function - runs once at the end
export function teardown(data) {
  console.log('Tearing down load test...');
  // Cleanup can happen here if needed
}

// Default function - main test logic
export default function (data) {
  const calendarId = data.calendarId;

  group('API Health Checks', function () {
    const healthRes = http.get(`${BASE_URL}/api/v1/health`);
    check(healthRes, {
      'health check status is 200': (r) => r.status === 200,
      'health check body is ok': (r) => r.body.includes('ok'),
    }) & errorRate.add(!check(healthRes, { 'status is 200': (r) => r.status === 200 }));
    duration.add(healthRes.timings.duration);
  });

  group('Calendar Operations', function () {
    // List calendars
    let listRes = http.get(`${BASE_URL}/api/v1/calendars?limit=10&offset=0`, {
      headers: {
        'Authorization': `Bearer ${JWT_TOKEN}`,
        'X-Hasura-Tenant-Id': TENANT_ID,
      },
    });

    check(listRes, {
      'list calendars status is 200': (r) => r.status === 200,
      'list calendars returns array': (r) => Array.isArray(r.json()),
    }) & errorRate.add(!check(listRes, { 'status is 200': (r) => r.status === 200 }));
    duration.add(listRes.timings.duration);

    // Get single calendar
    if (calendarId) {
      let getRes = http.get(`${BASE_URL}/api/v1/calendars/${calendarId}`, {
        headers: {
          'Authorization': `Bearer ${JWT_TOKEN}`,
          'X-Hasura-Tenant-Id': TENANT_ID,
        },
      });

      check(getRes, {
        'get calendar status is 200': (r) => r.status === 200,
        'get calendar has id': (r) => r.json('id') === calendarId,
      }) & errorRate.add(!check(getRes, { 'status is 200': (r) => r.status === 200 }));
      duration.add(getRes.timings.duration);
    }
  });

  group('Availability Checks', function () {
    // Check availability for a time range
    const availRes = http.post(`${BASE_URL}/api/v1/availability`, JSON.stringify({
      calendar_ids: [calendarId],
      start_time: new Date().toISOString(),
      end_time: new Date(Date.now() + 86400000).toISOString(), // +1 day
      duration_minutes: 60,
    }), {
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${JWT_TOKEN}`,
        'X-Hasura-Tenant-Id': TENANT_ID,
      },
    });

    check(availRes, {
      'availability check status is 200': (r) => r.status === 200,
      'availability check returns slots': (r) => Array.isArray(r.json('available_slots')),
    }) & errorRate.add(!check(availRes, { 'status is 200': (r) => r.status === 200 }));
    duration.add(availRes.timings.duration);
  });

  group('Profile Operations', function () {
    // Create profile
    const createProfileRes = http.post(`${BASE_URL}/api/v1/profiles`, JSON.stringify({
      profile_name: `LoadTest-Profile-${Date.now()}-${Math.random()}`,
      description: 'Load test profile',
      calendars: [calendarId],
      conflict_resolution: 'union',
      timezone: 'UTC',
    }), {
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${JWT_TOKEN}`,
        'X-Hasura-Tenant-Id': TENANT_ID,
      },
    });

    check(createProfileRes, {
      'create profile status is 201': (r) => r.status === 201,
      'create profile returns id': (r) => r.json('id'),
    }) & errorRate.add(!check(createProfileRes, { 'status is 201': (r) => r.status === 201 }));
    duration.add(createProfileRes.timings.duration);

    const profileId = createProfileRes.json('id');

    if (profileId) {
      // List profiles
      let listProfileRes = http.get(`${BASE_URL}/api/v1/profiles?limit=10&offset=0`, {
        headers: {
          'Authorization': `Bearer ${JWT_TOKEN}`,
          'X-Hasura-Tenant-Id': TENANT_ID,
        },
      });

      check(listProfileRes, {
        'list profiles status is 200': (r) => r.status === 200,
        'list profiles returns array': (r) => Array.isArray(r.json()),
      }) & errorRate.add(!check(listProfileRes, { 'status is 200': (r) => r.status === 200 }));
      duration.add(listProfileRes.timings.duration);

      // Get profile
      let getProfileRes = http.get(`${BASE_URL}/api/v1/profiles/${profileId}`, {
        headers: {
          'Authorization': `Bearer ${JWT_TOKEN}`,
          'X-Hasura-Tenant-Id': TENANT_ID,
        },
      });

      check(getProfileRes, {
        'get profile status is 200': (r) => r.status === 200,
        'get profile has id': (r) => r.json('id') === profileId,
      }) & errorRate.add(!check(getProfileRes, { 'status is 200': (r) => r.status === 200 }));
      duration.add(getProfileRes.timings.duration);

      // Update profile
      let updateRes = http.put(`${BASE_URL}/api/v1/profiles/${profileId}`, JSON.stringify({
        description: 'Updated profile description',
        timezone: 'America/New_York',
      }), {
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${JWT_TOKEN}`,
          'X-Hasura-Tenant-Id': TENANT_ID,
        },
      });

      check(updateRes, {
        'update profile status is 200': (r) => r.status === 200,
      }) & errorRate.add(!check(updateRes, { 'status is 200': (r) => r.status === 200 }));
      duration.add(updateRes.timings.duration);
    }
  });

  group('External Sync Operations', function () {
    // Create sync config
    const createSyncRes = http.post(`${BASE_URL}/api/v1/external-sync`, JSON.stringify({
      profile_id: data.calendarId, // Using calendar ID as placeholder
      provider: 'nager_date',
      country_code: 'US',
      sync_enabled: true,
      sync_frequency: 'monthly',
    }), {
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${JWT_TOKEN}`,
        'X-Hasura-Tenant-Id': TENANT_ID,
      },
    });

    check(createSyncRes, {
      'create sync config status is 201': (r) => r.status === 201,
      'create sync config returns id': (r) => r.json('id'),
    }) & errorRate.add(!check(createSyncRes, { 'status is 201': (r) => r.status === 201 }));
    duration.add(createSyncRes.timings.duration);

    const syncId = createSyncRes.json('id');

    if (syncId) {
      // List sync configs
      let listSyncRes = http.get(`${BASE_URL}/api/v1/external-sync`, {
        headers: {
          'Authorization': `Bearer ${JWT_TOKEN}`,
          'X-Hasura-Tenant-Id': TENANT_ID,
        },
      });

      check(listSyncRes, {
        'list sync configs status is 200': (r) => r.status === 200,
        'list sync configs returns array': (r) => Array.isArray(r.json()),
      }) & errorRate.add(!check(listSyncRes, { 'status is 200': (r) => r.status === 200 }));
      duration.add(listSyncRes.timings.duration);

      // Get sync logs
      let logsRes = http.get(`${BASE_URL}/api/v1/external-sync/${syncId}/logs?limit=10&offset=0`, {
        headers: {
          'Authorization': `Bearer ${JWT_TOKEN}`,
          'X-Hasura-Tenant-Id': TENANT_ID,
        },
      });

      check(logsRes, {
        'get sync logs status is 200': (r) => r.status === 200,
      }) & errorRate.add(!check(logsRes, { 'status is 200': (r) => r.status === 200 }));
      duration.add(logsRes.timings.duration);
    }
  });

  group('Error Handling', function () {
    // Test 404
    const notFoundRes = http.get(`${BASE_URL}/api/v1/calendars/00000000-0000-0000-0000-000000000000`, {
      headers: {
        'Authorization': `Bearer ${JWT_TOKEN}`,
        'X-Hasura-Tenant-Id': TENANT_ID,
      },
    });

    check(notFoundRes, {
      'not found returns 404': (r) => r.status === 404,
    });

    // Test missing auth
    const noAuthRes = http.get(`${BASE_URL}/api/v1/calendars`);
    check(noAuthRes, {
      'missing auth returns 401': (r) => r.status === 401 || r.status === 403,
    });

    // Test invalid tenant
    const crossTenantRes = http.get(`${BASE_URL}/api/v1/calendars`, {
      headers: {
        'Authorization': `Bearer ${JWT_TOKEN}`,
        'X-Hasura-Tenant-Id': '00000000-0000-0000-0000-000000000000',
      },
    });

    check(crossTenantRes, {
      'invalid tenant handling': (r) => r.status === 401 || r.status === 403 || r.status === 404,
    });
  });

  successCounter.add(1);
}

// Spike test scenario
export function spike() {
  const spikeRes = http.get(`${BASE_URL}/api/v1/health`);
  check(spikeRes, {
    'spike test health check': (r) => r.status === 200,
  });
}
