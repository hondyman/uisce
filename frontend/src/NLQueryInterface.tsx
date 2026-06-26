import { useState, useEffect as _useEffect } from 'react';
import { devError } from './utils/devLogger';
import { compileNLQuery, simulateNLQuery, getNLQuerySuggestions, startConversation, getConversationSummary } from './api';
import type { NLQueryRequest, NLQueryResponse, NLQuerySuggestion, ConversationContext, ConversationSummary } from './types';
import './NLQueryInterface.css';

interface NLQueryInterfaceProps {
  currentDatasource: string;
  currentUser: string;
  currentTenant: string;
  onQueryGenerated?: (query: NLQueryResponse) => void;
}

export default function NLQueryInterface({
  currentDatasource,
  currentUser,
  currentTenant,
  onQueryGenerated
}: NLQueryInterfaceProps) {
  const [text, setText] = useState('');
  const [loading, setLoading] = useState(false);
  const [mode, setMode] = useState<'compile' | 'simulate'>('compile');
  const [response, setResponse] = useState<NLQueryResponse | null>(null);
  const [suggestions, setSuggestions] = useState<NLQuerySuggestion[]>([]);
  const [showSuggestions, setShowSuggestions] = useState(false);

  // Conversation state
  const [conversationId, setConversationId] = useState<string | null>(null);
  const [conversationContext, setConversationContext] = useState<ConversationContext | null>(null);
  const [showConversationHistory, setShowConversationHistory] = useState(false);
  const [conversationSummary, setConversationSummary] = useState<ConversationSummary | null>(null);

  const handleSubmit = async () => {
    if (!text.trim()) return;
    setLoading(true);

    try {
      const request: NLQueryRequest = {
        text: text.trim(),
        user_id: currentUser,
        tenant_id: currentTenant,
        datasource: currentDatasource,
        conversation_id: conversationId || undefined,
      };

      let result: NLQueryResponse;
      if (mode === 'simulate') {
        result = await simulateNLQuery(request);
      } else {
        result = await compileNLQuery(request);
      }

      setResponse(result);
      onQueryGenerated?.(result);
    } catch (error) {
      devError("NL Query failed:", error);
      // TODO: Show error message to user
    } finally {
      setLoading(false);
    }
  };

  const handleGetSuggestions = async () => {
    try {
      const suggs = await getNLQuerySuggestions();
      setSuggestions(suggs);
      setShowSuggestions(true);
    } catch (error) {
      devError("Failed to get suggestions:", error);
    }
  };

  const handleSuggestionClick = (suggestion: NLQuerySuggestion) => {
    setText(suggestion.text);
    setShowSuggestions(false);
  };

  // Conversation handlers
  const handleStartConversation = async () => {
    try {
      const context = await startConversation(currentUser, currentTenant, currentDatasource);
      setConversationId(context.conversation_id);
      setConversationContext(context);
      setConversationSummary(null);
    } catch (error) {
      devError("Failed to start conversation:", error);
    }
  };

  const handleEndConversation = () => {
    setConversationId(null);
    setConversationContext(null);
    setConversationSummary(null);
    setShowConversationHistory(false);
  };

  const handleViewConversationHistory = async () => {
    if (!conversationId) return;

    try {
      const summary = await getConversationSummary(conversationId);
      setConversationSummary(summary);
      setShowConversationHistory(true);
    } catch (error) {
      devError("Failed to get conversation summary:", error);
    }
  };

  return (
    <div className="nl-query-interface">
      <div className="nl-query-input-section">
        <h3>Natural Language Query</h3>

        <div className="input-controls">
          <textarea
            value={text}
            onChange={(e) => setText(e.target.value)}
            placeholder="Ask a question in natural language...&#10;e.g., 'Show me average order value for EMEA last quarter'"
            rows={3}
            disabled={loading}
          />

          <div className="control-buttons">
            <select
              value={mode}
              onChange={(e) => setMode(e.target.value as 'compile' | 'simulate')}
              disabled={loading}
              aria-label="Query processing mode"
            >
              <option value="compile">Compile Query</option>
              <option value="simulate">Simulate Only</option>
            </select>

            <button
              onClick={handleSubmit}
              disabled={loading || !text.trim()}
              className="primary-button"
            >
              {loading ? 'Processing...' : mode === 'simulate' ? 'Simulate' : 'Generate Query'}
            </button>

            <button
              onClick={handleGetSuggestions}
              disabled={loading}
              className="secondary-button"
            >
              Get Suggestions
            </button>
          </div>
        </div>

        {/* Conversation Controls */}
        <div className="conversation-controls">
          <h4>Conversation Mode</h4>
          {!conversationId ? (
            <button
              onClick={handleStartConversation}
              className="secondary-button"
              disabled={loading}
            >
              Start New Conversation
            </button>
          ) : (
            <div className="conversation-info">
              <span className="conversation-id">Conversation: {conversationId.slice(0, 8)}...</span>
              <div className="conversation-buttons">
                <button
                  onClick={handleViewConversationHistory}
                  className="secondary-button"
                  disabled={loading}
                >
                  View History ({conversationContext?.query_history.length || 0} queries)
                </button>
                <button
                  onClick={handleEndConversation}
                  className="danger-button"
                  disabled={loading}
                >
                  End Conversation
                </button>
              </div>
            </div>
          )}
        </div>

        {showSuggestions && suggestions.length > 0 && (
          <div className="suggestions-panel">
            <h4>Suggested Queries</h4>
            <ul>
              {suggestions.map((suggestion) => (
                <li key={suggestion.text}>
                  <button
                    onClick={() => handleSuggestionClick(suggestion)}
                    className="suggestion-button"
                  >
                    {suggestion.text}
                  </button>
                  <span className="confidence">({Math.round(suggestion.confidence * 100)}% confidence)</span>
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>

      {response && (
        <div className="nl-query-results">
          <h3>Query Results</h3>

          <div className="parsed-intent">
            <h4>Parsed Intent</h4>
            <div className="intent-details">
              <div className="intent-item">
                <strong>Metrics:</strong> {response.parsed_intent.metrics.join(', ') || 'None detected'}
              </div>
              <div className="intent-item">
                <strong>Dimensions:</strong> {response.parsed_intent.dimensions.join(', ') || 'None detected'}
              </div>
              {response.parsed_intent.time_range && (
                <div className="intent-item">
                  <strong>Time Range:</strong> {response.parsed_intent.time_range.label}
                </div>
              )}
              <div className="intent-item">
                <strong>Confidence:</strong> {Math.round(response.parsed_intent.confidence * 100)}%
              </div>
            </div>
          </div>

          <div className="generated-query">
            <h4>Generated SQL</h4>
            <pre className="sql-code">{response.generated_query.sql}</pre>
          </div>

          {(response.governance_diff.blocked_metrics?.length ||
            response.governance_diff.blocked_dimensions?.length ||
            response.governance_diff.added_filters?.length) && (
            <div className="governance-diff">
              <h4>Governance Applied</h4>

              {(response.governance_diff.blocked_metrics?.length ?? 0) > 0 && (
                <div className="diff-section blocked">
                  <h5>🚫 Blocked Metrics</h5>
                  <ul>
                    {response.governance_diff.blocked_metrics?.map((metric, idx) => (
                      <li key={idx}>{metric}</li>
                    ))}
                  </ul>
                </div>
              )}

              {(response.governance_diff.blocked_dimensions?.length ?? 0) > 0 && (
                <div className="diff-section blocked">
                  <h5>🚫 Blocked Dimensions</h5>
                  <ul>
                    {response.governance_diff.blocked_dimensions?.map((dim, idx) => (
                      <li key={idx}>{dim}</li>
                    ))}
                  </ul>
                </div>
              )}

              {(response.governance_diff.added_filters?.length ?? 0) > 0 && (
                <div className="diff-section added">
                  <h5>✅ Added Security Filters</h5>
                  <ul>
                    {response.governance_diff.added_filters?.map((filter, idx) => (
                      <li key={idx}>{filter.field} {filter.operator} {filter.value}</li>
                    ))}
                  </ul>
                </div>
              )}

              {(response.governance_diff.applied_policies?.length ?? 0) > 0 && (
                <div className="diff-section policies">
                  <h5>📋 Applied Policies</h5>
                  <ul>
                    {response.governance_diff.applied_policies?.map((policy, idx) => (
                      <li key={idx}>
                        <strong>{policy.policy_id}:</strong> {policy.reason}
                      </li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
          )}

          {response.compliance_notes.length > 0 && (
            <div className="compliance-notes">
              <h4>Compliance Notes</h4>
              <ul>
                {response.compliance_notes.map((note, idx) => (
                  <li key={idx}>{note}</li>
                ))}
              </ul>
            </div>
          )}

          {response.warnings.length > 0 && (
            <div className="warnings">
              <h4>⚠️ Warnings</h4>
              <ul>
                {response.warnings.map((warning, idx) => (
                  <li key={idx}>{warning}</li>
                ))}
              </ul>
            </div>
          )}

          <div className="query-metadata">
            <div className="metadata-item">
              <strong>Query ID:</strong> {response.query_id}
            </div>
            <div className="metadata-item">
              <strong>Generated:</strong> {new Date(response.timestamp).toLocaleString()}
            </div>
          </div>
        </div>
      )}

      {showConversationHistory && conversationSummary && (
        <div className="conversation-history">
          <h3>Conversation History</h3>

          <div className="conversation-summary">
            <div className="summary-item">
              <strong>Total Queries:</strong> {conversationSummary.query_count}
            </div>
            <div className="summary-item">
              <strong>Duration:</strong> {conversationSummary.duration}
            </div>
            <div className="summary-item">
              <strong>Success Rate:</strong> {conversationSummary.insights.total_queries > 0 ?
                Math.round((conversationSummary.insights.successful_queries / conversationSummary.insights.total_queries) * 100) : 0}%
            </div>
            <div className="summary-item">
              <strong>Avg Confidence:</strong> {Math.round(conversationSummary.insights.avg_confidence * 100)}%
            </div>
          </div>

          {conversationSummary.insights.common_metrics.length > 0 && (
            <div className="common-insights">
              <h4>Common Metrics</h4>
              <ul>
                {conversationSummary.insights.common_metrics.map((metric, idx) => (
                  <li key={idx}>{metric}</li>
                ))}
              </ul>
            </div>
          )}

          {conversationSummary.insights.common_dimensions.length > 0 && (
            <div className="common-insights">
              <h4>Common Dimensions</h4>
              <ul>
                {conversationSummary.insights.common_dimensions.map((dim, idx) => (
                  <li key={idx}>{dim}</li>
                ))}
              </ul>
            </div>
          )}

          <div className="query-history-list">
            <h4>Query History</h4>
            <ul>
              {conversationSummary.query_history.map((query, idx) => (
                <li key={idx} className={`history-item ${query.success ? 'success' : 'failed'}`}>
                  <div className="query-text">{query.user_query}</div>
                  <div className="query-meta">
                    <span className="timestamp">{new Date(query.executed_at).toLocaleString()}</span>
                    <span className={`status ${query.success ? 'success' : 'failed'}`}>
                      {query.success ? '✓' : '✗'}
                    </span>
                  </div>
                </li>
              ))}
            </ul>
          </div>

          <button
            onClick={() => setShowConversationHistory(false)}
            className="secondary-button"
          >
            Close History
          </button>
        </div>
      )}
    </div>
  );
}
