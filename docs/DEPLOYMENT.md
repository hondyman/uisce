# WealthVision Deployment Guide

## Prerequisites

### System Requirements
- **OS**: Linux (Ubuntu 22.04 LTS recommended) or macOS
- **CPU**: 4+ cores
- **RAM**: 16GB minimum, 32GB recommended
- **Storage**: 100GB SSD minimum

### Software Dependencies
- **Go**: 1.21+
- **PostgreSQL**: 15+
- **Redis**: 7+
- **Node.js**: 20+ (for frontend)
- **Docker**: 24+ (optional)
- **Kubernetes**: 1.28+ (for production)

---

## Local Development Setup

### 1. Clone Repository
```bash
git clone https://github.com/your-org/semlayer.git
cd semlayer
```

### 2. Install Dependencies
```bash
# Backend
cd backend
go mod download

# Frontend
cd ../frontend
npm install
```

### 3. Configure Environment
```bash
cp .env.example .env
```

Edit `.env`:
```env
# Database
DATABASE_URL=postgres://user:password@localhost:5432/wealthvision
DATABASE_MAX_CONNECTIONS=50

# Redis
REDIS_URL=redis://localhost:6379

# JWT
JWT_SECRET=your-256-bit-secret
JWT_EXPIRATION=24h

# External APIs
ZOOM_API_KEY=your_zoom_key
DOCUSIGN_API_KEY=your_docusign_key

# AWS (for document storage)
AWS_REGION=us-east-1
AWS_S3_BUCKET=wealthvision-docs

# Encryption
MESSAGE_ENCRYPTION_KEY=your-encryption-key
```

### 4. Run Database Migrations
```bash
cd backend
go run cmd/migrate/main.go up
```

This will create all 27 database tables:
- Phase 1: 16 tables (tax, multi-gen, alt investments, AI, ESG)
- Phase 2: 5 tables (risk management)
- Portal: 5 tables (messaging, e-signature, meetings)
- Compliance: 6 tables (Form ADV, GIPS, surveillance, audit)

### 5. Seed Test Data (Optional)
```bash
go run cmd/seed/main.go
```

### 6. Start Backend Server
```bash
cd backend
go run cmd/server/main.go
```

Server starts on `http://localhost:8080`

### 7. Start Frontend (if applicable)
```bash
cd frontend
npm run dev
```

Frontend starts on `http://localhost:3000`

---

## Docker Deployment

### 1. Build Docker Images
```bash
# Backend
docker build -t wealthvision-backend:latest -f backend/Dockerfile .

# Frontend
docker build -t wealthvision-frontend:latest -f frontend/Dockerfile .
```

### 2. Docker Compose
```yaml
# docker-compose.yml
version: '3.8'

services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: wealthvision
      POSTGRES_USER: wealthuser
      POSTGRES_PASSWORD: securepassword
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  backend:
    image: wealthvision-backend:latest
    depends_on:
      - postgres
      - redis
    environment:
      DATABASE_URL: postgres://wealthuser:securepassword@postgres:5432/wealthvision
      REDIS_URL: redis://redis:6379
    ports:
      - "8080:8080"

  frontend:
    image: wealthvision-frontend:latest
    depends_on:
      - backend
    environment:
      API_URL: http://backend:8080
    ports:
      - "3000:3000"

volumes:
  postgres_data:
```

### 3. Start Services
```bash
docker-compose up -d
```

---

## Kubernetes Deployment

### 1. Create Namespace
```bash
kubectl create namespace wealthvision
```

### 2. Apply ConfigMap
```yaml
# k8s/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: wealthvision-config
  namespace: wealthvision
data:
  DATABASE_HOST: postgres-service
  REDIS_HOST: redis-service
```

### 3. Create Secrets
```bash
kubectl create secret generic wealthvision-secrets \
  --from-literal=database-password=securepassword \
  --from-literal=jwt-secret=your-256-bit-secret \
  -n wealthvision
```

### 4. Deploy PostgreSQL
```yaml
# k8s/postgres-deployment.yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
  namespace: wealthvision
spec:
  serviceName: postgres-service
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:15
        env:
        - name: POSTGRES_DB
          value: wealthvision
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: wealthvision-secrets
              key: database-password
        volumeMounts:
        - name: postgres-storage
          mountPath: /var/lib/postgresql/data
  volumeClaimTemplates:
  - metadata:
      name: postgres-storage
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 100Gi
```

### 5. Deploy Backend
```yaml
# k8s/backend-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: wealthvision-backend
  namespace: wealthvision
spec:
  replicas: 3
  selector:
    matchLabels:
      app: wealthvision-backend
  template:
    metadata:
      labels:
        app: wealthvision-backend
    spec:
      containers:
      - name: backend
        image: wealthvision-backend:latest
        ports:
        - containerPort: 8080
        envFrom:
        - configMapRef:
            name: wealthvision-config
        env:
        - name: DATABASE_PASSWORD
          valueFrom:
            secretKeyRef:
              name: wealthvision-secrets
              key: database-password
        resources:
          requests:
            memory: "2Gi"
            cpu: "1000m"
          limits:
            memory: "4Gi"
            cpu: "2000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 5
```

### 6. Create Service & Ingress
```yaml
# k8s/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: wealthvision-backend
  namespace: wealthvision
spec:
  selector:
    app: wealthvision-backend
  ports:
  - port: 80
    targetPort: 8080
---
# k8s/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: wealthvision-ingress
  namespace: wealthvision
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
spec:
  tls:
  - hosts:
    - api.wealthvision.com
    secretName: wealthvision-tls
  rules:
  - host: api.wealthvision.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: wealthvision-backend
            port:
              number: 80
```

### 7. Apply All Manifests
```bash
kubectl apply -f k8s/
```

---

## Production Checklist

### Security
- [ ] Enable HTTPS/TLS everywhere
- [ ] Implement rate limiting
- [ ] Set up WAF (Web Application Firewall)
- [ ] Enable audit logging
- [ ] Implement secrets rotation
- [ ] Use least-privilege IAM roles

### Monitoring
- [ ] Set up Prometheus for metrics
- [ ] Configure Grafana dashboards
- [ ] Enable distributed tracing (Jaeger)
- [ ] Set up log aggregation (ELK/Loki)
- [ ] Configure alerting (PagerDuty/Opsgenie)

### Backup & Recovery
- [ ] Daily PostgreSQL backups
- [ ] Point-in-time recovery enabled
- [ ] Test restore procedures monthly
- [ ] Document disaster recovery plan
- [ ] Set RPO/RTO targets

### Performance
- [ ] Enable database connection pooling
- [ ] Implement Redis caching
- [ ] Set up CDN for static assets
- [ ] Enable gzip compression
- [ ] Optimize database indexes

### Compliance
- [ ] Enable encryption at rest
- [ ] Implement data retention policies
- [ ] Set up compliance monitoring
- [ ] Document security controls
- [ ] Conduct security audits

---

## Scaling

### Horizontal Scaling
```bash
# Scale backend
kubectl scale deployment wealthvision-backend --replicas=10 -n wealthvision

# Auto-scaling
kubectl autoscale deployment wealthvision-backend \
  --min=3 --max=20 --cpu-percent=70 -n wealthvision
```

### Database Scaling
- **Read Replicas**: Set up 2-3 read replicas
- **Connection Pooling**: Use PgBouncer
- **Partitioning**: Partition large tables by family_id
- **Archival**: Archive old audit logs to S3

---

## Monitoring & Alerts

### Key Metrics
- **API Latency**: p50, p95, p99
- **Error Rate**: 4xx, 5xx responses
- **Database**: Query time, connection pool usage
- **Queue Depth**: Background job queue
- **CPU/Memory**: Per-service utilization

### Sample Alerts
```yaml
# prometheus-alerts.yaml
groups:
- name: wealthvision
  rules:
  - alert: HighErrorRate
    expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.05
    annotations:
      summary: "High error rate detected"
      
  - alert: SlowAPIResponse
    expr: histogram_quantile(0.95, http_request_duration_seconds) > 2
    annotations:
      summary: "API response time > 2s"
```

---

## Troubleshooting

### Database Connection Issues
```bash
# Check connections
SELECT count(*) FROM pg_stat_activity;

# Kill idle connections
SELECT pg_terminate_backend(pid) 
FROM pg_stat_activity 
WHERE state = 'idle' AND state_change < now() - interval '10 minutes';
```

### High Memory Usage
```bash
# Check Go heap
curl http://localhost:8080/debug/pprof/heap

# Analyze with pprof
go tool pprof http://localhost:8080/debug/pprof/heap
```

### Slow Queries
```sql
-- Enable slow query log
ALTER SYSTEM SET log_min_duration_statement = 1000; -- 1s

-- Find slow queries
SELECT query, mean_exec_time, calls 
FROM pg_stat_statements 
ORDER BY mean_exec_time DESC 
LIMIT 10;
```

---

## Support & Resources

- **Documentation**: https://docs.wealthvision.com
- **Status Page**: https://status.wealthvision.com
- **Support Email**: support@wealthvision.com
- **Slack Community**: https://wealthvision-community.slack.com
