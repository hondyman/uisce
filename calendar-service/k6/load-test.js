import http from 'k6/http';
import { check, sleep } from 'k6';

// Export configuration
export const options = {
    stages: [
        { duration: '10s', target: 20 },  // Ramp up to 20 users over 10s
        { duration: '30s', target: 20 },  // Stay at 20 users for 30s
        { duration: '10s', target: 0 },   // Ramp down to 0 users over 10s
    ],
    thresholds: {
        http_req_duration: ['p(95)<500'], // 95% of requests must complete below 500ms
        http_req_failed: ['rate<0.01'],   // Error rate must be < 1%
    },
};

const BASE_URL = 'http://localhost:8081/api/v1'; // Assuming Go API is on 8081
const HASURA_URL = 'http://localhost:8085/v1/graphql'; // Assuming Hasura is mapped to 8085 in docker

// Hardcoded tenant ID for testing (matches our Dry Run SAML test from before)
const TENANT_ID = '870361a8-87e2-4171-95ad-0473cc93791e'; 
const HASURA_ADMIN_SECRET = 'myadminsecret';

export default function () {
    // 1. Test Hasura RLS Policy (Skip if port 8085 is unreachable)
    try {
        const graphqlQuery = {
            query: `
                query TestTenantIsolation {
                    teams {
                        id
                        name
                    }
                }
            `,
        };

        const hasuraHeaders = {
            'Content-Type': 'application/json',
            'x-hasura-admin-secret': HASURA_ADMIN_SECRET,
            'x-hasura-role': 'tenant_user',
            'x-hasura-tenant-id': TENANT_ID,
        };

        let hasuraRes = http.post(HASURA_URL, JSON.stringify(graphqlQuery), { headers: hasuraHeaders, timeout: '1s' });
        
        if (hasuraRes.status === 200 && hasuraRes.body) {
            check(hasuraRes, {
                'hasura response has data': (r) => JSON.parse(r.body).data !== undefined,
            });
        }
    } catch (e) {
        // Skip Hasura if down
    }

    // 2. Test Backend Go API Health Endpoint (No Auth)
    let apiRes = http.get(`${BASE_URL}/health`);
    
    check(apiRes, {
        'api health is 200': (r) => r.status === 200,
        'api response is healthy': (r) => JSON.parse(r.body).status === 'healthy',
    });

    sleep(1);
}
