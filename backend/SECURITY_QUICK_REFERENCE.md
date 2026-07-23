# Security Subsystem - Quick Reference

## 🎯 Key Files

### Backend
```
internal/
├── api/security_rules_handler.go      # REST endpoints
├── services/
│   ├── access_rule_service.go         # Business logic
│   ├── dsl_validator.go               # DSL validation
│   └── impact_analyzer.go             # Impact analysis
├── repository/access_rule_repository.go # Database access
├── dsl/parser.go                       # DSL parser
├── models/access_rule.go              # Data models
└── activities/access_rule_activities.go # Temporal activities

workflows/access_rule_promotion.go      # Promotion workflow
api/openapi-security-rules.yaml         # API spec
```

### Frontend
```
src/
├── api/accessRules.ts                 # API client
└── features/security/
    ├── pages/
    │   ├── AccessRulesPage.tsx        # List view
    │   └── AccessRuleEditorPage.tsx   # Editor with tabs
    └── components/
        ├── RulePreview.tsx            # Preview component
        └── RuleTest.tsx               # Test component
```

## 🔑 Quick Commands

### Create Rule (cURL)
```bash
curl -X POST http://localhost:8080/security/rules \
  -H "Content-Type: application/json" \
  -d @rule.json
```

### Validate DSL
```bash
curl -X POST http://localhost:8080/security/rules/validate \
  -d '{"businessObjectId":"bo:portfolio","rowFilterDsl":"region='\''EMEA'\''"}' \
  -H "Content-Type: application/json"
```

### Get Impact
```bash
curl http://localhost:8080/security/rules/{ruleId}/impact
```

### Run Tests
```bash
# Backend tests
cd backend
go test ./internal/services -v

# Frontend lint
cd frontend
pnpm lint
```

## 📐 DSL Cheatsheet

| Expression | Example |
|------------|---------|
| Equality | `region = 'EMEA'` |
| Inequality | `status != 'deleted'` |
| AND | `region = 'EMEA' AND status = 'active'` |
| OR | `region = 'EMEA' OR region = 'APAC'` |
| IN | `region IN ('EMEA', 'APAC')` |
| NULL | `deleted_at IS NULL` |
| LIKE | `name LIKE '%Smith%'` |
| Nested | `(a = 1 OR a = 2) AND b = 3` |

## 🔐 Access Levels

| Level | Description |
|-------|-------------|
| NONE | No access |
| READ | Query only |
| WRITE | Query + mutate |

## 🎭 Mask Types

| Type | Effect |
|------|--------|
| NONE | No masking |
| MASK | Obfuscate value (e.g., `***-**-1234`) |
| HIDE | Remove field entirely |

## 🔄 Rule Composition

When user in multiple groups:
- **Row filters:** Combined with OR
- **Access level:** Highest wins
- **Masks:** Most restrictive wins (HIDE > MASK > NONE)

## 🎬 Workflow States

1. **DRAFT** → Rule being created
2. **REVIEW** → Awaiting approval
3. **APPROVED** → Active in production
4. **DEPRECATED** → Replaced/retired

## 🚀 Promotion Flow

```
[DRAFT] → Validate → Impact → Test → Approval → [APPROVED]
```

## 📊 Metrics to Monitor

- `security_rule_resolution_ms` - Rule lookup time
- `security_sql_rewrite_ms` - Query rewrite time
- `security_cache_hit_ratio` - Cache effectiveness
- `security_filtered_rows` - Rows filtered

## 🛠️ Common Tasks

### Add New Rule via UI
1. Navigate to `/security/access-rules`
2. Click "New Rule"
3. Fill form (tenant, BO, group, DSL)
4. Validate DSL
5. Preview impact
6. Test with sample group
7. Save

### Promote Rule to Production
```go
params := workflows.PromoteRuleParams{
    RuleID:      "rule-123",
    TargetEnv:   "prod",
    RequestedBy: "admin@example.com",
}
we, _ := temporalClient.ExecuteWorkflow(ctx, opts, workflows.PromoteAccessRuleWorkflow, params)

// Later, approve:
temporalClient.SignalWorkflow(ctx, "promote-rule-123", "", "approval-decision", true)
```

### Debug Security Issue
1. Check user's groups: Query LDAP
2. List rules for BO + groups: `GET /security/rules?businessObjectId=X&groupDn=Y`
3. Check composed decision: Add logging in `ComposeAccessDecision`
4. Verify SQL injection: Check query logs
5. Test rule: Use "Test" tab in UI

## 🔍 Troubleshooting

| Issue | Solution |
|-------|----------|
| DSL validation fails | Check field names against catalog |
| Rule not applying | Verify status is APPROVED |
| Cache stale | Invalidate cache after rule change |
| Slow queries | Add index on filtered columns |
| Wrong access level | Check all user groups, verify composition |

## 📚 Additional Resources

- Full docs: [SECURITY_SUBSYSTEM_README.md](SECURITY_SUBSYSTEM_README.md)
- OpenAPI spec: [api/openapi-security-rules.yaml](api/openapi-security-rules.yaml)
- Integration guide: [INTEGRATION_GUIDE.go](INTEGRATION_GUIDE.go)
- Test examples: `internal/services/*_test.go`
