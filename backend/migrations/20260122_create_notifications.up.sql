-- Notification Rules
CREATE TABLE IF NOT EXISTS notification_rule (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  bp_def_id UUID,                      -- if NULL, applies to all workflows
  step_key TEXT,                       -- if NULL, applies to all steps
  trigger_event TEXT NOT NULL,         -- 'step_assigned', 'sla_warning', 'sla_breach', 'approved', 'rejected'
  channels TEXT[] NOT NULL,            -- ['email', 'slack', 'sms']
  template_key TEXT NOT NULL,          -- 'approval_needed', 'sla_warning', etc.
  delay_seconds INT DEFAULT 0,         -- for reminders (3600 = 1 hour)
  recipient_role TEXT,                 -- 'current_approver', 'initiator', 'admin'
  recipient_user_id TEXT,              -- specific user
  recipient_group_id TEXT,             -- group/team
  enabled BOOLEAN DEFAULT true,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  UNIQUE (tenant_id, trigger_event, step_key, recipient_role)
);
CREATE INDEX IF NOT EXISTS idx_notif_rule_lookup ON notification_rule(tenant_id, trigger_event);

-- Notification Templates
CREATE TABLE IF NOT EXISTS notification_template (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  template_key TEXT NOT NULL,
  name TEXT NOT NULL,
  subject TEXT,                        -- for email
  body_text TEXT,                      -- plain text fallback
  body_html TEXT,                      -- HTML email
  slack_template JSONB,                -- Slack Block Kit JSON
  enabled BOOLEAN DEFAULT true,
  created_at TIMESTAMP DEFAULT NOW(),
  UNIQUE (tenant_id, template_key)
);
CREATE INDEX IF NOT EXISTS idx_notif_tmpl_lookup ON notification_template(tenant_id, template_key);

-- Notification Log
CREATE TABLE IF NOT EXISTS notification_log (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  instance_id UUID NOT NULL,
  rule_id UUID NOT NULL,
  trigger_event TEXT NOT NULL,
  recipient_user_id TEXT,
  recipient_email TEXT,
  recipient_slack_id TEXT,
  channels_attempted TEXT[],
  channels_succeeded TEXT[],
  body JSONB,
  error_message TEXT,
  sent_at TIMESTAMP DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_notif_log_instance ON notification_log(instance_id, trigger_event);
CREATE INDEX IF NOT EXISTS idx_notif_log_user_sent ON notification_log(recipient_user_id, sent_at);

-- User Preferences
CREATE TABLE IF NOT EXISTS user_notification_preference (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id TEXT NOT NULL,
  tenant_id UUID NOT NULL,
  notification_type TEXT,              -- 'approval_needed', 'sla_warning', 'all'
  channels TEXT[],                     -- ['email', 'slack']
  enabled BOOLEAN DEFAULT true,
  quiet_hours_start TIME,              -- don't send between 6pm-8am
  quiet_hours_end TIME,
  created_at TIMESTAMP DEFAULT NOW(),
  UNIQUE (user_id, tenant_id, notification_type)
);

-- Slack Integration
CREATE TABLE IF NOT EXISTS slack_integration (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  workspace_id TEXT NOT NULL,
  workspace_name TEXT,
  bot_token TEXT NOT NULL,             -- encrypted
  app_id TEXT,
  signing_secret TEXT NOT NULL,        -- encrypted
  installed_at TIMESTAMP DEFAULT NOW(),
  UNIQUE (tenant_id, workspace_id)
);

-- Slack User Mapping
CREATE TABLE IF NOT EXISTS user_slack_mapping (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id TEXT NOT NULL,
  tenant_id UUID NOT NULL,
  slack_user_id TEXT NOT NULL,
  slack_email TEXT,
  verified_at TIMESTAMP,
  UNIQUE (user_id, tenant_id),
  UNIQUE (slack_user_id, tenant_id)
);

-- Scheduled Reminders
CREATE TABLE IF NOT EXISTS notification_schedule (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  instance_id UUID NOT NULL,
  rule_id UUID NOT NULL,
  trigger_event TEXT NOT NULL,
  scheduled_for TIMESTAMP NOT NULL,
  is_sent BOOLEAN DEFAULT false,
  attempts INT DEFAULT 0,
  next_retry_at TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_notif_sched_run ON notification_schedule(scheduled_for, is_sent);
