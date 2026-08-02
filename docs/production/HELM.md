# Uisce Semantic OS — Helm Chart Usage Guide (RC1)

## Overview

The `deploy/helm/` chart deploys the Uisce Semantic OS control plane into Kubernetes. It is designed for G-SIFI compliance, multi-region Raft consensus, and local source builds (no external container registry required).

---

## Prerequisites

- Kubernetes 1.28+
- Helm 3.12+
- `kubectl` configured with cluster credentials
- Infisical CLI (optional, for production secret injection)

---

## Quickstart

### 1. Clone & Navigate

```bash
git clone https://github.com/hondyman/uisce
cd uisce
```

### 2. Provision Infrastructure

First, ensure the target Kubernetes cluster, PostgreSQL, Redis, and Redpanda are provisioned. For AWS EKS, use the Terraform modules in `terraform/environments/production-us/main.tf`:

```bash
cd terraform/environments/production-us
terraform init
terraform plan -var="aws_region=us-east-1"
terraform apply -var="aws_region=us-east-1"
```

### 3. Create Kubernetes Namespace

```bash
kubectl create namespace uisce
```

### 4. Provision Secrets

Before installing, create a Kubernetes Secret containing the required credentials:

```bash
# Option A: Manual secret (development only)
kubectl create secret generic uisce-infisical-secrets-us-east \
  --namespace uisce \
  --from-literal=POSTGRES_DSN="postgres://uisce_admin:PASSWORD@postgres:5432/uisce_control_plane?sslmode=disable" \
  --from-literal=JWT_SECRET="$(openssl rand -base64 32)" \
  --from-literal=RAFT_ENCRYPTION_KEY="$(openssl rand -base64 32)"

# Option B: Via Infisical CLI (production)
infisical secrets create uisce-infisical-secrets-us-east \
  --project-id=YOUR_PROJECT_ID \
  --env=production \
  --namespace=uisce \
  --secret POSTGRES_DSN="postgres://..." \
  --secret JWT_SECRET="$(openssl rand -base64 32)"
```

### 5. Install via Helm

```bash
# Dry-run first (validate templates without installing)
helm template uisce ./deploy/helm \
  --values ./deploy/helm/values-prod-us-east.yaml \
  --namespace uisce

# Lint
helm lint ./deploy/helm

# Install
helm install uisce ./deploy/helm \
  --values ./deploy/helm/values-prod-us-east.yaml \
  --namespace uisce \
  --create-namespace

# Verify
kubectl get pods -n uisce
kubectl get svc -n uisce
```

### 6. Upgrade

```bash
helm upgrade uisce ./deploy/helm \
  --values ./deploy/helm/values-prod-us-east.yaml \
  --namespace uisce
```

---

## Region Values

| Region | Values File | Raft Node ID |
|--------|-------------|---------------|
| US East 1 | `values-prod-us-east.yaml` | `uisce-node-us-east-1` |
| EU Central 1 | `values-prod-eu-central.yaml` | `uisce-node-eu-central-1` |
| AP East 1 | `values-prod-ap-east.yaml` | `uisce-node-ap-east-1` |

---

## Terraform Integration

The chart is integrated into Terraform via the `helm_release` resource. Example (from `terraform/environments/production-us/main.tf`):

```hcl
resource "helm_release" "uisce" {
  name       = "uisce"
  chart      = "${path.module}/../../deploy/helm"
  namespace  = "uisce"
  create_namespace = true

  values = [
    file("${path.module}/../../deploy/helm/values-prod-us-east.yaml"),
  ]

  set {
    name  = "global.region"
    value = "us-east-1"
  }

  set {
    name  = "raftLedger.enabled"
    value = "true"
  }

  set {
    name  = "raftLedger.nodeID"
    value = "uisce-node-us-east-1"
  }

  set {
    name  = "infisical.secretRef"
    value = "uisce-infisical-secrets-us-east"
  }

  set {
    name  = "image.pullPolicy"
    value = "Never"
  }

  depends_on = [
    module.eks,
    module.rds,
    module.elasticache,
  ]
}
```

Mirrored for `terraform/environments/production-eu-central/main.tf` and `terraform/environments/production-ap-east/main.tf`.

---

## Configuration Reference

### Global Settings

```yaml
global:
  environment: production          # production | staging
  complianceLevel: g-sifi         # g-sifi
  region: us-east-1               # us-east-1 | eu-central-1 | ap-east-1
```

### Raft Consensus Ledger

```yaml
raftLedger:
  enabled: true
  nodeID: "uisce-node-us-east-1"
  dataDir: "/var/lib/uisce/raft"
  clusterPeers:
    - "uisce-node-us-east-1.uisce.svc.cluster.local:7946"
    - "uisce-node-eu-central-1.uisce.svc.cluster.local:7946"
    - "uisce-node-ap-east-1.uisce.svc.cluster.local:7946"
```

### Security Context (G-SIFI)

```yaml
podSecurityContext:
  runAsNonRoot: true
  runAsUser: 1000
  seccompProfile:
    type: RuntimeDefault
  capabilities:
    drop: [ALL]

containerSecurityContext:
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
  capabilities:
    drop: [ALL]
```

### Autoscaling

```yaml
autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 12
  targetCPUUtilizationPercentage: 70
```

### Infisical

```yaml
infisical:
  enabled: true
  secretRef: uisce-infisical-secrets-us-east
```

---

## Aspirational Ports (Post-RC1)

| Port | Service | RC1 Status |
|------|---------|------------|
| 8080 | REST / MCP API Gateway | **Active** |
| 8090 | Arrow Flight SQL | Aspirational stub — listener active, no query handler |
| 8980 | QuickFIX Acceptor | Aspirational stub — server starts, no session configured |
| 7946 | Raft Consensus (TCP) | **Active** (StatefulSet) |

---

## Pod Resource Layout

```
uisce-0 (StatefulSet / Raft node)
  ├── init (postgres:15-alpine)     # runs migrations
  └── uisce-core                    # main application

uisce-deployment-xxxxx (Deployment / 3 replicas)
  └── uisce-core                    # main application

Headless Service: uisce-raft-headless (for Raft peer DNS)
ClusterIP Service: uisce (gateway :8080, flight :8090, fix :8980)
```

---

## Troubleshooting

```bash
# Check pod status
kubectl get pods -n uisce -l app.kubernetes.io/component=backend
kubectl get pods -n uisce -l app.kubernetes.io/component=raft-ledger

# View logs
kubectl logs -n uisce -l app.kubernetes.io/component=backend --tail=100
kubectl logs -n uisce -l app.kubernetes.io/component=raft-ledger --tail=100

# Check Raft leadership
kubectl exec -n uisce uisce-raft-0 -- sh -c 'echo "info" | nc localhost 7946' 2>/dev/null || \
  kubectl logs -n uisce -l app.kubernetes.io/component=raft-ledger | grep -i leader

# Check HPA status
kubectl get hpa -n uisce

# Check PVC (Raft WAL)
kubectl get pvc -n uisce

# Port-forward for local testing
kubectl port-forward -n uisce svc/uisce 8080:8080

# Helm uninstall
helm uninstall uisce --namespace uisce
```

---

## Terraform Module References

| Module | Path | Purpose |
|--------|------|---------|
| `raft-consensus` | `terraform/modules/raft-consensus/` | Subnet, SG, IAM for Raft TCP 7946 |
| `infisical-secret` | `terraform/modules/infisical-secret/` | Infisical → K8s Secret mapping |
