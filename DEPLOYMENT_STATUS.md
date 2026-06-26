# ✅ SemLayer Local Development - Deployment Complete

## Status: READY FOR TESTING

**Deployed:** February 13, 2026

All services are running and responding correctly.

---

## 🎯 Quick Access

| Service | URL | Status |
|---------|-----|--------|
| **Frontend** | http://localhost:5173 | ✅ Running |
| **Backend API** | http://localhost:8080 | ✅ Running |
| **Auth Service** | http://localhost:8001 | ✅ Running |
| **Hasura GraphQL** | http://100.84.126.19:8080/v1/graphql | ✅ External |
| **PostgreSQL** | 100.84.126.19:5432 | ✅ External |

---

## 🔐 Test Credentials

```
Email:    test@example.com
Password: password123
Role:     global_ops
```

---

## 🚀 Running Services

### Backend & Auth (Docker)
```bash
cd /Users/eganpj/GitHub/semlayer
./docker-mac-local.sh
```

**Status:**
- Backend container: ✅ Running on port 8080
- Auth service container: ✅ Running on port 8001
- Network: semlayer (bridge)

### Frontend (Local Development)
```bash
cd /Users/eganpj/GitHub/semlayer/frontend
sh scripts/start-dev.sh
```

**Status:**
- Vite dev server: ✅ Running on port 5173
- Hot reload: ✅ Enabled
- React refresh: ✅ Active

---

## 📋 Deployment Architecture

**Hybrid Approach:**
- ✅ Backend & Auth: Containerized (Docker)
- ✅ Frontend: Local development server
- ✅ Databases: External (100.84.126.19)

**Why This Setup?**
- Backend in containers ensures consistency with production
- Frontend on host provides:
  - ✅ Proper native modules (macOS ARM64)
  - ✅ Fast npm install and builds
  - ✅ Better hot reload experience
  - ✅ Improved debugging

---

## 🔗 Service Endpoints

### Backend API (Port 8080)
```bash
# All endpoints on http://localhost:8080
GET    /api/tenants/all              # List tenants
GET    /api/glossary/semantic-terms  # Get semantic terms
GET    /api/bp-notifications/logs    # Get logs
POST   /api/glossary/edges           # Create edges
```

### Auth Service (Port 8001)
```bash
# Authentication endpoints
POST   http://localhost:8001/api/auth/login
GET    http://localhost:8001/api/auth/verify
POST   http://localhost:8001/api/auth/refresh
```

### Frontend (Port 5173)
```bash
http://localhost:5173/                # Main app
http://localhost:5173/glossary        # Glossary page
http://localhost:5173/login           # Login page
```

---

## 📊 Key Configuration

### Backend Environment (Docker)
```yaml
POSTGRES_HOST: 100.84.126.19
POSTGRES_PORT: 5432
POSTGRES_DB: alpha
HASURA_ENDPOINT: http://100.84.126.19:8080
PORT: 8080
ENVIRONMENT: dev
```

### Auth Service Environment (Docker)
```yaml
AUTH_SERVICE_PORT: 8001
POSTGRES_HOST: 100.84.126.19
POSTGRES_PORT: 5432
POSTGRES_DB: alpha
JWT_SECRET: dev-jwt-secret-key-change-in-production
ALLOWED_ORIGINS: http://localhost:*
```

### Frontend Environment (Local)
```bash
VITE_API_BASE_URL: http://localhost:8080
VITE_AUTH_SERVICE_URL: http://localhost:8001
VITE_GRAPHQL_ENDPOINT: http://100.84.126.19:8080/v1/graphql
```

---

## ✨ Features

### Backend API
- ✅ RESTful API on port 8080
- ✅ Health checks every 30 seconds
- ✅ Integrated with Hasura GraphQL
- ✅ Multi-tenant support
- ✅ Semantic layer management

### Auth Service  
- ✅ JWT-based authentication
- ✅ Multi-tenant user management
- ✅ CORS configured for localhost
- ✅ Refresh token support
- ✅ Audit logging

### Frontend
- ✅ React 18 + TypeScript
- ✅ Vite 5.4.21 dev server
- ✅ Hot module replacement (HMR)
- ✅ GraphQL integration
- ✅ Responsive UI

---

## 🛠️ Common Tasks

### View Backend Logs
```bash
./docker-mac-local.sh logs backend
```

### View Auth Service Logs
```bash
./docker-mac-local.sh logs auth-service
```

### Restart All Services
```bash
./docker-mac-local.sh down && sleep 2 && ./docker-mac-local.sh up
```

### Stop Services
```bash
./docker-mac-local.sh down
```

### Development: Edit Frontend Code
```bash
cd frontend
# Edit any .tsx, .ts, or .css files
# Browser auto-updates with hot reload!
```

### Development: Edit Backend Code
```bash
cd backend
# Edit Go files, then rebuild:
docker compose -f docker-compose.mac-local.yml build backend
docker compose -f docker-compose.mac-local.yml restart backend
```

---

## 📝 Next Steps

1. **Login to Frontend**
   - Open http://localhost:5173
   - Use credentials: test@example.com / password123

2. **Navigate to Glossary**
   - Click "Glossary" in the UI
   - View semantic terms

3. **Manage Data**
   - Select uisce tenant
   - Choose northwinds datasource
   - Create semantic objects

---

## 🐛 Troubleshooting

**Frontend not loading?**
```bash
# Check if Vite is running
lsof -i :5173

# Restart frontend
cd frontend && sh scripts/start-dev.sh
```

**Backend returning errors?**
```bash
# Check backend logs
./docker-mac-local.sh logs backend

# Verify database connectivity
nc -zv 100.84.126.19 5432
```

**Port already in use?**
```bash
# Kill process on port
lsof -i :5173
kill -9 <PID>
```

---

## 📞 Support

For detailed deployment information, see: [DOCKER_LOCAL_DEPLOYMENT.md](DOCKER_LOCAL_DEPLOYMENT.md)

For troubleshooting guide, see: [DOCKER_LOCAL_DEPLOYMENT.md#troubleshooting](DOCKER_LOCAL_DEPLOYMENT.md#troubleshooting)

---

**Deployment Date:** February 13, 2026  
**Status:** ✅ PRODUCTION READY FOR LOCAL DEVELOPMENT  
**Last Updated:** $(date)
