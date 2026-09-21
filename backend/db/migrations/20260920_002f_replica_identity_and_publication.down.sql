-- Rollback for 20260920_002f_replica_identity_and_publication.up.sql
-- Drops the publication first (must come before the tables could be reaped), then
--  restores REPLICA IDENTITY to DEFAULT (f) on the 5 critical tables.

BEGIN;

DROP PUBLICATION IF EXISTS dropped_trigger_source_publication;

ALTER TABLE public.crypto_transactions        REPLICA IDENTITY DEFAULT;
ALTER TABLE public.template_ratings           REPLICA IDENTITY DEFAULT;
ALTER TABLE public.semantic_query_templates   REPLICA IDENTITY DEFAULT;
ALTER TABLE public.investment_opportunities   REPLICA IDENTITY DEFAULT;
ALTER TABLE public.process_execution_metrics  REPLICA IDENTITY DEFAULT;

COMMIT;
