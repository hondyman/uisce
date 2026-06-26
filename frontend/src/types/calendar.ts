export interface GoogleCalendar {
  id: string;
  summary: string;
  description?: string;
  timezone: string;
  primary: boolean;
  accessRole: string;
}

export interface SyncStatus {
  id: string;
  user_id: string;
  tenant_id: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  started_at?: string;
  completed_at?: string;
  events_processed: number;
  events_created: number;
  events_updated: number;
  events_deleted: number;
  errors: string[];
}

export interface SyncedEvent {
  id: string;
  tenant_id: string;
  internal_event_id?: string;
  google_calendar_id: string;
  google_event_id: string;
  title: string;
  start_time: string;
  end_time: string;
  status: string;
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
