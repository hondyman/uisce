# Kubernetes Production Deployment

This directory contains Kubernetes manifests for deploying the Semlayer microservices platform in production with monitoring, scaling, and high availability.

## Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Ingress       │    │   Services      │    │   Monitoring    │
│   (nginx)       │    │                 │    │                 │
│                 │    │ • AI Builder    │    │ • Prometheus    │
│ api.semlayer.com│◄──►│ • Compliance    │◄──►│ • Grafana       │
│ temporal.       │    │   Engine        │    │ • AlertManager │
│ semlayer.com    │    │ • API Gateway   │    │                 │
└─────────────────┘    │ • Temporal      │    └─────────────────┘
                       │ • PostgreSQL    │            ▲
                       │ • RabbitMQ      │            │
                       └─────────────────┘            ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Autoscaling   │    │   Security      │    │   Storage       │
│   (HPA)         │    │                 │    │                 │
│                 │    │ • Network       │    │ • PVCs          │
│ • CPU/Memory    │    │   Policies      │    │ • ConfigMaps    │
│ • Custom        │    │ • RBAC          │    │ • Secrets       │
│   Metrics       │    │ • Secrets       │    └─────────────────┘
└─────────────────┘    └─────────────────┘
```

## Prerequisites

- Kubernetes cluster (v1.24+)
- kubectl configured
- Helm 3.x
- cert-manager (for TLS certificates)
- Prometheus Operator (for monitoring)

### Required Operators

```bash
# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Install Prometheus Operator
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/kube-prometheus-stack
```

## Deployment Order

1. **ConfigMaps and Secrets** - Base configuration
2. **Database and Message Queue** - Infrastructure services
3. **Microservices** - Application services
4. **Ingress and Monitoring** - External access and observability

## Quick Start

### 1. Deploy Configuration
```bash
kubectl apply -f k8s/config-and-secrets.yaml
```

### 2. Deploy Infrastructure
```bash
# Deploy PostgreSQL and RabbitMQ (from existing docker-compose or separate manifests)
kubectl apply -f k8s/postgres-deployment.yaml
kubectl apply -f k8s/rabbitmq-deployment.yaml
kubectl apply -f k8s/temporal-deployment.yaml
```

### 3. Deploy Microservices
```bash
kubectl apply -f k8s/ai-builder-deployment.yaml
kubectl apply -f k8s/compliance-engine-deployment.yaml
```

### 4. Deploy Ingress and Monitoring
```bash
kubectl apply -f k8s/ingress.yaml
kubectl apply -f k8s/monitoring.yaml
```

## Service Configuration

### AI Builder Service
- **Replicas**: 2-10 (autoscaled)
- **Resources**: 256Mi-512Mi RAM, 250m-500m CPU
- **Health Checks**: HTTP `/health` endpoint
- **Scaling Metrics**: CPU (70%), Memory (80%)

### Compliance Engine Service
- **Replicas**: 2-8 (autoscaled)
- **Resources**: 512Mi-1Gi RAM, 500m-1000m CPU
- **Health Checks**: HTTP `/health` endpoint
- **Scaling Metrics**: CPU (70%), Memory (80%), Custom metrics

## Monitoring and Observability

### Metrics Collected
- **Application Metrics**:
  - HTTP request rates and latencies
  - Compliance event counts
  - Workflow check results
  - ABAC evaluation statistics

- **System Metrics**:
  - CPU and memory usage
  - Network I/O
  - Disk usage
  - Container restarts

### Dashboards
Access Grafana at: `http://grafana.local`

Pre-configured dashboards:
- **Microservices Overview**: Request rates, error rates, latency
- **Compliance Monitoring**: Event processing, violation detection
- **System Resources**: CPU, memory, network usage

### Alerting Rules
- **High Compliance Event Rate**: >10 events/second for 5 minutes
- **Workflow Compliance Failures**: >5 failures in 5 minutes
- **Service Down**: Any service unavailable for 5 minutes
- **High Resource Usage**: CPU >80%, Memory >90%

## Security Configuration

### Network Policies
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: microservice-policy
spec:
  podSelector:
    matchLabels:
      component: microservice
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 8081  # AI Builder
    - protocol: TCP
      port: 8082  # Compliance Engine
```

### RBAC Configuration
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: microservice-role
rules:
- apiGroups: [""]
  resources: ["configmaps", "secrets"]
  verbs: ["get", "list", "watch"]
```

## Scaling Configuration

### Horizontal Pod Autoscaling (HPA)

#### AI Builder
```yaml
minReplicas: 2
maxReplicas: 10
metrics:
- type: Resource
  resource:
    name: cpu
    target:
      type: Utilization
      averageUtilization: 70
```

#### Compliance Engine
```yaml
minReplicas: 2
maxReplicas: 8
metrics:
- type: Resource
  resource:
    name: cpu
    target:
      type: Utilization
      averageUtilization: 70
- type: Pods
  pods:
    metric:
      name: compliance_events_total
    target:
      type: AverageValue
      averageValue: 100
```

### Scaling Behavior
- **Scale Up**: Rapid response to increased load
- **Scale Down**: Conservative to prevent thrashing
- **Stabilization**: 5-minute windows to prevent oscillations

## Backup and Recovery

### Database Backups
```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: postgres-backup
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: postgres:15-alpine
            command:
            - pg_dump
            - -h
            - postgres-service
            - -U
            - postgres
            - alpha
            volumeMounts:
            - name: backup-storage
              mountPath: /backup
          volumes:
          - name: backup-storage
            persistentVolumeClaim:
              claimName: backup-pvc
```

### Disaster Recovery
1. **Pod Failure**: Kubernetes automatically restarts containers
2. **Node Failure**: Pods rescheduled to healthy nodes
3. **Service Failure**: Load balancer routes traffic to healthy instances
4. **Data Recovery**: Restore from automated backups

## Troubleshooting

### Common Issues

#### Service Unavailable
```bash
# Check pod status
kubectl get pods -l app=ai-builder

# Check service endpoints
kubectl get endpoints ai-builder-service

# Check logs
kubectl logs -l app=ai-builder --tail=100
```

#### High Resource Usage
```bash
# Check resource usage
kubectl top pods

# Check HPA status
kubectl get hpa

# Scale manually if needed
kubectl scale deployment ai-builder --replicas=5
```

#### Monitoring Issues
```bash
# Check Prometheus targets
kubectl get servicemonitors

# Check alert status
kubectl get prometheusrules

# Access Grafana
kubectl port-forward svc/prometheus-grafana 3000:80
```

### Health Checks

#### Application Health
```bash
# AI Builder health
curl http://ai-builder-service:8081/health

# Compliance Engine health
curl http://compliance-engine-service:8082/health
```

#### System Health
```bash
# Check cluster status
kubectl get nodes
kubectl get componentstatuses

# Check resource usage
kubectl describe nodes
```

## Performance Tuning

### Resource Limits
- **AI Builder**: 256Mi-512Mi RAM, 250m-500m CPU
- **Compliance Engine**: 512Mi-1Gi RAM, 500m-1000m CPU
- **Database**: 2Gi-4Gi RAM, 1000m-2000m CPU

### Network Optimization
- **Service Mesh**: Consider Istio for advanced traffic management
- **Ingress Tuning**: Adjust nginx worker processes and connections
- **Load Balancing**: Use session affinity for stateful operations

### Database Optimization
- **Connection Pooling**: Use pgbouncer for PostgreSQL
- **Indexing**: Ensure proper indexes on frequently queried columns
- **Caching**: Implement Redis for frequently accessed data

## CI/CD Integration

### GitOps Workflow
```yaml
# .github/workflows/deploy.yml
name: Deploy to Kubernetes
on:
  push:
    branches: [main]
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - name: Build and push images
      run: |
        docker build -t semlayer/ai-builder:latest ./services/ai-builder
        docker build -t semlayer/compliance-engine:latest ./services/compliance-engine
        docker push semlayer/ai-builder:latest
        docker push semlayer/compliance-engine:latest
    - name: Deploy to Kubernetes
      run: |
        kubectl apply -f k8s/
        kubectl rollout status deployment/ai-builder
        kubectl rollout status deployment/compliance-engine
```

### Blue-Green Deployment
```bash
# Create blue deployment
kubectl apply -f k8s/ai-builder-blue.yaml

# Switch ingress to blue
kubectl patch ingress semlayer-ingress -p '{"spec":{"rules":[{"host":"api.semlayer.com","http":{"paths":[{"path":"/ai-builder","backend":{"service":{"name":"ai-builder-blue","port":{"number":8081}}}}]}}]}}'

# Verify and delete green
kubectl delete deployment ai-builder-green
```

## Cost Optimization

### Resource Rightsizing
- **Monitoring**: Use Prometheus metrics to identify over-provisioned resources
- **Autoscaling**: Configure appropriate min/max replica counts
- **Spot Instances**: Use spot instances for non-critical workloads

### Storage Optimization
- **PVC Sizing**: Right-size persistent volumes based on usage patterns
- **Cleanup Jobs**: Implement automated cleanup of old logs and temporary files
- **Compression**: Enable compression for log storage

## Compliance and Security

### Security Scanning
```bash
# Container image scanning
trivy image semlayer/ai-builder:latest
trivy image semlayer/compliance-engine:latest

# Kubernetes security
kube-bench run
```

### Audit Logging
- **Application Logs**: Structured JSON logging with correlation IDs
- **System Logs**: Kubernetes audit logs for cluster operations
- **Access Logs**: Ingress controller logs for external access

### Compliance Checks
- **SOC 2**: Regular security assessments and penetration testing
- **GDPR**: Data encryption and access controls
- **FINRA**: Financial industry compliance requirements

## Support and Maintenance

### Regular Tasks
- **Certificate Renewal**: cert-manager handles Let's Encrypt certificates
- **Security Updates**: Regular image updates with security patches
- **Log Rotation**: Configure log aggregation and retention policies
- **Backup Verification**: Regular backup integrity checks

### Emergency Procedures
1. **Service Outage**: Check pod status and restart if necessary
2. **Data Loss**: Restore from latest backup
3. **Security Incident**: Isolate affected components and investigate
4. **Performance Issues**: Scale resources or optimize queries

### Contact Information
- **DevOps Team**: devops@semlayer.com
- **Security Team**: security@semlayer.com
- **On-call Engineer**: +1-555-0123