# ✅ Docker Compose - Ready to Use

## Current Status: ALL SYSTEMS GO! 🚀

Your Docker Compose environment is **fully running and tested**.

## What's Running

```
✓ Hasura GraphQL        → http://localhost:8888
✓ RabbitMQ              → amqp://localhost:5672
✓ RabbitMQ Management   → http://localhost:15672 (guest/guest)
✓ Temporal Server       → localhost:7233
✓ Temporal UI           → http://localhost:8088
✓ Frontend Dev Server   → http://localhost:5173
```

## Quick Commands

```bash
# Start services
docker compose -f docker-compose.dev.simple.yml up -d

# Check status
./scripts/check-services.sh

# View logs
docker compose -f docker-compose.dev.simple.yml logs -f

# Stop services
docker compose -f docker-compose.dev.simple.yml down

# Restart everything
docker compose -f docker-compose.dev.simple.yml restart
```

## Next Steps

### 1. Update Your Shell RC File (Optional but Recommended)

Add these helpful aliases to your `~/.zshrc`:

```bash
source /Users/eganpj/GitHub/semlayer/.docker-aliases.sh
```

Then you can use:
```bash
dcup      # Start
dcdown    # Stop
dcps      # Status
dclogs    # View logs
dcstatus  # Check connectivity
```

### 2. Start Your Backend

In a new terminal:
```bash
cd /Users/eganpj/GitHub/semlayer/services/fabric-builder
go run main.go
```

The backend will run on **http://localhost:8080** and will be properly configured to use your Docker services.

### 3. Start Your Frontend

In another terminal:
```bash
cd /Users/eganpj/GitHub/semlayer/frontend
npm run dev
```

The frontend runs on **http://localhost:5173** with automatic reload.

### 4. Open the App

Visit **http://localhost:5173** in your browser and you're ready to go!

## Architecture Overview

```
┌─────────────────────────────────────────┐
│         Your Browser (5173)             │
│      http://localhost:5173              │
└──────────────┬──────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────┐
│    Frontend Dev Server (Vite)           │
│    Auto-reload on file changes          │
└──────────────┬──────────────────────────┘
               │
               ├─────────────► http://localhost:8080 (Backend API)
               │
               └─────────────► http://localhost:8080/v1/graphql (Hasura GraphQL)
                                     │
                                     └─── PostgreSQL (host machine :5432)
                                           RabbitMQ (:5672)
                                           Temporal (:7233)
```

## API Endpoints Reference

### GraphQL
```
POST http://localhost:8888/v1/graphql
```

### REST APIs
```
GET    http://localhost:8080/api/business-entities
GET    http://localhost:8080/api/business-entities/{id}
POST   http://localhost:8080/api/relationships/discover
POST   http://localhost:8080/api/relationships/existing
POST   http://localhost:8080/api/relationships/apply
```

### Admin Consoles
```
Hasura            → http://localhost:8888/console
RabbitMQ          → http://localhost:15672 (guest/guest)
Temporal UI       → http://localhost:8088
```

## Environment Configuration

Your `.env.local` files are already set up:

**Root `.env.local`**:
```
VITE_API_BASE_URL=http://localhost:8080
VITE_BACKEND_TARGET=http://localhost:8080
VITE_GRAPHQL_ENDPOINT=http://localhost:8080/v1/graphql
JWT_SECRET=development-secret-key
```

**Frontend `.env.local`**:
```
VITE_USE_PROXY=false
VITE_BACKEND_TARGET=http://localhost:8080
VITE_API_BASE_URL=http://localhost:8080
VITE_GRAPHQL_ENDPOINT=http://localhost:8080/v1/graphql
```

## Tenant Configuration

Default tenant is pre-configured:
- **Tenant ID**: `910638ba-a459-4a3f-bb2d-78391b0595f6`
- **Datasource ID**: `982aef38-418f-46dc-acd0-35fe8f3b97b0`
- **Already seeded in localStorage** by `setupTenantFetch.ts`

## Troubleshooting

### Services won't start?
```bash
# Clean everything and restart
docker compose -f docker-compose.dev.simple.yml down --remove-orphans
docker compose -f docker-compose.dev.simple.yml up -d
```

### Getting 404 errors on API calls?
```bash
# Make sure your .env files have correct endpoints
cat frontend/.env.local
cat .env.local
```

### PostgreSQL connection issues?
```bash
# Verify PostgreSQL is running on your host
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable -c "SELECT 1"
```

### Check service health
```bash
./scripts/check-services.sh
```

## Files Modified/Created

- ✅ `docker-compose.dev.simple.yml` - Simplified, working compose file
- ✅ `scripts/check-services.sh` - Service status checker script
- ✅ `DOCKER_COMPOSE_SETUP.md` - Comprehensive setup guide
- ✅ `.docker-aliases.sh` - Helpful shell aliases
- ✅ `frontend/.env.local` - Frontend environment configuration
- ✅ `.env.local` - Root environment configuration

## You're All Set! 🎉

Your complete development environment is ready:

1. ✅ Docker Compose services running
2. ✅ All infrastructure configured
3. ✅ Environment variables set
4. ✅ Frontend configured to use backend on :8080
5. ✅ Tenant scope pre-seeded

**Start your backend and frontend services and you're ready to develop!**
