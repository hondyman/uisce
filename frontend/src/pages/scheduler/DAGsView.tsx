/**
 * DAGs View - Manage Directed Acyclic Graphs of jobs
 */

import React, { useState, useMemo } from 'react';
import { 
  Plus, 
  Search, 
  Play, 
  Pause, 
  Trash2, 
  Edit2,
  GitBranch,
  Clock,
  CheckCircle,
  XCircle,
  Eye
} from 'lucide-react';
import { useTenantContext } from '../../hooks/useTenantContext';
import { useDAGs, DAG, triggerDAG, deleteDAG, updateDAG } from '../../api/schedulerApi';
import SemanticBindingsList from '../../components/scheduler/SemanticBindingsList';

const DAGsView: React.FC = () => {
  const { selectedTenant } = useTenantContext();
  const tenantId = selectedTenant?.id;
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedDAG, setSelectedDAG] = useState<DAG | null>(null);
  const [activeOnly, setActiveOnly] = useState(false);
  
  const { dags, loading, error, refetch } = useDAGs(tenantId || '', activeOnly);

  const filteredDAGs = useMemo(() => {
    if (!dags) return [];
    return dags.filter((dag: DAG) => {
      return dag.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        dag.description?.toLowerCase().includes(searchQuery.toLowerCase());
    });
  }, [dags, searchQuery]);

  const handleTrigger = async (dag: DAG, mode?: string) => {
    try {
      await triggerDAG(dag.id, mode);
      refetch();
    } catch (err) {
      console.error(`Failed to trigger DAG (${mode || 'normal'}):`, err);
    }
  };

  const handleToggleActive = async (dag: DAG) => {
    try {
      await updateDAG(dag.id, { is_active: !dag.is_active });
      refetch();
    } catch (err) {
      console.error('Failed to update DAG:', err);
    }
  };

  const handleDelete = async (dag: DAG) => {
    if (!confirm(`Delete DAG "${dag.name}"?`)) return;
    try {
      await deleteDAG(dag.id);
      refetch();
    } catch (err) {
      console.error('Failed to delete DAG:', err);
    }
  };

  if (loading) {
    return <div className="loading-state">Loading DAGs...</div>;
  }

  if (error) {
    return <div className="error-state">Error loading DAGs: {error.message}</div>;
  }

  return (
    <div className="dags-view">
      {/* Toolbar */}
      <div className="toolbar">
        <div className="toolbar-left">
          <div className="search-box">
            <Search className="search-icon" />
            <input
              type="text"
              placeholder="Search DAGs..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
          </div>
          
          <label className="checkbox-filter">
            <input 
              type="checkbox" 
              checked={activeOnly}
              onChange={(e) => setActiveOnly(e.target.checked)}
            />
            Active Only
          </label>
        </div>
        
        <div className="toolbar-right">
          <button className="btn-primary">
            <Plus /> New DAG
          </button>
        </div>
      </div>

      {/* DAGs Grid */}
      <div className="dags-grid">
        {filteredDAGs.length === 0 ? (
          <div className="empty-state">
            <GitBranch className="empty-icon" />
            <h3>No DAGs Found</h3>
            <p>Create a new DAG to orchestrate multiple jobs</p>
          </div>
        ) : (
          filteredDAGs.map((dag: DAG) => (
            <div 
              key={dag.id} 
              className={`dag-card ${!dag.is_active ? 'inactive' : ''}`}
            >
              <div className="dag-card-header">
                <div className="dag-info">
                  <h3 className="dag-name">{dag.name}</h3>
                  {dag.category && (
                    <span className="dag-category">{dag.category}</span>
                  )}
                </div>
                <span className={`status-indicator ${dag.is_active ? 'active' : 'inactive'}`} />
              </div>
              
              {dag.description && (
                <p className="dag-description">{dag.description}</p>
              )}
              
              <div className="dag-stats">
                <div className="stat">
                  <span className="stat-value">{dag.nodes?.length || 0}</span>
                  <span className="stat-label">Nodes</span>
                </div>
                <div className="stat">
                  <span className="stat-value">{dag.edges?.length || 0}</span>
                  <span className="stat-label">Edges</span>
                </div>
                <div className="stat">
                  <span className="stat-value">{dag.max_parallel_jobs}</span>
                  <span className="stat-label">Max Parallel</span>
                </div>
              </div>
              
              {dag.next_run_at && (
                <div className="dag-next-run">
                  <Clock className="icon" />
                  <span>Next: {formatDateTime(dag.next_run_at)}</span>
                </div>
              )}
              
              <div className="dag-card-actions">
                <button 
                  className="action-btn"
                  title="View Graph"
                  onClick={() => setSelectedDAG(dag)}
                >
                  <Eye />
                </button>
                <button 
                  className="action-btn"
                  title="Run Now"
                  onClick={() => handleTrigger(dag)}
                >
                  <Play />
                </button>
                <button 
                  className="action-btn secondary"
                  title="Dry Run"
                  onClick={() => handleTrigger(dag, 'DRY_RUN')}
                  style={{ fontSize: '0.8em' }}
                >
                  Dry
                </button>
                <button 
                  className="action-btn"
                  title={dag.is_active ? 'Pause' : 'Resume'}
                  onClick={() => handleToggleActive(dag)}
                >
                  <Pause />
                </button>
                <button 
                  className="action-btn"
                  title="Edit"
                >
                  <Edit2 />
                </button>
                <button 
                  className="action-btn danger"
                  title="Delete"
                  onClick={() => handleDelete(dag)}
                >
                  <Trash2 />
                </button>
              </div>
            </div>
          ))
        )}
      </div>

      {/* DAG Graph Modal */}
      {selectedDAG && (
        <DAGGraphModal 
          dag={selectedDAG} 
          onClose={() => setSelectedDAG(null)} 
        />
      )}
    </div>
  );
};

interface DAGGraphModalProps {
  dag: DAG;
  onClose: () => void;
}

const DAGGraphModal: React.FC<DAGGraphModalProps> = ({ dag, onClose }) => {
  // Simple text-based DAG visualization
  const nodeMap = useMemo(() => {
    const map = new Map<string, string>();
    dag.nodes?.forEach(node => {
      map.set(node.id, node.job_id);
    });
    return map;
  }, [dag.nodes]);

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="dag-graph-modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>{dag.name}</h2>
          <button className="close-btn" onClick={onClose}>&times;</button>
        </div>
        
        <div className="modal-content">
          <div className="dag-info-section">
            <h3>Configuration</h3>
            <dl>
              <dt>Max Parallel Jobs</dt>
              <dd>{dag.max_parallel_jobs}</dd>
              <dt>Fail Fast</dt>
              <dd>{dag.fail_fast ? 'Yes' : 'No'}</dd>
              <dt>Timeout</dt>
              <dd>{dag.timeout_seconds}s</dd>
            </dl>
          </div>

          <div className="dag-info-section">
            {dag.slo_critical && (
              <div className="slo-alert" style={{marginBottom: '10px'}}>
                  <span className="slo-badge critical">SLO Critical</span>
                  <a href={`/intelligence/slo?dag=${dag.id}`} className="slo-link" style={{marginLeft: '10px'}}>
                    View Forecast
                  </a>
              </div>
            )}
            <SemanticBindingsList bindings={dag.semantic_bindings} />
          </div>
          
          <div className="dag-structure-section">
            <h3>Structure</h3>
            <div className="nodes-list">
              <h4>Nodes ({dag.nodes?.length || 0})</h4>
              {dag.nodes?.map((node) => (
                <div key={node.id} className="node-item">
                  <span className="node-id">{node.id}</span>
                  <span className="node-job">Job: {node.job_id}</span>
                </div>
              ))}
            </div>
            
            <div className="edges-list">
              <h4>Edges ({dag.edges?.length || 0})</h4>
              {dag.edges?.map((edge, idx) => (
                <div key={idx} className="edge-item">
                  <span>{edge.from_node_id}</span>
                  <span className="arrow">→</span>
                  <span>{edge.to_node_id}</span>
                  {edge.type && <span className="edge-type">({edge.type})</span>}
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

function formatDateTime(isoString: string): string {
  const date = new Date(isoString);
  return date.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  });
}

export default DAGsView;
