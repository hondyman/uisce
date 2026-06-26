# AML Screening - Integration Guide for Developers

## Overview

This guide explains how to integrate the new AML screening service into your existing Client Onboarding system. It covers database setup, backend wiring, Temporal workflow updates, and testing.

---

## Step 1: Database Setup

### 1.1 Run Migrations

The `kyc_aml_results` table is already defined in the existing schema. Verify it exists:

```bash
# Run migration
psql postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable < migrations/client_onboarding_schema.sql

# Verify table exists
psql postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable \
  -c "\d kyc_aml_results"

# Expected output: Table with 23 columns (id, tenant_id, datasource_id, client_id, ...)
```

### 1.2 Verify Indices

```sql
-- Check indices are created for performance
SELECT indexname FROM pg_indexes 
WHERE tablename = 'kyc_aml_results'
ORDER BY indexname;

-- Expected indices:
-- - idx_kyc_aml_results_client_date
-- - idx_kyc_aml_results_status
-- - idx_kyc_aml_results_overall_status
```

---

## Step 2: Backend Service Integration

### 2.1 Import AML Service in Your Router Setup

```go
// In backend/internal/api/api.go or your main router file

import (
    "github.com/jmoiron/sqlx"
    "github.com/go-chi/chi/v5"
)

func setupOnboardingRoutes(router *chi.Mux, db *sqlx.DB) {
    // Existing onboarding routes
    onboardingHandler := NewClientOnboardingHandler(db)
    RegisterClientOnboardingRoutes(router, onboardingHandler)
    
    // NEW: AML Screening routes
    amlService := NewAMLScreeningService(db)
    onboardingService := NewClientOnboardingService(db)
    RegisterAMLScreeningRoutes(router, amlService, onboardingService)
}
```

### 2.2 Verify Types Import

The AML types are defined in `client_aml_screening.go`. Ensure they're imported:

```go
import (
    "your-module/backend/internal/api"
)

// Now available:
// - api.AMLScreeningService
// - api.AMLScreeningResult
// - api.AMLRiskScoreInput
// - api.AMLScreeningRequest
```

### 2.3 Register Routes in Chi Router

```go
// In your main router setup (e.g., api.go)

func SetupRouter(db *sqlx.DB) *chi.Mux {
    r := chi.NewRouter()
    
    // Existing middleware
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    r.Use(tenantContextMiddleware) // Your ABAC middleware
    
    // Register all onboarding routes
    onboardingHandler := NewClientOnboardingHandler(db)
    RegisterClientOnboardingRoutes(r, onboardingHandler)
    
    // Register AML routes (NEW)
    amlService := NewAMLScreeningService(db)
    RegisterAMLScreeningRoutes(r, amlService, onboardingHandler.service)
    
    return r
}
```

---

## Step 3: Temporal Workflow Integration

### 3.1 Update ClientOnboardingWorkflow

Replace the existing Step 1-5 workflow with the new 6-step version:

```go
// In temporal/workflows/client_onboarding_workflow.go

func ClientOnboardingWorkflow(ctx workflow.Context, input ClientOnboardingWorkflowInput) (*ClientOnboardingWorkflowState, error) {
    state := &ClientOnboardingWorkflowState{
        ClientID:    input.ClientID,
        CurrentStep: 1,
    }

    logger := workflow.GetLogger(ctx)

    // STEP 1: Validate Client Data
    logger.Info("Step 1: Validating client data", "clientID", input.ClientID)
    validationResult := executeActivity(ctx, ValidateClientDataActivity, input)
    if validationResult.HasErrors {
        state.ValidationPassed = false
        state.ValidationErrors = validationResult.Errors
        return state, nil
    }

    // STEP 2: AML SCREENING (NEW)
    logger.Info("Step 2: Performing AML screening", "clientID", input.ClientID)
    state.CurrentStep = 2
    
    amlInput := &AMLScreeningInput{
        ClientID:             input.ClientID,
        ScreeningProvider:    input.AMLProvider, // From workflow input
        // ... other fields populated from client data
    }
    
    amlResult := executeActivity(ctx, PerformAMLScreeningActivity, amlInput)
    
    // Handle critical findings
    if amlResult.RiskLevel == "critical" || amlResult.SanctionsMatch {
        logger.Info("Critical AML finding - escalating",
            "clientID", input.ClientID,
            "riskScore", amlResult.RiskScore)
        
        // Escalation workflow with 24-hour deadline
        escalationResult := executeChildWorkflow(ctx, AMLEscalationWorkflow, &AMLEscalationInput{
            ScreeningID:      amlResult.ScreeningID,
            RiskScore:        amlResult.RiskScore,
            ReasonString:     amlResult.ManualReviewReason,
            DeadlineMinutes:  1440, // 24 hours
        })
        
        // Wait for escalation decision
        escalationState, err := waitForChildWorkflowResult(escalationResult)
        if err != nil || escalationState.Decision == "rejected" {
            state.OverallStatus = "rejected"
            state.RejectionReason = "AML screening rejected"
            return state, nil
        }
    }

    // If we get here, AML screening passed or was approved
    state.CurrentStep = 3

    // STEP 3: Route for Advisor Review
    logger.Info("Step 3: Routing for advisor review")
    advisorResult := executeActivity(ctx, RouteForAdvisorReviewActivity, input)
    
    // Set up 48-hour approval deadline signal
    advisorSignalCtx, advisorCancel := workflow.WithDeadline(ctx, time.Hour*48)
    defer advisorCancel()
    
    var advisorApprovalSignal ApprovalSignal
    err = workflow.ReceiveSignal(advisorSignalCtx, "advisor_approval", &advisorApprovalSignal)
    if err != nil {
        // Timeout or rejected
        logger.Info("Advisor approval timeout or rejected")
        return state, err
    }

    state.ApprovedBy = &advisorApprovalSignal.ApprovedBy
    state.CurrentStep = 4

    // STEP 4: Generate Agreements
    logger.Info("Step 4: Generating agreements")
    agreementsResult := executeActivity(ctx, GenerateAndSendAgreementsActivity, input)

    // Set up 7-day signature deadline
    signatureSignalCtx, signatureCancel := workflow.WithDeadline(ctx, time.Hour*168)
    defer signatureCancel()

    var signatureSignal SignatureSignal
    err = workflow.ReceiveSignal(signatureSignalCtx, "agreements_signed", &signatureSignal)
    if err != nil {
        logger.Info("Signature timeout")
        return state, err
    }

    state.CurrentStep = 5

    // STEP 5: Create Accounts
    logger.Info("Step 5: Creating accounts and portfolios")
    accountsResult := executeActivity(ctx, CreateAccountsAndPortfoliosActivity, input)

    state.AccountsCreatedCount = len(accountsResult.Accounts)
    state.CurrentStep = 6

    // STEP 6: Notify Client
    logger.Info("Step 6: Notifying client and activating portal")
    notificationResult := executeActivity(ctx, NotifyClientOnCompletionActivity, input)

    state.NotificationSentTime = &notificationResult.SentAt
    state.OverallStatus = "completed"

    logger.Info("Onboarding workflow completed successfully",
        "clientID", input.ClientID,
        "accountsCreated", state.AccountsCreatedCount)

    return state, nil
}
```

### 3.2 Wire Temporal Activities

```go
// In temporal/worker setup (main.go or temporal.go)

import (
    activities_pkg "your-module/temporal/activities"
)

func registerActivities(worker worker.Worker) {
    // Existing activities
    worker.RegisterActivityWithOptions(
        activities_pkg.ValidateClientDataActivity,
        activity.RegisterOptions{Name: "ValidateClientDataActivity"},
    )
    
    // NEW: AML Screening activities
    worker.RegisterActivityWithOptions(
        activities_pkg.PerformAMLScreeningActivity,
        activity.RegisterOptions{Name: "PerformAMLScreeningActivity"},
    )
    worker.RegisterActivityWithOptions(
        activities_pkg.EscalateAMLFindingsActivity,
        activity.RegisterOptions{Name: "EscalateAMLFindingsActivity"},
    )
    worker.RegisterActivityWithOptions(
        activities_pkg.SendAMLApprovalNotificationActivity,
        activity.RegisterOptions{Name: "SendAMLApprovalNotificationActivity"},
    )
    worker.RegisterActivityWithOptions(
        activities_pkg.RecordAMLScreeningAuditActivity,
        activity.RegisterOptions{Name: "RecordAMLScreeningAuditActivity"},
    )
    
    // ... remaining activities
}
```

---

## Step 4: ABAC Middleware Integration

### 4.1 Add ABAC Context Extraction

Your existing tenant middleware should already extract headers. Verify it includes:

```go
func tenantContextMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Extract tenant context
        tenantID := r.Header.Get("X-Tenant-ID")
        datasourceID := r.Header.Get("X-Tenant-Datasource-ID")
        userID := r.Header.Get("X-User-ID")
        userRole := r.Header.Get("X-User-Role")  // NEW for ABAC
        
        // Validate context exists
        if tenantID == "" || userID == "" {
            http.Error(w, "Missing tenant context", http.StatusBadRequest)
            return
        }
        
        // Store in context
        ctx := r.Context()
        ctx = context.WithValue(ctx, "tenant_id", tenantID)
        ctx = context.WithValue(ctx, "user_id", userID)
        ctx = context.WithValue(ctx, "user_role", userRole)  // NEW
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### 4.2 Verify Role-Based Access Enforcement

The AML handlers already check roles. Test with different user roles:

```bash
# Test 1: ComplianceOfficer initiates screening (should succeed)
curl -X POST \
  -H "X-Tenant-ID: tenant-123" \
  -H "X-User-Role: ComplianceOfficer" \
  -H "Content-Type: application/json" \
  "http://localhost:8080/api/onboarding/cli-123/step2-aml-screening"
# Expected: 202 Accepted

# Test 2: Advisor tries to initiate screening (should fail)
curl -X POST \
  -H "X-Tenant-ID: tenant-123" \
  -H "X-User-Role: Advisor" \
  "http://localhost:8080/api/onboarding/cli-123/step2-aml-screening"
# Expected: 403 Forbidden
```

---

## Step 5: External AML Provider Configuration

### 5.1 Add Provider Credentials to config.yaml

```yaml
aml:
  enabled: true
  default_provider: "lexis_nexis"  # lexis_nexis, worldcheck, dow_jones, internal
  
  lexis_nexis:
    enabled: true
    api_url: "https://api.lexisnexis.com/risk-intelligence/v1"
    api_key: "${AML_LEXIS_NEXIS_KEY}"
    timeout_seconds: 30
    
  worldcheck:
    enabled: true
    api_url: "https://api.refinitiv.com/worldcheck/v1"
    api_key: "${AML_WORLDCHECK_KEY}"
    timeout_seconds: 30
    
  dow_jones:
    enabled: true
    api_url: "https://api.dowjones.com/risk-intelligence/v1"
    api_key: "${AML_DOW_JONES_KEY}"
    timeout_seconds: 30
    
  internal:
    enabled: true  # Fallback, always enabled
```

### 5.2 Load Configuration in Service

```go
// In client_aml_screening.go or wherever you initialize services

type AMLConfig struct {
    Provider    string
    LexisNexis  ProviderConfig
    WorldCheck  ProviderConfig
    DowJones    ProviderConfig
}

type ProviderConfig struct {
    Enabled        bool
    APIURL         string
    APIKey         string
    TimeoutSeconds int
}

// Load from config.yaml
amlConfig := loadAMLConfig("config.yaml")
amlService := NewAMLScreeningService(db, amlConfig)
```

---

## Step 6: Testing

### 6.1 Unit Test: Risk Scoring

```go
// In backend/internal/api/client_aml_screening_test.go

func TestComputeAMLRiskScore(t *testing.T) {
    service := NewAMLScreeningService(db)
    
    tests := []struct {
        name     string
        input    AMLRiskScoreInput
        expected float64
        level    string
    }{
        {
            name: "Low risk client",
            input: AMLRiskScoreInput{
                NetWorth: ptrFloat64(2000000),
                WatchlistMatch: false,
                SanctionsMatch: false,
                PEPStatus: false,
            },
            expected: 0,
            level: "low",
        },
        {
            name: "High net worth + unknown funds",
            input: AMLRiskScoreInput{
                NetWorth: ptrFloat64(10000000),
                UnknownSourceOfFunds: true,
                WatchlistMatch: false,
                SanctionsMatch: false,
            },
            expected: 45, // 20 + 25
            level: "medium",
        },
        {
            name: "Critical - sanctions match",
            input: AMLRiskScoreInput{
                SanctionsMatch: true,
                WatchlistMatch: true,
            },
            expected: 90, // 50 + 40
            level: "critical",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            score, level := service.ComputeAMLRiskScore(tt.input)
            assert.Equal(t, tt.expected, score)
            assert.Equal(t, tt.level, level)
        })
    }
}
```

### 6.2 Integration Test: End-to-End

```go
// In integration test suite

func TestAMLScreeningFlow(t *testing.T) {
    // 1. Create client
    client := createTestClient(t, db)
    
    // 2. Initiate AML screening
    req := &AMLScreeningRequest{
        ClientID: client.ID,
        ScreeningProvider: "internal",
    }
    screening, err := amlService.CreateAMLScreening(ctx, tenant, datasource, user, req, client)
    require.NoError(t, err)
    assert.NotNil(t, screening.ID)
    assert.Equal(t, "pending", screening.ScreeningStatus)
    
    // 3. Simulate screening completion (in real workflow, Temporal activity does this)
    updated, err := amlService.UpdateAMLScreeningStatus(
        ctx, screening.ID, tenant, user, "approved", "Passed internal screening", nil,
    )
    require.NoError(t, err)
    assert.Equal(t, "completed", updated.ScreeningStatus)
    assert.Equal(t, "clear", updated.OverallStatus)
    
    // 4. Retrieve screening
    retrieved, err := amlService.GetAMLScreening(ctx, tenant, screening.ID)
    require.NoError(t, err)
    assert.Equal(t, screening.ID, retrieved.ID)
}
```

### 6.3 API Test: REST Endpoints

```bash
#!/bin/bash

# Start test server
go run main.go &
SERVER_PID=$!
sleep 2

# Create test client
CLIENT_ID=$(curl -s -X POST \
  -H "X-Tenant-ID: test-tenant" \
  -H "X-User-ID: test-user" \
  -H "Content-Type: application/json" \
  -d '{
    "first_name": "John",
    "last_name": "Doe",
    "email": "john@example.com",
    "risk_profile": "moderate"
  }' \
  "http://localhost:8080/api/clients" | jq -r '.id')

echo "Created client: $CLIENT_ID"

# Test 1: Initiate AML screening
echo "Test 1: Initiate AML screening"
curl -X POST \
  -H "X-Tenant-ID: test-tenant" \
  -H "X-User-Role: ComplianceOfficer" \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "'$CLIENT_ID'",
    "screening_provider": "internal"
  }' \
  "http://localhost:8080/api/onboarding/$CLIENT_ID/step2-aml-screening" | jq

# Test 2: Get screening results
SCREENING_ID=$(curl -s "http://localhost:8080/api/onboarding/$CLIENT_ID/aml-screening/latest" \
  -H "X-Tenant-ID: test-tenant" \
  -H "X-User-Role: ComplianceOfficer" | jq -r '.id')

echo "Test 2: Get screening results"
curl "http://localhost:8080/api/onboarding/$CLIENT_ID/aml-screening/$SCREENING_ID" \
  -H "X-Tenant-ID: test-tenant" \
  -H "X-User-Role: ComplianceOfficer" | jq

# Test 3: Approve screening
echo "Test 3: Approve screening"
curl -X POST \
  -H "X-Tenant-ID: test-tenant" \
  -H "X-User-ID: compliance-manager-001" \
  -H "X-User-Role: ComplianceManager" \
  -H "Content-Type: application/json" \
  -d '{
    "screening_id": "'$SCREENING_ID'",
    "approval_status": "approved",
    "compliance_notes": "Internal screening completed successfully"
  }' \
  "http://localhost:8080/api/onboarding/aml-screening/$SCREENING_ID/review" | jq

# Cleanup
kill $SERVER_PID
```

---

## Step 7: Deployment Checklist

```
Pre-Deployment:
  [ ] All tests passing (unit + integration + API)
  [ ] Database migrations applied
  [ ] External AML provider credentials configured
  [ ] ABAC policies reviewed by security team
  [ ] Compliance team reviewed risk algorithm
  [ ] Load test completed (1000+ concurrent requests)

Deployment:
  [ ] Merge to main branch
  [ ] Build Docker image
  [ ] Deploy to staging
  [ ] Verify endpoints responding
  [ ] Test with compliance team
  [ ] Run smoke tests in staging
  [ ] Deploy to production

Post-Deployment:
  [ ] Monitor error rates
  [ ] Verify audit logging
  [ ] Check database performance
  [ ] Validate external API calls
  [ ] Train support team
  [ ] Document runbooks
```

---

## Troubleshooting Integration Issues

### Issue 1: Compilation Error - "undefined: AMLScreeningService"

**Solution**: Ensure you're importing from the correct package:
```go
import (
    api "your-module/backend/internal/api"
)

// Then use:
amlService := api.NewAMLScreeningService(db)
```

### Issue 2: ABAC Check Failing for ComplianceOfficer

**Solution**: Verify header is being passed:
```bash
# Check header is present
curl -v http://localhost:8080/api/onboarding/cli-123/step2-aml-screening \
  -H "X-User-Role: ComplianceOfficer"

# Look for "X-User-Role: ComplianceOfficer" in request headers
```

### Issue 3: Temporal Activity Not Triggering

**Solution**: Verify activity is registered:
```go
// Check in Temporal worker setup
worker.RegisterActivityWithOptions(
    activities_pkg.PerformAMLScreeningActivity,
    activity.RegisterOptions{Name: "PerformAMLScreeningActivity"},
)
```

### Issue 4: External API Timeout

**Solution**: Check provider configuration:
```yaml
# In config.yaml
aml:
  lexis_nexis:
    timeout_seconds: 30  # Increase if needed
```

---

## Next Steps

1. Follow Steps 1-7 above in order
2. Reference `AML_SCREENING_ENHANCEMENT_GUIDE.md` for detailed implementation
3. Run all test scenarios
4. Get compliance team sign-off
5. Deploy to production
6. Monitor for 1 week

---

**Integration Guide Version**: 1.0  
**Last Updated**: October 28, 2025  
**Status**: Ready for Developer Integration
