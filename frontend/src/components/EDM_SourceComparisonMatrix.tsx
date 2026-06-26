import React, { useState } from 'react';
import {
  Box,
  Card,
  CardContent,
  CardHeader,
  Button,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Chip,
  Grid,
  Alert,
  LinearProgress,
  Typography,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
} from '@mui/material';
import TrendingUpIcon from '@mui/icons-material/TrendingUp';
import TrendingDownIcon from '@mui/icons-material/TrendingDown';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import DownloadIcon from '@mui/icons-material/Download';
import RefreshIcon from '@mui/icons-material/Refresh';

interface SourceComparison {
  metric: string;
  primary: {
    label: string;
    value: string | number;
    trend?: 'up' | 'down';
    percentage?: number;
  };
  secondary: {
    label: string;
    value: string | number;
    percentage?: number;
  };
  delta: string;
  severity?: 'critical' | 'warning' | 'info';
}

interface SourceData {
  name: string;
  preference: '1st' | '2nd' | '3rd';
  uptime: number;
}

const mockComparisons: SourceComparison[] = [
  {
    metric: 'Confidence Score',
    primary: { label: 'TradingHours', value: '98%', trend: 'up', percentage: 98 },
    secondary: { label: 'EODHD', value: '82%', percentage: 82 },
    delta: '+16.0%',
    severity: 'info',
  },
  {
    metric: 'Coverage %',
    primary: { label: 'Full Universe', value: '99.42%', percentage: 99.42 },
    secondary: { label: 'Lagging', value: '91.20%', percentage: 91.20 },
    delta: 'Significant',
    severity: 'info',
  },
  {
    metric: 'Timeliness',
    primary: { label: 'Real-time (15ms)', value: '15ms', trend: 'up' },
    secondary: { label: '150ms Delay', value: '150ms' },
    delta: 'Critical',
    severity: 'critical',
  },
  {
    metric: 'Last Updated',
    primary: { label: '2 mins ago', value: '2m' },
    secondary: { label: '1 hour ago', value: '1h' },
    delta: 'N/A',
    severity: 'info',
  },
  {
    metric: 'Error Count',
    primary: { label: '12 errors', value: 12 },
    secondary: { label: '48 errors', value: 48 },
    delta: '-36',
    severity: 'info',
  },
];

const mockSources: SourceData[] = [
  { name: 'TradingHours', preference: '1st', uptime: 99.99 },
  { name: 'EODHD', preference: '2nd', uptime: 98.42 },
];

export const SourceComparisonMatrix: React.FC = () => {
  const [comparisons, setComparisons] = useState(mockComparisons);
  const [openDetails, setOpenDetails] = useState(false);
  const [selectedMetric, setSelectedMetric] = useState<SourceComparison | null>(null);

  const handleOpenDetails = (comparison: SourceComparison) => {
    setSelectedMetric(comparison);
    setOpenDetails(true);
  };

  const getPreferenceColor = (pref: string): 'success' | 'warning' | 'error' => {
    if (pref === '1st') return 'success';
    if (pref === '2nd') return 'warning';
    return 'error';
  };

  const getSeverityColor = (sev?: string): 'default' | 'error' | 'warning' | 'info' | 'success' => {
    if (sev === 'critical') return 'error';
    if (sev === 'warning') return 'warning';
    return 'info';
  };

  return (
    <Box sx={{ p: 3 }}>
      <Card>
        <CardHeader
          title="Source Comparison Matrix"
          subheader="Detailed evaluation of primary and secondary data providers"
          action={
            <Box sx={{ display: 'flex', gap: 1 }}>
              <Button
                variant="outlined"
                startIcon={<RefreshIcon />}
                onClick={() => console.log('Refresh')}
              >
                Refresh
              </Button>
              <Button
                variant="contained"
                startIcon={<DownloadIcon />}
                onClick={() => console.log('Export')}
              >
                Export Report
              </Button>
            </Box>
          }
        />
        <CardContent>
          <Grid container spacing={3} sx={{ mb: 3 }}>
            {/* Source Cards */}
            <Grid item xs={12}>
              <Grid container spacing={2}>
                {mockSources.map((source) => (
                  <Grid item xs={12} sm={6} key={source.name}>
                    <Card sx={{ bgcolor: source.preference === '1st' ? '#f0f9ff' : '#fffbf0' }}>
                      <CardContent>
                        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'start', mb: 2 }}>
                          <Box>
                            <Typography variant="h6" sx={{ fontWeight: 600 }}>
                              {source.name}
                            </Typography>
                            <Chip
                              label={`${source.preference} Preference`}
                              color={getPreferenceColor(source.preference)}
                              size="small"
                              sx={{ mt: 1 }}
                            />
                          </Box>
                        </Box>
                        <Box>
                          <Typography variant="caption" color="textSecondary">
                            Uptime
                          </Typography>
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mt: 0.5 }}>
                            <LinearProgress
                              variant="determinate"
                              value={source.uptime}
                              sx={{ flex: 1, height: 6, borderRadius: 3 }}
                            />
                            <Typography variant="body2" sx={{ fontWeight: 600, minWidth: 45 }}>
                              {source.uptime}%
                            </Typography>
                          </Box>
                        </Box>
                      </CardContent>
                    </Card>
                  </Grid>
                ))}
              </Grid>
            </Grid>

            {/* Comparison Table */}
            <Grid item xs={12}>
              <TableContainer component={Paper}>
                <Table>
                  <TableHead sx={{ bgcolor: '#f3f4f6' }}>
                    <TableRow>
                      <TableCell sx={{ fontWeight: 700 }}>Metric</TableCell>
                      <TableCell sx={{ fontWeight: 700 }}>TradingHours</TableCell>
                      <TableCell sx={{ fontWeight: 700 }}>EODHD</TableCell>
                      <TableCell align="right" sx={{ fontWeight: 700 }}>
                        Delta
                      </TableCell>
                      <TableCell align="center" sx={{ fontWeight: 700 }}>
                        Severity
                      </TableCell>
                      <TableCell align="center" sx={{ fontWeight: 700 }}>
                        Action
                      </TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {comparisons.map((comp, idx) => (
                      <TableRow key={idx} hover>
                        <TableCell sx={{ fontWeight: 600 }}>{comp.metric}</TableCell>
                        <TableCell>
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                            {comp.primary.trend === 'up' && <TrendingUpIcon sx={{ color: 'success.main', fontSize: 18 }} />}
                            {comp.primary.trend === 'down' && <TrendingDownIcon sx={{ color: 'error.main', fontSize: 18 }} />}
                            <Box>
                              <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                                {comp.primary.value}
                              </Typography>
                              {comp.primary.percentage !== undefined && (
                                <LinearProgress
                                  variant="determinate"
                                  value={comp.primary.percentage > 100 ? 100 : comp.primary.percentage}
                                  sx={{ mt: 0.5, height: 4, borderRadius: 2 }}
                                />
                              )}
                            </Box>
                          </Box>
                        </TableCell>
                        <TableCell>
                          <Box>
                            <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                              {comp.secondary.value}
                            </Typography>
                            {comp.secondary.percentage !== undefined && (
                              <LinearProgress
                                variant="determinate"
                                value={comp.secondary.percentage > 100 ? 100 : comp.secondary.percentage}
                                sx={{ mt: 0.5, height: 4, borderRadius: 2 }}
                              />
                            )}
                          </Box>
                        </TableCell>
                        <TableCell align="right">
                          <Typography
                            variant="body2"
                            sx={{
                              fontWeight: 600,
                              color: comp.delta.includes('-') ? 'success.main' : 'info.main',
                            }}
                          >
                            {comp.delta}
                          </Typography>
                        </TableCell>
                        <TableCell align="center">
                          <Chip
                            label={comp.severity?.toUpperCase()}
                            color={getSeverityColor(comp.severity)}
                            size="small"
                          />
                        </TableCell>
                        <TableCell align="center">
                          <Button
                            size="small"
                            onClick={() => handleOpenDetails(comp)}
                          >
                            View Details
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            </Grid>

            {/* Business Impact Card */}
            <Grid item xs={12} md={6}>
              <Card sx={{ bgcolor: '#ecf4ff', borderLeft: '4px solid #137fec' }}>
                <CardContent>
                  <Box sx={{ display: 'flex', alignItems: 'start', gap: 2, mb: 1 }}>
                    <CheckCircleIcon sx={{ color: 'success.main', mt: 0.5 }} />
                    <Box flex={1}>
                      <Typography variant="subtitle2" sx={{ fontWeight: 700, mb: 1 }}>
                        Business Impact
                      </Typography>
                      <Typography variant="body2" color="textSecondary" sx={{ mb: 1 }}>
                        TradingHours provides superior coverage for US micro-caps and OTC listings, justifying its Rank 1
                        status. The 135ms latency difference has direct consequences for high-frequency algorithms,
                        potentially saving $2.4M in annual slippage.
                      </Typography>
                      <Chip
                        label="Maintain TradingHours as primary source"
                        color="primary"
                        variant="outlined"
                        size="small"
                      />
                    </Box>
                  </Box>
                </CardContent>
              </Card>
            </Grid>

            {/* Risk Assessment */}
            <Grid item xs={12} md={6}>
              <Card>
                <CardHeader title="Risk Assessment" />
                <CardContent>
                  <Alert severity="warning" sx={{ mb: 2 }}>
                    EODHD has 48 errors vs TradingHours 12. Recommend verification before using as backup.
                  </Alert>
                  <Typography variant="caption" color="textSecondary">
                    Primary source recommended for mission-critical portfolios. Secondary source suitable for
                    non-critical data only.
                  </Typography>
                </CardContent>
              </Card>
            </Grid>
          </Grid>
        </CardContent>
      </Card>

      {/* Details Dialog */}
      <Dialog open={openDetails} onClose={() => setOpenDetails(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Metric Details: {selectedMetric?.metric}</DialogTitle>
        <DialogContent sx={{ pt: 2 }}>
          <Typography variant="body2" color="textSecondary" sx={{ mb: 2 }}>
            Detailed analysis and historical trends for this metric.
          </Typography>
          <Box sx={{ bgcolor: '#f3f4f6', p: 2, borderRadius: 1, mb: 2 }}>
            <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1 }}>
              TradingHours: {selectedMetric?.primary.value}
            </Typography>
            <Typography variant="body2" color="textSecondary" sx={{ mb: 2 }}>
              Current value shows {selectedMetric?.delta} improvement over EODHD.
            </Typography>
            <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1 }}>
              EODHD: {selectedMetric?.secondary.value}
            </Typography>
            <Typography variant="body2" color="textSecondary">
              Secondary source maintains acceptable service levels but lags in key metrics.
            </Typography>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpenDetails(false)}>Close</Button>
          <Button variant="contained">Accept Comparison</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
