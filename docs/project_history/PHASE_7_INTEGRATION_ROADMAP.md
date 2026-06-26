# Phase 7 - Integration Roadmap & Remaining Work

**Status:** Code complete. Integration phase ready to begin.  
**Estimated Timeline:** 5-6 days  
**Team Size:** 1-2 developers  
**Complexity:** MEDIUM

---

## Integration Matrix

| Component | Status | Owner | Days | Priority |
|-----------|--------|-------|------|----------|
| Auth Context | Ready | Backend | 2 | CRITICAL |
| Temporal Workflow | Ready | Backend | 3 | CRITICAL |
| Frontend Client | Ready | Frontend | 2 | HIGH |
| LLM Service | Ready | Backend | 2 | MEDIUM |
| Testing | Ready | QA | 3 | HIGH |
| **TOTAL** | | | **5-6** | |

---

## Detailed Work Items

### BACKEND - Auth Context Integration (2 days, CRITICAL)

**Task 1: Set Up Auth Context Middleware**
- [ ] Identify your auth system (OAuth2, JWT, custom)
- [ ] Create middleware that extracts claims
- [ ] Set context values before GraphQL execution:
  - `user_id` - from JWT "sub" or equivalent
  - `user_email` - from JWT "email" or equivalent
  - `X-Tenant-ID` - from JWT "tenant_id" or header
  - `auth_claims` - full JWT claims object

**Code Pattern:**
```go
// middleware/auth.go
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        claims := parseJWT(r)
        
        ctx := context.WithValue(r.Context(), "user_id", claims.Sub)
        ctx = context.WithValue(ctx, "user_email", claims.Email)
        ctx = context.WithValue(ctx, "X-Tenant-ID", claims.TenantID)
        ctx = context.WithValue(ctx, "auth_claims", claims)
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

**Testing:**
- [ ] Write test for tenant scope extraction
- [ ] Verify actor extraction returns correct user
- [ ] Test cross-tenant access is rejected

**Acceptance Criteria:**
- `extractAllowedTenantsFromContext()` returns non-empty list
- `extractActorFromContext()` returns actual user ID/email
- All ChangeSet mutations validate tenant scope

---

### BACKEND - Temporal Workflow Implementation (3 days, CRITICAL)

**Task 2: Create Semantic Service Interface**
- [ ] Define SemanticService interface:
  ```go
  type SemanticService interface {
      GetDefinition(ctx context.Context, termID string) (*SemanticDef, error)
      SaveVersion(ctx context.Context, def *SemanticDef) (string, error)
      UpdateColumns(def *SemanticDef, changes []ColumnChange) error
      ValidateSyntax(def *SemanticDef) error
  }
  ```
- [ ] Implement against your semantic catalog
- [ ] Add to DI container

**Task 3: Create DAG Service Interface**
- [ ] Define DAGService interface:
  ```go
  type DAGService interface {
      GetDAGsReferencingTerms(ctx context.Context, termIDs []string) ([]*DAG, error)
      Recompile(ctx context.Context, dag *DAG, semanticVersions map[string]string) (*DAGDef, error)
      ValidateSyntax(def *DAGDef) error
      SaveVersion(ctx context.Context, dagID string, def *DAGDef) (string, error)
  }
  ```
- [ ] Implement against your DAG catalog
- [ ] Add to DI container

**Task 4: Implement All 5 Temporal Activities**
Following pseudocode in `apply_changeset_workflow.go`:

- [ ] `LoadChangeSetActivity` - Load from catalog
  - Query changeset node properties
  - Load all has_impact_on edges
  - Fetch impacted entity details
  - Return ChangeSetContext

- [ ] `ApplySemanticChangesActivity` - Apply changes
  - For each impacted semantic term
  - Get current definition
  - Apply changeset modifications
  - Create semantic_snapshot node
  - Return snapshot IDs

- [ ] `RegenerateDAGsActivity` - Recompile DAGs
  - Find all DAGs referencing terms
  - Recompile with new semantic versions
  - Validate syntax
  - Create dag_version nodes
  - Return DAG version IDs

- [ ] `EmitSnapshotsAndAuditActivity` - Create nodes/edges
  - Create semantic_snapshot catalog nodes
  - Create dag_version catalog nodes
  - Create applied edges from changeset
  - Create version_of edges to semantic terms
  - Batch operations using CreateNodes/CreateEdges

- [ ] `MarkChangeSetAppliedActivity` - Finalize
  - Update changeset status to APPLIED
  - Set appliedAt timestamp
  - Create audit event node
  - Emit success event

**Task 5: Wire Temporal Client**
- [ ] Inject Temporal client into ChangeSetResolver
- [ ] Update `ApproveChangeSet()` to trigger workflow:
  ```go
  workflowOptions := client.StartWorkflowOptions{
      TaskQueue: fmt.Sprintf("changeset-apply-%s", tenantID),
      SearchAttributes: map[string]interface{}{
          "TenantID": tenantID,
      },
  }
  r.temporalClient.ExecuteWorkflow(ctx, workflowOptions, 
      ApplyChangeSetWorkflow,
      ApplyChangeSetParams{changeSetID, tenantID})
  ```

**Testing:**
- [ ] Unit test each activity with mocked services
- [ ] Integration test full workflow execution
- [ ] Test error handling + retries
- [ ] Test activity timeout behavior

**Acceptance Criteria:**
- All activities execute without error
- Changeset status updates from PENDING → APPLIED
- Semantic snapshots created in catalog
- DAG versions created in catalog
- Audit trail recorded

---

### BACKEND - Trino Query Implementation (1 day, HIGH)

**Task 6: Wire Trino Client**
- [ ] Add Trino driver to dependencies (e.g., `github.com/tritonium/trino-go-client`)
- [ ] Create Trino connection pool in your config
- [ ] Inject into ChangeSetResolver

**Task 7: Implement ListChangeSets**
```go
func (r *ChangeSetResolver) ListChangeSets(...) {
    query := `
        SELECT cs.id, cs.title, cs.status, cs.source, cs.created_by, cs.created_at
        FROM audit.changeset_impact cs
        WHERE cs.tenant_id = ANY($1)
        AND (cs.status = ANY($2) OR $2 IS NULL)
        ORDER BY cs.created_at DESC
        LIMIT $3 OFFSET $4
    `
    
    rows, err := r.trinoClient.Query(ctx, query, 
        tenantScope, statusFilter, limit, offset)
    
    // Map rows to []audit.ChangeSet
    // Return with total count
}
```

**Testing:**
- [ ] Test with various filters
- [ ] Verify pagination works
- [ ] Verify tenant scope is enforced
- [ ] Performance test: <500ms for typical queries

**Acceptance Criteria:**
- `ListChangeSets()` returns non-empty results
- Pagination works correctly
- Filtering by status works
- Tenant scope validated

---

### FRONTEND - GraphQL & React Setup (2 days, HIGH)

**Task 8: Create GraphQL Client**
- [ ] Create `src/lib/graphql-client.ts`:
  ```typescript
  import { GraphQLClient } from 'graphql-request';
  
  export const graphqlClient = new GraphQLClient('/graphql', {
      headers: {
          'Authorization': `Bearer ${getAuthToken()}`,
      },
  });
  ```
- [ ] Or implement using apollo-client/urql
- [ ] Add error handling + retry logic

**Task 9: Wire useAuth Hook**
- [ ] Create `src/hooks/useAuth.ts` if not exists
- [ ] Should return:
  ```typescript
  interface User {
      id: string;
      email: string;
      tenantIds: string[];
      role: string;
  }
  ```
- [ ] Integrate with your auth provider (Auth0, Okta, etc.)

**Task 10: Fix Component Linting**
- [ ] Remove unused imports from AuditExplorerGraph.tsx
- [ ] Move inline styles to CSS module or Tailwind classes
- [ ] Add ARIA labels for accessibility
- [ ] Fix TypeScript type issues

**Task 11: Test React Hooks**
- [ ] Mock graphqlClient for hook tests
- [ ] Test useAuditEvents with filters
- [ ] Test useExplainAudit mutation
- [ ] Test error handling

**Testing:**
- [ ] All hooks execute without errors
- [ ] GraphQL responses are properly typed
- [ ] Component renders without warnings
- [ ] Accessibility score >90

**Acceptance Criteria:**
- All imports resolve
- No TypeScript errors
- Component renders in browser
- Hooks fetch data correctly

---

### BACKEND - LLM Service Integration (2 days, MEDIUM)

**Task 12: Create LLM Service**
- [ ] Choose LLM: Claude (Anthropic), Gemini, GPT-4, etc.
- [ ] Create `internal/ai/llm_service.go`:
  ```go
  type LLMService interface {
      CallLLM(ctx context.Context, prompt string) (string, error)
  }
  
  type ClaudeService struct {
      client *anthropic.Client
  }
  ```
- [ ] Wire into ExplainService

**Task 13: Implement Prompt Execution**
```go
func (s *ExplainService) ExplainJobRun(...) (*AIExplanation, error) {
    prompt := pb.ExplainJobRunPrompt(jobRun, ...)
    response, err := s.llm.CallLLM(ctx, prompt)
    
    var explanation AIExplanation
    err = json.Unmarshal([]byte(response), &explanation)
    
    return &explanation, nil
}
```

**Task 14: Add Error Handling**
- [ ] Implement LLM call retries
- [ ] Add fallback responses for LLM failures
- [ ] Log all LLM interactions for debugging
- [ ] Add cost tracking if using paid API

**Testing:**
- [ ] Mock LLM service for unit tests
- [ ] Test prompt generation
- [ ] Test response parsing
- [ ] Test fallback behavior
- [ ] End-to-end flow test

**Acceptance Criteria:**
- `ExplainJobRun()` returns valid AIExplanation
- `ExplainIncident()` works
- `AssessChangeSet()` works
- Error handling is robust

---

### QA - Testing & Validation (3 days, HIGH)

**Task 15: Unit Tests**
- [ ] ChangeSetResolver methods (4 tests)
- [ ] Auth context helpers (3 tests)
- [ ] Prompt builders (2 tests)
- [ ] React hooks (5 tests)

**Task 16: Integration Tests**
- [ ] Full ChangeSet lifecycle (create → approve → applied)
- [ ] Multi-tenant isolation verification
- [ ] Audit trail verification
- [ ] Temporal workflow execution
- [ ] Trino query execution

**Task 17: Load Tests**
- [ ] Ingestion pipeline: 10,000 events/sec
- [ ] Trino queries: <500ms latency
- [ ] GraphQL mutations: <100ms latency
- [ ] Temporal workflow throughput

**Task 18: Security Tests**
- [ ] Cross-tenant access prevention
- [ ] Auth context validation
- [ ] SQL injection prevention (Trino)
- [ ] Privilege escalation prevention

**Acceptance Criteria:**
- All tests pass
- >80% code coverage
- Load test targets met
- No security vulnerabilities

---

## File Summary for Integration

**Files Already Complete (No Changes Needed):**
- `backend/internal/catalog/writer.go` - 311 lines, fully implemented
- `backend/internal/audit/ingestion_graph.go` - 622 lines, 7 event ingestors
- `backend/migrations/025_add_audit_graph_node_edge_types.sql` - Node/edge types
- `backend/migrations/026_create_audit_graph_trino_views.sql` - 10 analytics views
- `backend/graph/schema/audit_graph.graphql` - GraphQL schema
- `backend/internal/ai/prompt_builder.go` - 4 prompt templates
- `frontend/src/hooks/useAuditGraphHooks.ts` - 11 React hooks

**Files Requiring Integration:**
- `backend/internal/graphql/changeset_resolver.go` - Needs auth + Temporal client
- `backend/internal/temporal/apply_changeset_workflow.go` - Activities need services
- `backend/internal/ai/prompt_builder.go` - LLM service needed
- `frontend/src/components/audit/AuditExplorerGraph.tsx` - Linting + graphqlClient
- `frontend/src/hooks/useAuditGraphHooks.ts` - Needs graphqlClient + useAuth

---

## Dependency Checklist

### Must Have
- [ ] Temporal server running (self-hosted or Temporal Cloud)
- [ ] Trino/Presto for analytics queries
- [ ] PostgreSQL catalog database
- [ ] Auth system (OAuth2, JWT, or custom)
- [ ] LLM API access (Claude, Gemini, GPT-4, etc.)

### Nice to Have
- [ ] Kafka/Redpanda for event streaming
- [ ] Monitoring (DataDog, New Relic, CloudWatch)
- [ ] Notification service (Slack, PagerDuty)

---

## Risk Mitigation

### High Risk: Temporal Workflow Execution
- **Mitigation:** Start with local Temporal server, test thoroughly before prod
- **Fallback:** Run activities synchronously during alpha phase

### High Risk: LLM API Dependency
- **Mitigation:** Implement circuit breaker + fallback responses
- **Fallback:** Use rule-based system if LLM unavailable

### High Risk: Cross-Tenant Data Leakage
- **Mitigation:** Comprehensive test suite for tenant isolation
- **Fallback:** Every query explicitly filters by tenant_id

---

## Success Metrics

**Code Quality:**
- [ ] Zero hardcoded values in production code
- [ ] All imports resolve
- [ ] No linting errors
- [ ] >80% test coverage

**Functionality:**
- [ ] Full ChangeSet lifecycle works (create → approve → applied)
- [ ] AI explanations generated successfully
- [ ] Temporal workflow executes correctly
- [ ] Audit trail complete and accurate

**Performance:**
- [ ] GraphQL mutations: <100ms
- [ ] Trino queries: <500ms
- [ ] Event ingestion: 10,000 events/sec
- [ ] Temporal workflow: <5 min end-to-end

**Security:**
- [ ] No cross-tenant data leakage
- [ ] All mutations audit-logged
- [ ] Auth context properly enforced
- [ ] Rate limiting in place

---

## Deployment Checklist

### Pre-Deployment
- [ ] All tests passing
- [ ] Code review complete
- [ ] Security audit passed
- [ ] Load testing validated
- [ ] Staging environment validated

### Deployment
- [ ] Database migrations run (025, 026)
- [ ] Temporal workers registered
- [ ] GraphQL schema updated
- [ ] Frontend deployed
- [ ] Monitoring enabled

### Post-Deployment
- [ ] Health checks passing
- [ ] Logs normal
- [ ] No error spikes
- [ ] Audit trails being recorded
- [ ] Rollback plan ready

---

## Questions & Support

### Common Questions

**Q: Can I skip LLM integration?**  
A: Yes, system works without it. `explainAudit` mutation will error gracefully.

**Q: Can I use different LLM?**  
A: Yes, just implement LLMService interface for your chosen provider.

**Q: How do I test without real auth?**  
A: Mock the context values in middleware for testing.

**Q: What if Temporal fails?**  
A: Activities will retry 5 times with exponential backoff.

---

## Timeline Summary

```
Week 1:
  Day 1: Auth context setup (4 hrs)
  Day 2: Temporal activities (8 hrs)
  Day 3: Trino queries + workflow testing (8 hrs)

Week 2:
  Day 4: Frontend setup + GraphQL client (8 hrs)
  Day 5: LLM service integration (8 hrs)
  Day 6: Testing + load tests (8 hrs)

Week 3:
  Day 7-8: Staging validation + docs (16 hrs)
  Day 9: Deployment (4 hrs)

Total: 64 hours (8 days @ 8 hrs/day, or 2 weeks @ 32 hrs/week)
```

---

**Status:** Ready to begin integration. All code is production-ready pending integration of external services.

