/**
 * World-Class Enterprise Scheduler - Dependency Visualizer
 * DAG visualization of job dependencies with interactive graph
 */

import React, { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { Link, useSearchParams } from 'react-router-dom';
import * as schedulerService from '../services/schedulerService';
import {
  Job,
  JobExecution,
  Dependency,
  DependencyType,
  JobStatus,
} from '../../../types/scheduler';
import '../styles/SchedulerDashboard.css';

// ============================================================================
// Types
// ============================================================================

interface GraphNode {
  id: string;
  job: Job;
  x: number;
  y: number;
  level: number;
  execution?: JobExecution;
  selected: boolean;
}

interface GraphEdge {
  id: string;
  source: string;
  target: string;
  dependency: Dependency;
}

interface GraphLayout {
  nodes: GraphNode[];
  edges: GraphEdge[];
  width: number;
  height: number;
}

// ============================================================================
// Main Component
// ============================================================================

export function DependencyVisualizerPage() {
  const { t } = useTranslation();
  const [searchParams] = useSearchParams();
  const jobId = searchParams.get('job');
  
  const [jobs, setJobs] = useState<Job[]>([]);
  const [executions, setExecutions] = useState<Map<string, JobExecution>>(new Map());
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedNode, setSelectedNode] = useState<string | null>(jobId);
  const [zoomLevel, setZoomLevel] = useState(1);
  const [viewMode, setViewMode] = useState<'dag' | 'tree' | 'list'>('dag');
  const [showLegend, setShowLegend] = useState(true);
  
  const svgRef = useRef<SVGSVGElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  
  // Load jobs with dependencies
  const loadJobs = useCallback(async () => {
    try {
      setLoading(true);
      const jobsResponse = await schedulerService.listJobs({ page: 1, limit: 100 });
      setJobs(jobsResponse.jobs);
      
      // Load latest executions for all jobs
      const executionMap = new Map<string, JobExecution>();
      const executionPromises = jobsResponse.jobs.map(async (job) => {
        try {
          const execResponse = await schedulerService.listExecutions(job.id, { page: 1, limit: 1 });
          if (execResponse.executions.length > 0) {
            executionMap.set(job.id, execResponse.executions[0]);
          }
        } catch {
          // Ignore - job might not have any executions
        }
      });
      
      await Promise.all(executionPromises);
      setExecutions(executionMap);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load jobs');
    } finally {
      setLoading(false);
    }
  }, []);
  
  useEffect(() => {
    loadJobs();
  }, [loadJobs]);
  
  // Build graph layout
  const layout = useMemo(() => {
    if (jobs.length === 0) return null;
    return buildGraphLayout(jobs, executions, selectedNode);
  }, [jobs, executions, selectedNode]);
  
  // Handle zoom
  const handleZoom = (delta: number) => {
    setZoomLevel(prev => Math.max(0.25, Math.min(2, prev + delta)));
  };
  
  // Handle node click
  const handleNodeClick = (nodeId: string) => {
    setSelectedNode(prev => prev === nodeId ? null : nodeId);
  };
  
  // Get selected job details
  const selectedJob = selectedNode ? jobs.find(j => j.id === selectedNode) : null;
  const selectedExecution = selectedNode ? executions.get(selectedNode) : null;
  
  if (loading) {
    return (
      <div className="scheduler-dashboard">
        <div className="loading-spinner">
          <div className="spinner" />
        </div>
      </div>
    );
  }
  
  if (error) {
    return (
      <div className="scheduler-dashboard">
        <div className="empty-state">
          <div className="empty-state-icon">⚠️</div>
          <div className="empty-state-text">{error}</div>
          <button className="btn btn-primary" onClick={loadJobs}>
            {t('scheduler.retry', 'Retry')}
          </button>
        </div>
      </div>
    );
  }
  
  return (
    <div className="scheduler-dashboard">
      {/* Header */}
      <div className="scheduler-header">
        <div>
          <h1>🔗 {t('scheduler.dependencyVisualizer', 'Dependency Visualizer')}</h1>
          <p className="header-subtitle">
            {t('scheduler.dependencyVisualizerDesc', 'Interactive visualization of job dependencies and execution flow')}
          </p>
        </div>
        <div className="scheduler-header-actions">
          <div className="view-toggle">
            <button
              className={`btn btn-sm ${viewMode === 'dag' ? 'btn-primary' : 'btn-secondary'}`}
              onClick={() => setViewMode('dag')}
              title={t('scheduler.dagView', 'DAG View')}
            >
              🕸️
            </button>
            <button
              className={`btn btn-sm ${viewMode === 'tree' ? 'btn-primary' : 'btn-secondary'}`}
              onClick={() => setViewMode('tree')}
              title={t('scheduler.treeView', 'Tree View')}
            >
              🌳
            </button>
            <button
              className={`btn btn-sm ${viewMode === 'list' ? 'btn-primary' : 'btn-secondary'}`}
              onClick={() => setViewMode('list')}
              title={t('scheduler.listView', 'List View')}
            >
              📋
            </button>
          </div>
          <button
            className="btn btn-sm btn-secondary"
            onClick={() => setShowLegend(!showLegend)}
          >
            {showLegend ? '🙈' : '👁️'} {t('scheduler.legend', 'Legend')}
          </button>
        </div>
      </div>
      
      {/* Main Content */}
      <div className="visualizer-container">
        {/* Graph Canvas */}
        <div className="graph-canvas" ref={containerRef}>
          {/* Zoom Controls */}
          <div className="zoom-controls">
            <button className="btn btn-sm btn-secondary" onClick={() => handleZoom(0.25)} title="Zoom In">
              ➕
            </button>
            <span className="zoom-level">{Math.round(zoomLevel * 100)}%</span>
            <button className="btn btn-sm btn-secondary" onClick={() => handleZoom(-0.25)} title="Zoom Out">
              ➖
            </button>
            <button className="btn btn-sm btn-secondary" onClick={() => setZoomLevel(1)} title="Reset Zoom">
              🔄
            </button>
          </div>
          
          {viewMode === 'dag' && layout && (
            <DAGView
              layout={layout}
              zoomLevel={zoomLevel}
              selectedNode={selectedNode}
              onNodeClick={handleNodeClick}
              svgRef={svgRef}
            />
          )}
          
          {viewMode === 'tree' && layout && (
            <TreeView
              jobs={jobs}
              executions={executions}
              selectedNode={selectedNode}
              onNodeClick={handleNodeClick}
            />
          )}
          
          {viewMode === 'list' && (
            <ListView
              jobs={jobs}
              executions={executions}
              selectedNode={selectedNode}
              onNodeClick={handleNodeClick}
              t={t}
            />
          )}
          
          {jobs.length === 0 && (
            <div className="empty-state">
              <div className="empty-state-icon">🔗</div>
              <div className="empty-state-text">
                {t('scheduler.noDependencies', 'No jobs with dependencies found')}
              </div>
              <Link to="/scheduler/jobs/new" className="btn btn-primary">
                {t('scheduler.createJob', 'Create Job')}
              </Link>
            </div>
          )}
        </div>
        
        {/* Legend */}
        {showLegend && (
          <div className="graph-legend">
            <h4>{t('scheduler.legend', 'Legend')}</h4>
            <div className="legend-section">
              <h5>{t('scheduler.nodeStatus', 'Node Status')}</h5>
              <div className="legend-item">
                <div className="legend-color status-completed" />
                <span>{t('scheduler.status.completed', 'Completed')}</span>
              </div>
              <div className="legend-item">
                <div className="legend-color status-running" />
                <span>{t('scheduler.status.running', 'Running')}</span>
              </div>
              <div className="legend-item">
                <div className="legend-color status-failed" />
                <span>{t('scheduler.status.failed', 'Failed')}</span>
              </div>
              <div className="legend-item">
                <div className="legend-color status-pending" />
                <span>{t('scheduler.status.pending', 'Pending')}</span>
              </div>
            </div>
            <div className="legend-section">
              <h5>{t('scheduler.dependencyTypes', 'Dependency Types')}</h5>
              <div className="legend-item">
                <div className="legend-line finish-to-start" />
                <span>{t('scheduler.dependency.finishToStart', 'Finish to Start')}</span>
              </div>
              <div className="legend-item">
                <div className="legend-line start-to-start" />
                <span>{t('scheduler.dependency.startToStart', 'Start to Start')}</span>
              </div>
              <div className="legend-item">
                <div className="legend-line finish-to-finish" />
                <span>{t('scheduler.dependency.finishToFinish', 'Finish to Finish')}</span>
              </div>
              <div className="legend-item">
                <div className="legend-line start-to-finish" />
                <span>{t('scheduler.dependency.startToFinish', 'Start to Finish')}</span>
              </div>
            </div>
          </div>
        )}
        
        {/* Selected Job Details */}
        {selectedJob && (
          <div className="node-details-panel">
            <div className="panel-header">
              <h4>
                <StatusIcon status={selectedExecution?.status} />
                {selectedJob.name}
              </h4>
              <button
                className="btn btn-sm btn-ghost"
                onClick={() => setSelectedNode(null)}
              >
                ✕
              </button>
            </div>
            <div className="panel-content">
              <p className="job-description">{selectedJob.description}</p>
              
              <div className="detail-row">
                <span className="label">{t('scheduler.fields.type', 'Type')}:</span>
                <span className="value">{selectedJob.type}</span>
              </div>
              
              <div className="detail-row">
                <span className="label">{t('scheduler.fields.priority', 'Priority')}:</span>
                <span className="value">{selectedJob.priority}</span>
              </div>
              
              {selectedExecution && (
                <>
                  <div className="detail-row">
                    <span className="label">{t('scheduler.fields.status', 'Status')}:</span>
                    <StatusBadge status={selectedExecution.status} />
                  </div>
                  
                  {selectedExecution.duration_ms && (
                    <div className="detail-row">
                      <span className="label">{t('scheduler.fields.duration', 'Duration')}:</span>
                      <span className="value">{formatDuration(selectedExecution.duration_ms)}</span>
                    </div>
                  )}
                </>
              )}
              
              {/* Dependencies */}
              {selectedJob.dependencies && selectedJob.dependencies.length > 0 && (
                <div className="dependencies-section">
                  <h5>{t('scheduler.dependencies', 'Dependencies')}</h5>
                  <ul className="dependency-list">
                    {selectedJob.dependencies.map((dep, index) => {
                      const depJob = jobs.find(j => j.id === dep.job_id);
                      return (
                        <li key={index}>
                          <button
                            className="dependency-link"
                            onClick={() => setSelectedNode(dep.job_id)}
                          >
                            {depJob?.name || dep.job_id}
                          </button>
                          <span className="dependency-type">
                            {formatDependencyType(dep.type)}
                          </span>
                        </li>
                      );
                    })}
                  </ul>
                </div>
              )}
              
              {/* Dependents */}
              {getDependents(selectedJob.id, jobs).length > 0 && (
                <div className="dependencies-section">
                  <h5>{t('scheduler.dependents', 'Dependents')}</h5>
                  <ul className="dependency-list">
                    {getDependents(selectedJob.id, jobs).map((dep, index) => (
                      <li key={index}>
                        <button
                          className="dependency-link"
                          onClick={() => setSelectedNode(dep.id)}
                        >
                          {dep.name}
                        </button>
                      </li>
                    ))}
                  </ul>
                </div>
              )}
              
              <div className="panel-actions">
                <Link to={`/scheduler/jobs/${selectedJob.id}`} className="btn btn-primary btn-sm">
                  {t('scheduler.viewJob', 'View Job')}
                </Link>
                <Link to={`/scheduler/jobs/${selectedJob.id}/edit`} className="btn btn-secondary btn-sm">
                  {t('scheduler.editJob', 'Edit Job')}
                </Link>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

// ============================================================================
// DAG View Component
// ============================================================================

interface DAGViewProps {
  layout: GraphLayout;
  zoomLevel: number;
  selectedNode: string | null;
  onNodeClick: (nodeId: string) => void;
  svgRef: React.RefObject<SVGSVGElement | null>;
}

function DAGView({ layout, zoomLevel, selectedNode, onNodeClick, svgRef }: DAGViewProps) {
  const nodeWidth = 180;
  const nodeHeight = 60;
  
  return (
    <svg
      ref={svgRef}
      className="dag-svg"
      width={layout.width * zoomLevel}
      height={layout.height * zoomLevel}
      viewBox={`0 0 ${layout.width} ${layout.height}`}
    >
      {/* Arrow marker definitions */}
      <defs>
        <marker
          id="arrowhead"
          markerWidth="10"
          markerHeight="7"
          refX="9"
          refY="3.5"
          orient="auto"
        >
          <polygon points="0 0, 10 3.5, 0 7" fill="#666" />
        </marker>
        <marker
          id="arrowhead-selected"
          markerWidth="10"
          markerHeight="7"
          refX="9"
          refY="3.5"
          orient="auto"
        >
          <polygon points="0 0, 10 3.5, 0 7" fill="#2563eb" />
        </marker>
      </defs>
      
      {/* Edges */}
      <g className="edges">
        {layout.edges.map(edge => {
          const sourceNode = layout.nodes.find(n => n.id === edge.source);
          const targetNode = layout.nodes.find(n => n.id === edge.target);
          
          if (!sourceNode || !targetNode) return null;
          
          const isSelected = selectedNode === edge.source || selectedNode === edge.target;
          const edgeClass = `edge ${edge.dependency.type} ${isSelected ? 'selected' : ''}`;
          
          // Calculate path points
          const startX = sourceNode.x + nodeWidth;
          const startY = sourceNode.y + nodeHeight / 2;
          const endX = targetNode.x;
          const endY = targetNode.y + nodeHeight / 2;
          
          // Bezier curve control points
          const controlX1 = startX + (endX - startX) * 0.5;
          const controlY1 = startY;
          const controlX2 = startX + (endX - startX) * 0.5;
          const controlY2 = endY;
          
          return (
            <path
              key={edge.id}
              className={edgeClass}
              d={`M ${startX} ${startY} C ${controlX1} ${controlY1}, ${controlX2} ${controlY2}, ${endX} ${endY}`}
              fill="none"
              stroke={isSelected ? '#2563eb' : '#999'}
              strokeWidth={isSelected ? 2 : 1}
              strokeDasharray={getDashArray(edge.dependency.type)}
              markerEnd={isSelected ? 'url(#arrowhead-selected)' : 'url(#arrowhead)'}
            />
          );
        })}
      </g>
      
      {/* Nodes */}
      <g className="nodes">
        {layout.nodes.map(node => {
          const isSelected = selectedNode === node.id;
          const status = node.execution?.status;
          
          return (
            <g
              key={node.id}
              className={`node ${isSelected ? 'selected' : ''}`}
              transform={`translate(${node.x}, ${node.y})`}
              onClick={() => onNodeClick(node.id)}
              style={{ cursor: 'pointer' }}
            >
              {/* Node background */}
              <rect
                width={nodeWidth}
                height={nodeHeight}
                rx={8}
                ry={8}
                fill={getNodeFill(status)}
                stroke={isSelected ? '#2563eb' : getNodeStroke(status)}
                strokeWidth={isSelected ? 3 : 1}
              />
              
              {/* Status indicator */}
              <circle
                cx={15}
                cy={nodeHeight / 2}
                r={6}
                fill={getStatusColor(status)}
              />
              
              {/* Job name */}
              <text
                x={30}
                y={nodeHeight / 2 - 5}
                className="node-name"
                fontSize={13}
                fontWeight={600}
                fill="#1a1a1a"
              >
                {truncate(node.job.name, 18)}
              </text>
              
              {/* Job type */}
              <text
                x={30}
                y={nodeHeight / 2 + 12}
                className="node-type"
                fontSize={11}
                fill="#666"
              >
                {node.job.type}
              </text>
              
              {/* Running indicator */}
              {status === JobStatus.RUNNING && (
                <circle
                  cx={nodeWidth - 15}
                  cy={nodeHeight / 2}
                  r={5}
                  fill="#3b82f6"
                  className="pulse-animation"
                />
              )}
            </g>
          );
        })}
      </g>
    </svg>
  );
}

// ============================================================================
// Tree View Component
// ============================================================================

interface TreeViewProps {
  jobs: Job[];
  executions: Map<string, JobExecution>;
  selectedNode: string | null;
  onNodeClick: (nodeId: string) => void;
}

function TreeView({ jobs, executions, selectedNode, onNodeClick }: TreeViewProps) {
  // Build tree structure
  const rootJobs = jobs.filter(job => !job.dependencies || job.dependencies.length === 0);
  
  return (
    <div className="tree-view">
      {rootJobs.map(job => (
        <TreeNode
          key={job.id}
          job={job}
          jobs={jobs}
          executions={executions}
          selectedNode={selectedNode}
          onNodeClick={onNodeClick}
          level={0}
        />
      ))}
    </div>
  );
}

interface TreeNodeProps {
  job: Job;
  jobs: Job[];
  executions: Map<string, JobExecution>;
  selectedNode: string | null;
  onNodeClick: (nodeId: string) => void;
  level: number;
}

function TreeNode({ job, jobs, executions, selectedNode, onNodeClick, level }: TreeNodeProps) {
  const [expanded, setExpanded] = useState(true);
  const dependents = getDependents(job.id, jobs);
  const execution = executions.get(job.id);
  const isSelected = selectedNode === job.id;
  
  return (
    <div className="tree-node-container">
      <div
        className={`tree-node ${isSelected ? 'selected' : ''}`}
        onClick={() => onNodeClick(job.id)}
      >
        {dependents.length > 0 && (
          <button
            className="expand-btn"
            onClick={(e) => {
              e.stopPropagation();
              setExpanded(!expanded);
            }}
          >
            {expanded ? '▼' : '▶'}
          </button>
        )}
        {dependents.length === 0 && <span className="expand-placeholder" />}
        <StatusIcon status={execution?.status} />
        <span className="tree-node-name">{job.name}</span>
        <span className="tree-node-type">{job.type}</span>
      </div>
      
      {expanded && dependents.length > 0 && (
        <div className="tree-children">
          {dependents.map(dep => (
            <TreeNode
              key={dep.id}
              job={dep}
              jobs={jobs}
              executions={executions}
              selectedNode={selectedNode}
              onNodeClick={onNodeClick}
              level={level + 1}
            />
          ))}
        </div>
      )}
    </div>
  );
}

// ============================================================================
// List View Component
// ============================================================================

interface ListViewProps {
  jobs: Job[];
  executions: Map<string, JobExecution>;
  selectedNode: string | null;
  onNodeClick: (nodeId: string) => void;
  t: (key: string, defaultValue: string) => string;
}

function ListView({ jobs, executions, selectedNode, onNodeClick, t }: ListViewProps) {
  return (
    <div className="list-view">
      <table className="data-table">
        <thead>
          <tr>
            <th>{t('scheduler.fields.name', 'Name')}</th>
            <th>{t('scheduler.fields.type', 'Type')}</th>
            <th>{t('scheduler.fields.status', 'Status')}</th>
            <th>{t('scheduler.dependencies', 'Dependencies')}</th>
            <th>{t('scheduler.dependents', 'Dependents')}</th>
          </tr>
        </thead>
        <tbody>
          {jobs.map(job => {
            const execution = executions.get(job.id);
            const dependents = getDependents(job.id, jobs);
            const isSelected = selectedNode === job.id;
            
            return (
              <tr
                key={job.id}
                className={isSelected ? 'selected' : ''}
                onClick={() => onNodeClick(job.id)}
              >
                <td>
                  <div className="job-name-cell">
                    <StatusIcon status={execution?.status} />
                    {job.name}
                  </div>
                </td>
                <td>{job.type}</td>
                <td>
                  <StatusBadge status={execution?.status} />
                </td>
                <td>
                  {job.dependencies?.length || 0} {t('scheduler.jobs', 'jobs')}
                </td>
                <td>
                  {dependents.length} {t('scheduler.jobs', 'jobs')}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

// ============================================================================
// Helper Components
// ============================================================================

function StatusIcon({ status }: { status?: JobStatus }) {
  const icons: Record<JobStatus, string> = {
    [JobStatus.PENDING]: '⏳',
    [JobStatus.QUEUED]: '📥',
    [JobStatus.RUNNING]: '▶️',
    [JobStatus.COMPLETED]: '✅',
    [JobStatus.FAILED]: '❌',
    [JobStatus.CANCELLED]: '⏹️',
    [JobStatus.PAUSED]: '⏸️',
    [JobStatus.WAITING_DEPENDENCY]: '🔗',
    [JobStatus.WAITING_CALENDAR]: '📅',
    [JobStatus.RETRYING]: '🔄',
    [JobStatus.SKIPPED]: '⏭️',
  };
  
  return <span className="status-icon">{status ? icons[status] : '⚪'}</span>;
}

function StatusBadge({ status }: { status?: JobStatus }) {
  if (!status) return <span className="badge badge-secondary">No runs</span>;
  
  const badges: Record<JobStatus, { className: string; label: string }> = {
    [JobStatus.PENDING]: { className: 'badge-warning', label: 'Pending' },
    [JobStatus.QUEUED]: { className: 'badge-info', label: 'Queued' },
    [JobStatus.RUNNING]: { className: 'badge-info', label: 'Running' },
    [JobStatus.COMPLETED]: { className: 'badge-success', label: 'Completed' },
    [JobStatus.FAILED]: { className: 'badge-danger', label: 'Failed' },
    [JobStatus.CANCELLED]: { className: 'badge-secondary', label: 'Cancelled' },
    [JobStatus.PAUSED]: { className: 'badge-warning', label: 'Paused' },
    [JobStatus.WAITING_DEPENDENCY]: { className: 'badge-info', label: 'Waiting' },
    [JobStatus.WAITING_CALENDAR]: { className: 'badge-info', label: 'Scheduled' },
    [JobStatus.RETRYING]: { className: 'badge-warning', label: 'Retrying' },
    [JobStatus.SKIPPED]: { className: 'badge-secondary', label: 'Skipped' },
  };
  
  const config = badges[status] || badges[JobStatus.PENDING];
  return <span className={`badge ${config.className}`}>{config.label}</span>;
}

// ============================================================================
// Utility Functions
// ============================================================================

function buildGraphLayout(
  jobs: Job[],
  executions: Map<string, JobExecution>,
  selectedNode: string | null
): GraphLayout {
  const nodeWidth = 180;
  const nodeHeight = 60;
  const horizontalGap = 80;
  const verticalGap = 40;
  const padding = 50;
  
  // Calculate levels (topological sort)
  const levels = new Map<string, number>();
  const visited = new Set<string>();
  
  function calculateLevel(jobId: string): number {
    if (levels.has(jobId)) return levels.get(jobId)!;
    
    const job = jobs.find(j => j.id === jobId);
    if (!job || !job.dependencies || job.dependencies.length === 0) {
      levels.set(jobId, 0);
      return 0;
    }
    
    let maxDepLevel = 0;
    for (const dep of job.dependencies) {
      if (!visited.has(dep.job_id)) {
        visited.add(dep.job_id);
        maxDepLevel = Math.max(maxDepLevel, calculateLevel(dep.job_id) + 1);
      }
    }
    
    levels.set(jobId, maxDepLevel);
    return maxDepLevel;
  }
  
  jobs.forEach(job => calculateLevel(job.id));
  
  // Group jobs by level
  const levelGroups = new Map<number, Job[]>();
  jobs.forEach(job => {
    const level = levels.get(job.id) || 0;
    if (!levelGroups.has(level)) {
      levelGroups.set(level, []);
    }
    levelGroups.get(level)!.push(job);
  });
  
  // Calculate positions
  const nodes: GraphNode[] = [];
  const maxLevel = Math.max(...Array.from(levels.values()), 0);
  
  levelGroups.forEach((levelJobs, level) => {
    levelJobs.forEach((job, index) => {
      nodes.push({
        id: job.id,
        job,
        x: padding + level * (nodeWidth + horizontalGap),
        y: padding + index * (nodeHeight + verticalGap),
        level,
        execution: executions.get(job.id),
        selected: job.id === selectedNode,
      });
    });
  });
  
  // Build edges
  const edges: GraphEdge[] = [];
  jobs.forEach(job => {
    if (job.dependencies) {
      job.dependencies.forEach(dep => {
        edges.push({
          id: `${dep.job_id}-${job.id}`,
          source: dep.job_id,
          target: job.id,
          dependency: dep,
        });
      });
    }
  });
  
  // Calculate dimensions
  const maxNodesInLevel = Math.max(...Array.from(levelGroups.values()).map(g => g.length), 1);
  const width = padding * 2 + (maxLevel + 1) * (nodeWidth + horizontalGap);
  const height = padding * 2 + maxNodesInLevel * (nodeHeight + verticalGap);
  
  return { nodes, edges, width: Math.max(width, 800), height: Math.max(height, 400) };
}

function getDependents(jobId: string, jobs: Job[]): Job[] {
  return jobs.filter(job => 
    job.dependencies?.some(dep => dep.job_id === jobId)
  );
}

function getNodeFill(status?: JobStatus): string {
  if (!status) return '#ffffff';
  
  const fills: Record<JobStatus, string> = {
    [JobStatus.COMPLETED]: '#d1fae5',
    [JobStatus.RUNNING]: '#dbeafe',
    [JobStatus.FAILED]: '#fee2e2',
    [JobStatus.PENDING]: '#f3f4f6',
    [JobStatus.QUEUED]: '#e0e7ff',
    [JobStatus.CANCELLED]: '#f3f4f6',
    [JobStatus.PAUSED]: '#fef3c7',
    [JobStatus.WAITING_DEPENDENCY]: '#e0e7ff',
    [JobStatus.WAITING_CALENDAR]: '#e0e7ff',
    [JobStatus.RETRYING]: '#fef3c7',
    [JobStatus.SKIPPED]: '#f3f4f6',
  };
  
  return fills[status] || '#ffffff';
}

function getNodeStroke(status?: JobStatus): string {
  if (!status) return '#e5e7eb';
  
  const strokes: Record<JobStatus, string> = {
    [JobStatus.COMPLETED]: '#059669',
    [JobStatus.RUNNING]: '#2563eb',
    [JobStatus.FAILED]: '#dc2626',
    [JobStatus.PENDING]: '#9ca3af',
    [JobStatus.QUEUED]: '#4f46e5',
    [JobStatus.CANCELLED]: '#6b7280',
    [JobStatus.PAUSED]: '#d97706',
    [JobStatus.WAITING_DEPENDENCY]: '#4f46e5',
    [JobStatus.WAITING_CALENDAR]: '#4f46e5',
    [JobStatus.RETRYING]: '#d97706',
    [JobStatus.SKIPPED]: '#6b7280',
  };
  
  return strokes[status] || '#e5e7eb';
}

function getStatusColor(status?: JobStatus): string {
  if (!status) return '#9ca3af';
  
  const colors: Record<JobStatus, string> = {
    [JobStatus.COMPLETED]: '#059669',
    [JobStatus.RUNNING]: '#2563eb',
    [JobStatus.FAILED]: '#dc2626',
    [JobStatus.PENDING]: '#9ca3af',
    [JobStatus.QUEUED]: '#4f46e5',
    [JobStatus.CANCELLED]: '#6b7280',
    [JobStatus.PAUSED]: '#d97706',
    [JobStatus.WAITING_DEPENDENCY]: '#4f46e5',
    [JobStatus.WAITING_CALENDAR]: '#4f46e5',
    [JobStatus.RETRYING]: '#d97706',
    [JobStatus.SKIPPED]: '#6b7280',
  };
  
  return colors[status] || '#9ca3af';
}

function getDashArray(type: DependencyType): string {
  switch (type) {
    case DependencyType.FINISH_TO_START:
      return '';
    case DependencyType.START_TO_START:
      return '5,5';
    case DependencyType.FINISH_TO_FINISH:
      return '10,5';
    case DependencyType.START_TO_FINISH:
      return '5,10';
    default:
      return '';
  }
}

function formatDependencyType(type: DependencyType): string {
  const labels: Record<DependencyType, string> = {
    [DependencyType.FINISH_TO_START]: 'FS',
    [DependencyType.START_TO_START]: 'SS',
    [DependencyType.FINISH_TO_FINISH]: 'FF',
    [DependencyType.START_TO_FINISH]: 'SF',
  };
  return labels[type] || type;
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  if (ms < 3600000) return `${Math.floor(ms / 60000)}m ${Math.floor((ms % 60000) / 1000)}s`;
  return `${Math.floor(ms / 3600000)}h ${Math.floor((ms % 3600000) / 60000)}m`;
}

function truncate(str: string, maxLength: number): string {
  if (str.length <= maxLength) return str;
  return str.slice(0, maxLength - 1) + '…';
}

export default DependencyVisualizerPage;
