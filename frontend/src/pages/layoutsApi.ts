import apiClient from '@/utils/apiClient';

export type SavedLayoutItem = { id: string; name: string; updated_at?: string };

export async function fetchSavedLayoutsApi(): Promise<SavedLayoutItem[]> {
  const res = await apiClient<Response>('/layouts');
  if (!res.ok) throw new Error(`status ${res.status}`);
  const body = await res.json();
  return (body.layouts || []).map((l: any) => ({ id: l.id, name: l.name, updated_at: l.updated_at }));
}

export async function saveLayoutApi(body: { id?: string; name: string; layout: any }) {
  const res = await apiClient<Response>('/layouts', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  });
  if (!res.ok) throw new Error(`status ${res.status}`);
  return res.json();
}

export async function loadLayoutApi(id: string) {
  const res = await apiClient<Response>(`/layouts/${id}`);
  if (!res.ok) throw new Error(`status ${res.status}`);
  return res.json();
}

export async function deleteLayoutApi(id: string) {
  const res = await apiClient<Response>(`/layouts/${id}`, { method: 'DELETE' });
  if (!res.ok) throw new Error(`status ${res.status}`);
  return true;
}
