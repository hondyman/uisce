import type { FC } from 'react';
import {
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Chip,
  Typography
} from '@mui/material';
import { format } from 'date-fns';

const DecisionDiffTable: FC<{ diffs: any[] }> = ({ diffs }) => {
  const getDecisionChip = (decision: string) => {
    if (decision === 'block') {
      return <Chip label="Block" color="error" size="small" />;
    }
    return <Chip label="Allow" color="success" size="small" />;
  };

  return (
    <>
      <Typography variant="h6" gutterBottom sx={{ p: 2 }}>
        Runs with Changed Decisions
      </Typography>
      <TableContainer component={Paper} variant="outlined">
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>Run ID</TableCell>
              <TableCell>Timestamp</TableCell>
              <TableCell>Version A Decision</TableCell>
              <TableCell>Version B Decision</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {diffs.length === 0 && (
              <TableRow>
                <TableCell colSpan={4} align="center">
                  No decision changes found for this period.
                </TableCell>
              </TableRow>
            )}
            {diffs.map((d) => (
              <TableRow key={d.run_id}>
                <TableCell sx={{ fontFamily: 'monospace' }}>{d.change_id.substring(0, 8)}</TableCell>
                <TableCell>{format(new Date(d.timestamp), 'yyyy-MM-dd HH:mm')}</TableCell>
                <TableCell>{getDecisionChip(d.decision_a)}</TableCell>
                <TableCell>{getDecisionChip(d.decision_b)}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </>
  );
};

export default DecisionDiffTable;