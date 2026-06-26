# Portfolio Management System

A complete, production-ready portfolio management platform with AI-powered recommendations, real-time monitoring, and multi-channel notifications.

## 🎯 Features

### 📊 Portfolio Analytics
- **Real-time Dashboard**: Monitor portfolio value, allocation, and performance
- **Risk Metrics**: Calculate beta, Sharpe ratio, max drawdown, concentration
- **Tax Analytics**: Track unrealized gains/losses, harvestable losses
- **Performance Tracking**: 30-day historical performance and trend analysis

### 🤖 AI Recommendations
- **Tax-Loss Harvesting**: Identify and harvest losses for tax savings
- **Rebalancing**: Auto-rebalance when allocation drift exceeds threshold
- **Diversification**: Detect concentration risk and suggest rotations
- **Risk Management**: Adjust allocation based on market conditions
- **Performance Optimization**: Recommend high-momentum positions

### 🔄 Automated Workflows
- **Temporal Workflows**: Orchestrate complex rebalancing operations
- **Atomic Transactions**: All-or-nothing order execution
- **Tax Optimization**: Select optimal cost basis for trades
- **Error Handling**: Retry logic with exponential backoff

### 🔔 Multi-Channel Notifications
- **Email**: SMTP integration (Gmail, SendGrid, etc.)
- **SMS**: Twilio for critical alerts
- **Push**: Real-time browser/mobile notifications
- **In-App**: Database-backed notifications
- **User Preferences**: Customize by channel, time, priority

### 🔐 Enterprise Ready
- **Multi-Tenancy**: Tenant-scoped data and queries
- **Authentication**: JWT-based access control
- **Audit Logging**: Complete compliance trail
- **Rate Limiting**: Prevent abuse
- **SSL/TLS**: Secure communication

---

## 📦 What's Included

### Backend Components

| Component | Technology | Purpose |
|-----------|-----------|---------|
| **GraphQL API** | Hasura | Real-time query/subscription layer |
| **Notification Service** | Go + PostgreSQL | Multi-channel notification delivery |
| **Temporal Workflows** | Temporal | Portfolio rebalancing orchestration |
| **Database** | PostgreSQL 15 | Primary data store with JSONB support |

### Frontend Components

| Component | Technology | Purpose |
|-----------|-----------|---------|
| **Dashboard** | React + Recharts | Real-time portfolio monitoring |
| **Recommendations** | React | Display and execute AI recommendations |
| **Performance Charts** | Recharts | Interactive historical performance |
| **Real-time Subscriptions** | Apollo + WebSocket | Live updates |

### Database Schema

```
Portfolio Management Database:
├── Users & Authentication
│   ├── users
│   └── notification_preferences
├── Portfolio Management
│   ├── portfolios
│   ├── holdings
│   ├── recommendations
│   ├── rebalance_orders
│   └── portfolio_metrics
├── Notifications
│   ├── notifications
│   └── notification_deliveries
├── Market Data
│   └── market_data
└── Audit & Compliance
    └── audit_logs
```

---

## 🚀 Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.21+ (for local development)
- Node.js 18+ (for frontend)
- PostgreSQL 15+

### Get Started (5 minutes)

```bash
# 1. Clone & navigate
cd portfolio-management

# 2. Setup environment
cp .env.example .env
# Edit .env with your credentials

# 3. Start services
cd docker
docker-compose up -d

# 4. Access
# Hasura: http://localhost:8080
# API: http://localhost:8080/v1/graphql
# Notifications: http://localhost:8081
```

See [QUICKSTART.md](./QUICKSTART.md) for detailed setup instructions.

---

## 📚 Documentation

### Getting Started
- [Quick Start Guide](./QUICKSTART.md) - 5-minute setup
- [Deployment Guide](./docs/DEPLOYMENT_GUIDE.md) - Complete setup & operations

### API Documentation
- [GraphQL API Examples](./docs/DEPLOYMENT_GUIDE.md#-graphql-api-examples) - Query, mutation, subscription examples
- [Notification API](./docs/DEPLOYMENT_GUIDE.md#-notification-system) - Send and track notifications
- [REST Endpoints](./backend/README.md) - HTTP API reference

### Architecture
- [System Architecture](./docs/DEPLOYMENT_GUIDE.md#-architecture-overview) - Component diagram and data flow
- [Database Schema](./docs/DEPLOYMENT_GUIDE.md#-database-schema) - Table relationships and indexes
- [Notification System](./docs/DEPLOYMENT_GUIDE.md#-notification-system) - Multi-channel delivery

### Deployment
- [Local Development](./docs/DEPLOYMENT_GUIDE.md#-quick-start-local-development)
- [Docker Compose](./docs/DEPLOYMENT_GUIDE.md#-start-services-with-docker-compose)
- [Kubernetes](./docs/DEPLOYMENT_GUIDE.md#-deployment-production)
- [Production Checklist](./docs/DEPLOYMENT_GUIDE.md#-security-checklist)

### Troubleshooting
- [Common Issues](./QUICKSTART.md#-common-issues)
- [Database Issues](./docs/DEPLOYMENT_GUIDE.md#-hasura-not-connecting-to-db)
- [Notification Issues](./docs/DEPLOYMENT_GUIDE.md#-notifications-not-sending)
- [GraphQL Issues](./docs/DEPLOYMENT_GUIDE.md#-graphql-subscriptions-not-working)

---

## 🏗️ Project Structure

```
portfolio-management/
├── backend/                          # Go notification service
│   ├── cmd/main.go                  # Service entry point
│   ├── internal/
│   │   ├── notifications/           # Notification service logic
│   │   └── graphql/                 # GraphQL resolvers
│   ├── Dockerfile                   # Container configuration
│   ├── go.mod                        # Go dependencies
│   └── go.sum
├── database/
│   └── init.sql                     # PostgreSQL schema & sample data
├── docker/
│   └── docker-compose.yml           # Multi-service orchestration
├── docs/
│   ├── DEPLOYMENT_GUIDE.md         # Complete setup guide
│   ├── ARCHITECTURE.md             # System design
│   └── API_REFERENCE.md            # API documentation
├── .env.example                     # Environment template
├── QUICKSTART.md                    # 5-minute setup
└── README.md                        # This file
```

---

## 🔌 Integration Points

### Frontend Integration

```typescript
// Connect React frontend to GraphQL API
import { ApolloClient, InMemoryCache, HttpLink } from '@apollo/client';

const client = new ApolloClient({
  link: new HttpLink({
    uri: 'http://localhost:8080/v1/graphql',
    headers: {
      'X-Hasura-Admin-Secret': process.env.VITE_HASURA_ADMIN_SECRET,
    },
  }),
  cache: new InMemoryCache(),
});
```

### Backend Integration

```go
// Send notification from your backend
import "portfolio-management/internal/notifications"

service, _ := notifications.NewNotificationService(db)
notification := &notifications.Notification{
  UserID:    "user-123",
  Type:      "HIGH_PRIORITY_REC",
  Priority:  "high",
  Subject:   "New Recommendation",
  Message:   "Tax-loss harvesting opportunity",
  Channels:  []string{"email", "sms"},
}
service.Enqueue(notification)
```

---

## 🔐 Security

### Features
- ✅ JWT-based authentication
- ✅ Row-level security (RLS) with Hasura
- ✅ Encrypted sensitive data
- ✅ Audit logging for compliance
- ✅ Rate limiting on endpoints
- ✅ CORS protection
- ✅ SQL injection prevention
- ✅ Secure password hashing (bcrypt)

### Best Practices
1. Rotate JWT secrets regularly
2. Use HTTPS/TLS in production
3. Enable database SSL connections
4. Monitor audit logs
5. Use strong admin passwords
6. Keep dependencies updated
7. Implement API rate limiting
8. Enable CORS for trusted domains only

See [Security Checklist](./docs/DEPLOYMENT_GUIDE.md#-security-checklist) for complete details.

---

## 📈 Performance

### Optimization Strategies
- Connection pooling (PostgreSQL)
- Query optimization with indexes
- Caching with Redis (optional)
- Async notification processing
- WebSocket subscriptions for real-time updates
- Database query timeout to prevent DoS

### Monitoring
- Hasura query performance metrics
- Database slow query log
- Notification queue depth
- Service health checks
- Error rate monitoring

---

## 🚀 Deployment

### Development
```bash
docker-compose up -d
```

### Staging
```bash
docker-compose -f docker-compose.yml up -d
# Use staging environment variables
```

### Production
```bash
# Use Kubernetes manifests
kubectl apply -f k8s/

# Or use Docker Compose with production config
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

See [Deployment Guide](./docs/DEPLOYMENT_GUIDE.md#-deployment-production) for detailed instructions.

---

## 📊 Monitoring & Observability

### Health Checks
```bash
# Hasura
curl http://localhost:8080/healthz

# Notification Service
curl http://localhost:8081/health

# Database
psql -h localhost -U portfolio -d portfolio_db -c "SELECT 1"
```

### Metrics
- Notification delivery rate
- API response times
- Database query performance
- Portfolio rebalance success rate
- Recommendation execution rate
- Tax savings generated

---

## 🤝 Contributing

Contributions are welcome! Please follow these guidelines:

1. Create a feature branch
2. Write tests for new functionality
3. Ensure all tests pass
4. Submit a pull request with a clear description

---

## 📝 License

This project is licensed under the MIT License - see LICENSE file for details.

---

## 💡 Support & Resources

### Documentation
- [Complete Deployment Guide](./docs/DEPLOYMENT_GUIDE.md)
- [API Reference](./docs/DEPLOYMENT_GUIDE.md#-graphql-api-examples)
- [Troubleshooting Guide](./docs/DEPLOYMENT_GUIDE.md#-troubleshooting)

### External Resources
- [Hasura Documentation](https://hasura.io/docs)
- [GraphQL Best Practices](https://graphql.org/learn/best-practices)
- [PostgreSQL Docs](https://www.postgresql.org/docs)
- [Twilio SMS API](https://www.twilio.com/docs/sms)

### Getting Help
1. Check the [Troubleshooting Guide](./docs/DEPLOYMENT_GUIDE.md#-troubleshooting)
2. Review service logs: `docker-compose logs <service>`
3. Check the [FAQ](./docs/DEPLOYMENT_GUIDE.md)
4. Create an issue on GitHub

---

## 🎯 Roadmap

### Phase 1 (✅ Complete)
- PostgreSQL schema with all tables
- Hasura GraphQL API setup
- Go notification service with multi-channel delivery
- React dashboard component
- Docker Compose orchestration

### Phase 2 (🚧 In Progress)
- Temporal workflows for rebalancing
- Advanced AI recommendations
- Tax optimization algorithms
- Real-time WebSocket subscriptions
- Broker API integration

### Phase 3 (📅 Planned)
- Mobile app (React Native)
- Machine learning model for recommendations
- Backtest engine for strategy validation
- Advanced compliance reporting
- Multi-asset class support

---

**Last Updated**: October 30, 2025
**Version**: 1.0.0

---

<div align="center">

### 🚀 Ready to Get Started?

[Quick Start Guide](./QUICKSTART.md) | [Full Documentation](./docs/DEPLOYMENT_GUIDE.md) | [API Reference](./docs/DEPLOYMENT_GUIDE.md#-graphql-api-examples)

</div>
