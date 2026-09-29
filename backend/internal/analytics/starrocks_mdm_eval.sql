-- StarRocks Hot Analytical Mart for MDM Source Scoring & Vendor Displacement
-- Connect to StarRocks FE (e.g. mysql -h 127.0.0.1 -P 9030 -u root)

-- 1. Mount Iceberg via Lakekeeper REST Catalog (Raw Vendor Payloads)
CREATE EXTERNAL CATALOG IF NOT EXISTS lakekeeper_iceberg
PROPERTIES (
    "type" = "iceberg",
    "iceberg.catalog.type" = "rest",
    "iceberg.catalog.uri" = "http://100.84.50.65:8182/v1",
    "aws.s3.access_key" = "minioadmin",
    "aws.s3.secret_key" = "minioadmin",
    "aws.s3.endpoint" = "http://100.84.50.65:9000",
    "aws.s3.enable_path_style_access" = "true"
);

-- 2. Mount PostgreSQL alpha OLTP via JDBC (Golden Values, Tolerances, Overrides)
CREATE EXTERNAL CATALOG IF NOT EXISTS pg_alpha
PROPERTIES (
    "type" = "jdbc",
    "user" = "postgres",
    "password" = "postgres",
    "jdbc_uri" = "jdbc:postgresql://100.84.50.65:5432/alpha",
    "driver_url" = "https://repo1.maven.org/maven2/org/postgresql/postgresql/42.6.0/postgresql-42.6.0.jar",
    "driver_class" = "org.postgresql.Driver"
);

-- 3. Internal Analytical Database
CREATE DATABASE IF NOT EXISTS mdm_analytics;
USE mdm_analytics;

-- 4. Hot Daily Scoring Mart (Primary Key Table for fast dashboard reads)
CREATE TABLE IF NOT EXISTS mdm_analytics.vendor_substitution_daily (
    as_of_date           DATE NOT NULL,
    attribute_code       VARCHAR(64) NOT NULL,
    vendor_id            VARCHAR(36) NOT NULL,
    tier                 INT NOT NULL,
    in_scope_entities    BIGINT DEFAULT "0",
    valid_matches        BIGINT DEFAULT "0",
    valid_differs        BIGINT DEFAULT "0",
    absent_count         BIGINT DEFAULT "0",
    invalid_count        BIGINT DEFAULT "0",
    solo_count           BIGINT DEFAULT "0",
    golden_wins          BIGINT DEFAULT "0"
)
ENGINE = OLAP
PRIMARY KEY (as_of_date, attribute_code, vendor_id)
DISTRIBUTED BY HASH(attribute_code, vendor_id) BUCKETS 16
PROPERTIES (
    "replication_num" = "1"
);
