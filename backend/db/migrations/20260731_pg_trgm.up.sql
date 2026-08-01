-- Migration: pg_trgm extension for fuzzy similarity matching
-- Date: 2026-07-31
-- Purpose: Required by governance.SelfHealingService for trigram-based
--          field name matching when generating schema drift repair proposals.
--          The similarity() and word_similarity() functions are used by
--          drift_healer.go to suggest candidate field names from
--          tenant_custom_attributes when a VM rule references an unknown symbol.

CREATE EXTENSION IF NOT EXISTS pg_trgm;
