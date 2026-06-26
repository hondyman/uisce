#!/bin/bash
set -e

# Wait for Debezium to be up
echo "Waiting for Debezium to start..."
while ! curl -s http://localhost:8083/ > /dev/null; do
  sleep 1
done

echo "Registering Postgres Connector..."
curl -i -X POST -H "Accept:application/json" -H "Content-Type:application/json" http://localhost:8083/connectors/ -d '{
  "name": "postgres-connector",
  "config": {
    "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
    "database.hostname": "postgres",
    "database.port": "5432",
    "database.user": "dbz",
    "database.password": "dbz",
    "database.dbname": "demo",
    "database.server.name": "dbserver",
    "plugin.name": "pgoutput",
    "slot.name": "debezium",
    "publication.name": "dbz_pub",
    "database.history.kafka.bootstrap.servers": "redpanda:9092",
    "database.history.kafka.topic": "schema-changes.history",
    "key.converter": "io.confluent.connect.avro.AvroConverter",
    "key.converter.schema.registry.url": "http://schema-registry:8080",
    "value.converter": "io.confluent.connect.avro.AvroConverter",
    "value.converter.schema.registry.url": "http://schema-registry:8080"
  }
}'

echo "Done."
