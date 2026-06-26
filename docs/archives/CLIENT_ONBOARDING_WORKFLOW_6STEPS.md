# Client Onboarding Workflow - Updated 6-Step Process with AML Integration

## Overview

The enhanced Client Onboarding process now incorporates a dedicated, detailed AML (Anti-Money Laundering) screening step that runs after initial client validation. This ensures comprehensive compliance with regulatory requirements while maintaining operational efficiency through automation and workflow orchestration.

---

## Updated Process Flow

```
┌─────────────────────────────────────────────────────────────────────┐
│                  CLIENT ONBOARDING WORKFLOW - 6 STEPS               │
└─────────────────────────────────────────────────────────────────────┘

START: "Client Application Submitted"
  │
  ├─► STEP 1: VALIDATE CLIENT DATA (Initial KYC)
  │   - Verify name, address, identification numbers
  │   - Check basic KYC requirements (age, documents, etc.)
  │   - Compute preliminary risk profile
  │   - Duration: 5 minutes
  │   - Status: pending_validation → validation_complete / validation_failed
  │
  ├─► STEP 2: PERFORM AML SCREENING ⭐ [NEW]
  │   - Screen against global watchlists (OFAC, UN, HMT, etc.)
  │   - Perform PEP (Politically Exposed Person) checks
  │   - Screen against sanctions lists
  │   - Check for adverse media / negative news
  │   - Verify source of funds
  │   - Compute comprehensive AML risk score (0-100)
  │   - Determine if manual review required
  │   ├─► If CRITICAL (score ≥ 80 or sanctions match)
  │   │   └─► Escalation Workflow (24h deadline)
  │   │       ├─► Manager Review → Approval/Rejection
  │   │       └─► If rejected → STOP (compliance failure)
  │   ├─► If HIGH (score 60-79)
  │   │   └─► Compliance Officer Review (48h deadline)
  │   │       └─► Approval/Rejection
  │   └─► If CLEAR (score < 60)
  │       └─► Auto-proceed to next step
  │   Duration: 2-30 seconds (depending on provider)
  │   Status: pending → in_progress → completed → flagged / rejected / clear
  │
  ├─► STEP 3: ROUTE FOR ADVISOR REVIEW
  │   - Assign to appropriate advisor (match by risk profile, expertise)
  │   - Create review task in advisor dashboard
  │   - Send assignment notification
  │   - Set 48-hour approval deadline (temporal policy)
  │   - Duration: <1 second
  │   - Status: pending_advisor_review → advisor_assigned
  │
  ├─► STEP 4: GENERATE & SEND AGREEMENTS FOR E-SIGNATURE
  │   - Generate service agreements (customized based on AML status)
  │   - Populate with client and advisor information
  │   - Integrate with DocuSign / HelloSign API
  │   - Send agreements via email
  │   - Set 7-day signature deadline
  │   - Track e-signature request IDs
  │   - Duration: 5-10 seconds
  │   - Status: pending_agreements → agreements_sent → agreements_signed / agreements_expired
  │
  ├─► STEP 5: CREATE ACCOUNTS & PORTFOLIOS
  │   - Create investment accounts per client preferences
  │   - Allocate portfolio based on risk profile
  │   - Set initial funding amount
  │   - Configure account permissions
  │   - Create portfolio holdings
  │   - Duration: 5-10 seconds
  │   - Status: pending_accounts → accounts_created
  │
  └─► STEP 6: NOTIFY CLIENT & ACTIVATE PORTAL
      - Generate welcome email with portal credentials
      - Activate client portal access
      - Create initial support ticket
      - Send onboarding completion confirmation
      - Duration: 2-5 seconds
      - Status: pending_notification → notification_sent → onboarding_complete

END: Client is active and can access portal
```

---

## Step-by-Step Details

### STEP 1: Validate Client Data (Initial KYC)

**Business Process**:
1. Extract client-provided information
2. Verify completeness against KYC checklist
3. Validate identification numbers (SSN/passport format)
4. Collect supporting documents (ID proof, proof of address)
5. Conduct preliminary risk assessment

**Validation Rules Applied**:
- Identification number required (SSN/TIN/Passport)
- Age verification (minimum 18 years)
- Country of citizenship specified
- Contact information valid (email, phone)
- Document uploads received

**Outcome**:
- ✓ **Pass**: Proceed to Step 2 (AML Screening)
- ✗ **Fail**: Request additional information, retry validation

**Temporal Details**:
- Activity Timeout: 5 minutes
- Retry Policy: 2 retries on failure
- SLA: 1 business day

---

### STEP 2: Perform AML Screening ⭐ [NEW DEDICATED STEP]

**Business Process**:
1. Extract client identifiers (name, DOB, SSN, address)
2. Call external AML provider API (LexisNexis, WorldCheck, Dow Jones, or internal)
3. Check multiple databases simultaneously:
   - OFAC Specially Designated Nationals (SDN) List
   - UN Security Council Sanctions List
   - HM Treasury Consolidated List
   - SECO (Switzerland) Sanctions List
   - Global PEP databases
   - Adverse media feeds
4. Analyze results and compute risk score
5. Determine if manual review required
6. Create compliance audit trail entry

**Risk Score Calculation**:

| Factor | Points | Trigger |
|--------|--------|---------|
| Watchlist Match | +50 | Found on any global watchlist |
| Sanctions | +40 | OFAC/EU/UN/UK designations |
| PEP Match | +25 | Identified as politically exposed person |
| High Net Worth (>$5M) | +20 | Complexity and volume risk |
| Unknown Source of Funds | +15-25 | Higher impact for wealthy clients |
| High-Risk Countries | +30 | FATF grey/black list countries |
| Adverse Media | +20 | Negative news, criminal allegations |

**Risk Level Determination**:

| Score Range | Level | Action |
|------------|-------|--------|
| 0-20 | LOW | Auto-approved, proceed to Step 3 |
| 20-40 | MEDIUM_LOW | Standard advisor review in Step 3 |
| 40-60 | MEDIUM | Advisor + compliance oversight |
| 60-80 | HIGH | Mandatory manual compliance review (48h) |
| 80-100 | CRITICAL | Escalation to manager (24h), potential rejection |

**Escalation Workflow (for CRITICAL findings)**:

```
Critical Finding (score ≥ 80 or sanctions match)
  │
  ├─► Create escalation task
  ├─► Route to AML Manager
  ├─► Set 24-hour deadline
  ├─► Send priority notification
  │
  └─► Manager Decision (within 24h)
      ├─► APPROVED → Proceed to Step 3
      ├─► REJECTED → Client application denied (compliance failure)
      └─► TIMEOUT → Auto-escalate to Compliance Director
```

**External Provider Integration**:

```javascript
// Provider Options
if (screeningProvider === "lexis_nexis") {
  result = await callLexisNexisAPI({
    firstName: client.firstName,
    lastName: client.lastName,
    dateOfBirth: client.dateOfBirth,
    ssn: client.identificationNumber,
    address: client.address
  });
} else if (screeningProvider === "worldcheck") {
  result = await callWorldCheckAPI({...});
} else if (screeningProvider === "dow_jones") {
  result = await callDowJonesAPI({...});
} else {
  result = performInternalAMLScreening({...});
}

// Normalize response
return {
  riskScore: computeRiskScore(result),
  watchlistMatch: result.hasWatchlistMatch,
  sanctionsMatch: result.hasSanctionsDesignation,
  pepMatch: result.isPoliticallyExposed,
  riskLevel: determineRiskLevel(riskScore),
  overallStatus: riskScore >= 80 ? "flagged" : "clear"
};
```

**Outcome**:
- ✓ **Clear**: Continue to Step 3
- ⚠ **Flagged**: Manual review required
- ✗ **Rejected**: Application denied

**ABAC Controls**:
- Only `ComplianceOfficer` role can initiate screening
- Only `ComplianceManager`/`ComplianceDirector` can approve/reject
- Approval deadline enforced by temporal policy

**Temporal Details**:
- Activity Timeout: 30 seconds
- Retry Policy: 1 retry on API failure
- Provider Response SLA: 1-30 seconds
- Approval Deadline: 24h (critical), 48h (high)

---

### STEP 3: Route for Advisor Review

**Business Process**:
1. Load client profile and AML screening results
2. Query available advisors
3. Match advisor based on:
   - Risk profile alignment (client risk vs advisor expertise)
   - Workload (fewest active clients)
   - Specialization (high-net-worth, emerging markets, etc.)
   - Office location (proximity or timezone)
4. Create review task in advisor's dashboard
5. Send assignment notification
6. Set 48-hour review deadline

**Advisor Assignment Logic**:

```go
// Match advisor based on client characteristics
if client.RiskProfile == "high" || client.NetWorth > 5000000 {
  advisors = filterBySpecialization("high_net_worth")
} else if client.NetWorth > 1000000 {
  advisors = filterBySpecialization("wealth_management")
} else {
  advisors = filterBySpecialization("general")
}

// Among candidates, choose lowest workload
assignedAdvisor = advisors.MinBy(activeClientCount)

// Create task with deadline
task = {
  type: "client_review",
  deadline: now + 48*hours,
  priority: client.RiskLevel == "high" ? "urgent" : "normal"
}
```

**Outcome**:
- ✓ **Assigned**: Advisor receives notification, task created
- Task requires advisor approval or rejection

**Temporal Details**:
- Activity Timeout: 5 seconds
- Advisor Response Deadline: 48 hours (monitored)
- Escalation: If no response in 48h, reassign to manager

---

### STEP 4: Generate & Send Agreements for E-Signature

**Business Process**:
1. Load agreement templates
2. Customize based on client attributes (name, advisor, AML status)
3. Add special clauses if high-risk:
   - Enhanced AML compliance acknowledgment
   - Source of funds verification attestation
   - Ongoing monitoring notifications
4. Generate PDF documents
5. Integrate with e-signature provider (DocuSign/HelloSign)
6. Send signature request via email
7. Track signature status

**Agreement Customization Example**:

```go
// Standard clause for all clients
agreements = []string{"service_agreement", "privacy_policy"}

// Additional clauses for flagged AML screening
if amlScreening.RiskLevel == "high" {
  agreements = append(agreements,
    "aml_compliance_acknowledgment",
    "source_of_funds_verification",
    "ongoing_monitoring_notice"
  )
}

// Additional for critical risk
if amlScreening.RiskLevel == "critical" {
  agreements = append(agreements,
    "enhanced_due_diligence_notice",
    "transaction_monitoring_consent",
    "annual_certification_requirement"
  )
}

// Populate template with client data
for _, template := range agreements {
  doc = populateTemplate(template, {
    clientName: client.FirstName + " " + client.LastName,
    advisorName: advisor.FirstName + " " + advisor.LastName,
    amlStatus: amlScreening.OverallStatus,
    effectiveDate: time.Now()
  })
  
  docuSignRequest = createDocuSignEnvelope(doc)
}
```

**E-Signature Integration**:

```go
// Send via DocuSign
response = docuSignClient.CreateEnvelope({
  Documents: agreements,
  Recipients: [{
    Email: client.Email,
    Name: client.FirstName + " " + client.LastName,
    RecipientType: "signer"
  }],
  Status: "sent"
})

// Store tracking IDs
for _, doc := range agreements {
  storeDocumentTracking({
    ClientID: client.ID,
    DocumentType: doc.Type,
    DocuSignRequestID: response.RequestID,
    SentAt: time.Now(),
    SignatureDeadline: time.Now().Add(7 * 24 * time.Hour)
  })
}
```

**Outcome**:
- ✓ **Sent**: Client receives email with signature links
- Client must sign within 7 days
- Reminder email sent if not signed by day 5

**Temporal Details**:
- Activity Timeout: 10 seconds
- Signature Deadline: 7 days
- Reminder: Day 5 and Day 7
- Escalation: After 7 days, require manual intervention

---

### STEP 5: Create Accounts & Portfolios

**Business Process**:
1. Verify AML clearance (flagged screenings cannot proceed)
2. Create primary investment account
3. Allocate portfolio based on risk profile:
   - Conservative: 40% bonds, 50% large-cap, 10% cash
   - Moderate: 30% bonds, 50% large-cap, 15% mid-cap, 5% cash
   - Aggressive: 70% equities, 20% alternatives, 10% cash
   - Very Aggressive: 80% equities, 15% alternatives, 5% cash
4. Set initial holdings
5. Configure account permissions and restrictions
6. Link to advisor

**Portfolio Allocation Formula**:

```go
func allocatePortfolio(riskProfile string, netWorth float64) map[string]float64 {
  alloc := make(map[string]float64)
  
  switch riskProfile {
  case "low":
    alloc["bonds"] = 0.50
    alloc["large_cap"] = 0.40
    alloc["cash"] = 0.10
  case "moderate":
    alloc["bonds"] = 0.30
    alloc["large_cap"] = 0.50
    alloc["mid_cap"] = 0.15
    alloc["cash"] = 0.05
  case "high":
    alloc["large_cap"] = 0.50
    alloc["mid_cap"] = 0.20
    alloc["small_cap"] = 0.15
    alloc["alternatives"] = 0.10
    alloc["cash"] = 0.05
  case "very_high":
    alloc["large_cap"] = 0.40
    alloc["mid_cap"] = 0.25
    alloc["small_cap"] = 0.15
    alloc["alternatives"] = 0.15
    alloc["international"] = 0.05
  }
  
  // For very high net worth, add alternatives
  if netWorth > 10000000 {
    alloc["alternatives"] = alloc["alternatives"] + 0.10
    alloc["cash"] = alloc["cash"] - 0.10 // Maintain 100%
  }
  
  return alloc
}
```

**Account Creation Logic**:

```go
// Step 1: Create account
account = {
  Type: "brokerage",
  Status: "active",
  Currency: "USD",
  MinimumBalance: 10000,
  FundingSource: client.BankAccount
}

// Step 2: Verify AML clearance
if !verifyAMLClearance(client.ID) {
  return error("AML clearance required before account creation")
}

// Step 3: Create portfolio
portfolio = {
  AccountID: account.ID,
  RiskProfile: client.RiskProfile,
  Allocation: allocatePortfolio(client.RiskProfile, client.NetWorth),
  TargetValue: client.InitialFunding
}

// Step 4: Create initial holdings
for asset, allocation := range portfolio.Allocation {
  quantity = (portfolio.TargetValue * allocation) / getCurrentPrice(asset)
  holding = {
    Asset: asset,
    Quantity: quantity,
    PurchasePrice: getCurrentPrice(asset),
    PurchaseDate: time.Now()
  }
  portfolio.Holdings.Add(holding)
}
```

**Outcome**:
- ✓ **Created**: Account active, portfolio initialized
- Client can access account via portal

**Temporal Details**:
- Activity Timeout: 10 seconds
- Retry Policy: 2 retries on failure
- Account activation SLA: Same-day

---

### STEP 6: Notify Client & Activate Portal

**Business Process**:
1. Generate welcome email with:
   - Congratulation message
   - Portal login credentials
   - Quick start guide
   - Advisor contact information
2. Activate portal access
3. Create welcome notification in portal
4. Send SMS notification (if opt-in)
5. Schedule welcome call with advisor (auto-calendar)
6. Mark client as "active"

**Welcome Email Template**:

```html
<h1>Welcome to ${advisorFirmName}!</h1>

<p>Dear ${clientFirstName},</p>

<p>Your investment account is now active. Here's what you need to know:</p>

<h2>Your Account Details</h2>
<ul>
  <li><strong>Account Number:</strong> ${accountNumber}</li>
  <li><strong>Portfolio Value:</strong> $${portfolioValue}</li>
  <li><strong>Your Advisor:</strong> ${advisorName} (${advisorEmail})</li>
  <li><strong>AML Screening Status:</strong> ${amlStatus}</li>
</ul>

<h2>Access Your Portal</h2>
<p>Login to your portal: <a href="${portalUrl}">${portalUrl}</a></p>
<p>Username: ${username}</p>

<h2>Next Steps</h2>
<ol>
  <li>Review your portfolio allocation</li>
  <li>Verify your contact information</li>
  <li>Schedule a call with ${advisorName}</li>
  <li>Review compliance documents (AML screening results available)</li>
</ol>

<p>If you have any questions, contact ${advisorName} at ${advisorPhone}</p>
```

**Portal Activation**:

```go
// Update client status
client.Status = "active"
client.PortalAccessDate = time.Now()
client.LastLoginDate = nil // Not yet logged in

// Create portal account
portalAccount = {
  ClientID: client.ID,
  Username: generateUsername(client),
  Email: client.Email,
  TemporaryPassword: generateSecurePassword(),
  LastPasswordChange: time.Now(),
  MFAEnabled: true,
  ExpiryDate: nil  // No expiration
}

// Send credentials securely
sendSecureEmail(client.Email, {
  Subject: "Welcome to Your Investment Portal",
  Template: "welcome_email",
  TemporaryPassword: portalAccount.TemporaryPassword,
  PortalURL: portalURL
})
```

**Outcome**:
- ✓ **Complete**: Client onboarded and active
- Client receives welcome email and can access portal
- Advisor receives notification to schedule first meeting

**Temporal Details**:
- Activity Timeout: 5 seconds
- Client activation SLA: Immediate
- Welcome call scheduling SLA: Within 2 business days

---

## Exception Handling & Escalation Paths

### Exception 1: AML Screening Rejected

```
AML Screening Rejected
  │
  ├─► Stop onboarding workflow
  ├─► Update client status to "onboarding_rejected"
  ├─► Notify client via email (compliant template)
  ├─► Create compliance ticket for review
  ├─► Retain all documentation for audit trail (7 years)
  └─► No further contact (per regulations)
```

### Exception 2: Advisor Assignment Timeout

```
Advisor Assignment Timeout (48h exceeded)
  │
  ├─► Escalate to manager
  ├─► Reassign to available advisor
  ├─► Send escalation notification
  ├─► Extend deadline to 24 hours
  └─► Log for team performance review
```

### Exception 3: E-Signature Not Completed

```
E-Signature Deadline Exceeded (7 days)
  │
  ├─► Send urgent reminder email (day 7)
  ├─► Call client to follow up
  ├─► If unresponsive after 2 weeks:
  │   ├─► Mark onboarding as "pending_signature"
  │   ├─► Create task for advisor follow-up
  │   └─► Re-send agreements
  └─► If 30 days passed → Close application
```

### Exception 4: AML Critical Finding Timeout

```
Manager Review Timeout (24h for critical findings)
  │
  ├─► Auto-escalate to Compliance Director
  ├─► Send escalation notification
  ├─► Create priority ticket
  ├─► Extend deadline to 12 hours
  └─► If still unresolved → Auto-reject application
```

---

## Compliance & Audit Trail

### Events Logged

```
Event Type                  Timestamp    Actor          Details
─────────────────────────────────────────────────────────
client_application_started  [timestamp]  [client]       Application opened
step_1_validation_start     [timestamp]  [system]       KYC validation initiated
step_1_validation_complete  [timestamp]  [system]       KYC data collected
step_2_aml_screening_start  [timestamp]  [compliance]   AML screening initiated
step_2_aml_screening_result [timestamp]  [provider]     Risk score: 45, Level: medium
step_2_aml_approved         [timestamp]  [officer]      Screening approved
step_3_advisor_assigned     [timestamp]  [system]       Advisor: John Smith
step_4_agreements_sent      [timestamp]  [system]       DocuSign request sent
step_4_agreements_signed    [timestamp]  [client]       Client signed agreements
step_5_account_created      [timestamp]  [system]       Account activated
step_6_portal_activated     [timestamp]  [system]       Portal access granted
onboarding_complete         [timestamp]  [system]       Client fully onboarded
```

### Audit Retention

- **All records**: 7 years (regulatory requirement)
- **AML screening results**: 7 years (FinCEN requirement)
- **E-signature records**: 7 years (SEC requirement)
- **Portal access logs**: 3 years minimum

---

## Summary

The updated 6-step Client Onboarding workflow with dedicated AML screening step provides:

✅ **Compliance**: Full AML/KYC/OFAC screening at enterprise-grade  
✅ **Efficiency**: Automated steps, parallel processing  
✅ **Security**: ABAC access control, audit trail  
✅ **Visibility**: Detailed status tracking, escalation workflows  
✅ **Flexibility**: Multi-provider AML support, configurable rules  

**Next Deployment**: See `AML_SCREENING_ENHANCEMENT_GUIDE.md` for implementation details

---

**Document Version**: 2.0 (Updated with AML Screening Step)  
**Last Updated**: October 28, 2025  
**Status**: Ready for Development
