# Permanent Port Allocation Standard

This document defines the PERMANENT port allocation for all services. These ports are never to be changed to avoid conflicts and confusion.

## Core Backend Services (8000-8099)

| Service | Port | Protocol | Purpose | Config File |
|---------|------|----------|---------|-------------|
| Backend API | **8080** | HTTP/REST | Main backend API server | docker-compose.dev.simple.yml |
| Fabric Builder | **8081** | HTTP/REST | Fabric model building service | docker-compose.dev.simple.yml |
| API Gateway | 8001 | HTTP/REST | Legacy API gateway (deprecated) | infrastructure/docker/docker-compose.yml |

## GraphQL & Data Services (8200-8299)

| Service | Port | Protocol | Purpose | Config File |
|---------|------|----------|---------|-------------|
| Hasura GraphQL | **8888** | HTTP/GraphQL | GraphQL engine with admin panel | docker-compose.dev.simple.yml |
| Hasura Internal | 8080 | HTTP/Internal | Hasura internal port (containerized) | docker-compose.dev.simple.yml |

## Message Queue & Async (5600-5700)

| Service | Port | Protocol | Purpose | Config File |
|---------|------|----------|---------|-------------|
| RabbitMQ AMQP | **5672** | AMQP | Message queue protocol | docker-compose.dev.simple.yml |
| RabbitMQ Management | **15672** | HTTP | RabbitMQ admin UI | docker-compose.dev.simple.yml |

## Workflow Engine (7200-7300)

| Service | Port | Protocol | Purpose | Config File |
|---------|------|----------|---------|-------------|
| Temporal Server | **7233** | gRPC | Temporal workflow engine | docker-compose.dev.simple.yml |
| Temporal UI | **8088** | HTTP | Temporal admin UI | docker-compose.dev.simple.yml |

## Frontend (5000-5200)

| Service | Port | Protocol | Purpose | Config File |
|---------|------|----------|---------|-------------|
| Frontend Dev Server | **5173** | HTTP | Vite dev server (run locally, not Docker) | vite.config.ts |

## Database (5400-5500)

| Service | Port | Protocol | Purpose | Notes |
|---------|------|----------|---------|-------|
| PostgreSQL | **5432** | TCP | Database server | Runs on HOST (localhost), NOT in Docker |

## Observability (9000-9100)

| Service | Port | Protocol | Purpose | Config File |
|---------|------|----------|---------|-------------|
| Prometheus | 9090 | HTTP | Metrics collection | infrastructure/docker/docker-compose.observability.yml |
| Grafana | 3000 | HTTP | Metrics visualization | infrastructure/docker/docker-compose.observability.yml |

## Development Tools (8900-8999)

| Service | Port | Protocol | Purpose | Config File |
|---------|------|----------|---------|-------------|
| Adminer | 8099 | HTTP | Database admin UI | infrastructure/docker/docker-compose.yml |

---

## Environment Variable Mapping

These PERMANENT ports must be reflected in all environment variable files:

### Frontend (.env & .env.local)

```bash
# REST API
VITE_API_BASE_URL=http://localhost:8080
VITE_BACKEND_TARGET=http://localhost:8080

# GraphQL
VITE_GRAPHQL_ENDPOINT=http://localhost:8888/v1/graphql
VITE_GRAPHQL_WS_ENDPOINT=ws://localhost:8888/v1/graphql
```

### Backend Services (docker-compose.dev.simple.yml)

```yaml
environment:
  - HASURA_ENDPOINT=http://hasura:8080           # Internal docker DNS
  - TEMPORAL_HOSTPORT=temporal:7233              # Internal docker DNS
  - RABBIT_URL=amqp://guest:guest@rabbitmq:5672  # Internal docker DNS
```

---

## Network Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     HOST MACHINE (macOS)                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  Frontend (Vite)                 PostgreSQL                      │
│  localhost:5173                  localhost:5432                  │
│       │                                ▲                         │
│       │                                │                         │
│       └─────────────────┬──────────────┘                         │
│                         │                                         │
│                    localhost:8080                                │
│                    (Docker bridge)                               │
│                         │                                         │
├─────────────────────────┼─────────────────────────────────────┤
│                DOCKER CONTAINERS (semlayer network)             │
│                         │                                       │
│                         ▼                                        │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ Backend API (semlayer-backend)                           │  │
│  │ Port: 8080                                               │  │
│  │ Health: http://localhost:8080/health                    │  │
│  └──────────────────────────────────────────────────────────┘  │
│                         │                                       │
│         ┌───────────────┼───────────────┐                       │
│         ▼               ▼               ▼                       │
│  ┌─────────────┐ ┌──────────────┐ ┌─────────────┐             │
│  │ Hasura      │ │ RabbitMQ     │ │ Temporal    │             │
│  │ (8888→8080) │ │ (5672)       │ │ (7233)      │             │
│  │ GraphQL     │ │ Message Q    │ │ Workflows   │             │
│  │             │ │              │ │             │             │
│  │ Admin:      │ │ Mgmt: 15672  │ │ UI: 8088    │             │
│  │ 8888        │ │              │ │             │             │
│  └─────────────┘ └──────────────┘ └─────────────┘             │
│         │               ▲               ▲                       │
│         └───────────────┴───────────────┘                       │
│                         │                                       │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ Fabric Builder (semlayer-fabric-builder)                 │  │
│  │ Port: 8081                                               │  │
│  │ Health: http://localhost:8081/health                    │  │
│  └──────────────────────────────────────────────────────────┘  │
│                         │                                       │
│                         └─→ host.docker.internal:5432           │
│                             (PostgreSQL on host)                │
└─────────────────────────────────────────────────────────────────┘
```

---

## Verification Checklist

- [x] All ports are unique (no overlap)
- [x] Ranges are logically grouped
- [x] Internal docker networking uses container DNS
- [x] External access uses localhost:PORT
- [x] PostgreSQL is on HOST (not Docker)
- [x] Environment variables match these ports
- [x] No conflicting services

## Quick Reference

| For... | Use Port | Example |
|--------|----------|---------|
| REST API calls from browser | **8080** | http://localhost:8080/api/business-entities |
| GraphQL queries from frontend | **8888** | http://localhost:8888/v1/graphql |
| RabbitMQ messages | **5672** | amqp://localhost:5672 |
| Temporal workflows | **7233** | localhost:7233 |
| RabbitMQ Management UI | **15672** | http://localhost:15672 |
| Temporal UI | **8088** | http://localhost:8088 |
| PostgreSQL (host only) | **5432** | psql postgres://postgres@localhost:5432/alpha |

---

## If You Need to Add a Service

1. Choose a port from the appropriate range (don't reuse)
2. Update this document
3. Update docker-compose.dev.simple.yml
4. Update frontend .env/.env.local if applicable
5. Update backend service environment variables if applicable
6. Commit all changes together

---

**Last Updated:** November 12, 2025  
**Status:** PERMANENT - DO NOT CHANGE WITHOUT TEAM CONSENSUS
