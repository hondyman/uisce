# 🚀 QUICK START - Port Allocation System

## START EVERYTHING (Copy & Paste)

```bash
# In semlayer root directory
bash scripts/validate-ports.sh
docker compose --env-file .env.ports -f docker-compose.dev.simple.yml up -d

# In separate terminal, inside frontend directory  
cd frontend && npm run dev
```

## ACCESS YOUR APP

```
Frontend:        http://localhost:5173
REST API:        http://localhost:8080
GraphQL:         http://localhost:8888/v1/graphql
RabbitMQ UI:     http://localhost:15672 (guest/guest)
Temporal UI:     http://localhost:8088
```

## WHERE PORTS ARE DEFINED

| What | File | Why | Edit When |
|------|------|-----|-----------|
| Docker ports | `.env.ports` | Single source of truth | Changing service port |
| Frontend endpoints | `frontend/.env` | Vite build time | Changing port for Vite |
| Port validation | `scripts/validate-ports.sh` | Check for duplicates | Adding service |

## CHANGE A PORT (Step by Step)

**Example: Change Hasura from 8888 to 8889**

```bash
# 1. Edit .env.ports
sed -i '' 's/PORT_HASURA_GRAPHQL=8888/PORT_HASURA_GRAPHQL=8889/' .env.ports

# 2. Edit frontend/.env
sed -i '' 's/localhost:8888/localhost:8889/' frontend/.env

# 3. Validate
bash scripts/validate-ports.sh

# 4. Restart services
docker compose --env-file .env.ports -f docker-compose.dev.simple.yml down
docker compose --env-file .env.ports -f docker-compose.dev.simple.yml up -d

# 5. Restart frontend (Ctrl+C and)
cd frontend && npm run dev
```

## KEY POINTS

✅ **Always** use `--env-file .env.ports` with docker compose  
✅ **Always** hardcode values in `frontend/.env` (NOT `${VAR}`)  
✅ **Always** run `bash scripts/validate-ports.sh` after changes  
✅ **All 10 ports are unique** - verified by script  

## TROUBLESHOOTING

### Frontend says "ERR_CONNECTION_REFUSED"
- Check: `docker compose --env-file .env.ports -f docker-compose.dev.simple.yml ps`
- Restart: `cd frontend && npm run dev`

### Port already in use
- Kill: `lsof -i :8888` (replace with your port)
- Restart: `docker compose --env-file .env.ports -f docker-compose.dev.simple.yml down && up -d`

### Validation fails
- Run: `bash scripts/validate-ports.sh`
- Look for duplicate port numbers

## PORT REFERENCE

```
8080 = Backend API          (REST)
8081 = Fabric Builder       (Service)
8888 = Hasura GraphQL       (GraphQL)
5672 = RabbitMQ             (AMQP)
15672= RabbitMQ UI          (Management)
7233 = Temporal             (Workflow)
8088 = Temporal UI          (Admin)
5173 = Vite Dev Server      (Frontend)
5432 = PostgreSQL           (Database)
```

## THAT'S IT!

Your system is now:
- ✅ Permanent (no more port changes)
- ✅ Centralized (one config file)
- ✅ Automatic (variable substitution)
- ✅ Validated (script checks everything)

**Go build something awesome!** 🎉
