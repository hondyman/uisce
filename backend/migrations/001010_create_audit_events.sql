-- 001010_create_audit_events.sql
-- Idempotent migration: create audit_events table required by backend audit service
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'audit_events') THEN
    CREATE TABLE IF NOT EXISTS public.audit_events (
      id VARCHAR(255) PRIMARY KEY,
      timestamp TIMESTAMPTZ NOT NULL,
      event_type VARCHAR(255),
      severity VARCHAR(255),
      user_id VARCHAR(255),
      tenant_id VARCHAR(255),
      session_id VARCHAR(255),
      resource_id VARCHAR(255),
      resource_type VARCHAR(255),
      action VARCHAR(255),
      ip_address INET,
      user_agent TEXT,
      request_id VARCHAR(255),
      details JSONB,
      old_values JSONB,
      new_values JSONB,
      success BOOLEAN,
      error_message TEXT,
      compliance_flags TEXT[]
    );

    EXECUTE 'CREATE INDEX IF NOT EXISTS idx_audit_events_tenant ON public.audit_events(tenant_id)';
    EXECUTE 'CREATE INDEX IF NOT EXISTS idx_audit_events_user ON public.audit_events(user_id)';
    EXECUTE 'CREATE INDEX IF NOT EXISTS idx_audit_events_event_type ON public.audit_events(event_type)';
    EXECUTE 'CREATE INDEX IF NOT EXISTS idx_audit_events_timestamp ON public.audit_events(timestamp)';
  END IF;
END$$;
