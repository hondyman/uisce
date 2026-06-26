import type { FC } from 'react';
import {
  Box,
  Typography,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  CircularProgress,
  Alert,
  Chip,
  LinearProgress,
} from '@mui/material';

interface Forecast {
  policy_id: string;
  policy_name: string;
  block_probability: number;
  confidence: number;
  top_factors: string[];
}

interface ForecastPanelProps {
  data: Forecast[] | null | undefined;
  loading: boolean;
  error?: any;
}

const ProbabilityBar: FC<{ value: number }> = ({ value }) => {
  const percentage = value * 100;
  const color = percentage > 60 ? 'error' : percentage > 20 ? 'warning' : 'success';

  return (
    <Box sx={{ display: 'flex', alignItems: 'center' }}>
      <Box sx={{ width: '100%', mr: 1 }}>
        <LinearProgress variant="determinate" value={percentage} color={color} />
      </Box>
      <Box sx={{ minWidth: 35 }}>
        <Typography variant="body2" color="text.secondary">{`${Math.round(percentage)}%`}</Typography>
      </Box>
    </Box>
  );
};

const ForecastPanel: FC<ForecastPanelProps> = ({ data, loading, error }) => {
  if (loading) {
    return (
      <Paper sx={{ p: 3, mt: 4, textAlign: 'center' }}>
        <CircularProgress />
        <Typography sx={{ mt: 1 }} color="text.secondary">
          Generating Forecast...
        </Typography>
      </Paper>
    );
  }

  if (error) {
    return (
      <Alert severity="error" sx={{ mt: 3 }}>
        Forecast failed: {error.message}
      </Alert>
    );
  }

  if (!data || data.length === 0) {
    return null; // Don't render anything if there's no data
  }

  return (
    <Paper sx={{ p: 3, mt: 4, border: '1px solid', borderColor: 'divider' }}>
      <Typography variant="h5" gutterBottom>
        Policy Impact Forecast
      </Typography>
      <TableContainer>
        <Table size="small" aria-label="policy forecast table">
          <TableHead>
            <TableRow>
              <TableCell>Policy</TableCell>
              <TableCell>Block Probability</TableCell>
              <TableCell>Confidence</TableCell>
              <TableCell>Top Factors</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {data.map((row) => (
              <TableRow key={row.policy_id}>
                <TableCell component="th" scope="row">
                  {row.policy_name}
                </TableCell>
                <TableCell>
                  <ProbabilityBar value={row.block_probability / 100} />
                </TableCell>
                <TableCell>
                  <Typography variant="body2">{(row.confidence * 100).toFixed(0)}%</Typography>
                </TableCell>
                <TableCell>
                  {row.top_factors.map((f) => (
                    <Chip key={f} label={f} size="small" sx={{ mr: 0.5, mb: 0.5 }} />
                  ))}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </Paper>
  );
};

export default ForecastPanel;