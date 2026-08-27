import React, { useState } from 'react';
import ReactECharts from 'echarts-for-react';
import { format } from 'date-fns';
import type { ChatSession, ChatMessage } from '../api/chatHistoryApi';
import { ObservabilityDrawer } from './ObservabilityDrawer';

interface Props {
  session: ChatSession | null;
  messages: ChatMessage[];
  open: boolean;
  onClose: () => void;
}

export function ConversationPanel({ session, messages, open, onClose }: Props) {
  const [showDetails, setShowDetails] = useState(false);
  const [traceOpen, setTraceOpen] = useState(false);
  const [activeTraceId, setActiveTraceId] = useState<string | undefined>(undefined);

  if (!open || !session) return null;

  const feedbackBadge = session.feedback_score === 1 ? (
    <span className="px-2 py-1 text-xs font-semibold rounded-full bg-green-100 text-green-800">👍</span>
  ) : session.feedback_score === -1 ? (
    <span className="px-2 py-1 text-xs font-semibold rounded-full bg-red-100 text-red-800">👎</span>
  ) : (
    <span className="px-2 py-1 text-xs font-semibold rounded-full bg-gray-100 text-gray-600">No feedback</span>
  );

  return (
    <>
      <div className="fixed inset-0 z-40 bg-black/40 flex justify-end">
        <div className="bg-white w-full max-w-3xl h-full overflow-y-auto shadow-xl">
          <header className="sticky top-0 z-10 bg-white border-b border-gray-200 px-6 py-4">
            <div className="flex items-start justify-between gap-4">
              <div>
                <h2 className="text-lg font-semibold text-gray-900">
                  Conversation {session.id.split('-')[0]}…
                </h2>
                <div className="mt-2 flex flex-wrap items-center gap-2 text-xs text-gray-500">
                  <span className="px-2 py-0.5 rounded-full bg-blue-100 text-blue-800 font-medium">
                    {session.view_type === 'admin' ? 'Admin View' : 'End User View'}
                  </span>
                  <span className={`px-2 py-0.5 rounded-full font-medium ${session.embedded ? 'bg-purple-100 text-purple-800' : 'bg-gray-100 text-gray-600'}`}>
                    {session.embedded ? `Embedded · ${session.embed_surface ?? 'studio'}` : 'Studio'}
                  </span>
                  <span className="px-2 py-0.5 rounded-full bg-indigo-100 text-indigo-800 font-medium">
                    Agent: {session.agent_id}
                  </span>
                  {feedbackBadge}
                </div>
                {session.feedback_comment && (
                  <p className="mt-2 text-xs text-gray-600 italic">
                    Feedback: &ldquo;{session.feedback_comment}&rdquo;
                  </p>
                )}
              </div>
              <div className="flex flex-col items-end gap-3">
                <label className="flex items-center gap-2 text-sm text-gray-700">
                  <input
                    type="checkbox"
                    className="rounded"
                    checked={showDetails}
                    onChange={(e) => setShowDetails(e.target.checked)}
                  />
                  Show Details
                </label>
                <button
                  onClick={onClose}
                  className="text-gray-400 hover:text-gray-700 text-2xl leading-none"
                  aria-label="Close"
                >
                  ×
                </button>
              </div>
            </div>
          </header>

          <section className="px-6 py-5 space-y-4 bg-gray-50">
            {messages.length === 0 ? (
              <p className="text-sm text-gray-500 text-center py-12">No messages recorded.</p>
            ) : (
              messages.map((m) => (
                <article
                  key={m.id}
                  className={`flex ${m.role === 'user' ? 'justify-end' : 'justify-start'}`}
                >
                  <div
                    className={`max-w-[80%] rounded-2xl px-4 py-3 shadow-sm border ${
                      m.role === 'user'
                        ? 'bg-blue-600 text-white border-blue-700 rounded-br-sm'
                        : 'bg-white text-gray-900 border-gray-200 rounded-bl-sm'
                    }`}
                  >
                    <header className="flex items-center justify-between text-xs opacity-80 mb-1">
                      <span className="font-medium uppercase tracking-wide">{m.role}</span>
                      <span>{format(new Date(m.created_at), 'HH:mm:ss')}</span>
                    </header>
                    <p className="whitespace-pre-wrap text-sm leading-relaxed">{m.content}</p>

                    {m.chart_spec ? (
                      <div className="mt-3 -mx-1">
                        <ReactECharts
                          option={m.chart_spec as Record<string, unknown>}
                          style={{ height: 240, width: '100%' }}
                          notMerge
                          lazyUpdate
                        />
                      </div>
                    ) : null}

                    {showDetails ? (
                      <footer className="mt-3 pt-3 border-t border-current/10 space-y-2">
                        {typeof m.latency_ms === 'number' ? (
                          <p className="text-xs opacity-80">latency: {m.latency_ms} ms</p>
                        ) : null}
                        {m.trace_id ? (
                          <button
                            onClick={() => {
                              setActiveTraceId(m.trace_id ?? undefined);
                              setTraceOpen(true);
                            }}
                            className="text-xs underline opacity-90 hover:opacity-100"
                          >
                            View Observability →
                          </button>
                        ) : null}
                        {m.tool_calls ? (
                          <details className="text-xs opacity-80">
                            <summary className="cursor-pointer">tool_calls</summary>
                            <pre className="mt-1 bg-black/10 rounded p-2 overflow-x-auto">
                              {JSON.stringify(m.tool_calls, null, 2)}
                            </pre>
                          </details>
                        ) : null}
                        {m.error ? (
                          <p className="text-xs text-red-200">error: {m.error}</p>
                        ) : null}
                      </footer>
                    ) : null}
                  </div>
                </article>
              ))
            )}
          </section>
        </div>
      </div>
      <ObservabilityDrawer
        open={traceOpen}
        onOpenChange={setTraceOpen}
        traceId={activeTraceId}
      />
    </>
  );
}