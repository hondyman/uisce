import React, { useState, useRef, useEffect } from 'react';
import { fetchAPI } from '../../api';
import { useTenant } from '../../contexts/TenantContext';
import './NLQPage.css';

interface SourceReference {
  path: string;
  name: string;
  type: string;
  description?: string;
  metadata?: Record<string, any>;
}

interface AskResponse {
  answer: string;
  calculation_breakdown?: Record<string, any>;
  sources: SourceReference[];
  confidence: string;
  resolved_entity_path?: string;
  caveats?: string[];
}

interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  response?: AskResponse;
  timestamp: Date;
  loading?: boolean;
}

export default function NLQPage() {
  const { tenant, datasource } = useTenant();
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!input.trim() || !tenant || !datasource) return;

    const userMessage: Message = {
      id: Date.now().toString(),
      role: 'user',
      content: input,
      timestamp: new Date(),
    };

    setMessages(prev => [...prev, userMessage]);
    setInput('');
    setLoading(true);

    // Add loading message
    const loadingMessage: Message = {
      id: `${Date.now()}-loading`,
      role: 'assistant',
      content: 'Thinking...',
      timestamp: new Date(),
      loading: true,
    };
    setMessages(prev => [...prev, loadingMessage]);

    try {
      const response = await fetchAPI<AskResponse>('/nlq/ask', {
        method: 'POST',
        body: JSON.stringify({
          question: input,
          // target_entity_path is optional - will be auto-discovered
        }),
      });

      // Replace loading message with actual response
      setMessages(prev => 
        prev.filter(m => m.id !== loadingMessage.id).concat({
          id: `${Date.now()}-response`,
          role: 'assistant',
          content: response.answer,
          response,
          timestamp: new Date(),
        })
      );
    } catch (error) {
      console.error('NLQ request failed:', error);
      
      // Replace loading message with error
      setMessages(prev =>
        prev.filter(m => m.id !== loadingMessage.id).concat({
          id: `${Date.now()}-error`,
          role: 'assistant',
          content: `Sorry, I encountered an error: ${error instanceof Error ? error.message : 'Unknown error'}`,
          timestamp: new Date(),
        })
      );
    } finally {
      setLoading(false);
    }
  };

  const exampleQuestions = [
    'How is monthly revenue calculated?',
    'What metrics are available for customer analysis?',
    'Show me the data quality for orders table',
    'What are the dependencies for the revenue metric?',
  ];

  const handleExampleClick = (question: string) => {
    setInput(question);
  };

  if (!tenant || !datasource) {
    return (
      <div className="nlq-page">
        <div className="nlq-warning">
          <h2>⚠️ Tenant Scope Required</h2>
          <p>Please select a tenant and datasource to use the Natural Language Q&A feature.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="nlq-page">
      <div className="nlq-header">
        <h1>📚 Ask About Your Data Catalog</h1>
        <p>Ask questions about metrics, calculations, data quality, and more</p>
        <div className="nlq-scope-badge">
          <span className="scope-label">Tenant:</span> {tenant.display_name}
          <span className="scope-separator">|</span>
          <span className="scope-label">Datasource:</span> {datasource.source_name}
        </div>
      </div>

      <div className="nlq-container">
        {messages.length === 0 ? (
          <div className="nlq-welcome">
            <h2>Welcome to Natural Language Q&A</h2>
            <p>Ask me anything about your data catalog. I can help you understand:</p>
            <ul>
              <li>How calculations and metrics are built</li>
              <li>What data sources are used</li>
              <li>Data quality and freshness information</li>
              <li>Dependencies and relationships between entities</li>
            </ul>
            
            <div className="example-questions">
              <h3>Try these examples:</h3>
              {exampleQuestions.map((q, idx) => (
                <button
                  key={idx}
                  className="example-question"
                  onClick={() => handleExampleClick(q)}
                >
                  {q}
                </button>
              ))}
            </div>
          </div>
        ) : (
          <div className="messages-container">
            {messages.map((message) => (
              <div key={message.id} className={`message message-${message.role}`}>
                <div className="message-header">
                  <span className="message-role">
                    {message.role === 'user' ? '👤 You' : '🤖 Assistant'}
                  </span>
                  <span className="message-timestamp">
                    {message.timestamp.toLocaleTimeString()}
                  </span>
                </div>
                
                <div className="message-content">
                  {message.loading ? (
                    <div className="loading-indicator">
                      <span className="dot"></span>
                      <span className="dot"></span>
                      <span className="dot"></span>
                    </div>
                  ) : (
                    <p>{message.content}</p>
                  )}
                </div>

                {message.response && (
                  <div className="message-metadata">
                    {message.response.resolved_entity_path && (
                      <div className="metadata-item">
                        <strong>Resolved Entity:</strong>
                        <code>{message.response.resolved_entity_path}</code>
                      </div>
                    )}

                    {message.response.sources && message.response.sources.length > 0 && (
                      <div className="metadata-item">
                        <strong>Sources:</strong>
                        <div className="sources-list">
                          {message.response.sources.map((source, idx) => (
                            <div key={idx} className="source-item">
                              <code className="source-path">{source.path}</code>
                              <span className="source-type">{source.type}</span>
                              {source.metadata?.data_quality && (
                                <div className="source-metadata">
                                  {source.metadata.data_quality.freshness && (
                                    <span className="metadata-badge">
                                      Freshness: {source.metadata.data_quality.freshness}
                                    </span>
                                  )}
                                  {source.metadata.data_quality.null_rate && (
                                    <span className="metadata-badge">
                                      Null Rate: {source.metadata.data_quality.null_rate}
                                    </span>
                                  )}
                                </div>
                              )}
                            </div>
                          ))}
                        </div>
                      </div>
                    )}

                    {message.response.caveats && message.response.caveats.length > 0 && (
                      <div className="metadata-item caveats">
                        <strong>⚠️ Caveats:</strong>
                        <ul>
                          {message.response.caveats.map((caveat, idx) => (
                            <li key={idx}>{caveat}</li>
                          ))}
                        </ul>
                      </div>
                    )}

                    {message.response.calculation_breakdown && (
                      <details className="calculation-breakdown">
                        <summary>
                          <strong>📊 Calculation Details</strong>
                        </summary>
                        <pre>{JSON.stringify(message.response.calculation_breakdown, null, 2)}</pre>
                      </details>
                    )}

                    <div className="metadata-item confidence">
                      <strong>Confidence:</strong>
                      <span className={`confidence-badge confidence-${message.response.confidence.toLowerCase()}`}>
                        {message.response.confidence}
                      </span>
                    </div>
                  </div>
                )}
              </div>
            ))}
            <div ref={messagesEndRef} />
          </div>
        )}

        <form className="nlq-input-form" onSubmit={handleSubmit}>
          <input
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="Ask a question about your data catalog..."
            disabled={loading}
            className="nlq-input"
          />
          <button
            type="submit"
            disabled={loading || !input.trim()}
            className="nlq-submit-button"
          >
            {loading ? 'Thinking...' : 'Ask'}
          </button>
        </form>
      </div>
    </div>
  );
}
