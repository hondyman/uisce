// frontend/src/components/editor/AiActions.tsx
import React, { useState } from 'react';
import styles from './AiActions.module.css';

interface AILayoutOption {
  id: string;
  name: string;
  layoutType: string;
  sections?: any[];
}

interface AIResponse {
  generatedLayout: AILayoutOption;
  confidence: number;
  alternatives: AILayoutOption[];
  explanation: string;
  modelVersion: string;
  generatedAt: string;
  draftId: string;
}

export const AiActions: React.FC<{
  primaryBO: string;
  tenantId: string;
  onApplyLayout: (layout: any, draftId: string) => void;
  loading?: boolean;
}> = ({ primaryBO, tenantId, onApplyLayout, loading = false }) => {
  const [prompt, setPrompt] = useState('Create a detail layout with basic information and related records');
  const [suggestions, setSuggestions] = useState<AIResponse | null>(null);
  const [isGenerating, setIsGenerating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const generate = async () => {
    if (!prompt.trim()) {
      setError('Please enter a prompt');
      return;
    }

    setIsGenerating(true);
    setError(null);

    try {
      const res = await fetch('/api/ai/generate-layout', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
        },
        body: JSON.stringify({ prompt, primaryBO }),
      });

      if (!res.ok) {
        throw new Error(`API error: ${res.status}`);
      }

      const data: AIResponse = await res.json();
      setSuggestions(data);
    } catch (err) {
      setError(`Error generating layout: ${err instanceof Error ? err.message : 'Unknown error'}`);
    } finally {
      setIsGenerating(false);
    }
  };

  const handleApply = (layout: AILayoutOption, draftId: string) => {
    onApplyLayout(layout, draftId);
    setSuggestions(null);
    setPrompt('');
  };

  return (
    <div className={styles.container}>
      <div className={styles.inputGroup}>
        <input
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
          placeholder="Describe your desired layout (e.g., 'Customer detail with 3 columns')"
          disabled={isGenerating || loading}
          className={styles.input}
          onKeyDown={(e) => {
            if (e.key === 'Enter') generate();
          }}
        />
        <button 
          onClick={generate} 
          disabled={isGenerating || loading || !prompt.trim()}
          className={styles.generateBtn}
        >
          {isGenerating ? '✨ Generating…' : '✨ Generate with AI'}
        </button>
      </div>

      {error && (
        <div className={styles.error}>
          {error}
        </div>
      )}

      {suggestions && (
        <div className={styles.suggestionsPanel}>
          <div className={styles.header}>
            <h3>AI Suggestions</h3>
            <button 
              className={styles.closeBtn}
              onClick={() => setSuggestions(null)}
            >
              ✕
            </button>
          </div>

          <div className={styles.mainSuggestion}>
            <div className={styles.suggestionCard}>
              <div className={styles.title}>{suggestions.generatedLayout.name}</div>
              <div className={styles.meta}>
                <span>Confidence: {(suggestions.confidence * 100).toFixed(0)}%</span>
                <span>Type: {suggestions.generatedLayout.layoutType}</span>
                <span>Sections: {suggestions.generatedLayout.sections?.length || 0}</span>
              </div>
              <div className={styles.explanation}>{suggestions.explanation}</div>
              <button 
                className={styles.applyBtn}
                onClick={() => handleApply(suggestions.generatedLayout, suggestions.draftId)}
              >
                Apply
              </button>
            </div>
          </div>

          {suggestions.alternatives && suggestions.alternatives.length > 0 && (
            <div className={styles.alternatives}>
              <div className={styles.altHeader}>Alternative Options</div>
              <div className={styles.altGrid}>
                {suggestions.alternatives.map((alt, i) => (
                  <div key={alt.id} className={styles.altCard}>
                    <div className={styles.altTitle}>{alt.name}</div>
                    <div className={styles.altMeta}>
                      <span>Type: {alt.layoutType}</span>
                      <span>Sections: {alt.sections?.length || 0}</span>
                    </div>
                    <button 
                      className={styles.applyBtn}
                      onClick={() => handleApply(alt, `alt-${i}`)}
                    >
                      Apply
                    </button>
                  </div>
                ))}
              </div>
            </div>
          )}

          <div className={styles.footer}>
            <small>Model: {suggestions.modelVersion} • Generated: {new Date(suggestions.generatedAt).toLocaleTimeString()}</small>
          </div>
        </div>
      )}
    </div>
  );
};
