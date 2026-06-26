# Calendar Service API Specification

## Overview

This is the REST API for the Epic 31 Holiday & Calendar Intelligence system.

**Base URL**: `http://localhost:8081` (local) or `https://calendar.example.com` (production)

**Authentication**: `X-Hasura-Tenant-Id` header (tenant isolation)

**Content-Type**: `application/json`

---

## Health & Status

### GET /health

Check if the service is healthy.

```bash
curl http://localhost:8081/health
```

**Response** (200 OK):
```json
{
  "status": "healthy",
  "timestamp": "2026-03-01T10:00:00Z"
}
```

### GET /metrics

Prometheus metrics for monitoring.

```bash
curl http://localhost:8081/metrics
```

**Response** (200 OK):
```
# HELP calendar_service_calendars_total Total number of calendars
# TYPE calendar_service_calendars_total gauge
calendar_service_calendars_total{tenant="550e8400-e29b-41d4-a716-446655440000"} 5

# HELP calendar_service_availability_checks_total Total availability checks
# TYPE calendar_service_availability_checks_total counter
calendar_service_availability_checks_total 1234
```

---

## Calendars API

### GET /api/v1/calendars

List all calendars for the tenant.

```bash
curl -X GET "http://localhost:8081/api/v1/calendars?limit=10&offset=0" \
  -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000"
```

**Query Parameters**:
- `limit` (optional, default=10, max=1000): Number of results
- `offset` (optional, default=0): Pagination offset
- `region` (optional): Filter by region (e.g., "US", "EMEA", "APAC")
- `sort` (optional): Sort field ("created_at", "name"; prefix `-` for desc)

**Response** (200 OK):
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440001",
      "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "USA Federal Holidays",
      "region": "US",
      "holidays": [
        {
          "date": "2026-01-01",
          "name": "New Year",
          "severity": "HIGH"
        },
        {
          "date": "2026-07-04",
          "name": "Independence Day",
          "severity": "HIGH"
        }
      ],
      "valid_from": "2026-01-01T00:00:00Z",
      "valid_to": null,
      "created_at": "2026-01-01T10:00:00Z",
      "updated_at": "2026-01-01T10:00:00Z"
    }
  ],
  "total": 1,
  "limit": 10,
  "offset": 0
}
```

**Error Response** (401 Unauthorized):
```json
{
  "error": "Missing X-Hasura-Tenant-Id header"
}
```

---

### GET /api/v1/calendars/:id

Fetch a specific calendar.

```bash
curl -X GET http://localhost:8081/api/v1/calendars/550e8400-e29b-41d4-a716-446655440001 \
  -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000"
```

**Response** (200 OK):
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440001",
    "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "USA Federal Holidays",
    "region": "US",
    "holidays": [...],
    "valid_from": "2026-01-01T00:00:00Z",
    "valid_to": null,
    "created_at": "2026-01-01T10:00:00Z",
    "updated_at": "2026-01-01T10:00:00Z"
  }
}
```

**Error Response** (404 Not Found):
```json
{
  "error": "Calendar not found"
}
```

---

### GET /api/v1/calendars/:id/history

Get all versions of a calendar (bitemporal history).

```bash
curl -X GET http://localhost:8081/api/v1/calendars/550e8400-e29b-41d4-a716-446655440001/history \
  -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000"
```

**Response** (200 OK):
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440001",
      "valid_from": "2026-02-01T00:00:00Z",
      "valid_to": null,
      "holidays": [
        {"date": "2026-03-15", "name": "Added Holiday", "severity": "MEDIUM"}
      ]
    },
    {
      "id": "550e8400-e29b-41d4-a716-446655440001",
      "valid_from": "2026-01-01T00:00:00Z",
      "valid_to": "2026-02-01T00:00:00Z",
      "holidays": [...]
    }
  ]
}
```

---

### POST /api/v1/calendars

Create a new calendar.

```bash
curl -X POST http://localhost:8081/api/v1/calendars \
  -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Asia-Pacific Holidays",
    "region": "APAC",
    "holidays": [
      {"date": "2026-02-10", "name": "Lunar New Year", "severity": "HIGH"},
      {"date": "2026-10-01", "name": "National Day", "severity": "HIGH"}
    ]
  }'
```

**Request Body**:
```json
{
  "name": "string (required, max 255)",
  "region": "string (optional, e.g., 'US', 'EMEA')",
  "holidays": "array of holiday objects (required)"
}
```

**Holiday Object Schema**:
```json
{
  "date": "YYYY-MM-DD (required)",
  "name": "string (required)",
  "severity": "HIGH | MEDIUM | LOW (optional, default=MEDIUM)"
}
```

**Response** (201 Created):
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440002",
    "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "Asia-Pacific Holidays",
    "region": "APAC",
    "holidays": [...],
    "valid_from": "2026-03-01T10:00:00Z",
    "valid_to": null,
    "created_at": "2026-03-01T10:00:00Z",
    "updated_at": "2026-03-01T10:00:00Z"
  }
}
```

**Error Response** (400 Bad Request):
```json
{
  "error": "Invalid holiday date format. Expected YYYY-MM-DD"
}
```

---

### PATCH /api/v1/calendars/:id

Update a calendar (bitemporal versioning).

```bash
curl -X PATCH http://localhost:8081/api/v1/calendars/550e8400-e29b-41d4-a716-446655440001 \
  -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000" \
  -H "Content-Type: application/json" \
  -d '{
    "holidays": [
      {"date": "2026-01-01", "name": "New Year", "severity": "HIGH"},
      {"date": "2026-03-17", "name": "St. Patricks Day (NEW)", "severity": "MEDIUM"}
    ]
  }'
```

**Request Body** (all fields optional):
```json
{
  "name": "string (optional)",
  "region": "string (optional)",
  "holidays": "array (optional)"
}
```

**Response** (200 OK):
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440001",
    "name": "USA Federal Holidays",
    "region": "US",
    "holidays": [...],
    "valid_from": "2026-03-01T11:00:00Z",
    "valid_to": null,
    "updated_at": "2026-03-01T11:00:00Z"
  }
}
```

---

### DELETE /api/v1/calendars/:id

Soft-delete a calendar (sets `valid_to = now()`).

```bash
curl -X DELETE http://localhost:8081/api/v1/calendars/550e8400-e29b-41d4-a716-446655440001 \
  -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000"
```

**Response** (204 No Content):
```
(empty body)
```

**Note**: The calendar is soft-deleted (marked with `valid_to`), not actually deleted.

---

## Availability API

### POST /api/v1/check-availability

Check if a job can run at a given time.

```bash
curl -X POST http://localhost:8081/api/v1/check-availability \
  -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000" \
  -H "Content-Type: application/json" \
  -d '{
    "profile_name": "default",
    "start_time": "2026-03-01T10:00:00Z",
    "end_time": "2026-03-01T11:00:00Z"
  }'
```

**Request Body**:
```json
{
  "profile_name": "string (required, e.g., 'default')",
  "start_time": "RFC3339 timestamp (required)",
  "end_time": "RFC3339 timestamp (required)"
}
```

**Response** (200 OK - Available):
```json
{
  "available": true,
  "reasons": [],
  "checked_at": "2026-03-01T10:00:00Z"
}
```

**Response** (200 OK - NOT Available):
```json
{
  "available": false,
  "reasons": [
    "Holiday: New Year (2026-01-01)",
    "Blackout: Maintenance Window (2026-03-01 02:00-04:00)"
  ],
  "checked_at": "2026-03-01T10:00:00Z"
}
```

---

### POST /api/v1/find-available-slot

Find the next available time slot for a job.

```bash
curl -X POST http://localhost:8081/api/v1/find-available-slot \
  -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000" \
  -H "Content-Type: application/json" \
  -d '{
    "profile_name": "default",
    "duration_minutes": 60,
    "start_after": "2026-03-01T10:00:00Z",
    "max_days_ahead": 7
  }'
```

**Request Body**:
```json
{
  "profile_name": "string (required)",
  "duration_minutes": "integer (required, job duration)",
  "start_after": "RFC3339 timestamp (required, search from this time)",
  "max_days_ahead": "integer (optional, default=30, limit search window)"
}
```

**Response** (200 OK):
```json
{
  "available": true,
  "suggested_slot": {
    "start_time": "2026-03-02T10:00:00Z",
    "end_time": "2026-03-02T11:00:00Z",
    "reason": "Next available after holidays"
  }
}
```

**Response** (200 OK - No availability):
```json
{
  "available": false,
  "suggested_slot": null,
  "reason": "No available slots in the next 30 days"
}
```

---

## Schedule Profiles API

### GET /api/v1/profiles

List all schedule profiles for the tenant.

```bash
curl -X GET http://localhost:8081/api/v1/profiles \
  -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000"
```

**Response** (200 OK):
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440003",
      "name": "default",
      "description": "Default schedule profile",
      "timezone": "UTC",
      "conflict_resolution": "UNION",
      "calendars": [
        {
          "id": "550e8400-e29b-41d4-a716-446655440001",
          "name": "USA Federal Holidays",
          "weight": 100
        }
      ]
    }
  ]
}
```

---

### POST /api/v1/profiles

Create a new schedule profile.

```bash
curl -X POST http://localhost:8081/api/v1/profiles \
  -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "europe",
    "description": "European schedule profile",
    "timezone": "Europe/London",
    "conflict_resolution": "UNION",
    "calendar_ids": [
      "550e8400-e29b-41d4-a716-446655440001",
      "550e8400-e29b-41d4-a716-446655440002"
    ]
  }'
```

**Request Body**:
```json
{
  "name": "string (required, unique per tenant)",
  "description": "string (optional)",
  "timezone": "string (required, e.g., 'Europe/London')",
  "conflict_resolution": "UNION | INTERSECTION | PRIORITY (required)",
  "calendar_ids": "array of UUID (required)"
}
```

**Response** (201 Created):
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440004",
    "name": "europe",
    "timezone": "Europe/London",
    "conflict_resolution": "UNION",
    "calendars": [...]
  }
}
```

---

## Audit Log API

### GET /api/v1/audit-log

Fetch audit log entries for compliance/debugging.

```bash
curl -X GET "http://localhost:8081/api/v1/audit-log?entity_type=CALENDAR&limit=50" \
  -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000"
```

**Query Parameters**:
- `entity_type` (optional): CALENDAR, PROFILE, BLACKOUT, etc.
- `action` (optional): CREATE, UPDATE, DELETE
- `start_date` (optional): Filter from this date (RFC3339)
- `end_date` (optional): Filter until this date (RFC3339)
- `limit` (optional, default=100, max=1000)
- `offset` (optional, default=0)

**Response** (200 OK):
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440005",
      "entity_type": "CALENDAR",
      "entity_id": "550e8400-e29b-41d4-a716-446655440001",
      "action": "UPDATE",
      "old_values": {
        "holidays": [{"date": "2026-01-01", "name": "New Year"}]
      },
      "new_values": {
        "holidays": [
          {"date": "2026-01-01", "name": "New Year"},
          {"date": "2026-12-25", "name": "Christmas"}
        ]
      },
      "changed_by": "user-uuid",
      "changed_at": "2026-03-01T10:00:00Z"
    }
  ],
  "total": 1
}
```

---

## Error Handling

### Standard Error Response

All errors follow this format:

```json
{
  "error": "Descriptive error message",
  "code": "ERROR_CODE",
  "details": {}
}
```

### HTTP Status Codes

| Status | Meaning |
|--------|---------|
| 200 | OK |
| 201 | Created |
| 204 | No Content |
| 400 | Bad Request (invalid input) |
| 401 | Unauthorized (missing tenant header) |
| 403 | Forbidden (insufficient permissions) |
| 404 | Not Found |
| 500 | Internal Server Error |

---

## Examples

### Example 1: Create Calendar & Check Availability

```bash
# 1. Create a calendar
CALENDAR_ID=$(curl -X POST http://localhost:8081/api/v1/calendars \
  -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Q1 Blackouts",
    "region": "US",
    "holidays": [{"date": "2026-03-17", "name": "St. Patricks", "severity": "LOW"}]
  }' | jq -r '.data.id')

echo "Created calendar: $CALENDAR_ID"

# 2. Check availability for March 17 (should be false)
curl -X POST http://localhost:8081/api/v1/check-availability \
  -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000" \
  -H "Content-Type: application/json" \
  -d '{
    "profile_name": "default",
    "start_time": "2026-03-17T10:00:00Z",
    "end_time": "2026-03-17T11:00:00Z"
  }' | jq .

# Expected: available = false, reasons include "St. Patricks"
```

### Example 2: List with Pagination

```bash
# List calendars with pagination
curl "http://localhost:8081/api/v1/calendars?limit=5&offset=0&sort=-created_at" \
  -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000" | jq .
```

### Example 3: Audit Trail

```bash
# Get all changes to a specific calendar
curl "http://localhost:8081/api/v1/audit-log?entity_type=CALENDAR&entity_id=550e8400-e29b-41d4-a716-446655440001" \
  -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000" | jq .
```

---

## Next Steps

- Implement handlers based on these specs (Phase 1)
- Add React client library for frontend integration
- Create Postman collection for API testing
- Document SDKs (Node.js, Python, etc.)
