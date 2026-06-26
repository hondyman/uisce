-- Migration: create admin_audit_logs table
-- Run this migration to enable persistence of admin audit actions

CREATE TABLE IF NOT EXISTS public.admin_audit_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NULL,
  actor_id TEXT NOT NULL,
  action TEXT NOT NULL,
  workflow_id TEXT NOT NULL,
  run_id TEXT NULL,
  reason TEXT NULL,
  input JSONB NULL,
  status TEXT NOT NULL,
  error_message TEXT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_admin_audit_workflow_id ON public.admin_audit_logs (workflow_id);
CREATE INDEX IF NOT EXISTS idx_admin_audit_tenant_id ON public.admin_audit_logs (tenant_id);
