import React, { useState } from 'react';
import { ReactFlowProvider } from 'reactflow';
import { Box, Paper, Typography, Toolbar, AppBar, Button, IconButton, CircularProgress, Alert, Snackbar, Divider } from '@mui/material';
import Sidebar from './components/Sidebar';
import StreamCanvas from './components/StreamCanvas';
import ConfigPanel from './components/ConfigPanel';
import DebugPanel from './components/DebugPanel';
import BOSelector, { BusinessObject } from './components/BOSelector';
import TemplateGallery, { PipelineTemplate } from './components/TemplateGallery';
import useUisceStore from './hooks/useUisceStore';
import { runDebugTrace, runSimulation } from './api/uisceApi';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import SaveIcon from '@mui/icons-material/Save';
import DashboardIcon from '@mui/icons-material/Dashboard';
import AutoAwesomeIcon from '@mui/icons-material/AutoAwesome';

import HelpCenter from '../../components/HelpCenter/HelpCenter';

interface UisceBuilderProps {
  filterCategories?: any[]; // Using any[] for now to avoid circular deps or complex imports, but should be FilterCategory[]
}

const UisceBuilderContent = ({ filterCategories }: UisceBuilderProps) => {
  const { selectedNodeId, nodes, isDebugging, setIsDebugging, applyTraceResult, clearTraceResult, traceResult, setNodes, setEdges, addNode, selectedBO, setSelectedBO } = useUisceStore();
  // ... state ...

  const [currentStepIndex, setCurrentStepIndex] = useState(0);
  const [showSuccess, setShowSuccess] = useState(false);
  const [templateGalleryOpen, setTemplateGalleryOpen] = useState(false);

  const handleStep = () => setCurrentStepIndex((prev) => prev + 1);
  const handleContinue = () => {
    // Logic to run until end
    if (traceResult && traceResult.steps) {
        setCurrentStepIndex(traceResult.steps.length - 1);
    }
  };
  const handleStop = () => {
    setIsDebugging(false);
    clearTraceResult();
    setCurrentStepIndex(0);
  };
  const handleRestart = () => {
    setCurrentStepIndex(0);
  };

  const handleApplyTemplate = (template: PipelineTemplate) => {
      const startNode = {
          id: 'node-trigger',
          type: 'default',
          position: { x: 50, y: 100 },
          data: { label: 'API Trigger', filterType: 'Trigger', config: { triggerType: 'api' } },
          style: { backgroundColor: '#eff6ff', borderColor: '#3b82f6', color: '#1e40af', borderRadius: '50%', width: 80, height: 80, display: 'flex', justifyContent: 'center', alignItems: 'center', fontSize: '0.75rem', fontWeight: 'bold' }
      };

      const templateNodes = template.nodes.map((n, i) => ({
          id: `node-${Date.now()}-${i}`,
          type: n.type,
          position: { ...n.position, x: n.position.x + 200 },
          data: { label: n.label, filterType: n.type, config: {} }
      }));
      const allNodes = [startNode, ...templateNodes];
      
      const newEdges = [];
      for (let i = 0; i < allNodes.length - 1; i++) {
        newEdges.push({
          id: `edge-${allNodes[i].id}-${allNodes[i+1].id}`,
          source: allNodes[i].id,
          target: allNodes[i+1].id,
          type: 'smoothstep',
          animated: true,
          style: { stroke: '#64748b', strokeWidth: 2 }
        });
      }

      setNodes(allNodes);
      setEdges(newEdges);
      setShowSuccess(true);
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100vh', overflow: 'hidden', bgcolor: '#f1f5f9' }}>
      <AppBar position="static" color="default" elevation={1} sx={{ zIndex: 3, bgcolor: 'white', borderBottom: '1px solid rgba(0,0,0,0.06)' }}>
        <Toolbar variant="dense">
          <Typography variant="h6" noWrap component="div" sx={{ color: '#334155', fontWeight: 'bold', display: 'flex', alignItems: 'center', gap: 1, mr: 4 }}>
            <AutoAwesomeIcon sx={{ color: '#6366f1' }} />
            Workflow Studio
          </Typography>

          <Box sx={{ flexGrow: 1, display: 'flex', alignItems: 'center' }}>
             <BOSelector 
                selectedBOId={selectedBO?.id || null} 
                onSelectBO={setSelectedBO} 
             />
          </Box>
          
          <Button 
            startIcon={<DashboardIcon />} 
            onClick={() => setTemplateGalleryOpen(true)}
            sx={{ mr: 2, color: '#64748b', '&:hover': { bgcolor: 'rgba(0,0,0,0.04)' } }}
          >
            Templates
          </Button>
          
          <Button startIcon={<SaveIcon />} variant="contained" size="small" sx={{ mr: 1, bgcolor: '#0f172a' }}>
            Save Pipeline
          </Button>
          <Button startIcon={<PlayArrowIcon />} variant="outlined" size="small" color="success">
            Simulate
          </Button>
        </Toolbar>
      </AppBar>
      
      {/* Main Content */}
      <Box sx={{ display: 'flex', flexGrow: 1, overflow: 'hidden' }}>
        {/* 1. The Reservoir (Sidebar) */}
        <Paper square elevation={0} sx={{ width: 280, zIndex: 1, overflowY: 'auto', borderRight: '1px solid rgba(0,0,0,0.06)' }}>
          <Sidebar categories={filterCategories} />
        </Paper>

        {/* ... Canvas & Config Panel ... */}
        <Box sx={{ flexGrow: 1, position: 'relative' }}>
          <StreamCanvas />
        </Box>

        <Paper 
            square 
            elevation={4} 
            sx={{ 
                width: selectedNodeId ? 380 : 0,
                zIndex: 2, 
                borderLeft: '1px solid rgba(0,0,0,0.06)',
                overflowY: 'auto',
                transition: 'width 0.3s ease',
            }}
        >
            <ConfigPanel />
        </Paper>
      </Box>
      
      {/* ... Debug Panel & Notifications ... */} 
      {(isDebugging || traceResult) && (
        <DebugPanel
          traceResult={traceResult}
          isDebugging={isDebugging}
          currentStepIndex={currentStepIndex}
          onStep={handleStep}
          onContinue={handleContinue}
          onStop={handleStop}
          onRestart={handleRestart}
        />
      )}

      <Snackbar open={showSuccess} autoHideDuration={4000} onClose={() => setShowSuccess(false)} anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}>
        <Alert onClose={() => setShowSuccess(false)} severity="success" variant="filled" sx={{ width: '100%' }}>
          Pipeline saved successfully! Policy v2.1 is now active.
        </Alert>
      </Snackbar>

      <TemplateGallery 
        open={templateGalleryOpen}
        onClose={() => setTemplateGalleryOpen(false)}
        onApplyTemplate={handleApplyTemplate}
      />
      
      <HelpCenter context="workflow-studio" />
    </Box>
  );
};

export const UisceBuilder = ({ filterCategories }: UisceBuilderProps) => {
  return (
    <ReactFlowProvider>
      <UisceBuilderContent filterCategories={filterCategories} />
    </ReactFlowProvider>
  );
};

export default UisceBuilder;
