-- Phase 3: Insert Test Data
INSERT INTO tenants (id, name, allowed_regions) 
VALUES ('550e8400-e29b-41d4-a716-446655440000', 'Test Tenant', '{"us-east-1"}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO calendars (tenant_id, name, region, holidays) 
VALUES (
  '550e8400-e29b-41d4-a716-446655440000',
  'USA Federal Holidays',
  'US',
  jsonb_build_array(
    jsonb_build_object('date', '2026-01-01', 'name', 'New Year', 'severity', 'HIGH'),
    jsonb_build_object('date', '2026-07-04', 'name', 'Independence Day', 'severity', 'HIGH'),
    jsonb_build_object('date', '2026-12-25', 'name', 'Christmas', 'severity', 'HIGH'),
    jsonb_build_object('date', '2026-11-26', 'name', 'Thanksgiving', 'severity', 'MEDIUM'),
    jsonb_build_object('date', '2026-02-16', 'name', 'Presidents Day', 'severity', 'MEDIUM')
  )
)
ON CONFLICT DO NOTHING;

INSERT INTO schedule_profiles (tenant_id, name, timezone, conflict_resolution)
VALUES ('550e8400-e29b-41d4-a716-446655440000', 'default', 'UTC', 'UNION')
ON CONFLICT DO NOTHING;

-- Link calendar to profile using correct tenant ID
DO $$
DECLARE
  test_tenant_id UUID := '550e8400-e29b-41d4-a716-446655440000';
  calendar_id UUID;
  profile_id UUID;
BEGIN
  SELECT id INTO calendar_id FROM calendars WHERE tenant_id = test_tenant_id AND name = 'USA Federal Holidays' LIMIT 1;
  SELECT id INTO profile_id FROM schedule_profiles WHERE tenant_id = test_tenant_id AND name = 'default' LIMIT 1;
  
  IF calendar_id IS NOT NULL AND profile_id IS NOT NULL THEN
    INSERT INTO profile_calendars (profile_id, calendar_id, weight)
    VALUES (profile_id, calendar_id, 100)
    ON CONFLICT (profile_id, calendar_id) DO NOTHING;
  END IF;
END $$;

-- Insert blackouts (recurring and one-time)
DO $$
DECLARE
  test_tenant_id UUID := '550e8400-e29b-41d4-a716-446655440000';
  profile_id UUID;
BEGIN
  SELECT id INTO profile_id FROM schedule_profiles WHERE tenant_id = test_tenant_id AND name = 'default' LIMIT 1;
  
  IF profile_id IS NOT NULL THEN
    -- One-time maintenance window
    INSERT INTO blackouts (tenant_id, profile_id, name, description, start_time, end_time, reason, severity)
    VALUES (
      test_tenant_id,
      profile_id,
      'Monthly Maintenance',
      'Scheduled database maintenance',
      '2026-02-20 02:00:00+00',
      '2026-02-20 04:00:00+00',
      'MAINTENANCE',
      'HIGH'
    )
    ON CONFLICT DO NOTHING;
    
    -- Recurring: Every Monday 11 PM - 1 AM UTC for 52 weeks
    INSERT INTO blackouts (tenant_id, profile_id, name, description, start_time, end_time, reason, severity, recurrence_rule)
    VALUES (
      test_tenant_id,
      profile_id,
      'Weekly Batch Job',
      'Recurring batch processing window',
      '2026-02-23 23:00:00+00',
      '2026-02-24 01:00:00+00',
      'PLANNED_DOWNTIME',
      'MEDIUM',
      'FREQ=WEEKLY;BYDAY=MO;COUNT=52'
    )
    ON CONFLICT DO NOTHING;
    
    -- Recurring: Every Friday 3 PM - 5 PM UTC
    INSERT INTO blackouts (tenant_id, profile_id, name, description, start_time, end_time, reason, severity, recurrence_rule)
    VALUES (
      test_tenant_id,
      profile_id,
      'Weekly Deployment Window',
      'Scheduled deployments every Friday afternoon',
      '2026-02-20 15:00:00+00',
      '2026-02-20 17:00:00+00',
      'PLANNED_DOWNTIME',
      'LOW',
      'FREQ=WEEKLY;BYDAY=FR;COUNT=52'
    )
    ON CONFLICT DO NOTHING;
  END IF;
END $$;

-- Summary
SELECT 'Test Data Inserted' as status;
SELECT category, COUNT(*) as count FROM (
  SELECT 'Tenants' as category FROM tenants WHERE id = '550e8400-e29b-41d4-a716-446655440000'
  UNION ALL
  SELECT 'Calendars' FROM calendars WHERE tenant_id = '550e8400-e29b-41d4-a716-446655440000'
  UNION ALL
  SELECT 'Profiles' FROM schedule_profiles WHERE tenant_id = '550e8400-e29b-41d4-a716-446655440000'
  UNION ALL
  SELECT 'Blackouts' FROM blackouts WHERE tenant_id = '550e8400-e29b-41d4-a716-446655440000'
) AS data
GROUP BY category;
