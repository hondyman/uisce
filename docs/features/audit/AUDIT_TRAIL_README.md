# Audit Trail & Compliance System

This document describes the comprehensive audit trail and compliance system implemented for regulatory compliance and security monitoring.

## Overview

The audit trail system provides:
- **Complete audit logging** of all user activities and system events
- **Compliance reporting** for regulatory requirements
- **Real-time monitoring** and alerting
- **Data retention policies** and automated cleanup
- **Export capabilities** for external audits

## Architecture

### Backend Components

#### 1. Audit Models (`internal/models/audit.go`)
- `AuditEvent`: Core audit event structure
- `AuditEventType`: Predefined event types (login, data_access, etc.)
- `AuditEventSeverity`: Severity levels (low, medium, high, critical)
- `AuditSummary`: Summary statistics for reporting
- `ComplianceReport`: Regulatory compliance reports

#### 2. Audit Service (`internal/audit/service.go`)
- `LogEvent()`: Core event logging function
- `LogUserAction()`: User action logging
- `LogDataAccess()`: Data access logging
- `LogDataModification()`: Data change logging
- `QueryEvents()`: Event querying with filters
- `GetAuditSummary()`: Generate audit summaries
- `CleanupOldEvents()`: Automated cleanup based on retention policies

#### 3. Audit Middleware (`internal/audit/middleware.go`)
- `AuditLoggingMiddleware()`: Gin middleware for automatic HTTP request logging
- Event type detection based on request patterns
- Severity assessment based on response codes
- Compliance flag assignment

#### 4. Audit Handlers (`internal/audit/handler.go`)
- REST API endpoints for audit data
- Event querying and filtering
- Export functionality (JSON, CSV)
- Compliance report generation

### Database Schema

#### Core Tables

**audit_events**
- Primary audit log table
- Stores all audit events with full details
- Indexed for efficient querying

**audit_summaries**
- Daily summary statistics
- Aggregated data for reporting

**compliance_reports**
- Generated compliance reports
- Links to audit events and summaries

**audit_retention_policies**
- Configurable retention policies
- Automated cleanup rules

**audit_alerts**
- Alert configuration
- Real-time monitoring rules

**user_sessions**
- Session tracking
- Login/logout events

**data_access_log**
- Detailed data access tracking
- Field-level access logging

## Event Types

### Authentication Events
- `login` / `logout`
- `login_failed`
- `password_reset`

### Data Events
- `data_access` / `data_export`
- `data_modify` / `data_delete`
- `calculation_run` / `model_execute`

### Configuration Events
- `config_change`
- `bundle_create` / `bundle_update` / `bundle_delete`

### Security Events
- `policy_violation`
- `access_denied`
- `compliance_check`

## API Endpoints

### Query Audit Events
```
GET /api/audit/events
```
Query parameters:
- `user_id`: Filter by user
- `event_type`: Filter by event type
- `severity`: Filter by severity
- `start_time` / `end_time`: Time range
- `limit` / `offset`: Pagination

### Get Audit Summary
```
GET /api/audit/summary
```
Returns aggregated statistics for the specified time range.

### Export Audit Events
```
POST /api/audit/export
```
Request body:
```json
{
  "filter": { ... },
  "format": "csv|json",
  "report_name": "optional_name"
}
```

### Compliance Reports
```
GET /api/audit/compliance-reports
POST /api/audit/compliance-reports
```

## Frontend Components

### AuditDashboard (`frontend/src/features/audit/AuditDashboard.tsx`)
- **Overview Tab**: Summary statistics and recent events
- **Audit Events Tab**: Detailed event listing with filtering
- **Compliance Reports Tab**: Report generation and management
- **Real-time Updates**: Live event monitoring

### Key Features
- Advanced filtering by user, event type, severity, time range
- Export functionality for compliance reviews
- Event detail modal with full context
- Responsive design for mobile and desktop

## Setup Instructions

### 1. Database Migration
```bash
# Run the audit schema migration
go run migrate_audit.go
```

### 2. Backend Integration
```go
// Add audit service to your main application
auditService := audit.NewService(db)

// Add audit middleware to Gin router
router.Use(auditMiddleware.AuditLoggingMiddleware())

// Register audit routes
auditHandler := audit.NewHandler(auditService)
auditRoutes := router.Group("/api/audit")
{
    auditRoutes.GET("/events", auditHandler.GetAuditEvents)
    auditRoutes.GET("/summary", auditHandler.GetAuditSummary)
    auditRoutes.POST("/export", auditHandler.ExportAuditEvents)
    // ... other routes
}
```

### 3. Configuration
Update your `config.yaml`:
```yaml
audit:
  enabled: true
  retention_days: 2555  # 7 years for financial data
  cleanup_interval: "24h"
  max_export_records: 10000
```

### 4. Retention Policies
Default retention policies are automatically created:
- Authentication events: 365 days
- Data access events: 2555 days (7 years)
- Configuration changes: 2555 days
- System events: 365 days

## Security Considerations

### Data Protection
- Audit logs are immutable once written
- Sensitive data is masked in logs
- Encryption at rest recommended

### Access Control
- Audit logs accessible only to authorized users
- Role-based access (admin, auditor, compliance_officer)
- All audit log access is itself audited

### Compliance Features
- **SOX Compliance**: Financial transaction auditing
- **GDPR Compliance**: Data access and processing logs
- **PCI DSS**: Payment data handling logs
- **HIPAA**: Healthcare data access logs

## Monitoring & Alerting

### Built-in Alerts
- Multiple failed login attempts
- Policy violations
- Unauthorized data access
- Configuration changes

### Custom Alerts
Configure additional alerts through the audit_alerts table:
```sql
INSERT INTO audit_alerts (name, event_type, severity, conditions)
VALUES ('Suspicious Data Access', 'data_access', 'high',
        '{"unusual_hours": true, "unusual_location": true}');
```

## Performance Considerations

### Indexing Strategy
- Composite indexes on commonly queried fields
- Time-based partitioning for large datasets
- Archive old data to separate tables

### Cleanup Automation
- Automated cleanup based on retention policies
- Archive to compressed storage before deletion
- Configurable cleanup intervals

### Query Optimization
- Use appropriate filters to limit result sets
- Implement pagination for large datasets
- Consider read replicas for audit queries

## Troubleshooting

### Common Issues

**High Database Load**
- Check query patterns and add missing indexes
- Implement query result caching
- Consider audit log archiving

**Missing Audit Events**
- Verify middleware is properly configured
- Check database connectivity
- Review error logs for failed writes

**Performance Issues**
- Monitor database query performance
- Implement audit event buffering
- Consider async logging for high-traffic systems

## Compliance Checklist

- [ ] Audit logging enabled for all user actions
- [ ] Data access logging implemented
- [ ] Authentication events captured
- [ ] Retention policies configured
- [ ] Regular compliance reports generated
- [ ] Audit log integrity verified
- [ ] Access to audit logs restricted
- [ ] Automated alerts configured
- [ ] Backup and recovery procedures documented

## Future Enhancements

- **Real-time Dashboards**: Live audit event streaming
- **Machine Learning**: Anomaly detection in audit patterns
- **Blockchain Integration**: Immutable audit trails
- **Multi-tenant Isolation**: Enhanced tenant data separation
- **Advanced Analytics**: Pattern analysis and trend detection
