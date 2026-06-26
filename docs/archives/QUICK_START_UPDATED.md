# ⚡ UPDATED - API Gateway Quick Start (November 4, 2025)

## What's New

This is the UPDATED quick start after fixing the API Gateway Docker build issues.

## Run This First

```bash
cd /Users/eganpj/GitHub/semlayer
./start-docker.sh
```

This does everything automatically:
1. ✅ Checks PostgreSQL is running
2. ✅ Creates .env with sensible defaults
3. ✅ Builds all Docker images
4. ✅ Starts all services
5. ✅ Shows you the access points

## Or Manual Steps

```bash
# Ensure PostgreSQL is running
psql postgres://postgres:postgres@localhost:5432/alpha -c "SELECT 1"

# Start all services
docker compose up -d

# Wait 30 seconds for them to stabilize
sleep 30

# Check status
docker compose ps

# Verify API Gateway is working
curl http://localhost:8001/health
```

## Access Your Services

- **API Gateway**: http://localhost:8001
- **Frontend**: http://localhost:5173  
- **Backend**: http://localhost:8080
- **Grafana**: http://localhost:3000
- **RabbitMQ**: http://localhost:15672 (guest:guest)

## What Was Fixed

✅ API Gateway Docker image builds correctly
✅ No more confusing bcrypt hashes on startup  
✅ All services run in Docker (PostgreSQL only local)
✅ Proper environment variable configuration
✅ Working `docker compose` commands

## Common Issues

**"Connection refused on :8001"**
- Wait 30+ seconds, services are still starting
- Or check logs: `docker compose logs api-gateway`

**"PostgreSQL not accessible"**
- Start PostgreSQL: `brew services start postgresql@14`
- Or verify: `psql -U postgres`

**"Temporal keeps restarting"**
- Known issue with missing config
- API Gateway handles this gracefully (times out after 120 sec)
- Doesn't block other functionality

**"Port already in use"**
- Edit `.env` and change port numbers
- Or kill the process using the port

## See Also

- `API_GATEWAY_STARTUP_GUIDE.md` - Detailed setup guide
- `DOCKER_SETUP.md` - Architecture documentation
- `API_GATEWAY_COMPLETE_STATUS.md` - Full technical report
- `agents.md` - Tenant scoping reference

## One Command to Rule Them All

```bash
docker compose up -d && sleep 5 && docker compose ps
```

---

**Everything working?** You're done! 🎉  
**Something broken?** Check the logs: `docker compose logs -f`
