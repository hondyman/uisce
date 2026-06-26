# Investment Validation Rules Integration Guide

**Time Required:** 15-30 minutes  
**Complexity:** Medium  
**Status:** Ready to Deploy  

---

## 🎯 Overview

This guide shows you how to integrate the complete **Investment Validation Rules Engine** into your Fabric Builder application.

### Components Being Added
1. ✅ **Validation Rules Builder** - Create and manage validation rules
2. ✅ **Validation Execution Dashboard** - Run validations and see results
3. ✅ **Navigation Menu Items** - Easy access from main menu
4. ✅ **REST API Integration** - Backend-to-frontend communication
5. ✅ **Database Schema** - validation_rules and validation_results tables

---

## Step 1: Import Components into AppRoutes

Locate your main routing file (`frontend/src/AppRoutes.tsx` or similar) and add these imports:

```typescript
import { ValidationRulesBuilderPage } from './pages/ValidationRulesBuilderPage';
import { InvestmentValidationPage } from './pages/InvestmentValidationPage';
```

## Step 2: Add Routes

In your `<Routes>` section, add:

```tsx
<Routes>
  {/* ... existing routes ... */}
  
  {/* Validation Rules Management */}
  <Route 
    path="/investment/validation/rules" 
    element={<ProtectedRoute><ValidationRulesBuilderPage /></ProtectedRoute>} 
  />
  
  {/* Validation Execution Dashboard */}
  <Route 
    path="/investment/validation" 
    element={<ProtectedRoute><InvestmentValidationPage /></ProtectedRoute>} 
  />
  
  {/* ... other routes ... */}
</Routes>
```

## Step 3: Add Navigation Menu Items

In your navigation/sidebar component (typically in the top nav bar), add these links:

```tsx
<nav className="p-4 bg-gray-100 flex gap-4 mb-4 app-top-nav">
  {/* Existing links */}
  <BlockableLink to="/bundles" className="hover:underline">Micro-Bundle Catalog</BlockableLink>
  <BlockableLink to="/bundle-explorer" className="hover:underline">Bundle Explorer</BlockableLink>
  
  {/* ADD THESE TWO LINES */}
  <BlockableLink to="/investment/validation/rules" className="hover:underline">📋 Validation Rules</BlockableLink>
  <BlockableLink to="/investment/validation" className="hover:underline">✓ Run Validations</BlockableLink>
  
  <BlockableLink to="/fixed-income" className="hover:underline">Fixed Income Analytics</BlockableLink>
  {/* ... rest of navigation ... */}
</nav>
```

## Step 4: Verify TenantContext Integration

Both pages require TenantContext. Ensure your app is wrapped with the provider:

```tsx
<TenantProvider>
  <ProtectedApp />
</TenantProvider>
```

And that you have the context setup:

```typescript
import { useTenant } from '@/context/TenantContext';

// Inside component:
const { tenant, datasource } = useTenant();
```

## Step 5: Set Up Database Tables

Run this SQL in your PostgreSQL database:

```sql
-- Validation Rules Table
CREATE TABLE IF NOT EXISTS public.validation_rules (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    datasource_id uuid NOT NULL,
    rule_name varchar(255) NOT NULL,
    rule_type varchar(50) NOT NULL,
    description text NULL,
    account_types text[] DEFAULT '{}'::text[] NOT NULL,
    parameters jsonb NOT NULL,
    severity varchar(20) NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    evaluation_order int4 DEFAULT 100 NOT NULL,
    allow_override bool DEFAULT false NOT NULL,
    required_authority varchar(50) NULL,
    created_by uuid NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT validation_rules_pkey PRIMARY KEY (id),
    CONSTRAINT validation_rules_tenant_datasource_name_key UNIQUE (tenant_id, datasource_id, rule_name),
    CONSTRAINT validation_rules_severity_check CHECK (severity = ANY (ARRAY['BLOCK'::character varying, 'WARNING'::character varying, 'INFO'::character varying])),
    CONSTRAINT validation_rules_tenant_fk FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE
);

-- Indices for performance
CREATE INDEX idx_validation_rules_tenant ON public.validation_rules USING btree (tenant_id, datasource_id);
CREATE INDEX idx_validation_rules_type ON public.validation_rules USING btree (rule_type);
CREATE INDEX idx_validation_rules_active ON public.validation_rules USING btree (is_active) WHERE (is_active = true);

-- Validation Results Table
CREATE TABLE IF NOT EXISTS public.validation_results (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    datasource_id uuid NOT NULL,
    account_id varchar(255) NOT NULL,
    account_type varchar(50) NOT NULL,
    rule_id uuid NOT NULL,
    rule_type varchar(50) NOT NULL,
    passed bool NOT NULL,
    severity varchar(20) NOT NULL,
    message text NULL,
    failed_value jsonb NULL,
    threshold_value jsonb NULL,
    details jsonb NULL,
    executed_at timestamptz DEFAULT now() NOT NULL,
    expires_at timestamptz NULL,
    CONSTRAINT validation_results_pkey PRIMARY KEY (id),
    CONSTRAINT validation_results_tenant_fk FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE,
    CONSTRAINT validation_results_rule_fk FOREIGN KEY (rule_id) REFERENCES public.validation_rules(id) ON DELETE CASCADE
);

-- Indices for performance
CREATE INDEX idx_validation_results_tenant ON public.validation_results USING btree (tenant_id, datasource_id);
CREATE INDEX idx_validation_results_account ON public.validation_results USING btree (account_id);
CREATE INDEX idx_validation_results_executed ON public.validation_results USING btree (executed_at DESC);
```

## Step 6: Backend API Verification

The backend automatically exposes these endpoints (verify they're configured in `backend/internal/api/api.go`):

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/validation-rules` | List all rules for tenant |
| POST | `/api/validation-rules` | Create new rule |
| PUT | `/api/validation-rules/:id` | Update existing rule |
| DELETE | `/api/validation-rules/:id` | Delete rule |
| POST | `/api/validate` | Execute validations |
| GET | `/api/validation-results/:accountId` | Get validation history |

**All requests must include tenant headers:**
```
X-Tenant-ID: <tenant-uuid>
X-Tenant-Datasource-ID: <datasource-uuid>
```

(These are automatically added by the TenantContext fetch shim)

## Step 7: Test the Integration

### Quick Test Flow:

1. **Start the app** and navigate to `http://localhost:3000`
2. **Select a tenant** using the tenant picker in the Fabric Builder shell
3. **Navigate to Validation Rules** - Click the new "📋 Validation Rules" link
4. **Create a test rule:**
   - Click "New Rule" button
   - **Name:** "Test Concentration Rule"
   - **Type:** CONCENTRATION
   - **Severity:** WARNING
   - **Click Save**
5. **Run a validation:**
   - Click "✓ Run Validations" link
   - Select an account
   - Click "Run Validation"
   - Verify test rule appears in results

---

## 📊 Component Architecture

```
Frontend Structure:
├── ValidationRulesBuilderPage.tsx      (Create/Edit/Delete rules)
│   ├── RulesList                       (Display all rules)
│   ├── RuleForm (Modal)                (Create/Edit form)
│   ├── TenantContext integration       (Tenant scoping)
│   └── validationEngine.ts API calls   (CRUD operations)
│
└── InvestmentValidationPage.tsx        (Execute validations)
    ├── ValidationForm                  (Select account, run)
    ├── ResultsList                     (Display results)
    ├── TenantContext integration       (Tenant scoping)
    └── validationEngine.ts API calls   (Execute, get history)

Backend Structure:
├── validation_engine.go                (Orchestrator)
├── validation_rules_routes.go          (REST endpoints)
└── Database:
    ├── validation_rules table          (Rule definitions)
    └── validation_results table        (Execution history)
```

---

## ✅ Integration Checklist

After completing all steps, verify:

- [ ] Routes added to AppRoutes.tsx
- [ ] Navigation links visible in top nav
- [ ] Database tables created in PostgreSQL
- [ ] Tenant selector works (shows tenants, products, datasources)
- [ ] `/investment/validation/rules` page loads
- [ ] `/investment/validation` page loads
- [ ] Can create new rule (form opens, saves)
- [ ] Can edit existing rule
- [ ] Can delete rule (confirmation dialog works)
- [ ] Can run validation (results display)
- [ ] Rules persist after page refresh
- [ ] Validation results show correct severity badges

---

## 🚀 Your First Validation Rule

Once integration is complete, create a rule to test the system:

**Example: Portfolio Concentration Limit**
- **Name:** Max Single Asset Concentration
- **Type:** CONCENTRATION
- **Description:** No single asset > 20% of portfolio
- **Account Types:** Fund, Portfolio
- **Severity:** WARNING
- **Evaluation Order:** 10
- **Allow Override:** Yes (with Director approval)

Then run a validation to see it in action!

---

## 🆘 Troubleshooting

| Issue | Solution |
|-------|----------|
| Pages show "Select a tenant" warning | Use tenant picker in header to select tenant/product/datasource |
| Rules don't persist | Check database tables exist; verify API responses in Network tab |
| Validation results empty | Ensure account_id format matches your data; check backend logs |
| Buttons disabled on Rules page | TenantContext not selected - use tenant picker first |
| Form validation failing | Check required fields marked with asterisk (*) |
| CORS errors in console | Verify backend is running on correct port (8080) |

---

## 📚 Related Documentation

- `INVESTMENT_VALIDATION_ENGINE_DEPLOYMENT.md` - Full architecture details
- `INVESTMENT_VALIDATION_QUICK_START.md` - Fast setup walkthrough
- `INVESTMENT_VALIDATION_DELIVERY_SUMMARY.md` - Complete overview with examples
- `validationConstants.ts` - Rule types and enum definitions

---

## ✨ You're Done!

The Investment Validation Rules Engine is now fully integrated into your Fabric Builder application.

**Next steps:**
1. Create your first validation rule
2. Run a validation to test the flow
3. Customize rules for your business logic
4. Deploy to production when ready
5. View results

---

## Complete Example

Here's what the relevant sections should look like:

```typescript
// At the top of AppRoutes.tsx
import React from "react";
import { BrowserRouter as Router, Routes, Route } from "react-router-dom";
// ... other imports ...
import InvestmentValidationPage from "./pages/InvestmentValidationPage";  // ADD THIS

export function AppRoutes() {
  return (
    <Router>
      <RouteBlockerProvider>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/*" element={<ProtectedApp />} />
        </Routes>
      </RouteBlockerProvider>
    </Router>
  );
}

function ProtectedApp() {
  const navigate = useBlockableNavigate();
  // ... handler functions ...

  return (
    <>
      <nav className="p-4 bg-gray-100 flex gap-4 mb-4 app-top-nav">
        <BlockableLink to="/bundles" className="hover:underline">Micro-Bundle Catalog</BlockableLink>
        <BlockableLink to="/bundle-explorer" className="hover:underline">Bundle Explorer</BlockableLink>
        <BlockableLink to="/investment/validation" className="hover:underline">Investment Validation</BlockableLink>
        <BlockableLink to="/fixed-income" className="hover:underline">Fixed Income Analytics</BlockableLink>
        <BlockableLink to="/jit-request" className="hover:underline">JIT Request Panel</BlockableLink>
        <BlockableLink to="/access-explanation" className="hover:underline">Access Explanation</BlockableLink>
        <CoreMenu />
        <MegaMenu />
      </nav>

      <Routes>
        {/* ... existing routes ... */}
        <Route path="/investment/validation" element={<ProtectedRoute><InvestmentValidationPage /></ProtectedRoute>} />
        {/* ... more routes ... */}
      </Routes>
    </>
  );
}
```

---

## Features Available on the Page

Once added, users will have access to:

✅ **Portfolio Validation**
- Select account and account type
- View portfolio summary (value, positions, cash, concentration)
- Run validation with one click

✅ **Real-Time Results**
- Pass/fail status
- Blocked rules (red)
- Warning rules (yellow)
- Info messages (blue)
- Complete results table

✅ **Validation History**
- Last 30 days of validation runs
- History table with results
- Trend analysis

✅ **Rule Management**
- View all active rules
- Create custom rules
- Edit existing rules
- Disable rules

---

## Navigation Structure

After integration, your navigation will look like:

```
Top Navigation Bar:
├─ Micro-Bundle Catalog
├─ Bundle Explorer
├─ Investment Validation ← NEW
├─ Fixed Income Analytics
├─ JIT Request Panel
├─ Access Explanation
├─ ... (Core Menu, Mega Menu)
```

---

## Database Requirements

Make sure you've already created these tables (from the quick start guide):

```sql
CREATE TABLE validation_rules (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  ...
  tenant_id TEXT NOT NULL,
  datasource_id TEXT NOT NULL
);

CREATE TABLE validation_results (
  id SERIAL PRIMARY KEY,
  account_id TEXT NOT NULL,
  ...
  tenant_id TEXT NOT NULL,
  datasource_id TEXT NOT NULL,
  ...
);
```

---

## Environment Setup

Make sure your backend is running with:

```bash
# Backend environment variables (config.yaml)
HASURA_ENDPOINT=http://localhost:8080/v1/graphql
RABBITMQ_URL=amqp://localhost:5672  # Optional
```

---

## Verification

After adding the route, verify it works:

1. ✅ Link appears in navigation
2. ✅ Page loads without errors
3. ✅ Tenant/datasource dropdown shows options
4. ✅ Can select account and account type
5. ✅ Run Validation button executes (after DB setup)
6. ✅ Results display properly

---

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Page not found (404) | Make sure route path and import are correct |
| Tenant scope error | Select a tenant in TenantContext before navigating |
| Database error | Run SQL setup from quick start guide |
| No rules showing | Create rules via POST /api/validation-rules or seed data |
| Validation doesn't run | Check backend is running and database tables exist |

---

## Done! 🎉

Your investment management platform now has a fully integrated validation dashboard. Users can start validating accounts immediately after you complete these 4 steps.

For more information, see:
- `INVESTMENT_VALIDATION_QUICK_START.md` - Quick start guide
- `INVESTMENT_VALIDATION_ENGINE_DEPLOYMENT.md` - Full documentation
