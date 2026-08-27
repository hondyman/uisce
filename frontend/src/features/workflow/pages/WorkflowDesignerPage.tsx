import React, { useCallback, useState } from 'react';
import { ReactFlowProvider, Node, Edge, useNodesState, useEdgesState, Connection, addEdge } from 'reactflow';
import { Box, Paper, Typography, Stack, Button, Chip, Divider, IconButton, Tooltip } from '@mui/material';
import {
  AccountTree as WorkflowIcon,
  PlayArrow as DeployIcon,
  Save as SaveIcon,
  Rule as AuditIcon,
  History as DiffIcon,
  CheckCircle as ValidIcon,
  Speed as PerformanceIcon
} from '@mui/icons-material';
import { DesignerCanvas } from '../components/DesignerCanvas';
import { PropertiesPanel } from '../components/PropertiesPanel';
import 'reactflow/dist/style.css';
import { devDebug } from '../../../utils/devLogger';

const initialNodes: Node[] = [
  { id: 'start', type: 'start', position: { x: 50, y: 250 }, data: { label: 'Order Placement Trigger' } },
  { id: 'node-1', type: 'activity', position: { x: 250, y: 100 }, data: { label: 'Vectorized Compliance Check (WASM)' } },
  { id: 'node-2', type: 'approval', position: { x: 450, y: 250 }, data: { label: 'Risk Officer Approval', sla: '4h' } },
  { id: 'node-3', type: 'decision', position: { x: 700, y: 100 }, data: { label: 'Exceeds Concentration Limit?' } },
  { id: 'node-4', type: 'event', position: { x: 250, y: 400 }, data: { label: 'Notify Portfolio Manager' } },
  { id: 'end', type: 'end', position: { x: 900, y: 250 }, data: { label: 'Route to CRIMS Execution' } },
];

const initialEdges: Edge[] = [
  { id: 'e1', source: 'start', target: 'node-1' },
  { id: 'e2', source: 'node-1', target: 'node-2' },
  { id: 'e3', source: 'node-2', target: 'node-3' },
  { id: 'e4', source: 'node-3', target: 'end', label: 'Within Limits' },
  { id: 'e5', source: 'node-3', target: 'node-2', label: 'Breach Detected', type: 'step' },
  { id: 'e6', source: 'node-4', target: 'end' },
];

export const WorkflowDesignerPage: React.FC = () => {
  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const [statusState, setStatusState] = useState<'Draft' | 'Active' | 'Deprecated'>('Active');

  const onConnect = useCallback((params: Connection) => setEdges((eds) => addEdge(params, eds)), [setEdges]);

  const onNodeSelect = useCallback((id: string | null) => {
    setSelectedNodeId(id);
  }, []);

  const selectedNode = nodes.find((n) => n.id === selectedNodeId) || null;

  const handleSave = useCallback(async (status: 'Draft' | 'Active' | 'Deprecated') => {
    setStatusState(status);
    const template = {
      name: "Institutional Order Routing Workflow",
      description: "Automated pre-trade compliance and maker-checker approval mesh",
      status,
      steps: nodes.map(n => ({
        id: n.id,
        name: n.data.label,
        type: n.type,
        sla: n.data.sla,
        role: n.data.role,
        activity_ref: n.data.activityRef
      })),
      transitions: edges.map(e => ({
        from: e.source,
        to: e.target
      })),
      audit: { hash_chain: true, policy_refs: [] }
    };

    devDebug("Saving Institutional Process:", template);
  }, [nodes, edges]);

  return (
    <ReactFlowProvider>
      <Box sx={{ display: 'flex', height: '100vh', width: '100%', flexDirection: 'column', bgcolor: '#071526', color: '#F8FAFC' }}>
        {/* Header Bar */}
        <Paper
          elevation={0}
          sx={{
            height: 60,
            px: 2.5,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            borderBottom: '1px solid #1E293B',
            bgcolor: '#071526',
            zIndex: 10
          }}
        >
          {/* Title & Badge */}
          <Stack direction="row" spacing={1.5} alignItems="center">
            <WorkflowIcon sx={{ color: '#00D4FF', fontSize: 26 }} />
            <Box>
              <Typography variant="subtitle1" sx={{ fontWeight: 700, fontSize: 15, lineHeight: 1.2 }}>
                Institutional Workflow Studio
              </Typography>
              <Typography variant="caption" sx={{ color: '#94A3B8' }}>
                Bitemporal Temporal Mesh & Multi-Party Maker-Checker Designer
              </Typography>
            </Box>
            <Chip
              icon={<ValidIcon sx={{ fontSize: 13, color: '#34D399 !important' }} />}
              label={statusState}
              size="small"
              sx={{
                bgcolor: statusState === 'Active' ? '#064E3B' : '#0B1E36',
                color: statusState === 'Active' ? '#34D399' : '#94A3B8',
                border: '1px solid #1E293B',
                fontWeight: 700,
                fontSize: 10,
                height: 20
              }}
            />
          </Stack>

          {/* Lifecycle State Switcher */}
          <Paper sx={{ display: 'flex', bgcolor: '#0B1E36', p: 0.5, border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Button
              size="small"
              onClick={() => handleSave('Draft')}
              sx={{
                fontSize: 11,
                fontWeight: 600,
                textTransform: 'none',
                color: statusState === 'Draft' ? '#00D4FF' : '#94A3B8',
                bgcolor: statusState === 'Draft' ? 'rgba(0, 212, 255, 0.1)' : 'transparent'
              }}
            >
              Draft
            </Button>
            <Button
              size="small"
              onClick={() => handleSave('Active')}
              sx={{
                fontSize: 11,
                fontWeight: 600,
                textTransform: 'none',
                color: statusState === 'Active' ? '#34D399' : '#94A3B8',
                bgcolor: statusState === 'Active' ? 'rgba(52, 211, 153, 0.1)' : 'transparent'
              }}
            >
              Published
            </Button>
            <Button
              size="small"
              onClick={() => handleSave('Deprecated')}
              sx={{
                fontSize: 11,
                fontWeight: 600,
                textTransform: 'none',
                color: statusState === 'Deprecated' ? '#F87171' : '#94A3B8',
                bgcolor: statusState === 'Deprecated' ? 'rgba(239, 68, 68, 0.1)' : 'transparent'
              }}
            >
              Deprecated
            </Button>
          </Paper>

          {/* Action Tools */}
          <Stack direction="row" spacing={1.5} alignItems="center">
            <Button
              variant="outlined"
              size="small"
              startIcon={<DiffIcon sx={{ fontSize: 15 }} />}
              sx={{ color: '#38BDF8', borderColor: '#0284C7', fontSize: 11, fontWeight: 600, textTransform: 'none' }}
            >
              Diff Viewer
            </Button>
            <Button
              variant="contained"
              size="small"
              startIcon={<DeployIcon sx={{ fontSize: 15 }} />}
              onClick={() => handleSave('Active')}
              sx={{ bgcolor: '#0284C7', color: '#FFF', fontSize: 11, fontWeight: 700, textTransform: 'none', '&:hover': { bgcolor: '#0369A1' } }}
            >
              Deploy to Temporal Mesh
            </Button>
          </Stack>
        </Paper>

        {/* Main Canvas & Properties */}
        <Box sx={{ display: 'flex', flex: 1, position: 'relative', overflow: 'hidden' }}>
          {/* Visual Canvas Area */}
          <Box sx={{ flex: 1, position: 'relative', bgcolor: '#0B1E36' }}>
            <DesignerCanvas
              nodes={nodes}
              edges={edges}
              onNodesChange={onNodesChange}
              onEdgesChange={onEdgesChange}
              onConnect={onConnect}
              onNodeSelect={onNodeSelect}
              setNodes={setNodes}
            />
          </Box>

          {/* Properties Drawer */}
          {selectedNode && (
            <Paper
              elevation={0}
              sx={{
                width: 320,
                bgcolor: '#071526',
                borderLeft: '1px solid #1E293B',
                zIndex: 5,
                overflowY: 'auto'
              }}
            >
              <PropertiesPanel node={selectedNode} />
            </Paper>
          )}
        </Box>
      </Box>
    </ReactFlowProvider>
  );
};

export default WorkflowDesignerPage;
