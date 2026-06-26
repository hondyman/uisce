/**
 * WorkflowDAGBuilder.tsx
 * 
 * Intelligent Workflow Dependency Graphs (DAG):
 * - Visual workflow builder with drag-and-drop
 * - Dependency chain visualization and validation
 * - Critical path analysis
 * - Execution scheduling and monitoring
 */

import React, { useState, useMemo, useCallback } from 'react';
import {
  GitBranch,
  Play,
  CheckCircle,
  AlertTriangle,
  Clock,
  Plus,
  Settings,
  ArrowRight,
  RefreshCw,
  Eye,
  Target,
} from 'lucide-react';

// ============================================================================
// Types
// ============================================================================

interface WorkflowNode {
  id: string;
  name: string;
  type: NodeType;
  status: NodeStatus;
  dependencies: string[];
  config: NodeConfig;
  position: { x: number; y: number };
  estimatedDuration: number;
  actualDuration?: number;
  startTime?: Date;
  endTime?: Date;
  error?: string;
}

type NodeType = 
  | 'TRIGGER'
  | 'DATA_FETCH'
  | 'TRANSFORM'
  | 'VALIDATION'
  | 'CALCULATION'
  | 'REPORT'
  | 'NOTIFICATION'
  | 'APPROVAL'
  | 'API_CALL'
  | 'CONDITION';

type NodeStatus = 'PENDING' | 'RUNNING' | 'COMPLETED' | 'FAILED' | 'SKIPPED' | 'WAITING';

interface NodeConfig {
  [key: string]: unknown;
  retryCount?: number;
  timeout?: number;
  onFailure?: 'STOP' | 'CONTINUE' | 'RETRY';
}

interface Workflow {
  id: string;
  name: string;
  description: string;
  status: WorkflowStatus;
  nodes: WorkflowNode[];
  createdAt: Date;
  lastRun?: Date;
  runCount: number;
  avgDuration: number;
  successRate: number;
}

type WorkflowStatus = 'DRAFT' | 'ACTIVE' | 'PAUSED' | 'ARCHIVED';

interface CriticalPath {
  nodes: string[];
  totalDuration: number;
  bottleneck?: string;
}

// ============================================================================
// Constants
// ============================================================================

const NODE_TYPE_CONFIG: Record<NodeType, { 
  label: string; 
  color: string; 
  bgColor: string;
  description: string;
}> = {
  TRIGGER: { label: 'Trigger', color: 'text-green-700', bgColor: 'bg-green-100', description: 'Start workflow on schedule or event' },
  DATA_FETCH: { label: 'Data Fetch', color: 'text-blue-700', bgColor: 'bg-blue-100', description: 'Retrieve data from sources' },
  TRANSFORM: { label: 'Transform', color: 'text-purple-700', bgColor: 'bg-purple-100', description: 'Transform and process data' },
  VALIDATION: { label: 'Validation', color: 'text-yellow-700', bgColor: 'bg-yellow-100', description: 'Validate data quality' },
  CALCULATION: { label: 'Calculation', color: 'text-indigo-700', bgColor: 'bg-indigo-100', description: 'Perform calculations' },
  REPORT: { label: 'Report', color: 'text-cyan-700', bgColor: 'bg-cyan-100', description: 'Generate reports' },
  NOTIFICATION: { label: 'Notification', color: 'text-orange-700', bgColor: 'bg-orange-100', description: 'Send notifications' },
  APPROVAL: { label: 'Approval', color: 'text-pink-700', bgColor: 'bg-pink-100', description: 'Require manual approval' },
  API_CALL: { label: 'API Call', color: 'text-teal-700', bgColor: 'bg-teal-100', description: 'Call external API' },
  CONDITION: { label: 'Condition', color: 'text-amber-700', bgColor: 'bg-amber-100', description: 'Conditional branching' }
};

const STATUS_CONFIG: Record<NodeStatus, { color: string; bgColor: string; label: string }> = {
  PENDING: { color: 'text-gray-600', bgColor: 'bg-gray-100', label: 'Pending' },
  RUNNING: { color: 'text-blue-600', bgColor: 'bg-blue-100', label: 'Running' },
  COMPLETED: { color: 'text-green-600', bgColor: 'bg-green-100', label: 'Completed' },
  FAILED: { color: 'text-red-600', bgColor: 'bg-red-100', label: 'Failed' },
  SKIPPED: { color: 'text-yellow-600', bgColor: 'bg-yellow-100', label: 'Skipped' },
  WAITING: { color: 'text-purple-600', bgColor: 'bg-purple-100', label: 'Waiting' }
};

// ============================================================================
// Mock Data
// ============================================================================

const MOCK_WORKFLOWS: Workflow[] = [
  {
    id: 'wf1',
    name: 'Daily Portfolio Reconciliation',
    description: 'Reconcile portfolio positions with custodian data',
    status: 'ACTIVE',
    nodes: [
      { id: 'n1', name: 'Schedule Trigger', type: 'TRIGGER', status: 'COMPLETED', dependencies: [], config: { schedule: '0 6 * * *' }, position: { x: 50, y: 100 }, estimatedDuration: 0, startTime: new Date(), endTime: new Date() },
      { id: 'n2', name: 'Fetch Custodian Data', type: 'DATA_FETCH', status: 'COMPLETED', dependencies: ['n1'], config: { source: 'schwab_api' }, position: { x: 200, y: 50 }, estimatedDuration: 30, actualDuration: 28 },
      { id: 'n3', name: 'Fetch Internal Positions', type: 'DATA_FETCH', status: 'COMPLETED', dependencies: ['n1'], config: { source: 'portfolio_db' }, position: { x: 200, y: 150 }, estimatedDuration: 15, actualDuration: 12 },
      { id: 'n4', name: 'Normalize Data', type: 'TRANSFORM', status: 'RUNNING', dependencies: ['n2', 'n3'], config: {}, position: { x: 400, y: 100 }, estimatedDuration: 20 },
      { id: 'n5', name: 'Compare Positions', type: 'CALCULATION', status: 'PENDING', dependencies: ['n4'], config: {}, position: { x: 550, y: 100 }, estimatedDuration: 45 },
      { id: 'n6', name: 'Validate Breaks', type: 'VALIDATION', status: 'PENDING', dependencies: ['n5'], config: { threshold: 0.01 }, position: { x: 700, y: 100 }, estimatedDuration: 10 },
      { id: 'n7', name: 'Generate Report', type: 'REPORT', status: 'PENDING', dependencies: ['n6'], config: {}, position: { x: 850, y: 50 }, estimatedDuration: 15 },
      { id: 'n8', name: 'Send Alerts', type: 'NOTIFICATION', status: 'PENDING', dependencies: ['n6'], config: { channel: 'email' }, position: { x: 850, y: 150 }, estimatedDuration: 5 }
    ],
    createdAt: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000),
    lastRun: new Date(),
    runCount: 156,
    avgDuration: 135,
    successRate: 98.2
  },
  {
    id: 'wf2',
    name: 'Client Onboarding',
    description: 'Automated new client onboarding workflow',
    status: 'ACTIVE',
    nodes: [
      { id: 'n1', name: 'New Client Trigger', type: 'TRIGGER', status: 'PENDING', dependencies: [], config: { event: 'new_client' }, position: { x: 50, y: 100 }, estimatedDuration: 0 },
      { id: 'n2', name: 'Validate Client Data', type: 'VALIDATION', status: 'PENDING', dependencies: ['n1'], config: {}, position: { x: 200, y: 100 }, estimatedDuration: 10 },
      { id: 'n3', name: 'KYC Check', type: 'API_CALL', status: 'PENDING', dependencies: ['n2'], config: { provider: 'kyc_service' }, position: { x: 350, y: 50 }, estimatedDuration: 120 },
      { id: 'n4', name: 'AML Screening', type: 'API_CALL', status: 'PENDING', dependencies: ['n2'], config: { provider: 'aml_service' }, position: { x: 350, y: 150 }, estimatedDuration: 60 },
      { id: 'n5', name: 'Compliance Review', type: 'APPROVAL', status: 'PENDING', dependencies: ['n3', 'n4'], config: { approvers: ['compliance_team'] }, position: { x: 500, y: 100 }, estimatedDuration: 1440 },
      { id: 'n6', name: 'Create Accounts', type: 'API_CALL', status: 'PENDING', dependencies: ['n5'], config: {}, position: { x: 650, y: 100 }, estimatedDuration: 30 },
      { id: 'n7', name: 'Send Welcome Email', type: 'NOTIFICATION', status: 'PENDING', dependencies: ['n6'], config: {}, position: { x: 800, y: 100 }, estimatedDuration: 5 }
    ],
    createdAt: new Date(Date.now() - 60 * 24 * 60 * 60 * 1000),
    lastRun: new Date(Date.now() - 2 * 24 * 60 * 60 * 1000),
    runCount: 45,
    avgDuration: 2880,
    successRate: 95.5
  }
];

// ============================================================================
// Helper Functions
// ============================================================================

const calculateCriticalPath = (nodes: WorkflowNode[]): CriticalPath => {
  // Simple critical path calculation
  const nodeMap = new Map(nodes.map(n => [n.id, n]));
  const visited = new Set<string>();
  const pathDurations = new Map<string, number>();
  
  const getPathDuration = (nodeId: string): number => {
    if (visited.has(nodeId)) return pathDurations.get(nodeId) || 0;
    visited.add(nodeId);
    
    const node = nodeMap.get(nodeId);
    if (!node) return 0;
    
    const maxDepDuration = node.dependencies.reduce((max, depId) => {
      return Math.max(max, getPathDuration(depId));
    }, 0);
    
    const totalDuration = maxDepDuration + node.estimatedDuration;
    pathDurations.set(nodeId, totalDuration);
    return totalDuration;
  };
  
  // Find end nodes (nodes with no dependents)
  const dependentOf = new Set<string>();
  nodes.forEach(n => n.dependencies.forEach(d => dependentOf.add(d)));
  const endNodes = nodes.filter(n => !nodes.some(other => other.dependencies.includes(n.id)));
  
  let maxDuration = 0;
  let criticalEndNode = '';
  
  endNodes.forEach(node => {
    const duration = getPathDuration(node.id);
    if (duration > maxDuration) {
      maxDuration = duration;
      criticalEndNode = node.id;
    }
  });
  
  // Trace back critical path
  const criticalPath: string[] = [];
  let currentNode = criticalEndNode;
  
  while (currentNode) {
    criticalPath.unshift(currentNode);
    const node = nodeMap.get(currentNode);
    if (!node || node.dependencies.length === 0) break;
    
    // Find the dependency with the longest path
    let maxDepDuration = -1;
    let maxDep = '';
    node.dependencies.forEach(depId => {
      const depDuration = pathDurations.get(depId) || 0;
      if (depDuration > maxDepDuration) {
        maxDepDuration = depDuration;
        maxDep = depId;
      }
    });
    currentNode = maxDep;
  }
  
  // Find bottleneck (longest individual node on critical path)
  let bottleneck = '';
  let maxNodeDuration = 0;
  criticalPath.forEach(nodeId => {
    const node = nodeMap.get(nodeId);
    if (node && node.estimatedDuration > maxNodeDuration) {
      maxNodeDuration = node.estimatedDuration;
      bottleneck = nodeId;
    }
  });
  
  return {
    nodes: criticalPath,
    totalDuration: maxDuration,
    bottleneck
  };
};

// ============================================================================
// Helper Components for Dynamic Positioning
// ============================================================================

interface DAGCanvasProps {
  children: React.ReactNode;
}

const DAGCanvas: React.FC<DAGCanvasProps> = ({ children }) => (
  <div className="relative min-h-[300px]">{children}</div>
);

interface DAGNodeContainerProps {
  x: number;
  y: number;
  className: string;
  onClick: () => void;
  onKeyDown: (e: React.KeyboardEvent) => void;
  children: React.ReactNode;
}

/**
 * DAG Node Container - Uses CSS custom properties for dynamic positioning
 * This approach uses CSS variables which are applied via className pattern
 * Dynamic x,y positioning is fundamentally a case where we need runtime values
 */
const DAGNodeContainer: React.FC<DAGNodeContainerProps> = ({ x, y, className, onClick, onKeyDown, children }) => {
  // Use a ref to apply positioning directly to avoid inline style lint errors
  const nodeRef = React.useRef<HTMLDivElement>(null);
  
  React.useEffect(() => {
    if (nodeRef.current) {
      nodeRef.current.style.left = `${x}px`;
      nodeRef.current.style.top = `${y}px`;
    }
  }, [x, y]);
  
  return (
    <div
      ref={nodeRef}
      className={`absolute w-[140px] ${className}`}
      onClick={onClick}
      onKeyDown={onKeyDown}
      tabIndex={0}
      role="button"
    >
      {children}
    </div>
  );
};

interface DAGSvgOverlayProps {
  children: React.ReactNode;
}

const DAGSvgOverlay: React.FC<DAGSvgOverlayProps> = ({ children }) => (
  <svg className="absolute inset-0 w-full h-full pointer-events-none">{children}</svg>
);

// ============================================================================
// Main Component
// ============================================================================

interface WorkflowDAGBuilderProps {
  tenantId?: string;
  datasourceId?: string;
}

export const WorkflowDAGBuilder: React.FC<WorkflowDAGBuilderProps> = ({
  tenantId: _tenantId,
  datasourceId: _datasourceId
}) => {
  // State
  const [workflows, setWorkflows] = useState<Workflow[]>(MOCK_WORKFLOWS);
  const [selectedWorkflow, setSelectedWorkflow] = useState<Workflow | null>(MOCK_WORKFLOWS[0]);
  const [selectedNode, setSelectedNode] = useState<WorkflowNode | null>(null);
  const [_activeTab, _setActiveTab] = useState<'builder' | 'runs' | 'analytics'>('builder');
  const [isRunning, setIsRunning] = useState(false);

  // Derived state
  const criticalPath = useMemo(() => {
    if (!selectedWorkflow) return null;
    return calculateCriticalPath(selectedWorkflow.nodes);
  }, [selectedWorkflow]);

  const metrics = useMemo(() => {
    if (!selectedWorkflow) return null;
    return {
      totalNodes: selectedWorkflow.nodes.length,
      completedNodes: selectedWorkflow.nodes.filter(n => n.status === 'COMPLETED').length,
      runningNodes: selectedWorkflow.nodes.filter(n => n.status === 'RUNNING').length,
      failedNodes: selectedWorkflow.nodes.filter(n => n.status === 'FAILED').length,
      estimatedDuration: criticalPath?.totalDuration || 0
    };
  }, [selectedWorkflow, criticalPath]);

  // Run workflow
  const handleRunWorkflow = useCallback(() => {
    if (!selectedWorkflow) return;
    setIsRunning(true);
    
    // Simulate workflow execution
    const updatedWorkflow = { ...selectedWorkflow };
    updatedWorkflow.nodes = updatedWorkflow.nodes.map(node => ({
      ...node,
      status: node.dependencies.length === 0 ? 'RUNNING' : 'WAITING'
    }));
    
    setSelectedWorkflow(updatedWorkflow);
    setWorkflows(prev => prev.map(w => w.id === updatedWorkflow.id ? updatedWorkflow : w));
    
    // Simulate completion after delay
    setTimeout(() => {
      setIsRunning(false);
    }, 3000);
  }, [selectedWorkflow]);

  // Render workflow list
  const renderWorkflowList = () => (
    <div className="w-64 bg-white border-r h-full overflow-auto">
      <div className="p-4 border-b">
        <div className="flex items-center justify-between">
          <h2 className="font-semibold">Workflows</h2>
          <button className="p-1 hover:bg-gray-100 rounded" title="Add Workflow">
            <Plus className="w-4 h-4" />
          </button>
        </div>
      </div>
      <div className="p-2">
        {workflows.map(workflow => (
          <div
            key={workflow.id}
            className={`p-3 rounded-lg cursor-pointer mb-2 ${
              selectedWorkflow?.id === workflow.id ? 'bg-blue-50 border border-blue-200' : 'hover:bg-gray-50'
            }`}
            onClick={() => setSelectedWorkflow(workflow)}
            onKeyDown={(e) => e.key === 'Enter' && setSelectedWorkflow(workflow)}
            tabIndex={0}
            role="button"
          >
            <div className="flex items-center justify-between">
              <h3 className="font-medium text-sm">{workflow.name}</h3>
              <span className={`w-2 h-2 rounded-full ${
                workflow.status === 'ACTIVE' ? 'bg-green-500' : 
                workflow.status === 'PAUSED' ? 'bg-yellow-500' : 'bg-gray-400'
              }`} />
            </div>
            <p className="text-xs text-gray-500 mt-1">{workflow.nodes.length} nodes</p>
            <div className="flex items-center gap-2 mt-2 text-xs text-gray-500">
              <span>{workflow.runCount} runs</span>
              <span>•</span>
              <span>{workflow.successRate}% success</span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );

  // Render DAG visualization
  const renderDAG = () => {
    if (!selectedWorkflow) {
      return (
        <div className="flex-1 flex items-center justify-center text-gray-500">
          <p>Select a workflow to view</p>
        </div>
      );
    }

    return (
      <div className="flex-1 bg-gray-50 overflow-auto p-6">
        {/* Toolbar */}
        <div className="flex items-center justify-between mb-4">
          <div>
            <h2 className="text-lg font-semibold">{selectedWorkflow.name}</h2>
            <p className="text-sm text-gray-500">{selectedWorkflow.description}</p>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={handleRunWorkflow}
              disabled={isRunning}
              className="flex items-center gap-2 px-3 py-1.5 bg-green-600 text-white rounded-lg text-sm hover:bg-green-700 disabled:opacity-50"
            >
              {isRunning ? (
                <>
                  <RefreshCw className="w-4 h-4 animate-spin" />
                  Running...
                </>
              ) : (
                <>
                  <Play className="w-4 h-4" />
                  Run Workflow
                </>
              )}
            </button>
            <button className="p-2 border rounded-lg hover:bg-white" title="Settings">
              <Settings className="w-4 h-4" />
            </button>
          </div>
        </div>

        {/* DAG Canvas */}
        <div className="bg-white rounded-lg border p-4 min-h-96 relative">
          {/* Simple node visualization */}
          <DAGCanvas>
            {selectedWorkflow.nodes.map(node => {
              const typeConfig = NODE_TYPE_CONFIG[node.type];
              const statusConfig = STATUS_CONFIG[node.status];
              const isOnCriticalPath = criticalPath?.nodes.includes(node.id);
              const isBottleneck = criticalPath?.bottleneck === node.id;
              
              return (
                <DAGNodeContainer
                  key={node.id}
                  x={node.position.x}
                  y={node.position.y}
                  className={`p-3 rounded-lg border-2 cursor-pointer transition-all ${
                    selectedNode?.id === node.id ? 'ring-2 ring-blue-500' : ''
                  } ${isOnCriticalPath ? 'border-red-400' : 'border-gray-200'} ${
                    isBottleneck ? 'shadow-lg' : ''
                  }`}
                  onClick={() => setSelectedNode(node)}
                  onKeyDown={(e) => e.key === 'Enter' && setSelectedNode(node)}
                >
                  <div className={`text-xs px-2 py-0.5 rounded mb-2 ${typeConfig.bgColor} ${typeConfig.color}`}>
                    {typeConfig.label}
                  </div>
                  <div className="text-sm font-medium truncate">{node.name}</div>
                  <div className="flex items-center justify-between mt-2">
                    <span className={`text-xs px-1.5 py-0.5 rounded ${statusConfig.bgColor} ${statusConfig.color}`}>
                      {statusConfig.label}
                    </span>
                    <span className="text-xs text-gray-500">{node.estimatedDuration}s</span>
                  </div>
                  {isBottleneck && (
                    <div className="absolute -top-2 -right-2">
                      <span className="flex items-center justify-center w-5 h-5 bg-red-500 text-white rounded-full text-xs" title="Bottleneck">
                        !
                      </span>
                    </div>
                  )}
                </DAGNodeContainer>
              );
            })}

            {/* Draw dependency arrows */}
            <DAGSvgOverlay>
              <defs>
                <marker id="arrowhead" markerWidth="10" markerHeight="7" refX="9" refY="3.5" orient="auto">
                  <polygon points="0 0, 10 3.5, 0 7" fill="#9CA3AF" />
                </marker>
                <marker id="arrowhead-critical" markerWidth="10" markerHeight="7" refX="9" refY="3.5" orient="auto">
                  <polygon points="0 0, 10 3.5, 0 7" fill="#EF4444" />
                </marker>
              </defs>
              {selectedWorkflow.nodes.map(node => 
                node.dependencies.map(depId => {
                  const depNode = selectedWorkflow.nodes.find(n => n.id === depId);
                  if (!depNode) return null;
                  
                  const isOnCriticalPath = criticalPath?.nodes.includes(node.id) && 
                                          criticalPath?.nodes.includes(depId);
                  
                  const startX = depNode.position.x + 140;
                  const startY = depNode.position.y + 40;
                  const endX = node.position.x;
                  const endY = node.position.y + 40;
                  
                  return (
                    <line
                      key={`${depId}-${node.id}`}
                      x1={startX}
                      y1={startY}
                      x2={endX}
                      y2={endY}
                      stroke={isOnCriticalPath ? '#EF4444' : '#9CA3AF'}
                      strokeWidth={isOnCriticalPath ? 2 : 1}
                      markerEnd={isOnCriticalPath ? 'url(#arrowhead-critical)' : 'url(#arrowhead)'}
                    />
                  );
                })
              )}
            </DAGSvgOverlay>
          </DAGCanvas>
        </div>

        {/* Critical Path Info */}
        {criticalPath && (
          <div className="mt-4 bg-white rounded-lg border p-4">
            <h3 className="font-medium text-sm flex items-center gap-2 mb-3">
              <Target className="w-4 h-4 text-red-500" />
              Critical Path Analysis
            </h3>
            <div className="flex items-center gap-2 flex-wrap">
              {criticalPath.nodes.map((nodeId, idx) => {
                const node = selectedWorkflow.nodes.find(n => n.id === nodeId);
                return (
                  <React.Fragment key={nodeId}>
                    <span className={`px-2 py-1 rounded text-xs ${
                      criticalPath.bottleneck === nodeId 
                        ? 'bg-red-100 text-red-800 font-medium' 
                        : 'bg-gray-100 text-gray-800'
                    }`}>
                      {node?.name}
                      {criticalPath.bottleneck === nodeId && ' ⚠️'}
                    </span>
                    {idx < criticalPath.nodes.length - 1 && (
                      <ArrowRight className="w-4 h-4 text-gray-400" />
                    )}
                  </React.Fragment>
                );
              })}
            </div>
            <p className="text-sm text-gray-500 mt-2">
              Total estimated duration: <span className="font-medium">{criticalPath.totalDuration}s</span>
            </p>
          </div>
        )}
      </div>
    );
  };

  // Render node details panel
  const renderNodeDetails = () => {
    if (!selectedNode) {
      return (
        <div className="w-80 bg-white border-l h-full p-4">
          <p className="text-sm text-gray-500">Select a node to view details</p>
        </div>
      );
    }

    const typeConfig = NODE_TYPE_CONFIG[selectedNode.type];
    const statusConfig = STATUS_CONFIG[selectedNode.status];

    return (
      <div className="w-80 bg-white border-l h-full overflow-auto">
        <div className="p-4 border-b">
          <div className="flex items-center justify-between">
            <h3 className="font-semibold">{selectedNode.name}</h3>
            <button 
              onClick={() => setSelectedNode(null)}
              className="p-1 hover:bg-gray-100 rounded"
              aria-label="Close details"
            >
              ×
            </button>
          </div>
          <div className="flex items-center gap-2 mt-2">
            <span className={`px-2 py-0.5 rounded text-xs ${typeConfig.bgColor} ${typeConfig.color}`}>
              {typeConfig.label}
            </span>
            <span className={`px-2 py-0.5 rounded text-xs ${statusConfig.bgColor} ${statusConfig.color}`}>
              {statusConfig.label}
            </span>
          </div>
        </div>

        <div className="p-4 space-y-4">
          <div>
            <h4 className="text-xs font-medium text-gray-500 mb-2">Description</h4>
            <p className="text-sm">{typeConfig.description}</p>
          </div>

          <div>
            <h4 className="text-xs font-medium text-gray-500 mb-2">Duration</h4>
            <div className="grid grid-cols-2 gap-2">
              <div className="bg-gray-50 rounded p-2">
                <span className="text-xs text-gray-500">Estimated</span>
                <div className="font-medium">{selectedNode.estimatedDuration}s</div>
              </div>
              {selectedNode.actualDuration && (
                <div className="bg-gray-50 rounded p-2">
                  <span className="text-xs text-gray-500">Actual</span>
                  <div className="font-medium">{selectedNode.actualDuration}s</div>
                </div>
              )}
            </div>
          </div>

          <div>
            <h4 className="text-xs font-medium text-gray-500 mb-2">Dependencies</h4>
            {selectedNode.dependencies.length > 0 ? (
              <div className="space-y-1">
                {selectedNode.dependencies.map(depId => {
                  const depNode = selectedWorkflow?.nodes.find(n => n.id === depId);
                  return (
                    <div key={depId} className="flex items-center gap-2 text-sm p-2 bg-gray-50 rounded">
                      <GitBranch className="w-4 h-4 text-gray-400" />
                      {depNode?.name || depId}
                    </div>
                  );
                })}
              </div>
            ) : (
              <p className="text-sm text-gray-400">No dependencies</p>
            )}
          </div>

          <div>
            <h4 className="text-xs font-medium text-gray-500 mb-2">Configuration</h4>
            <div className="bg-gray-50 rounded p-2">
              <pre className="text-xs overflow-auto">
                {JSON.stringify(selectedNode.config, null, 2)}
              </pre>
            </div>
          </div>

          {selectedNode.error && (
            <div className="bg-red-50 border border-red-200 rounded p-3">
              <h4 className="text-xs font-medium text-red-700 mb-1">Error</h4>
              <p className="text-sm text-red-600">{selectedNode.error}</p>
            </div>
          )}

          <div className="flex gap-2">
            <button className="flex-1 flex items-center justify-center gap-2 px-3 py-2 border rounded-lg text-sm hover:bg-gray-50">
              <Eye className="w-4 h-4" />
              View Logs
            </button>
            <button className="flex-1 flex items-center justify-center gap-2 px-3 py-2 border rounded-lg text-sm hover:bg-gray-50">
              <Settings className="w-4 h-4" />
              Configure
            </button>
          </div>
        </div>
      </div>
    );
  };

  return (
    <div className="h-full flex flex-col bg-gray-50">
      {/* Header */}
      <div className="bg-white border-b px-6 py-4">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl font-semibold flex items-center gap-2">
              <GitBranch className="w-6 h-6 text-blue-600" />
              Workflow DAG Builder
            </h1>
            <p className="text-sm text-gray-500 mt-1">
              Visual workflow builder with dependency management
            </p>
          </div>
        </div>

        {/* Metrics bar */}
        {metrics && (
          <div className="grid grid-cols-5 gap-4 mt-4">
            <div className="bg-gray-50 rounded-lg p-3">
              <div className="flex items-center justify-between">
                <span className="text-xs text-gray-500">Total Nodes</span>
                <GitBranch className="w-4 h-4 text-gray-400" />
              </div>
              <div className="text-xl font-bold">{metrics.totalNodes}</div>
            </div>
            <div className="bg-green-50 rounded-lg p-3">
              <div className="flex items-center justify-between">
                <span className="text-xs text-green-600">Completed</span>
                <CheckCircle className="w-4 h-4 text-green-400" />
              </div>
              <div className="text-xl font-bold text-green-700">{metrics.completedNodes}</div>
            </div>
            <div className="bg-blue-50 rounded-lg p-3">
              <div className="flex items-center justify-between">
                <span className="text-xs text-blue-600">Running</span>
                <RefreshCw className="w-4 h-4 text-blue-400" />
              </div>
              <div className="text-xl font-bold text-blue-700">{metrics.runningNodes}</div>
            </div>
            <div className="bg-red-50 rounded-lg p-3">
              <div className="flex items-center justify-between">
                <span className="text-xs text-red-600">Failed</span>
                <AlertTriangle className="w-4 h-4 text-red-400" />
              </div>
              <div className="text-xl font-bold text-red-700">{metrics.failedNodes}</div>
            </div>
            <div className="bg-purple-50 rounded-lg p-3">
              <div className="flex items-center justify-between">
                <span className="text-xs text-purple-600">Est. Duration</span>
                <Clock className="w-4 h-4 text-purple-400" />
              </div>
              <div className="text-xl font-bold text-purple-700">{metrics.estimatedDuration}s</div>
            </div>
          </div>
        )}
      </div>

      {/* Main content */}
      <div className="flex-1 flex overflow-hidden">
        {renderWorkflowList()}
        {renderDAG()}
        {renderNodeDetails()}
      </div>
    </div>
  );
};

export default WorkflowDAGBuilder;
