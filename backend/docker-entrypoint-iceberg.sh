#!/usr/bin/env bash
set -euo pipefail

# Entrypoint for kafka-connect-iceberg that handles:
# 1. Runtime CA import (covers bind-mounted cert, survives container recreate)
# 2. Delegates to the standard Confluent Connect entrypoint

# Path where Keycloak CA cert may be bind-mounted
CA_CERT="/etc/kafka-connect-iceberg/certs/keycloak-ca.crt"

import_ca_if_present() {
    if [ ! -f "$CA_CERT" ]; then
        echo "[entrypoint] No CA cert at $CA_CERT — skipping runtime import"
        return
    fi

    JVM_CACERTS=$(find /usr/lib/jvm -name cacerts -head -1 || true)
    if [ -z "$JVM_CACERTS" ]; then
        echo "[entrypoint] WARNING: could not find JVM cacerts — CA import skipped"
        return
    fi

    # Check if already imported (idempotent)
    if keytool -list -cacerts -storepass changeit 2>/dev/null | grep -q "^keycloak-ca$"; then
        echo "[entrypoint] Keycloak CA already present in truststore — skipping import"
    else
        echo "[entrypoint] Importing Keycloak CA into $JVM_CACERTS"
        keytool -importcert -noprompt -cacerts -storepass changeit \
            -file "$CA_CERT" -alias keycloak-ca
    fi
}

import_ca_if_present

# Delegate to the standard Confluent entrypoint (handles Kafka Connect worker startup)
exec /etc/confluent/docker/run "$@"
