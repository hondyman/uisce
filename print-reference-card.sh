#!/bin/bash
# =============================================================================
# SEMLAYER DISTRIBUTED PLATFORM - REFERENCE CARD
# =============================================================================
# Quick reference for common commands and configuration
# Print or save for quick access
# =============================================================================

cat << 'EOF'
╔═══════════════════════════════════════════════════════════════════════════╗
║              SEMLAYER DISTRIBUTED PLATFORM - REFERENCE CARD               ║
╚═══════════════════════════════════════════════════════════════════════════╝

┌───────────────────────────────────────────────────────────────────────────┐
│ ARCHITECTURE                                                              │
├───────────────────────────────────────────────────────────────────────────┤
│                                                                           │
│  MACBOOK PRO                          REMOTE (100.84.126.19)             │
│  ├─ Frontend (npm dev) :5173          ├─ PostgreSQL :5432                │
│  └─ Backend (Docker) :8080    ◄───────┤─ Hasura :8085                    │
│                               TCP/IP  ├─ Redpanda :19092                 │
│                                       ├─ Temporal :7233                  │
│                                       ├─ Debezium :8083                  │
│                                       ├─ Trino :8094                     │
│                                       └─ MinIO :9010/9011                │
│                                                                           │
└───────────────────────────────────────────────────────────────────────────┘

┌───────────────────────────────────────────────────────────────────────────┐
│ STARTUP                                                                   │
├───────────────────────────────────────────────────────────────────────────┤
│                                                                           │
│  1. Start Remote Services (on 100.84.126.19)                             │
│     docker compose -f docker-compose.remote.yml up -d                    │
│                                                                           │
│  2. Test Connectivity (on MacBook)                                       │
│     ./test-distributed-connectivity.sh                                   │
│                                                                           │
│  3. Start Backend (on MacBook)                                           │
│     ./start-distributed-platform.sh                                      │
│                                                                           │
│  4. Start Frontend (new terminal)                                        │
│     cd frontend && npm run dev                                           │
│                                                                           │
│  5. Open Browser                                                         │
│     http://localhost:5173                                               │
│                                                                           │
└───────────────────────────────────────────────────────────────────────────┘

┌───────────────────────────────────────────────────────────────────────────┐
│ SERVICE ENDPOINTS                                                         │
├───────────────────────────────────────────────────────────────────────────┤
│                                                                           │
│  LOCAL (MacBook)                                                         │
│  ├─ Frontend                 http://localhost:5173                       │
│  └─ Backend API              http://localhost:8080                       │
│                                                                           │
│  REMOTE (100.84.126.19)                                                  │
│  ├─ Hasura GraphQL           http://100.84.126.19:8085                   │
│  ├─ Redpanda Console         http://100.84.126.19:8096                   │
│  ├─ Temporal UI              http://100.84.126.19:8088                   │
│  ├─ Trino                    http://100.84.126.19:8094                   │
│  └─ MinIO Console            http://100.84.126.19:9011                   │
│                                                                           │
└───────────────────────────────────────────────────────────────────────────┘

┌───────────────────────────────────────────────────────────────────────────┐
│ ESSENTIAL COMMANDS                                                        │
├───────────────────────────────────────────────────────────────────────────┤
│                                                                           │
│  Backend Status                                                          │
│  └─ docker ps | grep backend                                            │
│                                                                           │
│  Backend Logs                                                            │
│  └─ docker compose -f docker-compose.mac-distributed.yml logs -f backend│
│                                                                           │
│  Backend Health                                                          │
│  └─ curl http://localhost:8080/health                                   │
│                                                                           │
│  Stop Everything                                                         │
│  └─ docker compose -f docker-compose.mac-distributed.yml down           │
│                                                                           │
│  Restart Backend                                                         │
│  └─ docker compose -f docker-compose.mac-distributed.yml restart backend│
│                                                                           │
│  Remote Services Status                                                  │
│  └─ ssh user@100.84.126.19                                              │
│     docker compose -f docker-compose.remote.yml ps                      │
│                                                                           │
│  Test Connectivity                                                       │
│  └─ ./test-distributed-connectivity.sh                                  │
│                                                                           │
└───────────────────────────────────────────────────────────────────────────┘

┌───────────────────────────────────────────────────────────────────────────┐
│ ENVIRONMENT CONFIGURATION (.env)                                          │
├───────────────────────────────────────────────────────────────────────────┤
│                                                                           │
│  Database                                                                │
│  DB_HOST=100.84.126.19                                                   │
│  DATABASE_URL=postgresql://postgres:postgres@100.84.126.19:5432/alpha   │
│                                                                           │
│  Hasura                                                                  │
│  HASURA_URL=http://100.84.126.19:8085                                    │
│  HASURA_ADMIN_SECRET=myadminsecret                                       │
│                                                                           │
│  Kafka/Redpanda                                                          │
│  KAFKA_BROKERS=100.84.126.19:19092                                       │
│  KAFKA_SCHEMA_REGISTRY=http://100.84.126.19:8081                         │
│                                                                           │
│  Temporal                                                                │
│  TEMPORAL_HOSTPORT=100.84.126.19:7233                                    │
│                                                                           │
│  Security                                                                │
│  JWT_SECRET=dev-jwt-secret-key-change-in-production                      │
│                                                                           │
│  CORS                                                                    │
│  ALLOWED_ORIGINS=http://localhost:5173,http://127.0.0.1:5173            │
│                                                                           │
└───────────────────────────────────────────────────────────────────────────┘

┌───────────────────────────────────────────────────────────────────────────┐
│ TROUBLESHOOTING                                                           │
├───────────────────────────────────────────────────────────────────────────┤
│                                                                           │
│  Backend Won't Start                                                     │
│  1. docker ps                    (Check if running)                      │
│  2. docker logs <ID>             (View error logs)                       │
│  3. Verify PostgreSQL accessible: psql postgresql://..@100.84.126.19    │
│                                                                           │
│  Frontend Can't Reach Backend                                            │
│  1. curl http://localhost:8080/health                                    │
│  2. Check browser console for errors (F12)                               │
│  3. Verify CORS in .env: ALLOWED_ORIGINS                                │
│                                                                           │
│  Remote Services Not Reachable                                           │
│  1. ping 100.84.126.19           (Check network)                         │
│  2. Verify remote services running                                       │
│  3. Check firewall allows ports                                          │
│                                                                           │
│  Port Already in Use                                                     │
│  1. lsof -i :8080                (Find process)                          │
│  2. kill -9 <PID>                (Terminate)                             │
│                                                                           │
│  Docker Not Running                                                      │
│  1. Start Docker Desktop                                                 │
│  2. docker ps                    (Verify running)                        │
│                                                                           │
└───────────────────────────────────────────────────────────────────────────┘

┌───────────────────────────────────────────────────────────────────────────┐
│ TEST CONNECTIVITY                                                         │
├───────────────────────────────────────────────────────────────────────────┤
│                                                                           │
│  Run Comprehensive Test                                                  │
│  ./test-distributed-connectivity.sh                                      │
│                                                                           │
│  Individual Tests                                                        │
│  nc -zv 100.84.126.19 5432        (PostgreSQL)                           │
│  nc -zv 100.84.126.19 8085        (Hasura)                               │
│  nc -zv 100.84.126.19 19092       (Kafka)                                │
│  nc -zv 100.84.126.19 7233        (Temporal)                             │
│                                                                           │
│  Network Latency                                                         │
│  ping 100.84.126.19               (Check latency)                        │
│                                                                           │
└───────────────────────────────────────────────────────────────────────────┘

┌───────────────────────────────────────────────────────────────────────────┐
│ PERFORMANCE TIPS                                                          │
├───────────────────────────────────────────────────────────────────────────┤
│                                                                           │
│  1. Increase Docker Resources                                            │
│     Docker Desktop → Settings → Resources → Increase CPU/Memory          │
│                                                                           │
│  2. Check Network Latency                                                │
│     ping 100.84.126.19  (should be <10ms local, <50ms over internet)     │
│                                                                           │
│  3. Monitor Resource Usage                                               │
│     docker stats semlayer-backend                                        │
│                                                                           │
│  4. Enable Connection Pooling                                            │
│     Check backend database configuration                                 │
│                                                                           │
└───────────────────────────────────────────────────────────────────────────┘

┌───────────────────────────────────────────────────────────────────────────┐
│ FILES REFERENCE                                                           │
├───────────────────────────────────────────────────────────────────────────┤
│                                                                           │
│  docker-compose.mac-distributed.yml     Backend configuration            │
│  start-distributed-platform.sh           Automated startup script ⭐      │
│  test-distributed-connectivity.sh        Connectivity test               │
│                                                                           │
│  DISTRIBUTED_PLATFORM_SETUP.md           Full setup guide (40+ pages)    │
│  DISTRIBUTED_QUICK_START.md              Quick reference                 │
│  FIRST_TIME_SETUP_VERIFICATION.md        Verification checklist          │
│  DISTRIBUTED_PLATFORM_SUMMARY.md         This summary                    │
│                                                                           │
└───────────────────────────────────────────────────────────────────────────┘

┌───────────────────────────────────────────────────────────────────────────┐
│ QUICK START FLOWCHART                                                     │
├───────────────────────────────────────────────────────────────────────────┤
│                                                                           │
│  START HERE                                                              │
│        ↓                                                                 │
│  1. Verify Remote Services (100.84.126.19)                              │
│        ↓                                                                 │
│  2. ./test-distributed-connectivity.sh                                  │
│        ↓                                                                 │
│  All tests pass? ──NO──→ Fix connection issues, retest                  │
│        ↓                                                                 │
│      YES                                                                 │
│        ↓                                                                 │
│  3. ./start-distributed-platform.sh                                     │
│        ↓                                                                 │
│  Backend running? ──NO──→ Check logs, fix issues                        │
│        ↓                                                                 │
│      YES                                                                 │
│        ↓                                                                 │
│  4. cd frontend && npm run dev                                          │
│        ↓                                                                 │
│  Frontend running? ──NO──→ Check logs, fix issues                       │
│        ↓                                                                 │
│      YES                                                                 │
│        ↓                                                                 │
│  5. open http://localhost:5173                                          │
│        ↓                                                                 │
│  ✅ DONE! Platform is running                                           │
│                                                                           │
└───────────────────────────────────────────────────────────────────────────┘

Quick Help: grep -r "100.84.126.19" .env docker-compose.*.yml
Last Updated: February 2026
EOF

echo ""
echo "════════════════════════════════════════════════════════════════════════════"
echo "          Save this as a text file for quick reference"
echo "════════════════════════════════════════════════════════════════════════════"
