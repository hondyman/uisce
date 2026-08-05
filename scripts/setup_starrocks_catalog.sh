#!/bin/bash
set -e

echo "Setting up StarRocks Backend and Iceberg External Catalog..."

# 1. Add Backend to Frontend
docker exec -i starrocks-fe mysql -h 127.0.0.1 -P 9030 -u root --password=password -e "ALTER SYSTEM ADD BACKEND \"starrocks-be:9050\";" || true

# 2. Create External Catalog for Iceberg MinIO Lakehouse
docker exec -i starrocks-fe mysql -h 127.0.0.1 -P 9030 -u root --password=password -e "
CREATE EXTERNAL CATALOG IF NOT EXISTS iceberg_lakehouse
PROPERTIES (
  \"type\" = \"iceberg\",
  \"iceberg.catalog.type\" = \"rest\",
  \"iceberg.catalog.uri\" = \"http://iceberg-rest:8181\",
  \"iceberg.catalog.warehouse\" = \"iceberg-bucket\",
  \"aws.s3.access_key\" = \"minioadmin\",
  \"aws.s3.secret_key\" = \"minioadmin\",
  \"aws.s3.endpoint\" = \"http://minio:9000\",
  \"aws.s3.enable_ssl\" = \"false\",
  \"aws.s3.use_aws_sdk_default_behavior\" = \"false\"
);
" || true

echo "StarRocks Iceberg Catalog registered successfully!"
