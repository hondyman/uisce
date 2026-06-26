# Events package

This package contains publisher/consumer implementations and helpers for event delivery.

Note on transport:
- The project has migrated from RabbitMQ (AMQP) to Redpanda/Kafka for its event transport.
- Use `kafka_publisher.go` / `kafka_consumer.go` / `publisher.go` for new implementations and integrations.

Legacy artifacts:
- `rabbitmq_publisher.go` and `rabbitmq_consumer.go` (if present) are retained for historical reference and should be considered **deprecated**.

Tips:
- Prefer `KafkaPublisher`'s `PublishToTopic(ctx, topic, key, payload)` for generic events.
- If you must bridge legacy RabbitMQ-specific logic, add an adapter that maps event payloads to Kafka topics.
