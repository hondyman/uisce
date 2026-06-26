# Security Subsystem Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              FRONTEND (React/TypeScript)                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────────┐         ┌──────────────────────┐                  │
│  │ AccessRulesPage     │────────▶│ AccessRuleEditorPage │                  │
│  │ - List rules        │         │ - Edit tab           │                  │
│  │ - Filter/search     │         │ - Preview tab        │                  │
│  │ - Create/edit links │         │ - Test tab           │                  │
│  └─────────────────────┘         └──────────────────────┘                  │
│           │                                │                                 │
│           │                                │                                 │
│           ▼                                ▼                                 │
│  ┌────────────────────────────────────────────────┐                         │
│  │         API Client (api/accessRules.ts)        │                         │
│  │  - listRules()   - createRule()                │                         │
│  │  - getRule()     - updateRule()                │                         │
│  │  - validateRule()  - getRuleImpact()           │                         │
│  └────────────────────────────────────────────────┘                         │
│           │                                                                  │
└───────────┼──────────────────────────────────────────────────────────────────┘
            │ HTTP/JSON
            ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         BACKEND (Go)                                         │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌──────────────────────────────────────────────────────────────────┐       │
│  │               API Layer (internal/api)                           │       │
│  │  ┌────────────────────────────────────────────────────────────┐  │       │
│  │  │ SecurityRulesHandler                                       │  │       │
│  │  │  - ListRules()      → GET /security/rules                 │  │       │
│  │  │  - GetRule()        → GET /security/rules/{id}            │  │       │
│  │  │  - CreateRule()     → POST /security/rules                │  │       │
│  │  │  - UpdateRule()     → PUT /security/rules/{id}            │  │       │
│  │  │  - ValidateRuleDsl()→ POST /security/rules/validate       │  │       │
│  │  │  - GetRuleImpact()  → GET /security/rules/{id}/impact     │  │       │
│  │  └────────────────────────────────────────────────────────────┘  │       │
│  └──────────────────────────────────────────────────────────────────┘       │
│           │                                                                  │
│           ▼                                                                  │
│  ┌──────────────────────────────────────────────────────────────────┐       │
│  │            Service Layer (internal/services)                     │       │
│  │  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────┐  │       │
│  │  │ AccessRuleService│  │  DslValidator    │  │ImpactAnalyzer│  │       │
│  │  │ - List()         │  │  - Validate()    │  │ - ComputeImpact│ │       │
│  │  │ - Get()          │  │  - Parse()       │  │ - GetTerms() │  │       │
│  │  │ - Create()       │◀─┤  - CheckFields() │  │ - GetApis()  │  │       │
│  │  │ - Update()       │  │  - ToSQL()       │  │ - GetBI()    │  │       │
│  │  │ - ValidateDsl()  │  └──────────────────┘  │ - GetAI()    │  │       │
│  │  │ - GetImpact()    │           │             └──────────────┘  │       │
│  │  └──────────────────┘           │                      │         │       │
│  │           │                      ▼                      │         │       │
│  │           │         ┌────────────────────┐             │         │       │
│  │           │         │   DSL Parser       │             │         │       │
│  │           │         │  (internal/dsl)    │             │         │       │
│  │           │         │  - Parse()         │             │         │       │
│  │           │         │  - ToSQL()         │             │         │       │
│  │           │         │  - GetFields()     │             │         │       │
│  │           │         └────────────────────┘             │         │       │
│  │           │                                            │         │       │
│  └───────────┼────────────────────────────────────────────┼─────────┘       │
│              │                                            │                 │
│              ▼                                            ▼                 │
│  ┌──────────────────────────────────────────────────────────────────┐       │
│  │          Repository Layer (internal/repository)                  │       │
│  │  ┌────────────────────────────────────────────────────────────┐  │       │
│  │  │ AccessRuleRepository                                       │  │       │
│  │  │  - List()      → SELECT with filters                      │  │       │
│  │  │  - Get()       → SELECT by ID                             │  │       │
│  │  │  - Create()    → INSERT with JSONB column_masks           │  │       │
│  │  │  - Update()    → UPDATE with version check                │  │       │
│  │  │  - GetByBusinessObjectAndGroups() → Security enforcement  │  │       │
│  │  └────────────────────────────────────────────────────────────┘  │       │
│  └──────────────────────────────────────────────────────────────────┘       │
│              │                                                               │
└──────────────┼───────────────────────────────────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         DATABASE (PostgreSQL)                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌───────────────────────────────────────────────────────────┐              │
│  │ access_rule                                               │              │
│  ├───────────────────────────────────────────────────────────┤              │
│  │ id (PK)                           TEXT                    │              │
│  │ tenant_id                         TEXT                    │              │
│  │ business_object_id                TEXT                    │              │
│  │ group_dn                          TEXT                    │              │
│  │ access_level                      TEXT (NONE/READ/WRITE) │              │
│  │ status                            TEXT (DRAFT/APPROVED..) │              │
│  │ row_filter_dsl                    TEXT                    │              │
│  │ column_masks                      JSONB                   │              │
│  │ applies_to_apis                   BOOLEAN                 │              │
│  │ applies_to_bi                     BOOLEAN                 │              │
│  │ applies_to_ai                     BOOLEAN                 │              │
│  │ created_by, created_at, ...       metadata                │              │
│  └───────────────────────────────────────────────────────────┘              │
│                                                                              │
│  Indexes:                                                                    │
│  - idx_access_rule_tenant (tenant_id)                                       │
│  - idx_access_rule_bo (business_object_id)                                  │
│  - idx_access_rule_group (group_dn)                                         │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                    TEMPORAL WORKFLOWS (Promotion)                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌────────────────────────────────────────────────────────────┐             │
│  │ PromoteAccessRuleWorkflow                                  │             │
│  └────────────────────────────────────────────────────────────┘             │
│         │                                                                    │
│         ├─▶ LoadRuleActivity                                                │
│         │                                                                    │
│         ├─▶ ValidateRuleSyntaxActivity                                      │
│         │                                                                    │
│         ├─▶ ImpactAnalysisActivity                                          │
│         │                                                                    │
│         ├─▶ RunSecurityTestsActivity                                        │
│         │                                                                    │
│         ├─▶ ⏸️  Wait for approval signal (7 days timeout)                   │
│         │                                                                    │
│         ├─▶ PromoteRuleActivity                                             │
│         │                                                                    │
│         └─▶ EmitAuditAndInvalidateCacheActivity                             │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                    RUNTIME SECURITY ENFORCEMENT                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  User Request                                                                │
│       │                                                                      │
│       ├─▶ 1. Extract Principal (userID + groups) from JWT/session           │
│       │                                                                      │
│       ├─▶ 2. Resolve rules for (principal.groups × business_object_id)      │
│       │        - Check cache first                                          │
│       │        - Query DB if cache miss                                     │
│       │                                                                      │
│       ├─▶ 3. Compose AccessDecision                                         │
│       │        - Combine row predicates with OR                             │
│       │        - Pick max access level                                      │
│       │        - Pick most restrictive masks                                │
│       │                                                                      │
│       ├─▶ 4. Inject row predicate into SQL WHERE clause                     │
│       │        - Original: SELECT * FROM instances WHERE bo_id = ?          │
│       │        - Modified: SELECT * FROM instances                          │
│       │                    WHERE bo_id = ? AND (region = 'EMEA')            │
│       │                                                                      │
│       ├─▶ 5. Execute query                                                  │
│       │                                                                      │
│       ├─▶ 6. Apply column masks to result                                   │
│       │        - HIDE: Remove field                                         │
│       │        - MASK: Obfuscate value                                      │
│       │                                                                      │
│       └─▶ 7. Return secured data to user                                    │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘

LEGEND:
  ───▶   Data flow
  ◀───   Dependency
  ⏸️     Wait/signal point
  ┌──┐   Component boundary
```

## Data Flow Example

### Creating a Rule

```
Frontend          API Handler       Service          Validator       Repository      DB
   │                  │                │                 │               │           │
   ├─POST /rules────▶│                │                 │               │           │
   │                  ├─CreateRule()─▶│                 │               │           │
   │                  │                ├─ValidateDsl()─▶│               │           │
   │                  │                │                 ├─Parse DSL    │           │
   │                  │                │                 ├─Check fields │           │
   │                  │                │                 │   vs catalog │           │
   │                  │                │◀────Valid───────┤               │           │
   │                  │                ├─Create()───────────────────────▶│           │
   │                  │                │                 │               ├─INSERT──▶│
   │                  │                │                 │               │◀─ID──────┤
   │                  │                │◀──────────Rule──────────────────┤           │
   │                  │◀─────201───────┤                 │               │           │
   │◀─────Rule────────┤                │                 │               │           │
```

### Querying with Security

```
Client           Auth MW       BO Service      Security Repo     Cache        DB
   │                │               │                 │            │          │
   ├─GET /bo/123───▶│               │                 │            │          │
   │                ├─Extract groups│                 │            │          │
   │                ├─Store Principal                 │            │          │
   │                │◀──────────────┤                 │            │          │
   │                ├───────────────▶│                 │            │          │
   │                │               ├─ResolveAccess()─┤            │          │
   │                │               │                 ├─Check cache▶│          │
   │                │               │                 │◀───MISS─────┤          │
   │                │               │                 ├─GetRules()─────────────▶│
   │                │               │                 │             │◀─Rules───┤
   │                │               │                 ├─ComposeDecision        │
   │                │               │                 ├─Store cache────────────▶│
   │                │               │◀──Decision──────┤             │          │
   │                │               ├─Build SQL with predicate      │          │
   │                │               ├─Query()─────────────────────────────────▶│
   │                │               │◀──────────────────────────Instances──────┤
   │                │               ├─ApplyMasks()    │             │          │
   │                │◀─Secured data─┤                 │             │          │
   │◀───JSON────────┤               │                 │             │          │
```

## Key Architectural Principles

1. **Separation of Concerns**
   - API layer handles HTTP
   - Service layer handles business logic
   - Repository layer handles data access

2. **Security by Default**
   - All queries go through security enforcement
   - No direct database access without principal

3. **Performance First**
   - Cache composed decisions
   - Predicate pushdown to database
   - Lazy graph traversal for impact

4. **Governance & Auditability**
   - All changes logged
   - Promotion workflow for production
   - Version tracking on rules

5. **Extensibility**
   - Pluggable DSL parser
   - Multiple mask types
   - Scope-based enforcement

6. **Type Safety**
   - OpenAPI contract
   - Strong typing in Go
   - TypeScript on frontend
