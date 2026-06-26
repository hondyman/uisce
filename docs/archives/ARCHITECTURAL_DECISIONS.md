# Architectural Decision Records (ADR)

## ADR-001: Event-Driven Architecture with RabbitMQ

**Date:** 2025-10-18  
**Status:** ACCEPTED ✅  
**Scope:** Business Object event publishing and consumption

### Context
The Northwind Business Objects system requires a way to:
- Publish changes for audit compliance
- Trigger downstream processes (notifications, workflows)
- Enable future microservices decomposition
- Maintain system resilience if consumers are down

### Decision
Use **RabbitMQ** as the central message broker with **topic-based routing** for event distribution.

### Rationale

| Criterion | RabbitMQ | Direct Events | Redis Pub/Sub |
|-----------|----------|---------------|---------------|
| **Durability** | ✅ Persists | ❌ Lost | ❌ Lost |
| **At-Least-Once** | ✅ Guaranteed | ❌ Best effort | ❌ Best effort |
| **Ordering** | ✅ FIFO | ✅ In-memory | ❌ No guarantee |
| **Enterprise** | ✅ Production | ⚠️ Tight coupling | ⚠️ Limited |
| **Microservices** | ✅ Decoupled | ❌ Monolith only | ⚠️ Limited |

### Implementation
- **Event Types:** 8 core (bo.*, instance.*, workflow.*)
- **Exchange:** `semlayer.bo` (topic type)
- **Queues:** Per event type with TTL and DLQ
- **Graceful Degradation:** Silently disable if unavailable

### Consequences
✅ **Pros:**
- True decoupling between services
- Audit trail in message history
- Can replay events for recovery
- Easy to add new consumers

⚠️ **Cons:**
- Additional operational complexity
- Requires running broker service
- Event schema versioning needed
- Consumer lag monitoring required

---

## ADR-002: Monolith with Event Bus (Not Microservices Yet)

**Date:** 2025-10-18  
**Status:** ACCEPTED ✅  
**Scope:** Deployment architecture

### Context
Team must decide between:
1. Build monolith + add microservices later
2. Start with separate microservices from day one
3. Use event bus inside monolith (hybrid approach)

### Decision
Start with **monolith + event bus**, keeping door open for microservices extraction.

### Rationale
- **Pragmatic:** Faster to develop and deploy initially
- **Scalable:** Event bus allows independent scaling later
- **Risk-Managed:** Avoid microservices complexity tax early
- **Proven:** Netflix, Uber all started monolithic

### Architecture
```
Monolithic Backend
    ↓
EventPublisher (in-process)
    ↓
RabbitMQ (external)
    ↓
Future Consumers (can be extracted to separate services)
```

### Migration Path
**Phase 1 (Now):** Monolith publishes events
```
Backend
├── BO Service (CRUD)
├── Event Publisher (to RabbitMQ)
└── All consumers in-process
```

**Phase 2 (Month 2):** Extract Audit Service
```
Backend → RabbitMQ ← Audit Service (separate container)
```

**Phase 3 (Month 3):** Extract Workflow Engine
```
Backend → RabbitMQ ← Workflow Service (separate container)
         ↘ Audit Service
```

### Consequences
✅ **Pros:**
- Low initial complexity
- Single deployment unit
- Shared database resources
- Easy debugging

⚠️ **Cons:**
- Single point of failure (while monolith)
- All services scale together (initially)
- Eventually need decomposition work

---

## ADR-003: Multi-Tenancy at All Layers

**Date:** 2025-10-18  
**Status:** ACCEPTED ✅  
**Scope:** Data isolation and query filtering

### Context
System must support multiple customers (tenants) with:
- Complete data isolation
- No cross-tenant data leakage
- Audit compliance per tenant
- Separate billing/usage tracking

### Decision
**Tenant ID is mandatory** at every layer:
- Database queries (WHERE tenant_id = ?)
- API headers (X-Tenant-ID, X-Tenant-Datasource-ID)
- Event messages (tenant_id field)
- Audit logs (tenant_id column)

### Implementation
```
Frontend (Tenant Selector)
         ↓
HTTP Header (X-Tenant-ID)
         ↓
Handler (Validates header)
         ↓
Service (Filters by tenant)
         ↓
Database (WHERE tenant_id = ?)
```

### Validation Example
```go
// Handler always validates
tenantID := r.Header.Get("X-Tenant-ID")
if tenantID == "" {
    http.Error(w, "Missing X-Tenant-ID", http.StatusBadRequest)
    return
}

// Service filters all queries
instances, err := service.ListInstances(ctx, tenantID, boKey, offset, limit)
// ↑ tenantID is required parameter, not optional
```

### Consequences
✅ **Pros:**
- Cannot accidentally return other tenant's data
- Scalable to thousands of tenants
- Audit trail per tenant
- Compliance-ready (PCI-DSS, HIPAA, GDPR)

⚠️ **Cons:**
- Tenant ID in every query (minor performance)
- Requires discipline across entire codebase
- Cross-tenant reporting needs special handling

---

## ADR-004: Soft Deletes for Instances, Hard Deletes for BOs

**Date:** 2025-10-18  
**Status:** ACCEPTED ✅  
**Scope:** Data deletion strategy

### Context
Should deletion be:
1. Permanent (hard delete)
2. Reversible (soft delete with restore)
3. Different by entity type

### Decision
- **Instances (BO Data):** Soft delete with `is_deleted` flag
- **Business Objects (Definitions):** Hard delete (only by admin)
- **Audit Log:** Never delete (immutable)

### Rationale

**Instances (Soft Delete):**
- Business users may need to "undo" deletions
- Audit requirements demand historical data
- GDPR "right to be forgotten" handled separately
- Supports workflow reversions

**Business Objects (Hard Delete):**
- Rare operation (schema cleanup)
- Admin-only capability
- Instances referencing them are cascade-deleted
- Audit log is preserved

**Audit Log (Never Delete):**
- Compliance requirement
- Proof of who changed what
- Immutable record

### Implementation
```sql
-- Instances: Soft delete
UPDATE bo_instances 
SET is_deleted = true, deleted_at = NOW() 
WHERE id = 'instance-123';

-- BOs: Hard delete (admin only)
DELETE FROM business_objects WHERE id = 'bo-456';

-- Queries: Always filter
SELECT * FROM bo_instances 
WHERE is_deleted = false AND tenant_id = 'tenant-1';
```

### Consequences
✅ **Pros:**
- Business users can recover deleted data
- Audit trail preserved
- GDPR-compliant with proper handling

⚠️ **Cons:**
- Database grows (soft deletes not freed)
- Need periodic archive/purge process
- Queries always need `is_deleted = false` filter

---

## ADR-005: GraphQL as Secondary API (Not Primary)

**Date:** 2025-10-18  
**Status:** ACCEPTED ✅  
**Scope:** API strategy

### Context
Should the system have:
1. Only REST API
2. Only GraphQL
3. Both (REST primary, GraphQL secondary)

### Decision
**Both APIs, with REST as primary** and GraphQL as optional secondary.

### Rationale

**Why REST is Primary:**
- Simpler for CRUD operations
- Already built and tested
- Lower learning curve for new devs
- Works great for instance CRUD

**Why GraphQL is Added:**
- Better for complex queries (search across fields)
- Reduces over-fetching on mobile
- Enables advanced filtering
- Frontend developers prefer it

### Endpoints
```
REST (Primary)
POST   /api/bo/{key}/instances          # Create
GET    /api/bo/{key}/instances          # List with pagination
GET    /api/bo/{key}/instances/{id}     # Get one
PUT    /api/bo/{key}/instances/{id}     # Update
DELETE /api/bo/{key}/instances/{id}     # Delete

GraphQL (Secondary)
POST   /graphql                         # Query/mutation
GET    /graphql/playground              # Dev UI
```

### Example: When to Use Each
```
# Simple CRUD → Use REST
POST /api/bo/customer/instances
  { "coreFields": { "name": "Acme" } }

# Complex search → Use GraphQL
query {
  instances(
    boKey: "customer"
    filter: { field: "name", op: CONTAINS, value: "Acme" }
  ) { ... }
}
```

### Consequences
✅ **Pros:**
- Familiar REST for basic operations
- Powerful GraphQL for complex cases
- Can deprecate REST later if needed

⚠️ **Cons:**
- Two APIs to maintain
- Documentation burden
- Client confusion about which to use

---

## ADR-006: JSON Custom Fields (Not Strict Schema)

**Date:** 2025-10-18  
**Status:** ACCEPTED ✅  
**Scope:** Instance data storage

### Context
How to store instance data:
1. Strict schema per BO (requires migrations)
2. Flexible JSONB with core + custom fields
3. NoSQL for complete flexibility

### Decision
**Hybrid approach:**
- **Core fields:** Columns in database (queryable, indexed)
- **Custom fields:** JSONB column (flexible, no migration)

### Rationale

**Core Fields Example (Customer BO):**
```sql
CREATE TABLE bo_instances (
    id UUID,
    business_object_key TEXT,
    created_by UUID,
    core_field_values JSONB,  -- {name, email, phone, ...}
    custom_field_values JSONB -- {vip_status, credit_limit, ...}
);
```

**Queries:**
```sql
-- Filter on core field
SELECT * FROM bo_instances 
WHERE core_field_values->>'name' = 'Acme Corp';

-- Any custom field
SELECT * FROM bo_instances 
WHERE custom_field_values->>'vip_status' = 'true';
```

### Consequences
✅ **Pros:**
- Core fields are queryable/indexed
- Custom fields for extensibility
- No migrations needed for new fields
- Works with GraphQL and REST

⚠️ **Cons:**
- JSONB queries less performant than columns
- Need indexes on frequently-searched custom fields
- Type validation in application layer

---

## ADR-007: Single PostgreSQL Database (Not Polyglot)

**Date:** 2025-10-18  
**Status:** ACCEPTED ✅  
**Scope:** Database technology

### Context
Should use:
1. Single PostgreSQL database
2. PostgreSQL + Redis
3. Multiple databases (polyglot)

### Decision
**PostgreSQL for all persistence**, with optional Redis caching later.

### Rationale

**Why PostgreSQL:**
- JSONB support (custom fields)
- ACID compliance (audit requirements)
- Advanced indexing (GiST, GIN)
- Range queries (efficient pagination)
- Window functions (analytics)

**Why Not Redis Primary:**
- Requires persistence fallback anyway
- JSONB queries less elegant in Redis
- Team more experienced with SQL

**Caching Layer (Optional Future):**
```
Redis Cache (optional)
    ↓
PostgreSQL (source of truth)
```

### Consequences
✅ **Pros:**
- Simpler operational model
- Single source of truth
- ACID guarantees
- Strong audit trail

⚠️ **Cons:**
- Single database is bottleneck (eventually need sharding)
- Not ideal for time-series data
- May need read replicas at scale

---

## ADR-008: Cloning Duplicates All Fields and Subtypes

**Date:** 2025-10-18  
**Status:** ACCEPTED ✅  
**Scope:** BO cloning behavior

### Context
When cloning a Business Object, should clone:
1. Only the structure (empty fields)
2. With default values
3. Complete copy of all fields and subtypes

### Decision
**Clone everything:** All fields, subtypes, and configurations are copied.

### Rationale

**User Expectations:**
- "Clone customer" means exact copy
- Easier than starting from scratch
- Can edit after cloning

**Implementation:**
```go
// CloneBusinessObject copies:
// - Field definitions (name, type, validation)
// - Subtype definitions (structure)
// - Field ordering
// - But NOT instances (data)
```

### Example
```
Source BO: Customer
├── Fields: name, email, phone, company
└── Subtypes:
    ├── Standard
    ├── VIP
    └── Enterprise

Cloned BO: Customer_VIP
├── Fields: name, email, phone, company (copied)
└── Subtypes:
    ├── Standard (copied)
    ├── VIP (copied)
    └── Enterprise (copied)
```

### Consequences
✅ **Pros:**
- Users get a complete starting point
- Metadata integrity preserved
- Reduces manual setup

⚠️ **Cons:**
- Expensive operation (many inserts)
- Need to verify all relationships copied
- Users must rename afterward

---

## ADR-009: Audit Log Never Purged

**Date:** 2025-10-18  
**Status:** ACCEPTED ✅  
**Scope:** Data retention

### Context
Should audit logs be:
1. Permanent (never deleted)
2. Archived after time period
3. Sampled (every 10th entry)

### Decision
**Permanent audit log** - never automatically purged.

### Rationale

**Compliance:**
- SOX, HIPAA, PCI-DSS require audit trails
- GDPR "right to be forgotten" doesn't override audit
- Regulatory audits expect complete history

**Implementation:**
```
bo_audit_log table
├── NEVER delete rows
├── Archive to cold storage (optional)
└── Immutable once written
```

### Retention Policy
- **Hot Storage:** 2 years (PostgreSQL)
- **Cold Storage:** 7 years (S3 Glacier, optional)
- **Legal Hold:** Override purge if litigation pending

### Consequences
✅ **Pros:**
- Compliance-ready
- Complete audit trail
- Litigation support

⚠️ **Cons:**
- Database grows indefinitely
- Need archival strategy
- Query performance over time

---

## ADR-010: Environment-Specific Configuration

**Date:** 2025-10-18  
**Status:** ACCEPTED ✅  
**Scope:** Configuration management

### Context
How to handle dev/stage/prod differences:
1. Hardcoded values
2. Environment variables
3. Config files + env vars

### Decision
**Environment variables + config.yaml hybrid:**
- Secrets → Environment variables
- Settings → config.yaml
- Overrides → Environment variables

### Configuration Files
```yaml
# config.yaml (dev)
database:
  host: host.docker.internal
  port: 5432
rabbitmq:
  url: amqp://guest:guest@localhost:5672
```

```bash
# .env.production
DATABASE_URL=postgresql://user:pass@prod-db:5432/alpha
KAFKA_BROKERS=broker1:9092,broker2:9092 # production Kafka bootstrap servers
```

### Consequences
✅ **Pros:**
- Secrets not in git
- Easy to change per environment
- Standard 12-factor app pattern

⚠️ **Cons:**
- Multiple config sources to understand
- Need validation on startup
- Ops team must maintain .env files

---

## Summary Table

| ADR | Decision | Status |
|-----|----------|--------|
| 001 | RabbitMQ for events | ✅ ACCEPTED |
| 002 | Monolith + event bus | ✅ ACCEPTED |
| 003 | Multi-tenancy everywhere | ✅ ACCEPTED |
| 004 | Soft deletes for instances | ✅ ACCEPTED |
| 005 | REST primary + GraphQL optional | ✅ ACCEPTED |
| 006 | JSON custom fields | ✅ ACCEPTED |
| 007 | PostgreSQL only | ✅ ACCEPTED |
| 008 | Clone all fields/subtypes | ✅ ACCEPTED |
| 009 | Permanent audit log | ✅ ACCEPTED |
| 010 | Env vars + config files | ✅ ACCEPTED |

---

## Future ADRs (To Be Considered)

- **ADR-011:** GraphQL subscriptions for real-time updates
- **ADR-012:** Elasticsearch for full-text search
- **ADR-013:** Kafka alternative to RabbitMQ
- **ADR-014:** Database sharding strategy
- **ADR-015:** API rate limiting approach
