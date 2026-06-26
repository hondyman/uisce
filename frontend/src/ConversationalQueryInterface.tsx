import { useState, useEffect, useRef } from 'react';
import { devError } from './utils/devLogger';
import {
  startConversation,
  sendConversationMessage,
  commitConversationQuery,
  getConversationSummary
} from './api';
import type {
  RefinementContext,
  ConversationMessage,
  ConversationSummary as ConvSummary
} from './types';
import './ConversationalQueryInterface.css';

interface ConversationalQueryInterfaceProps {
  currentDatasource: string;
  currentUser: string;
  currentTenant: string;
  onQueryGenerated?: (query: any) => void;
}

export default function ConversationalQueryInterface({
  currentDatasource,
  currentUser,
  currentTenant,
  onQueryGenerated
}: ConversationalQueryInterfaceProps) {
  const [conversationId, setConversationId] = useState<string | null>(null);
  const [refinementContext, setRefinementContext] = useState<RefinementContext | null>(null);
  const [inputMessage, setInputMessage] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [showSummary, setShowSummary] = useState(false);
  const [conversationSummary, setConversationSummary] = useState<ConvSummary | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [refinementContext?.messages]);

  const handleStartConversation = async () => {
    try {
      setIsLoading(true);
      const context = await startConversation(currentUser, currentTenant, currentDatasource);
      setConversationId(context.conversation_id);
      // Initialize refinement context
      const initialRefinement: RefinementContext = {
        conversation_id: context.conversation_id,
        current_state: 'initializing',
        current_query: null,
        messages: [{
          id: 'system-welcome',
          type: 'system',
          content: 'Welcome to conversational query building! Describe what data you need and I\'ll help you create a compliant query.',
          timestamp: new Date().toISOString(),
        }],
        clarifications: [],
        suggestions: [],
        compliance_status: {
          is_compliant: true,
          violations: [],
          applied_policies: [],
          risk_level: 'low',
        },
        last_updated: new Date().toISOString(),
      };
      setRefinementContext(initialRefinement);
    } catch (error) {
      devError('Failed to start conversation:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleSendMessage = async () => {
    if (!inputMessage.trim() || !conversationId) return;

    try {
      setIsLoading(true);
      const userMessage: ConversationMessage = {
        id: `user-${Date.now()}`,
        type: 'user',
        content: inputMessage.trim(),
        timestamp: new Date().toISOString(),
      };

      // Add user message to local state
      setRefinementContext(prev => prev ? {
        ...prev,
        messages: [...prev.messages, userMessage],
      } : null);

      setInputMessage('');

      // Send to backend
      const updatedContext = await sendConversationMessage(conversationId, inputMessage.trim());
      setRefinementContext(updatedContext);
    } catch (error) {
      devError('Failed to send message:', error);
      // Add error message
      const errorMessage: ConversationMessage = {
        id: `error-${Date.now()}`,
        type: 'system',
        content: 'Sorry, I encountered an error processing your message. Please try again.',
        timestamp: new Date().toISOString(),
      };
      setRefinementContext(prev => prev ? {
        ...prev,
        messages: [...prev.messages, errorMessage],
      } : null);
    } finally {
      setIsLoading(false);
    }
  };

  const handleAcceptSuggestion = async (suggestionId: string) => {
    if (!conversationId) return;

    try {
      setIsLoading(true);
      const response = await sendConversationMessage(conversationId, `Accept suggestion: ${suggestionId}`);
      setRefinementContext(response);
    } catch (error) {
      devError('Failed to accept suggestion:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleAnswerClarification = async (clarificationId: string, answer: string) => {
    if (!conversationId) return;

    try {
      setIsLoading(true);
      const response = await sendConversationMessage(conversationId, `Clarification ${clarificationId}: ${answer}`);
      setRefinementContext(response);
    } catch (error) {
      devError('Failed to answer clarification:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleCommitQuery = async () => {
    if (!conversationId) return;

    try {
      setIsLoading(true);
      const queryResponse = await commitConversationQuery(conversationId);
      onQueryGenerated?.(queryResponse);

      // Add success message
      const successMessage: ConversationMessage = {
        id: `commit-${Date.now()}`,
        type: 'system',
        content: 'Query committed successfully! You can now use this query in your analysis.',
        timestamp: new Date().toISOString(),
      };
      setRefinementContext(prev => prev ? {
        ...prev,
        messages: [...prev.messages, successMessage],
        current_state: 'committed',
      } : null);
    } catch (error) {
      devError('Failed to commit query:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleViewSummary = async () => {
    if (!conversationId) return;

    try {
      const summary = await getConversationSummary(conversationId);
      setConversationSummary(summary);
      setShowSummary(true);
    } catch (error) {
      devError('Failed to get conversation summary:', error);
    }
  };

  const handleEndConversation = () => {
    setConversationId(null);
    setRefinementContext(null);
    setConversationSummary(null);
    setShowSummary(false);
  };

  return (
    <div className="conversational-query-interface">
      <div className="conversation-header">
        <h3>Conversational Query Builder</h3>
        {!conversationId ? (
          <button
            onClick={handleStartConversation}
            disabled={isLoading}
            className="start-conversation-btn"
          >
            {isLoading ? 'Starting...' : 'Start New Conversation'}
          </button>
        ) : (
          <div className="conversation-controls">
            <div className="conversation-info">
              <span className="conversation-id">ID: {conversationId.slice(0, 8)}...</span>
              <span className={`conversation-state state-${refinementContext?.current_state || 'initializing'}`}>
                {refinementContext?.current_state || 'initializing'}
              </span>
            </div>
            <div className="control-buttons">
              <button
                onClick={handleViewSummary}
                disabled={isLoading}
                className="summary-btn"
              >
                Summary
              </button>
              <button
                onClick={handleEndConversation}
                disabled={isLoading}
                className="end-btn"
              >
                End Conversation
              </button>
            </div>
          </div>
        )}
      </div>

      {refinementContext && (
        <div className="conversation-content">
          <div className="messages-container">
            <div className="messages-list">
              {refinementContext.messages.map((message) => (
                <div key={message.id} className={`message ${message.type}`}>
                  <div className="message-content">
                    {message.content}
                  </div>
                  <div className="message-timestamp">
                    {new Date(message.timestamp).toLocaleTimeString()}
                  </div>
                </div>
              ))}

              {/* Clarifications */}
              {refinementContext.clarifications.map((clarification) => (
                <div key={clarification.id} className="clarification">
                  <div className="clarification-question">
                    {clarification.question}
                  </div>
                  {clarification.options ? (
                    <div className="clarification-options">
                      {clarification.options.map((option, idx) => (
                        <button
                          key={idx}
                          onClick={() => handleAnswerClarification(clarification.id, option)}
                          disabled={isLoading}
                          className="option-btn"
                        >
                          {option}
                        </button>
                      ))}
                    </div>
                  ) : (
                    <div className="clarification-input">
                      <input
                        type="text"
                        placeholder="Your answer..."
                        onKeyPress={(e) => {
                          if (e.key === 'Enter') {
                            handleAnswerClarification(clarification.id, (e.target as HTMLInputElement).value);
                          }
                        }}
                        disabled={isLoading}
                      />
                    </div>
                  )}
                </div>
              ))}

              {/* Suggestions */}
              {refinementContext.suggestions.map((suggestion) => (
                <div key={suggestion.id} className="suggestion">
                  <div className="suggestion-header">
                    <div className="suggestion-description">{suggestion.description}</div>
                    <div className="suggestion-confidence">
                      {Math.round(suggestion.confidence * 100)}% confidence
                    </div>
                  </div>
                  <div className="suggestion-reasoning">{suggestion.reasoning}</div>
                  <div className="suggestion-actions">
                    <button
                      onClick={() => handleAcceptSuggestion(suggestion.id)}
                      disabled={isLoading}
                      className="accept-btn"
                    >
                      Accept
                    </button>
                    <button
                      onClick={() => handleSendMessage()}
                      disabled={isLoading}
                      className="reject-btn"
                    >
                      Reject
                    </button>
                  </div>
                </div>
              ))}

              <div ref={messagesEndRef} />
            </div>
          </div>

          {/* Live Query Preview */}
          {refinementContext.current_query && (
            <div className="query-preview">
              <h4>Current Query Preview</h4>
              <div className="query-sql">
                <pre>{refinementContext.current_query.sql}</pre>
              </div>

              {/* Compliance Status */}
              <div className="compliance-status">
                <div className={`compliance-badge compliance-${refinementContext.compliance_status.is_compliant ? 'compliant' : 'non-compliant'} risk-${refinementContext.compliance_status.risk_level}`}>
                  {refinementContext.compliance_status.is_compliant ? '✅ Compliant' : '❌ Non-Compliant'}
                </div>
                {refinementContext.compliance_status.violations.length > 0 && (
                  <div className="violations-list">
                    <h5>Policy Violations:</h5>
                    <ul>
                      {refinementContext.compliance_status.violations.map((violation, idx) => (
                        <li key={idx} className={`violation ${violation.severity}`}>
                          {violation.description}
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
              </div>

              {refinementContext.current_state === 'ready' && (
                <button
                  onClick={handleCommitQuery}
                  disabled={isLoading}
                  className="commit-btn"
                >
                  {isLoading ? 'Committing...' : 'Commit Query'}
                </button>
              )}
            </div>
          )}

          {/* Message Input */}
          <div className="message-input-container">
            <div className="message-input">
              <input
                type="text"
                value={inputMessage}
                onChange={(e) => setInputMessage(e.target.value)}
                onKeyPress={(e) => e.key === 'Enter' && handleSendMessage()}
                placeholder="Describe your data needs or respond to clarifications..."
                disabled={isLoading}
              />
              <button
                onClick={handleSendMessage}
                disabled={isLoading || !inputMessage.trim()}
                className="send-btn"
              >
                {isLoading ? '...' : 'Send'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Conversation Summary Modal */}
      {showSummary && conversationSummary && (
        <div className="summary-modal">
          <div className="summary-content">
            <h3>Conversation Summary</h3>

            <div className="summary-stats">
              <div className="stat-item">
                <strong>Total Queries:</strong> {conversationSummary.query_count}
              </div>
              <div className="stat-item">
                <strong>Success Rate:</strong> {conversationSummary.insights.total_queries > 0 ?
                  Math.round((conversationSummary.insights.successful_queries / conversationSummary.insights.total_queries) * 100) : 0}%
              </div>
              <div className="stat-item">
                <strong>Avg Confidence:</strong> {Math.round(conversationSummary.insights.avg_confidence * 100)}%
              </div>
              <div className="stat-item">
                <strong>Duration:</strong> {conversationSummary.duration}
              </div>
            </div>

            {conversationSummary.insights.common_metrics.length > 0 && (
              <div className="common-insights">
                <h4>Common Metrics</h4>
                <div className="tags">
                  {conversationSummary.insights.common_metrics.map((metric, idx) => (
                    <span key={idx} className="tag">{metric}</span>
                  ))}
                </div>
              </div>
            )}

            {conversationSummary.insights.common_dimensions.length > 0 && (
              <div className="common-insights">
                <h4>Common Dimensions</h4>
                <div className="tags">
                  {conversationSummary.insights.common_dimensions.map((dim, idx) => (
                    <span key={idx} className="tag">{dim}</span>
                  ))}
                </div>
              </div>
            )}

            <button
              onClick={() => setShowSummary(false)}
              className="close-summary-btn"
            >
              Close
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
