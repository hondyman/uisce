import React from 'react';
import { SemanticBinding } from '../../api/schedulerApi';
import { Database, Link, Layout, GitBranch, Box, ArrowRight, Snowflake } from 'lucide-react';

interface SemanticBindingsListProps {
  bindings?: SemanticBinding;
  showTitle?: boolean;
  coldBOs?: string[]; // IDs of BOs that are in cold storage
}

export const SemanticBindingsList: React.FC<SemanticBindingsListProps> = ({ bindings, showTitle = true, coldBOs = [] }) => {
  if (!bindings) return null;

  const hasBindings = (
    (bindings.bo_ids && bindings.bo_ids.length > 0) ||
    (bindings.api_ids && bindings.api_ids.length > 0) ||
    (bindings.page_ids && bindings.page_ids.length > 0) ||
    (bindings.workflow_ids && bindings.workflow_ids.length > 0) ||
    (bindings.preagg_ids && bindings.preagg_ids.length > 0)
  );

  if (!hasBindings) return null;

  return (
    <div className="semantic-bindings-section">
      {showTitle && <h4>Semantic Context</h4>}
      <div className="semantic-bindings-list">
        {bindings.bo_ids?.map(id => {
          const isCold = coldBOs.includes(id);
          return (
            <div key={id} className={`semantic-item bo ${isCold ? 'cold-storage' : ''}`} title={isCold ? 'In Cold Storage' : undefined}>
              <Database size={14} className="icon" />
              <span className="semantic-label">BO</span>
              <span className="semantic-value">
                {id}
                {isCold && <Snowflake size={12} className="cold-icon" style={{ marginLeft: 4, fill: '#add8e6', stroke: '#add8e6' }} />}
              </span>
              <a href={`/catalog/bo/${id}`} className="deep-link" title="View Business Object">
                <ArrowRight size={12} />
              </a>
            </div>
          );
        })}
        {bindings.api_ids?.map(id => (
          <div key={id} className="semantic-item api">
            <Link size={14} className="icon" />
            <span className="semantic-label">API</span>
            <span className="semantic-value">{id}</span>
             <a href={`/studio/api/${id}`} className="deep-link" title="View API">
              <ArrowRight size={12} />
            </a>
          </div>
        ))}
        {bindings.page_ids?.map(id => (
          <div key={id} className="semantic-item page">
            <Layout size={14} className="icon" />
            <span className="semantic-label">PAGE</span>
            <span className="semantic-value">{id}</span>
             <a href={`/studio/page/${id}`} className="deep-link" title="View Page">
              <ArrowRight size={12} />
            </a>
          </div>
        ))}
        {bindings.workflow_ids?.map(id => (
          <div key={id} className="semantic-item workflow">
            <GitBranch size={14} className="icon" />
            <span className="semantic-label">WF</span>
            <span className="semantic-value">{id}</span>
             <a href={`/studio/workflow/${id}`} className="deep-link" title="View Workflow">
              <ArrowRight size={12} />
            </a>
          </div>
        ))}
        {bindings.preagg_ids?.map(id => (
          <div key={id} className="semantic-item preagg">
            <Box size={14} className="icon" />
            <span className="semantic-label">AGG</span>
            <span className="semantic-value">{id}</span>
             <a href={`/build/semantic-models/${id}`} className="deep-link" title="View Model">
              <ArrowRight size={12} />
            </a>
          </div>
        ))}
      </div>
    </div>
  );
};

export default SemanticBindingsList;
