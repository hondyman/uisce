import type { FC } from 'react';
import { Box, Button, Tooltip } from '@mui/material';

interface RiskMitigationToolbarProps {
  sql: string;
  setSql: (newSql: string) => void;
}

const RISK_OPS = [
  {
    label: 'Comment out DROP TABLEs',
    // This regex is simple and targets single-line statements.
    // A production version might need a more advanced SQL parser for multi-line statements.
    regex: /DROP\s+TABLE\s+.*?;/gi,
    description: 'Comments out all DROP TABLE statements, preserving them for review.',
  },
  {
    label: 'Comment out DROP COLUMNs',
    regex: /ALTER\s+TABLE\s+.*?\s+DROP\s+COLUMN\s+.*?;/gi,
    description: 'Comments out all DROP COLUMN statements.',
  },
  {
    label: 'Comment out ALTER TYPEs',
    regex: /ALTER\s+TABLE\s+.*?\s+ALTER\s+COLUMN\s+.*?\s+TYPE\s+.*?;/gi,
    description: 'Comments out all data type alterations, which can be breaking changes.',
  },
];

const RiskMitigationToolbar: FC<RiskMitigationToolbarProps> = ({ sql, setSql }) => {
  const applyMitigation = (regex: RegExp) => {
    // This approach comments out matching lines, which is non-destructive.
    const newSql = sql
      .split('\n')
      .map((line) => {
        // Important: Reset regex state for each line since we're using a global regex in a loop.
        regex.lastIndex = 0;
        if (regex.test(line)) {
          return `-- MITIGATED: ${line}`;
        }
        return line;
      })
      .join('\n');
    setSql(newSql);
  };

  return (
    <Box sx={{ display: 'flex', gap: 1, mb: 2, flexWrap: 'wrap' }}>
      {RISK_OPS.map((op) => (
        <Tooltip key={op.label} title={op.description}>
          <Button variant="outlined" size="small" onClick={() => applyMitigation(op.regex)}>
            {op.label}
          </Button>
        </Tooltip>
      ))}
    </Box>
  );
};

export default RiskMitigationToolbar;