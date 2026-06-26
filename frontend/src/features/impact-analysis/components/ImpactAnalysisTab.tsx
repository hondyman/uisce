import React, { useState } from 'react';
import { Box, IconButton, Tooltip, Typography, Paper } from '@mui/material';
import { NodeType } from '../types';
import { ImpactGraph } from './ImpactGraph';
import { ImpactExplanation } from './ImpactExplanation';
import { ImpactQA } from './ImpactQA';
import GraphIcon from '@mui/icons-material/Hub';
import TextIcon from '@mui/icons-material/Description';
import ChatIcon from '@mui/icons-material/Chat';
import SidebarIcon from '@mui/icons-material/ViewSidebar';
import CloseIcon from '@mui/icons-material/Close';
import './ImpactAnalysis.css';

interface ImpactAnalysisTabProps {
  nodeType: NodeType;
  nodeId: string;
}

export const ImpactAnalysisTab: React.FC<ImpactAnalysisTabProps> = ({ nodeType, nodeId }) => {
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [sidebarMode, setSidebarMode] = useState<'explanation' | 'assistant'>('explanation');
  const [highlightedNodeIds, setHighlightedNodeIds] = useState<string[]>([]);

  const toggleSidebar = () => setSidebarOpen(!sidebarOpen);

  return (
    <div className="impact-analysis-container">
      <div className="impact-analysis-main">
        {/* Floating Controls Sidebar - Lineage Style */}
        <div className="impact-controls-overlay">
          <Tooltip title="Graph View" placement="right">
            <IconButton 
              size="small" 
              className={`impact-control-btn active`}
              sx={{ color: '#6366f1' }}
            >
              <GraphIcon />
            </IconButton>
          </Tooltip>
          
          <Tooltip title="Explanation" placement="right">
            <IconButton 
              size="small" 
              className={`impact-control-btn ${sidebarOpen && sidebarMode === 'explanation' ? 'active' : ''}`}
              onClick={() => {
                setSidebarMode('explanation');
                setSidebarOpen(true);
              }}
              sx={{ color: sidebarOpen && sidebarMode === 'explanation' ? '#fff' : '#6b7280' }}
            >
              <TextIcon />
            </IconButton>
          </Tooltip>

          <Tooltip title="AI Assistant" placement="right">
            <IconButton 
              size="small" 
              className={`impact-control-btn ${sidebarOpen && sidebarMode === 'assistant' ? 'active' : ''}`}
              onClick={() => {
                setSidebarMode('assistant');
                setSidebarOpen(true);
              }}
              sx={{ color: sidebarOpen && sidebarMode === 'assistant' ? '#fff' : '#6b7280' }}
            >
              <ChatIcon />
            </IconButton>
          </Tooltip>

          <Box sx={{ flex: 1 }} />

          <Tooltip title={sidebarOpen ? "Hide Sidebar" : "Show Sidebar"} placement="right">
            <IconButton 
              size="small" 
              onClick={toggleSidebar}
              className="impact-control-btn"
            >
              <SidebarIcon sx={{ transform: sidebarOpen ? 'none' : 'rotate(180deg)' }} />
            </IconButton>
          </Tooltip>
        </div>

        {/* Main Graph Area */}
        <div className="impact-analysis-graph-container">
          <ImpactGraph 
            nodeType={nodeType} 
            nodeId={nodeId} 
            highlightedNodeIds={highlightedNodeIds}
          />
        </div>

        {/* Dynamic Sidebar */}
        <div className={`impact-analysis-sidebar ${sidebarOpen ? '' : 'collapsed'}`}>
          <div className="impact-analysis-sidebar-header">
            <Typography variant="subtitle1" fontWeight="600">
              {sidebarMode === 'explanation' ? 'Impact Explanation' : 'AI Assistant'}
            </Typography>
            <IconButton size="small" onClick={toggleSidebar}>
              <CloseIcon fontSize="small" />
            </IconButton>
          </div>
          <div className="impact-analysis-sidebar-content">
            {sidebarMode === 'explanation' ? (
              <ImpactExplanation nodeType={nodeType} nodeId={nodeId} />
            ) : (
              <ImpactQA 
                nodeType={nodeType} 
                nodeId={nodeId} 
                onHighlightNodes={setHighlightedNodeIds}
              />
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
