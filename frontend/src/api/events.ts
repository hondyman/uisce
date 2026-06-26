import { fetchAPI } from '../api';

export interface CreateEventPayload {
  bo_type?: string;
  bo_id: string;
  field_name: string;
  old_value: any;
  new_value: any;
  changed_by: string; // user id
  bp_step?: string | null;
  custom_data?: any;
}

export async function createEvent(payload: CreateEventPayload): Promise<any> {
  return fetchAPI('/events', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function getEventsForBO(bo_id: string): Promise<any> {
  return fetchAPI(`/events?bo_id=${encodeURIComponent(bo_id)}`, { method: 'GET' });
}
