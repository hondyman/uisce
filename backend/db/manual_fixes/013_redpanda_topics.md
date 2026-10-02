# 013_redpanda_topics.md
# Redpanda topics for the streaming price pipeline

No crims schema coupling — topic names/partitions only. Safe to create
before or after `011`/`012` apply.

## Topics

| Topic | Partitions | Replicas | Cleanup | Purpose |
|---|---|---|---|---|
| `price.security.official.eod` | 3 | 1 | delete | Official EOD marks (FactSet/BBG/Refinitiv) |
| `price.security.quote.realtime` | 3 | 1 | delete | NBBO / quote ticks |
| `price.security.trade.realtime` | 3 | 1 | delete | Trade prints |
| `price.corrections` | 3 | 1 | **compact** | Correction / bust stream (key = tick id) |

## Create

```bash
rpk topic create price.security.official.eod   --partitions 3 --replicas 1
rpk topic create price.security.quote.realtime --partitions 3 --replicas 1
rpk topic create price.security.trade.realtime --partitions 3 --replicas 1
rpk topic create price.corrections --partitions 3 --replicas 1 \
    --config cleanup.policy=compact
```

## Verify

```bash
rpk topic list | grep '^price\.'
# expect: 4 topics
```

## Consumer groups (stream-ingest)

| Group | Topic |
|---|---|
| `stream-ingest-factset-eod` | `price.security.official.eod` |
| `stream-ingest-bbg-eod` | `price.security.official.eod` |
| `stream-ingest-rdp-eod` | `price.security.official.eod` |

One process per (source, topic) so `--source-id` stays fixed per instance.

## Message envelope

```json
{
  "source_system_cd": "FACTSET",
  "entity_type": "SECURITY",
  "entity_ref": "TEST001",
  "price_type_cd": "OFFICIAL",
  "observation_type": "EOD",
  "event_time_ms": 1759000000000,
  "value": 100.50,
  "currency": "USD"
}
```

`source_system_cd` must be one of `BLOOMBERG` | `FACTSET` | `REFINITIV`
(matches `mdm.source_systems.code`). The consumer maps cd → uuid via
`--source-id` (resolved once at startup, not per message).
