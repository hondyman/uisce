import { Chip, ChipProps } from '@mui/material';

const COLORS: Record<string, string> = {
  PASS: '#2ECC71',
  FAIL: '#E74C3C',
  WARN: '#F1C40F',
  INFO: '#3498DB',
  PENDING: '#95A5A6',
  RUNNING: '#9B59B6',
  SUCCESS: '#2ECC71',
  FAILED: '#E74C3C',
  COMPLETED: '#2ECC71',
  QUEUED: '#95A5A6',
};

export interface StatusBadgeProps extends Omit<ChipProps, 'label'> {
  status: string;
}

export function StatusBadge({ status, ...rest }: StatusBadgeProps) {
  const color = COLORS[status] ?? '#95A5A6';
  
  return (
    <Chip
      label={status}
      size="small"
      sx={{
        backgroundColor: color,
        color: 'white',
        fontWeight: 600,
        textTransform: 'uppercase',
        fontSize: '0.7rem',
      }}
      {...rest}
    />
  );
}
