import React, { memo } from 'react';
import { Handle, Position, NodeProps } from 'reactflow';
import { Box, Typography, Chip, Tooltip, useTheme } from '@mui/material';
import {
  Database,
  ArrowRightLeft,
  ShieldCheck,
  Share2,
  FileText,
  Filter,
  Network,
  Award,
  GitBranch,
  Layers,
  Zap,
  CheckCircle2,
  AlertCircle,
  Clock,
  Loader2,
  Globe,
  Code2,
  Workflow,
  Edit3,
  PlayCircle,
  PlusCircle,
  Trash2,
  LineChart,
} from 'lucide-react';
import { PipelineNodeData } from '../../types/pipeline';

const getIcon = (iconName?: string, category?: string, subType?: string) => {
  const props = { size: 18, style: { marginRight: 6, flexShrink: 0 } };
  
  if (subType === 'api_caller') return <Code2 {...props} />;
  if (subType === 'workflow_caller') return <Workflow {...props} />;
  if (subType === 'bo_crud') return <Edit3 {...props} />;
  if (subType === 'bloomberg_field_mapper') return <LineChart {...props} />;

  switch (iconName) {
    case 'Database': return <Database {...props} />;
    case 'ArrowRightLeft': return <ArrowRightLeft {...props} />;
    case 'ShieldCheck': return <ShieldCheck {...props} />;
    case 'Share2': return <Share2 {...props} />;
    case 'FileText': return <FileText {...props} />;
    case 'Filter': return <Filter {...props} />;
    case 'Network': return <Network {...props} />;
    case 'Award': return <Award {...props} />;
    case 'GitBranch': return <GitBranch {...props} />;
    case 'Layers': return <Layers {...props} />;
    case 'Globe': return <Globe {...props} />;
    case 'Code2': return <Code2 {...props} />;
    case 'Workflow': return <Workflow {...props} />;
    case 'Edit3': return <Edit3 {...props} />;
    case 'LineChart': return <LineChart {...props} />;
    default:
      if (category === 'source') return <FileText {...props} />;
      if (category === 'transform') return <ArrowRightLeft {...props} />;
      if (category === 'validator') return <ShieldCheck {...props} />;
      if (category === 'loader') return <Database {...props} />;
      if (category === 'graph_synthesizer') return <Network {...props} />;
      return <Zap {...props} />;
  }
};

const getCategoryStyles = (category: string, subType: string | undefined, isDark: boolean) => {
  if (subType === 'api_caller') {
    return {
      border: '#0ea5e9',
      bg: isDark ? 'rgba(14, 165, 233, 0.15)' : 'rgba(14, 165, 233, 0.08)',
      chip: '#0284c7',
      text: isDark ? '#7dd3fc' : '#0369a1',
    };
  }
  if (subType === 'workflow_caller') {
    return {
      border: '#ec4899',
      bg: isDark ? 'rgba(236, 72, 153, 0.15)' : 'rgba(236, 72, 153, 0.08)',
      chip: '#db2777',
      text: isDark ? '#f472b6' : '#be185d',
    };
  }
  if (subType === 'bo_crud') {
    return {
      border: '#f97316',
      bg: isDark ? 'rgba(249, 115, 22, 0.15)' : 'rgba(249, 115, 22, 0.08)',
      chip: '#ea580c',
      text: isDark ? '#fdba74' : '#c2410c',
    };
  }
  if (subType === 'bloomberg_field_mapper') {
    return {
      border: '#ff6d00',
      bg: isDark ? 'rgba(255, 109, 0, 0.15)' : 'rgba(255, 109, 0, 0.08)',
      chip: '#e65100',
      text: isDark ? '#ffab40' : '#d84315',
    };
  }

  switch (category) {
    case 'source':
      return {
        border: '#3b82f6',
        bg: isDark ? 'rgba(59, 130, 246, 0.15)' : 'rgba(59, 130, 246, 0.08)',
        chip: '#2563eb',
        text: isDark ? '#93c5fd' : '#1d4ed8',
      };
    case 'transform':
      return {
        border: '#8b5cf6',
        bg: isDark ? 'rgba(139, 92, 246, 0.15)' : 'rgba(139, 92, 246, 0.08)',
        chip: '#7c3aed',
        text: isDark ? '#c4b5fd' : '#6d28d9',
      };
    case 'validator':
      return {
        border: '#f59e0b',
        bg: isDark ? 'rgba(245, 158, 11, 0.15)' : 'rgba(245, 158, 11, 0.08)',
        chip: '#d97706',
        text: isDark ? '#fcd34d' : '#b45309',
      };
    case 'loader':
      return {
        border: '#10b981',
        bg: isDark ? 'rgba(16, 185, 129, 0.15)' : 'rgba(16, 185, 129, 0.08)',
        chip: '#059669',
        text: isDark ? '#6ee7b7' : '#047857',
      };
    case 'graph_synthesizer':
      return {
        border: '#06b6d4',
        bg: isDark ? 'rgba(6, 182, 212, 0.15)' : 'rgba(6, 182, 212, 0.08)',
        chip: '#0891b2',
        text: isDark ? '#67e8f9' : '#0e7490',
      };
    default:
      return {
        border: '#64748b',
        bg: isDark ? 'rgba(100, 116, 139, 0.15)' : 'rgba(100, 116, 139, 0.08)',
        chip: '#475569',
        text: isDark ? '#cbd5e1' : '#334155',
      };
  }
};

export const PipelineTileNode: React.FC<NodeProps<PipelineNodeData>> = memo(({ data, selected }) => {
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';
  const style = getCategoryStyles(data.category, data.subType, isDark);
  const metrics = data.metrics;

  const isRunning = metrics?.status === 'running';
  const isCompleted = metrics?.status === 'completed';
  const isFailed = metrics?.status === 'failed';

  return (
    <Box
      sx={{
        width: 290,
        backgroundColor: isDark ? theme.palette.background.paper : '#ffffff',
        borderRadius: '10px',
        border: selected ? `2px solid ${style.border}` : `1px solid ${theme.palette.divider}`,
        boxShadow: selected
          ? `0 0 0 3px rgba(59, 130, 246, 0.3), 0 4px 12px rgba(0,0,0,0.2)`
          : isDark
          ? '0 2px 8px rgba(0,0,0,0.3)'
          : '0 2px 6px rgba(0,0,0,0.04)',
        overflow: 'hidden',
        transition: 'all 0.2s ease-in-out',
        position: 'relative',
        '&:hover': {
          boxShadow: isDark ? '0 6px 20px rgba(0,0,0,0.4)' : '0 6px 16px rgba(0,0,0,0.08)',
          borderColor: style.border,
        },
      }}
    >
      {/* Target input handle (Left) */}
      {data.category !== 'source' && (
        <Handle
          type="target"
          position={Position.Left}
          style={{
            background: style.border,
            width: 10,
            height: 10,
            border: `2px solid ${isDark ? theme.palette.background.paper : '#ffffff'}`,
          }}
        />
      )}

      {/* Header bar */}
      <Box
        sx={{
          backgroundColor: style.bg,
          borderBottom: `1px solid ${theme.palette.divider}`,
          px: 1.5,
          py: 1,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', color: style.text, overflow: 'hidden', mr: 1 }}>
          {getIcon(data.icon, data.category, data.subType)}
          <Typography variant="subtitle2" sx={{ fontWeight: 700, fontSize: '0.85rem' }} noWrap>
            {data.label}
          </Typography>
        </Box>
        {data.badge && (
          <Chip
            label={data.badge}
            size="small"
            sx={{
              height: 20,
              fontSize: '0.65rem',
              fontWeight: 700,
              backgroundColor: style.chip,
              color: '#ffffff',
              flexShrink: 0,
            }}
          />
        )}
      </Box>

      {/* Body content */}
      <Box sx={{ p: 1.5 }}>
        {data.description && (
          <Typography
            variant="body2"
            sx={{
              fontSize: '0.75rem',
              color: theme.palette.text.secondary,
              mb: 1,
              display: '-webkit-box',
              WebkitLineClamp: 2,
              WebkitBoxOrient: 'vertical',
              overflow: 'hidden',
              minHeight: 32,
            }}
          >
            {data.description}
          </Typography>
        )}

        {/* Live execution telemetry pills */}
        {metrics && (
          <Box
            sx={{
              mt: 1,
              pt: 1,
              borderTop: `1px dashed ${theme.palette.divider}`,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              fontSize: '0.7rem',
            }}
          >
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
              {isRunning && <Loader2 size={12} className="animate-spin text-blue-500" />}
              {isCompleted && <CheckCircle2 size={12} color="#10b981" />}
              {isFailed && <AlertCircle size={12} color="#ef4444" />}
              {!isRunning && !isCompleted && !isFailed && <Clock size={12} color={isDark ? '#94a3b8' : '#64748b'} />}
              <Typography variant="caption" sx={{ fontWeight: 600, textTransform: 'capitalize', color: theme.palette.text.primary }}>
                {metrics.status || 'Ready'}
              </Typography>
            </Box>

            {metrics.recordsOut !== undefined && (
              <Tooltip title="Output Records / Throughput">
                <Chip
                  label={`${metrics.recordsOut.toLocaleString()} rows ${metrics.rowsPerSec ? `(${Math.round(metrics.rowsPerSec)} r/s)` : ''}`}
                  size="small"
                  variant="outlined"
                  sx={{ height: 18, fontSize: '0.65rem', fontWeight: 600 }}
                />
              </Tooltip>
            )}
          </Box>
        )}
      </Box>

      {/* Source output handle (Right) */}
      {data.category !== 'loader' && data.category !== 'sink' && (
        <Handle
          type="source"
          position={Position.Right}
          style={{
            background: style.border,
            width: 10,
            height: 10,
            border: `2px solid ${isDark ? theme.palette.background.paper : '#ffffff'}`,
          }}
        />
      )}
    </Box>
  );
});

PipelineTileNode.displayName = 'PipelineTileNode';
