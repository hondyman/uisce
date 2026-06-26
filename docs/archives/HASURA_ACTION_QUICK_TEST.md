# Hasura Action Integration: Quick Test Guide

## 🎯 Quick Setup

```bash
# Set these variables for all tests
export TENANT_ID="00000000-0000-0000-0000-000000000000"
export DATASOURCE_ID="11111111-1111-1111-1111-111111111111"
```

## ⚡ 3-Minute Integration Check

### Check 1: Services Running
```bash
docker ps | grep -E "hasura|api-gateway|backend"
# Should show 3 running services
```

### Check 2: Backend Endpoint
```bash
curl -s -X POST "http://localhost:8080/business-terms/search" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: ${TENANT_ID}" \
  -H "X-Tenant-Datasource-ID: ${DATASOURCE_ID}" \
  -d '{"search_term": "revenue", "limit": 5}' | jq .
```

✅ **Expected**: JSON array of business terms (even if empty)  
❌ **Problem**: 404 or connection error → backend not running

### Check 3: API Gateway Route
```bash
curl -s -X POST "http://localhost:8001/api/search/business-terms" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: ${TENANT_ID}" \
  -H "X-Tenant-Datasource-ID: ${DATASOURCE_ID}" \
  -d '{"search_term": "revenue", "limit": 5}' | jq .
```

✅ **Expected**: JSON array of business terms  
❌ **404**: Route not registered in api-gateway  
❌ **500**: Likely backend connection issue

### Check 4: Hasura Action
```bash
curl -s -X POST "http://localhost:8080/v1/graphql" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: ${TENANT_ID}" \
  -H "X-Tenant-Datasource-ID: ${DATASOURCE_ID}" \
  -d '{
    "query": "query { search_business_terms(search_term: \"revenue\", limit: 5) { id term_name } }"
  }' | jq .
```

✅ **Expected**: `{ "data": { "search_business_terms": [...] } }`  
❌ **Error**: Check Hasura logs and action metadata

---

## 📋 Full Test Suite

### Test A: Backend Service Check
```bash
#!/bin/bash
set -e

echo "🧪 Test A: Backend Service"
echo "URL: http://localhost:8080/business-terms/search"
echo "Method: POST"

curl -v -X POST "http://localhost:8080/business-terms/search" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: ${TENANT_ID}" \
  -H "X-Tenant-Datasource-ID: ${DATASOURCE_ID}" \
  -d '{"search_term": "revenue", "limit": 10}'

echo -e "\n✅ Test A Complete"
```

### Test B: API Gateway Route Check
```bash
#!/bin/bash
set -e

echo "🧪 Test B: API Gateway Route"
echo "URL: http://localhost:8001/api/search/business-terms"
echo "Method: POST"

curl -v -X POST "http://localhost:8001/api/search/business-terms" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: ${TENANT_ID}" \
  -H "X-Tenant-Datasource-ID: ${DATASOURCE_ID}" \
  -d '{"search_term": "revenue", "limit": 10}'

echo -e "\n✅ Test B Complete"
```

### Test C: Hasura GraphQL Action
```bash
#!/bin/bash
set -e

echo "🧪 Test C: Hasura GraphQL Action"
echo "URL: http://localhost:8080/v1/graphql"
echo "Query: search_business_terms"

curl -v -X POST "http://localhost:8080/v1/graphql" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: ${TENANT_ID}" \
  -H "X-Tenant-Datasource-ID: ${DATASOURCE_ID}" \
  -d '{
    "query": "query { search_business_terms(search_term: \"revenue\", limit: 10) { id term_name } }"
  }'

echo -e "\n✅ Test C Complete"
```

---

## 🔍 Debug Steps

### 1. Check if services are running
```bash
docker-compose ps
```

### 2. View service logs
```bash
# Backend
docker logs backend --tail 50

# API Gateway
docker logs api-gateway --tail 50

# Hasura
docker logs hasura --tail 50
```

### 3. Test backend is responding
```bash
curl http://localhost:8080/api/health
# Should return 200 OK
```

### 4. Test api-gateway is responding
```bash
curl http://localhost:8001/api/health
# Should return 200 OK
```

### 5. Check Hasura metadata loaded
```bash
curl http://localhost:8080/v1/metadata \
  -H "X-Hasura-Admin-Secret: myadminsecretkey" | jq '.actions'
# Should show search_business_terms action
```

---

## 📊 Response Examples

### Success Response
```json
{
  "terms": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "term_name": "Revenue",
      "term_type": "metric",
      "description": "Total income from sales",
      "created_at": "2025-10-24T10:00:00Z",
      "updated_at": "2025-10-24T10:00:00Z"
    }
  ]
}
```

### Tenant Header Error
```json
{
  "error": "X-Tenant-ID and X-Tenant-Datasource-ID headers are required"
}
```

### Route Not Found Error
```json
{
  "error": "404 page not found"
}
```

---

## 🛠️ Common Issues & Fixes

| Issue | Cause | Fix |
|-------|-------|-----|
| `Connection refused` | Service not running | `docker-compose up -d` |
| `404 Not Found` | Route not registered | Check api-gateway main.go routes |
| `Missing tenant headers` | Headers not forwarded | Check api-gateway log, verify header passing |
| `Timeout` | Slow backend | Check backend logs for slow queries |
| `Empty results` | No matching terms | Insert test data or use different search term |

---

## ✅ Success Criteria

- [ ] **Check 2 passes**: Backend returns JSON (200 OK)
- [ ] **Check 3 passes**: API Gateway returns JSON (200 OK)
- [ ] **Check 4 passes**: Hasura GraphQL returns `data.search_business_terms`
- [ ] **All tests pass**: Integration is complete

---

## 📞 Getting Help

If tests fail at a specific layer:

1. **Backend fails (Check 2)**: Backend service issue - check logs
2. **API Gateway fails (Check 3)**: Route registration issue - verify main.go
3. **Hasura fails (Check 4)**: Action metadata issue - verify actions.yaml

See `HASURA_ACTION_COMPLETION_GUIDE.md` for detailed troubleshooting.
