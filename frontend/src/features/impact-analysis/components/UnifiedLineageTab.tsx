import React, { useState, useEffect } from 'react';
import { Box, IconButton, Tooltip, Typography, ToggleButtonGroup, ToggleButton, Chip } from '@mui/material';
import { NodeType } from '../types';
import { ImpactGraph } from './ImpactGraph';
import { ImpactExplanation } from './ImpactExplanation';
import { ImpactQA } from './ImpactQA';
import GraphIcon from '@mui/icons-material/Hub';
import TextIcon from '@mui/icons-material/Description';
import ChatIcon from '@mui/icons-material/Chat';
import SidebarIcon from '@mui/icons-material/ViewSidebar';
import CloseIcon from '@mui/icons-material/Close';
import ArrowUpwardIcon from '@mui/icons-material/ArrowUpward';
import ArrowDownwardIcon from '@mui/icons-material/ArrowDownward';
import SwapVertIcon from '@mui/icons-material/SwapVert';
import './ImpactAnalysis.css';

type DirectionMode = 'upstream' | 'downstream' | 'both';

interface UnifiedLineageTabProps {
  nodeType: NodeType;
  nodeId: string;
  initialDirection?: DirectionMode;
}

export const UnifiedLineageTab: React.FC<UnifiedLineageTabProps> = ({ 
  nodeType, 
  nodeId,
  initialDirection = 'both' 
}) => {
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [sidebarMode, setSidebarMode] = useState<'explanation' | 'assistant'>('explanation');
  const [highlightedNodeIds, setHighlightedNodeIds] = useState<string[]>([]);
  const [directionMode, setDirectionMode] = useState<DirectionMode>(initialDirection);
  const [graphStats, setGraphStats] = useState<{
    upstreamCount: number;
    downstreamCount: number;
    totalCount: number;
  }>({ upstreamCount: 0, downstreamCount: 0, totalCount: 0 });

  const toggleSidebar = () => setSidebarOpen(!sidebarOpen);

  const handleDirectionChange = (_event: React.MouseEvent<HTMLElement>, newDirection: DirectionMode | null) => {
    if (newDirection !== null) {
      setDirectionMode(newDirection);
    }
  };

  const handleGraphStats = (stats: { upstreamCount: number; downstreamCount: number; totalCount: number }) => {
    setGraphStats(stats);
  };

  return (
    <div className="impact-analysis-container">
      <div className="impact-analysis-main">
        {/* Floating Controls Sidebar */}
        <div className="impact-controls-overlay">
          <Tooltip title="Graph View" placement="right">
            <IconButton 
              size="small" 
              className="impact-control-btn active"
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

        {/* Main Graph Area with Direction Controls */}
        <div className="impact-analysis-graph-container">
          {/* Direction Toggle */}
          <Box 
            sx={{ 
              position: 'absolute', 
              top: 16, 
              left: 16, 
              zIndex: 1000,
              background: 'rgba(255, 255, 255, 0.95)',
              borderRadius: 2,
              padding: 1.5,
              boxShadow: '0 2px 8px rgba(0,0,0,0.1)'
            }}
          >
            <Typography variant="caption" sx={{ mb: 1, display: 'block', fontWeight: 600, color: '#374151' }}>
              Direction
            </Typography>
            <ToggleButtonGroup
              value={directionMode}
              exclusive
              onChange={handleDirectionChange}
              size="small"
              sx={{ 
                '& .MuiToggleButton-root': {
                  px: 1.5,
                  py: 0.5,
                  fontSize: '0.75rem',
                  textTransform: 'none',
                  border: '1px solid #e5e7eb',
                  '&.Mui-selected': {
                    backgroundColor: '#6366f1',
                    color: 'white',
                    '&:hover': {
                      backgroundColor: '#5558e3'
                    }
                  }
                }
              }}
            >
              <ToggleButton value="upstream">
                <ArrowUpwardIcon sx={{ fontSize: 16, mr: 0.5 }} />
                Lineage
              </ToggleButton>
              <ToggleButton value="both">
                <SwapVertIcon sx={{ fontSize: 16, mr: 0.5 }} />
                Both
              </ToggleButton>
              <ToggleButton value="downstream">
                <ArrowDownwardIcon sx={{ fontSize: 16, mr: 0.5 }} />
                Impact
              </ToggleButton>
            </ToggleButtonGroup>

            {/* Stats Display */}
            {graphStats.totalCount > 0 && (
              <Box sx={{ mt: 1.5, display: 'flex', gap: 0.5, flexWrap: 'wrap' }}>
                {directionMode !== 'downstream' && graphStats.upstreamCount > 0 && (
                  <Chip 
                    label={`${graphStats.upstreamCount} upstream`}
                    size="small"
                    icon={<ArrowUpwardIcon />}
                    sx={{ 
                      fontSize: '0.7rem', 
                      height: 20,
                      backgroundColor: '#dbeafe',
                      color: '#1e40af',
                      '& .MuiChip-icon': { fontSize: 14 }
                    }}
                  />
                )}
                {directionMode !== 'upstream' && graphStats.downstreamCount > 0 && (
                  <Chip 
                    label={`${graphStats.downstreamCount} downstream`}
                    size="small"
                    icon={<ArrowDownwardIcon />}
                    sx={{ 
                      fontSize: '0.7rem', 
                      height: 20,
                      backgroundColor: '#fef3c7',
                      color: '#92400e',
                      '& .MuiChip-icon': { fontSize: 14 }
                    }}
                  />
                )}
              </Box>
            )}
          </Box>

          <ImpactGraph 
            nodeType={nodeType} 
            nodeId={nodeId} 
            highlightedNodeIds={highlightedNodeIds}
            directionMode={directionMode}
            onStatsUpdate={handleGraphStats}
            useLineageAPI={true}
          />
        </div>

        {/* Dynamic Sidebar */}
        <div className={`impact-analysis-sidebar ${sidebarOpen ? '' : 'collapsed'}`}>
          <div className="impact-analysis-sidebar-header">
            <Typography variant="subtitle1" fontWeight="600">
              {sidebarMode === 'explanation' 
                ? directionMode === 'upstream' ? 'Lineage Explanation' 
                  : directionMode === 'downstream' ? 'Impact Explanation'
                  : 'Lineage & Impact Explanation'
                : 'AI Assistant'}
            </Typography>
            <IconButton size="small" onClick={toggleSidebar}>
              <CloseIcon fontSize="small" />
            </IconButton>
          </div>
          <div className="impact-analysis-sidebar-content">
            {sidebarMode === 'explanation' ? (
              <ImpactExplanation 
                nodeType={nodeType} 
                nodeId={nodeId} 
                directionMode={directionMode}
              />
            ) : (
              <ImpactQA 
                nodeType={nodeType} 
                nodeId={nodeId} 
                directionMode={directionMode}
                onHighlightNodes={setHighlightedNodeIds}
              />
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
