# Node.js/Express JWT Integration Guide

## Overview

This guide shows how to integrate JWT validation into Node.js/Express services using the centralized JWT middleware.

## Quick Start

### 1. Import JWT Middleware

```typescript
import { 
  jwtMiddleware, 
  injectTenantFromClaims,
  getClaims,
  requireRole,
  requireTenant,
  getClaims 
} from '../../libs/jwt-middleware-node.js';
```

### 2. Add Middleware to Express App

```typescript
// Define public paths that don't require JWT
const publicPaths = ['/health', '/ready', '/docs', '/api/docs'];

// Apply JWT middleware before routes
app.use(jwtMiddleware(publicPaths));

// Automatically inject tenant ID from JWT claims if not provided
app.use(injectTenantFromClaims());

// Then setup your routes
setupRoutes(app);
```

### 3. Access JWT Claims in Handlers

```typescript
import { getClaims } from '../../libs/jwt-middleware-node.js';

router.get('/accounts', (req, res) => {
  const claims = getClaims(req);
  
  if (!claims) {
    return res.status(401).json({ error: 'Unauthorized' });
  }
  
  // Get user info from JWT
  const userId = claims.user_id;
  const tenantId = claims.tenant_id;
  const roles = claims.roles;
  
  // Scope database queries by tenant
  // SELECT * FROM accounts WHERE tenant_id = $1
  
  res.json({ accounts: [] });
});
```

### 4. Enforce Tenant Isolation (CRITICAL)

**All database queries MUST be scoped by tenant_id:**

```typescript
// WRONG - exposes all company data to wrong tenant
const accounts = await db.query('SELECT * FROM accounts');

// CORRECT - scoped to tenant from JWT claims
const claims = getClaims(req);
const accounts = await db.query(
  'SELECT * FROM accounts WHERE tenant_id = $1',
  [claims.tenant_id]
);
```

### 5. Enforce Role Requirements

```typescript
import { requireRole } from '../../libs/jwt-middleware-node.js';

// Require admin role
router.post('/admin/settings', 
  requireRole('admin'),  // Middleware validates role
  async (req, res) => {
    // Only admin users reach here
    res.json({ message: 'Settings updated' });
  }
);

// Require specific roles
router.post('/approvals/:id/approve',
  requireRole('approver', 'admin'),  // Either role is acceptable
  async (req, res) => {
    res.json({ message: 'Approval processed' });
  }
);
```

### 6. Validate Tenant Access

```typescript
import { requireTenant, validateTenantAccess, getClaims } from '../../libs/jwt-middleware-node.js';

// Middleware approach - for route parameters
router.get('/tenants/:tenantId/data',
  requireTenant('param'),  // Extract from URL parameter
  async (req, res) => {
    res.json({ data: [] });
  }
);

// Manual approach - for headers or data validation
router.put('/data', async (req, res) => {
  const claims = getClaims(req);
  const requestedTenantId = req.headers['x-tenant-id'] as string;
  
  if (!validateTenantAccess(claims, requestedTenantId)) {
    return res.status(403).json({ error: 'Access denied' });
  }
  
  // Process request
  res.json({ message: 'Data updated' });
});
```

## JWT Claims Structure

```typescript
interface JWTClaims {
  user_id: string;              // Unique user ID
  email: string;                // User email
  tenant_id: string;            // Primary tenant
  tenant_ids: string[];         // All accessible tenants
  roles: string[];              // User roles (admin, user, approver, etc)
  is_active: boolean;           // Account active status
  is_core_admin: boolean;       // System admin flag
  org_id?: string;              // Organization ID
  iat: number;                  // Issued at (Unix timestamp)
  exp: number;                  // Expiration (Unix timestamp)
}
```

## Implementation Checklist - Entity Manager

### server.ts
- [x] Import JWT middleware from libs
- [x] Add jwtMiddleware() before setupRoutes()
- [x] Define public paths (/health, /ready, /docs)
- [x] Add injectTenantFromClaims() middleware

### API Route Files (accounts.ts, trades.ts, etc)

#### accounts.ts
- [ ] Update POST /personal to use JWT tenant_id instead of request body
- [ ] Update POST /ira to use JWT tenant_id
- [ ] Update POST /trust to use JWT tenant_id
- [ ] Scope all account queries: `WHERE tenant_id = $1`
- [ ] Add role checks for admin operations
- [ ] Validate user can only modify their own accounts

#### trades.ts
- [ ] Scope all trade queries by tenant_id and user_id
- [ ] Add validation that users can only see their trades
- [ ] Enforce tenant isolation in trade creation

#### approvals.ts
- [ ] Add requireRole('approver') or requireRole('admin') to approval endpoints
- [ ] Scope approvals by tenant_id
- [ ] Track which user approved (from claims.user_id)

#### compliance.ts
- [ ] Scope compliance checks by tenant_id
- [ ] Add role checks for compliance officers
- [ ] Log compliance actions with user_id from claims

### Example Updates

**Before (Trusts Request Body):**
```typescript
router.post('/personal', async (req, res) => {
  const { id, tenantId, accountNumber, name, ownerId } = req.body;
  // Dangerous: tenantId could be anything
  const account = new PersonalAccount(id, tenantId, ...);
});
```

**After (Uses JWT Tenant):**
```typescript
import { getClaims } from '../../libs/jwt-middleware-node.js';

router.post('/personal', async (req, res) => {
  const claims = getClaims(req);
  const { id, accountNumber, name, ownerId } = req.body;
  
  // Safe: tenantId from authenticated JWT
  const account = new PersonalAccount(id, claims.tenant_id, ...);
  
  // Database query scoped by tenant
  await db.query(
    'INSERT INTO personal_accounts (id, tenant_id, account_number, name, owner_id) VALUES ($1, $2, $3, $4, $5)',
    [id, claims.tenant_id, accountNumber, name, ownerId]
  );
});
```

## Database Query Patterns

### Pattern 1: Single Tenant Query
```typescript
// Get accounts for current user's tenant
const claims = getClaims(req);
const result = await db.query(
  'SELECT * FROM accounts WHERE tenant_id = $1',
  [claims.tenant_id]
);
```

### Pattern 2: User-Scoped Query
```typescript
// Get only this user's data
const result = await db.query(
  'SELECT * FROM accounts WHERE tenant_id = $1 AND created_by_user_id = $2',
  [claims.tenant_id, claims.user_id]
);
```

### Pattern 3: Role-Based Query
```typescript
// Admins see all account approvals, others see only their own
let query = 'SELECT * FROM account_approvals WHERE tenant_id = $1';
let params = [claims.tenant_id];

if (!claims.roles.includes('admin')) {
  query += ' AND assigned_to_user_id = $2';
  params.push(claims.user_id);
}

const result = await db.query(query, params);
```

### Pattern 4: Multi-Tenant Query
```typescript
// User can access multiple tenants
const result = await db.query(
  'SELECT * FROM accounts WHERE tenant_id = ANY($1::text[])',
  [claims.tenant_ids]
);
```

## Testing

### Test 1: Get JWT Token
```bash
TOKEN=$(curl -X POST http://localhost:8001/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}' \
  | jq -r '.access_token')

echo "Token: $TOKEN"
```

### Test 2: Call Protected Endpoint
```bash
curl -X GET http://localhost:4000/api/accounts \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: tenant-123"
```

### Test 3: Test Without JWT (Should Fail)
```bash
curl -X GET http://localhost:4000/api/accounts
# Expected: 401 Unauthorized
```

### Test 4: Test Tenant Isolation
```bash
curl -X GET http://localhost:4000/api/accounts \
  -H "Authorization: Bearer $TOKEN_FROM_TENANT_A" \
  -H "X-Tenant-ID: tenant-b"
# Expected: 403 Forbidden or filtered to tenant-a only
```

### Test 5: Health Check (No JWT Needed)
```bash
curl -X GET http://localhost:4000/health
# Expected: 200 OK
```

## Service-to-Service Communication

When entity-manager calls other services (e.g., calls validation-engine), include JWT:

```typescript
import jwt from 'jsonwebtoken';

export async function callValidationEngine(data: any, claims: JWTClaims) {
  // Sign a service token with current user's context
  const token = jwt.sign(
    {
      user_id: claims.user_id,
      tenant_id: claims.tenant_id,
      roles: claims.roles,
    },
    process.env.JWT_SECRET || 'dev-jwt-secret-key-change-in-production',
    { expiresIn: '1h' }
  );
  
  const response = await fetch('http://validation-engine:8090/api/validate', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
      'X-Tenant-ID': claims.tenant_id,
    },
    body: JSON.stringify(data),
  });
  
  return response.json();
}
```

## Troubleshooting

### "JWT claims not found" Error
- ✅ Service has jwtMiddleware added
- ✅ Route is not in publicPaths list
- ✅ Request includes valid Authorization header
- ✅ Bearer token is valid and not expired

### "Access denied to this tenant" Error
- ✅ Verify X-Tenant-ID header matches user's tenant
- ✅ Check if user has multi-tenant access
- ✅ Verify JWT expires at correct time (default: 1 hour)

### "Requires role: admin" Error
- ✅ User logged in with account that has admin role
- ✅ Check JWT token claims for roles array
- ✅ Verify role is spelled correctly

### Tenant Data Leakage
- ✅ ALL database queries must include tenant_id in WHERE clause
- ✅ Use pattern: `WHERE tenant_id = ?` or `WHERE tenant_id = ANY(?)`
- ✅ Never skip tenant scope for "data cleanup" or "admin queries"

## Environment Variables

```bash
# Required in .env
JWT_SECRET=your-secret-key-change-in-production

# Optional
NODE_ENV=development  # or production
ENABLE_SECURITY=true  # Forces JWT on all endpoints
```

## Production Checklist

- [ ] Generate strong JWT_SECRET (use: `openssl rand -base64 32`)
- [ ] Never commit JWT_SECRET to git
- [ ] Use secrets manager (AWS Secrets Manager, HashiCorp Vault, etc)
- [ ] All database queries scoped by tenant_id
- [ ] All environment variables set correctly
- [ ] Tested with multiple users from different tenants
- [ ] Tested role-based access control
- [ ] Monitored logs for JWT validation errors
- [ ] Set up alerts for failed authentication attempts
- [ ] Documented service-to-service communication

## Related Documentation

- [JWT_SECURITY_IMPLEMENTATION.md](./JWT_SECURITY_IMPLEMENTATION.md)
- [JWT_DEPLOYMENT_GUIDE.md](./JWT_DEPLOYMENT_GUIDE.md)
- [JWT_MICROSERVICE_INTEGRATION_CHECKLIST.md](./JWT_MICROSERVICE_INTEGRATION_CHECKLIST.md)
- [libs/jwt-middleware-node.ts](./libs/jwt-middleware-node.ts)

---

**Last Updated**: 2026-02-23
**Status**: Node.js Integration Guide Ready
**Maintainer**: Security Team
