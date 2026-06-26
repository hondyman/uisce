# Enhanced AML Screening Implementation for Client Onboarding

## Executive Summary

This document outlines the comprehensive AML (Anti-Money Laundering) screening enhancements integrated into the Client Onboarding workflow. The implementation adds a dedicated **Step 2: AML Screening** that performs detailed risk assessment using external watchlists, sanctions lists, PEP databases, and internal rule-based checks. The system enforces ABAC-based access control to ensure only authorized compliance personnel can view and approve AML results.

---

## 1. Enhanced Onboarding Process Flow

The Client Onboarding process now includes a dedicated AML screening step:

```
Step 1: Validate Client Data (Initial KYC)
    ↓
Step 2: Perform AML Screening (NEW - Comprehensive Watchlist/PEP/Sanctions)
    ↓
Step 3: Route for Advisor Review/Approval
    ↓
Step 4: Generate Agreements for E-Signature
    ↓
Step 5: Create Accounts & Portfolios
    ↓
Step 6: Notify Client & Activate Portal
```

### Key Enhancement: Step 2 - AML Screening

**Purpose**: Detect potential money laundering activities by screening client data against:
- Global watchlists (OFAC, UN Security Council, etc.)
- Sanctions lists (OFAC, EU, UK, etc.)
- Politically Exposed Persons (PEP) databases
- Adverse media / negative news
- High-risk country assessments
- Source of funds verification

**Timing**: Immediately after Step 1 validation, before advisor review

**Integration**: Seamless integration with external providers (LexisNexis, WorldCheck, Dow Jones) via API connectors

---

## 2. AML Risk Scoring Algorithm

The risk scoring engine uses a weighted, point-based system aligned with FATF and FinCEN guidelines:

### Risk Score Calculation

```
Total Risk Score (0-100) = Sum of weighted factors:

1. Watchlist Match              → +50 points (if found on any watchlist)
2. Sanctions Match              → +40 points (OFAC/EU/UN designations)
3. PEP (Politically Exposed)    → +25 points (identified as PEP)
4. High Net Worth (>$5M)        → +20 points (volume/complexity risk)
5. Unknown Source of Funds      → +15-25 points (higher for HNW clients)
6. High-Risk Countries          → +30 points (FATF grey/black list)
7. Adverse Media                → +20 points (reputation/legal risk)

Risk Level Mapping:
- 0-20:    LOW         → Clear to proceed
- 20-40:   MEDIUM_LOW  → Standard review
- 40-60:   MEDIUM      → Advisor + compliance review
- 60-80:   HIGH        → Mandatory manual compliance review
- 80-100:  CRITICAL    → Escalation to compliance manager (24h deadline)
```

### Example Scoring Scenarios

**Scenario A: Low-Risk Client**
- Net Worth: $2M
- Country: US (low-risk)
- Source of Funds: Verified employment
- Watchlist: No match
- **Risk Score: 0** → Status: Clear ✓

**Scenario B: Medium-Risk Client**
- Net Worth: $10M (high net worth)
- Country: Emerging market
- Source of Funds: Business ownership
- Watchlist: No match
- PEP: No
- **Risk Score: 50** → Status: High, Manual Review Required ⚠

**Scenario C: Critical-Risk Client**
- Net Worth: $50M
- Country: Sanctioned jurisdiction
- Source of Funds: Unknown
- Watchlist: Match found
- Sanctions: Yes (OFAC SDN List)
- **Risk Score: 140 → capped at 100** → Status: Critical, Escalation Required 🚨

---

## 3. AML Screening Implementation

### 3.1 Backend Service Layer (`client_aml_screening.go`)

```go
// AMLScreeningService handles all AML screening operations
type AMLScreeningService struct {
    db *sqlx.DB
}

// Core Methods:
- CreateAMLScreening()              // Initialize screening
- ComputeAMLRiskScore()             // Calculate risk using algorithm
- GetAMLScreening()                 // Retrieve results
- UpdateAMLScreeningStatus()        // Approve/reject
- GetClientAMLScreeningHistory()    // Audit trail
```

#### Key Data Structures

```go
type AMLScreeningResult struct {
    ID                    string      // Unique screening ID
    ClientID              string      // Link to client
    ScreeningProvider     string      // lexis_nexis, worldcheck, dow_jones, internal
    ScreeningStatus       string      // pending → in_progress → completed → failed
    RiskScore             float64     // 0-100
    RiskLevel             string      // low, medium, high, critical
    OverallStatus         string      // clear, flagged, rejected
    
    // Detailed Findings
    WatchlistMatch        bool
    WatchlistMatches      []string    // List of matching lists
    SanctionsMatch        bool
    SanctionsDetails      *string
    PEPMatch              bool
    PEPLevel              *string     // low, medium, high
    
    // Risk Flags
    HighNetWorthFlag      bool        // > $5M
    UnknownFundsFlag      bool        // Source of funds unverified
    RiskyCountriesFlag    bool        // FATF grey/black list
    AdverseMediaFlag      bool        // Negative news found
    
    // Approval/Rejection
    ManualReviewRequired  bool
    ManualReviewReason    *string
    ApprovedBy            *string
    ApprovedAt            *time.Time
    RejectedBy            *string
    RejectedAt            *time.Time
    RejectionReason       *string
    ComplianceNotes       *string
}
```

### 3.2 REST API Endpoints (`client_aml_handlers.go`)

#### POST `/api/onboarding/{clientID}/step2-aml-screening`
**Initiates AML screening**

```
Headers:
  X-Tenant-ID: <tenant-id>
  X-Tenant-Datasource-ID: <datasource-id>
  X-User-ID: <user-id>
  X-User-Role: ComplianceOfficer  [ABAC enforcement]

Request Body:
{
  "client_id": "cli_12345",
  "screening_provider": "lexis_nexis",  // lexis_nexis | worldcheck | dow_jones | internal
  "perform_manual_review": false,
  "requires_due_diligence": false
}

Response (202 Accepted):
{
  "screening_id": "scr_cli_12345_1729084800",
  "status": "pending",
  "risk_score": 45.5,
  "risk_level": "medium",
  "message": "AML screening initiated - results will be available shortly"
}
```

**ABAC Control**: Only `ComplianceOfficer`, `ComplianceManager`, `ComplianceDirector`, or `Admin` roles

#### GET `/api/onboarding/{clientID}/aml-screening/{screeningID}`
**Retrieves screening results**

```
Response (200 OK):
{
  "id": "scr_cli_12345_1729084800",
  "client_id": "cli_12345",
  "screening_date": "2025-10-28T15:30:00Z",
  "screening_provider": "lexis_nexis",
  "screening_status": "completed",
  "risk_score": 45.5,
  "risk_level": "medium",
  "overall_status": "flagged",
  "watchlist_match": false,
  "sanctions_match": false,
  "pep_match": false,
  "pep_level": null,
  "high_net_worth_flag": true,
  "unknown_funds_flag": false,
  "risky_countries_flag": false,
  "manual_review_required": true,
  "manual_review_reason": "High net worth client requires enhanced due diligence",
  "approved_by": null,
  "approved_at": null,
  "compliance_notes": null
}
```

**ABAC Control**: `ComplianceOfficer`, `ComplianceManager`, `Advisor`, `Client` (own results only)

#### POST `/api/onboarding/aml-screening/{screeningID}/review`
**Approves or rejects AML screening**

```
Headers:
  X-User-Role: ComplianceManager  [ABAC: Requires higher privilege]

Request Body:
{
  "screening_id": "scr_cli_12345_1729084800",
  "approval_status": "approved",  // approved | rejected
  "compliance_notes": "Client verified via video call. Source of funds confirmed.",
  "rejection_reason": null
}

Response (200 OK):
{
  "screening_id": "scr_cli_12345_1729084800",
  "overall_status": "clear",
  "approval_status": "approved",
  "reviewed_at": "2025-10-28T16:00:00Z",
  "reviewed_by": "compliance_officer_001"
}
```

**ABAC Control**: Only `ComplianceManager` or `ComplianceDirector` can approve critical screenings
Temporal Policy: Approval must complete within 24h for critical findings, 48h for standard

#### GET `/api/onboarding/{clientID}/aml-screening/history`
**Retrieves screening audit trail**

```
Response (200 OK):
{
  "client_id": "cli_12345",
  "count": 3,
  "screenings": [
    {
      "id": "scr_cli_12345_1729084800",
      "screening_date": "2025-10-28T15:30:00Z",
      "risk_score": 45.5,
      "overall_status": "clear",
      "approved_at": "2025-10-28T16:00:00Z"
    },
    ...
  ]
}
```

---

## 4. Temporal Workflow Integration

### Updated Onboarding Workflow

```go
// ClientOnboardingWorkflow orchestrates the 6-step process
func ClientOnboardingWorkflow(ctx workflow.Context, input ClientOnboardingWorkflowInput) {
    
    // STEP 1: Validate Client Data
    validationResult := executeActivity(ValidateClientDataActivity)
    
    // STEP 2: AML SCREENING (NEW)
    amlInput := &AMLScreeningInput{
        ClientID: input.ClientID,
        ScreeningProvider: input.AMLProvider,
        NetWorth: client.NetWorth,
        CountryOfCitizenship: client.CountryOfCitizenship,
    }
    amlResult := executeActivity(PerformAMLScreeningActivity, amlInput)
    
    // Handle critical AML findings
    if amlResult.RiskLevel == "critical" {
        // Escalation workflow with 24h deadline
        escalationResult := executeChildWorkflow(
            AMLEscalationWorkflow,
            amlResult
        )
        // Wait for escalation decision (approval/rejection)
    }
    
    // STEP 3: Route for Advisor Review
    advisorResult := executeActivity(RouteForAdvisorReviewActivity)
    
    // ... Continue with steps 4-6
}
```

### AML Screening Activities

#### `PerformAMLScreeningActivity`
**Executes external AML screening API calls**

```go
type AMLScreeningInput struct {
    ClientID             string    // Client ID
    ScreeningProvider    string    // lexis_nexis | worldcheck | dow_jones | internal
    NetWorth             *float64  // For risk computation
    CountryOfCitizenship *string
    TaxResidencyCountry  *string
    SourceOfFunds        *string
    RequiresDueDiligence bool
}

type AMLScreeningOutput struct {
    ScreeningID          string
    RiskScore            float64   // 0-100
    RiskLevel            string    // low, medium, high, critical
    OverallStatus        string    // clear, flagged, rejected
    WatchlistMatch       bool
    SanctionsMatch       bool
    PEPMatch             bool
    ManualReviewRequired bool
    ApprovalDeadline     time.Time
    ScreeningCompletedAt time.Time
}
```

#### `EscalateAMLFindingsActivity`
**Routes critical findings to management**

Called when `riskScore >= 80` or `sanctionsMatch == true`

#### `SendAMLApprovalNotificationActivity`
**Notifies compliance team of pending reviews**

#### `RecordAMLScreeningAuditActivity`
**Creates compliance audit trail entry**

---

## 5. ABAC Integration

### Access Control by Role

| Role | View Results | Approve | Escalate | View History |
|------|-------------|---------|----------|--------------|
| Client | Own only | ✗ | ✗ | Own only |
| Advisor | Yes | ✗ | ✗ | Yes |
| ComplianceOfficer | Yes | ✓ | ✗ | Yes |
| ComplianceManager | Yes | ✓ (all) | ✓ | Yes |
| ComplianceDirector | Yes | ✓ (all) | ✓ | Yes |
| Admin | Yes | ✓ | ✓ | Yes |

### Temporal Policies

**Policy 1: Compliance Officer Review Window**
- Requirement: All AML approvals completed within 24h (critical) or 48h (standard)
- Enforcement: Temporal cron job escalates overdue screenings
- Action: Auto-escalate to ComplianceManager

**Policy 2: Geographic Restriction**
- Requirement: AML result access restricted by office location
- Header: `X-Office-Country` checked against allowlist
- Enforcement: HTTP middleware validates before handler execution

**Policy 3: Audit Logging**
- Requirement: All AML operations logged with user context
- Events: screening_created, screening_approved, screening_rejected, screening_escalated
- Retention: 7 years for regulatory compliance

---

## 6. External Provider Integration

### Supported Providers

#### 1. **LexisNexis**
- API: Lexis Nexis Risk Intelligence
- Databases: OFAC, UN, HMT, GCSB, SECO
- Response Time: 2-5 seconds
- Cost: $1-2 per lookup

#### 2. **World Check**
- API: Refinitiv World Check
- Databases: Global watchlists, PEP, sanctions
- Response Time: 1-3 seconds
- Cost: $0.50-1 per lookup

#### 3. **Dow Jones**
- API: Dow Jones Risk & Compliance
- Databases: Watchlists, adverse media, regulatory filings
- Response Time: 3-8 seconds
- Cost: $1-3 per lookup

#### 4. **Internal**
- Rule-based screening using regulatory lists
- No external API call
- Response Time: <100ms
- Cost: Free (maintenance only)

### Implementation Pattern

```go
// Switch on provider type
switch input.ScreeningProvider {
case "lexis_nexis":
    result, err := callLexisNexisAPI(clientData)
    
case "worldcheck":
    result, err := callWorldCheckAPI(clientData)
    
case "dow_jones":
    result, err := callDowJonesAPI(clientData)
    
case "internal":
    result := performInternalAMLScreening(clientData)
}

// Normalize response to AMLScreeningOutput
return normalizeProviderResponse(result)
```

---

## 7. Database Schema Updates

### New Table: `kyc_aml_results` (Enhanced)

```sql
CREATE TABLE kyc_aml_results (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    client_id UUID NOT NULL,
    screening_date TIMESTAMP NOT NULL,
    screening_provider VARCHAR(50),           -- lexis_nexis, worldcheck, etc.
    screening_status VARCHAR(50),             -- pending, in_progress, completed, failed
    risk_score DECIMAL(5,2),
    risk_level VARCHAR(20),                   -- low, medium, high, critical
    overall_status VARCHAR(20),               -- clear, flagged, rejected
    
    -- Detailed findings (JSONB)
    watchlist_match BOOLEAN DEFAULT FALSE,
    watchlist_matches JSONB,                  -- Array of matching watchlists
    sanctions_match BOOLEAN DEFAULT FALSE,
    sanctions_details TEXT,
    pep_match BOOLEAN DEFAULT FALSE,
    pep_level VARCHAR(20),                    -- low, medium, high
    
    -- Risk flags
    high_net_worth_flag BOOLEAN DEFAULT FALSE,
    unknown_funds_flag BOOLEAN DEFAULT FALSE,
    risky_countries_flag BOOLEAN DEFAULT FALSE,
    risky_countries JSONB,                    -- Array of countries
    source_of_funds_verified BOOLEAN DEFAULT FALSE,
    beneficial_owner_flags BOOLEAN DEFAULT FALSE,
    adverse_media_flag BOOLEAN DEFAULT FALSE,
    adverse_media_details TEXT,
    
    -- Manual review
    manual_review_required BOOLEAN DEFAULT FALSE,
    manual_review_reason TEXT,
    
    -- Approval/Rejection
    approved_by UUID,
    approved_at TIMESTAMP,
    rejected_by UUID,
    rejected_at TIMESTAMP,
    rejection_reason TEXT,
    
    -- Audit
    raw_api_response JSONB,
    compliance_notes TEXT,
    created_by UUID NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    
    FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    FOREIGN KEY (datasource_id) REFERENCES datasources(id),
    FOREIGN KEY (client_id) REFERENCES clients(id),
    
    INDEX (tenant_id, client_id, screening_date),
    INDEX (risk_level),
    INDEX (overall_status),
    INDEX (manual_review_required)
);
```

---

## 8. Deployment Checklist

- [ ] Deploy database migrations (schema updates)
- [ ] Implement `AMLScreeningService` (backend/internal/api/client_aml_screening.go)
- [ ] Implement `AMLScreeningHandler` with ABAC checks (backend/internal/api/client_aml_handlers.go)
- [ ] Register routes in API router
- [ ] Update `ClientOnboardingWorkflow` to include Step 2
- [ ] Implement `PerformAMLScreeningActivity` with provider connectors
- [ ] Configure external API credentials (LexisNexis, WorldCheck, etc.) in config.yaml
- [ ] Add ABAC policies to policy engine
- [ ] Configure temporal deadlines (24h for critical, 48h for standard)
- [ ] Setup audit logging to compliance database
- [ ] Test end-to-end with test clients (low-risk, medium-risk, critical-risk)
- [ ] Train compliance team on new approval workflow
- [ ] Document incident procedures for sanctions matches
- [ ] Schedule quarterly regulatory update review

---

## 9. Compliance & Regulatory Reference

**Regulatory Frameworks Aligned**:
- ✓ FinCEN (US Financial Crimes Enforcement Network)
- ✓ FATF (Financial Action Task Force) Recommendations
- ✓ FINRA (Financial Industry Regulatory Authority) Rules
- ✓ OCC (Office of the Comptroller of the Currency) Guidelines
- ✓ OFAC (Office of Foreign Assets Control) Requirements

**Audit Trail Retention**: 7 years minimum
**Screening Frequency**: Initial on onboarding, periodic re-screening (quarterly/annually)
**Escalation Requirements**: Mandatory for sanctions matches, PEP at senior level, high-risk countries

---

## 10. Next Steps

1. **Phase 1 (Weeks 1-2)**: Implement AML service layer and handlers
2. **Phase 2 (Weeks 3-4)**: Integrate Temporal activities, test with internal provider
3. **Phase 3 (Weeks 5-6)**: Connect external providers (LexisNexis, WorldCheck)
4. **Phase 4 (Weeks 7-8)**: ABAC policy enforcement, audit logging
5. **Phase 5 (Week 9)**: UAT with compliance team, regulatory review
6. **Phase 6 (Week 10)**: Production deployment, monitoring

---

**Document Version**: 1.0  
**Last Updated**: October 28, 2025  
**Status**: Ready for Implementation
