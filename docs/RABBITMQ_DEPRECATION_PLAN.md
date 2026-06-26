# RabbitMQ (AMQP) Deprecation Plan — Redpanda/Kafka Migration

Status: In progress — most runtime/config/docs migrated to **Redpanda/Kafka**. Legacy RabbitMQ (AMQP) artifacts retained for reference and marked **DEPRECATED**.

Goal: Remove runtime dependency on RabbitMQ and replace all AMQP-specific code and docs with Kafka/Redpanda equivalents, with a clear migration plan and backward compatibility adapters where needed.

Short checklist
- [x] Add local/CI smoke test for Redpanda (`scripts/redpanda_smoke_test.sh`, GitHub Actions) ✅
- [x] Update rebalancing compose/docs to use Redpanda ✅
- [x] Update several infra compose files to replace RabbitMQ with Redpanda ✅
- [x] Add deprecation notes to RabbitMQ-specific code and docs ✅
- [x] Provide compatibility shims (e.g., `NewRabbitMQListener` -> KafkaListener) ✅

Remaining tasks (recommended order)
1. Repository sweep & annotate: find all remaining user-facing RabbitMQ references and either replace or mark deprecated (docs, scripts, examples). (In progress)
2. Convert active services: for each service still using AMQP producers/consumers, ensure they use `KafkaPublisher` / `KafkaConsumer` and update tests. Start with highest-impact services (event-router, event-syndication, command bus).
3. Add adapter layers where immediate full migration is non-trivial: provide `amqp->kafka` adapter or bridge for a bounded time window.
4. CI & Test coverage: add integration tests that bring up Redpanda and verify publish/consume across services (smoke + component tests).
5. Remove AMQP dependencies from go.mod and Dockerfiles once all code paths use Kafka and tests pass.
6. Remove RabbitMQ composer services and cleanup volumes (after a deprecation window).

Files identified as needing attention (examples)
- `backend/internal/events/rabbitmq_publisher.go` (LEGACY)
- `backend/internal/events/rabbitmq_consumer.go` (LEGACY)
- `backend/cmd/e2e_amqp/*` (legacy test) — mark deprecated
- `infrastructure/docker/docker-compose.uma.yml` (replaced rabbitmq with redpanda) — verify
- Various docs: `EVENT_SYNDICATION_*`, `PROJECT_INDEX.md`, `COMPLETION_SUMMARY.md` (mostly updated)

Suggested migration tickets
- Migrate `event-router` publish/consume to Kafka & update health checks (PR + tests)
- Replace `rabbitmq_publisher` usage across repo with `KafkaPublisher` (add adapter if needed)
- Remove `github.com/rabbitmq/amqp091-go` dependency and run `go mod tidy` (after code migration)
- Add CI matrix test that boots a minimal set of services with Redpanda and runs end-to-end publish/consume

Notes & Considerations
- Keep legacy artifacts for historical reference for a short window, then archive/delete after removal
- Coordinate with ops if any production systems still run RabbitMQ
- Ensure messaging semantics mapping (exchange/routing-to-topic/key mapping) is documented

If you'd like, I can:
- Convert one specific service (you pick) fully to Kafka (code + tests + compose), or
- Create the migration tickets and PRs for all remaining items (I can open branch + PRs), or
- Run a final sweep and open a PR that removes all user-facing RabbitMQ references after you approve the timeline.

Pick one and I’ll proceed.
