import React, { memo } from 'react';
import { Handle, NodeProps, Position } from 'reactflow';
import { Box, Chip, Stack, Tooltip, Typography, alpha, useTheme } from '@mui/material';
import InsertDriveFileIcon from '@mui/icons-material/InsertDriveFile';
import BusinessIcon from '@mui/icons-material/Business';
import FactCheckIcon from '@mui/icons-material/FactCheck';
import GavelIcon from '@mui/icons-material/Gavel';
import SwapHorizIcon from '@mui/icons-material/SwapHoriz';
import TableChartIcon from '@mui/icons-material/TableChart';
import FileDownloadIcon from '@mui/icons-material/FileDownload';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import type { NodeKind, NodeStats } from './api';

export const NODE_META: Record<NodeKind, { icon: React.ReactElement; category: 'source' | 'step' | 'destination' }> = {
  file_source: { icon: <InsertDriveFileIcon fontSize="small" />, category: 'source' },
  bo_source: { icon: <BusinessIcon fontSize="small" />, category: 'source' },
  validate: { icon: <FactCheckIcon fontSize="small" />, category: 'step' },
  rule_check: { icon: <GavelIcon fontSize="small" />, category: 'step' },
  map: { icon: <SwapHorizIcon fontSize="small" />, category: 'step' },
  bo_sink: { icon: <BusinessIcon fontSize="small" />, category: 'destination' },
  staging_sink: { icon: <TableChartIcon fontSize="small" />, category: 'destination' },
  file_sink: { icon: <FileDownloadIcon fontSize="small" />, category: 'destination' },
};

export interface PipelineNodeData {
  kind: NodeKind;
  title: string;
  summary: string; // one line: what this step is configured to do
  issues: string[];
  stats?: NodeStats; // from the last preview
}

function PipelineNodeImpl({ data, selected }: NodeProps<PipelineNodeData>) {
  const theme = useTheme();
  const meta = NODE_META[data.kind];
  const color = meta.category === 'source' ? theme.palette.info.main
    : meta.category === 'destination' ? theme.palette.success.main : theme.palette.secondary.main;
  const hasIssues = data.issues.length > 0;
  return (
    <Box
      sx={{
        width: 230, borderRadius: 2, bgcolor: 'background.paper',
        border: `2px solid ${hasIssues ? theme.palette.error.main : selected ? color : alpha(color, 0.35)}`,
        boxShadow: selected ? `0 0 0 3px ${alpha(color, 0.2)}` : 1,
      }}
    >
      {meta.category !== 'source' && <Handle type="target" position={Position.Left} />}
      <Stack direction="row" spacing={1} alignItems="center" sx={{ px: 1.25, py: 0.75, bgcolor: alpha(color, 0.1), borderRadius: '6px 6px 0 0' }}>
        <Box sx={{ color, display: 'flex' }}>{meta.icon}</Box>
        <Typography variant="subtitle2" noWrap sx={{ flex: 1, fontWeight: 700 }}>{data.title}</Typography>
        {hasIssues && (
          <Tooltip title={data.issues.join(' · ')}>
            <ErrorOutlineIcon fontSize="small" color="error" />
          </Tooltip>
        )}
      </Stack>
      <Box sx={{ px: 1.25, py: 0.75 }}>
        <Typography variant="caption" color="text.secondary" sx={{ display: 'block' }} noWrap title={data.summary}>
          {data.summary || 'Click to configure'}
        </Typography>
        {data.stats && (
          <Stack direction="row" spacing={0.5} sx={{ mt: 0.5 }}>
            <Chip size="small" label={`in ${data.stats.In}`} />
            <Chip size="small" color="success" variant="outlined" label={`out ${data.stats.Out}`} />
            {data.stats.Errors > 0 && <Chip size="small" color="error" label={`${data.stats.Errors} rejected`} />}
            {data.stats.Warnings > 0 && <Chip size="small" color="warning" label={`${data.stats.Warnings} warned`} />}
          </Stack>
        )}
      </Box>
      {meta.category !== 'destination' && <Handle type="source" position={Position.Right} />}
    </Box>
  );
}

export const PipelineNode = memo(PipelineNodeImpl);
