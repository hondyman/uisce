// Integration Example - How to wire Admin UI into main app

import React from "react";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { adminRoutes } from "./admin";

// Example main App component showing admin routes integration

export function App() {
  return (
    <BrowserRouter>
      <Routes>
        {/* Admin Routes */}
        {adminRoutes.map((route) => (
          <Route key={route.path} {...route} />
        ))}

        {/* Other app routes would go here */}
        {/* <Route path="/login" element={<LoginPage />} />
        <Route path="/dashboard" element={<Dashboard />} /> */}

        {/* Fallback */}
        <Route path="/" element={<Navigate to="/admin" replace />} />
      </Routes>
    </BrowserRouter>
  );
}

export default App;

/*
 * ============================================================================
 * SETUP INSTRUCTIONS
 * ============================================================================
 *
 * 1. Environment Setup (.env):
 *    REACT_APP_API_URL=http://localhost:8082/api
 *
 * 2. Install Dependencies:
 *    npm install react-router-dom
 *
 * 3. Add to App.tsx:
 *    import { adminRoutes } from "@/admin";
 *    
 *    Then include adminRoutes in your Routes:
 *    {adminRoutes.map(route => <Route key={route.path} {...route} />)}
 *
 * 4. Ensure Backend is Running:
 *    - Backend listening on port 8082
 *    - All /api/admin/* endpoints implemented
 *    - CORS configured to allow frontend origin
 *
 * 5. Authentication:
 *    - Admin UI expects JWT token in localStorage['token']
 *    - Token must contain roles: ["GLOBAL_OPS"]
 *    - Get token via login endpoint and store it
 *
 * 6. Access Admin Panel:
 *    - http://localhost:3000/admin
 *    - Or configure as your app's primary route
 *
 * ============================================================================
 * API RESPONSE CONTRACTS (Backend should follow these)
 * ============================================================================
 *
 * GET /api/admin/tenants?limit=50&offset=0
 * Response:
 * {
 *   "tenants": [
 *     {
 *       "id": "uuid",
 *       "name": "Tenant Name",
 *       "code": "tenant-code",
 *       "region": "us-east-1",
 *       "plan": "pro",
 *       "max_requests": 10000,
 *       "window_seconds": 86400,
 *       "is_suspended": false,
 *       "created_at": "2025-02-08T10:00:00Z",
 *       "updated_at": "2025-02-08T10:00:00Z"
 *     }
 *   ],
 *   "total": 25,
 *   "limit": 50,
 *   "offset": 0
 * }
 *
 * POST /api/admin/tenants
 * Request:
 * {
 *   "name": "New Tenant",
 *   "code": "new-tenant",
 *   "region": "us-east-1",
 *   "plan": "free"
 * }
 * Response (201):
 * {
 *   "tenant": { ... full tenant object }
 * }
 *
 * PATCH /api/admin/tenants/{id}
 * Request (all fields optional):
 * {
 *   "name": "Updated Name",
 *   "plan": "pro"
 * }
 * Response:
 * {
 *   "tenant": { ... updated tenant object }
 * }
 *
 * POST /api/admin/tenants/{id}/suspend
 * Response: 200 OK
 *
 * POST /api/admin/tenants/{id}/unsuspend
 * Response: 200 OK
 *
 * GET /api/admin/api-keys?limit=50&offset=0
 * Response:
 * {
 *   "api_keys": [
 *     {
 *       "id": "uuid",
 *       "name": "Key Name",
 *       "user_id": "uuid",
 *       "roles": ["USER"],
 *       "tenant_ids": ["uuid1", "uuid2"],
 *       "is_revoked": false,
 *       "created_at": "2025-02-08T10:00:00Z",
 *       "updated_at": "2025-02-08T10:00:00Z"
 *     }
 *   ],
 *   "total": 10,
 *   "limit": 50,
 *   "offset": 0
 * }
 *
 * POST /api/admin/api-keys
 * Request:
 * {
 *   "name": "New Key",
 *   "tenant_ids": ["uuid1"],
 *   "roles": ["USER", "TENANT_ADMIN"]
 * }
 * Response (201):
 * {
 *   "api_key": {
 *     "key": "sk_xxxxxxxxxxxxx",  // Only returned on creation
 *     ... full key object
 *   }
 * }
 *
 * GET /api/admin/tenants/{id}/usage/daily?days=30
 * Response:
 * {
 *   "tenant_id": "uuid",
 *   "days": 30,
 *   "data": [
 *     {
 *       "day": "2025-02-08",
 *       "count": 5000
 *     }
 *   ]
 * }
 *
 * GET /api/admin/tenants/{id}/usage/endpoints?limit=20
 * Response:
 * {
 *   "tenant_id": "uuid",
 *   "top_endpoints": [
 *     {
 *       "path": "/api/semantic/analyze",
 *       "count": 1200
 *     }
 *   ]
 * }
 *
 * ============================================================================
 * AUTHENTICATION FLOW
 * ============================================================================
 *
 * 1. User logs in via POST /api/auth/login
 * 2. Backend returns JWT token with GLOBAL_OPS role
 * 3. Frontend stores token in localStorage['token']
 * 4. All admin API requests include Authorization header:
 *    Authorization: Bearer <token>
 * 5. Backend validates token and role in auth middleware
 * 6. If unauthorized or token expired, return 401
 * 7. Frontend catches 401 and redirects to /login
 *
 * ============================================================================
 */
