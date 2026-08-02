export type CalendarProvider = 'google' | 'microsoft' | 'apple';

export interface ExternalCalendar {
  id: string;
  name: string;
  description?: string;
  color?: string;
  primary: boolean;
  provider: CalendarProvider;
}

export interface SyncStatus {
  id: string;
  user_id: string;
  tenant_id: string;
  provider: CalendarProvider;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  progress: number;
  total_events: number;
  processed_events: number;
  errors: string[];
  started_at?: string;
  completed_at?: string;
  time_range?: {
    start: string;
    end: string;
  };
}

export interface SyncedEvent {
  id: string;
  provider: CalendarProvider;
  tenant_id: string;
  connection_id?: string;
  internal_event_id?: string;
  external_event_id: string;
  external_calendar_id: string;
  title: string;
  description?: string;
  location?: string;
  start_time: string;
  end_time: string;
  is_all_day: boolean;
  is_recurring: boolean;
  recurrence_rule?: string;
  recurrence_id?: string;
  sync_status: string;
  last_synced_at: string;
}

export interface InternalEvent {
  id: string;
  tenant_id: string;
  title: string;
  description?: string;
  location?: string;
  start_time: string;
  end_time: string;
  timezone: string;
  is_all_day: boolean;
  rrule?: string;
}
