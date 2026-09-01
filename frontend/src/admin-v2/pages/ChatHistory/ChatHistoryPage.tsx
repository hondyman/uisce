import React, { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { format } from 'date-fns';
import { chatHistoryApi, ChatSession, ChatHistoryFilters, ChatSessionDetail } from '../../api/chatHistoryApi';
import { ConversationPanel } from '../../components/ConversationPanel';
import { useAuth } from '../../../contexts/AuthContext';

const PAGE_SIZE = 50;

export function ChatHistoryPage() {
  const { isGlobalAdmin } = useAuth();
  const isAdmin = typeof isGlobalAdmin === 'function' ? isGlobalAdmin() : Boolean(isGlobalAdmin);
  const [filters, setFilters] = useState<ChatHistoryFilters>({
    all_tenants: isAdmin,
    limit: PAGE_SIZE,
    offset: 0,
  });
  const [search, setSearch] = useState('');
  const [tenantFilter, setTenantFilter] = useState('');
  const [viewType, setViewType] = useState('');
  const [embedded, setEmbedded] = useState('');
  const [feedback, setFeedback] = useState('');
  const [openSession, setOpenSession] = useState<ChatSession | null>(null);
  const [messages, setMessages] = useState<ChatSessionDetail['messages']>([]);
  const [open, setOpen] = useState(false);

  const debouncedFilters = useMemo<ChatHistoryFilters>(
    () => ({
      ...filters,
      search: search || undefined,
      tenant_id: tenantFilter || undefined,
      view_type: viewType ? (viewType as 'end_user' | 'admin') : undefined,
      embedded: embedded === '' ? undefined : embedded === 'true',
      feedback: feedback ? (feedback as 'positive' | 'negative' | 'unrated') : undefined,
    }),
    [filters, search, tenantFilter, viewType, embedded, feedback]
  );

  const { data, isLoading, error } = useQuery({
    queryKey: ['chat-history', debouncedFilters],
    queryFn: () => chatHistoryApi.listSessions(debouncedFilters),
  });

  const onRowClick = async (session: ChatSession) => {
    setOpenSession(session);
    setOpen(true);
    try {
      const detail = await chatHistoryApi.getSession(session.id);
      setMessages(detail.messages);
    } catch {
      setMessages([]);
    }
  };

  const onExport = async () => {
    try {
      const qs = new URLSearchParams();
      if (debouncedFilters.all_tenants) qs.append('all_tenants', 'true');
      if (debouncedFilters.tenant_id) qs.append('tenant_id', debouncedFilters.tenant_id);
      if (debouncedFilters.agent_id) qs.append('agent_id', debouncedFilters.agent_id);
      if (debouncedFilters.view_type) qs.append('view_type', debouncedFilters.view_type);
      if (debouncedFilters.embedded !== undefined) qs.append('embedded', String(debouncedFilters.embedded));
      if (debouncedFilters.feedback) qs.append('feedback', debouncedFilters.feedback);
      if (debouncedFilters.search) qs.append('search', debouncedFilters.search);

      const url = `/api/chat-history/sessions/export.csv${qs.toString() ? `?${qs}` : ''}`;
      const resp = await fetch(url, { credentials: 'include', headers: { Authorization: `Bearer ${localStorage.getItem('auth_token') ?? ''}` } });
      const blob = await resp.blob();
      const link = document.createElement('a');
      link.href = URL.createObjectURL(blob);
      link.download = `chat-history-${format(new Date(), 'yyyy-MM-dd')}.csv`;
      link.click();
      URL.revokeObjectURL(link.href);
    } catch (err) {
      console.error('CSV export failed', err);
    }
  };

  const sessions: ChatSession[] = data?.sessions ?? [];
  const total = data?.total ?? 0;

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Chat History</h1>
          <p className="text-sm text-gray-500 mt-1">
            Review every conversation handled by the agent, including chats from deployed end users.
          </p>
        </div>
        <button
          onClick={onExport}
          className="px-4 py-2 text-sm font-medium rounded-md bg-blue-600 text-white hover:bg-blue-700"
        >
          Export CSV
        </button>
      </div>

      <div className="mb-6 grid grid-cols-1 md:grid-cols-5 gap-3 p-4 bg-white rounded-lg shadow-sm border border-gray-200">
        <div>
          <label className="block text-xs font-medium text-gray-600 mb-1">Search</label>
          <input
            type="text"
            className="w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 sm:text-sm px-3 py-2 border"
            placeholder="Find a message…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        {isAdmin && (
          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">Tenant ID</label>
            <input
              type="text"
              className="w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 sm:text-sm px-3 py-2 border"
              placeholder="Filter by tenant…"
              value={tenantFilter}
              onChange={(e) => setTenantFilter(e.target.value)}
            />
          </div>
        )}
        <div>
          <label className="block text-xs font-medium text-gray-600 mb-1">View Type</label>
          <select
            className="w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 sm:text-sm px-3 py-2 border"
            value={viewType}
            onChange={(e) => setViewType(e.target.value)}
          >
            <option value="">All</option>
            <option value="end_user">End User</option>
            <option value="admin">Admin</option>
          </select>
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-600 mb-1">Embedded</label>
          <select
            className="w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 sm:text-sm px-3 py-2 border"
            value={embedded}
            onChange={(e) => setEmbedded(e.target.value)}
          >
            <option value="">All</option>
            <option value="false">Studio</option>
            <option value="true">Embedded</option>
          </select>
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-600 mb-1">Feedback</label>
          <select
            className="w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 sm:text-sm px-3 py-2 border"
            value={feedback}
            onChange={(e) => setFeedback(e.target.value)}
          >
            <option value="">All</option>
            <option value="positive">👍 Positive</option>
            <option value="negative">👎 Negative</option>
            <option value="unrated">No feedback</option>
          </select>
        </div>
      </div>

      <div className="bg-white shadow-sm rounded-lg border border-gray-200 overflow-hidden">
        {isLoading ? (
          <div className="p-8 text-center text-gray-500">Loading conversations…</div>
        ) : error ? (
          <div className="p-8 text-center text-red-500">Failed to load chat history.</div>
        ) : (
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Date</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Duration</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Tenant</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Agent</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">View Type</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Embedded</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Feedback</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {sessions.length === 0 ? (
                <tr>
                  <td colSpan={7} className="px-6 py-8 text-center text-sm text-gray-500">
                    No conversations match the current filters.
                  </td>
                </tr>
              ) : (
                sessions.map((s) => {
                  const duration =
                    s.ended_at
                      ? `${Math.round(
                          (new Date(s.ended_at).getTime() - new Date(s.started_at).getTime()) / 1000
                        )}s`
                      : '—';
                  return (
                    <tr
                      key={s.id}
                      className="hover:bg-gray-50 cursor-pointer"
                      onClick={() => onRowClick(s)}
                    >
                      <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-900">
                        {format(new Date(s.started_at), 'MMM d, yyyy HH:mm')}
                      </td>
                      <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500">{duration}</td>
                      <td className="px-4 py-3 whitespace-nowrap text-xs font-mono text-gray-500">
                        {s.tenant_id.split('-')[0]}…
                      </td>
                      <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-700">{s.agent_id}</td>
                      <td className="px-4 py-3 whitespace-nowrap">
                        <span className={`px-2 py-1 text-xs font-medium rounded-full ${
                          s.view_type === 'admin'
                            ? 'bg-indigo-100 text-indigo-800'
                            : 'bg-gray-100 text-gray-700'
                        }`}>
                          {s.view_type}
                        </span>
                      </td>
                      <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-600">
                        {s.embedded ? `Yes · ${s.embed_surface ?? 'studio'}` : 'No'}
                      </td>
                      <td className="px-4 py-3 whitespace-nowrap text-sm">
                        {s.feedback_score === 1 ? (
                          <span className="text-green-600">👍</span>
                        ) : s.feedback_score === -1 ? (
                          <span className="text-red-600">👎</span>
                        ) : (
                          <span className="text-gray-400">—</span>
                        )}
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        )}
      </div>

      <div className="flex items-center justify-between mt-4 text-xs text-gray-500">
        <span>
          {total === 0 ? 'No results.' : `${total} conversation${total === 1 ? '' : 's'} total.`}
        </span>
        <div className="flex gap-2">
          <button
            disabled={!filters.offset || (filters.offset ?? 0) <= 0}
            onClick={() =>
              setFilters((f) => ({ ...f, offset: Math.max(0, (f.offset ?? 0) - PAGE_SIZE) }))
            }
            className="px-3 py-1 border rounded disabled:opacity-50"
          >
            ← Prev
          </button>
          <button
            disabled={sessions.length < PAGE_SIZE}
            onClick={() => setFilters((f) => ({ ...f, offset: (f.offset ?? 0) + PAGE_SIZE }))}
            className="px-3 py-1 border rounded disabled:opacity-50"
          >
            Next →
          </button>
        </div>
      </div>

      <ConversationPanel
        session={openSession}
        messages={messages}
        open={open}
        onClose={() => setOpen(false)}
      />
    </div>
  );
}