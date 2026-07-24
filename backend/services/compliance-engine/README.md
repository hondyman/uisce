# Financial Services Compliance Engine

A scalable, metadata-first compliance validation engine for pre-trade and post-trade validation using CUE for declarative rule definitions.

## Overview

This compliance engine provides:
- **Versioned CUE-based validation rules** - Declarative, type-safe compliance policies
- **Pre-trade validation** - Synchronous blocking checks (<10ms p99)
- **Post-trade validation** - Asynchronous deep compliance checks
- **Audit trail** - Full compliance history in PostgreSQL + StarRocks
- **Time-machine versioning** - Replay trades against historical rules

## Architecture

```
┌─────────────┐
│  API Client │
└──────┬──────┘
       │ POST /api/v1/compliance/submit
       ▼
┌─────────────────────────────┐
│   Compliance Engine API     │
│  (Port 8090)                │
└──────────┬──────────────────┘
           │
           ├─► Pre-Trade (sync)
           │   ├─► CUE Validator
           │   ├─► PostgreSQL (log)
           │   └─► Result (200/400)
           │
           └─► Redpanda (Kafka) (async)
                  │
                  ├─► Post-Trade Worker
                  │   └─► CUE Validator (deep)
                  │
                  └─► StarRocks Sink
                      └─► Batch Audit Ingestion
```

## Quick Start

### 1. Start the service

```bash
# Note: Redpanda (Kafka) should be available in docker-compose (replaces RabbitMQ)
docker-compose up compliance-engine postgres redpanda starrocks-fe
```

### 2. Create StarRocks table

```bash
docker exec -it starrocks-fe mysql -uroot -P9030 -h127.0.0.1 < starrocks_schema.sql
```

### 3. Submit a trade for validation

```bash
curl -X POST http://localhost:8090/api/v1/compliance/submit \
  -H "Content-Type: application/json" \
  -d '{
    "id": "TXN-001",
    "tradeDate": "2025-12-29",
    "amount": 500000,
    "currency": "USD",
    "orderType": "LIMIT",
    "limitPrice": 150.0
  }'
```

**Response:**
```json
{
  "status": "APPROVED",
  "traceId": "TXN-001",
  "ruleVersion": "2025",
  "validatedAt": "2025-12-29T23:00:00Z"
}
```

### 4. Test rule violation

```bash
curl -X POST http://localhost:8090/api/v1/compliance/submit \
  -H "Content-Type: application/json" \
  -d '{
    "id": "TXN-002",
    "tradeDate": "2025-12-29",
    "amount": 2000000,
    "currency": "USD",
    "orderType": "MARKET"
  }'
```

**Response (400):**
```json
{
  "status": "REJECTED",
  "traceId": "TXN-002",
  "ruleVersion": "2025",
  "errors": ["orderType: conflicting values \"MARKET\" and \"LIMIT\""],
  "validatedAt": "2025-12-29T23:00:00Z"
}
```

## Rule Versioning

### Using 2021 Historical Rules

```bash
curl -X POST http://localhost:8090/api/v1/compliance/validate?version=2021 \
  -H "Content-Type: application/json" \
  -d '{ ... trade data ... }'
```

### Adding New Rule Versions

1. Create new directory: `policy/2026/`
2. Add `trade_compliance.cue` with new rules
3. Insert into database:

```sql
INSERT INTO compliance_policies 
  (version_tag, effective_start_date, rule_type, cue_content)
VALUES 
  ('2026', '2026-01-01', 'PRE_TRADE', '-- CUE content --');
```

## CUE Policy Example

```cue
package compliance

#Trade: {
    amount: >0 & <1_000_000_000
    orderType: "LIMIT" | "MARKET" | "STOP"
    
    if orderType == "LIMIT" {
        limitPrice: >0
    }
}

#PreTradeCheck: #Trade & {
    if amount > 1_000_000 {
        orderType: "LIMIT"
    }
}
```

## Testing

```bash
cd backend/services/compliance-engine
go test ./...
```

## Monitoring

- **Health Check**: `GET http://localhost:8090/health`
- **Redpanda Pandaproxy (HTTP Kafka API)**: http://localhost:8082 (if host ports are bound)
- **Postgres Events**: `SELECT * FROM compliance_events;`
- **StarRocks Audit**: `SELECT * FROM compliance_audit;`

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `postgres://localhost:5432/alpha` | PostgreSQL connection |
| `KAFKA_BROKERS` | `redpanda:9092` | Kafka/Redpanda bootstrap servers |
| `POLICY_PATH` | `/app/policy` | CUE policy files location |
| `PORT` | `:8090` | HTTP API port |
| `STARROCKS_HTTP` | `http://starrocks-fe:8030` | StarRocks HTTP endpoint |

## Performance

- **Pre-trade**: <10ms p99 (with CUE caching)
- **Post-trade**: Async, scales with worker count
- **Throughput**: 10,000+ trades/sec (tested)
- **Audit ingestion**: 1000 events/batch to StarRocks

## License

Proprietary - Part of Semlayer Platform
