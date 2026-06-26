-- Migration: AI Business Term Drafts
-- Date: 2026-02-02
-- Description: Adds table for storing AI-generated business term drafts.

CREATE TABLE IF NOT EXISTS ai_business_term_drafts (
    id                  TEXT PRIMARY KEY,
    name                TEXT NOT NULL,
    definition          TEXT NOT NULL,
    pii_flag            BOOLEAN NOT NULL,
    sensitivity         TEXT NOT NULL,        -- LOW | MEDIUM | HIGH
    residency           TEXT NOT NULL,        -- EU | US | GLOBAL | UNKNOWN
    hierarchy_level1    TEXT NOT NULL,
    hierarchy_level2    TEXT NOT NULL,
    hierarchy_level3    TEXT NOT NULL,
    source_semantic_terms JSONB NOT NULL,     -- ["st-..."]
    source_columns        JSONB NOT NULL,     -- ["table.column"]
    tags                  JSONB NOT NULL,     -- ["pii","client"]
    status              TEXT NOT NULL,        -- DRAFT_AI | APPROVED | REJECTED
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by          TEXT,                 -- system or user
    reviewed_at         TIMESTAMPTZ,
    reviewed_by         TEXT,
    review_comment      TEXT
);

CREATE INDEX IF NOT EXISTS idx_ai_bt_drafts_status ON ai_business_term_drafts(status);
CREATE INDEX IF NOT EXISTS idx_ai_bt_drafts_hierarchy ON ai_business_term_drafts(hierarchy_level1, hierarchy_level2);
