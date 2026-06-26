import React, { useState, useEffect } from 'react';
import {
  Box,
  Container,
  Typography,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
  CircularProgress,
  Button,
  Alert,
  TextField,
  InputAdornment,
} from '@mui/material';
import { Speed, TrendingUp, Search, Refresh } from '@mui/icons-material';

interface PreAggCostEstimate {
  estimated_queries_per_day: number;
  estimated_speedup_factor: number;
  estimated_storage_bytes: number;
  score: number;
}

interface PreAggRecommendation {
  tenant_id: string;
  bo_name: string;
  grain: string[];
  measures: string[];
  cost_estimate: PreAggCostEstimate;
  recommendation_type: string;
}

export const GlobalAdvisorPage: React.FC = () => {
  const [recommendations, setRecommendations] = useState<PreAggRecommendation[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filter, setFilter] = useState('');

  const fetchRecommendations = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch('/api/advisor/global?window_days=7');
      if (!res.ok) throw new Error(await res.text());
      setRecommendations(await res.json());
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchRecommendations();
  }, []);

  const formatBytes = (bytes: number) => {
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
  };

  const filteredRecs = recommendations.filter(
    (r) =>
      r.bo_name.toLowerCase().includes(filter.toLowerCase()) ||
      r.tenant_id.toLowerCase().includes(filter.toLowerCase()) ||
      r.grain.some((g) => g.toLowerCase().includes(filter.toLowerCase()))
  );

  return (
    <Container maxWidth="xl" sx={{ py: 3 }}>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Typography variant="h4">
          <Speed sx={{ mr: 1, verticalAlign: 'text-bottom' }} />
          Global Performance Advisor
        </Typography>
        <Button onClick={fetchRecommendations} startIcon={<Refresh />} variant="outlined">
          Refresh
        </Button>
      </Box>

      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      <Paper sx={{ mb: 2, p: 2 }}>
        <TextField
          fullWidth
          size="small"
          placeholder="Filter by tenant, BO, or grain..."
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <Search />
              </InputAdornment>
            ),
          }}
        />
      </Paper>

      {loading ? (
        <Box display="flex" justifyContent="center" py={4}>
          <CircularProgress />
        </Box>
      ) : filteredRecs.length === 0 ? (
        <Paper sx={{ p: 4, textAlign: 'center' }}>
          <Typography color="text.secondary">
            {filter ? 'No recommendations match your filter.' : 'No pre-aggregation recommendations at this time.'}
          </Typography>
        </Paper>
      ) : (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Tenant</TableCell>
                <TableCell>Business Object</TableCell>
                <TableCell>Grain</TableCell>
                <TableCell>Measures</TableCell>
                <TableCell align="center">Speedup</TableCell>
                <TableCell align="right">Queries/Day</TableCell>
                <TableCell align="right">Est. Storage</TableCell>
                <TableCell align="right">Score</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {filteredRecs.map((rec, idx) => (
                <TableRow key={idx} hover>
                  <TableCell>{rec.tenant_id}</TableCell>
                  <TableCell>
                    <Typography fontWeight="medium">{rec.bo_name}</Typography>
                  </TableCell>
                  <TableCell>
                    {rec.grain.map((g) => (
                      <Chip key={g} label={g} size="small" sx={{ mr: 0.5, mb: 0.5 }} />
                    ))}
                  </TableCell>
                  <TableCell>
                    {rec.measures.map((m) => (
                      <Chip key={m} label={m} size="small" color="primary" variant="outlined" sx={{ mr: 0.5, mb: 0.5 }} />
                    ))}
                  </TableCell>
                  <TableCell align="center">
                    <Chip
                      icon={<TrendingUp />}
                      label={`${rec.cost_estimate.estimated_speedup_factor.toFixed(1)}x`}
                      color="success"
                      size="small"
                    />
                  </TableCell>
                  <TableCell align="right">{rec.cost_estimate.estimated_queries_per_day}</TableCell>
                  <TableCell align="right">{formatBytes(rec.cost_estimate.estimated_storage_bytes)}</TableCell>
                  <TableCell align="right">
                    <Typography fontWeight="bold" color="primary">
                      {rec.cost_estimate.score.toFixed(1)}
                    </Typography>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      <Typography variant="body2" color="text.secondary" sx={{ mt: 2, textAlign: 'center' }}>
        Showing top {filteredRecs.length} recommendations based on query workload from the last 7 days.
      </Typography>
    </Container>
  );
};

export default GlobalAdvisorPage;
