# Phase 4: Advanced Features & Scale - Implementation Guide

**Status:** Planning & Initiation  
**Date:** February 20, 2026  
**Priority Features:** Rule Templates → Bulk Operations → Rule Composition

---

## Phase 4 Overview

Phase 4 extends Phase 3's rule engine with enterprise-grade features: template reusability, bulk operations for efficiency, rule composition for complex logic, event streaming integration, and ML-assisted suggestions.

**Key Goals:**
- ✅ Reduce rule creation time via templates (target: 50% faster)
- ✅ Enable bulk operations (create/update/delete 1000+ rules)
- ✅ Support nested rule composition (complex business logic)
- ✅ Integrate with Redpanda event streaming
- ✅ Add search/filtering for rule discovery
- ✅ ML suggestions for rule optimization

---

## Phase 4 Features Breakdown

### Feature 1: Rule Templates (Priority 1)

**Purpose:** Reusable rule patterns for common scenarios (e.g., "Weekend Override", "Holiday Classification", "Region-Based Rules")

**Architecture:**

```sql
-- New table: edm.rule_templates
CREATE TABLE edm.rule_templates (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  business_object VARCHAR(100),
  name VARCHAR(255),
  description TEXT,
  category VARCHAR(100),  -- 'weekend', 'holiday', 'region', etc.
  
  -- Template structure
  base_rule_steps JSONB,  -- Common priority steps
  parameter_schema JSONB, -- JSON Schema for template params
  
  -- Governance
  status VARCHAR(20),     -- approved, draft, deprecated
  created_at TIMESTAMP,
  created_by UUID,
  updated_at TIMESTAMP,
  updated_by UUID,
  
  CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  CONSTRAINT fk_bo FOREIGN KEY (business_object) REFERENCES business_objects(name)
);

-- Index template discovery
CREATE INDEX idx_templates_bo_status ON edm.rule_templates(business_object, status);
CREATE INDEX idx_templates_category ON edm.rule_templates(category);
```

**API Endpoints (4 new):**
- `POST /api/v1/templates` - Create template
- `GET /api/v1/templates` - List templates (filtered by BO, category)
- `POST /api/v1/templates/{id}/create-rule` - Instantiate rule from template
- `GET /api/v1/templates/{id}/preview` - Show template with sample parameters

**Example Template:**
```json
{
  "name": "Weekend Override Template",
  "category": "weekend",
  "businessObject": "calendar",
  "baseRuleSteps": [
    {
      "priority": 1,
      "condition": {
        "semanticTerm": "IsBusinessDay",
        "operator": "equals",
        "value": false
      },
      "confidence": 90
    },
    {
      "priority": 2,
      "condition": {
        "semanticTerm": "RegionCode",
        "operator": "in",
        "value": "{{regions}}"  // Parameter placeholder
      },
      "confidence": "{{confidence}}"
    }
  ],
  "parameterSchema": {
    "type": "object",
    "properties": {
      "regions": {
        "type": "string",
        "description": "Comma-separated region codes"
      },
      "confidence": {
        "type": "number",
        "minimum": 0,
        "maximum": 100
      }
    },
    "required": ["regions"]
  }
}
```

**Implementation Effort:** 2-3 days
- Database schema + indexes
- 4 REST endpoints
- Template parameter resolution engine
- Frontend template browser component
- Template instantiation workflow

---

### Feature 2: Bulk Operations (Priority 2)

**Purpose:** Create/update/delete multiple rules in single API call (target: 1000+ rules/request)

**Architecture:**

```sql
-- Track bulk jobs
CREATE TABLE edm.bulk_operations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  operation_type VARCHAR(20),  -- 'create', 'update', 'delete'
  status VARCHAR(20),           -- 'pending', 'processing', 'completed', 'failed'
  
  total_items INT,
  processed_items INT,
  failed_items INT,
  
  request_payload JSONB,
  error_results JSONB,  -- Errors from failed items
  
  created_at TIMESTAMP,
  created_by UUID,
  completed_at TIMESTAMP,
  
  CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

CREATE INDEX idx_bulk_ops_status ON edm.bulk_operations(status, completed_at DESC);
```

**API Endpoints (3 new):**
- `POST /api/v1/rules/bulk-create` - Create multiple rules
- `POST /api/v1/rules/bulk-update` - Update multiple rules
- `GET /api/v1/bulk-operations/{id}` - Track bulk job progress/errors

**Example: Bulk Create**
```bash
POST /api/v1/rules/bulk-create
{
  "rules": [
    {
      "businessObject": "calendar",
      "name": "US Weekend Rule",
      "steps": [...],
      "tags": ["us", "weekend"]
    },
    {
      "businessObject": "calendar",
      "name": "GB Holiday Rule",
      "steps": [...],
      "tags": ["gb", "holiday"]
    },
    ... up to 1000 rules
  ],
  "onError": "continue"  // or "rollback"
}

Response: 202 Accepted
{
  "jobId": "bulk-op-uuid",
  "status": "processing",
  "statusUrl": "/api/v1/bulk-operations/bulk-op-uuid"
}

// Track progress
GET /api/v1/bulk-operations/bulk-op-uuid
{
  "status": "completed",
  "totalItems": 1000,
  "processedItems": 1000,
  "failedItems": 0,
  "results": [
    { "ruleId": "...", "success": true },
    ...
  ]
}
```

**Implementation Effort:** 3-4 days
- Bulk operations table + indexes
- Async job queue (Redis/database pooling)
- 3 REST endpoints
- Progress tracking system
- Error aggregation & reporting
- Frontend bulk operation monitor

---

### Feature 3: Rule Composition (Priority 3)

**Purpose:** Create complex rules by combining existing rules (AND/OR/NOT logic)

**Architecture:**

```sql
-- Composition tree (rules can contain other rules)
CREATE TABLE edm.rule_compositions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  parent_rule_id UUID NOT NULL,
  child_rule_id UUID NOT NULL,
  
  operator VARCHAR(10),  -- 'AND', 'OR', 'NOT'
  priority INT,
  
  created_at TIMESTAMP,
  
  CONSTRAINT fk_parent FOREIGN KEY (parent_rule_id) REFERENCES edm.rules(id) ON DELETE CASCADE,
  CONSTRAINT fk_child FOREIGN KEY (child_rule_id) REFERENCES edm.rules(id) ON DELETE CASCADE,
  CONSTRAINT unique_composition UNIQUE(parent_rule_id, child_rule_id)
);

CREATE INDEX idx_compositions_parent ON edm.rule_compositions(parent_rule_id);
CREATE INDEX idx_compositions_child ON edm.rule_compositions(child_rule_id);
```

**UI Component: Composition Builder**
```
┌─────────────────────────────────────┐
│   Rule Composition Builder          │
├─────────────────────────────────────┤
│                                     │
│  ┌─────────────────────────────┐   │
│  │ Parent Rule: "Complex Logic"│   │
│  └──────────────┬──────────────┘   │
│                 │                   │
│         ┌───────┴────────┐          │
│         │                │          │
│      [AND]            [OR]          │
│      /    \           /  \          │
│     ▼     ▼          ▼    ▼        │
│   Rule   Rule     Rule  Rule       │
│   1      2        3      4         │
│                                     │
│  [+ Add Child Rule]                │
│  [Preview Composition]             │
│  [Save]                            │
│                                     │
└─────────────────────────────────────┘
```

**Example: Complex Rule**
```json
{
  "name": "Weekend + Holiday Override",
  "composition": {
    "operator": "AND",
    "rules": [
      {
        "ruleId": "is-business-day-false",
        "operator": "OR",
        "sub-rules": [
          "weekend-rule",
          "holiday-rule"
        ]
      },
      {
        "ruleId": "region-specific",
        "operator": "AND",
        "sub-rules": [
          "us-holidays",
          "gb-bank-holidays"
        ]
      }
    ]
  }
}
```

**Execution Logic:**
```go
func (e *Engine) EvaluateComposition(ctx context.Context, ruleID string, data interface{}) (bool, float64) {
    rule := e.db.GetRule(ruleID)
    
    if !rule.IsComposition {
        // Base case: evaluate single rule
        return e.EvaluateRule(ctx, ruleID, data)
    }
    
    // Recursive: evaluate composition
    compositions := e.db.GetCompositions(ruleID)
    
    switch compositions.Operator {
    case "AND":
        for _, child := range compositions.Children {
            ok, conf := e.EvaluateComposition(ctx, child.RuleID, data)
            if !ok { return false, 0 }
            // Aggregate confidence
        }
        return true, avgConfidence
        
    case "OR":
        for _, child := range compositions.Children {
            ok, conf := e.EvaluateComposition(ctx, child.RuleID, data)
            if ok { return true, conf }
        }
        return false, 0
        
    case "NOT":
        ok, conf := e.EvaluateComposition(ctx, compositions.Children[0].RuleID, data)
        return !ok, conf
    }
}
```

**Implementation Effort:** 3-4 days
- Rule composition table + indexes
- Recursive composition evaluation engine
- Composition UI builder component
- 2 new endpoints: create composition, get composition tree
- Validation: prevent circular dependencies

---

### Feature 4: Event Publishing to Redpanda (Priority 4)

**Purpose:** Stream rule mutations to Redpanda for real-time subscribers (dashboards, audit systems, ML pipelines)

**Architecture:**

```go
// Event types
type RuleEvent struct {
    EventID       string    `json:"eventId"`
    EventType     string    `json:"eventType"`  // CREATED, PUBLISHED, PROMOTED, etc.
    RuleID        string    `json:"ruleId"`
    TenantID      string    `json:"tenantId"`
    Actor         string    `json:"actor"`
    Timestamp     time.Time `json:"timestamp"`
    Data          interface{} `json:"data"`
}

// Redpanda topics
Topics:
  - "rule-events"           (all rule mutations)
  - "rule-created"          (created rules)
  - "rule-published"        (published to testing)
  - "rule-promoted"         (promoted between stages)
  - "approval-requested"    (approval workflows)
  - "rule-executed"         (simulation results)
```

**Producer Integration:**
```go
// In rules_handler_impl.go
func (h *RuleHandler) PublishRule(w http.ResponseWriter, r *http.Request) {
    // ... existing logic ...
    
    // Emit event to Redpanda
    event := RuleEvent{
        EventID:    uuid.New().String(),
        EventType:  "RULE_PUBLISHED",
        RuleID:     rule.ID,
        TenantID:   tenantID,
        Actor:      userID,
        Timestamp:  time.Now(),
        Data: map[string]interface{}{
            "version": rule.Version,
            "status":  rule.Status,
        },
    }
    
    h.eventProducer.Publish("rule-events", event)
}
```

**Consumer Example (Real-time Dashboard):**
```golang
// Subscribe to rule events
consumer := redpanda.NewConsumer("rule-events", "dashboard-group")

for event := range consumer.Messages() {
    switch event.EventType {
    case "RULE_PUBLISHED":
        dashboard.UpdateRuleStatus(event.RuleID, "testing")
        
    case "RULE_PROMOTED":
        dashboard.UpdateRuleStatus(event.RuleID, event.Data.ToStage)
    }
}
```

**Implementation Effort:** 2-3 days
- Redpanda client setup
- Event producer integration (in handlers)
- 6 event types per mutation
- Example consumers (dashboard, audit system)
- Monitoring topic lag

---

### Feature 5: Advanced Search & Filtering

**Purpose:** Discover rules by tags, content, similarity, approval status

**Architecture:**

```sql
-- Rule tags for classification
CREATE TABLE edm.rule_tags (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rule_id UUID NOT NULL,
  tag VARCHAR(50),  -- 'weekend', 'holiday', 'us-only', 'high-priority', etc.
  
  CONSTRAINT fk_rule FOREIGN KEY (rule_id) REFERENCES edm.rules(id) ON DELETE CASCADE
);

-- Full-text search index on description
CREATE INDEX idx_rules_description_fts ON edm.rules USING GIN(to_tsvector('english', description));

-- Rule similarity index (for ML recommendations)
-- Uses vector embeddings (Phase 4+)
CREATE TABLE edm.rule_embeddings (
  rule_id UUID PRIMARY KEY,
  embedding vector(384),  -- pgvector extension
  
  CONSTRAINT fk_rule FOREIGN KEY (rule_id) REFERENCES edm.rules(id) ON DELETE CASCADE
);

CREATE INDEX idx_rule_embeddings ON edm.rule_embeddings USING ivfflat (embedding vector_cosine_ops);
```

**Search APIs:**
```bash
# By tag
GET /api/v1/rules?tag=weekend
GET /api/v1/rules?tags=us-only,holiday

# Full-text search
GET /api/v1/rules/search?q=business%20day

# Filter by multiple conditions
GET /api/v1/rules?
  businessObject=calendar&
  status=production&
  approvalStatus=approved&
  tag=weekend&
  createdAfter=2026-01-01&
  confidenceMin=80

# Find similar rules (vector similarity)
GET /api/v1/rules/{id}/similar
```

**Response:**
```json
{
  "results": [
    {
      "id": "rule-uuid",
      "name": "Weekend Override",
      "businessObject": "calendar",
      "status": "production",
      "tags": ["weekend", "us-only"],
      "similarity": 0.87  // If similarity search
    }
  ],
  "total": 42,
  "page": 1,
  "pageSize": 20
}
```

**Implementation Effort:** 2-3 days
- Rule tags schema + indexes
- Full-text search endpoint
- Vector similarity search (if pgvector installed)
- Tag management UI
- Search result highlighting

---

### Feature 6: ML-Assisted Suggestions (Priority 5)

**Purpose:** Recommend rule optimizations and suggest missing rules based on execution patterns

**Architecture:**

```
┌─────────────────────────────────┐
│   Rule Execution History        │
│   (edm.rule_execution_history)  │
└──────────┬──────────────────────┘
           │
           ▼
┌─────────────────────────────────┐
│   ML Pipeline                   │
│   ├── Extract features          │
│   ├── Pattern detection         │
│   ├── Confidence analysis       │
│   └── Generate suggestions      │
└──────────┬──────────────────────┘
           │
           ▼
┌─────────────────────────────────┐
│   Rule Suggestions              │
│   ├── Merge rules               │
│   ├── Adjust confidence         │
│   ├── Add missing conditions    │
│   └── Refactor composition      │
└─────────────────────────────────┘
```

**Suggestion Types:**

1. **Merge Suggestions**
   - Two rules always match together → merge
   - Recommendation: Merge Rule A + Rule B for 20% efficiency gain

2. **Confidence Tuning**
   - Rule matches 95% of cases → increase confidence
   - Recommendation: Increase Rule A confidence from 75 to 92

3. **Completeness**
   - Unmatched records detected → new condition needed
   - Recommendation: Add condition for dates > 2026-06-01

4. **Performance**
   - Rule evaluation slow → optimize order
   - Recommendation: Move frequent condition to priority 1

**API:**
```bash
GET /api/v1/rules/{id}/suggestions
{
  "suggestions": [
    {
      "type": "merge",
      "targetRuleId": "other-rule-uuid",
      "reason": "Always match together",
      "expectedGain": "20% performance improvement",
      "action": "POST /api/v1/rules/{id}/merge?with=other-rule-uuid"
    },
    {
      "type": "confidence_tune",
      "recommendedConfidence": 92,
      "reason": "Matches 95% of test data",
      "action": "PUT /api/v1/rules/{id} with confidence=92"
    }
  ]
}
```

**ML Model Training:**
```python
# Periodic batch job (daily)
from sklearn.ensemble import IsolationForest
import numpy as np

def analyze_rule_patterns(rule_execution_history):
    """Detect anomalies and patterns in rule execution"""
    
    features = extract_features(rule_execution_history)
    
    # Detect anomalies (rules that underperform)
    detector = IsolationForest()
    anomalies = detector.fit_predict(features)
    
    # Generate suggestions
    suggestions = []
    for anomaly in anomalies:
        rule = rule_execution_history[anomaly]
        if rule.match_rate < 0.8:
            suggestions.append({
                "type": "completeness",
                "ruleId": rule.id,
                "matchRate": rule.match_rate,
                "recommendation": "Add conditions for patterns"
            })
    
    return suggestions
```

**Implementation Effort:** 4-5 days
- ML pipeline setup (Python microservice)
- Feature extraction from execution history
- Pattern detection algorithms
- Suggestion generation
- REST API for ML service
- Frontend suggestion display

---

## Implementation Roadmap

### Week 1: Rule Templates
**Mon-Wed: Development**
- Database schema + migrations
- 4 REST endpoints
- Template parameter resolution
- Template browser UI

**Thu: Testing**
- E2E tests for template workflow
- Template instantiation tests

**Fri: Integration**
- Wire template creation to existing rules
- Documentation

### Week 2: Bulk Operations
**Mon-Tue: Development**
- Bulk operation job queue
- 3 REST endpoints
- Progress tracking

**Wed-Thu: Testing**
- 1000+ rule bulk create test
- Error handling & rollback

**Fri: Optimization**
- Performance tuning

### Week 3: Rule Composition
**Mon-Tue: Development**
- Composition tree schema
- Recursive evaluation engine
- UI builder

**Wed-Thu: Testing**
- Circular dependency validation
- Complex composition tests

**Fri: Integration**

### Week 4: Event Publishing & Advanced Features
**Mon-Wed: Event Publishing**
- Redpanda integration
- Event producer setup
- Consumer examples

**Thu-Fri: Search/ML Foundation**
- Tag schema
- Full-text search
- ML pipeline setup

---

## Performance Targets

| Feature | Target | Notes |
|---------|--------|-------|
| Template instantiation | <100ms | Create rule from template |
| Bulk create (1000 rules) | <5s | Async job |
| Composition evaluation | <200ms | With caching |
| Event publish | <10ms | Fire and forget to Redpanda |
| Search latency | <50ms | Full-text or tag search |
| ML suggestion generation | <1s | Batch job, not real-time |

---

## Database Impact

**New Tables:**
- edm.rule_templates (template definitions)
- edm.rule_compositions (composition tree)
- edm.bulk_operations (job tracking)
- edm.rule_tags (classification)
- edm.rule_embeddings (ML vectors)

**Total new tables:** 5  
**Total new columns:** ~30  
**Storage impact:** ~500KB for templates, 50MB for embeddings (if 10K rules)

---

## API Summary (Phase 4 Additions)

**Templates (4 endpoints)**
- POST /api/v1/templates
- GET /api/v1/templates
- POST /api/v1/templates/{id}/create-rule
- GET /api/v1/templates/{id}/preview

**Bulk Operations (3 endpoints)**
- POST /api/v1/rules/bulk-create
- POST /api/v1/rules/bulk-update
- GET /api/v1/bulk-operations/{id}

**Composition (2 endpoints)**
- POST /api/v1/rules/{id}/compositions
- GET /api/v1/rules/{id}/compositions

**Search & Discovery (3 endpoints)**
- GET /api/v1/rules/search
- GET /api/v1/rules/{id}/similar
- GET /api/v1/tags

**Suggestions (1 endpoint)**
- GET /api/v1/rules/{id}/suggestions

**Total new Phase 4 endpoints:** 13

---

## Deployment Notes

- Redpanda cluster required (separate infrastructure)
- pgvector extension needed for similarity search (optional)
- ML service can run as separate microservice or batch job
- Template migrations need to be backward compatible

---

## Success Criteria

- ✅ All 13 new endpoints implemented
- ✅ E2E tests for each feature
- ✅ Performance targets met
- ✅ Bulk operations handle 1000+ rules
- ✅ Rule composition supports 5+ depth
- ✅ Events published to Redpanda in real-time
- ✅ Search finds 90%+ of intended rules
- ✅ ML suggestions with >80% accuracy

---

**Next: Select Feature 1 (Templates) to begin implementation**
