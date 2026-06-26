import type { FC } from 'react';
import {
  Box,
  Paper,
  Typography,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip
} from '@mui/material';

interface ExitTrackingTableProps {
  selectedFunds?: string[];
}

export const ExitTrackingTable: FC<ExitTrackingTableProps> = () => {
  // Mock exit tracking data
  const exitData = [
    {
      company: 'TechStart Inc.',
      exitType: 'IPO',
      exitDate: '2023-06-15',
      exitValue: 450,
      costBasis: 50,
      multiple: 9.0,
      status: 'completed'
    },
    {
      company: 'DataFlow Corp',
      exitType: 'Acquisition',
      exitDate: '2023-08-22',
      exitValue: 320,
      costBasis: 80,
      multiple: 4.0,
      status: 'completed'
    },
    {
      company: 'CloudTech Ltd',
      exitType: 'IPO',
      exitDate: '2023-11-10',
      exitValue: 280,
      costBasis: 40,
      multiple: 7.0,
      status: 'pending'
    },
    {
      company: 'BioHealth Inc',
      exitType: 'Acquisition',
      exitDate: '2024-02-28',
      exitValue: 180,
      costBasis: 60,
      multiple: 3.0,
      status: 'pending'
    }
  ];

  const getStatusColor = (status: string): 'success' | 'warning' | 'default' => {
    switch (status) {
      case 'completed': return 'success';
      case 'pending': return 'warning';
      default: return 'default';
    }
  };

  const totalRealized = exitData
    .filter(exit => exit.status === 'completed')
    .reduce((sum, exit) => sum + exit.exitValue, 0);

  const totalPending = exitData
    .filter(exit => exit.status === 'pending')
    .reduce((sum, exit) => sum + exit.exitValue, 0);

  return (
    <Paper sx={{ p: 2 }}>
      <Typography variant="h6" gutterBottom>
        Exit Tracking
      </Typography>

      {/* Summary Cards */}
      <Box sx={{ display: 'flex', gap: 2, mb: 2 }}>
        <Box>
          <Typography variant="body2" color="text.secondary">Realized Value</Typography>
          <Typography variant="h6" color="success.main">
            ${totalRealized}M
          </Typography>
        </Box>
        <Box>
          <Typography variant="body2" color="text.secondary">Pending Value</Typography>
          <Typography variant="h6" color="warning.main">
            ${totalPending}M
          </Typography>
        </Box>
      </Box>

      <TableContainer>
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>Company</TableCell>
              <TableCell>Exit Type</TableCell>
              <TableCell>Exit Date</TableCell>
              <TableCell align="right">Exit Value ($M)</TableCell>
              <TableCell align="right">Cost Basis ($M)</TableCell>
              <TableCell align="right">Multiple</TableCell>
              <TableCell>Status</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {exitData.map((exit, index) => (
              <TableRow key={index}>
                <TableCell component="th" scope="row">
                  {exit.company}
                </TableCell>
                <TableCell>{exit.exitType}</TableCell>
                <TableCell>{new Date(exit.exitDate).toLocaleDateString()}</TableCell>
                <TableCell align="right">${exit.exitValue}</TableCell>
                <TableCell align="right">${exit.costBasis}</TableCell>
                <TableCell align="right">{exit.multiple}x</TableCell>
                <TableCell>
                    <Chip
                    size="small"
                    label={exit.status}
                    color={getStatusColor(exit.status)}
                  />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </Paper>
  );
};
