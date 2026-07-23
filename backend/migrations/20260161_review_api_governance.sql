-- Migration: 20260161_review_api_governance.sql
-- Goal: Add API-specific governance fields to change reviews

ALTER TABLE semantic.change_reviews ADD COLUMN IF NOT EXISTS api_breaking_changes jsonb;
