# SemLayer Steward Playbooks

## Overview
This document provides comprehensive playbooks for SemLayer stewards to manage governance, access control, and system performance.

## Table of Contents
1. [Access Request Triage](#access-request-triage)
2. [Policy Change Management](#policy-change-management)
3. [Bundle Optimization Review](#bundle-optimization-review)
4. [Incident Response](#incident-response)
5. [Performance Monitoring](#performance-monitoring)

---

## Access Request Triage

### Process Overview
Access requests are triaged through automated checks and manual review based on risk assessment.

### Inputs
- **User**: Identity and role information
- **Asset**: Data asset being requested
- **Permission**: Type of access requested (read, write, admin)
- **Justification**: Business reason for access
- **Policy Simulation**: Current policy evaluation result

### Automated Checks
1. **Separation of Duties (SoD) Conflicts**
   - Check for conflicting roles/permissions
   - Validate against SoD matrix
   - Flag potential conflicts for review

2. **Tenant Isolation**
   - Verify request doesn't cross tenant boundaries
   - Check tenant membership and data classification
   - Ensure proper multi-tenancy controls

3. **Certification Sensitivity**
   - Assess data sensitivity level
   - Check certification requirements
   - Apply appropriate approval workflows

4. **Usage History**
   - Review user's access patterns
   - Check for anomalous request patterns
   - Validate against behavioral baselines

### Decision Outcomes

#### ✅ Approve
- **Criteria**: Low-risk request, all checks pass, standard business justification
- **Actions**:
  - Grant immediate access
  - Log approval with rationale
  - Send confirmation to requestor
  - Schedule periodic review

#### 🔄 Modify Scope
- **Criteria**: Request too broad, partial approval possible
- **Actions**:
  - Reduce permission scope
  - Apply time-limited access
  - Suggest alternative data sources
  - Require additional justification

#### ⚡ Assign Micro-bundle/JIT
- **Criteria**: Time-sensitive need, temporary access required
- **Actions**:
  - Create just-in-time access bundle
  - Set automatic expiration
  - Monitor usage patterns
  - Require post-use review

#### ❌ Reject
- **Criteria**: High-risk request, policy violation, insufficient justification
- **Actions**:
  - Provide detailed rejection reason
  - Suggest alternative approaches
  - Escalate to senior steward if needed
  - Log for pattern analysis

### SLA Commitments
- **Standard Requests**: ≤ 1 business day
- **JIT Requests**: ≤ 15 minutes
- **Emergency Requests**: Immediate review

---

## Policy Change Management

### Process Overview
Policy changes require simulation, impact analysis, and controlled rollout.

### Pre-Change Activities

#### 1. Impact Analysis
```bash
# Run policy simulation
curl -X POST /api/v1/policy/simulate \
  -H "Content-Type: application/json" \
  -d '{
    "policy_change": "...",
    "simulation_scope": "tenant_a",
    "impact_analysis": true
  }'
```

#### 2. Risk Assessment
- **Low Risk**: Cosmetic changes, documentation updates
- **Medium Risk**: Permission adjustments, new rules
- **High Risk**: Breaking changes, security policy modifications

#### 3. Rollout Planning
- **Canary Scope**: Start with 10% of users
- **Time Windows**: Business hours for low-risk, maintenance windows for high-risk
- **Rollback Plan**: Documented rollback procedures

### Change Execution

#### Phase 1: Validation
```bash
# Validate policy syntax
semlayer-cli policy validate --file=new_policy.yaml

# Run automated tests
semlayer-cli policy test --policy=new_policy.yaml --test-suite=governance
```

#### Phase 2: Deployment
```bash
# Deploy to canary
semlayer-cli policy deploy --file=new_policy.yaml --scope=canary --percentage=10

# Monitor for 1 hour
watch -n 60 semlayer-cli monitor --scope=canary
```

#### Phase 3: Full Rollout
```bash
# Gradual rollout
semlayer-cli policy deploy --file=new_policy.yaml --strategy=gradual --duration=2h

# Monitor throughout rollout
semlayer-cli monitor --policy-rollout --alert-threshold=0.05
```

### Post-Change Activities

#### Documentation
- Update change log with rationale
- Document risk assessment and mitigation
- Record stakeholder approvals

#### Monitoring
- Monitor error rates and performance
- Track user feedback and issues
- Validate policy effectiveness

---

## Bundle Optimization Review

### Weekly Review Process

#### 1. Queue Analysis
```bash
# Get optimization recommendations
semlayer-cli bundle review-queue --sort-by=impact

# Analyze high-impact items
semlayer-cli bundle analyze --id=<bundle_id> --impact-analysis
```

#### 2. Review Criteria

##### Auto-Approval (Low Risk)
- **Certified Assets Only**: Changes affect only certified data
- **Read-Only Operations**: No write permissions modified
- **Usage-Based**: Recommendations based on actual usage patterns
- **Performance Impact**: < 5% performance change

##### Manual Review (Medium Risk)
- **Mixed Assets**: Includes both certified and uncertified data
- **Permission Changes**: Modifications to access levels
- **Cross-Tenant**: Affects multiple tenants
- **High Usage**: Frequently accessed bundles

##### Escalation Required (High Risk)
- **Security Impact**: Changes affect security controls
- **Compliance**: Impacts regulatory compliance
- **Breaking Changes**: May break existing functionality

#### 3. Approval Workflow

##### For Auto-Approval
```bash
# Apply optimization automatically
semlayer-cli bundle optimize --id=<bundle_id> --auto-approve

# Verify application
semlayer-cli bundle verify --id=<bundle_id>
```

##### For Manual Review
```bash
# Review optimization details
semlayer-cli bundle review --id=<bundle_id> --show-impact

# Apply with approval
semlayer-cli bundle optimize --id=<bundle_id> --approved-by=<steward_id>
```

### Success Metrics
- **Auto-Approval Rate**: Target > 70%
- **Review Cycle Time**: Target < 2 hours
- **Performance Improvement**: Target > 10% query performance gain

---

## Incident Response

### Incident Classification

#### Critical Incidents
- **Data Exposure**: Unauthorized data access
- **System Compromise**: Security breach detected
- **Cross-Tenant Breach**: Tenant isolation failure

#### High Priority
- **SoD Violation**: Separation of duties breach
- **Mass Access Denial**: Widespread access issues
- **Performance Degradation**: > 50% performance drop

#### Medium Priority
- **Single User Impact**: Individual user access issues
- **Policy Conflicts**: Policy rule conflicts
- **Configuration Errors**: Misconfigurations

### Response Procedures

#### Phase 1: Detection & Assessment (0-15 minutes)
```bash
# Check system status
semlayer-cli status --detailed

# Review recent alerts
semlayer-cli alerts --last=1h --severity=high

# Assess impact scope
semlayer-cli impact-analysis --incident=<incident_id>
```

#### Phase 2: Containment (15-60 minutes)
```bash
# Freeze affected claims
semlayer-cli claims freeze --scope=<affected_scope> --reason=incident

# Notify stakeholders
semlayer-cli notify --stakeholders --incident=<incident_id> --priority=high

# Isolate affected systems
semlayer-cli isolate --scope=<affected_scope>
```

#### Phase 3: Investigation (1-4 hours)
```bash
# Run lineage analysis
semlayer-cli lineage analyze --scope=<affected_scope> --depth=3

# Review audit logs
semlayer-cli audit search --time-range=24h --scope=<affected_scope>

# Identify root cause
semlayer-cli forensics analyze --incident=<incident_id>
```

#### Phase 4: Recovery (4-24 hours)
```bash
# Implement fix
semlayer-cli fix apply --incident=<incident_id> --fix=<fix_id>

# Gradual restoration
semlayer-cli restore --scope=<affected_scope> --strategy=gradual

# Validate restoration
semlayer-cli validate --scope=<affected_scope>
```

#### Phase 5: Post-Mortem (24-72 hours)
```bash
# Document incident
semlayer-cli incident document --id=<incident_id> --findings=<findings>

# Update playbooks
semlayer-cli playbook update --incident=<incident_id>

# Stakeholder communication
semlayer-cli report generate --incident=<incident_id> --audience=executives
```

### Communication Templates

#### Stakeholder Notification
```
Subject: SemLayer Incident - {SEVERITY} - {INCIDENT_ID}

Dear {STAKEHOLDER_NAME},

An incident has been detected in the SemLayer platform:

- Incident ID: {INCIDENT_ID}
- Severity: {SEVERITY}
- Affected Scope: {SCOPE}
- Current Status: {STATUS}
- ETA Resolution: {ETA}

Immediate Actions Taken:
- {ACTION_1}
- {ACTION_2}
- {ACTION_3}

Next Steps:
- {NEXT_STEP_1}
- {NEXT_STEP_2}

Contact: {ON_CALL_STEWARD}
```

---

## Performance Monitoring

### Key Metrics Dashboard

#### RED Metrics (Rate, Errors, Duration)
```bash
# Request rate monitoring
semlayer-cli metrics query --name=request_rate --period=5m

# Error rate monitoring
semlayer-cli metrics query --name=error_rate --period=5m

# Duration percentiles
semlayer-cli metrics query --name=request_duration --percentile=95 --period=5m
```

#### USE Metrics (Utilization, Saturation, Errors)
```bash
# Cache utilization
semlayer-cli metrics query --name=cache_utilization --period=1m

# Database connections
semlayer-cli metrics query --name=db_connections --period=1m

# Memory utilization
semlayer-cli metrics query --name=memory_utilization --period=1m
```

### Alert Configuration

#### Performance Alerts
```yaml
alerts:
  - name: high_error_rate
    condition: error_rate > 0.05
    duration: 5m
    severity: critical
    channels: [slack, email, pager]

  - name: high_latency
    condition: request_duration_p95 > 1000ms
    duration: 10m
    severity: warning
    channels: [slack]

  - name: cache_saturation
    condition: cache_utilization > 0.9
    duration: 5m
    severity: warning
    channels: [slack]
```

#### Governance Alerts
```yaml
alerts:
  - name: sod_violation
    condition: sod_violations > 0
    duration: 1m
    severity: critical
    channels: [slack, email, pager]

  - name: policy_drift
    condition: policy_drift_rate > 0.1
    duration: 1h
    severity: warning
    channels: [slack]
```

### Troubleshooting Guides

#### High Latency Issues
1. **Check Cache Performance**
   ```bash
   semlayer-cli cache stats --detailed
   semlayer-cli cache analyze --bottlenecks
   ```

2. **Database Performance**
   ```bash
   semlayer-cli db analyze --slow-queries
   semlayer-cli db optimize --recommendations
   ```

3. **Resource Utilization**
   ```bash
   semlayer-cli system resources --detailed
   semlayer-cli system bottlenecks --identify
   ```

#### Memory Issues
1. **Profile Memory Usage**
   ```bash
   ./scripts/capture-profiles.sh --type=heap --duration=30s
   ```

2. **Analyze Goroutine Leaks**
   ```bash
   ./scripts/capture-profiles.sh --type=goroutine
   ```

3. **Cache Optimization**
   ```bash
   semlayer-cli cache optimize --memory-pressure
   ```

This playbook should be reviewed and updated quarterly based on incident learnings and system evolution.
