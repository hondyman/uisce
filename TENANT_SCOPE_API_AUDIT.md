# Tenant-Scoped API Endpoints Audit Report

## Summary
Searched frontend codebase for all tenant-scoped API calls. **IMPORTANT NOTE:** None of the calls currently use `X-Tenant-Region` header. Most use `X-Tenant-ID` and `X-Tenant-Instance-ID` instead.

---

## FETCH Calls - 14 Total

### ✅ WITH TENANT HEADERS (8)

| File | Line | Endpoint | Headers | Status |
|------|------|----------|---------|--------|
| [src/pages/BusinessObjectsPage.tsx](src/pages/BusinessObjectsPage.tsx#L179) | 179 | `/api/business-objects` | `X-Tenant-ID`, `X-Tenant-Instance-ID` | ✅ Has headers, NO Region |
| [src/pages/BusinessObjectsPage.tsx](src/pages/BusinessObjectsPage.tsx#L232) | 232 | `/api/business-objects/list` | `X-Tenant-ID`, `X-Tenant-Instance-ID` | ✅ Has headers, NO Region |
| [src/pages/BusinessObjectDetailsPage.tsx](src/pages/BusinessObjectDetailsPage.tsx#L1531) | 1531 | `/api/business-objects` (POST) | `Content-Type`, `X-Tenant-ID`, `X-Tenant-Instance-ID` | ✅ Has headers, NO Region |
| [src/components/AddSemanticTermDialog.tsx](src/components/AddSemanticTermDialog.tsx#L77) | 77 | `/api/glossary/terms` (POST) | `Content-Type`, `X-Tenant-ID`, `X-Tenant-Instance-ID` | ✅ Has headers, NO Region |
| [src/components/AddEdgeDialog.tsx](src/components/AddEdgeDialog.tsx#L152) | 152 | `/api/semantic-terms/search` (POST) | `Content-Type`, `X-Tenant-ID`, `X-Tenant-Instance-ID` | ✅ Has headers, NO Region |
| [src/utils/marketplaceImportHandler.ts](src/utils/marketplaceImportHandler.ts#L78) | 78 | `/api/validation-rules` (POST) | `Content-Type`, `X-Tenant-ID`, `X-Tenant-Instance-ID` | ✅ Has headers, NO Region |
| [src/pages/TabbedModal/tabs/SemanticTermDetails.tsx](src/pages/TabbedModal/tabs/SemanticTermDetails.tsx#L140) | 140 | `/api/glossary/terms/{id}` (GET) | `X-Tenant-ID`, `X-Tenant-Instance-ID` | ✅ Has headers, NO Region |
| [src/pages/TabbedModal/tabs/SemanticTermDetails.tsx](src/pages/TabbedModal/tabs/SemanticTermDetails.tsx#L377) | 377 | `/api/glossary/edges/{edgeId}` (DELETE) | `X-Tenant-ID`, `X-Tenant-Instance-ID` | ✅ Has headers, NO Region |

### ❌ WITHOUT TENANT HEADERS (6)

| File | Line | Endpoint | Headers | Status |
|------|------|----------|---------|--------|
| [src/pages/SemanticCatalogPage.tsx](src/pages/SemanticCatalogPage.tsx#L29) | 29 | `/api/semantic-terms` | ❌ NONE | ❌ MISSING HEADERS |
| [src/components/ValidationRules/ValidationRuleSimulator.tsx](src/components/ValidationRules/ValidationRuleSimulator.tsx#L33) | 33 | `/api/validation-rules/simulate` (POST) | `Content-Type` only | ❌ MISSING TENANT HEADERS |
| [frontend/src/services/rulesApi.ts](frontend/src/services/rulesApi.ts#L194) | 194 | `/api/validation-rules` | ❌ NONE | ❌ MISSING HEADERS |
| [src/features/fabric/pages/CalculationsLibraryPage.tsx](src/features/fabric/pages/CalculationsLibraryPage.tsx#L152) | 152 | `/api/semantic-terms` | ❌ NONE | ❌ MISSING HEADERS |
| [src/features/expressions/components/CalculatedFieldBuilder.tsx](src/features/expressions/components/CalculatedFieldBuilder.tsx#L74) | 74 | `/api/semantic-terms?tenant_instance_id=default` | ❌ NONE | ❌ MISSING HEADERS |
| [src/features/security/components/BusinessObjectSelector.tsx](src/features/security/components/BusinessObjectSelector.tsx#L53) | 53 | `/api/business-objects` | ❌ NONE | ❌ MISSING HEADERS |

---

## AXIOS Calls - 6 Total

### ✅ WITH TENANT HEADERS (3)

| File | Line | Endpoint | Headers | Status |
|------|------|----------|---------|--------|
| [src/features/query-builder/pages/BusinessObjectQueryBuilder.tsx](src/features/query-builder/pages/BusinessObjectQueryBuilder.tsx#L215) | 215 | `/api/business-objects` (GET) | `X-Tenant-ID`, `X-Tenant-Instance-ID` | ✅ Has headers, NO Region |
| [src/features/query-builder/pages/BusinessObjectQueryBuilder.tsx](src/features/query-builder/pages/BusinessObjectQueryBuilder.tsx#L361) | 361 | `/api/business-objects/generate-sql` (POST) | `X-Tenant-ID` only | ⚠️ Partial headers, NO Region |
| [src/features/query-builder/pages/BusinessObjectQueryBuilder.tsx](src/features/query-builder/pages/BusinessObjectQueryBuilder.tsx#L368) | 368 | `/api/business-objects/execute-sql` (POST) | `X-Tenant-ID` only | ⚠️ Partial headers, NO Region |

### ❌ WITHOUT TENANT HEADERS (3)

| File | Line | Endpoint | Headers | Status |
|------|------|----------|---------|--------|
| [src/features/uisce-builder/components/Sidebar.tsx](src/features/uisce-builder/components/Sidebar.tsx#L148) | 148 | `/api/semantic-terms` | ❌ NONE | ❌ MISSING HEADERS |
| [src/features/uisce-builder/components/BOSelector.tsx](src/features/uisce-builder/components/BOSelector.tsx#L48) | 48 | `/api/business-objects` | ❌ NONE | ❌ MISSING HEADERS |
| [src/features/uisce-builder/components/ConfigPanel.tsx](src/features/uisce-builder/components/ConfigPanel.tsx#L46) | 46 | `/api/business-objects` | ❌ NONE | ❌ MISSING HEADERS |

---

## Additional Endpoints Found in src/api/

### From glossary.ts (React Query hooks)

| File | Endpoint | Headers | Status |
|------|----------|---------|--------|
| [src/api/glossary.ts](src/api/glossary.ts#L92) | `/api/glossary/semantic-terms` (GET) | `X-Tenant-ID`, `X-Tenant-Instance-ID` | ✅ Has headers, NO Region |
| [src/api/glossary.ts](src/api/glossary.ts#L451) | `/api/glossary/terms/{id}` (PUT) | `Content-Type`, `X-Tenant-ID`, `X-Tenant-Instance-ID` | ✅ Has headers, NO Region |
| [src/api/glossary.ts](src/api/glossary.ts#L552) | `/api/glossary/terms` (POST) | `Content-Type`, `X-Tenant-ID`, `X-Tenant-Instance-ID` | ✅ Has headers, NO Region |
| [src/api/glossary.ts](src/api/glossary.ts#L616) | `/api/glossary/terms/{id}` (DELETE) | `X-Tenant-ID`, `X-Tenant-Instance-ID` | ✅ Has headers, NO Region |

### Lineage Endpoints

| File | Line | Endpoint | Headers | Status |
|------|------|----------|---------|--------|
| [src/pages/TabbedModal/TabbedModal.tsx](src/pages/TabbedModal/TabbedModal.tsx#L706) | 706 | `/api/lineage/hierarchical/{datasourceId}` (POST) | `Content-Type` only | ❌ MISSING TENANT HEADERS |
| [src/pages/TabbedModal/tabs/SemanticTermDetails.tsx](src/pages/TabbedModal/tabs/SemanticTermDetails.tsx#L172) | 172 | `/api/lineage/node/{id}/graph` (GET) | ❌ NONE | ❌ MISSING HEADERS |

### nodeTypes API

| File | Endpoint | Headers | Status |
|------|----------|---------|--------|
| [src/api/nodeTypes.ts](src/api/nodeTypes.ts#L157) | `/api/glossary/semantic-terms` (fallback) | `X-Tenant-ID` only | ⚠️ Partial headers |

---

## Additional Calls in glossary.ts (Not Yet Audited in Components)

| Function | Endpoint | Headers | Status |
|----------|----------|---------|--------|
| useBusinessTerms() | `/api/catalog/nodes` | `X-Tenant-ID`, `X-Tenant-Instance-ID` | ✅ Has headers, NO Region |
| useCreateTermEdge() | `/api/glossary/edges` | `Content-Type`, `X-Tenant-ID`, `X-Tenant-Instance-ID` | ✅ Has headers, NO Region |
| useDeleteTermEdge() | `/api/glossary/edges/{id}` | `X-Tenant-ID`, `X-Tenant-Instance-ID` | ✅ Has headers, NO Region |
| useUpdateTermEdge() | `/api/glossary/edges/{id}` | `Content-Type`, `X-Tenant-ID`, `X-Tenant-Instance-ID` | ✅ Has headers, NO Region |

---

## Key Findings

### 1. **No X-Tenant-Region Header Usage** 🔴
- **Status:** NONE of the 20+ audited API calls include `X-Tenant-Region` header
- **Current Pattern:** All calls use `X-Tenant-ID` and `X-Tenant-Instance-ID`
- **Impact:** This is either:
  - Expected if backend doesn't require region header
  - A design choice using instance ID as region identifier
  - Or a missing implementation need

### 2. **Missing Tenant Headers** 🔴 (6 Instances)
These components make unscoped API calls and should add tenant context:

**CRITICAL - Unscoped Calls:**
1. SemanticCatalogPage - `/api/semantic-terms` (L29)
2. ValidationRuleSimulator - `/api/validation-rules/simulate` (L33)
3. CalculationsLibraryPage - `/api/semantic-terms` (L152)
4. CalculatedFieldBuilder - `/api/semantic-terms?tenant_instance_id=default` (L74)
5. BusinessObjectSelector - `/api/business-objects` (L53)
6. Sidebar (uisce-builder) - `/api/semantic-terms` (L148)
7. BOSelector (uisce-builder) - `/api/business-objects` (L48)
8. ConfigPanel (uisce-builder) - `/api/business-objects` (L46)

**Lineage Endpoints Missing Headers:**
- TabbedModal - `/api/lineage/hierarchical/{datasourceId}` (L706)
- SemanticTermDetails - `/api/lineage/node/{id}/graph` (L172)

### 3. **Partial Header Coverage** ⚠️ (2 Instances)
- BusinessObjectQueryBuilder `/api/business-objects/generate-sql` (L361) - Has `X-Tenant-ID` but missing `X-Tenant-Instance-ID`
- BusinessObjectQueryBuilder `/api/business-objects/execute-sql` (L368) - Has `X-Tenant-ID` but missing `X-Tenant-Instance-ID`
- nodeTypes.ts - Fallback call has `X-Tenant-ID` only

### 4. **Properly Scoped Calls** ✅ (12+ Instances)
The following components properly include tenant context:
- BusinessObjectsPage (both GET and POST)
- BusinessObjectDetailsPage (POST)
- AddSemanticTermDialog
- AddEdgeDialog
- marketplaceImportHandler
- SemanticTermDetails (GET, DELETE)
- glossary.ts hooks (all operations)
- BusinessObjectQueryBuilder (main GET call)

---

## Recommendations

### Priority 1: Clarify X-Tenant-Region Strategy
1. Confirm with backend team whether `X-Tenant-Region` header is actually needed
2. If needed, update header passing middleware/interceptors
3. If not needed, document that region is determined by `X-Tenant-Instance-ID`

### Priority 2: Fix Missing Headers (6 files)
Add `X-Tenant-ID` and `X-Tenant-Instance-ID` headers to:
- SemanticCatalogPage.tsx:29
- ValidationRuleSimulator.tsx:33
- CalculationsLibraryPage.tsx:152
- CalculatedFieldBuilder.tsx:74
- BusinessObjectSelector.tsx:53
- Sidebar.tsx (uisce-builder):148
- BOSelector.tsx (uisce-builder):48
- ConfigPanel.tsx (uisce-builder):46

### Priority 3: Fix Lineage Endpoints (2 files)
Add tenant context headers:
- TabbedModal.tsx:706 (`/api/lineage/hierarchical/{datasourceId}`)
- SemanticTermDetails.tsx:172 (`/api/lineage/node/{id}/graph`)

### Priority 4: Complete Partial Headers (2 files)
Add `X-Tenant-Instance-ID` to:
- BusinessObjectQueryBuilder.tsx:361 (/api/business-objects/generate-sql)
- BusinessObjectQueryBuilder.tsx:368 (/api/business-objects/execute-sql)

---

## Helper Pattern

Standard tenant-scoped fetch pattern found in well-implemented calls:

```typescript
const response = await fetch('/api/endpoint', {
  method: 'GET/POST/PUT/DELETE',
  headers: {
    'Content-Type': 'application/json',
    'X-Tenant-ID': tenantId,
    'X-Tenant-Instance-ID': datasourceId,
  },
});
```

For React Query hooks (src/api/glossary.ts pattern):
```typescript
const { tenant, datasource } = useTenant();
const res = await fetch(url, {
  headers: {
    ...(tenant?.id && { 'X-Tenant-ID': tenant.id }),
    ...(datasource?.id && { 'X-Tenant-Instance-ID': datasource.id }),
  },
});
```
