import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  IconButton,
  Divider,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  Collapse,
  Tooltip,
  Chip,
  Stack,
  LinearProgress,
} from '@mui/material';
import {
  PlayArrow as PlayIcon,
  Pause as PauseIcon,
  Stop as StopIcon,
  SkipNext as StepOverIcon,
  ArrowDownward as StepIntoIcon,
  ArrowUpward as StepOutIcon,
  RestartAlt as RestartIcon,
  BugReport as BugIcon,
  CheckCircle as PassIcon,
  Error as FailIcon,
  ExpandMore as ExpandIcon,
  ExpandLess as CollapseIcon,
  DataObject as DataIcon,
  Functions as FunctionIcon,
  Layers as StackIcon,
  Terminal as ConsoleIcon,
} from '@mui/icons-material';
import { DebugStep, TraceResult } from '../api/uisceApi';

interface DebugPanelProps {
  traceResult: TraceResult | null;
  isDebugging: boolean;
  currentStepIndex: number;
  onStep: () => void;
  onContinue: () => void;
  onStop: () => void;
  onRestart: () => void;
}

const DebugPanel: React.FC<DebugPanelProps> = ({
  traceResult,
  isDebugging,
  currentStepIndex,
  onStep,
  onContinue,
  onStop,
  onRestart,
}) => {
  const [activeTab, setActiveTab] = useState<'stack' | 'variables' | 'console'>('stack');
  const [expandedSteps, setExpandedSteps] = useState<Set<number>>(new Set([0]));

  const toggleStep = (index: number) => {
    const newExpanded = new Set(expandedSteps);
    if (newExpanded.has(index)) {
      newExpanded.delete(index);
    } else {
      newExpanded.add(index);
    }
    setExpandedSteps(newExpanded);
  };

  const currentStep = traceResult?.steps[currentStepIndex];

  return (
    <Paper
      elevation={0}
      sx={{
        height: 300,
        display: 'flex',
        flexDirection: 'column',
        bgcolor: '#1e1e1e',
        color: '#cccccc',
        overflow: 'hidden',
        borderTop: '1px solid #3c3c3c',
      }}
    >
      {/* Toolbar */}
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: 0.5,
          px: 1,
          py: 0.5,
          bgcolor: '#252526',
          borderBottom: '1px solid #3c3c3c',
        }}
      >
        <Tooltip title="Continue (F5)">
          <IconButton size="small" onClick={onContinue} disabled={!isDebugging} sx={{ color: '#4fc1ff' }}>
            <PlayIcon fontSize="small" />
          </IconButton>
        </Tooltip>
        <Tooltip title="Step Over (F10)">
          <IconButton size="small" onClick={onStep} disabled={!isDebugging} sx={{ color: '#4fc1ff' }}>
            <StepOverIcon fontSize="small" />
          </IconButton>
        </Tooltip>
        <Tooltip title="Step Into (F11)">
          <IconButton size="small" disabled sx={{ color: '#6a6a6a' }}>
            <StepIntoIcon fontSize="small" />
          </IconButton>
        </Tooltip>
        <Tooltip title="Step Out (Shift+F11)">
          <IconButton size="small" disabled sx={{ color: '#6a6a6a' }}>
            <StepOutIcon fontSize="small" />
          </IconButton>
        </Tooltip>
        <Divider orientation="vertical" flexItem sx={{ bgcolor: '#3c3c3c', mx: 1 }} />
        <Tooltip title="Restart (Ctrl+Shift+F5)">
          <IconButton size="small" onClick={onRestart} sx={{ color: '#89d185' }}>
            <RestartIcon fontSize="small" />
          </IconButton>
        </Tooltip>
        <Tooltip title="Stop (Shift+F5)">
          <IconButton size="small" onClick={onStop} sx={{ color: '#f48771' }}>
            <StopIcon fontSize="small" />
          </IconButton>
        </Tooltip>
        
        <Box sx={{ flexGrow: 1 }} />
        
        {isDebugging && (
          <Chip 
            label="Debugging" 
            size="small" 
            sx={{ bgcolor: '#007acc', color: 'white', fontSize: '0.7rem' }} 
          />
        )}
        {traceResult && (
          <Typography variant="caption" sx={{ color: '#9cdcfe' }}>
            Trade: {traceResult.tradeId}
          </Typography>
        )}
      </Box>

      {/* Progress */}
      {isDebugging && (
        <LinearProgress 
          variant="determinate" 
          value={traceResult ? ((currentStepIndex + 1) / traceResult.steps.length) * 100 : 0} 
          sx={{ height: 2 }}
        />
      )}

      {/* Main Content */}
      <Box sx={{ display: 'flex', flexGrow: 1, overflow: 'hidden' }}>
        {/* Tab Bar */}
        <Box sx={{ width: 40, bgcolor: '#333333', display: 'flex', flexDirection: 'column', alignItems: 'center', py: 1 }}>
          <Tooltip title="Call Stack" placement="right">
            <IconButton 
              size="small" 
              onClick={() => setActiveTab('stack')}
              sx={{ color: activeTab === 'stack' ? '#4fc1ff' : '#858585' }}
            >
              <StackIcon fontSize="small" />
            </IconButton>
          </Tooltip>
          <Tooltip title="Variables" placement="right">
            <IconButton 
              size="small" 
              onClick={() => setActiveTab('variables')}
              sx={{ color: activeTab === 'variables' ? '#4fc1ff' : '#858585' }}
            >
              <DataIcon fontSize="small" />
            </IconButton>
          </Tooltip>
          <Tooltip title="Console" placement="right">
            <IconButton 
              size="small" 
              onClick={() => setActiveTab('console')}
              sx={{ color: activeTab === 'console' ? '#4fc1ff' : '#858585' }}
            >
              <ConsoleIcon fontSize="small" />
            </IconButton>
          </Tooltip>
        </Box>

        {/* Panel Content */}
        <Box sx={{ flexGrow: 1, overflow: 'auto', p: 1 }}>
          {activeTab === 'stack' && (
            <Box>
              <Typography variant="caption" sx={{ color: '#858585', textTransform: 'uppercase', fontSize: '0.65rem', fontWeight: 'bold' }}>
                Call Stack
              </Typography>
              <List dense sx={{ py: 0 }}>
                {traceResult?.steps.map((step, index) => (
                  <React.Fragment key={index}>
                    <ListItem
                      sx={{
                        py: 0.5,
                        px: 1,
                        cursor: 'pointer',
                        bgcolor: index === currentStepIndex ? '#094771' : 'transparent',
                        borderRadius: 1,
                        '&:hover': { bgcolor: index === currentStepIndex ? '#094771' : '#2a2d2e' },
                      }}
                      onClick={() => toggleStep(index)}
                    >
                      <ListItemIcon sx={{ minWidth: 24 }}>
                        {step.status === 'PASS' ? (
                          <PassIcon sx={{ fontSize: 14, color: '#89d185' }} />
                        ) : (
                          <FailIcon sx={{ fontSize: 14, color: '#f48771' }} />
                        )}
                      </ListItemIcon>
                      <ListItemText
                        primary={
                          <Typography variant="body2" sx={{ fontSize: '0.75rem', color: '#cccccc' }}>
                            {step.filterName}
                          </Typography>
                        }
                        secondary={
                          <Typography variant="caption" sx={{ color: '#858585', fontSize: '0.65rem' }}>
                            {step.durationMs}ms
                          </Typography>
                        }
                      />
                      {expandedSteps.has(index) ? <CollapseIcon sx={{ fontSize: 14 }} /> : <ExpandIcon sx={{ fontSize: 14 }} />}
                    </ListItem>
                    <Collapse in={expandedSteps.has(index)}>
                      <Box sx={{ pl: 4, pr: 1, py: 1, bgcolor: '#1e1e1e' }}>
                        {step.errorDetails && (
                          <Typography variant="caption" sx={{ color: '#f48771', display: 'block', mb: 1 }}>
                            ❌ {step.errorDetails}
                          </Typography>
                        )}
                        <Typography variant="caption" sx={{ color: '#6a9955', fontFamily: 'monospace', fontSize: '0.7rem' }}>
                          // Input snapshot at this step
                        </Typography>
                      </Box>
                    </Collapse>
                  </React.Fragment>
                ))}
              </List>
            </Box>
          )}

          {activeTab === 'variables' && (
            <Box>
              <Typography variant="caption" sx={{ color: '#858585', textTransform: 'uppercase', fontSize: '0.65rem', fontWeight: 'bold' }}>
                Variables
              </Typography>
              {currentStep?.inputSnapshot ? (
                <Box sx={{ mt: 1, fontFamily: 'monospace', fontSize: '0.75rem' }}>
                  {Object.entries(currentStep.inputSnapshot).map(([key, value]) => (
                    <Box key={key} sx={{ display: 'flex', gap: 1, py: 0.25 }}>
                      <Typography sx={{ color: '#9cdcfe' }}>{key}:</Typography>
                      <Typography sx={{ color: '#ce9178' }}>{JSON.stringify(value)}</Typography>
                    </Box>
                  ))}
                </Box>
              ) : (
                <Typography variant="caption" sx={{ color: '#858585' }}>
                  No variables at current step
                </Typography>
              )}
            </Box>
          )}

          {activeTab === 'console' && (
            <Box>
              <Typography variant="caption" sx={{ color: '#858585', textTransform: 'uppercase', fontSize: '0.65rem', fontWeight: 'bold' }}>
                Debug Console
              </Typography>
              <Box sx={{ mt: 1, fontFamily: 'monospace', fontSize: '0.75rem' }}>
                {traceResult?.steps.map((step, i) => (
                  <Typography key={i} sx={{ color: step.status === 'PASS' ? '#89d185' : '#f48771' }}>
                    [{step.status}] {step.filterName} - {step.durationMs}ms
                    {step.errorDetails && ` - ${step.errorDetails}`}
                  </Typography>
                ))}
              </Box>
            </Box>
          )}
        </Box>
      </Box>
    </Paper>
  );
};

export default DebugPanel;
