-- Migration: 002 - CDC target tables for the orm schema
--
-- The stream loader (backend/cmd/stream_loader) writes Debezium change events into
-- these tables. They MUST be PRIMARY KEY tables:
--
--   * op=c / op=r / op=u  -> upsert by key. A DUPLICATE KEY table would append a new
--     row per update, so a row updated five times would exist five times.
--   * op=d                 -> DELETE ... WHERE <key>. StarRocks has no per-row delete
--     in the load API, so the loader issues a keyed DELETE; a PK table is what makes
--     that cheap rather than a full predicate scan.
--
-- The loader batch-suppresses repeated loads with a `label`, so a redelivered Kafka
-- batch is rejected as a duplicate instead of double-loading. That makes the whole
-- path at-least-once from Kafka and effectively exactly-once in StarRocks.
--
-- Column names are lowercase on purpose: Debezium emits the Postgres column names and
-- StarRocks matches JSON object keys to columns by name.
--
-- Type mapping applied by the loader before the write:
--   uuid                   -> VARCHAR(36)
--   numeric(p,s)           -> DECIMAL(p,s)   (Debezium base64 Decimal is decoded)
--   timestamptz            -> DATETIME        (ZonedTimestamp is converted to UTC)
--   date                   -> DATE            (epoch days are converted)
--   jsonb                  -> STRING          (Debezium Json is base64-decoded)

CREATE DATABASE IF NOT EXISTS oms;

-- orm."order"
CREATE TABLE IF NOT EXISTS oms.orm_order (
    `id`                VARCHAR(36)  NOT NULL COMMENT 'uuid primary key',
    `sec_id`            DECIMAL(18,0)     NULL COMMENT 'raw security id',
    `security_id`       VARCHAR(36)  NULL COMMENT 'uuid security reference',
    `side`              VARCHAR(10)  NULL,
    `order_type`        VARCHAR(20)  NULL,
    `status`            VARCHAR(20)  NULL,
    `target_qty`        DECIMAL(18,4)     NULL,
    `executed_qty`      DECIMAL(18,4)     NULL,
    `leaves_qty`        DECIMAL(18,4)     NULL,
    `limit_price`       DECIMAL(18,9)     NULL,
    `avg_price`         DECIMAL(18,9)     NULL,
    `time_in_force`     VARCHAR(10)  NULL,
    `trade_date`        DATE         NULL,
    `manager_id`        VARCHAR(36)  NULL,
    `trader_id`         VARCHAR(36)  NULL,
    `custom_attributes` STRING       NULL COMMENT 'jsonb, stored as JSON text',
    `tenant_id`         VARCHAR(36)  NOT NULL COMMENT 'owning tenant',
    `created_at`        DATETIME     NULL,
    `updated_at`        DATETIME     NULL
)
ENGINE=OLAP
PRIMARY KEY(`id`)
DISTRIBUTED BY HASH(`id`) BUCKETS 8
PROPERTIES (
    "replication_num"          = "1",
    "enable_persistent_index"  = "true",
    "compression"              = "zstd"
);

-- orm.execution
CREATE TABLE IF NOT EXISTS oms.orm_execution (
    `id`             VARCHAR(36)  NOT NULL COMMENT 'uuid primary key',
    `placement_id`   VARCHAR(36)  NULL,
    `order_id`       VARCHAR(36)  NULL,
    `exec_qty`       DECIMAL(18,4)     NULL,
    `exec_price`     DECIMAL(18,9)     NULL,
    `broker_id`      VARCHAR(50)  NULL,
    `exec_time`      DATETIME     NULL,
    `transact_time`  DATETIME     NULL,
    `status`         VARCHAR(20)  NULL,
    `broker_exec_id` VARCHAR(120) NULL,
    `last_capacity`  VARCHAR(1)   NULL,
    `tenant_id`      VARCHAR(36)  NOT NULL COMMENT 'owning tenant',
    `created_at`     DATETIME     NULL,
    `updated_at`     DATETIME     NULL
)
ENGINE=OLAP
PRIMARY KEY(`id`)
DISTRIBUTED BY HASH(`id`) BUCKETS 8
PROPERTIES (
    "replication_num"          = "1",
    "enable_persistent_index"  = "true",
    "compression"              = "zstd"
);

-- orm.placement
CREATE TABLE IF NOT EXISTS oms.orm_placement (
    `id`           VARCHAR(36)  NOT NULL COMMENT 'uuid primary key',
    `order_id`     VARCHAR(36)  NULL,
    `broker_id`    VARCHAR(20)  NULL,
    `venue_id`     VARCHAR(20)  NULL,
    `routed_qty`   DECIMAL(18,4)     NULL,
    `executed_qty` DECIMAL(18,4)     NULL,
    `leaves_qty`   DECIMAL(18,4)     NULL,
    `status`       VARCHAR(20)  NULL,
    `fix_clordid`  VARCHAR(120) NULL,
    `tenant_id`    VARCHAR(36)  NOT NULL COMMENT 'owning tenant',
    `created_at`   DATETIME     NULL,
    `updated_at`   DATETIME     NULL
)
ENGINE=OLAP
PRIMARY KEY(`id`)
DISTRIBUTED BY HASH(`id`) BUCKETS 8
PROPERTIES (
    "replication_num"          = "1",
    "enable_persistent_index"  = "true",
    "compression"              = "zstd"
);

-- orm.order_allocation
CREATE TABLE IF NOT EXISTS oms.orm_order_allocation (
    `id`            VARCHAR(36)  NOT NULL COMMENT 'uuid primary key',
    `order_id`      VARCHAR(36)  NULL,
    `account_id`    VARCHAR(20)  NULL,
    `target_qty`    DECIMAL(18,4)     NULL,
    `allocated_qty` DECIMAL(18,4)     NULL,
    `status`        VARCHAR(20)  NULL,
    `tenant_id`     VARCHAR(36)  NOT NULL COMMENT 'owning tenant',
    `created_at`    DATETIME     NULL,
    `updated_at`    DATETIME     NULL
)
ENGINE=OLAP
PRIMARY KEY(`id`)
DISTRIBUTED BY HASH(`id`) BUCKETS 8
PROPERTIES (
    "replication_num"          = "1",
    "enable_persistent_index"  = "true",
    "compression"              = "zstd"
);

-- orm.execution_allocation
CREATE TABLE IF NOT EXISTS oms.orm_execution_allocation (
    `id`                  VARCHAR(36)  NOT NULL COMMENT 'uuid primary key',
    `execution_id`        VARCHAR(36)  NULL,
    `order_allocation_id` VARCHAR(36)  NULL,
    `alloc_exec_qty`      DECIMAL(18,4)     NULL,
    `alloc_exec_price`    DECIMAL(18,9)     NULL,
    `tenant_id`           VARCHAR(36)  NOT NULL COMMENT 'owning tenant',
    `created_at`          DATETIME     NULL
)
ENGINE=OLAP
PRIMARY KEY(`id`)
DISTRIBUTED BY HASH(`id`) BUCKETS 8
PROPERTIES (
    "replication_num"          = "1",
    "enable_persistent_index"  = "true",
    "compression"              = "zstd"
);

-- ---------------------------------------------------------------------------
-- SOURCE-SIDE REQUIREMENT (Postgres, not StarRocks)
-- ---------------------------------------------------------------------------
-- This file creates the DESTINATION tables. It cannot set anything on the Postgres
-- source, and there is one property of the source that silently breaks the path this
-- file exists to serve.
--
-- Every source table feeding these destinations needs REPLICA IDENTITY FULL:
--
--     ALTER TABLE orm.order REPLICA IDENTITY FULL;
--
-- Because routing dispatches each row by the tenant_id carried on the event, and a
-- DELETE under the default replica identity carries no tenant_id at all -- the logical
-- decoding plugin sends only the key columns and Debezium zero-fills the rest, so
-- before.tenant_id arrives as "". The loader then correctly refuses to guess a
-- destination, the delete dead-letters as ERR_TENANT_UNAVAILABLE_ON_DELETE, and the
-- row stays here forever with no counterpart in Postgres. Inserts and updates are
-- unaffected, so nothing looks broken until a delete is quietly missed.
--
-- Applied and kept current by
--     backend/db/migrations/20261226_001_cdc_replica_identity_from_publication.up.sql
-- which derives the set from the Debezium publication rather than a hardcoded list, and
-- by
--     backend/db/tenant_migrations/orm/0002_replica_identity_full.up.sql
-- for a tenant's own Postgres database.
--
-- If a source table is added to a publication or to a tenant schema, that coverage
-- picks it up on the next migration run. Do not add a table to any capture list without
-- it.