import React, { useState, useEffect } from 'react';
import { fetchAPI } from '../../api';
import './LLMConfigPage.css';

interface LLMConfig {
  provider: string;
  model: string;
  embedding_model: string;
  api_key?: string;
  params?: {
    temperature?: number;
    [key: string]: any;
  };
}

const PROVIDERS = [
  { value: 'gemini', label: 'Google Gemini', models: ['gemini-2.0-flash-exp', 'gemini-1.5-pro', 'gemini-1.5-flash'] },
  { value: 'openai', label: 'OpenAI', models: ['gpt-4', 'gpt-4-turbo', 'gpt-3.5-turbo'] },
  { value: 'anthropic', label: 'Anthropic Claude', models: ['claude-3-opus', 'claude-3-sonnet', 'claude-3-haiku'] },
];

export default function LLMConfigPage() {
  const [config, setConfig] = useState<LLMConfig>({
    provider: 'gemini',
    model: 'gemini-2.0-flash-exp',
    embedding_model: 'text-embedding-004',
    api_key: '', // Initialize with empty string so it's falsy but defined
    params: {
      temperature: 0.2,
    },
  });

  const [apiKey, setApiKey] = useState('');
  const [testQuestion, setTestQuestion] = useState('');
  const [testResult, setTestResult] = useState<string | null>(null);
  const [testLoading, setTestLoading] = useState(false);
  const [saveLoading, setSaveLoading] = useState(false);
  const [saveSuccess, setSaveSuccess] = useState(false);
  const [configLoading, setConfigLoading] = useState(true);

  useEffect(() => {
    loadConfig();
  }, []);

  const loadConfig = async () => {
    try {
      const response = await fetchAPI<LLMConfig>('/admin/llm/config');
      console.log('Loaded LLM config:', response);
      console.log('API Key present:', !!response.api_key);
      setConfig(response);
    } catch (error) {
      console.error('Failed to load LLM config:', error);
    } finally {
      setConfigLoading(false);
    }
  };

  const handleSave = async () => {
    setSaveLoading(true);
    setSaveSuccess(false);

    try {
      await fetchAPI('/admin/llm/config', {
        method: 'PUT',
        body: JSON.stringify({
          ...config,
          api_key: apiKey || undefined,
        }),
      });

      setSaveSuccess(true);
      setApiKey(''); // Clear the API key input after saving
      
      setTimeout(() => setSaveSuccess(false), 3000);
    } catch (error) {
      console.error('Failed to save LLM config:', error);
      alert('Failed to save configuration. Please try again.');
    } finally {
      setSaveLoading(false);
    }
  };

  const handleTest = async () => {
    if (!testQuestion.trim()) {
      alert('Please enter a test question');
      return;
    }

    // Use provided API key or the saved one
    const effectiveApiKey = apiKey || config.api_key;
    if (!effectiveApiKey) {
      alert('Please provide an API key before testing');
      return;
    }

    setTestLoading(true);
    setTestResult(null);

    try {
      const response = await fetchAPI<{ response: string }>('/admin/llm/test', {
        method: 'POST',
        body: JSON.stringify({
          prompt: testQuestion,
          api_key: apiKey || undefined, // Only send if user provided a new one
        }),
      });

      setTestResult(response.response);
    } catch (error) {
      console.error('LLM test failed:', error);
      setTestResult(`Error: ${error instanceof Error ? error.message : 'Unknown error'}`);
    } finally {
      setTestLoading(false);
    }
  };

  const selectedProvider = PROVIDERS.find(p => p.value === config.provider);

  return (
    <div className="llm-config-page">
      <div className="llm-config-header">
        <h1>🤖 LLM Provider Configuration</h1>
        <p>Configure the AI model used for Natural Language Q&A</p>
        <div className="config-version">v1.2 - Status: {configLoading ? 'Loading...' : 'Ready'}</div>
      </div>

      <div className="llm-config-container">
        <div className="config-section">
          <h2>Provider Settings</h2>

          <div className="form-group">
            <label htmlFor="provider-select">Provider</label>
            <select
              id="provider-select"
              value={config.provider}
              onChange={(e) => setConfig({ ...config, provider: e.target.value })}
              className="form-control"
            >
              {PROVIDERS.map((provider) => (
                <option key={provider.value} value={provider.value}>
                  {provider.label}
                </option>
              ))}
            </select>
          </div>

          <div className="form-group">
            <label htmlFor="model-select">Model</label>
            <select
              id="model-select"
              value={config.model}
              onChange={(e) => setConfig({ ...config, model: e.target.value })}
              className="form-control"
            >
              {selectedProvider?.models.map((model) => (
                <option key={model} value={model}>
                  {model}
                </option>
              ))}
            </select>
          </div>

          <div className="form-group">
            <label>API Key</label>
            <input
              type="password"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              placeholder={config.api_key ? '••••••••' : 'Enter API key'}
              className="form-control"
            />
            <small className="form-help">
              {config.api_key 
                ? '✅ API key is configured. Leave blank to keep current key.'
                : '⚠️ No API key configured. Enter a key to enable the service.'}
            </small>
          </div>

          {config.provider === 'gemini' && (
            <div className="form-group">
              <label htmlFor="embedding-model-input">Embedding Model</label>
              <input
                id="embedding-model-input"
                type="text"
                value={config.embedding_model}
                onChange={(e) => setConfig({ ...config, embedding_model: e.target.value })}
                className="form-control"
                aria-label="Embedding model name"
              />
            </div>
          )}
        </div>

        <div className="config-section">
          <h2>Model Parameters</h2>

          <div className="form-group">
            <label htmlFor="temperature-slider">Temperature: {config.params?.temperature ?? 0.2}</label>
            <input
              id="temperature-slider"
              type="range"
              min="0"
              max="1"
              step="0.1"
              value={config.params?.temperature ?? 0.2}
              onChange={(e) => setConfig({ ...config, params: { ...config.params, temperature: parseFloat(e.target.value) } })}
              className="form-range"
              aria-label="Temperature setting"
            />
            <small className="form-help">
              Lower values = more focused, higher values = more creative
            </small>
          </div>
        </div>

        <div className="config-section">
          <h2>Test Configuration</h2>

          <div className="form-group">
            <label>Test Question</label>
            <textarea
              value={testQuestion}
              onChange={(e) => setTestQuestion(e.target.value)}
              placeholder="Enter a test question to verify the LLM configuration (e.g., 'What is the capital of France?')..."
              className="form-control"
              rows={3}
            />
          </div>

          <button
            onClick={handleTest}
            disabled={testLoading || configLoading || !testQuestion.trim()}
            className="btn btn-secondary"
            title={configLoading ? 'Loading config...' : !testQuestion.trim() ? 'Please enter a test question' : ''}
          >
            {testLoading ? 'Testing...' : 'Test Configuration'}
          </button>

          {testResult && (
            <div className={`test-result ${testResult.startsWith('Error:') ? 'error' : 'success'}`}>
              <h4>Test Result:</h4>
              <pre>{testResult}</pre>
            </div>
          )}
        </div>

        <div className="config-actions">
          <button
            onClick={handleSave}
            disabled={saveLoading}
            className="btn btn-primary"
          >
            {saveLoading ? 'Saving...' : 'Save Configuration'}
          </button>

          {saveSuccess && (
            <div className="save-success">
              ✅ Configuration saved successfully!
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
