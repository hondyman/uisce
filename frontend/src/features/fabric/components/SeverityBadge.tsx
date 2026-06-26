// React default import removed — using automatic JSX runtime
import { Chip, Tooltip } from '@mui/material';

interface SeverityBadgeProps {
  severity: 'breaking' | 'medium' | 'low';
  count?: number;
}

const SeverityBadge: React.FC<SeverityBadgeProps> = ({ severity, count }) => {
  if (!count || count === 0) return null;

  const color = {
    breaking: 'error',
    medium: 'warning',
    low: 'success',
  }[severity];

  const icon = {
    breaking: '🚨',
    medium: '⚠️',
    low: '✅',
  }[severity];

  return (
    <Tooltip title={`${count} ${severity} change(s)`}>
      <Chip
  icon={<span className="severity-icon">{icon}</span>}
        label={count}
        color={color as 'error' | 'warning' | 'success'}
        size="small"
        sx={{ mr: 1 }}
      />
    </Tooltip>
  );
};

export default SeverityBadge;