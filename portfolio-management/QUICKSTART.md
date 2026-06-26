# Portfolio Management System - Quick Start

## 🚀 Get Started in 5 Minutes

### 1. Copy Environment File
```bash
cd portfolio-management
cp .env.example .env
```

### 2. Start Docker Services
```bash
cd docker
docker-compose up -d
```

### 3. Verify Services
```bash
docker-compose ps

# Expected output (all should be "Up"):
# NAME                        STATUS
# portfolio_postgres          Up (healthy)
# portfolio_hasura            Up (healthy)
# portfolio_notification_service  Up (healthy)
```

### 4. Access Services

| Service | URL | Credentials |
|---------|-----|-------------|
| **Hasura Console** | http://localhost:8080 | Admin Secret: value from `.env` |
| **GraphQL API** | http://localhost:8080/v1/graphql | Use Hasura Console |
| **Notification Health** | http://localhost:8081/health | No auth needed |
| **Database** | localhost:5432 | User: `portfolio`, Password: value from `.env` |

### 5. Test Your Setup

#### Test GraphQL Query
```bash
curl -X POST http://localhost:8080/v1/graphql \
  -H "Content-Type: application/json" \
  -H "X-Hasura-Admin-Secret: admin_secret_key" \
  -d '{
    "query": "{ users { id email full_name } }"
  }'
```

#### Test Notification Service
```bash
curl http://localhost:8081/health
```

#### Send a Test Notification
```bash
curl -X POST http://localhost:8081/notifications/send \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "test-user",
    "type": "TEST",
    "priority": "normal",
    "subject": "Test Notification",
    "message": "This is a test notification",
    "channels": ["in_app"]
  }'
```

---

## 🐛 Common Issues

### Services won't start
```bash
# Check if ports are already in use
lsof -i :5432
lsof -i :8080
lsof -i :8081

# Stop conflicting services
kill -9 <PID>

# Restart Docker Compose
docker-compose down
docker-compose up -d
```

### Database not initializing
```bash
# Check logs
docker-compose logs postgres

# Run init script manually
docker-compose exec postgres psql -U portfolio -d portfolio_db -f \
  /docker-entrypoint-initdb.d/01-init.sql
```

### Hasura not connecting to database
```bash
# Check Hasura logs
docker-compose logs hasura

# Verify database connection
docker-compose exec postgres psql -U portfolio -d portfolio_db -c "SELECT 1"
```

---

## 📱 Next Steps

1. **Integrate Frontend**: Add the portfolio dashboard component to your React app
2. **Configure Email**: Update SMTP credentials for email notifications
3. **Setup Twilio**: Add SMS notification capability
4. **Deploy to Production**: Use Kubernetes manifests in `k8s/` folder
5. **Monitor**: Set up Prometheus/Grafana for metrics

---

## ✅ Checklist

- [ ] Environment variables configured (`.env`)
- [ ] Docker services running (`docker-compose ps`)
- [ ] Hasura console accessible (http://localhost:8080)
- [ ] GraphQL API responding
- [ ] Database initialized with sample data
- [ ] Notification service healthy
- [ ] Ready for frontend integration

---

## 📞 Support

**Having issues?** Check the full [DEPLOYMENT_GUIDE.md](./DEPLOYMENT_GUIDE.md) for detailed troubleshooting.
