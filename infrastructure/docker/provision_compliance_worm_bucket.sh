#!/usr/bin/env bash
set -euo pipefail

# Provision MinIO WORM Bucket with Object Lock enabled for dev/test parity
MINIO_ALIAS="${MINIO_ALIAS:-local}"
MINIO_ENDPOINT="${MINIO_ENDPOINT:-http://localhost:9000}"
MINIO_ACCESS_KEY="${MINIO_ACCESS_KEY:-minioadmin}"
MINIO_SECRET_KEY="${MINIO_SECRET_KEY:-minioadmin}"
BUCKET_NAME="${BUCKET_NAME:-uisce-compliance-cold-archive}"
RETENTION_YEARS="${RETENTION_YEARS:-15}"

echo "Configuring MinIO client alias for ${MINIO_ENDPOINT}..."
mc alias set "${MINIO_ALIAS}" "${MINIO_ENDPOINT}" "${MINIO_ACCESS_KEY}" "${MINIO_SECRET_KEY}"

if mc ls "${MINIO_ALIAS}/${BUCKET_NAME}" >/dev/null 2>&1; then
    echo "Bucket ${BUCKET_NAME} already exists."
else
    echo "Creating bucket ${BUCKET_NAME} with Object Lock enabled..."
    mc mb --with-lock "${MINIO_ALIAS}/${BUCKET_NAME}"
fi

echo "Setting default Compliance Mode retention to ${RETENTION_YEARS} years on ${BUCKET_NAME}..."
mc retention set --default COMPLIANCE "${RETENTION_YEARS}y" "${MINIO_ALIAS}/${BUCKET_NAME}"

echo "MinIO WORM Compliance bucket provisioned successfully."
