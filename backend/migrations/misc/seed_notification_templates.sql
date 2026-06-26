-- ============================================================================
-- Business Process Notification Templates
-- Seed data for common notification scenarios
-- ============================================================================

-- 1. APPROVAL REQUIRED
INSERT INTO notification_templates (
    id, tenant_id, datasource_id, template_key, template_name, description, category,
    subject_template, body_template, template_variables, enabled_channels, default_channel,
    digest_mode, escalation_enabled, escalation_delay_minutes, escalation_recipient_roles,
    is_system, is_active, priority, include_quick_actions, quick_actions, created_by
) VALUES (
    gen_random_uuid(),
    '870361a8-87e2-4171-95ad-0473cc93791e', -- tenant_id (Legal and General)
    'fcbd3043-5076-4a5f-90a4-d0730830f885', -- datasource_id
    'bp_approval_required',
    'Approval Required',
    'Notification sent when a business process step requires approval',
    'approval',
    'Approval Required: {process_name}',
    E'Hi {user_name},\n\nYou have a pending approval for the following process:\n\nProcess: {process_name}\nStep: {step_name}\nRequested By: {requester_name}\nDue Date: {due_date}\n\n{description}\n\nPlease review and approve or reject at your earliest convenience.',
    ARRAY['{user_name}', '{process_name}', '{step_name}', '{requester_name}', '{due_date}', '{description}', '{process_link}'],
    ARRAY['email', 'slack', 'teams'],
    'email',
    'immediate',
    true,
    60, -- Escalate after 1 hour
    ARRAY['manager', 'director'],
    true,
    true,
    'high',
    true,
    '{"actions": [{"label": "Approve", "action": "approve", "style": "primary"}, {"label": "Reject", "action": "reject", "style": "danger"}, {"label": "View Details", "action": "view", "style": "default"}]}'::jsonb,
    'system'
) ON CONFLICT (tenant_id, datasource_id, template_key) DO NOTHING;

-- 2. APPROVAL REMINDER
INSERT INTO notification_templates (
    id, tenant_id, datasource_id, template_key, template_name, description, category,
    subject_template, body_template, template_variables, enabled_channels, default_channel,
    digest_mode, escalation_enabled, is_system, is_active, priority, created_by
) VALUES (
    gen_random_uuid(),
    '870361a8-87e2-4171-95ad-0473cc93791e',
    'fcbd3043-5076-4a5f-90a4-d0730830f885',
    'bp_approval_reminder',
    'Approval Reminder',
    'Reminder notification for pending approvals',
    'reminder',
    'Reminder: Pending Approval for {process_name}',
    E'Hi {user_name},\n\nThis is a reminder that you have a pending approval:\n\nProcess: {process_name}\nStep: {step_name}\nDays Pending: {days_pending}\nDue Date: {due_date}\n\nPlease take action to keep the process moving forward.',
    ARRAY['{user_name}', '{process_name}', '{step_name}', '{days_pending}', '{due_date}', '{process_link}'],
    ARRAY['email', 'slack'],
    'email',
    'immediate',
    false,
    true,
    true,
    'normal',
    'system'
) ON CONFLICT (tenant_id, datasource_id, template_key) DO NOTHING;

-- 3. PROCESS COMPLETED
INSERT INTO notification_templates (
    id, tenant_id, datasource_id, template_key, template_name, description, category,
    subject_template, body_template, template_variables, enabled_channels, default_channel,
    digest_mode, is_system, is_active, priority, created_by
) VALUES (
    gen_random_uuid(),
    '870361a8-87e2-4171-95ad-0473cc93791e',
    'fcbd3043-5076-4a5f-90a4-d0730830f885',
    'bp_process_completed',
    'Process Completed',
    'Notification sent when a business process completes successfully',
    'info',
    'Process Completed: {process_name}',
    E'Hi {user_name},\n\nGreat news! The following process has been completed successfully:\n\nProcess: {process_name}\nCompleted By: {completed_by}\nCompletion Date: {completion_date}\nTotal Duration: {duration}\n\nKey Metrics:\n- Total Steps: {total_steps}\n- Approvals: {approval_count}\n- Average Step Time: {avg_step_time}\n\nYou can view the full process history here: {process_link}',
    ARRAY['{user_name}', '{process_name}', '{completed_by}', '{completion_date}', '{duration}', '{total_steps}', '{approval_count}', '{avg_step_time}', '{process_link}'],
    ARRAY['email', 'slack', 'teams'],
    'email',
    'hourly', -- Can be batched into hourly digest
    true,
    true,
    'low',
    'system'
) ON CONFLICT (tenant_id, datasource_id, template_key) DO NOTHING;

-- 4. PROCESS FAILED
INSERT INTO notification_templates (
    id, tenant_id, datasource_id, template_key, template_name, description, category,
    subject_template, body_template, template_variables, enabled_channels, default_channel,
    digest_mode, escalation_enabled, escalation_delay_minutes, escalation_recipient_roles,
    is_system, is_active, priority, created_by
) VALUES (
    gen_random_uuid(),
    '870361a8-87e2-4171-95ad-0473cc93791e',
    'fcbd3043-5076-4a5f-90a4-d0730830f885',
    'bp_process_failed',
    'Process Failed',
    'Alert notification when a business process fails',
    'alert',
    '🚨 Process Failed: {process_name}',
    E'Hi {user_name},\n\nA process has failed and requires attention:\n\nProcess: {process_name}\nFailed Step: {step_name}\nError: {error_message}\nFailed At: {failed_at}\n\nPlease investigate and take corrective action immediately.\n\nProcess Link: {process_link}\nError Log: {error_log_link}',
    ARRAY['{user_name}', '{process_name}', '{step_name}', '{error_message}', '{failed_at}', '{process_link}', '{error_log_link}'],
    ARRAY['email', 'slack', 'teams', 'sms'],
    'email',
    'immediate',
    true,
    30, -- Escalate after 30 minutes
    ARRAY['admin', 'operations_manager'],
    true,
    true,
    'urgent',
    'system'
) ON CONFLICT (tenant_id, datasource_id, template_key) DO NOTHING;

-- 5. STEP ASSIGNMENT
INSERT INTO notification_templates (
    id, tenant_id, datasource_id, template_key, template_name, description, category,
    subject_template, body_template, template_variables, enabled_channels, default_channel,
    digest_mode, is_system, is_active, priority, include_quick_actions, quick_actions, created_by
) VALUES (
    gen_random_uuid(),
    '870361a8-87e2-4171-95ad-0473cc93791e',
    'fcbd3043-5076-4a5f-90a4-d0730830f885',
    'bp_step_assigned',
    'Step Assigned',
    'Notification when a process step is assigned to a user',
    'info',
    'New Task Assigned: {step_name}',
    E'Hi {user_name},\n\nYou have been assigned a new task:\n\nProcess: {process_name}\nStep: {step_name}\nAssigned By: {assigned_by}\nDue Date: {due_date}\n\nDescription:\n{description}\n\nPlease complete this step as soon as possible.',
    ARRAY['{user_name}', '{process_name}', '{step_name}', '{assigned_by}', '{due_date}', '{description}', '{process_link}'],
    ARRAY['email', 'slack', 'teams', 'push'],
    'push',
    'immediate',
    true,
    true,
    'normal',
    true,
    '{"actions": [{"label": "Start Now", "action": "start", "style": "primary"}, {"label": "View Details", "action": "view", "style": "default"}]}'::jsonb,
    'system'
) ON CONFLICT (tenant_id, datasource_id, template_key) DO NOTHING;

-- 6. SLA WARNING
INSERT INTO notification_templates (
    id, tenant_id, datasource_id, template_key, template_name, description, category,
    subject_template, body_template, template_variables, enabled_channels, default_channel,
    digest_mode, escalation_enabled, escalation_delay_minutes, escalation_recipient_roles,
    is_system, is_active, priority, created_by
) VALUES (
    gen_random_uuid(),
    '870361a8-87e2-4171-95ad-0473cc93791e',
    'fcbd3043-5076-4a5f-90a4-d0730830f885',
    'bp_sla_warning',
    'SLA Warning',
    'Warning when a process step is approaching SLA breach',
    'alert',
    '⚠️ SLA Warning: {process_name} - {step_name}',
    E'Hi {user_name},\n\nA process step is approaching its SLA deadline:\n\nProcess: {process_name}\nStep: {step_name}\nAssigned To: {assignee}\nTime Remaining: {time_remaining}\nSLA Deadline: {sla_deadline}\n\nCurrent Status: {current_status}\nCompleted: {completion_percentage}%\n\nPlease expedite to avoid SLA breach.',
    ARRAY['{user_name}', '{process_name}', '{step_name}', '{assignee}', '{time_remaining}', '{sla_deadline}', '{current_status}', '{completion_percentage}', '{process_link}'],
    ARRAY['email', 'slack', 'teams', 'sms'],
    'email',
    'immediate',
    true,
    15, -- Escalate after 15 minutes
    ARRAY['manager', 'operations_manager'],
    true,
    true,
    'high',
    'system'
) ON CONFLICT (tenant_id, datasource_id, template_key) DO NOTHING;

-- 7. ESCALATION NOTIFICATION
INSERT INTO notification_templates (
    id, tenant_id, datasource_id, template_key, template_name, description, category,
    subject_template, body_template, template_variables, enabled_channels, default_channel,
    digest_mode, is_system, is_active, priority, include_quick_actions, quick_actions, created_by
) VALUES (
    gen_random_uuid(),
    '870361a8-87e2-4171-95ad-0473cc93791e',
    'fcbd3043-5076-4a5f-90a4-d0730830f885',
    'bp_escalation',
    'Escalation Notice',
    'Notification sent when an item is escalated to management',
    'escalation',
    'Escalation: {process_name} - Action Required',
    E'Hi {user_name},\n\nAn item has been escalated to your attention:\n\nProcess: {process_name}\nStep: {step_name}\nOriginal Assignee: {original_assignee}\nEscalation Level: {escalation_level}\nDays Pending: {days_pending}\nReason: {escalation_reason}\n\nThis requires immediate action to resolve the blockage.\n\nOriginal Request:\n{original_description}',
    ARRAY['{user_name}', '{process_name}', '{step_name}', '{original_assignee}', '{escalation_level}', '{days_pending}', '{escalation_reason}', '{original_description}', '{process_link}'],
    ARRAY['email', 'slack', 'teams', 'sms'],
    'email',
    'immediate',
    true,
    true,
    'urgent',
    true,
    '{"actions": [{"label": "Take Ownership", "action": "claim", "style": "primary"}, {"label": "Reassign", "action": "reassign", "style": "default"}, {"label": "View History", "action": "history", "style": "default"}]}'::jsonb,
    'system'
) ON CONFLICT (tenant_id, datasource_id, template_key) DO NOTHING;

-- 8. DAILY DIGEST
INSERT INTO notification_templates (
    id, tenant_id, datasource_id, template_key, template_name, description, category,
    subject_template, body_template, template_variables, enabled_channels, default_channel,
    digest_mode, is_system, is_active, priority, created_by
) VALUES (
    gen_random_uuid(),
    '870361a8-87e2-4171-95ad-0473cc93791e',
    'fcbd3043-5076-4a5f-90a4-d0730830f885',
    'bp_daily_digest',
    'Daily Process Digest',
    'Daily summary of process activities',
    'info',
    'Your Daily Process Summary - {date}',
    E'Hi {user_name},\n\nHere''s your daily process summary for {date}:\n\nPending Actions: {pending_count}\n- Approvals Needed: {approvals_needed}\n- Tasks Assigned: {tasks_assigned}\n- Overdue Items: {overdue_count}\n\nCompleted Today: {completed_count}\n- Approvals Given: {approvals_given}\n- Tasks Completed: {tasks_completed}\n- Processes Finished: {processes_finished}\n\nKey Metrics:\n- Average Completion Time: {avg_completion_time}\n- SLA Compliance: {sla_compliance}%\n- Top Process: {top_process}\n\nView full dashboard: {dashboard_link}',
    ARRAY['{user_name}', '{date}', '{pending_count}', '{approvals_needed}', '{tasks_assigned}', '{overdue_count}', '{completed_count}', '{approvals_given}', '{tasks_completed}', '{processes_finished}', '{avg_completion_time}', '{sla_compliance}', '{top_process}', '{dashboard_link}'],
    ARRAY['email'],
    'email',
    'daily',
    true,
    true,
    'low',
    'system'
) ON CONFLICT (tenant_id, datasource_id, template_key) DO NOTHING;

-- 9. WEEKLY REPORT
INSERT INTO notification_templates (
    id, tenant_id, datasource_id, template_key, template_name, description, category,
    subject_template, body_template, template_variables, enabled_channels, default_channel,
    digest_mode, is_system, is_active, priority, include_attachments, created_by
) VALUES (
    gen_random_uuid(),
    '870361a8-87e2-4171-95ad-0473cc93791e',
    'fcbd3043-5076-4a5f-90a4-d0730830f885',
    'bp_weekly_report',
    'Weekly Process Report',
    'Comprehensive weekly summary and analytics',
    'info',
    'Weekly Process Report: {week_start} to {week_end}',
    E'Hi {user_name},\n\nYour weekly process report is ready.\n\nWeek of {week_start} to {week_end}\n\nHighlights:\n✅ Processes Completed: {processes_completed}\n⏱️ Average Cycle Time: {avg_cycle_time}\n📈 SLA Compliance: {sla_compliance}%\n🎯 Efficiency Gain: {efficiency_gain}%\n\nTop Performers:\n1. {top_performer_1}\n2. {top_performer_2}\n3. {top_performer_3}\n\nAreas for Improvement:\n- {improvement_area_1}\n- {improvement_area_2}\n\nTrending Processes:\n📊 Most Used: {most_used_process}\n⚡ Fastest: {fastest_process}\n🐌 Slowest: {slowest_process}\n\nFull report attached. View online: {report_link}',
    ARRAY['{user_name}', '{week_start}', '{week_end}', '{processes_completed}', '{avg_cycle_time}', '{sla_compliance}', '{efficiency_gain}', '{top_performer_1}', '{top_performer_2}', '{top_performer_3}', '{improvement_area_1}', '{improvement_area_2}', '{most_used_process}', '{fastest_process}', '{slowest_process}', '{report_link}'],
    ARRAY['email'],
    'email',
    'weekly',
    true,
    true,
    'low',
    true,
    'system'
) ON CONFLICT (tenant_id, datasource_id, template_key) DO NOTHING;

-- 10. PROCESS COMMENT / MENTION
INSERT INTO notification_templates (
    id, tenant_id, datasource_id, template_key, template_name, description, category,
    subject_template, body_template, template_variables, enabled_channels, default_channel,
    digest_mode, is_system, is_active, priority, include_quick_actions, quick_actions, created_by
) VALUES (
    gen_random_uuid(),
    '870361a8-87e2-4171-95ad-0473cc93791e',
    'fcbd3043-5076-4a5f-90a4-d0730830f885',
    'bp_comment_mention',
    'You Were Mentioned',
    'Notification when a user is mentioned in a process comment',
    'info',
    '{commenter_name} mentioned you in {process_name}',
    E'Hi {user_name},\n\n{commenter_name} mentioned you in a comment:\n\nProcess: {process_name}\nStep: {step_name}\nComment:\n"{comment_text}"\n\nPosted: {comment_time}\n\nView and reply: {process_link}#comments',
    ARRAY['{user_name}', '{commenter_name}', '{process_name}', '{step_name}', '{comment_text}', '{comment_time}', '{process_link}'],
    ARRAY['email', 'slack', 'teams', 'push'],
    'push',
    'immediate',
    true,
    true,
    'normal',
    true,
    '{"actions": [{"label": "Reply", "action": "reply", "style": "primary"}, {"label": "View Thread", "action": "view", "style": "default"}]}'::jsonb,
    'system'
) ON CONFLICT (tenant_id, datasource_id, template_key) DO NOTHING;

-- 11. COLLABORATION INVITE
INSERT INTO notification_templates (
    id, tenant_id, datasource_id, template_key, template_name, description, category,
    subject_template, body_template, template_variables, enabled_channels, default_channel,
    digest_mode, is_system, is_active, priority, include_quick_actions, quick_actions, created_by
) VALUES (
    gen_random_uuid(),
    '870361a8-87e2-4171-95ad-0473cc93791e',
    'fcbd3043-5076-4a5f-90a4-d0730830f885',
    'bp_collab_invite',
    'Collaboration Invite',
    'Invitation to collaborate on a process',
    'info',
    'Invitation to collaborate on {process_name}',
    E'Hi {user_name},\n\n{inviter_name} has invited you to collaborate on a process:\n\nProcess: {process_name}\nRole: {role}\nAccess Level: {access_level}\nMessage: {invite_message}\n\nAccept the invitation to start collaborating.',
    ARRAY['{user_name}', '{inviter_name}', '{process_name}', '{role}', '{access_level}', '{invite_message}', '{process_link}'],
    ARRAY['email', 'slack', 'teams'],
    'email',
    'immediate',
    true,
    true,
    'normal',
    true,
    '{"actions": [{"label": "Accept", "action": "accept", "style": "primary"}, {"label": "Decline", "action": "decline", "style": "default"}]}'::jsonb,
    'system'
) ON CONFLICT (tenant_id, datasource_id, template_key) DO NOTHING;

-- 12. PROCESS PERFORMANCE ALERT
INSERT INTO notification_templates (
    id, tenant_id, datasource_id, template_key, template_name, description, category,
    subject_template, body_template, template_variables, enabled_channels, default_channel,
    digest_mode, escalation_enabled, escalation_delay_minutes, is_system, is_active, priority, created_by
) VALUES (
    gen_random_uuid(),
    '870361a8-87e2-4171-95ad-0473cc93791e',
    'fcbd3043-5076-4a5f-90a4-d0730830f885',
    'bp_performance_alert',
    'Performance Alert',
    'Alert when process performance drops below threshold',
    'alert',
    '⚡ Performance Alert: {process_name}',
    E'Hi {user_name},\n\nA process has exhibited performance degradation:\n\nProcess: {process_name}\nMetric: {metric_name}\nCurrent Value: {current_value}\nThreshold: {threshold_value}\nVariance: {variance}%\nTime Period: {time_period}\n\nRecent Trend:\n{trend_description}\n\nRecommended Actions:\n{recommended_actions}\n\nView analytics: {analytics_link}',
    ARRAY['{user_name}', '{process_name}', '{metric_name}', '{current_value}', '{threshold_value}', '{variance}', '{time_period}', '{trend_description}', '{recommended_actions}', '{analytics_link}'],
    ARRAY['email', 'slack'],
    'email',
    'immediate',
    true,
    45,
    true,
    true,
    'high',
    'system'
) ON CONFLICT (tenant_id, datasource_id, template_key) DO NOTHING;

-- 13. AI OPTIMIZATION SUGGESTION
INSERT INTO notification_templates (
    id, tenant_id, datasource_id, template_key, template_name, description, category,
    subject_template, body_template, template_variables, enabled_channels, default_channel,
    digest_mode, is_system, is_active, priority, include_quick_actions, quick_actions, created_by
) VALUES (
    gen_random_uuid(),
    '870361a8-87e2-4171-95ad-0473cc93791e',
    'fcbd3043-5076-4a5f-90a4-d0730830f885',
    'bp_ai_suggestion',
    'AI Optimization Suggestion',
    'AI-powered suggestion to optimize a process',
    'info',
    '🤖 AI Suggestion: Optimize {process_name}',
    E'Hi {user_name},\n\nOur AI has analyzed {process_name} and found optimization opportunities:\n\nCurrent Performance:\n- Average Duration: {current_duration}\n- Success Rate: {success_rate}%\n- Cost per Instance: {cost_per_instance}\n\nSuggested Improvements:\n1. {suggestion_1}\n   Expected Improvement: {improvement_1}\n2. {suggestion_2}\n   Expected Improvement: {improvement_2}\n3. {suggestion_3}\n   Expected Improvement: {improvement_3}\n\nProjected Outcomes:\n- Duration Reduction: {duration_reduction}%\n- Cost Savings: {cost_savings}\n- Quality Improvement: {quality_improvement}%\n\nWould you like to apply these optimizations?',
    ARRAY['{user_name}', '{process_name}', '{current_duration}', '{success_rate}', '{cost_per_instance}', '{suggestion_1}', '{improvement_1}', '{suggestion_2}', '{improvement_2}', '{suggestion_3}', '{improvement_3}', '{duration_reduction}', '{cost_savings}', '{quality_improvement}', '{process_link}'],
    ARRAY['email', 'slack'],
    'email',
    'daily',
    true,
    true,
    'normal',
    true,
    '{"actions": [{"label": "Apply All", "action": "apply_all", "style": "primary"}, {"label": "Review Details", "action": "review", "style": "default"}, {"label": "Dismiss", "action": "dismiss", "style": "default"}]}'::jsonb,
    'system'
) ON CONFLICT (tenant_id, datasource_id, template_key) DO NOTHING;

-- 14. VERSION PUBLISHED
INSERT INTO notification_templates (
    id, tenant_id, datasource_id, template_key, template_name, description, category,
    subject_template, body_template, template_variables, enabled_channels, default_channel,
    digest_mode, is_system, is_active, priority, created_by
) VALUES (
    gen_random_uuid(),
    '870361a8-87e2-4171-95ad-0473cc93791e',
    'fcbd3043-5076-4a5f-90a4-d0730830f885',
    'bp_version_published',
    'New Version Published',
    'Notification when a new process version is published',
    'info',
    'New Version: {process_name} v{version_number}',
    E'Hi {user_name},\n\nA new version of {process_name} has been published:\n\nVersion: v{version_number}\nPublished By: {publisher_name}\nPublished At: {publish_date}\n\nChanges in this version:\n{change_summary}\n\nKey Improvements:\n- {improvement_1}\n- {improvement_2}\n- {improvement_3}\n\nMigration Notes:\n{migration_notes}\n\nView changelog: {changelog_link}\nView process: {process_link}',
    ARRAY['{user_name}', '{process_name}', '{version_number}', '{publisher_name}', '{publish_date}', '{change_summary}', '{improvement_1}', '{improvement_2}', '{improvement_3}', '{migration_notes}', '{changelog_link}', '{process_link}'],
    ARRAY['email', 'slack'],
    'email',
    'hourly',
    true,
    true,
    'low',
    'system'
) ON CONFLICT (tenant_id, datasource_id, template_key) DO NOTHING;

-- 15. INTEGRATION ERROR
INSERT INTO notification_templates (
    id, tenant_id, datasource_id, template_key, template_name, description, category,
    subject_template, body_template, template_variables, enabled_channels, default_channel,
    digest_mode, escalation_enabled, escalation_delay_minutes, escalation_recipient_roles,
    is_system, is_active, priority, created_by
) VALUES (
    gen_random_uuid(),
    '870361a8-87e2-4171-95ad-0473cc93791e',
    'fcbd3043-5076-4a5f-90a4-d0730830f885',
    'bp_integration_error',
    'Integration Error',
    'Alert when an integration fails during process execution',
    'alert',
    '🔌 Integration Error: {integration_name} in {process_name}',
    E'Hi {user_name},\n\nAn integration has failed during process execution:\n\nProcess: {process_name}\nIntegration: {integration_name}\nStep: {step_name}\nError: {error_message}\nError Code: {error_code}\nFailed At: {failed_at}\nRetry Attempt: {retry_count}/{max_retries}\n\nRequest Details:\n{request_details}\n\nResponse:\n{response_details}\n\nTroubleshooting:\n{troubleshooting_tips}\n\nView error log: {error_log_link}',
    ARRAY['{user_name}', '{integration_name}', '{process_name}', '{step_name}', '{error_message}', '{error_code}', '{failed_at}', '{retry_count}', '{max_retries}', '{request_details}', '{response_details}', '{troubleshooting_tips}', '{error_log_link}', '{process_link}'],
    ARRAY['email', 'slack', 'sms'],
    'email',
    'immediate',
    true,
    20,
    ARRAY['admin', 'integration_admin'],
    true,
    true,
    'urgent',
    'system'
) ON CONFLICT (tenant_id, datasource_id, template_key) DO NOTHING;

-- ============================================================================
-- Add comments for documentation
-- ============================================================================

COMMENT ON TABLE notification_templates IS 'Business process notification templates with multi-channel support';
COMMENT ON COLUMN notification_templates.template_variables IS 'Array of placeholder variables like {user_name}, {process_name} used in templates';
COMMENT ON COLUMN notification_templates.enabled_channels IS 'Array of enabled notification channels: email, sms, slack, teams, push';
COMMENT ON COLUMN notification_templates.send_conditions IS 'JSONB rules for conditional sending based on context';
COMMENT ON COLUMN notification_templates.digest_mode IS 'When to send: immediate, hourly, daily, weekly';
COMMENT ON COLUMN notification_templates.escalation_recipient_roles IS 'Array of roles to escalate to if no action taken';
COMMENT ON COLUMN notification_templates.quick_actions IS 'JSONB array of quick action buttons to include in notification';
