# Uisce Semantic OS — Helm Chart

A G-SIFI-hardened, multi-region Helm chart for the Uisce Semantic OS control plane. Built from local source (Kaniko/source-to-image) — no external container registry required.

## Quickstart

```bash
# Install with US East region values
helm install uisce ./deploy/helm \
  --values ./deploy/helm/values-prod-us-east.yaml \
  --namespace uisce --create-namespace

# Dry-run (validate templates without installing)
helm template uisce ./deploy/helm --values ./deploy/helm/values-prod-us-east.yaml

# Lint
helm lint ./deploy/helm
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  Ingress (nginx) → :8080 Gateway / :8090 Flight SQL stub   │
└─────────────────────────────────────────────────────────────┘
                              │
                    ┌─────────┴──────────┐
                    │                    │
              ┌─────▼─────┐       ┌──────▼──────┐
              │  Gateway   │       │ Raft Ledger  │
              │ Deployment │       │ StatefulSet  │
              │  (×3 pods) │       │  (PVC 10Gi) │
              └───────────┘        └─────────────┘
                    │                    │
         ┌───────────┼────────────┐       │
         │           │            │       │
    ┌────▼───┐  ┌────▼───┐  ┌───▼───┐    │
    │Redis   │  │Postgres│  │Redpanda│    │
    │:6379   │  │ :5432  │  │ :9092  │    │
    └────────┘  └────────┘  └────────┘    │
```

## Region Values

| Region | Values File |
|--------|-------------|
| US East 1 | `values-prod-us-east.yaml` |
| EU Central 1 | `values-prod-eu-central.yaml` |
| AP East 1 | `values-prod-ap-east.yaml` |

All three regions form a single Raft consensus cluster (3-node). Raft handles leader election automatically.

## Ports

| Port | Service | Status |
|------|---------|--------|
| 8080 | REST / MCP API Gateway | **Active** |
| 8090 | Arrow Flight SQL | Aspirational stub — listener active, no query handler yet |
| 8980 | QuickFIX Acceptor | Aspirational stub — server starts, no FIX session configured |
| 7946 | Raft Consensus (TCP) | **Active** (StatefulSet) |

## Aspirational Components (Post-RC1)

The following are wired as stubs in RC1 and require Post-RC1 implementation:

| Component | Port | Description |
|-----------|------|-------------|
| Arrow Flight SQL | 8090 | Listener starts (`flight.NewFlightServer(8090)`); needs gRPC query handler |
| QuickFIX Acceptor | 8980 | `fix.NewServer()` starts; needs FIX session + tickerplant wiring |
| StarRocks | 9030 | `STARROCKS_HOST` env var passed but not read by backend |
| eBPF XDP | — | `EBPF_ENABLED=false` in Helm; requires host network + `CAP_NET_ADMIN` |

## G-SIFI Security Posture

| Setting | Value |
|---------|-------|
| `readOnlyRootFilesystem` | `true` (with emptyDir for `/tmp`, `/var/lib/uisce/raft`, `/var/log`) |
| `capabilities.drop` | `[ALL]` at pod and container level |
| `seccompProfile` | `RuntimeDefault` |
| `runAsNonRoot` | `true` / `runAsUser: 1000` |
| `allowPrivilegeEscalation` | `false` |
| Network Policy | Ingress restricted to nginx ingress + namespace; Egress to DNS/5432/6379/9092 |

## Infisical Secret Integration

The chart expects a pre-provisioned Infisical Secret CRD in the target namespace containing:

| Key | Description |
|-----|-------------|
| `POSTGRES_DSN` | Primary PostgreSQL connection string |
| `JWT_SECRET` | JWT signing key |
| `RAFT_ENCRYPTION_KEY` | AES key for Raft log encryption at rest |
| `KAFKA_SASL_PASSWORD` | Redpanda SASL password (if SASL auth enabled) |

Provision the secret CRD before `helm install`:

```bash
infisical secrets create uisce-infisical-secrets-us-east \
  --env production \
  --secret POSTGRES_DSN="postgres://..." \
  --secret JWT_SECRET="$(openssl rand -base64 32)"
```

## Multi-Region Raft Consensus

Raft peers are configured via `raftLedger.clusterPeers`. All three regions are always listed; Raft automatically elects a leader and replicates the SHA-256 hash chain across the cluster.

```
us-east-1 ←→ eu-central-1 ←→ ap-east-1
    ↑_______________↑_______________↑  (Raft replication mesh)
```

## Migration Init Container

When `initContainers.enabled: true`, a `postgres:15-alpine` init container runs all `backend/db/migrations/*.up.sql` files against the target database before the core application starts.

## Troubleshooting

```bash
# Check pod status
kubectl get pods -n uisce

# View Raft leader election logs
kubectl logs -n uisce -l app.kubernetes.io/component=raft-ledger

# Check HPA status
kubectl get hpa -n uisce

# Port-forward for local testing
kubectl port-forward -n uisce svc/uisce 8080:8080
```

## Terraform Integration

The chart is integrated into `terraform/environments/production-{us,eu-central,ap-east}/main.tf` via:

```hcl
resource "helm_release" "uisce" {
  name       = "uisce"
  chart      = "${path.module}/../../deploy/helm"
  values     = ["${file("${path.module}/../../deploy/helm/values-prod-us-east.yaml")}"]
  namespace  = "uisce"
}
```
