import apiClient from '../../utils/apiClient';

const client = apiClient as <T = unknown>(input: string, init?: RequestInit) => Promise<T>;

export interface ChatSession {
  id: string;
  tenant_id: string;
  conversation_id: string;
  agent_id: string;
  agent_version?: string;
  user_id: string;
  user_email?: string;
  view_type: 'end_user' | 'admin';
  embedded: boolean;
  embed_surface?: string;
  started_at: string;
  ended_at?: string;
  message_count: number;
  feedback_score?: -1 | 1;
  feedback_comment?: string;
  trace_id?: string;
  first_message?: string;
  last_message?: string;
}

export interface ChatMessage {
  id: string;
  session_id: string;
  tenant_id: string;
  seq: number;
  role: 'user' | 'assistant' | 'system' | 'tool';
  content: string;
  content_json?: unknown;
  tool_calls?: unknown;
  chart_spec?: unknown;
  trace_steps?: Array<{ name: string; duration_ms: number; inputs?: unknown; outputs?: unknown }>;
  latency_ms?: number;
  token_in?: number;
  token_out?: number;
  trace_id?: string;
  span_id?: string;
  error?: string;
  created_at: string;
}

export interface ChatSessionDetail {
  session: ChatSession;
  messages: ChatMessage[];
}

export interface ChatHistoryFilters {
  all_tenants?: boolean;
  tenant_id?: string;
  agent_id?: string;
  view_type?: 'end_user' | 'admin';
  embedded?: boolean;
  feedback?: 'positive' | 'negative' | 'unrated';
  search?: string;
  from?: string;
  to?: string;
  limit?: number;
  offset?: number;
}

export const chatHistoryApi = {
  listSessions(filters: ChatHistoryFilters = {}) {
    const q = new URLSearchParams();
    if (filters.all_tenants) q.append('all_tenants', 'true');
    if (filters.tenant_id) q.append('tenant_id', filters.tenant_id);
    if (filters.agent_id) q.append('agent_id', filters.agent_id);
    if (filters.view_type) q.append('view_type', filters.view_type);
    if (filters.embedded !== undefined) q.append('embedded', String(filters.embedded));
    if (filters.feedback) q.append('feedback', filters.feedback);
    if (filters.search) q.append('search', filters.search);
    if (filters.from) q.append('from', filters.from);
    if (filters.to) q.append('to', filters.to);
    if (filters.limit) q.append('limit', String(filters.limit));
    if (filters.offset) q.append('offset', String(filters.offset));
    const qs = q.toString();
    return client<{ sessions: ChatSession[]; total: number }>(`/chat-history/sessions${qs ? `?${qs}` : ''}`);
  },

  getSession(id: string) {
    return client<ChatSessionDetail>(`/chat-history/sessions/${id}`);
  },

  submitFeedback(id: string, score: -1 | 1, comment?: string) {
    return client<{ ok: true }>(`/chat-history/sessions/${id}/feedback`, {
      method: 'POST',
      body: JSON.stringify({ score, comment }),
    });
  },

  exportCsv(filters: ChatHistoryFilters = {}) {
    const q = new URLSearchParams();
    if (filters.all_tenants) q.append('all_tenants', 'true');
    if (filters.tenant_id) q.append('tenant_id', filters.tenant_id);
    if (filters.agent_id) q.append('agent_id', filters.agent_id);
    if (filters.view_type) q.append('view_type', filters.view_type);
    if (filters.embedded !== undefined) q.append('embedded', String(filters.embedded));
    if (filters.feedback) q.append('feedback', filters.feedback);
    if (filters.search) q.append('search', filters.search);
    if (filters.from) q.append('from', filters.from);
    if (filters.to) q.append('to', filters.to);
    const qs = q.toString();
    return client<Blob>(`/chat-history/sessions/export.csv${qs ? `?${qs}` : ''}`);
  },

  getTrace(traceId: string) {
    return client<{ traceID?: string; spans?: unknown[] }>(`/tempo/traces/${traceId}`);
  },
};