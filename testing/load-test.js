import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const conversationDuration = new Trend('conversation_duration');

// Test configuration
export const options = {
  scenarios: {
    // Single-turn spike test
    single_turn_spike: {
      executor: 'ramping-vus',
      stages: [
        { duration: '30s', target: 10 },
        { duration: '1m', target: 100 },
        { duration: '2m', target: 500 },
        { duration: '1m', target: 1000 },
        { duration: '30s', target: 0 },
      ],
      tags: { test_type: 'single_turn' },
    },

    // Multi-turn conversation test
    multi_turn_conversation: {
      executor: 'constant-vus',
      vus: 50,
      duration: '10m',
      tags: { test_type: 'multi_turn' },
    },

    // Endurance test
    endurance_test: {
      executor: 'constant-vus',
      vus: 100,
      duration: '24h',
      tags: { test_type: 'endurance' },
    },

    // Adverse conditions test
    adverse_conditions: {
      executor: 'ramping-vus',
      stages: [
        { duration: '1m', target: 200 },
        { duration: '5m', target: 200 },
        { duration: '1m', target: 0 },
      ],
      tags: { test_type: 'adverse' },
    },
  },

  thresholds: {
    // Performance gates
    http_req_duration: ['p(95)<500', 'p(99)<1000'],
    http_req_failed: ['rate<0.1'],

    // Custom metrics
    errors: ['rate<0.05'],
    conversation_duration: ['p(95)<3000'],
  },
};

// Test data
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TENANTS = ['tenant1', 'tenant2', 'tenant3'];
const PRODUCTS = ['product_a', 'product_b', 'product_c'];
const DATASOURCES = ['ds1', 'ds2', 'ds3'];

// Sample queries for different scenarios
const QUERIES = {
  simple: [
    'Show me sales by region',
    'What are the top 10 products?',
    'How many customers do we have?',
  ],
  complex: [
    'Show me sales trends over the last 6 months by product category',
    'What is the customer churn rate by region and time period?',
    'Compare revenue growth between Q1 and Q2 this year',
  ],
  blocked: [
    'Show me sensitive customer PII data',
    'Display all credit card information',
    'Show me employee salaries by department',
  ],
  certified: [
    'Show me approved sales metrics',
    'Display certified customer counts',
    'Show me validated revenue figures',
  ],
};

// Multi-turn conversation scenarios
const CONVERSATION_SCENARIOS = [
  {
    name: 'Sales Analysis',
    steps: [
      'Show me sales by region',
      'Now show me the top 5 regions',
      'What about the bottom 3 regions?',
      'Compare Q1 vs Q2 for the top region',
    ],
  },
  {
    name: 'Customer Analysis',
    steps: [
      'How many customers do we have?',
      'Show me customer distribution by region',
      'What is the average order value?',
      'Show me customers with orders over $1000',
    ],
  },
];

// Helper functions
function getRandomElement(array) {
  return array[Math.floor(Math.random() * array.length)];
}

function selectRandomScenario() {
  const scenarios = ['simple', 'complex', 'blocked', 'certified'];
  const weights = [0.5, 0.3, 0.1, 0.1]; // Weighted selection

  const random = Math.random();
  let cumulativeWeight = 0;

  for (let i = 0; i < scenarios.length; i++) {
    cumulativeWeight += weights[i];
    if (random <= cumulativeWeight) {
      return scenarios[i];
    }
  }

  return 'simple';
}

function buildRequestPayload(query, tenant, product, datasource) {
  return {
    query: query,
    context: {
      tenant_id: tenant,
      product_id: product,
      datasource_id: datasource,
      user_id: `user_${Math.floor(Math.random() * 1000)}`,
    },
  };
}

// Main test functions
export function singleTurnTest() {
  const tenant = getRandomElement(TENANTS);
  const product = getRandomElement(PRODUCTS);
  const datasource = getRandomElement(DATASOURCES);
  const scenario = selectRandomScenario();
  const query = getRandomElement(QUERIES[scenario]);

  const payload = buildRequestPayload(query, tenant, product, datasource);

  const response = http.post(`${BASE_URL}/api/v1/query/nl`, JSON.stringify(payload), {
    headers: {
      'Content-Type': 'application/json',
    },
  });

  const checkResult = check(response, {
    'status is 200': (r) => r.status === 200,
    'response time < 500ms': (r) => r.timings.duration < 500,
    'has valid response': (r) => r.json() !== undefined,
    'no server errors': (r) => !r.body.includes('error'),
  });

  errorRate.add(!checkResult);

  sleep(Math.random() * 2 + 0.5); // Random sleep between 0.5-2.5s
}

export function multiTurnConversation() {
  const scenario = getRandomElement(CONVERSATION_SCENARIOS);
  const tenant = getRandomElement(TENANTS);
  const product = getRandomElement(PRODUCTS);
  const datasource = getRandomElement(DATASOURCES);

  const startTime = new Date().getTime();

  for (const step of scenario.steps) {
    const payload = buildRequestPayload(step, tenant, product, datasource);

    const response = http.post(`${BASE_URL}/api/v1/query/nl`, JSON.stringify(payload), {
      headers: {
        'Content-Type': 'application/json',
      },
    });

    const checkResult = check(response, {
      'status is 200': (r) => r.status === 200,
      'response time < 1000ms': (r) => r.timings.duration < 1000,
      'has valid response': (r) => r.json() !== undefined,
    });

    errorRate.add(!checkResult);

    // Simulate thinking time between conversation steps
    sleep(Math.random() * 3 + 1); // 1-4 seconds
  }

  const endTime = new Date().getTime();
  conversationDuration.add(endTime - startTime);
}

export function adverseConditionsTest() {
  const tenant = getRandomElement(TENANTS);
  const product = getRandomElement(PRODUCTS);
  const datasource = getRandomElement(DATASOURCES);

  // Mix of blocked and complex queries to test guardrails
  const queries = [
    ...QUERIES.blocked.slice(0, 2),
    ...QUERIES.complex.slice(0, 3),
  ];

  for (const query of queries) {
    const payload = buildRequestPayload(query, tenant, product, datasource);

    const response = http.post(`${BASE_URL}/api/v1/query/nl`, JSON.stringify(payload), {
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // In adverse conditions, we expect some requests to be blocked/denied
    const checkResult = check(response, {
      'status is valid': (r) => [200, 403, 422].includes(r.status),
      'response time < 2000ms': (r) => r.timings.duration < 2000,
      'has response': (r) => r.body.length > 0,
    });

    errorRate.add(!checkResult);
    sleep(Math.random() * 1 + 0.5);
  }
}

// Default function - runs based on scenario
export default function () {
  const scenario = __ENV.K6_SCENARIO || 'single_turn_spike';

  switch (scenario) {
    case 'multi_turn_conversation':
      multiTurnConversation();
      break;
    case 'adverse_conditions':
      adverseConditionsTest();
      break;
    default:
      singleTurnTest();
  }
}

// Setup and teardown
export function setup() {
  console.log('🚀 Starting SemLayer load test');
  console.log(`Target URL: ${BASE_URL}`);
  console.log(`Scenario: ${__ENV.K6_SCENARIO || 'single_turn_spike'}`);

  // Warm-up request
  const warmupResponse = http.get(`${BASE_URL}/health`);
  if (warmupResponse.status !== 200) {
    console.error('❌ Warm-up failed - service may not be ready');
    return;
  }

  console.log('✅ Warm-up successful');
}

export function teardown() {
  console.log('🏁 Load test completed');
}
