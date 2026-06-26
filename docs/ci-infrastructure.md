# WealthStream API & CI Infrastructure

## OpenAPI Contract

Location: `api/openapi.yml`

Generate client SDKs:
```bash
# JavaScript/TypeScript
npx @openapitools/openapi-generator-cli generate \
  -i api/openapi.yml \
  -g typescript-fetch \
  -o frontend/src/api/generated

# Go
oapi-codegen -package api api/openapi.yml > backend/pkg/api/generated.go
```

## CI Helper Scripts

### Feed Card Delivery Test
```bash
# Test that a specific card appears in feed and CTA works
TENANT=test_t1 \
CLIENT=test_c1 \
CARD_ID=card_dividend_income \
API_KEY=your-key \
./scripts/assert_feed_card_delivery.sh
```

### End-to-End Test
```bash
# Run complete E2E scenario
E2E_BASEURL=http://localhost:8080 \
API_KEY=your-key \
./scripts/e2e_run.sh --scenario basic-mvp
```

## GitHub Actions Workflow

Location: `.github/workflows/acceptance.yml`

**Jobs:**
1. **metadata-validation**: Validates YAML syntax and metadata
2. **unit-tests**: Go unit tests for policy evaluators
3. **integration-tests**: Full stack with Postgres + StarRocks
4. **e2e-tests**: Complete end-to-end scenario

**Secrets Required:**
- `API_KEY`: Test API key for authentication

## Local Development

```bash
# Start services
docker-compose up -d postgres starrocks-fe starrocks-be nessie

# Run backend
cd backend
go run cmd/server/main.go

# Run tests
./scripts/assert_feed_card_delivery.sh
./scripts/e2e_run.sh --scenario basic-mvp
```

## Test Data Fixtures

The E2E script expects `/internal/test/seed` endpoint. Implement in `backend/internal/api/api.go`:

```go
r.Post("/internal/test/seed", func(w http.ResponseWriter, r *http.Request) {
    // Only in test/staging builds
    if os.Getenv("ENV") == "production" {
        http.Error(w, "forbidden", http.StatusForbidden)
        return
    }
    // Parse fixture request and seed DB
})
```

## CI/CD Integration

The GitHub Actions workflow runs on every push/PR and validates:
- ✅ Metadata YAML syntax
- ✅ CEL/Rego expression compilation
- ✅ Unit tests (policy evaluators)
- ✅ Integration tests (feed + CTA)
- ✅ UAR chain verification
- ✅ End-to-end happy path
