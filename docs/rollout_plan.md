# Rollout Plan: Hybrid Analytics Stack

## Overview
This document outlines the phased rollout strategy for migrating from RabbitMQ/Postgres to the Redpanda/RisingWave/Iceberg stack.

## Phased Rollout
### 1. Sandbox POC (Current Phase)
- **Environment**: Local Docker Compose.
- **Goal**: Validate connectivity and end-to-end data flow (Postgres -> Debezium -> Redpanda -> RisingWave -> Iceberg -> Trino).
- **Validation**:
  - Insert row in Postgres.
  - Verify pointer event in Redpanda.
  - Verify payload in MinIO.
  - Verify canonical row in RisingWave MV.
  - Verify row queryable in Trino.

### 2. Staging Deployment
- **Infrastructure**: Deploy Redpanda (3-node), RisingWave Cluster, Iceberg (AWS Glue/S3), Trino Cluster.
- **Data**: Seed with anonymized production dump.
- **Testing**:
  - Run `dryrun-diff.js` to compare legacy semantic resolver output vs new stack output for top 100 accounts.
  - Load test Resolver with `k6`.

### 3. Canary Release (5% Traffic)
- **Routing**: Update `FeatureFlag` service to route 5% of `Holdings` semantic term evaluations to new stack.
- **Monitoring**:
  - Watch `SEMANTIC_ANOMALY_DETECTED` rate.
  - Compare latency distributions.

### 4. General Availability (100% Traffic)
- **Migration**: Full switch over.
- **Reconciliation**: Run nightly reconciliation job between Legacy Postgres snapshots and Iceberg tables.

## SLOs & Alerts
| Metric | SLO | Alert Threshold |
| -- | -- | -- |
| CDC Lag | < 30s | > 60s for 5m |
| Resolver Latency (P95) | < 300ms | > 500ms for 5m |
| RisingWave MV Freshness | < 5s | > 30s |
| Anomaly Rate | < 0.1% | > 1% spike |

## Rollback Procedures
### Scenario: High Anomaly Rate or Data Inconsistency
1. **Switch Traffic**: Update Feature Flag to route 100% back to Legacy Resolver/Postgres.
2. **Revert Catalog**: Rolling back `semantic_terms.cue` version if schema change caused issue.
3. **Data Cleanup**:
   - The new stack uses immutable Pointer Events. No data loss on rollback.
   - Truncate daily partition in Iceberg if corrupted.
4. **Communication**: Notify #data-eng and #analytics-users.
