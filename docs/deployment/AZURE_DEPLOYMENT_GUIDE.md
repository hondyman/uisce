# Semlayer Global Platform - Deployment Guide

## Overview

This guide covers deploying Semlayer as a globally distributed, multi-tenant platform on Azure, similar to enterprise SaaS platforms like Workday.

## Architecture Summary

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Azure Front Door (Global)                          │
│                    WAF / DDoS Protection / SSL Termination                   │
└─────────────────┬───────────────────────────────────────────┬───────────────┘
                  │                                           │
         ┌────────▼────────┐                         ┌────────▼────────┐
         │   US East AKS   │                         │  EU West AKS    │
         │   (Primary)     │                         │  (GDPR Region)  │
         └────────┬────────┘                         └────────┬────────┘
                  │                                           │
    ┌─────────────┴─────────────┐           ┌─────────────────┴───────────────┐
    │  ┌─────────────────────┐  │           │  ┌─────────────────────┐        │
    │  │    API Gateway      │  │           │  │    API Gateway      │        │
    │  └─────────┬───────────┘  │           │  └─────────┬───────────┘        │
    │            │              │           │            │                    │
    │  ┌─────────▼───────────┐  │           │  ┌─────────▼───────────┐        │
    │  │  Semantic Engine    │  │           │  │  Semantic Engine    │        │
    │  │  Rule Engine        │  │           │  │  Rule Engine        │        │
    │  │  Cube.js            │◄─┼───────────┼──│  Cube.js            │        │
    │  │  Compliance Engine  │  │           │  │  Compliance Engine  │        │
    │  │  AI Builder         │  │           │  │  AI Builder         │        │
    │  └─────────┬───────────┘  │           │  └─────────┬───────────┘        │
    │            │              │           │            │                    │
    │  ┌─────────▼───────────┐  │           │  ┌─────────▼───────────┐        │
    │  │ Azure PostgreSQL    │  │           │  │ Azure PostgreSQL    │        │
    │  │ Azure Redis         │◄─┼───Geo─────┼──│ Azure Redis         │        │
    │  │ Service Bus         │  │ Replication│ │ Service Bus         │        │
    │  └─────────────────────┘  │           │  └─────────────────────┘        │
    └───────────────────────────┘           └─────────────────────────────────┘
```

## Prerequisites

### Azure Resources Needed
- Azure Subscription with appropriate quotas
- Azure Container Registry (ACR)
- Azure Key Vault
- Azure Monitor / Log Analytics Workspace

### Local Tools Required
```bash
# Install Azure CLI
brew install azure-cli

# Install kubectl
brew install kubectl

# Install Helm
brew install helm

# Install Terraform
brew install terraform

# Login to Azure
az login
az account set --subscription "Your-Subscription-Name"
```

## Deployment Steps

### 1. Provision Azure Infrastructure

```bash
cd terraform/environments/production-azure

# Initialize Terraform
terraform init

# Review the plan
terraform plan -out=tfplan

# Apply the infrastructure
terraform apply tfplan
```

This creates:
- AKS clusters in multiple regions
- Azure Database for PostgreSQL (Flexible Server)
- Azure Cache for Redis (Enterprise)
- Azure Service Bus (Premium)
- Azure Key Vault
- Azure Front Door
- Azure Monitor / Log Analytics

### 2. Configure AKS Credentials

```bash
# Get credentials for each cluster
az aks get-credentials \
  --resource-group semlayer-prod-rg \
  --name semlayer-prod-eastus \
  --overwrite-existing

# Verify connection
kubectl get nodes
```

### 3. Install Base Components

```bash
# Add Helm repositories
helm repo add istio https://istio-release.storage.googleapis.com/charts
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update

# Install Istio
kubectl create namespace istio-system
helm install istio-base istio/base -n istio-system
helm install istiod istio/istiod -n istio-system --wait

# Install Istio Ingress Gateway
kubectl create namespace istio-ingress
helm install istio-ingress istio/gateway -n istio-ingress

# Apply Istio configuration
kubectl apply -f infrastructure/k8s/istio/istio-config.yaml

# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Install NGINX Ingress (optional, if not using Istio gateway)
helm install ingress-nginx ingress-nginx/ingress-nginx \
  --namespace ingress-nginx \
  --create-namespace
```

### 4. Install Monitoring Stack

```bash
# Create monitoring namespace
kubectl create namespace monitoring

# Install Prometheus Stack
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --set grafana.enabled=true \
  --set alertmanager.enabled=true

# Apply custom alert rules
kubectl apply -f monitoring/alerts/prometheus-rules.yaml

# Import Grafana dashboards
kubectl create configmap grafana-dashboards \
  --from-file=monitoring/dashboards/ \
  -n monitoring
```

### 5. Create Secrets

```bash
# Store secrets in Azure Key Vault (done by Terraform)
# Or create Kubernetes secrets directly:

kubectl create namespace semlayer-control

kubectl create secret generic semlayer-db-credentials \
  --namespace semlayer-control \
  --from-literal=username=semlayer \
  --from-literal=password='YOUR_DB_PASSWORD'

kubectl create secret generic semlayer-redis-credentials \
  --namespace semlayer-control \
  --from-literal=password='YOUR_REDIS_PASSWORD'

kubectl create secret generic semlayer-api-secrets \
  --namespace semlayer-control \
  --from-literal=cube-api-secret='YOUR_CUBE_SECRET' \
  --from-literal=jwt-secret='YOUR_JWT_SECRET'
```

### 6. Deploy Semlayer Platform

```bash
# Deploy to staging first
helm upgrade --install semlayer-staging ./charts/semlayer-platform \
  --namespace semlayer-staging \
  --create-namespace \
  -f charts/semlayer-platform/values.yaml \
  -f charts/semlayer-platform/values-azure.yaml \
  -f charts/semlayer-platform/values-staging.yaml \
  --wait --timeout 10m

# Verify staging deployment
kubectl get pods -n semlayer-staging
kubectl get svc -n semlayer-staging

# Run smoke tests
curl https://api-staging.semlayer.io/health

# Deploy to production (after staging validation)
helm upgrade --install semlayer ./charts/semlayer-platform \
  --namespace semlayer-control \
  --create-namespace \
  -f charts/semlayer-platform/values.yaml \
  -f charts/semlayer-platform/values-azure.yaml \
  -f charts/semlayer-platform/values-production.yaml \
  --wait --timeout 15m
```

### 7. Configure ArgoCD (GitOps)

```bash
# Install ArgoCD
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Apply ApplicationSet for multi-region deployment
kubectl apply -f gitops/applications/semlayer-platform.yaml

# Get ArgoCD admin password
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
```

### 8. Apply Network Policies

```bash
kubectl apply -f infrastructure/k8s/network-policies/semlayer-network-policies.yaml
```

## Multi-Region Deployment

For global deployment like Workday:

### Deploy to EU West (GDPR Region)

```bash
# Get EU cluster credentials
az aks get-credentials \
  --resource-group semlayer-prod-westeurope-rg \
  --name semlayer-prod-westeurope

# Deploy with EU-specific values
helm upgrade --install semlayer ./charts/semlayer-platform \
  --namespace semlayer-control \
  -f charts/semlayer-platform/values.yaml \
  -f charts/semlayer-platform/values-azure.yaml \
  -f charts/semlayer-platform/values-production.yaml \
  -f charts/semlayer-platform/values-eu.yaml \
  --set global.region=westeurope \
  --set global.dataResidency.enabled=true \
  --set global.dataResidency.region=eu
```

### Deploy to Southeast Asia

```bash
az aks get-credentials \
  --resource-group semlayer-prod-southeastasia-rg \
  --name semlayer-prod-southeastasia

helm upgrade --install semlayer ./charts/semlayer-platform \
  --namespace semlayer-control \
  -f charts/semlayer-platform/values.yaml \
  -f charts/semlayer-platform/values-azure.yaml \
  -f charts/semlayer-platform/values-production.yaml \
  --set global.region=southeastasia
```

## Azure Front Door Configuration

Configure global routing:

```bash
# Front Door is created by Terraform, but you can update:
az afd endpoint update \
  --resource-group semlayer-prod-rg \
  --profile-name semlayer-frontdoor \
  --endpoint-name api \
  --enabled-state Enabled
```

## Monitoring & Observability

### Access Grafana

```bash
# Port-forward to Grafana
kubectl port-forward svc/prometheus-grafana 3000:80 -n monitoring

# Open browser to http://localhost:3000
# Default credentials: admin / prom-operator
```

### View Logs in Azure

```bash
# Query logs via Azure CLI
az monitor log-analytics query \
  --workspace semlayer-prod-logs \
  --analytics-query "ContainerLog | where ContainerID contains 'api-gateway' | limit 100"
```

### Jaeger Tracing

```bash
kubectl port-forward svc/jaeger-query 16686:16686 -n monitoring
# Open http://localhost:16686
```

## Scaling Operations

### Manual Scaling

```bash
# Scale API Gateway
kubectl scale deployment api-gateway --replicas=10 -n semlayer-control

# Scale Cube workers
kubectl scale deployment cube-refresh-worker --replicas=5 -n semlayer-control
```

### Update HPA Limits

```bash
kubectl patch hpa api-gateway -n semlayer-control \
  --patch '{"spec":{"maxReplicas":50}}'
```

## Disaster Recovery

### Trigger Manual Failover

```bash
# Failover database to secondary region
az postgres flexible-server replica promote \
  --resource-group semlayer-prod-westeurope-rg \
  --name semlayer-postgresql-replica
```

### Backup Verification

```bash
# List available backups
az postgres flexible-server backup list \
  --resource-group semlayer-prod-rg \
  --name semlayer-postgresql

# Restore from backup
az postgres flexible-server restore \
  --resource-group semlayer-prod-rg \
  --name semlayer-postgresql-restored \
  --source-server semlayer-postgresql \
  --restore-time "2024-01-15T00:00:00Z"
```

## Troubleshooting

### Check Pod Status

```bash
kubectl get pods -n semlayer-control -o wide
kubectl describe pod <pod-name> -n semlayer-control
kubectl logs <pod-name> -n semlayer-control --tail=100
```

### Check Service Mesh

```bash
# Istio proxy status
istioctl proxy-status

# Debug Istio configuration
istioctl analyze -n semlayer-control
```

### Database Connection Issues

```bash
# Test database connectivity
kubectl run pg-test --rm -it --image=postgres:15 -- \
  psql "host=semlayer-postgresql.postgres.database.azure.com \
        dbname=semantic user=semlayer sslmode=require"
```

### Redis Connection Issues

```bash
# Test Redis connectivity
kubectl run redis-test --rm -it --image=redis:7 -- \
  redis-cli -h semlayer-redis.redis.cache.windows.net \
            -p 6380 --tls --askpass
```

## Security Checklist

- [ ] Network policies applied
- [ ] Istio mTLS in STRICT mode
- [ ] Pod security standards enforced
- [ ] Azure Key Vault integrated
- [ ] Azure AD Workload Identity configured
- [ ] WAF rules enabled on Front Door
- [ ] DDoS protection enabled
- [ ] Audit logging enabled
- [ ] Secrets rotated quarterly

## Cost Optimization

### Reserved Instances
- Consider Azure Reserved VMs for predictable workloads
- Use Spot instances for Cube.js pre-aggregation workers

### Auto-scaling
- Configure cluster autoscaler for node-level scaling
- Use KEDA for event-driven scaling

### Resource Right-sizing
- Review resource requests/limits monthly
- Use Azure Advisor recommendations

## Support

- **Documentation**: https://docs.semlayer.io
- **Status Page**: https://status.semlayer.io
- **Support**: support@semlayer.io
- **Emergency**: +1-XXX-XXX-XXXX (24/7)
