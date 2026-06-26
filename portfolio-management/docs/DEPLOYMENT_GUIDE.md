# Portfolio Management System - Deployment & Setup Guide

## 🚀 Quick Start (Local Development)

### Prerequisites
- Docker & Docker Compose 
- PostgreSQL 15+ (if running locally)
- Go 1.21+ (for local development)
- Node.js 18+ (for frontend development)
- Git

### 1. Clone & Setup

```bash
# Clone repository
cd /Users/eganpj/GitHub/semlayer
cd portfolio-management

# Copy environment configuration
cp .env.example .env

# Edit .env with your credentials
nano .env  # or use your preferred editor
```

### 2. Configure Environment Variables

Key variables to update in `.env`:

```env
# Database
DB_PASSWORD=your_secure_password

# Hasura
HASURA_ADMIN_SECRET=your_secure_admin_secret
JWT_SECRET=your_secure_jwt_secret

# Email (SMTP)
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password

# SMS (Twilio) - Optional
TWILIO_ACCOUNT_SID=your_sid
TWILIO_AUTH_TOKEN=your_token
TWILIO_PHONE_NUMBER=+1234567890

# Push (Pusher) - Optional
PUSHER_APP_ID=your_app_id
PUSHER_KEY=your_key
PUSHER_SECRET=your_secret
```

### 3. Start Services with Docker Compose

```bash
# Navigate to docker directory
cd docker

# Start all services (PostgreSQL, Hasura, Notification Service)
docker-compose up -d

# Verify services are running
docker-compose ps

# View logs
docker-compose logs -f

# To stop services
docker-compose down
```

### 4. Initialize Database

```bash
# The database is automatically initialized from init.sql
# Verify the schema was created:
docker-compose exec postgres psql -U portfolio -d portfolio_db -c "\dt"

# Or run migrations manually if needed
docker-compose exec postgres psql -U portfolio -d portfolio_db -f /docker-entrypoint-initdb.d/01-init.sql
```

### 5. Access Services

Once services are running:

- **Hasura Console**: http://localhost:8080
  - Admin Secret: (value from `.env` HASURA_ADMIN_SECRET)
  
- **GraphQL API**: http://localhost:8080/v1/graphql
  
- **Notification Service**: http://localhost:8081
  - Health Check: http://localhost:8081/health
  
- **PostgreSQL**: localhost:5432
  - User: portfolio
  - Password: (value from `.env` DB_PASSWORD)

---

## 📊 Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                        React Frontend                         │
│                    (Real-time Dashboard)                      │
└────────────────────────┬────────────────────────────────────┘
                         │
              ┌──────────┴──────────┐
              │                     │
        ┌─────▼──────┐      ┌──────▼──────┐
        │   Hasura    │      │  Websocket  │
        │ GraphQL API │      │  Real-time  │
        └─────┬──────┘      └──────┬──────┘
              │                    │
        ┌─────▼────────────────────▼──────┐
        │      PostgreSQL Database        │
        │  • Portfolios & Holdings        │
        │  • Recommendations              │
        │  • Notifications                │
        │  • Execution History            │
        └─────┬────────────────────┬──────┘
              │                    │
        ┌─────▼──────┐      ┌──────▼─────────┐
        │  Temporal   │      │ Notification   │
        │  Workflows  │      │ Service (Go)   │
        │ (Rebalance) │      │ • Email        │
        │             │      │ • SMS (Twilio) │
        │             │      │ • Push         │
        └─────────────┘      │ • In-app       │
                             └────────────────┘
```

---

## 📈 Database Schema

### Core Tables

| Table | Purpose | Key Columns |
|-------|---------|-------------|
| `users` | User accounts & auth | id, email, password_hash, role |
| `portfolios` | Portfolio metadata | id, user_id, total_value, allocations |
| `holdings` | Individual positions | id, portfolio_id, ticker, shares, price |
| `recommendations` | AI recommendations | id, portfolio_id, type, priority, actions |
| `rebalance_orders` | Execution records | id, portfolio_id, status, tax_savings |
| `notifications` | User notifications | id, user_id, type, priority, channels |
| `portfolio_metrics` | Performance metrics | id, portfolio_id, beta, sharpe, drawdown |

### Views (Used by Hasura)

```sql
-- Portfolio summary with aggregates
SELECT * FROM portfolio_summary;

-- Notification summary for users
SELECT * FROM user_notification_summary;

-- Recommendation execution rates
SELECT * FROM recommendation_execution_rate;
```

---

## 🔌 GraphQL API Examples

### Query: Get Portfolio with Recommendations

```graphql
query GetPortfolio($portfolioId: uuid!) {
  portfolios_by_pk(id: $portfolioId) {
    id
    name
    total_value
    current_allocation
    holdings {
      ticker
      shares
      current_price
      allocation_pct
    }
    recommendations(where: {status: {_eq: "pending"}}) {
      id
      type
      priority
      title
      expected_benefit
      recommended_actions
    }
  }
}
```

### Subscription: Real-time Notifications

```graphql
subscription NotificationUpdates($userId: uuid!) {
  notifications(
    where: {user_id: {_eq: $userId}}
    order_by: {created_at: desc}
    limit: 10
  ) {
    id
    subject
    message
    priority
    created_at
  }
}
```

### Mutation: Execute Recommendation

```graphql
mutation ExecuteRecommendation($recommendationId: uuid!, $portfolioId: uuid!) {
  executeRecommendation(
    recommendation_id: $recommendationId
    portfolio_id: $portfolioId
  ) {
    order_id
    status
    tax_savings
    execution_time_ms
  }
}
```

---

## 🔔 Notification System

### Notification Types

| Type | Trigger | Channels | Priority |
|------|---------|----------|----------|
| `HIGH_PRIORITY_REC` | New HIGH recommendation | Email, Push | High |
| `EXECUTION_COMPLETE` | Rebalance finished | Email, Push | Normal |
| `TAX_OPPORTUNITY` | Harvestable losses found | Email | Normal |
| `REBALANCE_ALERT` | Drift exceeded threshold | Email, SMS | High |
| `MARKET_ALERT` | VIX spike, trend change | Push | Normal |

### Notification Channels

- **Email**: SMTP (Gmail, SendGrid, etc)
- **SMS**: Twilio API
- **Push**: Pusher real-time channels
- **In-app**: Stored in DB, delivered via GraphQL subscriptions

### Sending Notifications

```bash
# Via HTTP API
curl -X POST http://localhost:8081/notifications/send \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user-123",
    "portfolio_id": "port-456",
    "type": "HIGH_PRIORITY_REC",
    "priority": "high",
    "subject": "New Investment Recommendation",
    "message": "Tax-loss harvesting opportunity identified",
    "channels": ["email", "sms", "push"]
  }'

# Response
{
  "status": "queued",
  "notification_id": "notif-1729804800123",
  "timestamp": "2025-10-30T12:00:00Z"
}
```

### User Preferences

```graphql
mutation UpdateNotificationPrefs {
  update_notification_preferences_by_pk(
    pk_columns: {user_id: "uuid"}
    _set: {
      email_high_priority: true
      sms_critical_alerts: true
      push_notifications: false
      timezone: "America/New_York"
    }
  ) {
    id
  }
}
```

---

## 🐳 Deployment (Production)

### Using Kubernetes

```bash
# Create namespace
kubectl create namespace portfolio

# Create secrets
kubectl create secret generic db-credentials \
  --from-literal=password=$DB_PASSWORD \
  -n portfolio

# Create configmaps
kubectl create configmap portfolio-config \
  --from-env-file=.env \
  -n portfolio

# Deploy PostgreSQL (or use managed database)
kubectl apply -f k8s/postgres-statefulset.yml -n portfolio

# Deploy Hasura
kubectl apply -f k8s/hasura-deployment.yml -n portfolio

# Deploy Notification Service
kubectl apply -f k8s/notification-service-deployment.yml -n portfolio

# Apply ingress rules
kubectl apply -f k8s/ingress.yml -n portfolio

# Verify deployment
kubectl get pods -n portfolio
kubectl get svc -n portfolio
```

### Environment Variables (Production)

```env
# Database (use managed DB for production)
DB_HOST=production-db.example.com
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=<secure-password>
DB_NAME=portfolio_prod
DB_SSL_MODE=require

# Hasura
HASURA_GRAPHQL_ADMIN_SECRET=<generate-secure-key>
HASURA_GRAPHQL_JWT_SECRET='{"type":"RS256","key":"<your-public-key>"}'
HASURA_GRAPHQL_CORS_DOMAIN=https://app.example.com
HASURA_GRAPHQL_LOG_LEVEL=warn

# Email (SendGrid)
SMTP_HOST=smtp.sendgrid.net
SMTP_PORT=587
SMTP_USERNAME=apikey
SMTP_PASSWORD=<sendgrid-api-key>

# SMS (Twilio)
TWILIO_ACCOUNT_SID=<your-sid>
TWILIO_AUTH_TOKEN=<your-token>
TWILIO_PHONE_NUMBER=+1234567890

# Push Notifications (Pusher)
PUSHER_APP_ID=<app-id>
PUSHER_KEY=<key>
PUSHER_SECRET=<secret>
PUSHER_CLUSTER=mt1
```

---

## 📈 Monitoring & Observability

### Health Checks

```bash
# Check Hasura
curl -H "X-Hasura-Admin-Secret: $HASURA_ADMIN_SECRET" \
  http://localhost:8080/healthz

# Check Notification Service
curl http://localhost:8081/health

# Check Database
psql -h $DB_HOST -U $DB_USER -c "SELECT 1"
```

### Docker Logs

```bash
# Hasura logs
docker-compose logs -f hasura

# Notification service logs
docker-compose logs -f notification-service

# Database logs
docker-compose logs -f postgres

# All logs
docker-compose logs -f
```

### Metrics to Track

- **Recommendation acceptance rate**: `executed_count / total_count`
- **Tax savings generated**: Sum of `total_tax_savings` per period
- **Notification delivery rate**: `sent_count / created_count`
- **API response times**: Hasura query latency
- **Rebalance execution time**: `execution_time_ms` average
- **Database query performance**: Slow query log
- **Queue depth**: Current size of notification queue
- **Service uptime**: % of time services are healthy

---

## 🔐 Security Checklist

- [ ] Use HTTPS/TLS in production
- [ ] Generate secure JWT signing key
- [ ] Rotate admin secrets regularly
- [ ] Use environment variables for all secrets (never commit)
- [ ] Enable database SSL connections
- [ ] Use strong passwords for DB & admin accounts
- [ ] Implement rate limiting on API endpoints
- [ ] Set up audit logging (audit_logs table)
- [ ] Enable query timeout to prevent DoS
- [ ] Regularly update dependencies
- [ ] Use signed headers for Hasura event webhooks
- [ ] Implement API authentication (JWT)
- [ ] Enable CORS only for trusted domains
- [ ] Monitor for suspicious database activity
- [ ] Encrypt sensitive data in transit and at rest

---

## 🚨 Troubleshooting

### Notifications not sending?

```bash
# Check notification service logs
docker-compose logs notification-service

# Check deliveries table
docker-compose exec postgres psql -U portfolio portfolio_db -c \
  "SELECT * FROM notification_deliveries WHERE status = 'failed' LIMIT 10;"

# Retry failed deliveries
docker-compose exec postgres psql -U portfolio portfolio_db -c \
  "UPDATE notification_deliveries 
   SET status = 'pending' 
   WHERE status = 'failed' AND retry_count < max_retries;"
```

### Hasura not connecting to DB?

```bash
# Check Hasura logs
docker-compose logs hasura

# Verify database is running and healthy
docker-compose ps postgres

# Test database connection
docker-compose exec postgres psql -U portfolio -d portfolio_db -c "SELECT 1"

# Check database connection string
docker-compose exec hasura env | grep DATABASE
```

### GraphQL subscriptions not working?

- Ensure WebSocket is enabled in Hasura (default)
- Check browser console for WebSocket connection errors
- Verify JWT token is valid if using authentication
- Check Pusher credentials if using real-time push
- Verify Hasura admin secret is correctly set
- Check that subscription query is properly formatted

### Docker Compose issues?

```bash
# Verify Docker is running
docker ps

# Rebuild containers
docker-compose down
docker-compose build --no-cache
docker-compose up -d

# Check Docker network
docker network ls
docker network inspect portfolio_network

# Remove all containers and volumes (WARNING: deletes data)
docker-compose down -v
```

### Database schema not initialized?

```bash
# Manually run schema migration
docker-compose exec -T postgres psql -U portfolio -d portfolio_db -f \
  /docker-entrypoint-initdb.d/01-init.sql

# Verify schema
docker-compose exec postgres psql -U portfolio -d portfolio_db \
  -c "\dt" # Tables
```

---

## 📚 Additional Resources

- **Hasura Docs**: https://hasura.io/docs
- **GraphQL Best Practices**: https://graphql.org/learn/best-practices
- **PostgreSQL Documentation**: https://www.postgresql.org/docs
- **Twilio SMS API**: https://www.twilio.com/docs/sms
- **Pusher Real-time**: https://pusher.com/docs
- **Temporal Workflows**: https://temporal.io/docs

---

## 🎯 Next Steps

1. ✅ Run `docker-compose up -d` to start all services
2. ✅ Access Hasura console at http://localhost:8080
3. ✅ Test GraphQL queries using the built-in explorer
4. ✅ Create test user and portfolio
5. ✅ Configure notification preferences
6. ✅ Deploy Temporal workflow for rebalancing
7. ✅ Connect React frontend to GraphQL API
8. ✅ Set up monitoring & alerting
9. ✅ Configure production deployment

---

## 💡 Support

For issues or questions:
- Check logs: `docker-compose logs <service>`
- Review error messages in database
- Consult service documentation
- Check event triggers are firing in Hasura console
- Verify environment variables are correctly set
- Test connectivity between services

---

**Last Updated**: October 30, 2025
**Version**: 1.0.0
