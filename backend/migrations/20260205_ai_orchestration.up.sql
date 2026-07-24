-- Migration: AI Orchestration Requests
-- Date: 2026-02-05
-- Description: Adds table for async AI request processing.

CREATE TABLE IF NOT EXISTS ai_requests (
    id          TEXT PRIMARY KEY,
    type        TEXT NOT NULL, -- BUSINESS_TERM, CHANGESET, INCIDENT, RISK, DRIFT, SLO
    payload     JSONB NOT NULL,
    status      TEXT NOT NULL DEFAULT 'PENDING', -- PENDING, RUNNING, SUCCESS, FAILED
    output      JSONB,
    error       TEXT,
    attempts    INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ai_requests_status ON ai_requests(status) WHERE status IN ('PENDING', 'RUNNING');
