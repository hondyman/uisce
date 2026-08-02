# Uisce Semantic OS — G-SIFI Compliance Alignment (RC1)

This document describes how the RC1 production deployment aligns with G-SIFI (Global Systemically Important Financial Institution) security and operational requirements.

---

## 1. Access Control

### 1.1 Role-Based Access Control (RBAC)

The Kubernetes `ServiceAccount` (`uisce-sa`) is provisioned without any default cluster roles. All access is governed by:
- `NetworkPolicy` restricting ingress to nginx ingress controller + namespace-local
- Pod-level `securityContext` with `runAsNonRoot: true`
- No `wildcard` permissions in RBAC manifests (none defined in RC1 Helm chart)

**Gap (Post-RC1):** Define `Role`/`RoleBinding` resources in the Helm chart for operator vs. viewer access levels.

### 1.2 Secrets Management

| Channel | Mechanism | Status |
|---------|-----------|--------|
| Infisical → K8s Secret | `InfisicalSecret` CRD or `kubectl create secret` | RC1 (manual) |
| K8s Secret → Pod env | `secretKeyRef` in Deployment | RC1 (wired) |
| Raft WAL encryption | AES key via Infisical `RAFT_ENCRYPTION_KEY` | RC1 (wired) |
| Postgres TLS | `?sslmode=disable` in compose; `?sslmode=require` for production | RC1 (env-var driven) |

---

## 2. Data Residency & Isolation

### 2.1 Tenant Isolation

PostgreSQL row-level security (RLS) is used for multi-tenant data isolation. The `pg_trgm` extension enables fuzzy trigram matching for field name resolution in the `SelfHealingService`.

**Migration:** `backend/db/migrations/20260731_pg_trgm.up.sql` initializes the extension.

### 2.2 Audit Ledger (Raft)

The SHA-256 hash chain audit ledger is replicated across three regions via HashiCorp Raft consensus (TCP 7946). Each node stores WAL in a `StatefulSet` PVC (10 GiB, `standard` storage class).

| Property | Setting |
|----------|---------|
| Raft consensus port | TCP 7946 |
| WAL storage | 10 GiB EBS `ReadWriteOnce` |
| Encryption at rest | AES key from Infisical (`RAFT_ENCRYPTION_KEY`) |
| Leadership election | Automatic Raft leader election |
| Snapshot persistence | `replicatorSnapshot.Persist()` → JSON over `raft.SnapshotSink` |

### 2.3 Data Gravity

| Tier | Storage | Engine |
|------|---------|--------|
| Control plane (tenants, ABAC, rules) | PostgreSQL 15 | ACID OLTP |
| Hot operational analytics | StarRocks FE (all-in-one) | Columnar MPP |
| Cold historical lakehouse | Apache Iceberg | S3 + Trino |

---

## 3. Network Security

### 3.1 Kubernetes NetworkPolicy

The `networkPolicy` template in `deploy/helm/templates/networkpolicy.yaml` enforces:

```yaml
ingress:
  - from:
      - namespaceSelector: { name: uisce }
      - podSelector: { app.kubernetes.io/name: ingress-nginx }
egress:
  - to: [{ podSelector: {} }]
    ports: [53, 5432, 6379, 9092]
```

### 3.2 Pod Security Context (G-SIFI Baseline)

| Field | Value |
|-------|-------|
| `runAsNonRoot` | `true` |
| `runAsUser` | `1000` |
| `fsGroup` | `1000` |
| `seccompProfile.type` | `RuntimeDefault` |
| `capabilities.drop` | `[ALL]` |

### 3.3 Container Security Context (G-SIFI Baseline)

| Field | Value |
|-------|-------|
| `allowPrivilegeEscalation` | `false` |
| `readOnlyRootFilesystem` | `true` |
| `capabilities.drop` | `[ALL]` |

**Note:** `readOnlyRootFilesystem: true` requires emptyDir volumes for `/tmp`, `/var/lib/uisce/raft`, `/var/log/uisce`. These are declared in `values.yaml` under `extraVolumes`.

---

## 4. Availability

### 4.1 Pod Disruption Budget

```yaml
spec:
  minAvailable: 2   # At least 2 replicas during node drain
```

### 4.2 Horizontal Pod Autoscaler

| Setting | Value |
|---------|-------|
| Min replicas | 3 |
| Max replicas | 12 |
| Scale-up CPU threshold | 70% |
| Custom metric | `uisce_vm_eval_duration_seconds:p99` (Post-RC1) |

### 4.3 Multi-Region Raft

Three-node Raft consensus spans `us-east-1`, `eu-central-1`, `ap-east-1`. Raft handles leader election and automatic log replication. A minimum of 2 of 3 nodes must be available for write quorum.

---

## 5. Audit & Accountability

### 5.1 Hash Chain Audit Ledger

Every compliance evaluation is recorded with:
- `entryID` (UUID)
- `PreviousHash` (SHA-256 of previous entry)
- `CurrentHash` (SHA-256 of entry + PreviousHash)
- `TenantID`
- `EventType`
- `RawPayload` (JSON)

The hash chain is persisted via the Raft FSM (`replicatorFSM.Apply()`) and snapshotted via `replicatorSnapshot.Persist()`.

### 5.2 Log Retention

| Log type | Retention | Storage |
|----------|-----------|---------|
| Application logs | 30 days | `/var/log/uisce` (emptyDir) |
| Raft WAL | Until snapshot | StatefulSet PVC |
| Audit ledger entries | Immutable | Raft replicated |
| Shadow replay diffs | 30 days | PostgreSQL `shadow_replay_diffs` |

---

## 6. Operational Security

### 6.1 Image Provenance

- `image.repository: uisce-core-local` — built from local `Dockerfile`
- `image.pullPolicy: Never` — never pulls from external registry
- Local build via Kaniko or source-to-image in CI/CD pipeline

### 6.2 eBPF

eBPF XDP is **disabled** in RC1 (`EBPF_ENABLED=false`) due to:
- Requirement for host network + `CAP_NET_ADMIN`
- Linux kernel header dependencies
- Cross-platform CI compatibility

**Aspirational:** See `docs/production/POST-RC1.md`.

### 6.3 Container Isolation

All containers run as non-root user (`1000`). No `privileged: true` containers. No hostPath mounts except for emptyDir working directories.

---

## 7. G-SIFI Compliance Gaps (Post-RC1)

| Gap | Severity | Issue |
|-----|----------|-------|
| No SOC 2 Type II report | High | Third-party audit required |
| No WAF / DDoS protection | High | AWS WAF + CloudFront integration needed |
| No VPC flow log analysis | Medium | Ship to SIEM (Splunk/Grafana) |
| No RBAC manifests | Medium | Define `Role`/`RoleBinding` for operator/viewer |
| No Vault integration | Medium | Migrate from Infisical-only to Vault for secrets rotation |
| No DAST / SAST in CI | Medium | Add Burp Suite / SonarQube to pipeline |
