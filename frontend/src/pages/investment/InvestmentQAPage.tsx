import React, { useState } from 'react';
import './InvestmentQAPage.css';

interface CalculationStep {
  step: string;
  source?: string;
  rule?: string;
  description?: string;
}

interface DataQuality {
  freshness: string;
  null_rate: number;
  sla: string;
  lineage_note?: string;
  freshness_status: string;
}

interface QAResponse {
  answer: string;
  calculation_breakdown?: CalculationStep[];
  sources: string[];
  caveats?: string[];
  confidence: string;
  provider?: string;
  data_quality?: DataQuality;
}

export const InvestmentQAPage: React.FC = () => {
  const [question, setQuestion] = useState('');
  const [response, setResponse] = useState<QAResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleAsk = async () => {
    if (!question.trim()) return;

    setLoading(true);
    setError(null);

    try {
      const res = await fetch('/nlq/ask', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          tenant_id: 'demo-tenant-123',  // Replace with actual tenant from auth
          question,
          portfolio_id: 'demo-portfolio', // Optional
        }),
      });

      if (!res.ok) {
        throw new Error(`API error: ${res.status}`);
      }

      const data = await res.json();
      setResponse(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to get answer');
    } finally {
      setLoading(false);
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleAsk();
    }
  };

  return (
    <div className="investment-qa-page">
      <div className="qa-container">
        <header className="qa-header">
          <h1>Investment Q&A</h1>
          <p>Ask questions about your portfolio, metrics, and market data</p>
        </header>

        <div className="qa-input-section">
          <textarea
            className="qa-input"
            placeholder="Ask a question... (e.g., 'What is my portfolio NAV?' or 'What is MSFT trading at?')"
            value={question}
            onChange={(e) => setQuestion(e.target.value)}
            onKeyPress={handleKeyPress}
            rows={3}
            disabled={loading}
          />
          <button
            className="qa-submit-btn"
            onClick={handleAsk}
            disabled={loading || !question.trim()}
          >
            {loading ? 'Thinking...' : 'Ask'}
          </button>
        </div>

        {error && (
          <div className="qa-error">
            <strong>Error:</strong> {error}
          </div>
        )}

        {response && (
          <div className="qa-response">
            <div className="answer-section">
              <h2>Answer</h2>
              <p className="answer-text">{response.answer}</p>
              
              <div className="metadata">
                <span className={`confidence confidence-${response.confidence.toLowerCase()}`}>
                  Confidence: {response.confidence}
                </span>
                {response.provider && (
                  <span className="provider">Provider: {response.provider}</span>
                )}
              </div>
            </div>

            {response.calculation_breakdown && response.calculation_breakdown.length > 0 && (
              <div className="breakdown-section">
                <h3>Calculation Breakdown</h3>
                <div className="breakdown-steps">
                  {response.calculation_breakdown.map((step, idx) => (
                    <div key={idx} className="breakdown-step">
                      <div className="step-number">{idx + 1}</div>
                      <div className="step-content">
                        <strong>{step.step}</strong>
                        {step.source && <div className="step-source">Source: {step.source}</div>}
                        {step.rule && <div className="step-rule">{step.rule}</div>}
                        {step.description && <div className="step-desc">{step.description}</div>}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {response.sources && response.sources.length > 0 && (
              <div className="sources-section">
                <h3>Sources</h3>
                <div className="source-list">
                  {response.sources.map((source, idx) => (
                    <span key={idx} className="source-badge">{source}</span>
                  ))}
                </div>
              </div>
            )}

            {response.caveats && response.caveats.length > 0 && (
              <div className="caveats-section">
                <h3>Data Quality Caveats</h3>
                <ul className="caveat-list">
                  {response.caveats.map((caveat, idx) => (
                    <li key={idx}>{caveat}</li>
                  ))}
                </ul>
              </div>
            )}

            {response.data_quality && (
              <div className="data-quality-section">
                <h3>Data Quality Metrics</h3>
                <div className="quality-grid">
                  <div className="quality-card">
                    <div className="quality-label">Freshness</div>
                    <div className={`quality-value freshness-${response.data_quality.freshness_status.toLowerCase()}`}>
                      {response.data_quality.freshness}
                    </div>
                    <div className="quality-status">{response.data_quality.freshness_status}</div>
                  </div>
                  <div className="quality-card">
                    <div className="quality-label">Null Rate</div>
                    <div className="quality-value">
                      {(response.data_quality.null_rate * 100).toFixed(2)}%
                    </div>
                  </div>
                  <div className="quality-card">
                    <div className="quality-label">SLA Compliance</div>
                    <div className="quality-value">{response.data_quality.sla}</div>
                  </div>
                  {response.data_quality.lineage_note && (
                    <div className="quality-card lineage-note">
                      <div className="quality-label">Lineage</div>
                      <div className="quality-value">{response.data_quality.lineage_note}</div>
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        )}

        <div className="example-questions">
          <h3>Example Questions</h3>
          <div className="example-grid">
            <button
              className="example-btn"
              onClick={() => setQuestion("What is my portfolio NAV?")}
            >
              What is my portfolio NAV?
            </button>
            <button
              className="example-btn"
              onClick={() => setQuestion("What is MSFT trading at?")}
            >
              What is MSFT trading at?
            </button>
            <button
              className="example-btn"
              onClick={() => setQuestion("What is my portfolio VaR?")}
            >
              What is my portfolio VaR?
            </button>
            <button
              className="example-btn"
              onClick={() => setQuestion("Show me factor exposures")}
            >
              Show me factor exposures
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
