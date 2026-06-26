# NLQ System Deployment Checklist

## Pre-Deployment

### 1. Environment Setup
- [ ] Gemini API key obtained from Google AI Studio
- [ ] API key added to environment: `export GEMINI_API_KEY="your-key"`
- [ ] Database access verified: `psql postgres://...` connects successfully
- [ ] pgvector extension available (check with `SELECT * FROM pg_available_extensions WHERE name = 'vector'`)

### 2. Database Preparation
- [ ] Run migration: `psql < backend/internal/migrations/nlq_support.sql`
- [ ] Verify `catalog_node.embedding` column exists
- [ ] Verify `get_calc_dag_with_metadata()` function exists
- [ ] Verify IVFFlat index on embedding column exists

### 3. Build Tools
- [ ] Build embedding generator: `./scripts/build-embedding-tool.sh`
- [ ] Verify binary created: `ls -lh bin/generate-embeddings`

### 4. Generate Embeddings
For each tenant/datasource combination:
- [ ] Run: `./bin/generate-embeddings --tenant=<ID> --datasource=<ID>`
- [ ] Verify embeddings created: `SELECT COUNT(*) FROM catalog_node WHERE embedding IS NOT NULL`
- [ ] Check for errors in output

## Backend Deployment

### 1. Code Integration
- [ ] Verify `backend/internal/llm/provider.go` exists
- [ ] Verify `backend/internal/services/nlq_service.go` exists
- [ ] Verify `backend/internal/services/catalog_embedding_service.go` exists
- [ ] Verify `backend/internal/handlers/nlq_handler.go` exists
- [ ] Verify `backend/internal/api/api.go` has NLQ initialization
- [ ] Verify route registered: `r.Post("/nlq/ask", srv.handleNLQAsk)`

### 2. Build & Test
- [ ] Build backend: `go build ./...`
- [ ] Run tests: `go test ./...`
- [ ] Check for compile errors
- [ ] Verify imports resolve correctly

### 3. Start Backend
- [ ] Set environment: `export GEMINI_API_KEY="..."`
- [ ] Start server: `go run cmd/server/main.go` (or your command)
- [ ] Check logs for "LLM provider initialized" or similar
- [ ] Verify route registered: check startup logs for `/api/nlq/ask`

### 4. API Testing
Test with curl:
```bash
curl -X POST http://localhost:8080/api/nlq/ask \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: YOUR_TENANT_ID" \
  -H "X-Tenant-Datasource-ID: YOUR_DATASOURCE_ID" \
  -d '{"question": "How is monthly revenue calculated?"}'
```

- [ ] Receives 200 response
- [ ] Response contains `answer` field
- [ ] Response contains `sources` array
- [ ] Response contains `confidence` field
- [ ] Check backend logs for no errors

## Frontend Deployment

### 1. Add Routes
In your app router (e.g., `AppRoutes.tsx`):
- [ ] Import `NLQPage` from `./pages/nlq/NLQPage`
- [ ] Import `LLMConfigPage` from `./pages/admin/LLMConfigPage`
- [ ] Add route: `<Route path="/nlq" element={<NLQPage />} />`
- [ ] Add route: `<Route path="/admin/llm" element={<LLMConfigPage />} />`

### 2. Navigation
Add to your nav menu:
- [ ] "Ask AI" or "Natural Language Q&A" link to `/nlq`
- [ ] "LLM Config" in admin section link to `/admin/llm` (admin only)

### 3. Build & Test
- [ ] Build frontend: `npm run build`
- [ ] Check for TypeScript errors
- [ ] Check for CSS issues
- [ ] Verify no missing imports

### 4. Browser Testing
Navigate to `/nlq`:
- [ ] Page loads without errors
- [ ] Tenant picker warning shows if no tenant selected
- [ ] Header shows tenant/datasource when selected
- [ ] Example questions render
- [ ] Input field accepts text
- [ ] Submit button works

Navigate to `/admin/llm`:
- [ ] Page loads without errors
- [ ] Provider dropdown works
- [ ] Model dropdown works
- [ ] API key field accepts input
- [ ] Save button works

## Integration Testing

### 1. End-to-End Flow
- [ ] Select tenant + datasource via picker
- [ ] Navigate to `/nlq`
- [ ] Click an example question
- [ ] Verify loading state shows (animated dots)
- [ ] Verify answer appears in message
- [ ] Verify sources list appears
- [ ] Verify caveats show (if any)
- [ ] Verify calculation breakdown expands

### 2. Admin Flow
- [ ] Navigate to `/admin/llm`
- [ ] Select different provider (e.g., OpenAI)
- [ ] Enter API key
- [ ] Adjust temperature slider
- [ ] Enter test question
- [ ] Click "Test Configuration"
- [ ] Verify test result appears
- [ ] Click "Save Configuration"
- [ ] Verify success message

### 3. Error Handling
Test error scenarios:
- [ ] Ask question without tenant scope → shows warning
- [ ] Ask question for non-existent entity → shows error message
- [ ] Ask question with invalid API key → shows error message
- [ ] Test network disconnection → shows error message

## Performance Validation

### 1. Response Times
- [ ] First question: < 5 seconds
- [ ] Subsequent questions: < 3 seconds
- [ ] Semantic search: < 500ms
- [ ] Embedding generation: Check batch progress

### 2. Database Performance
- [ ] Semantic search query plan uses vector index
- [ ] DAG query completes in < 1 second
- [ ] No full table scans on catalog_node

Check with:
```sql
EXPLAIN ANALYZE
SELECT qualified_path FROM catalog_node 
WHERE embedding <=> '[...]'::vector 
ORDER BY embedding <=> '[...]'::vector LIMIT 5;
```

### 3. LLM API Performance
- [ ] API calls complete in < 3 seconds
- [ ] No rate limit errors
- [ ] Token usage reasonable (< 2000 tokens per request)

## Security Validation

### 1. Tenant Isolation
- [ ] Requests without `X-Tenant-ID` are rejected (400)
- [ ] Requests without `X-Tenant-Datasource-ID` are rejected (400)
- [ ] User A cannot query user B's catalog nodes

### 2. API Key Security
- [ ] API keys stored in environment variables (not hardcoded)
- [ ] API keys not exposed in frontend
- [ ] API keys not logged

### 3. Input Validation
- [ ] Frontend validates question length
- [ ] Backend sanitizes inputs
- [ ] SQL injection not possible (using parameterized queries)

## Documentation

- [ ] Update main README with link to NLQ feature
- [ ] Add section on using NLQ in user guide
- [ ] Document admin LLM config process
- [ ] Document embedding generation schedule (e.g., nightly)

## Monitoring Setup

### 1. Logs
- [ ] Backend logs NLQ requests with tenant/datasource
- [ ] Backend logs LLM API calls with latency
- [ ] Backend logs embedding generation progress
- [ ] Frontend logs errors to console

### 2. Metrics (Optional)
- [ ] Track questions per minute
- [ ] Track average response time
- [ ] Track LLM token usage
- [ ] Track cache hit rate (if implemented)

### 3. Alerts (Optional)
- [ ] Alert on high error rate (> 10%)
- [ ] Alert on slow response times (> 10s)
- [ ] Alert on LLM API failures
- [ ] Alert on embedding generation failures

## Rollback Plan

### If Issues Occur

1. **Backend Issues:**
   - [ ] Stop backend server
   - [ ] Revert `api.go` changes (remove NLQ route)
   - [ ] Rebuild and restart
   - [ ] Feature disabled, main app unaffected

2. **Database Issues:**
   - [ ] Rollback migration:
     ```sql
     DROP FUNCTION IF EXISTS get_calc_dag_with_metadata(text, uuid);
     DROP FUNCTION IF EXISTS get_calc_dag(text, uuid);
     DROP FUNCTION IF EXISTS resolve_node(text, uuid);
     ALTER TABLE catalog_node DROP COLUMN IF EXISTS embedding;
     DROP EXTENSION IF EXISTS vector;
     ```

3. **Frontend Issues:**
   - [ ] Remove `/nlq` route from router
   - [ ] Remove navigation links
   - [ ] Rebuild frontend
   - [ ] Feature hidden, main app unaffected

## Post-Deployment

### 1. User Training
- [ ] Create video demo of NLQ feature
- [ ] Write user guide with examples
- [ ] Share example questions for each domain
- [ ] Announce feature in team channels

### 2. Feedback Collection
- [ ] Add feedback form in NLQ UI (future)
- [ ] Monitor user questions in logs
- [ ] Track most common questions
- [ ] Identify gaps in catalog coverage

### 3. Maintenance Schedule
- [ ] Schedule nightly embedding generation
- [ ] Plan monthly LLM model updates
- [ ] Review prompt templates quarterly
- [ ] Audit API costs monthly

## Success Metrics

After 1 week:
- [ ] At least 10 unique users tried NLQ
- [ ] At least 50 questions asked
- [ ] Average response time < 5 seconds
- [ ] Error rate < 5%
- [ ] Positive user feedback

After 1 month:
- [ ] NLQ used daily by data analysts
- [ ] Catalog coverage > 80% (embeddings generated)
- [ ] Response accuracy > 90% (based on feedback)
- [ ] Integration with BI tools (future)

---

## Sign-Off

- [ ] Backend Lead: Code reviewed and approved
- [ ] Frontend Lead: UI reviewed and approved
- [ ] Data Lead: Database changes approved
- [ ] Security Lead: Security review passed
- [ ] Product Owner: Feature tested and accepted

**Deployment Date:** _______________

**Deployed By:** _______________

**Rollback Plan Reviewed:** _______________

---

## Quick Reference

### Start Backend
```bash
export GEMINI_API_KEY="your-key"
go run cmd/server/main.go
```

### Generate Embeddings
```bash
./bin/generate-embeddings \
  --tenant=YOUR_TENANT_ID \
  --datasource=YOUR_DATASOURCE_ID
```

### Test API
```bash
curl -X POST http://localhost:8080/api/nlq/ask \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: ..." \
  -H "X-Tenant-Datasource-ID: ..." \
  -d '{"question": "How is revenue calculated?"}'
```

### Frontend URLs
- NLQ Interface: `http://localhost:3000/nlq`
- Admin Config: `http://localhost:3000/admin/llm`

### Troubleshooting
- No embeddings: Run `./bin/generate-embeddings`
- API errors: Check `GEMINI_API_KEY` is set
- Slow queries: Rebuild vector index with more lists
- 404 on `/nlq/ask`: Verify route registered in backend

---

**Status: □ Not Started | ⧗ In Progress | ✓ Complete**
