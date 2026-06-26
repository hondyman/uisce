// React import not required directly in this file (JSX runtime handles it)
import {
  Paper,
  Typography,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow
} from '@mui/material';

interface ValueAttributionBridgeProps {
  selectedFunds?: string[];
}

export const ValueAttributionBridge: React.FC<ValueAttributionBridgeProps> = ({ selectedFunds: _selectedFunds = [] }) => {
  // Mock value attribution data
  const attributionData = [
    {
      component: 'Asset Allocation',
      contribution: 2.3,
      weight: 45,
      attribution: 1.04
    },
    {
      component: 'Security Selection',
      contribution: 1.8,
      weight: 35,
      attribution: 0.63
    },
    {
      component: 'Currency',
      contribution: -0.2,
      weight: 10,
      attribution: -0.02
    },
    {
      component: 'Timing',
      contribution: 0.5,
      weight: 10,
      attribution: 0.05
    }
  ];

  const totalAttribution = attributionData.reduce((sum, item) => sum + item.attribution, 0);

  return (
    <Paper sx={{ p: 2 }}>
      <Typography variant="h6" gutterBottom>
        Value Attribution Bridge
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        Breakdown of performance attribution by component
      </Typography>

      <TableContainer>
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>Component</TableCell>
              <TableCell align="right">Weight (%)</TableCell>
              <TableCell align="right">Contribution (%)</TableCell>
              <TableCell align="right">Attribution (%)</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {attributionData.map((row) => (
              <TableRow key={row.component}>
                <TableCell component="th" scope="row">
                  {row.component}
                </TableCell>
                <TableCell align="right">{row.weight}%</TableCell>
                <TableCell align="right">{row.contribution.toFixed(1)}%</TableCell>
                <TableCell align="right">{row.attribution.toFixed(2)}%</TableCell>
              </TableRow>
            ))}
            <TableRow sx={{ '& td': { fontWeight: 'bold' } }}>
              <TableCell>Total</TableCell>
              <TableCell align="right">100%</TableCell>
              <TableCell align="right">
                {attributionData.reduce((sum, item) => sum + item.contribution, 0).toFixed(1)}%
              </TableCell>
              <TableCell align="right">{totalAttribution.toFixed(2)}%</TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </TableContainer>
    </Paper>
  );
};
