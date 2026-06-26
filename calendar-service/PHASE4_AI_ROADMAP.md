# Phase 4: AI Holiday & Calendar Intelligence Roadmap

**Status**: Ready to start (can begin parallel to Phase 3 staging validation)  
**Duration**: 3-4 weeks  
**Risk**: 🟡 MEDIUM (new AI dependency, requires OpenAI integration)  
**Architecture**: Temporal workflows + OpenAI API + React UI

---

## Overview

**Goal**: Extend calendar system with AI-powered intelligent scheduling and holiday management

**What it enables**:
- Auto-generate holidays based on region/industry
- AI-assisted availability scheduling
- Holiday conflict detection
- Intelligent capacity recommendations
- Multi-regional holiday synchronization

**Key components**:
1. Holiday AI Module (OpenAI client + Temporal workflows)
2. Holiday Database Schema (PostgreSQL)
3. Holiday React UI (Calendar component)
4. Temporal Activities for AI operations
5. Admin approval workflow

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Frontend (React)                          │
│  ┌────────────────────────────────────────────────────────┐ │
│  │ Calendar Component                                     │ │
│  │  - Holiday display                                     │ │
│  │  - AI suggestion panel (Approve/Reject/Regenerate)    │ │
│  │  - Admin approval workflow                            │ │
│  └────────────────────────────────────────────────────────┘ │
└──────────────────────┬──────────────────────────────────────┘
                       │ HTTP + WebSocket
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                 Backend API (Go)                             │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ Holiday Handler                                      │   │
│  │  - POST /api/v1/holidays/generate (admin only)       │   │
│  │  - GET /api/v1/holidays/{region}                     │   │
│  │  - POST /api/v1/holidays/{id}/approve                │   │
│  │  - POST /api/v1/holidays/{id}/reject                 │   │
│  │  - POST /api/v1/holidays/sync-global                 │   │
│  └──────────────────────────────────────────────────────┘   │
└──────────────────────┬──────────────────────────────────────┘
                       │
     ┌─────────────────┼─────────────────┐
     ▼                 ▼                  ▼
 ┌────────┐      ┌───────────┐      ┌──────────┐
 │   DB   │      │ Temporal  │      │  OpenAI  │
 │        │      │           │      │   API    │
 │Holiday │      │Workflows  │      │          │
 │Schema  │      │Activities │      │gpt-4o-   │
 │        │      │           │      │mini      │
 └────────┘      └───────────┘      └──────────┘
```

**Data Flow**:
1. Admin triggers: `POST /holidays/generate`
2. API calls Temporal Workflow: `GenerateHolidayWorkflow`
3. Workflow calls Activity: `GenerateHolidayAI` (calls OpenAI)
4. Activity returns suggestions
5. Workflow stores in `pending_holiday_suggestions` table
6. Frontend retrieves and displays to admin
7. Admin approves → Workflow calls `ApproveHolidayActivity`
8. Activity inserts into `holidays` table
9. WebSocket notification to all clients

---

## Phase 4: Task Breakdown

### Week 1: Database Schema & Core Data Model

#### Task 4.1: Holiday Schema Design (Backend, 1 day)

**Deliverable**: `docs/schema_phase4_holidays.sql`

```sql
-- Phase 4 Schema: Holidays & AI Suggestions

-- Core holidays table
CREATE TABLE holidays (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  region VARCHAR(50) NOT NULL,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  date_start DATE NOT NULL,
  date_end DATE NOT NULL,
  is_recurring BOOLEAN DEFAULT FALSE,
  recurring_pattern VARCHAR(50), -- "annual", "monthly", "custom"
  
  -- Audit
  created_by UUID NOT NULL REFERENCES users(id),
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  
  -- Indexes for query performance
  UNIQUE(tenant_id, region, date_start, name),
  FOREIGN KEY(tenant_id, region) REFERENCES tenant_regions(tenant_id, region)
);

-- AI-generated suggestions (pending admin approval)
CREATE TABLE pending_holiday_suggestions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  region VARCHAR(50) NOT NULL,
  workflow_id VARCHAR(255) NOT NULL, -- Temporal workflow ID
  
  -- Suggestions (JSON array of proposed holidays)
  suggestions JSONB NOT NULL, -- [{name, date_start, date_end, confidence, reason}]
  
  -- Status
  status VARCHAR(50) DEFAULT 'pending', -- pending, approved, rejected, expired
  rejection_reason TEXT,
  
  -- Temporal tracking
  created_at TIMESTAMP DEFAULT NOW(),
  expires_at TIMESTAMP DEFAULT NOW() + INTERVAL '24 hours',
  approved_by UUID REFERENCES users(id),
  approved_at TIMESTAMP,
  
  INDEX idx_tenant_status (tenant_id, status),
  INDEX idx_expires (expires_at)
);

-- Holiday conflicts detected by AI
CREATE TABLE holiday_conflicts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  holiday_id UUID NOT NULL REFERENCES holidays(id),
  conflict_type VARCHAR(50), -- "overlap", "capacity", "resource", "external"
  severity VARCHAR(50), -- "low", "medium", "high", "critical"
  description TEXT,
  suggested_resolution TEXT,
  created_at TIMESTAMP DEFAULT NOW()
);

-- AI interaction logs (for monitoring & debugging)
CREATE TABLE ai_interaction_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  operation_type VARCHAR(50), -- "generate_holidays", "detect_conflicts", "suggest_schedule"
  workflow_id VARCHAR(255),
  input_params JSONB,
  ai_response JSONB,
  tokens_used INT, -- OpenAI token count
  cost_cents DECIMAL(10, 2), -- Estimated cost (for billing)
  status VARCHAR(50), -- "success", "error", "timeout"
  error_message TEXT,
  execution_time_ms INT,
  created_at TIMESTAMP DEFAULT NOW(),
  
  INDEX idx_tenant_operation (tenant_id, operation_type),
  INDEX idx_workflow_id (workflow_id)
);
```

**Deployment**: `./deploy_phase4_schema.sh`

#### Task 4.2: Migration Script (Backend, 1 day)

**Deliverable**: `scripts/deploy_phase4_schema.sh`

```bash
#!/bin/bash
# Deploy Phase 4 holiday schema to PostgreSQL

# Usage: ./deploy_phase4_schema.sh <db_host> <db_port> <db_user> <db_name>

# Includes:
# - Create all 4 tables
# - Add indexes
# - Add constraints
# - Set up audit triggers
# - Add PostGIS for holiday geocoding (optional, for future)
# - Create views for common queries
```

### Week 1-2: OpenAI Integration & Temporal Activities

#### Task 4.3: OpenAI Client Module (Backend, 2 days)

**Deliverable**: `internal/ai/openai_client.go` (300 lines)

```go
// OpenAI integration for holiday generation
package ai

type HolidayAIClient struct {
    apiKey     string
    model      string // "gpt-4o-mini"
    maxTokens  int
    logger     *slog.Logger
}

// Main function: generate holidays for region
func (c *HolidayAIClient) GenerateHolidaysForRegion(ctx context.Context, 
    region, industry, language string, excludeHistoric bool) ([]*Holiday, error) {
    
    // Prompt engineering: Create system + user prompt
    // Calls OpenAI API
    // Parses JSON response
    // Validates holidays
    // Returns structured data
}

// Detect conflicts between holidays and existing calendar
func (c *HolidayAIClient) DetectHolidayConflicts(ctx context.Context,
    holidays []*Holiday, existingSchedule *ScheduleData) ([]*HolidayConflict, error) {
    // Analyze overlaps with existing bookings
    // Provide conflict recommendations
}

// Cost tracking (for billing Phase 5)
func (c *HolidayAIClient) GetEstimatedCost() float64 {
    // Calculate based on tokens used across all operations
}
```

**Features**:
- Retry logic (exponential backoff for rate limits)
- Token counting for cost estimation
- Response caching (1 week)
- Error handling (timeout, malformed response, insufficient quota)

#### Task 4.4: Temporal Activities (Backend, 2 days)

**Deliverable**: `internal/temporal/holiday_activities.go` (250 lines)

```go
// Temporal activities for holiday operations
package temporal

// Activity 1: Generate holidays via AI
func GenerateHolidayAI(ctx context.Context, req *GenerateHolidayRequest) (*GenerateHolidayResponse, error) {
    // Call OpenAI client
    // Return 5-10 suggested holidays
    // Duration: 3-5 seconds
}

// Activity 2: Validate holiday against existing schedule
func ValidateHolidayCapacity(ctx context.Context, holiday *Holiday) (*ValidationResult, error) {
    // Query database: existing bookings during holiday period
    // Calculate impact on availability
    // Return: "safe", "warning", "critical"
}

// Activity 3: Notify admin of suggestions
func NotifyAdminOfHolidaySuggestions(ctx context.Context, suggestions []*Holiday) error {
    // Send email to tenant admin
    // Create WebSocket notification
    // Duration: 1 second
}

// Activity 4: Persist holiday to database
func PersistHolidayToDB(ctx context.Context, holiday *Holiday) error {
    // Insert into holidays table
    // Update availability indexes
    // Clear cache
}

// Activity 5: Sync holidays across regions
func SyncHolidaysAcrossRegions(ctx context.Context, baseHoliday *Holiday) error {
    // Take single region holiday
    // Apply to all other regions where applicable
    // Update all regional caches
}
```

**Reliability**:
- All activities idempotent (can be retried without side effects)
- Timeouts: 10s for AI, 5s for DB operations
- Retry policy: Exponential backoff, max 3 retries

### Week 2: Temporal Workflows & Orchestration

#### Task 4.5: Holiday Generation Workflow (Backend, 1 day)

**Deliverable**: `internal/temporal/holiday_workflows.go` (200 lines)

```go
// Temporal workflow for complete holiday generation process
package temporal

func HolidayGenerationWorkflow(ctx workflow.Context, req *GenerateHolidayRequest) (*GenerateHolidayResponse, error) {
    // Orchestrate 5 activities:
    // 1. Call AI to generate holiday suggestions
    // 2. Validate against existing capacity
    // 3. Detect conflicts
    // 4. Check multi-region implications
    // 5. Notify admin for approval
    
    // Total workflow duration: ~10 seconds
    
    // Return: Suggestions + conflict info for admin UI
}

func HolidayApprovalWorkflow(ctx workflow.Context, approvalReq *ApproveHolidayRequest) (*ApprovalResponse, error) {
    // Triggered when admin clicks "Approve"
    // 1. Persist holiday to DB
    // 2. Sync to other regions if needed
    // 3. Clear availability cache
    // 4. Notify affected users (if configured)
    // 5. Log audit trail
}
```

**Workflow Characteristics**:
- Long-running: Up to 5 minutes max (while waiting for admin approval)
- Can be cancelled by admin (cleanup activities)
- Stores state in Temporal (survives server restart)

### Week 2-3: Frontend UI (React)

#### Task 4.6: Holiday Approval Component (Frontend, 3 days)

**Deliverable**: `admin-ui/src/components/HolidayApprovalPanel.tsx`

```typescript
// React component for holiday approval workflow

export const HolidayApprovalPanel: React.FC<Props> = ({ tenantId, region }) => {
  const [suggestions, setSuggestions] = useState<Holiday[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  // UI Elements:
  // - "Generate Holidays" button
  // - Loading spinner (while AI generates)
  // - Card layout showing:
  //   * Holiday name + dates
  //   * Confidence score
  //   * Conflict warnings (if any)
  //   * Approve / Reject / Regenerate buttons
  // - Calendar view showing holidays
  // - History of approved holidays
  
  return (
    <div className="holiday-approval-panel">
      {/* Implementation */}
    </div>
  );
};
```

**Features**:
- Real-time updates via WebSocket
- Bulk approve/reject
- Integration with calendar view
- Conflict visualization
- Admin-only access (role-based)

#### Task 4.7: Calendar View Enhancement (Frontend, 2 days)

**Deliverable**: Updates to existing calendar component

```typescript
// Enhancements:
// - Overlay holidays on calendar grid
// - Show holiday info on hover
// - Color-code by holiday type (national, regional, company)
// - Click to view details/conflicts
// - Drag to reschedule (future feature)
```

### Week 3-4: Integration & Testing

#### Task 4.8: Integration Testing (Both, 2 days)

**Test Scenarios**:
1. Happy path: Generate → AI returns holidays → Admin approves → DB updated
2. Conflict path: AI detects overlap → Shows warning → Admin resolves
3. Multi-region: Holiday approved in us-east-1 → Synced to eu-west-1
4. Timeout: AI takes > 10s → Graceful degradation + user notification
5. Failure: OpenAI API quota exceeded → Error handling + retry

**Test Data**:
- Tenant with 3 regions
- Existing bookings for conflict detection
- Multiple holiday types (national, regional, company)

#### Task 4.9: Monitoring & Observability (Both, 1 day)

**Metrics**:
- `ai_holiday_generation_duration_ms` (by region)
- `ai_suggestions_per_generation` (average)
- `admin_approval_rate` (% suggested → approved)
- `openai_api_tokens_used`
- `openai_api_cost_usd`
- `holiday_conflict_detection_accuracy`

**Alerts**:
- OpenAI API error rate > 5%
- Average response time > 10s
- Pending suggestions expiring soon

**Logs**:
- Every AI call logged (for debugging + cost tracking)
- Admin approvals tracked (audit trail)
- Errors captured with full context

### Week 4: Documentation & Deployment

#### Task 4.10: Documentation (Backend, 1 day)

**Deliverables**:
1. `PHASE4_AI_IMPLEMENTATION.md` (600+ lines)
   - Architecture diagram
   - OpenAI configuration (API key, model settings)
   - Temporal workflow diagrams
   - Database schema
   - API endpoints
   - Sample requests/responses
   - Error handling guide
   - Cost estimation

2. `PHASE4_DEPLOYMENT_GUIDE.md` (450+ lines)
   - Pre-deployment checklist
   - Database migration script
   - Environment variables
   - OpenAI account setup
   - Staging validation steps
   - Production rollout plan
   - Rollback procedures
   - Success criteria

#### Task 4.11: Code Review & Polish (Both, 2 days)

- Code quality check (linting, typing, coverage)
- Security review (API key handling, SQL injection prevention)
- Performance review (response times, token usage)
- Documentation review (completeness, accuracy)

---

## Phase 4: Deliverables Summary

| Week | Task | Files | LOC | Status |
|------|------|-------|-----|--------|
| 1 | DB Schema | 1 SQL + 1 Script | 400 | Design ready |
| 1-2 | OpenAI Integration | 1 Go file | 300 | Ready to implement |
| 1-2 | Temporal Activities | 1 Go file | 250 | Ready to implement |
| 2 | Workflows | 1 Go file | 200 | Ready to implement |
| 2-3 | React UI | 2 Components | 500 | Design ready |
| 3-4 | Integration Testing | 1 Test file | 800 | Ready to implement |
| 4 | Monitoring | Config files | 200 | Ready to implement |
| 4 | Documentation | 2 Docs | 1,000+ | To be written |
| **Total** | **11 tasks** | **~30 files** | **~3,650 lines** | **3-4 weeks** |

---

## Implementation Sequence (Recommendation)

**Week 1 (Mon-Fri)**:
- Mon-Tue: Database schema + migration script
- Wed-Fri: OpenAI client module (with unit tests)

**Week 2 (Mon-Fri)**:
- Mon-Tue: Temporal activities (5 activities = 250 lines)
- Wed: Workflows (orchestration)
- Thu-Fri: Integration testing (activities + workflows)

**Week 3 (Mon-Fri)**:
- Mon-Wed: React UI (Holiday Approval Panel)
- Thu-Fri: Calendar enhancement + integration

**Week 4 (Mon-Thu)**:
- Mon: Monitoring & observability setup
- Tue: Full end-to-end testing
- Wed: Documentation
- Thu: Code review + polish + staging deployment

**Friday (Week 4)**:
- Production deployment OR
- Phase 5 starts (cost billing integration)

---

## Parallel Work During Phase 3 Staging (Weeks 1-2)

**While Phase 3 validates in staging (Days 0-5 of this timeline)**:
- Start Phase 4.1 (Database schema) - no dependencies
- Start Phase 4.3 (OpenAI client) - standalone module
- Set up OpenAI account, get API key

**Critical Path**:
1. Schema + Migration (Day 1-2)
2. OpenAI Client (Day 3-4)
3. Temporal Activities (Day 5-6)
4. Workflows (Day 7)
5. React UI (Day 8-10)
6. Testing (Day 11-12)
7. Docs + Deployment (Day 13-15)

And with 48h staging validation for Phase 3, Phase 4 can be complete or near-complete by the time Phase 3 goes to production.

---

## Cost Estimation

**OpenAI API Usage (Monthly Estimate)**:
- Holiday generation: 50 calls/month × 500 tokens/call = 25K tokens
- Conflict detection: 30 calls/month × 300 tokens/call = 9K tokens
- Cost: ~$0.10/month (gpt-4o-mini is cheap)
- Budget: $5/month with headroom

**Development Cost**:
- Estimated: 80-100 engineering hours
- Team: 2 engineers (backend + frontend)
- Duration: 3-4 weeks

**Risk Mitigations**:
- Implement token rate limiting (max 1000 tokens/call)
- Cache AI responses (1 week TTL)
- Fallback to basic rules if API unavailable
- Cost monitoring dashboard (Phase 5)

---

## Success Criteria

| Criterion | Metric | Target |
|-----------|--------|--------|
| **Functionality** | Holiday generation accuracy | 95%+ (validated by domain experts) |
| **Performance** | AI response time | < 10 seconds (p95) |
| **Reliability** | Workflow success rate | > 99% |
| **UX** | Admin approval turnaround | < 1 minute |
| **Cost** | OpenAI API cost | < $5/month |
| **Operations** | Uptime | 99.9% |
| **Support** | Response to issues | < 1 hour |

---

## Next Phases (Phase 5+)

**(Phase 5)** Billing Integration
- Track OpenAI tokens per tenant
- Bill per API call or monthly estimate
- Dashboard with cost breakdown

**(Phase 6)** Advanced AI
- Holiday recommendations based on historical patterns
- Predictive capacity planning (ML model)
- Anomaly detection in scheduling

**(Phase 7)** Multi-language Support
- Translate holiday names/descriptions
- Locale-aware calendar formatting

**(Phase 8)** Mobile App
- Native apps (iOS/Android)
- Native notifications
- Offline mode for holidays

---

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| OpenAI API failure | 🟡 Medium | 🔴 High | Fallback rules, circuit breaker, caching |
| Holiday data quality | 🟡 Medium | 🟡 Medium | Validation + admin review before approval |
| Performance degradation | 🟢 Low | 🟡 Medium | Rate limiting, background processing |
| Cost overrun | 🟢 Low | 🟡 Medium | Token limits, cost monitoring |
| User adoption | 🟡 Medium | 🟡 Medium | Training, clear UX, gradual rollout |

---

## Ready to Start?

**Prerequisites Met**:
✅ Phase 3 staging validation in progress (enables parallel work)
✅ Database migrations infrastructure in place
✅ Temporal workflows proven
✅ React build system ready
✅ OpenAI API available

**First Action**:
1. Open OpenAI console
2. Create API key (if not already done)
3. Set `OPENAI_API_KEY` in staging .env
4. Create `docs/schema_phase4_holidays.sql`
5. See you next week!

---

**Status**: 🟢 Ready to begin  
**Estimated Completion**: 3-4 weeks  
**Parallel with Phase 3**: Yes (non-blocking)  
**Next Review Date**: Week 2 (after first 2 tasks complete)
