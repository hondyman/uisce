import React, { useState, useCallback, useRef, useEffect } from 'react';
import ReactFlow, {
  ReactFlowProvider,
  addEdge,
  useNodesState,
  useEdgesState,
  Controls,
  Background,
  MiniMap,
  Connection,
  Edge,
  Node,
  BackgroundVariant,
} from 'reactflow';
import 'reactflow/dist/style.css';

import {
  Box,
  Typography,
  Button,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Chip,
  CircularProgress,
  Dialog,
  DialogTitle,
  DialogContent,
  useTheme,
} from '@mui/material';
import {
  Play,
  Sparkles,
  Save,
  Zap,
  BookOpen,
} from 'lucide-react';
import axios from '@/utils/axiosClient';

import { PipelineTileNode } from '../components/nodes/PipelineTileNode';
import { PaletteSidebar } from '../components/PaletteSidebar';
import { TileConfigDrawer } from '../components/TileConfigDrawer';
import { ExecutionTelemetryHUD } from '../components/ExecutionTelemetryHUD';
import { LiveSimulationModal } from '../components/LiveSimulationModal';
import { PIPELINE_TEMPLATES, PipelineTemplate } from '../constants/pipelineTemplates';
import { PipelineDefinition, PipelineExecutionRun, PipelineNodeData } from '../types/pipeline';

const nodeTypes = {
  pipelineTile: PipelineTileNode,
};

export const DataPipelineStudioPage: React.FC = () => {
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';

  const reactFlowWrapper = useRef<HTMLDivElement>(null);
  const [nodes, setNodes, onNodesChange] = useNodesState<PipelineNodeData>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [reactFlowInstance, setReactFlowInstance] = useState<any>(null);

  // Pipeline metadata
  const [pipelineName, setPipelineName] = useState('Trade Order & Account Ingestion');
  const [pipelineDescription, setPipelineDescription] = useState('Parallel bulk loader replacing Informatica/Talend ETL flows');
  const [pipelineMode, setPipelineMode] = useState<'business_object' | 'catalog_graph' | 'hybrid'>('business_object');
  const [targetEntity, setTargetEntity] = useState('oms.trade_order');
  const [concurrency, setConcurrency] = useState(8);
  const [batchSize, setBatchSize] = useState(2000);

  // Selected node for config inspector
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);

  // Execution & Simulation states
  const [isExecuting, setIsExecuting] = useState(false);
  const [currentRun, setCurrentRun] = useState<PipelineExecutionRun | null>(null);
  const [showSimModal, setShowSimModal] = useState(false);
  const [showTemplateModal, setShowTemplateModal] = useState(false);
  const [showSampleOutputModal, setShowSampleOutputModal] = useState(false);

  // Load default template on initial mount
  useEffect(() => {
    applyTemplate(PIPELINE_TEMPLATES[0]);
  }, []);

  const applyTemplate = (tpl: PipelineTemplate) => {
    setPipelineName(tpl.name);
    setPipelineDescription(tpl.description);
    setPipelineMode(tpl.mode === 'external' ? 'business_object' : tpl.mode);
    setTargetEntity(tpl.targetEntity);
    setConcurrency(tpl.concurrency);
    setBatchSize(tpl.batchSize);
    setNodes(tpl.nodes);
    setEdges(tpl.edges);
    setSelectedNodeId(null);
    setShowTemplateModal(false);
  };

  const onConnect = useCallback(
    (params: Connection | Edge) =>
      setEdges((eds) =>
        addEdge(
          {
            ...params,
            animated: true,
            style: { stroke: '#3b82f6', strokeWidth: 2 },
          },
          eds
        )
      ),
    [setEdges]
  );

  const onDragOver = useCallback((event: React.DragEvent) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';
  }, []);

  const onDrop = useCallback(
    (event: React.DragEvent) => {
      event.preventDefault();

      const reactFlowBounds = reactFlowWrapper.current?.getBoundingClientRect();
      const rawData = event.dataTransfer.getData('application/reactflow');

      if (!rawData || !reactFlowBounds || !reactFlowInstance) return;

      const data = JSON.parse(rawData);
      const position = reactFlowInstance.project({
        x: event.clientX - reactFlowBounds.left,
        y: event.clientY - reactFlowBounds.top,
      });

      const newNode: Node<PipelineNodeData> = {
        id: `node-${Date.now()}`,
        type: 'pipelineTile',
        position,
        data,
      };

      setNodes((nds) => nds.concat(newNode));
      setSelectedNodeId(newNode.id);
    },
    [reactFlowInstance, setNodes]
  );

  const onNodeClick = (_: React.MouseEvent, node: Node) => {
    setSelectedNodeId(node.id);
  };

  const onPaneClick = () => {
    setSelectedNodeId(null);
  };

  const updateNodeConfig = (nodeId: string, updatedData: Partial<PipelineNodeData>) => {
    setNodes((nds) =>
      nds.map((node) => {
        if (node.id === nodeId) {
          return {
            ...node,
            data: {
              ...node.data,
              ...updatedData,
            },
          };
        }
        return node;
      })
    );
  };

  const selectedNode = nodes.find((n) => n.id === selectedNodeId) || null;

  // Build current pipeline definition
  const getCurrentPipelineDef = (): PipelineDefinition => ({
    id: 'active-pipeline',
    name: pipelineName,
    description: pipelineDescription,
    mode: pipelineMode,
    target_entity: targetEntity,
    concurrency,
    batch_size: batchSize,
    error_policy: 'skip_and_log',
    is_active: true,
    dag_json: {
      nodes: nodes.map((n) => ({
        id: n.id,
        type: n.data.category,
        subType: n.data.subType,
        label: n.data.label,
        config: n.data.config,
        position: n.position,
      })),
      edges: edges.map((e) => ({
        id: e.id,
        source: e.source,
        target: e.target,
      })),
    },
  });

  // Save Pipeline to Backend
  const handleSavePipeline = async () => {
    try {
      const def = getCurrentPipelineDef();
      await axios.post('/api/v1/data-pipelines', def);
      alert('Pipeline DAG persisted successfully to Uuisce database!');
    } catch (err: any) {
      alert('Failed to save pipeline: ' + (err.message || 'Unknown error'));
    }
  };

  // Trigger high-speed parallel run
  const handleExecuteParallelRun = async () => {
    try {
      setIsExecuting(true);
      const def = getCurrentPipelineDef();
      const res = await axios.post('/api/v1/data-pipelines/run', {
        pipeline: def,
      });
      setCurrentRun(res.data);

      // Update node visual status based on step telemetry
      if (res.data?.step_telemetry) {
        setNodes((nds) =>
          nds.map((n) => {
            const step = res.data.step_telemetry[n.id];
            if (step) {
              return {
                ...n,
                data: {
                  ...n.data,
                  metrics: {
                    status: step.status,
                    recordsIn: step.records_in,
                    recordsOut: step.records_out,
                    recordsError: step.records_error,
                    rowsPerSec: step.rows_per_sec,
                  },
                },
              };
            }
            return n;
          })
        );
      }
    } catch (err: any) {
      alert('Pipeline execution failed: ' + (err.message || 'Unknown error'));
    } finally {
      setIsExecuting(false);
    }
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: 'calc(100vh - 64px)', overflow: 'hidden', backgroundColor: theme.palette.background.default }}>
      {/* Top Studio Control Bar */}
      <Box
        sx={{
          height: 64,
          px: 2.5,
          backgroundColor: theme.palette.background.paper,
          borderBottom: `1px solid ${theme.palette.divider}`,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          zIndex: 10,
        }}
      >
        {/* Left: Title & Mode */}
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          <Box
            sx={{
              width: 38,
              height: 38,
              borderRadius: '8px',
              background: 'linear-gradient(135deg, #2563eb, #7c3aed)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: '#ffffff',
            }}
          >
            <Zap size={22} />
          </Box>
          <Box>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <Typography variant="subtitle1" sx={{ fontWeight: 800, color: theme.palette.text.primary, fontSize: '0.95rem' }}>
                {pipelineName}
              </Typography>
              <Chip
                label={pipelineMode === 'business_object' ? 'Mode 1: Business Objects (STI)' : 'Mode 2: Catalog Graph'}
                size="small"
                color={pipelineMode === 'business_object' ? 'primary' : 'secondary'}
                sx={{ height: 20, fontSize: '0.65rem', fontWeight: 700 }}
              />
            </Box>
            <Typography variant="caption" sx={{ color: theme.palette.text.secondary, display: 'block' }}>
              High-Throughput Parallel Pipeline Platform (Informatica / Talend Alternative)
            </Typography>
          </Box>
        </Box>

        {/* Center: Concurrency & Target Controls */}
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          <FormControl size="small" sx={{ minWidth: 150 }}>
            <InputLabel>Target Mode</InputLabel>
            <Select
              value={pipelineMode}
              label="Target Mode"
              onChange={(e: any) => setPipelineMode(e.target.value)}
              sx={{ height: 36, fontSize: '0.8rem' }}
            >
              <MenuItem value="business_object">Mode 1: STI Business Objects</MenuItem>
              <MenuItem value="catalog_graph">Mode 2: Catalog Graph</MenuItem>
            </Select>
          </FormControl>

          <FormControl size="small" sx={{ minWidth: 110 }}>
            <InputLabel>Workers</InputLabel>
            <Select
              value={concurrency}
              label="Workers"
              onChange={(e: any) => setConcurrency(e.target.value)}
              sx={{ height: 36, fontSize: '0.8rem' }}
            >
              <MenuItem value={2}>2 Workers</MenuItem>
              <MenuItem value={4}>4 Workers</MenuItem>
              <MenuItem value={8}>8 Workers</MenuItem>
              <MenuItem value={16}>16 Workers (Parallel)</MenuItem>
              <MenuItem value={32}>32 Workers (Turbo)</MenuItem>
            </Select>
          </FormControl>

          <Button
            size="small"
            variant="outlined"
            startIcon={<BookOpen size={14} />}
            onClick={() => setShowTemplateModal(true)}
            sx={{ height: 36, fontWeight: 700 }}
          >
            Templates
          </Button>
        </Box>

        {/* Right: Actions */}
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <Button
            size="small"
            variant="outlined"
            color="primary"
            startIcon={<Sparkles size={16} />}
            onClick={() => setShowSimModal(true)}
            sx={{ height: 36, fontWeight: 700 }}
          >
            Dry-Run Sim
          </Button>

          <Button
            size="small"
            variant="contained"
            color="primary"
            startIcon={isExecuting ? <CircularProgress size={16} color="inherit" /> : <Play size={16} />}
            onClick={handleExecuteParallelRun}
            disabled={isExecuting}
            sx={{
              height: 36,
              fontWeight: 700,
              background: 'linear-gradient(135deg, #2563eb, #1d4ed8)',
            }}
          >
            {isExecuting ? 'Streaming...' : 'Run Parallel Job'}
          </Button>

          <Button
            size="small"
            variant="contained"
            color="success"
            startIcon={<Save size={16} />}
            onClick={handleSavePipeline}
            sx={{ height: 36, fontWeight: 700 }}
          >
            Save Pipeline
          </Button>
        </Box>
      </Box>

      {/* Main Studio Canvas Area */}
      <Box sx={{ display: 'flex', flex: 1, position: 'relative', overflow: 'hidden' }}>
        {/* Left Drag Palette */}
        <PaletteSidebar />

        {/* Center Canvas */}
        <Box ref={reactFlowWrapper} sx={{ flex: 1, height: '100%', position: 'relative' }}>
          <ReactFlowProvider>
            <ReactFlow
              nodes={nodes}
              edges={edges}
              onNodesChange={onNodesChange}
              onEdgesChange={onEdgesChange}
              onConnect={onConnect}
              onInit={setReactFlowInstance}
              onDrop={onDrop}
              onDragOver={onDragOver}
              onNodeClick={onNodeClick}
              onPaneClick={onPaneClick}
              nodeTypes={nodeTypes}
              fitView
              snapToGrid
              snapGrid={[15, 15]}
              defaultEdgeOptions={{
                animated: true,
                style: { stroke: isDark ? '#60a5fa' : '#3b82f6', strokeWidth: 2 },
              }}
            >
              <Background
                variant={BackgroundVariant.Dots}
                gap={16}
                size={1}
                color={isDark ? '#334155' : '#cbd5e1'}
              />
              <Controls />
              <MiniMap
                nodeColor={(node: any) => {
                  switch (node.data?.category) {
                    case 'source': return '#3b82f6';
                    case 'transform': return '#8b5cf6';
                    case 'validator': return '#f59e0b';
                    case 'loader': return '#10b981';
                    case 'graph_synthesizer': return '#06b6d4';
                    default: return '#64748b';
                  }
                }}
                style={{
                  backgroundColor: isDark ? '#1e293b' : '#ffffff',
                  border: `1px solid ${theme.palette.divider}`,
                  borderRadius: 8,
                }}
                maskColor={isDark ? 'rgba(15, 23, 42, 0.7)' : 'rgba(240, 240, 240, 0.6)'}
              />
            </ReactFlow>
          </ReactFlowProvider>

          {/* Telemetry HUD at Bottom */}
          {currentRun && (
            <ExecutionTelemetryHUD
              run={currentRun}
              onClose={() => setCurrentRun(null)}
              onViewSample={() => setShowSampleOutputModal(true)}
            />
          )}
        </Box>

        {/* Right Node Inspector */}
        {selectedNode && (
          <TileConfigDrawer
            node={selectedNode}
            onClose={() => setSelectedNodeId(null)}
            onUpdateConfig={updateNodeConfig}
          />
        )}
      </Box>

      {/* Live Simulation Modal */}
      <LiveSimulationModal
        open={showSimModal}
        onClose={() => setShowSimModal(false)}
        pipeline={getCurrentPipelineDef()}
        onSimulationComplete={(run) => {
          setCurrentRun(run);
          if (run.step_telemetry) {
            setNodes((nds) =>
              nds.map((n) => {
                const step = run.step_telemetry[n.id];
                if (step) {
                  return {
                    ...n,
                    data: {
                      ...n.data,
                      metrics: {
                        status: step.status,
                        recordsIn: step.records_in,
                        recordsOut: step.records_out,
                        recordsError: step.records_error,
                        rowsPerSec: step.rows_per_sec,
                      },
                    },
                  };
                }
                return n;
              })
            );
          }
        }}
      />

      {/* Template Selection Dialog */}
      <Dialog
        open={showTemplateModal}
        onClose={() => setShowTemplateModal(false)}
        maxWidth="md"
        fullWidth
        PaperProps={{
          sx: {
            backgroundColor: theme.palette.background.paper,
            backgroundImage: 'none',
            border: `1px solid ${theme.palette.divider}`,
          },
        }}
      >
        <DialogTitle sx={{ fontWeight: 800, borderBottom: `1px solid ${theme.palette.divider}`, color: theme.palette.text.primary }}>
          Select Pre-Built Enterprise Pipeline Template
        </DialogTitle>
        <DialogContent sx={{ p: 2.5 }}>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            {PIPELINE_TEMPLATES.map((tpl) => (
              <Box
                key={tpl.id}
                onClick={() => applyTemplate(tpl)}
                sx={{
                  p: 2,
                  borderRadius: '10px',
                  backgroundColor: isDark ? 'rgba(255, 255, 255, 0.03)' : '#ffffff',
                  border: `1px solid ${theme.palette.divider}`,
                  cursor: 'pointer',
                  transition: 'all 0.2s ease',
                  '&:hover': {
                    borderColor: theme.palette.primary.main,
                    backgroundColor: isDark ? 'rgba(59, 130, 246, 0.1)' : 'rgba(59, 130, 246, 0.04)',
                    transform: 'translateY(-2px)',
                    boxShadow: isDark ? '0 4px 16px rgba(0,0,0,0.4)' : '0 4px 12px rgba(0,0,0,0.06)',
                  },
                }}
              >
                <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
                  <Typography variant="subtitle1" sx={{ fontWeight: 800, color: theme.palette.text.primary }}>
                    {tpl.name}
                  </Typography>
                  <Chip label={tpl.category} size="small" color="primary" sx={{ fontWeight: 700 }} />
                </Box>
                <Typography variant="body2" sx={{ color: theme.palette.text.secondary }}>
                  {tpl.description}
                </Typography>
              </Box>
            ))}
          </Box>
        </DialogContent>
      </Dialog>

      {/* Sample Output Viewer Dialog */}
      <Dialog
        open={showSampleOutputModal}
        onClose={() => setShowSampleOutputModal(false)}
        maxWidth="lg"
        fullWidth
        PaperProps={{
          sx: {
            backgroundColor: theme.palette.background.paper,
            backgroundImage: 'none',
            border: `1px solid ${theme.palette.divider}`,
          },
        }}
      >
        <DialogTitle sx={{ fontWeight: 800, borderBottom: `1px solid ${theme.palette.divider}`, color: theme.palette.text.primary }}>
          Pipeline Transformed Output Records ({currentRun?.sample_output?.length || 0})
        </DialogTitle>
        <DialogContent sx={{ p: 2.5 }}>
          <Box
            sx={{
              p: 2,
              backgroundColor: isDark ? 'rgba(0,0,0,0.3)' : '#f8fafc',
              borderRadius: '8px',
              border: `1px solid ${theme.palette.divider}`,
              maxHeight: 450,
              overflowY: 'auto',
            }}
          >
            <pre style={{ margin: 0, fontSize: '0.75rem', fontFamily: 'monospace', color: theme.palette.text.primary }}>
              {JSON.stringify(currentRun?.sample_output, null, 2)}
            </pre>
          </Box>
        </DialogContent>
      </Dialog>
    </Box>
  );
};

export default DataPipelineStudioPage;
