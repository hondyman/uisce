# Phase 5e: Microservice Extraction Plan

**Status:** IN-PROGRESS  
**Phase:** 5e (Extraction of modular services from Phase 5d)  
**Objective:** Create dedicated microservice containers for validation, rules, notifications, policies, and search

---

## 🎯 Microservices to Extract

### Service 1: **Validation Service** (Port 8082)
- Runs `ValidationHandler` from Phase 5d
- Handles synchronous and asynchronous BP validations
- Integrates with BPValidationCoordinator
- HTTP endpoints for validation execution and result polling

### Service 2: **Rule Engine Service** (Port 8083)
- Runs `ValidationRuleEngine` from Phase 5b
- Stores and evaluates validation rules
- Executes 13-operator condition evaluation
- Provides rule CRUD and template management

### Service 3: **Notifications Service** (Port 8084)
- Handles email, Slack, webhook delivery
- Consumes validation events from RabbitMQ
- Retries with exponential backoff
- Tracks delivery status and audit trail

### Service 4: **Policy Service** (Port 8085)
- Manages compliance and governance policies
- Policy enforcement at business process steps
- Audit trail recording for policy violations
- Policy template management

### Service 5: **Search Service** (Port 8086)
- Elasticsearch integration for full-text search
- Index validation results, audit logs, policies
- Real-time indexing via RabbitMQ events
- Advanced search and filtering

---

## 📊 Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    Frontend (React)                         │
└────────┬────────────────────────────────────────────────────┘
         │
┌────────┴────────────────────────────────────────────────────┐
│                 API Gateway (8001)                          │
│            (Request routing & auth)                         │
└────────┬────────────────────────────────────────────────────┘
         │
    ┌────┴────────────────────┬───────────────────────┐
    │                         │                       │
    v                         v                       v
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   Backend    │    │     Event    │    │  GraphQL     │
│   (8080)     │    │    Router    │    │  Engine      │
│              │    │   (8081)     │    │  (8083)      │
└──────┬───────┘    └──────┬───────┘    └──────────────┘
       │                   │
       │          ┌────────┴────────┐
       │          │                 │
    ┌──┴──────────┴──────────────┐  │
    │      RabbitMQ (5672)       │  │
    │   (Command Bus + Events)   │  │
    └──────┬──────────────────────┘  │
           │                         │
    ┌──────┴────────┬────────────┬──┴──────┬──────────┐
    │               │            │         │          │
    v               v            v         v          v
┌─────────┐  ┌──────────┐  ┌──────┐  ┌────────┐  ┌────────┐
│Validation│  │ Rule    │  │Policy│  │ Search │  │Notif.  │
│Service   │  │ Engine  │  │Svc   │  │ Svc    │  │ Svc    │
│(8082)    │  │(8083)   │  │(8085)│  │(8086)  │  │(8084)  │
└─────────┘  └──────────┘  └──────┘  └────────┘  └────────┘
    │            │           │         │          │
    └────────────┴───────────┴─────────┴──────────┘
              (All connected via RabbitMQ)
```

---

## 📝 Implementation Plan

### Phase 5e.1: Validation Service
**Steps:**
1. Create `/backend/cmd/validation-service/main.go`
2. Create `/backend/cmd/validation-service/Dockerfile`
3. Implement HTTP server with ValidationHandler endpoints
4. Add health check endpoint
5. Add docker-compose entry

### Phase 5e.2: Rule Engine Service
**Steps:**
1. Create `/backend/cmd/rule-engine-service/main.go`
2. Create `/backend/cmd/rule-engine-service/Dockerfile`
3. Implement HTTP server with RuleEngine endpoints
4. Add rule CRUD operations
5. Add docker-compose entry

### Phase 5e.3: Notifications Service
**Steps:**
1. Create `/backend/cmd/notifications-service/main.go`
2. Create `/backend/cmd/notifications-service/Dockerfile`
3. Implement RabbitMQ event consumer
4. Add email/Slack/webhook handlers
5. Add docker-compose entry

### Phase 5e.4: Policy Service
**Steps:**
1. Create `/backend/cmd/policy-service/main.go`
2. Create `/backend/cmd/policy-service/Dockerfile`
3. Implement HTTP server with Policy CRUD
4. Add policy enforcement logic
5. Add docker-compose entry

### Phase 5e.5: Search Service
**Steps:**
1. Create `/backend/cmd/search-service/main.go`
2. Create `/backend/cmd/search-service/Dockerfile`
3. Implement Elasticsearch integration
4. Add RabbitMQ event consumer for indexing
5. Add docker-compose entry

### Phase 5e.6: Updated docker-compose.yml
**Steps:**
1. Add all 5 new service entries
2. Set up inter-service dependencies
3. Configure RabbitMQ connections
4. Add health checks
5. Document port mappings

---

## 🏗️ Service Structure Template

Each microservice follows this structure:

```
backend/cmd/{service-name}/
├── main.go              (HTTP server + routing)
├── handlers.go          (HTTP endpoint handlers)
├── Dockerfile           (Multi-stage build)
└── .env.example         (Configuration template)
```

### Common HTTP Endpoints

**Health Check:**
```
GET /{service}/health
Response: {"status": "healthy", "timestamp": "2025-10-18T..."}
```

**Metrics:**
```
GET /{service}/metrics
Response: Prometheus-format metrics
```

---

## 🔌 Inter-Service Communication

### Via RabbitMQ (Event-driven)
- **Validation Events** → Notifications Service
- **Policy Violations** → Search Service + Audit Logging
- **Rule Changes** → Rule Engine Service broadcasts updates

### Via HTTP (Synchronous)
- Backend calls Validation Service for sync BP validations
- API Gateway calls services for specific operations
- Services query each other for dependencies

### Service Discovery
- All services registered in docker-compose DNS
- Service name = hostname (e.g., `validation-service:8082`)

---

## 📦 Docker Compose Additions

Each service will be added as:

```yaml
validation-service:
  build:
    context: ./backend/cmd/validation-service
    dockerfile: Dockerfile
  container_name: semlayer-validation-service
  restart: always
  environment:
    - PORT=8082
    - KAFKA_BROKERS=redpanda:9092
    - DATABASE_URL=${DATABASE_URL_DOCKER}
  ports:
    - "8082:8082"
  depends_on:
    - rabbitmq
    - backend
  healthcheck:
    test: ["CMD", "curl", "-f", "http://localhost:8082/health"]
    interval: 10s
    timeout: 5s
    retries: 3
```

---

## ✅ Completion Criteria

- [ ] Validation Service created and running (8082)
- [ ] Rule Engine Service created and running (8083)
- [ ] Notifications Service created and running (8084)
- [ ] Policy Service created and running (8085)
- [ ] Search Service created and running (8086)
- [ ] docker-compose.yml updated with all 5 services
- [ ] All services have health checks
- [ ] All services compile with 0 errors
- [ ] RabbitMQ inter-service communication working
- [ ] HTTP service-to-service calls working
- [ ] Deployment documentation complete

---

## 🔄 Next After Phase 5e

**Phase 6: Full Microservices Architecture**
- Service mesh (Istio/Consul)
- Distributed tracing (Jaeger)
- Load balancing across services
- Auto-scaling policies
- Advanced governance

---

## Starting with Phase 5e.1

Beginning Validation Service extraction now...
