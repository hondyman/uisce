import { Chip, ChipProps } from '@mui/material';

const SEVERITY_COLORS: Record<string, string> = {
  HARD: '#C0392B',
  SOFT: '#F39C12',
  INFO: '#2980B9',
};

export interface SeverityBadgeProps extends Omit<ChipProps, 'label'> {
  severity: string;
}

export function SeverityBadge({ severity, ...rest }: SeverityBadgeProps) {
  const color = SEVERITY_COLORS[severity] ?? '#7F8C8D';
  
  return (
    <Chip
      label={severity}
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
