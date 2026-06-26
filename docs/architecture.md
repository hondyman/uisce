# System Architecture

## High-Level Overview

The **Autonomous Portfolio Rebalancing Ecosystem** follows a microservices-based architecture (modular monolith in Go) orchestrated by **Temporal**. It leverages a **Unified Lakehouse Architecture** powered by **StarRocks on Iceberg** for both real-time analytics and historical queries against the same data.

```mermaid
graph TD
    subgraph "Clients"
        AdvisorUI[Advisor Dashboard (React)]
        RegulatorPortal[Regulator Portal]
    end

    subgraph "API Gateway (Go)"
        API[REST / GraphQL API]
    end

    subgraph "Core Engines (Go)"
        Rebalancer[Rebalancing Engine]
        Compliance[Compliance Engine]
        Analytics[Analytics Engine]
    end

    subgraph "Orchestration"
        Temporal[Temporal Cluster]
    end

    subgraph "Lakehouse (StarRocks + Iceberg)"
        StarRocks[(StarRocks\nQuery Engine)]
        Nessie[(Nessie\nIceberg Catalog)]
        Iceberg[(Iceberg/Parquet\nUnified Storage)]
        MinIO[(MinIO/S3\nObject Storage)]
    end

    subgraph "Operational Data"
        Postgres[(PostgreSQL)]
    end

    AdvisorUI --> API
    RegulatorPortal --> API
    API --> Temporal
    
    Temporal --> Rebalancer
    Temporal --> Compliance
    Temporal --> Analytics

    Rebalancer --> Postgres
    Compliance --> Postgres
    Analytics --> StarRocks

    Rebalancer -- "Events" --> Iceberg
    Rebalancer -- "Snapshots" --> Iceberg
    StarRocks -- "Queries" --> Nessie
    Nessie -- "Metadata" --> Iceberg
    Iceberg -- "Parquet Files" --> MinIO
```

## Component Details

### 1. Rebalancing Engine
- **Responsibility**: Portfolio optimization, drift detection, trade generation.
- **Tech**: Go, Gonum (Math), OSQP (Solver).
- **Key Workflows**: `PortfolioLifecycleWorkflow`.

### 2. Compliance Engine ("Glass Box")
- **Responsibility**: Policy enforcement, audit logging, PII redaction.
- **Tech**: Go, Regex, LLM (Guardrails).
- **Key Features**: Immutable Event Log, Deterministic Replay.

### 3. Analytics Engine
- **Responsibility**: Performance calculation (TWR/MWR), Factor Analysis.
- **Tech**: StarRocks (Aggregations via Iceberg), Go (Regression).
- **Key Features**: Log-Sum-Exp aggregation, Rolling Beta, Multi-tenant resource groups.

### 4. Unified Lakehouse Architecture
- **Query Engine**: StarRocks OSS for sub-second analytics on Iceberg tables.
- **Catalog**: Nessie for Git-like versioned Iceberg metadata management.
- **Storage**: Apache Iceberg (Parquet) for immutable, time-travel capable data.
- **Multi-Tenancy**: Date-based partitioning + tenant_id bucketing, resource groups per tenant tier.
