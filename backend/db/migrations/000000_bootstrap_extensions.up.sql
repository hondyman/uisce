-- Creates every Postgres extension the migration set actually depends on
-- (uuid_generate_v4(), pgcrypto's gen_random_uuid() on older callers,
-- btree_gin/btree_gist indexes, ltree paths, pgvector's vector type) but
-- that no migration ever created — all installed manually against alpha at
-- some unrecorded point. Discovered by dry-running the full migration set
-- against a fresh throwaway database while building the ephemeral CI
-- database (Feature A): the first migration using uuid_generate_v4() failed
-- outright with "function does not exist" on a clean Postgres 18 instance.
--
-- Sorted to apply first (000000, before the existing lowest filename
-- 000063_...) since later migrations assume these are already present.
-- pg_trgm is deliberately not repeated here — it already has its own
-- migration (20260731_pg_trgm.up.sql).

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS btree_gin;
CREATE EXTENSION IF NOT EXISTS btree_gist;
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS ltree;
CREATE EXTENSION IF NOT EXISTS vector;
