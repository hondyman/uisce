# Client Onboarding Process - Complete Implementation Guide

## Overview

This documentation describes the complete implementation of a wealth management client onboarding process. The system is built using:

- **Database**: PostgreSQL with multi-tenant support
- **Backend**: Go with Chi router and sqlx
- **Workflow Orchestration**: Temporal (long-running processes with timeout escalation)
- **Frontend**: React (TypeScript) with tenant-scoped requests
- **Compliance**: ABAC-based access control, KYC/AML screening, audit trails

## Architecture

### System Components

```
┌─────────────────────────────────────────────────────────────────┐
│                       Client Onboarding                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────┐        ┌──────────────┐      ┌────────────┐  │
│  │   Database   │◄──────►│  Go Backend  │◄────►│  Temporal  │  │
│  │  (PostgreSQL)│        │   (REST API) │      │ (Workflows)│  │
│  └──────────────┘        └──────────────┘      └────────────┘  │
│                                 ▲                                │
│                                 │                                │
│                          ┌──────▼──────┐                        │
│                          │   Tenant    │                        │
│                          │   Context   │                        │
│                          └─────────────┘                        │
│                                                                 │
│  ┌────────────────────────────────────────────────────────┐   │
│  │             5-Step Onboarding Process                  │   │
│  ├────────────────────────────────────────────────────────┤   │
│  │ ✓ Step 1: Validate Client Data (KYC/AML)             │   │
│  │ ✓ Step 2: Route for Advisor Review/Approval          │   │
│  │ ✓ Step 3: Generate & Send Agreements (e-signature)   │   │
│  │ ✓ Step 4: Create Accounts & Portfolios               │   │
│  │ ✓ Step 5: Notify Client Upon Completion              │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Database Schema

### Core Tables

#### 1. **clients** - Main client entity
```sql
- id (UUID, PK)
- tenant_id, datasource_id (Multi-tenancy)
- Personal info (name, email, phone, DOB)
- KYC/AML (identification_number, country, etc.)
- Risk profile (risk_profile, net_worth, annual_income)
- Onboarding tracking (onboarding_status, onboarding_stage)
- Workflow reference (temporal_workflow_id)
- Compliance (kyc_status, aml_status, kyc_aml_result JSONB)
- Advisor assignment (assigned_advisor_id)
- Audit trail (created_by, created_at, updated_by, updated_at)
```

#### 2. **client_documents** - Legal and verification documents
```sql
- id (UUID, PK)
- Document info (document_type, document_name, document_path)
- Verification (status, verification_status, verified_by, verified_at)
- Expiration tracking (issue_date, expiry_date, is_expired)
- E-signature (e_signature_request_id, e_signature_status, signed_at)
```

#### 3. **client_contacts** - Related people (advisors, emergency contacts, etc.)
```sql
- id (UUID, PK)
- Contact info (contact_type, name, email, phone)
- Permissions (can_access_accounts, can_make_trades, etc.)
- Advisor details (is_primary_advisor, employee_id, department)
```

#### 4. **client_accounts** - Investment accounts
```sql
- id (UUID, PK)
- Account info (account_number, account_type: brokerage/ira/etc.)
- Status (pending_funding, active, suspended, closed)
- Balance tracking (initial_balance, current_balance, currency)
- Custodian (custodian_name, custodian_account_id)
- Features (allows_margin, allows_options, allows_cryptocurrency)
```

#### 5. **client_portfolios** - Asset allocations
```sql
- id (UUID, PK)
- Portfolio info (portfolio_name, portfolio_type, status)
- Allocation (allocation_json JSONB: {equities, fixed_income, alternatives, cash})
- Rebalancing (rebalance_frequency, last_rebalance_date, next_rebalance_date)
- Performance (total_market_value, total_gain_loss, ytd_return)
```

#### 6. **portfolio_holdings** - Individual securities
```sql
- id (UUID, PK)
- Security (security_id, security_name, security_type)
- Position (quantity, unit_price, market_value)
- Allocation (target_allocation, actual_allocation)
- Performance (cost_basis, gain_loss, gain_loss_percentage)
```

#### 7. **onboarding_workflows** - Workflow state tracking
```sql
- id (UUID, PK)
- workflow_id (VARCHAR, Temporal workflow ID)
- Step statuses (step_1_validation_status, step_2_routing_status, etc.)
- Completion times (step_1_completed_at, step_2_completed_at, etc.)
- Approval/Rejection (approved_by, approved_at, rejected_by, rejection_reason)
- Timeout escalation (timeout_escalation_workflow_id, escalation_status, escalation_action)
```

#### 8. **onboarding_events** - Audit trail
```sql
- id (UUID, PK)
- Event (event_type, event_data JSONB, event_timestamp)
- Actor (triggered_by, actor_type, actor_role)
- Step reference (step_number)
```

#### 9. **kyc_aml_results** - Screening details
```sql
- id (UUID, PK)
- Screening (screening_type: kyc/aml, screening_provider, screening_date)
- Results (status: pass/fail/review_required, risk_score, risk_level)
- Findings (findings JSONB, matches pq.StringArray)
- Review (requires_review, reviewed_by, reviewed_at, review_notes)
- Escalation (escalation_level, escalated_to, escalation_reason)
```

#### 10. **onboarding_notes** - Internal documentation
```sql
- id (UUID, PK)
- Note (note_type, content)
- Visibility (is_internal, visible_to_client, required_role)
- Reference (related_step, related_document_id)
```

### Views

- **active_onboarding_clients** - All clients currently in onboarding
- **client_onboarding_summary** - Summary with account count and balances
- **pending_advisor_approvals** - Clients waiting for advisor review

## Backend API Endpoints

### Client Management

```http
POST   /api/clients
       Create new client record
       Request: ClientRequest
       Response: Client

GET    /api/clients
       List all clients in onboarding
       Query params: limit=20, offset=0
       Response: {clients: Client[], total: int}

GET    /api/clients/{clientID}
       Retrieve single client
       Response: Client
```

### 5-Step Onboarding Process

#### Step 1: Validate Client Data
```http
POST   /api/onboarding/step1/validate
       Validate KYC/AML and client information
       Request: ValidateClientDataRequest {
         client_id: string,
         verify_kyc: boolean,
         perform_aml_screening: boolean,
         aml_provider: string,
         requires_due_diligence: boolean,
         due_diligence_reason?: string
       }
       Response: {
         client_id: string,
         workflow_id: string,
         validation_passed: boolean,
         errors: object,
         kyc_status: string,
         aml_status: string,
         next_step: "route_for_advisor_review",
         requires_escalation: boolean
       }
```

#### Step 2: Route for Advisor Review
```http
POST   /api/onboarding/step2/route
       Assign client to advisor for review
       Request: RouteForReviewRequest {
         client_id: string,
         advisor_id: string,
         priority: "low"|"medium"|"high"|"urgent",
         review_notes?: string
       }
       Response: {
         client_id: string,
         advisor_id: string,
         status: "pending_review",
         next_step: "send_agreements",
         message: string
       }
```

#### Step 3: Generate & Send Agreements
```http
POST   /api/onboarding/step3/agreements
       Generate and send legal agreements for e-signature
       Request: GenerateAgreementsRequest {
         client_id: string,
         agreement_types: ["client_agreement", "disclosure", ...],
         e_signature_method: "docusign"|"hellosign",
         delivery_method: "email"|"portal"
       }
       Response: {
         client_id: string,
         agreements_sent: number,
         e_signature_method: string,
         status: "pending_agreements",
         next_step: "create_accounts"
       }
```

#### Step 4: Create Accounts & Portfolios
```http
POST   /api/onboarding/step4/accounts
       Create investment accounts and initial portfolios
       Request: CreateAccountsRequest {
         client_id: string,
         account_types: ["brokerage", "ira", ...],
         initial_funding?: number,
         custodian: string,
         banking_api_ref?: string
       }
       Response: {
         client_id: string,
         accounts_created: [{
           account_id: string,
           account_number: string,
           type: string
         }, ...],
         status: "pending_notification",
         next_step: "notify_client"
       }
```

#### Step 5: Notify Client
```http
POST   /api/onboarding/step5/notify
       Send completion notification
       Request: NotifyClientRequest {
         client_id: string,
         notification_type: "email"|"sms"|"portal",
         portal_access_url?: string
       }
       Response: {
         client_id: string,
         client_name: string,
         email: string,
         status: "active",
         notification_sent: boolean,
         portal_access_url?: string
       }
```

### Status & Management

```http
GET    /api/onboarding/status/{clientID}
       Get current onboarding progress
       Response: OnboardingStatusResponse {
         client_id: string,
         overall_status: string,
         current_step: int,
         step_statuses: {step_1_validation, step_2_routing, ...},
         completion_percent: int,
         workflow_id: string,
         estimated_completion?: timestamp,
         next_action: string,
         blocking_issues?: string[]
       }

POST   /api/onboarding/approve
       Approve onboarding as advisor
       Request: ApproveOnboardingRequest {
         workflow_id: string,
         advisor_id: string,
         notes?: string
       }
       Response: {workflow_id, approved_by, status, message}
```

## Temporal Workflow Implementation

### ClientOnboardingWorkflow

**Purpose**: Orchestrates the complete 5-step onboarding process with built-in timeout handling and escalation

**Flow**:

```
START
  │
  ├─► STEP 1: Validate Client (KYC/AML)
  │     Activity: ValidateClientDataActivity
  │     Timeout: 5 minutes
  │     Escalation: If validation fails, escalate to compliance director
  │     Signal: None (activities only)
  │
  ├─► STEP 2: Route for Advisor Review
  │     Activity: RouteForAdvisorReviewActivity
  │     Timeout: 5 minutes
  │     Signal: advisor_approval (48-hour timeout)
  │       ├─ Signal="approved" → Continue
  │       ├─ Timeout → Start escalation workflow
  │       └─ Signal≠"approved" → Reject & end
  │
  ├─► STEP 3: Generate & Send Agreements
  │     Activity: GenerateAndSendAgreementsActivity
  │     Timeout: 5 minutes
  │     Signal: agreements_signed (7-day timeout)
  │       ├─ Signature received → Continue
  │       ├─ Timeout → SendAgreementReminderActivity (3-day retry)
  │       └─ Continue regardless
  │
  ├─► STEP 4: Create Accounts & Portfolios
  │     Activity: CreateAccountsAndPortfoliosActivity
  │     Timeout: 5 minutes
  │     Signal: None (activities only)
  │
  ├─► STEP 5: Notify Client
  │     Activity: NotifyClientOnCompletionActivity
  │     Timeout: 5 minutes
  │     Signal: None (best-effort)
  │
  └─► END (status = "completed")
```

### Key Features

1. **Temporal Reliability**
   - Automatic retries with exponential backoff
   - Persistent state across failures
   - Guaranteed workflow completion

2. **Timeout Handling**
   - Step timeouts trigger escalation workflow
   - Manager/director can approve/reject
   - Auto-approve option after director timeout

3. **Signal-Based Approval**
   - Async signals from humans (advisors, compliance)
   - Non-blocking: other steps proceed independently
   - Audit trail of all signals

4. **Escalation Workflow**
   - Nested workflow for escalation decisions
   - 48-hour timeout with auto-approval
   - Logs all escalation events

## Implementation Files

### Database
- `migrations/client_onboarding_schema.sql` - Complete schema with 10 tables, views, indices

### Backend (Go)
- `backend/internal/api/client_onboarding_types.go` - All type definitions
- `backend/internal/api/client_onboarding_service.go` - Database service layer (~600 LOC)
- `backend/internal/api/client_onboarding_handlers.go` - REST API handlers (~700 LOC)

### Temporal
- `temporal/workflows/client_onboarding_workflow.go` - Main workflow + escalation workflow (~500 LOC)
- `temporal/activities/client_onboarding_activities.go` - Activity implementations (~400 LOC)

### Documentation
- This README

## Integration Guide

### 1. Database Setup

```bash
# Apply migrations
psql -h localhost -U postgres -d alpha -f migrations/client_onboarding_schema.sql
```

### 2. Backend Integration

Register routes in your main API setup:

```go
import "github.com/semlayer/backend/internal/api"

func setupRoutes(router chi.Router, db *sqlx.DB) {
    // ... existing routes ...
    api.RegisterClientOnboardingRoutes(router, db)
}
```

### 3. Temporal Worker Setup

Register workflows and activities:

```go
import (
    "github.com/semlayer/temporal/workflows"
    "github.com/semlayer/temporal/activities"
)

func startTemporalWorker() {
    // Initialize worker
    w := worker.New(client, taskQueue, worker.Options{})
    
    // Register workflows
    w.RegisterWorkflow(workflows.ClientOnboardingWorkflow)
    w.RegisterWorkflow(workflows.ClientOnboardingEscalationWorkflow)
    
    // Register activities
    w.RegisterActivity(activities.ValidateClientDataActivity)
    w.RegisterActivity(activities.RouteForAdvisorReviewActivity)
    w.RegisterActivity(activities.GenerateAndSendAgreementsActivity)
    w.RegisterActivity(activities.CreateAccountsAndPortfoliosActivity)
    w.RegisterActivity(activities.NotifyClientOnCompletionActivity)
    // ... register other activities ...
    
    err := w.Run(worker.InterruptCh())
    if err != nil {
        log.Fatalf("Unable to start worker: %v", err)
    }
}
```

### 4. Tenant Scoping

All endpoints require tenant context (automatically enforced by frontend shim):

```http
Headers:
  X-Tenant-ID: {tenant_uuid}
  X-Tenant-Datasource-ID: {datasource_uuid}
  X-User-ID: {user_uuid}

Query Params:
  ?tenant_id={tenant_uuid}&datasource_id={datasource_uuid}
```

## Usage Example

### Complete Onboarding Flow

```bash
# 1. Create Client
curl -X POST http://localhost:8080/api/clients \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  -H "X-Tenant-Datasource-ID: 11111111-1111-1111-1111-111111111111" \
  -H "X-User-ID: 22222222-2222-2222-2222-222222222222" \
  -H "Content-Type: application/json" \
  -d '{
    "first_name": "John",
    "last_name": "Doe",
    "email": "john@example.com",
    "risk_profile": "moderate",
    "net_worth": 500000
  }'

# Response:
# {
#   "id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
#   "onboarding_status": "pending_validation",
#   "created_at": "2025-10-28T..."
# }

# 2. Start Validation (Step 1)
curl -X POST http://localhost:8080/api/onboarding/step1/validate \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  -H "X-Tenant-Datasource-ID: 11111111-1111-1111-1111-111111111111" \
  -H "X-User-ID: 22222222-2222-2222-2222-222222222222" \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "verify_kyc": true,
    "perform_aml_screening": true,
    "aml_provider": "lexis_nexis"
  }'

# Response:
# {
#   "validation_passed": true,
#   "kyc_status": "approved",
#   "aml_status": "approved",
#   "workflow_id": "client-onboard-xxxxxxxx-..."
# }

# 3. Route to Advisor (Step 2)
curl -X POST http://localhost:8080/api/onboarding/step2/route \
  -d '{
    "client_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "advisor_id": "advisor-12345",
    "priority": "normal"
  }'

# At this point, the Temporal workflow waits for advisor_approval signal (48 hours)
# Advisor reviews client in their dashboard and signals approval

# 4. Generate Agreements (Step 3)
curl -X POST http://localhost:8080/api/onboarding/step3/agreements \
  -d '{
    "client_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "agreement_types": ["client_service_agreement", "disclosure_form"],
    "e_signature_method": "docusign"
  }'

# Client receives email with DocuSign link and signs agreements
# Workflow polls for agreements_signed signal

# 5. Create Accounts (Step 4)
curl -X POST http://localhost:8080/api/onboarding/step4/accounts \
  -d '{
    "client_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "account_types": ["brokerage", "ira"],
    "initial_funding": 50000,
    "custodian": "primary_custodian"
  }'

# Accounts created and portfolios allocated

# 6. Notify Client (Step 5)
curl -X POST http://localhost:8080/api/onboarding/step5/notify \
  -d '{
    "client_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "notification_type": "email",
    "portal_access_url": "https://portal.example.com/clients/..."
  }'

# 7. Check Status
curl -X GET http://localhost:8080/api/onboarding/status/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx

# Response:
# {
#   "overall_status": "completed",
#   "current_step": 5,
#   "completion_percent": 100,
#   "step_statuses": {
#     "step_1_validation": "completed",
#     "step_2_routing": "completed",
#     "step_3_agreements": "completed",
#     "step_4_accounts": "completed",
#     "step_5_notification": "completed"
#   }
# }
```

## ABAC Integration

### Step 1 Validation - Compliance Officer Only
```sql
-- Rule: Only compliance officers can view validation results
WHERE role = 'compliance_officer' AND is_active = true
```

### Step 2 Routing - Advisor Assignment
```sql
-- Advisor can see assigned clients
WHERE assigned_advisor_id = :current_user_id

-- Compliance can reassign
WHERE role = 'compliance_officer'
```

### Step 3 Agreements - Location-Based Access
```sql
-- Agreements only accessible from approved IPs/locations
WHERE created_location IN (:approved_locations)
```

### Step 4 Account Creation - Resource Management
```sql
-- Only authorized personnel can create accounts
WHERE role IN ('senior_advisor', 'operations_manager', 'system_admin')
```

### Step 5 Notifications - Audit Logging
```sql
-- All notifications logged for compliance
INSERT INTO onboarding_events (event_type, actor_type, actor_role, ...)
VALUES ('notification_sent', 'system', 'notification_service', ...)
```

## Timeout & Escalation Behavior

### Timeout Scenarios

| Step | Activity | Timeout | Escalation |
|------|----------|---------|-----------|
| 1 | Validation | 5 min | Auto-fail (retry 3x) |
| 2 | Route to Advisor | 48 hours (signal) | Escalate to Manager (24h) |
| 3 | Agreements | 7 days (signal) | Send reminder, retry 3 days |
| 4 | Create Accounts | 5 min | Auto-fail (retry 3x) |
| 5 | Notify Client | 5 min | Log & continue (best-effort) |

### Escalation Workflow

When a timeout occurs:
1. Manager is notified with context
2. Manager has 24 hours to approve/reject
3. If manager timeout, escalate to director
4. Director has 48 hours
5. Auto-approve if director timeout

## Validation Rules

Pre-built validation rules for KYC/AML integration:

```javascript
// Rule 1: Identification Required
condition: "identification_number IS NULL OR identification_type IS NULL"
severity: "error"
action: "block_until_provided"

// Rule 2: High Net Worth Due Diligence
condition: "net_worth > 5000000"
severity: "warning"
action: "require_additional_documentation"

// Rule 3: PEP Check
condition: "aml_screening_result.pep_match = true"
severity: "error"
action: "escalate_to_compliance"

// Rule 4: Sanctions List
condition: "aml_screening_result.sanctions_match = true"
severity: "critical"
action: "reject_immediately"
```

## Performance Considerations

- **Database**: Indices on tenant_id, datasource_id, onboarding_status, workflow_id
- **Temporal**: Task queue persistence, workflow history retention
- **Audit Trail**: Partitioned by created_at for performance
- **Bulk Operations**: Batch inserts for account/portfolio creation

## Security

1. **Tenant Isolation**: All queries scoped to tenant_id + datasource_id
2. **Audit Trail**: Every action logged with user_id, timestamp, IP
3. **E-Signature**: Signed agreements tied to IP and timestamp
4. **KYC/AML**: External API responses stored JSONB for compliance
5. **Access Control**: ABAC policies enforce role-based access

## Next Steps

1. **Frontend UI**: Create React components for workflow visualization and advisor dashboard
2. **Notifications**: Integrate with email/SMS service
3. **External APIs**: Connect to DocuSign, banking APIs, KYC/AML providers
4. **Analytics**: Dashboard for onboarding metrics (completion rates, time-to-complete, etc.)
5. **Mobile App**: Mobile-friendly onboarding flow for clients
6. **Testing**: E2E tests covering all 5 steps and timeout scenarios
