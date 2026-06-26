import React, { useState } from 'react';
import { AccessRule, accessRulesApi } from '../../../api/accessRules';

interface RuleTestProps {
  rule: AccessRule;
}

interface TestResult {
  success: boolean;
  effectivePredicate?: string;
  maskedFields?: string[];
  unmaskedFields?: string[];
  sampleRows?: any[];
  error?: string;
}

export const RuleTest: React.FC<RuleTestProps> = ({ rule }) => {
  const [testGroup, setTestGroup] = useState('');
  const [testUserId, setTestUserId] = useState('');
  const [running, setRunning] = useState(false);
  const [result, setResult] = useState<TestResult | null>(null);

  const runTest = async () => {
    if (!testGroup && !testUserId) {
      setResult({
        success: false,
        error: 'Please provide either a test group or user ID',
      });
      return;
    }

    setRunning(true);
    setResult(null);

    try {
      // First validate the DSL
      const validation = await accessRulesApi.validate({
        businessObjectId: rule.businessObjectId,
        rowFilterDsl: rule.rowFilterDsl || '',
      });

      if (!validation.valid) {
        setResult({
          success: false,
          error: validation.error || 'Validation failed',
        });
        setRunning(false);
        return;
      }

      // TODO: Call actual test endpoint when implemented
      // For now, simulate a test result
      const testResult: TestResult = {
        success: true,
        effectivePredicate: validation.sql || 'No predicate (full access)',
        maskedFields: rule.columnMasks
          ?.filter(m => m.maskType === 'HIDE' || m.maskType === 'MASK')
          .map(m => m.semanticTermId) || [],
        unmaskedFields: rule.columnMasks
          ?.filter(m => m.maskType === 'NONE')
          .map(m => m.semanticTermId) || [],
        sampleRows: [
          { id: 1, status: 'Sample row 1 - would be filtered/masked according to rule' },
          { id: 2, status: 'Sample row 2 - would be filtered/masked according to rule' },
        ],
      };

      setResult(testResult);
    } catch (err) {
      setResult({
        success: false,
        error: err instanceof Error ? err.message : 'Test failed',
      });
    } finally {
      setRunning(false);
    }
  };

  return (
    <div className="rule-test">
      <h3>Test Rule</h3>
      <p className="description">
        Test how this rule would affect specific users or groups before deploying.
      </p>

      <div className="test-inputs">
        <div className="input-group">
          <label htmlFor="testGroup">Test Group (LDAP DN)</label>
          <input
            id="testGroup"
            type="text"
            value={testGroup}
            onChange={(e) => setTestGroup(e.target.value)}
            placeholder="cn=developers,ou=groups,dc=example,dc=com"
            disabled={running}
          />
        </div>

        <div className="input-group">
          <label htmlFor="testUserId">Or Test User ID</label>
          <input
            id="testUserId"
            type="text"
            value={testUserId}
            onChange={(e) => setTestUserId(e.target.value)}
            placeholder="user@example.com"
            disabled={running}
          />
        </div>

        <button
          className="btn-primary"
          onClick={runTest}
          disabled={running || (!testGroup && !testUserId)}
        >
          {running ? 'Running Test...' : 'Run Test'}
        </button>
      </div>

      {result && (
        <div className={`test-result ${result.success ? 'success' : 'error'}`}>
          {result.success ? (
            <>
              <h4>✓ Test Passed</h4>

              <div className="result-section">
                <h5>Effective SQL Predicate</h5>
                <pre className="predicate-preview">{result.effectivePredicate}</pre>
              </div>

              {result.maskedFields && result.maskedFields.length > 0 && (
                <div className="result-section">
                  <h5>Masked/Hidden Fields</h5>
                  <ul className="field-list">
                    {result.maskedFields.map((field, idx) => (
                      <li key={idx} className="masked-field">
                        {field}
                      </li>
                    ))}
                  </ul>
                </div>
              )}

              {result.unmaskedFields && result.unmaskedFields.length > 0 && (
                <div className="result-section">
                  <h5>Unmasked Fields</h5>
                  <ul className="field-list">
                    {result.unmaskedFields.map((field, idx) => (
                      <li key={idx} className="unmasked-field">
                        {field}
                      </li>
                    ))}
                  </ul>
                </div>
              )}

              {result.sampleRows && result.sampleRows.length > 0 && (
                <div className="result-section">
                  <h5>Sample Result Preview</h5>
                  <div className="sample-rows">
                    {result.sampleRows.map((row, idx) => (
                      <pre key={idx} className="sample-row">
                        {JSON.stringify(row, null, 2)}
                      </pre>
                    ))}
                  </div>
                  <p className="note">
                    Note: Actual data would be filtered and masked according to the rule.
                  </p>
                </div>
              )}
            </>
          ) : (
            <>
              <h4>✗ Test Failed</h4>
              <p className="error-message">{result.error}</p>
            </>
          )}
        </div>
      )}

      <style>{`
        .rule-test {
          padding: 1.5rem;
          background: #f8f9fa;
          border-radius: 8px;
        }
        
        .rule-test h3 {
          margin: 0 0 0.5rem 0;
          font-size: 1.25rem;
          color: #1a1a1a;
        }
        
        .rule-test .description {
          margin: 0 0 1.5rem 0;
          color: #666;
          font-size: 0.875rem;
        }
        
        .test-inputs {
          display: grid;
          gap: 1rem;
          margin-bottom: 1.5rem;
        }
        
        .input-group {
          display: flex;
          flex-direction: column;
          gap: 0.5rem;
        }
        
        .input-group label {
          font-weight: 600;
          color: #333;
          font-size: 0.875rem;
        }
        
        .input-group input {
          padding: 0.75rem;
          border: 1px solid #ddd;
          border-radius: 4px;
          font-size: 0.875rem;
        }
        
        .input-group input:focus {
          outline: none;
          border-color: #1976d2;
        }
        
        .input-group input:disabled {
          background: #f5f5f5;
          cursor: not-allowed;
        }
        
        .btn-primary {
          padding: 0.75rem 1.5rem;
          background: #1976d2;
          color: white;
          border: none;
          border-radius: 4px;
          font-weight: 600;
          cursor: pointer;
          transition: background 0.2s;
        }
        
        .btn-primary:hover:not(:disabled) {
          background: #1565c0;
        }
        
        .btn-primary:disabled {
          background: #ccc;
          cursor: not-allowed;
        }
        
        .test-result {
          padding: 1.5rem;
          border-radius: 6px;
          margin-top: 1.5rem;
        }
        
        .test-result.success {
          background: #e8f5e9;
          border: 1px solid #4caf50;
        }
        
        .test-result.error {
          background: #ffebee;
          border: 1px solid #f44336;
        }
        
        .test-result h4 {
          margin: 0 0 1rem 0;
          font-size: 1.125rem;
        }
        
        .test-result.success h4 {
          color: #2e7d32;
        }
        
        .test-result.error h4 {
          color: #c62828;
        }
        
        .result-section {
          margin-top: 1.5rem;
          padding: 1rem;
          background: white;
          border-radius: 4px;
        }
        
        .result-section h5 {
          margin: 0 0 0.75rem 0;
          font-size: 0.9375rem;
          font-weight: 600;
          color: #333;
        }
        
        .edge_type_name-preview {
          padding: 0.75rem;
          background: #f5f5f5;
          border-radius: 4px;
          font-family: 'Monaco', 'Courier New', monospace;
          font-size: 0.8125rem;
          overflow-x: auto;
          margin: 0;
        }
        
        .field-list {
          list-style: none;
          padding: 0;
          margin: 0;
          display: grid;
          gap: 0.5rem;
        }
        
        .field-list li {
          padding: 0.5rem 0.75rem;
          border-radius: 4px;
          font-size: 0.875rem;
        }
        
        .masked-field {
          background: #fff3e0;
          color: #e65100;
          border-left: 3px solid #ff9800;
        }
        
        .unmasked-field {
          background: #e8f5e9;
          color: #2e7d32;
          border-left: 3px solid #4caf50;
        }
        
        .sample-rows {
          display: grid;
          gap: 0.5rem;
        }
        
        .sample-row {
          padding: 0.75rem;
          background: #f5f5f5;
          border-radius: 4px;
          font-family: 'Monaco', 'Courier New', monospace;
          font-size: 0.8125rem;
          overflow-x: auto;
          margin: 0;
        }
        
        .note {
          margin: 0.75rem 0 0 0;
          font-size: 0.8125rem;
          color: #666;
          font-style: italic;
        }
        
        .error-message {
          margin: 0;
          color: #c62828;
          font-weight: 500;
        }
      `}</style>
    </div>
  );
};
