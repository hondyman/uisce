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
  Typography,
  Tooltip,
} from '@mui/material';
import { diffRuns, DiffResult } from '../../../utils/diffUtils';
import { format } from 'date-fns';

interface DiffRunsTableProps {
  dataA: any[];
  dataB: any[];
}

const getRowStyle = (type: DiffResult['type']) => {
  switch (type) {
    case 'added':
      return { backgroundColor: 'rgba(74, 222, 128, 0.1)' }; // green
    case 'removed':
      return { backgroundColor: 'rgba(248, 113, 113, 0.1)' }; // red
    case 'changed':
      return { backgroundColor: 'rgba(251, 191, 36, 0.1)' }; // yellow
    default:
      return {};
  }
};

const getDecisionChip = (decision?: string) => {
  if (!decision) return <Typography variant="caption" color="text.secondary">N/A</Typography>;
  return <Chip label={decision} color={decision === 'block' ? 'error' : 'success'} size="small" />;
};

const DiffRunsTable: FC<DiffRunsTableProps> = ({ dataA, dataB }) => {
  const diffs = diffRuns(dataA, dataB);

  return (
    <TableContainer component={Paper}>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>Run ID</TableCell>
            <TableCell>Timestamp</TableCell>
            <TableCell>Decision A</TableCell>
            <TableCell>Decision B</TableCell>
            <TableCell>Violations Added</TableCell>
            <TableCell>Violations Removed</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {diffs.map((d) => {
            const timestamp = d.runA?.timestamp || d.runB?.timestamp;
            return (
              <TableRow key={d.run_id} sx={getRowStyle(d.type)}>
                <TableCell sx={{ fontFamily: 'monospace' }}>
                  <Tooltip title={`Change Type: ${d.type}`}>
                    <span>{d.run_id.substring(0, 8)}</span>
                  </Tooltip>
                </TableCell>
                <TableCell>
                  {timestamp ? format(new Date(timestamp), 'yyyy-MM-dd HH:mm') : ''}
                </TableCell>
                <TableCell>{getDecisionChip(d.runA?.decision_a)}</TableCell>
                <TableCell>{getDecisionChip(d.runB?.decision_a)}</TableCell>
                <TableCell>
                  {d.violationDelta.added.length > 0 ? (
                    d.violationDelta.added.map((code) => (
                      <Chip key={code} label={code} size="small" color="error" sx={{ mr: 0.5 }} />
                    ))
                  ) : (
                    <Typography variant="caption" color="text.secondary">None</Typography>
                  )}
                </TableCell>
                <TableCell>
                  {d.violationDelta.removed.length > 0 ? (
                    d.violationDelta.removed.map((code) => (
                      <Chip key={code} label={code} size="small" color="success" sx={{ mr: 0.5 }} />
                    ))
                  ) : (
                    <Typography variant="caption" color="text.secondary">None</Typography>
                  )}
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </TableContainer>
  );
};

export default DiffRunsTable;