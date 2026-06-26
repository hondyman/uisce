# Phase 5: Advanced Features - Google Calendar & Outlook Integration

**Status**: 🚀 **INITIATED**  
**Date Started**: February 18, 2026  
**Scope**: Advanced calendar integrations, timezone handling, and multi-region deployment

## Phase 5 Overview

### Objectives
1. **Google Calendar API Integration** - Sync holidays and events
2. **Outlook/Office 365 Integration** - Multi-calendar support
3. **Advanced RRULE Patterns** - Complex recurrence handling
4. **Timezone Management** - International business hours
5. **Multi-Region Deployment** - Geographic redundancy

### Success Criteria
- Google Calendar sync working with < 500ms latency
- Outlook integration supporting 10K+ concurrent calendars
- Advanced RRULE patterns (monthly, yearly, complex rules)
- Timezone-aware availability checking
- Multi-region failover capability

## 1. Google Calendar Integration

### 1.1 Architecture

```
┌──────────────────────────────────────────────────┐
│  Google Calendar API                             │
│  - OAuth2 authentication                         │
│  - Calendar list retrieval                       │
│  - Event sync (holidays, busy blocks)            │
└────────────────────┬─────────────────────────────┘
                     │
                ┌────▼─────────────┐
                │  Google Client   │
                │  - Token mgmt    │
                │  - Event fetch   │
                │  - Sync control  │
                └────┬─────────────┘
                     │
        ┌────────────┴────────────┐
        │                         │
   ┌────▼──────┐          ┌──────▼──────┐
   │  Cache    │          │  Database   │
   │  (Redis)  │          │  (Postgres) │
   └───────────┘          └─────────────┘
```

### 1.2 Implementation Plan

#### Phase 5.1a: Google OAuth2 Client
**Files**:
- `internal/oauth/google_provider.go` (new)
- `internal/oauth/token_manager.go` (new)

**Key Features**:
```go
// OAuth2 token management
type GoogleOAuth2Provider struct {
    clientID     string
    clientSecret string
    redirectURL  string
    tokenStore   TokenStore
}

// Token refresh and validation
func (p *GoogleOAuth2Provider) RefreshToken(ctx context.Context, refreshToken string) (*oauth2.Token, error)
func (p *GoogleOAuth2Provider) ValidateToken(token *oauth2.Token) error
```

#### Phase 5.1b: Google Calendar Client
**Files**:
- `internal/google/calendar_client.go` (new)
- `internal/google/calendar_sync.go` (new)

**Key Features**:
```go
// Google Calendar API integration
type GoogleCalendarClient struct {
    service    *calendar.Service
    rateLimiter RateLimiter
    cache      CacheClient
}

// Fetch calendars for user
func (c *GoogleCalendarClient) ListCalendars(ctx context.Context, userID string) ([]Calendar, error)

// Sync events for date range
func (c *GoogleCalendarClient) FetchEventsForRange(
    ctx context.Context,
    calendarID string,
    startDate, endDate time.Time,
) ([]Event, error)

// Monitor calendar changes (via push notifications)
func (c *GoogleCalendarClient) SubscribeToCalendarChanges(
    ctx context.Context,
    tenantID, calendarID string,
) error
```

#### Phase 5.1c: Sync Process
**Files**:
- `internal/sync/google_sync_processor.go` (new)

**Sync Flow**:
```
1. Get user's Google calendars
2. For each calendar:
   - Fetch events (last 90 days + next 90 days)
   - Filter for business hours / holidays
   - Store in Redis cache (1 hour TTL)
   - Update Postgres timestamp
3. Merge with existing calendars
4. Invalidate related cache entries
5. Record sync metrics
```

### 1.3 External Dependencies
- `google.golang.org/api/calendar/v3` - Google Calendar API
- `golang.org/x/oauth2` - OAuth2 authorization
- `golang.org/x/oauth2/google` - Google auth provider

### 1.4 Configuration
```yaml
google:
  client_id: "${GOOGLE_CLIENT_ID}"
  client_secret: "${GOOGLE_CLIENT_SECRET}"
  redirect_url: "${GOOGLE_REDIRECT_URL}"
  sync_interval: 3600  # seconds
  event_lookback: 90   # days
  event_lookahead: 90  # days
```

## 2. Outlook/Office 365 Integration

### 2.1 Architecture

```
┌──────────────────────────────────────────────────┐
│  Microsoft Graph API                             │
│  - OAuth2 (Azure AD)                             │
│  - Calendar retrieval                            │
│  - Event sync with webhooks                      │
└────────────────────┬─────────────────────────────┘
                     │
                ┌────▼──────────────┐
                │  Outlook Client   │
                │  - Token mgmt     │
                │  - Event fetch    │
                │  - Webhook mgmt   │
                └────┬──────────────┘
                     │
        ┌────────────┴────────────┐
        │                         │
   ┌────▼──────┐          ┌──────▼──────┐
   │  Cache    │          │  Database   │
   │  (Redis)  │          │  (Postgres) │
   └───────────┘          └─────────────┘
```

### 2.2 Implementation Plan

#### Phase 5.2a: Azure AD OAuth2 Client
**Files**:
- `internal/oauth/azure_provider.go` (new)

**Key Features**:
```go
type AzureOAuth2Provider struct {
    tenantID     string
    clientID     string
    clientSecret string
    tokenStore   TokenStore
}

// Azure-specific token flow
func (p *AzureOAuth2Provider) GetTokenWithScopes(
    ctx context.Context,
    grantType string,
    scopes []string,
) (*oauth2.Token, error)
```

#### Phase 5.2b: Microsoft Graph Calendar Client
**Files**:
- `internal/microsoft/calendar_client.go` (new)
- `internal/microsoft/calendar_sync.go` (new)

**Key Features**:
```go
type OutlookCalendarClient struct {
    graphClient *msgraphcore.GraphRequestAdapter
    rateLimiter RateLimiter
    webhookMgr  WebhookManager
}

// Fetch calendars from Outlook
func (c *OutlookCalendarClient) ListCalendars(ctx context.Context) ([]Calendar, error)

// Fetch calendar events
func (c *OutlookCalendarClient) FetchEvents(
    ctx context.Context,
    calendarID string,
    startDate, endDate time.Time,
) ([]Event, error)

// Register webhook for real-time updates
func (c *OutlookCalendarClient) RegisterWebhook(
    ctx context.Context,
    calendarID, webhookURL string,
) (string, error)
```

#### Phase 5.2c: Webhook Handler
**Files**:
- `internal/sync/outlook_webhook_handler.go` (new)

**Features**:
- Receive calendar change notifications
- Validate webhook signatures
- Trigger cache invalidation
- Update database

### 2.3 External Dependencies
- `github.com/microsoftgraph/msgraph-sdk-go` - Microsoft Graph SDK
- `golang.org/x/oauth2/clientcredentials` - Service principal auth

### 2.4 Configuration
```yaml
azure:
  tenant_id: "${AZURE_TENANT_ID}"
  client_id: "${AZURE_CLIENT_ID}"
  client_secret: "${AZURE_CLIENT_SECRET}"
  sync_interval: 3600
```

## 3. Advanced RRULE Patterns

### 3.1 Complex Recurrence Examples

```
// Monthly on specific weekday
FREQ=MONTHLY;BYDAY=2MO  # 2nd Monday of month

// Quarterly on last business day
FREQ=QUARTERLY;BYDAY=MO,TU,WE,TH,FR;BYSETPOS=-1

// Bi-weekly Monday & Thursday
FREQ=WEEKLY;INTERVAL=2;BYDAY=MO,TH

// Yearly on Easter (calculated)
FREQ=YEARLY;BYEASTER

// Complex: Every 6 weeks, specific months only
FREQ=WEEKLY;INTERVAL=6;BYMONTH=1,3,5,7,9,11
```

### 3.2 Implementation

**Files**:
- `internal/expansion/advanced_rrule.go` (new)
- `internal/expansion/rrule_calculator.go` (new)

**Features**:
```go
type RRuleCalculator struct {
    baseRule     *rrule.RRule
    timezone     *time.Location
    businessHours *BusinessHoursConfig
}

// Advanced expansion with business hours awareness
func (rc *RRuleCalculator) ExpandWithBusinessHours(
    start, end time.Time,
    includeWeekends bool,
) ([]time.Time, error)

// Holiday-aware expansion (skip holidays)
func (rc *RRuleCalculator) ExpandSkippingHolidays(
    start, end time.Time,
    holidays []time.Time,
) ([]time.Time, error)

// Intersection with business calendar
func (rc *RRuleCalculator) IntersectWithCalendar(
    events []Event,
) ([]time.Time, error)
```

### 3.3 Performance Optimization

**Caching Strategy**:
```
RRULE Expansion Cache:
- Key: hash(rrule + start + end)
- TTL: 7 days
- Hit rate target: > 95%
```

**Precomputation**:
```
// Pre-expand common patterns for 1-year window
func (rc *RRuleCalculator) PrecomputeNextYear(ctx context.Context) error
```

## 4. Timezone Management

### 4.1 Architecture

```
┌────────────────────────────────────────────┐
│  Timezone Context                          │
│  - User timezone                           │
│  - Calendar timezone                       │
│  - Business timezone                       │
└────────────────┬───────────────────────────┘
                 │
        ┌────────▼────────┐
        │ Timezone        │
        │ Converter       │
        │ Service         │
        └────────┬────────┘
                 │
    ┌────────────┼────────────┐
    │            │            │
┌───▼────┐  ┌────▼────┐  ┌────▼────┐
│Business│  │Calendar │  │  UTC    │
│Hours   │  │Local    │  │         │
└────────┘  └─────────┘  └─────────┘
```

### 4.2 Implementation

**Files**:
- `internal/timezone/converter.go` (new)
- `internal/timezone/business_hours.go` (new)

**Features**:
```go
type TimezoneConverter struct {
    cache TimezoneCache
}

// Convert availability to user's timezone
func (tc *TimezoneConverter) ConvertToUserTZ(
    availability Availability,
    userTZ string,
) (UserAvailability, error)

// Check if time is in business hours (timezone-aware)
func (tc *TimezoneConverter) IsBusinessHours(
    t time.Time,
    tzName string,
    config BusinessHoursConfig,
) bool

// Find common working hours across multiple timezones
func (tc *TimezoneConverter) FindCommonWorkingHours(
    timezones []string,
    configs []BusinessHoursConfig,
) ([]TimeRange, error)
```

### 4.3 Business Hours Configuration

```go
type BusinessHoursConfig struct {
    Timezone    string      // "America/New_York"
    StartTime   time.Time   // 09:00 (daily)
    EndTime     time.Time   // 17:00 (daily)
    WorkDays    []time.Weekday // Mon-Fri
    Holidays    []Holiday   // Easter, Thanksgiving, etc.
}
```

## 5. Multi-Region Deployment

### 5.1 Architecture

```
┌─────────────────────────────────────────────────────┐
│  Global Load Balancer (GeoDNS)                      │
└────────────┬──────────────┬───────────────┬─────────┘
             │              │               │
        ┌────▼────┐     ┌────▼────┐    ┌────▼────┐
        │ US East │     │ Europe  │    │ APAC    │
        │ Region  │     │ Region  │    │ Region  │
        └────┬────┘     └────┬────┘    └────┬────┘
             │              │               │
        ┌────▼────────┐ ┌────▼────────┐ ┌──▼─────────┐
        │PostgreSQL   │ │PostgreSQL   │ │PostgreSQL  │
        │(Primary)    │ │(Primary)    │ │(Primary)   │
        └─────────────┘ └─────────────┘ └────────────┘
             │              │               │
        ┌────▼────┐     ┌────▼────┐    ┌────▼────┐
        │Redis    │     │Redis    │    │Redis    │
        │Cache    │     │Cache    │    │Cache    │
        └─────────┘     └─────────┘    └─────────┘
```

### 5.2 Deployment Model

**Regional Services**:
```
US East:
  - Primary PostgreSQL (100.84.126.19:5432)
  - Redis cache (localhost:6379)
  - Calendar Service (port 9081)
  - Load: ~40% of traffic

Europe:
  - Primary PostgreSQL (replication slave)
  - Redis cache (read-replica)
  - Calendar Service (port 9081)
  - Load: ~35% of traffic

APAC:
  - Primary PostgreSQL (replication slave)
  - Redis cache (read-replica)
  - Calendar Service (port 9081)
  - Load: ~25% of traffic
```

### 5.3 Failover Handling

**Files**:
- `internal/failover/leader_election.go` (new)
- `internal/failover/replication_monitor.go` (new)

**Strategy**:
```go
type FailoverManager struct {
    regions    []Region
    monitor    ReplicationMonitor
    consensus  ConsensusDriver // etcd/Consul
}

// Automatic failover detection
func (fm *FailoverManager) MonitorReplication(ctx context.Context) error

// Route traffic to healthy regions
func (fm *FailoverManager) DetermineHealthiestRegion() Region

// Synchronize state across regions
func (fm *FailoverManager) SyncState(ctx context.Context) error
```

## 6. Implementation Timeline

### Week 1: Google Calendar Integration
- [ ] OAuth2 provider implementation
- [ ] Google Calendar API client
- [ ] Sync processor
- [ ] Integration tests

### Week 2: Outlook Integration
- [ ] Azure AD OAuth2 provider
- [ ] Microsoft Graph client
- [ ] Webhook handler
- [ ] Integration tests

### Week 3: Advanced Features
- [ ] Advanced RRULE expansion
- [ ] Business hours calculator
- [ ] Timezone converter
- [ ] Performance optimization

### Week 4: Multi-Region
- [ ] Regional deployment strategy
- [ ] Failover mechanism
- [ ] State synchronization
- [ ] Load testing across regions

## 7. Testing Strategy

### Unit Tests
- OAuth2 token refresh logic
- RRULE expansion edge cases
- Timezone conversion accuracy
- Cache invalidation logic

### Integration Tests
- End-to-end Google Calendar sync
- End-to-end Outlook sync
- Multi-region failover
- Cache coherence across regions

### Load Tests
```bash
# Test concurrent calendar sync
wrk -c 1000 -t 12 -d 60s \
  -s scripts/phase5-load-test.lua \
  http://calendar-service:9081/api/v1/availability

# Test geographic failover
wrk -c 100 -t 4 -d 300s \
  --latency \
  http://us-east.calendar-service/api/v1/availability
```

## 8. Monitoring & Observability

### New Metrics (Phase 5)
```
// Google Calendar Integration
calendar_google_sync_duration_seconds
calendar_google_sync_errors_total
calendar_google_api_rate_limit_remaining

// Outlook Integration
calendar_outlook_sync_duration_seconds
calendar_outlook_sync_errors_total
calendar_outlook_webhook_lag_seconds

// Timezone Processing
calendar_timezone_conversion_duration_seconds
calendar_timezone_cache_hit_rate

// Multi-Region
calendar_region_failover_total
calendar_replication_lag_seconds
calendar_cross_region_sync_latency_seconds
```

### Dashboards
- Google Calendar sync health
- Outlook webhook status
- Timezone conversion performance
- Regional failover status
- Cache effectiveness per region

## 9. Security Considerations

### OAuth2
- Secure token storage (encrypted Redis)
- Token refresh automation
- Scope validation
- Revocation handling

### API Keys
- Environment variable management
- Key rotation procedures
- Rate limiting per client
- API key audit logging

### Data Privacy
- GDPR compliance for EU users
- Data residency requirements
- Encryption at rest/in transit
- Access control logging

## 10. Phase 5 Success Criteria

- [ ] Google Calendar sync with < 500ms latency
- [ ] Outlook integration supporting 10K concurrent calendars
- [ ] Advanced RRULE patterns working with 95%+ accuracy
- [ ] Timezone handling with < 100ms conversion time
- [ ] Multi-region deployment with < 1s failover time
- [ ] All tests passing (unit + integration + load)
- [ ] Zero data loss during failover
- [ ] 99.95% uptime across regions

---

**Next Step**: Begin Phase 5.1 - Google Calendar OAuth2 Implementation
