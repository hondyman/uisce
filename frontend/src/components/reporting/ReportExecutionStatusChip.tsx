/**
 * ReportExecutionStatusChip
 *
 * Renders an MUI Chip with the correct icon, label, and color for each
 * ExecutionStatus value.
 */

import React from 'react';
import { Chip, keyframes } from '@mui/material';
import {
  Clock,
  RefreshCw,
  CheckCircle,
  XCircle,
  MinusCircle,
  AlertCircle,
} from 'lucide-react';
import type { ExecutionStatus } from '../../api/reportExecutionApi';

const spin = keyframes`
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
`;

interface ReportExecutionStatusChipProps {
  status: ExecutionStatus;
  sx?: object;
}

const STATUS_CONFIG: Record<
  ExecutionStatus,
  { label: string; color: string; borderColor: string; bg: string; icon: React.ReactNode }
> = {
  pending: {
    label: 'Pending',
    color: '#60A5FA',
    borderColor: 'rgba(96, 165, 250, 0.4)',
    bg: 'rgba(96, 165, 250, 0.12)',
    icon: <Clock size={12} />,
  },
  running: {
    label: 'Running',
    color: '#60A5FA',
    borderColor: 'rgba(96, 165, 250, 0.4)',
    bg: 'rgba(96, 165, 250, 0.12)',
    icon: (
      <RefreshCw
        size={12}
        style={{
          animation: `${spin.name} 1s linear infinite`,
        }}
      />
    ),
  },
  completed: {
    label: 'Completed',
    color: '#34D399',
    borderColor: 'rgba(52, 211, 153, 0.4)',
    bg: 'rgba(52, 211, 153, 0.12)',
    icon: <CheckCircle size={12} />,
  },
  failed: {
    label: 'Failed',
    color: '#F87171',
    borderColor: 'rgba(248, 113, 113, 0.4)',
    bg: 'rgba(248, 113, 113, 0.12)',
    icon: <XCircle size={12} />,
  },
  cancelled: {
    label: 'Cancelled',
    color: '#94A3B8',
    borderColor: 'rgba(148, 163, 184, 0.4)',
    bg: 'rgba(148, 163, 184, 0.12)',
    icon: <MinusCircle size={12} />,
  },
  error: {
    label: 'Error',
    color: '#F87171',
    borderColor: 'rgba(248, 113, 113, 0.4)',
    bg: 'rgba(248, 113, 113, 0.12)',
    icon: <AlertCircle size={12} />,
  },
};

export const ReportExecutionStatusChip: React.FC<ReportExecutionStatusChipProps> = ({
  status,
  sx,
}) => {
  const config = STATUS_CONFIG[status] ?? STATUS_CONFIG.error;

  return (
    <Chip
      size="small"
      label={config.label}
      icon={<>{config.icon}</>}
      sx={{
        color: config.color,
        border: `1px solid ${config.borderColor}`,
        bgcolor: config.bg,
        fontWeight: 700,
        fontSize: '0.68rem',
        height: 22,
        '& .MuiChip-icon': {
          color: config.color,
        },
        ...sx,
      }}
    />
  );
};

export default ReportExecutionStatusChip;
