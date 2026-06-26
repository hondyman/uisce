# Client Onboarding - Quick Start Guide

## 5-Minute Setup

### 1. Database

```bash
# Apply schema migration
psql postgres://postgres:postgres@localhost:5432/alpha -f migrations/client_onboarding_schema.sql

# Apply validation rules
psql postgres://postgres:postgres@localhost:5432/alpha -f migrations/client_onboarding_validation_rules.sql

# Verify tables created
psql postgres://postgres:postgres@localhost:5432/alpha -c "\dt+ client*"
```

### 2. Backend Integration

```go
// In backend/cmd/main.go or your main setup
import "github.com/semlayer/backend/internal/api"

func main() {
    // ... existing setup ...
    
    // Register onboarding routes
    api.RegisterClientOnboardingRoutes(router, db)
    
    // ... start server ...
}
```

### 3. Test the API

```bash
# Create a client
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

# Response includes client_id
# Save this for next steps
CLIENT_ID="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"

# Start validation (Step 1)
curl -X POST http://localhost:8080/api/onboarding/step1/validate \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  -H "X-Tenant-Datasource-ID: 11111111-1111-1111-1111-111111111111" \
  -H "X-User-ID: 22222222-2222-2222-2222-222222222222" \
  -H "Content-Type: application/json" \
  -d "{
    \"client_id\": \"$CLIENT_ID\",
    \"verify_kyc\": true,
    \"perform_aml_screening\": true,
    \"aml_provider\": \"lexis_nexis\"
  }"

# Check status
curl -X GET http://localhost:8080/api/onboarding/status/$CLIENT_ID \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000"
```

## File Structure

```
semlayer/
├── migrations/
│   ├── client_onboarding_schema.sql          # Database schema (10 tables)
│   └── client_onboarding_validation_rules.sql # 20 validation rules
│
├── backend/internal/api/
│   ├── client_onboarding_types.go            # All type definitions
│   ├── client_onboarding_service.go          # Database operations (~600 LOC)
│   └── client_onboarding_handlers.go         # REST API handlers (~700 LOC)
│
├── temporal/
│   ├── workflows/
│   │   └── client_onboarding_workflow.go     # Main + escalation workflows (~500 LOC)
│   └── activities/
│       └── client_onboarding_activities.go   # Business logic activities (~400 LOC)
│
└── CLIENT_ONBOARDING_IMPLEMENTATION.md       # Complete documentation
```

## API Endpoints Summary

| Method | Endpoint | Purpose |
|--------|----------|---------|
| POST | `/api/clients` | Create new client |
| GET | `/api/clients` | List clients in onboarding |
| GET | `/api/clients/{id}` | Get single client |
| POST | `/api/onboarding/step1/validate` | Step 1: Validate KYC/AML |
| POST | `/api/onboarding/step2/route` | Step 2: Route to advisor |
| POST | `/api/onboarding/step3/agreements` | Step 3: Send agreements |
| POST | `/api/onboarding/step4/accounts` | Step 4: Create accounts |
| POST | `/api/onboarding/step5/notify` | Step 5: Notify client |
| GET | `/api/onboarding/status/{id}` | Get onboarding progress |
| POST | `/api/onboarding/approve` | Approve onboarding |

## Key Features

✅ **5-Step Workflow**
- Step 1: Validate client data (KYC/AML)
- Step 2: Route for advisor review (48h timeout)
- Step 3: Send agreements for e-signature (7d timeout)
- Step 4: Create accounts & portfolios
- Step 5: Notify client of completion

✅ **Temporal Orchestration**
- Long-running process management
- Built-in timeout & escalation
- Approval signals from humans
- Automatic retries

✅ **Multi-Tenancy**
- Tenant-scoped queries
- Datasource isolation
- User context tracking

✅ **Compliance**
- KYC/AML screening integration
- 20 built-in validation rules
- Audit trail of all events
- ABAC role-based access

✅ **Document Management**
- E-signature integration hooks
- Document verification tracking
- Expiration monitoring

✅ **Risk Management**
- Risk profile-based routing
- High-net-worth due diligence
- Escalation workflows
- Advisor assignment

## Configuration

### Tenant-Specific Rules

Enable/disable rules per tenant:

```sql
UPDATE validation_rules 
SET is_active = FALSE 
WHERE tenant_id = '00000000-0000-0000-0000-000000000000'
  AND rule_name = 'kyc_very_high_net_worth_advisor_review';
```

### Timeout Customization

Modify Temporal workflow timeouts in:
- `temporal/workflows/client_onboarding_workflow.go` (line ~100)

```go
// Current: 48 hours for advisor review
approvalTimeout := 48 * time.Hour

// Change to: 24 hours
// approvalTimeout := 24 * time.Hour
```

### AML Providers

Supported in activities (add as needed):
- LexisNexis
- WorldCheck
- Internal screening
- Custom integrations

## Integration Checkpoints

- [ ] Database migrations applied
- [ ] Backend routes registered
- [ ] Temporal worker running (if using workflows)
- [ ] Tenant context headers present
- [ ] Client creation works
- [ ] Step 1 validation returns workflow_id
- [ ] Status endpoint shows progress
- [ ] All 5 steps callable

## Next Steps

1. **Frontend UI**
   - Advisor dashboard for client review
   - Client self-service onboarding portal
   - Admin monitoring dashboard

2. **External Integrations**
   - DocuSign e-signature API
   - Banking account creation APIs
   - KYC/AML screening providers
   - Email/SMS notification service

3. **Analytics**
   - Onboarding completion rates
   - Average time-to-complete per step
   - Rejection reasons analysis
   - Advisor productivity metrics

4. **Testing**
   - E2E tests for all 5 steps
   - Timeout scenario tests
   - Escalation workflow tests
   - Multi-tenant isolation tests

## Troubleshooting

**Table doesn't exist error**
```bash
psql postgres://user:pass@localhost:5432/alpha -c "SELECT * FROM clients;"
# If missing, re-run migration
psql postgres://user:pass@localhost:5432/alpha -f migrations/client_onboarding_schema.sql
```

**Foreign key constraint error**
```sql
-- Ensure datasources table exists and has matching records
SELECT * FROM datasources;

-- May need to insert test datasource
INSERT INTO datasources (tenant_id, id, source_name, source_type)
VALUES ('00000000-0000-0000-0000-000000000000', 
        '11111111-1111-1111-1111-111111111111',
        'test_datasource',
        'test');
```

**Workflow ID not returned**
- Check that onboarding_workflows table has write permissions
- Verify tenant context headers are correct
- Look for database errors in backend logs

## Performance Tips

1. **Database**
   - Indices created on tenant_id, onboarding_status, workflow_id
   - Use LIMIT/OFFSET for pagination
   - Consider partitioning onboarding_events by created_at for large volumes

2. **Temporal**
   - Task queue persistence configured
   - Workflow history retention set to 7 days
   - Activity timeouts appropriate for external APIs

3. **API**
   - Return pagination tokens for large result sets
   - Cache client status to reduce queries
   - Use database connection pooling

## Security Reminders

- All endpoints require `X-Tenant-ID` header
- All queries scoped to tenant automatically
- Passwords/credentials not stored (use external services)
- E-signature timestamps with IP tracking
- Audit trail for compliance

## Support & Resources

- `CLIENT_ONBOARDING_IMPLEMENTATION.md` - Full technical documentation
- `agents.md` - Tenant scoping and ABAC reference
- Database schema includes detailed comments
- API types have inline documentation

---

**Ready to deploy?** Follow the 5-Minute Setup above and test with curl. Success! 🎉
