# Compliance Engine Microservice

The Compliance Engine is a real-time compliance monitoring and workflow ABAC (Attribute-Based Access Control) service that integrates with Redpanda (Kafka) for event-driven processing (legacy RabbitMQ support may remain) and Temporal for workflow orchestration.

## Features

### 🔍 Real-Time Compliance Monitoring
- **Event Processing**: Listens to compliance events via RabbitMQ queues
- **Workflow Compliance Checks**: Validates workflow executions against compliance rules
- **Audit Logging**: Comprehensive audit trail of all compliance decisions
- **Violation Detection**: Automatic detection and reporting of compliance violations

### 🛡️ Workflow ABAC Engine
- **Policy-Based Access Control**: Fine-grained permissions for workflow operations
- **Risk-Based Evaluation**: Considers risk levels in access decisions
- **Time-Based Restrictions**: Business hours and time window controls
- **Approval Workflows**: Multi-level approval requirements for high-risk operations
- **Context-Aware Decisions**: Considers user roles, environment, and resource context

### 📊 Monitoring & Metrics
- **Health Checks**: Service health monitoring endpoints
- **Prometheus Metrics**: Integration with monitoring systems
- **Event Statistics**: Real-time compliance event metrics
- **Performance Monitoring**: Queue lengths and processing times

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   RabbitMQ      │    │ Compliance      │    │   Temporal      │
│   Queues        │◄──►│   Engine        │◄──►│   Workflows     │
│                 │    │                 │    │                 │
│ • compliance.   │    │ • Event         │    │ • Approval      │
│   events        │    │   Processing    │    │   Workflows     │
│ • workflow.     │    │ • ABAC          │    │                 │
│   compliance.   │    │   Evaluation    │    │                 │
│   checks        │    │ • Audit         │    └─────────────────┘
│ • abac.audit    │    │   Logging       │
│ • temporal.     │    │                 │
│   workflow.     │    └─────────────────┘
│   events        │            │
└─────────────────┘            ▼
                               ┌─────────────────┐
                               │   PostgreSQL    │
                               │   Database      │
                               │                 │
                               │ • Compliance    │
                               │   Events        │
                               │ • Workflow      │
                               │   Checks        │
                               │ • ABAC Policies │
                               └─────────────────┘
```

## API Endpoints

### Health & Monitoring
- `GET /health` - Service health check
- `GET /metrics` - Prometheus metrics endpoint

### Workflow ABAC
- `POST /workflow-abac/evaluate` - Evaluate workflow access permissions

### Request/Response Examples

#### Workflow ABAC Evaluation
```json
POST /workflow-abac/evaluate
{
  "subject": "user-123",
  "action": "execute",
  "resource": "investment-workflow-456",
  "workflow_type": "investment",
  "risk_assessment": {
    "level": "high",
    "amount": 1000000
  },
  "context": {
    "user_roles": ["portfolio_manager"],
    "department": "wealth_management"
  },
  "tenant_id": "00000000-0000-0000-0000-000000000000",
  "datasource_id": "11111111-1111-1111-1111-111111111111"
}
```

Response:
```json
{
  "allowed": true,
  "reason": "Access granted"
}
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | `100.84.126.19` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `postgres` | Database user |
| `DB_PASSWORD` | `postgres` | Database password |
| `DB_NAME` | `alpha` | Database name |
| `RABBITMQ_HOST` | `localhost` | RabbitMQ host |
| `RABBITMQ_PORT` | `5672` | RabbitMQ port |
| `RABBITMQ_USER` | `guest` | RabbitMQ user |
| `RABBITMQ_PASSWORD` | `guest` | RabbitMQ password |
| `TEMPORAL_HOST` | - | Temporal server host:port |
| `HTTP_PORT` | `8082` | HTTP server port |
| `DEMO_TENANT_ID` | `00000000-0000-0000-0000-000000000000` | Demo tenant ID |
| `DEMO_DATASOURCE_ID` | `11111111-1111-1111-1111-111111111111` | Demo datasource ID |

## Default Workflow Policies

The service initializes with default ABAC policies for common workflow types:

### Investment Workflows
- **Action**: `create`
- **Roles**: `advisor`, `portfolio_manager`
- **Risk Level**: `medium`
- **Time Restrictions**: Business hours only
- **Approval Required**: No

### Compliance Workflows
- **Action**: `execute`
- **Roles**: `compliance_officer`, `senior_compliance`
- **Risk Level**: `high`
- **Approval Required**: Yes (requires `senior_compliance` or `chief_compliance_officer`)

### Onboarding Workflows
- **Action**: `modify`
- **Roles**: `client_services`, `relationship_manager`
- **Risk Level**: `low`
- **Approval Required**: No

## Event topics (Redpanda/Kafka)

The service listens to the following queues:

- `compliance.events` - General compliance monitoring events
- `workflow.compliance.checks` - Workflow compliance validation requests
- `abac.audit` - ABAC decision audit events
- `temporal.workflow.events` - Temporal workflow state changes

## Database Schema

### Compliance Events
```sql
CREATE TABLE compliance_events (
    id UUID PRIMARY KEY,
    event_type VARCHAR(255),
    resource VARCHAR(255),
    action VARCHAR(255),
    user_id VARCHAR(255),
    tenant_id VARCHAR(255),
    datasource_id VARCHAR(255),
    timestamp TIMESTAMP,
    details JSONB,
    severity VARCHAR(50),
    status VARCHAR(50),
    abac_context JSONB
);
```

### Workflow Compliance Checks
```sql
CREATE TABLE workflow_compliance_checks (
    id UUID PRIMARY KEY,
    workflow_id VARCHAR(255),
    check_type VARCHAR(255),
    status VARCHAR(255),
    checked_at TIMESTAMP,
    compliance_data JSONB,
    violations JSONB
);
```

### Workflow ABAC Policies
```sql
CREATE TABLE workflow_abac_policies (
    id UUID PRIMARY KEY,
    tenant_id VARCHAR(255),
    datasource_id VARCHAR(255),
    workflow_type VARCHAR(255),
    action VARCHAR(255),
    resource_pattern VARCHAR(255),
    subject_rules JSONB,
    environment_rules JSONB,
    risk_level VARCHAR(50),
    requires_approval BOOLEAN,
    approval_roles JSONB,
    time_restrictions JSONB,
    enabled BOOLEAN,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

## Development

### Building
```bash
go mod tidy
go build -o compliance-engine .
```

### Running
```bash
./compliance-engine
```

### Testing
```bash
go test ./...
```

## Integration

The Compliance Engine integrates with:

- **AI Builder Service**: Receives workflow suggestions for compliance validation
- **Temporal Workflows**: Monitors workflow execution and triggers compliance checks
- **Frontend Applications**: Provides ABAC evaluation API for UI components
- **Monitoring Systems**: Exports metrics for Prometheus/Grafana dashboards

## Security Considerations

- All operations are tenant-scoped using `X-Tenant-ID` and `X-Tenant-Datasource-ID` headers
- ABAC policies prevent unauthorized workflow operations
- Audit logging captures all access decisions for compliance reporting
- Risk-based evaluation prevents high-risk operations without proper approvals
- Time-based restrictions limit access to business hours for sensitive operations