import React, { useEffect, useState } from 'react';
import { AccessRule, accessRulesApi, AccessRuleImpact } from '../../../api/accessRules';

interface RulePreviewProps {
  rule: AccessRule;
}

export const RulePreview: React.FC<RulePreviewProps> = ({ rule }) => {
  const [impact, setImpact] = useState<AccessRuleImpact | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (rule.ruleId) {
      loadImpact();
    }
  }, [rule.ruleId]);

  const loadImpact = async () => {
    if (!rule.ruleId) return;
    
    setLoading(true);
    setError(null);
    
    try {
      const result = await accessRulesApi.impact(rule.ruleId);
      setImpact(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load impact');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="rule-preview">
      <h3>Rule Preview</h3>
      
      <div className="preview-section">
        <h4>Effective Configuration</h4>
        <div className="config-item">
          <label>Business Object:</label>
          <span>{rule.businessObjectId}</span>
        </div>
        <div className="config-item">
          <label>Group:</label>
          <span>{rule.groupDn}</span>
        </div>
        <div className="config-item">
          <label>Access Level:</label>
          <span className={`access-level ${rule.accessLevel.toLowerCase()}`}>
            {rule.accessLevel}
          </span>
        </div>
        <div className="config-item">
          <label>Status:</label>
          <span className={`status ${rule.status.toLowerCase()}`}>
            {rule.status}
          </span>
        </div>
      </div>

      <div className="preview-section">
        <h4>Row Predicate</h4>
        {rule.rowFilterDsl ? (
          <pre className="dsl-preview">{rule.rowFilterDsl}</pre>
        ) : (
          <p className="empty-state">No row filter defined (full access)</p>
        )}
      </div>

      <div className="preview-section">
        <h4>Column Masks</h4>
        {rule.columnMasks && rule.columnMasks.length > 0 ? (
          <table className="masks-table">
            <thead>
              <tr>
                <th>Semantic Term</th>
                <th>Mask Type</th>
              </tr>
            </thead>
            <tbody>
              {rule.columnMasks.map((mask, idx) => (
                <tr key={idx}>
                  <td>{mask.semanticTermId}</td>
                  <td>
                    <span className={`mask-type ${mask.maskType.toLowerCase()}`}>
                      {mask.maskType}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <p className="empty-state">No column masks defined</p>
        )}
      </div>

      <div className="preview-section">
        <h4>Scope</h4>
        <div className="scope-indicators">
          <span className={`scope-badge ${rule.scope?.appliesToApis ? 'active' : 'inactive'}`}>
            APIs
          </span>
          <span className={`scope-badge ${rule.scope?.appliesToBi ? 'active' : 'inactive'}`}>
            BI
          </span>
          <span className={`scope-badge ${rule.scope?.appliesToAi ? 'active' : 'inactive'}`}>
            AI
          </span>
        </div>
      </div>

      <div className="preview-section">
        <h4>Impact Analysis</h4>
        {loading && <p className="loading-state">Loading impact...</p>}
        {error && <p className="error-state">{error}</p>}
        {impact && (
          <div className="impact-report">
            <div className="impact-item">
              <label>Semantic Terms Affected:</label>
              <span className="count">{impact.semanticTerms.length}</span>
              {impact.semanticTerms.length > 0 && (
                <ul className="impact-list">
                  {impact.semanticTerms.slice(0, 5).map((term, idx) => (
                    <li key={idx}>{term}</li>
                  ))}
                  {impact.semanticTerms.length > 5 && (
                    <li>... and {impact.semanticTerms.length - 5} more</li>
                  )}
                </ul>
              )}
            </div>

            <div className="impact-item">
              <label>APIs Affected:</label>
              <span className="count">{impact.apis.length}</span>
              {impact.apis.length > 0 && (
                <ul className="impact-list">
                  {impact.apis.slice(0, 5).map((api, idx) => (
                    <li key={idx}>{api}</li>
                  ))}
                  {impact.apis.length > 5 && (
                    <li>... and {impact.apis.length - 5} more</li>
                  )}
                </ul>
              )}
            </div>

            {rule.scope?.appliesToBi && (
              <div className="impact-item">
                <label>BI Artifacts Affected:</label>
                <span className="count">{impact.biArtifacts.length}</span>
                {impact.biArtifacts.length > 0 && (
                  <ul className="impact-list">
                    {impact.biArtifacts.slice(0, 5).map((artifact, idx) => (
                      <li key={idx}>{artifact}</li>
                    ))}
                    {impact.biArtifacts.length > 5 && (
                      <li>... and {impact.biArtifacts.length - 5} more</li>
                    )}
                  </ul>
                )}
              </div>
            )}

            {rule.scope?.appliesToAi && (
              <div className="impact-item">
                <label>AI Artifacts Affected:</label>
                <span className="count">{impact.aiArtifacts.length}</span>
                {impact.aiArtifacts.length > 0 && (
                  <ul className="impact-list">
                    {impact.aiArtifacts.slice(0, 5).map((artifact, idx) => (
                      <li key={idx}>{artifact}</li>
                    ))}
                    {impact.aiArtifacts.length > 5 && (
                      <li>... and {impact.aiArtifacts.length - 5} more</li>
                    )}
                  </ul>
                )}
              </div>
            )}
          </div>
        )}
      </div>

      <style>{`
        .rule-preview {
          padding: 1.5rem;
          background: #f8f9fa;
          border-radius: 8px;
        }
        
        .rule-preview h3 {
          margin: 0 0 1.5rem 0;
          font-size: 1.25rem;
          color: #1a1a1a;
        }
        
        .rule-preview h4 {
          margin: 0 0 1rem 0;
          font-size: 1rem;
          font-weight: 600;
          color: #333;
        }
        
        .preview-section {
          margin-bottom: 1.5rem;
          padding: 1rem;
          background: white;
          border-radius: 6px;
          border: 1px solid #e0e0e0;
        }
        
        .config-item {
          display: flex;
          margin-bottom: 0.75rem;
        }
        
        .config-item label {
          min-width: 150px;
          font-weight: 500;
          color: #666;
        }
        
        .config-item span {
          color: #1a1a1a;
        }
        
        .access-level, .status {
          padding: 0.25rem 0.75rem;
          border-radius: 4px;
          font-size: 0.875rem;
          font-weight: 600;
        }
        
        .access-level.read {
          background: #e3f2fd;
          color: #1976d2;
        }
        
        .access-level.write {
          background: #e8f5e9;
          color: #388e3c;
        }
        
        .access-level.none {
          background: #ffebee;
          color: #d32f2f;
        }
        
        .status.draft {
          background: #fff3e0;
          color: #f57c00;
        }
        
        .status.approved {
          background: #e8f5e9;
          color: #388e3c;
        }
        
        .status.review {
          background: #e3f2fd;
          color: #1976d2;
        }
        
        .dsl-preview {
          padding: 1rem;
          background: #f5f5f5;
          border-radius: 4px;
          font-family: 'Monaco', 'Courier New', monospace;
          font-size: 0.875rem;
          overflow-x: auto;
        }
        
        .masks-table {
          width: 100%;
          border-collapse: collapse;
        }
        
        .masks-table th,
        .masks-table td {
          padding: 0.75rem;
          text-align: left;
          border-bottom: 1px solid #e0e0e0;
        }
        
        .masks-table th {
          font-weight: 600;
          color: #666;
          font-size: 0.875rem;
        }
        
        .mask-type {
          padding: 0.25rem 0.5rem;
          border-radius: 4px;
          font-size: 0.8125rem;
          font-weight: 600;
        }
        
        .mask-type.hide {
          background: #ffebee;
          color: #d32f2f;
        }
        
        .mask-type.mask {
          background: #fff3e0;
          color: #f57c00;
        }
        
        .mask-type.none {
          background: #f5f5f5;
          color: #666;
        }
        
        .scope-indicators {
          display: flex;
          gap: 0.75rem;
        }
        
        .scope-badge {
          padding: 0.5rem 1rem;
          border-radius: 4px;
          font-size: 0.875rem;
          font-weight: 600;
        }
        
        .scope-badge.active {
          background: #e3f2fd;
          color: #1976d2;
        }
        
        .scope-badge.inactive {
          background: #f5f5f5;
          color: #999;
        }
        
        .empty-state {
          color: #999;
          font-style: italic;
          margin: 0;
        }
        
        .loading-state {
          color: #666;
          font-style: italic;
        }
        
        .error-state {
          color: #d32f2f;
          font-weight: 500;
        }
        
        .impact-report {
          display: grid;
          gap: 1rem;
        }
        
        .impact-item {
          padding: 0.75rem;
          background: #f9f9f9;
          border-radius: 4px;
        }
        
        .impact-item label {
          display: block;
          font-weight: 600;
          color: #333;
          margin-bottom: 0.5rem;
        }
        
        .impact-item .count {
          display: inline-block;
          padding: 0.25rem 0.75rem;
          background: #1976d2;
          color: white;
          border-radius: 12px;
          font-size: 0.875rem;
          font-weight: 600;
        }
        
        .impact-list {
          list-style: none;
          padding: 0;
          margin: 0.75rem 0 0 0;
        }
        
        .impact-list li {
          padding: 0.5rem;
          background: white;
          border-radius: 4px;
          margin-bottom: 0.25rem;
          font-size: 0.875rem;
          color: #666;
        }
      `}</style>
    </div>
  );
};
