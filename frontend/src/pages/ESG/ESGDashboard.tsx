import React, { useState, useEffect } from 'react';
import {
  Box,
  Grid,
  Card,
  CardContent,
  Typography,
  LinearProgress,
  Chip,
  Alert,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Tabs,
  Tab,
} from '@mui/material';
import {
  Eco as EcoIcon,
  People as SocialIcon,
  Business as GovernanceIcon,
  Warning as WarningIcon,
} from '@mui/icons-material';
import { RadarChart, PolarGrid, PolarAngleAxis, PolarRadiusAxis, Radar, ResponsiveContainer, BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Cell } from 'recharts';

interface ESGMetrics {
  weighted_esg_score: number;
  weighted_environmental_score: number;
  weighted_social_score: number;
  weighted_governance_score: number;
  carbon_intensity: number;
  esg_coverage_pct: number;
  vs_benchmark_esg_score: number;
  vs_benchmark_carbon: number;
  paris_aligned: boolean;
}

interface SDGImpact {
  sdg_number: number;
  sdg_name: string;
  allocation_pct: number;
  invested_amount: number;
}

interface Violation {
  violation_id: string;
  ticker: string;
  violation_type: string;
  violation_description: string;
  severity: string;
  market_value: number;
  recommended_action: string;
}

const SDG_COLORS = ['#E5243B', '#DDA63A', '#4C9F38', '#C5192D', '#FF3A21', '#26BDE2', '#FCC30B', '#A21942', '#FD6925', '#DD1367', '#FD9D24', '#BF8B2E', '#3F7E44', '#0A97D9', '#56C02B', '#00689D', '#19486A'];

export const ESGDashboard: React.FC = () => {
  const [metrics, setMetrics] = useState<ESGMetrics | null>(null);
  const [sdgImpacts, setSdgImpacts] = useState<SDGImpact[]>([]);
  const [violations, setViolations] = useState<Violation[]>([]);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState(0);

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    try {
      const [metricsRes, sdgRes, violationsRes] = await Promise.all([
        fetch('/api/esg/portfolio-metrics'),
        fetch('/api/esg/sdg-impact'),
        fetch('/api/esg/violations?status=OPEN'),
      ]);

      const metricsData = await metricsRes.json();
      const sdgData = await sdgRes.json();
      const violationsData = await violationsRes.json();

      setMetrics(metricsData);
      setSdgImpacts(sdgData.impacts || []);
      setViolations(violationsData.violations || []);
    } catch (error) {
      console.error('Failed to fetch ESG data:', error);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return <LinearProgress />;
  }

  if (!metrics) {
    return <Alert severity="info">No ESG data available</Alert>;
  }

  // Prepare radar chart data
  const radarData = [
    { factor: 'Environmental', score: metrics.weighted_environmental_score, fullMark: 100 },
    { factor: 'Social', score: metrics.weighted_social_score, fullMark: 100 },
    { factor: 'Governance', score: metrics.weighted_governance_score, fullMark: 100 },
  ];

  const getScoreColor = (score: number) => {
    if (score >= 70) return 'success';
    if (score >= 50) return 'warning';
    return 'error';
  };

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'CRITICAL': return 'error';
      case 'HIGH': return 'error';
      case 'MEDIUM': return 'warning';
      default: return 'info';
    }
  };

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h4" gutterBottom>
        ESG & Impact Investing
      </Typography>

      {/* Violations Alert */}
      {violations.length > 0 && (
        <Alert severity="warning" icon={<WarningIcon />} sx={{ mb: 3 }}>
          <Typography variant="subtitle2">
            {violations.length} ESG screening violation{violations.length > 1 ? 's' : ''} requiring attention
          </Typography>
        </Alert>
      )}

      {/* ESG Score Cards */}
      <Grid container spacing={3} sx={{ mb: 3 }}>
        <Grid item xs={12} md={3}>
          <Card>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                <EcoIcon color="success" sx={{ mr: 1 }} />
                <Typography color="text.secondary">ESG Score</Typography>
              </Box>
              <Typography variant="h4">{metrics.weighted_esg_score.toFixed(1)}</Typography>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mt: 1 }}>
                <Chip
                  label={`vs Benchmark: ${metrics.vs_benchmark_esg_score > 0 ? '+' : ''}${metrics.vs_benchmark_esg_score.toFixed(1)}`}
                  size="small"
                  color={metrics.vs_benchmark_esg_score > 0 ? 'success' : 'error'}
                />
              </Box>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>
                Environmental
              </Typography>
              <Typography variant="h4" color="success.main">
                {metrics.weighted_environmental_score.toFixed(1)}
              </Typography>
              <LinearProgress
                variant="determinate"
                value={metrics.weighted_environmental_score}
                color="success"
                sx={{ mt: 1 }}
              />
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>
                Social
              </Typography>
              <Typography variant="h4" color="info.main">
                {metrics.weighted_social_score.toFixed(1)}
              </Typography>
              <LinearProgress
                variant="determinate"
                value={metrics.weighted_social_score}
                color="info"
                sx={{ mt: 1 }}
              />
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>
                Governance
              </Typography>
              <Typography variant="h4" color="primary.main">
                {metrics.weighted_governance_score.toFixed(1)}
              </Typography>
              <LinearProgress
                variant="determinate"
                value={metrics.weighted_governance_score}
                color="primary"
                sx={{ mt: 1 }}
              />
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Carbon Metrics */}
      <Grid container spacing={3} sx={{ mb: 3 }}>
        <Grid item xs={12} md={6}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                Carbon Footprint
              </Typography>
              <Typography variant="h3" color="text.secondary">
                {metrics.carbon_intensity.toFixed(1)}
              </Typography>
              <Typography variant="body2" color="text.secondary">
                Tons CO2e per $1M invested
              </Typography>
              <Box sx={{ mt: 2 }}>
                <Chip
                  label={metrics.paris_aligned ? 'Paris Aligned' : 'Not Paris Aligned'}
                  color={metrics.paris_aligned ? 'success' : 'warning'}
                  size="small"
                />
                <Chip
                  label={`vs Benchmark: ${metrics.vs_benchmark_carbon > 0 ? '+' : ''}${metrics.vs_benchmark_carbon.toFixed(1)}%`}
                  color={metrics.vs_benchmark_carbon < 0 ? 'success' : 'error'}
                  size="small"
                  sx={{ ml: 1 }}
                />
              </Box>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={6}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                ESG Profile
              </Typography>
              <ResponsiveContainer width="100%" height={200}>
                <RadarChart data={radarData}>
                  <PolarGrid />
                  <PolarAngleAxis dataKey="factor" />
                  <PolarRadiusAxis angle={90} domain={[0, 100]} />
                  <Radar name="Score" dataKey="score" stroke="#1976d2" fill="#1976d2" fillOpacity={0.6} />
                </RadarChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Tabs */}
      <Box sx={{ borderBottom: 1, borderColor: 'divider', mb: 3 }}>
        <Tabs value={tab} onChange={(_, v) => setTab(v)}>
          <Tab label="SDG Impact" />
          <Tab label={`Violations (${violations.length})`} />
        </Tabs>
      </Box>

      {/* SDG Impact Tab */}
      {tab === 0 && (
        <Grid container spacing={3}>
          <Grid item xs={12}>
            <Paper sx={{ p: 3 }}>
              <Typography variant="h6" gutterBottom>
                UN Sustainable Development Goals Alignment
              </Typography>
              <ResponsiveContainer width="100%" height={300}>
                <BarChart data={sdgImpacts}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="sdg_number" label={{ value: 'SDG #', position: 'insideBottom', offset: -5 }} />
                  <YAxis label={{ value: '% of Portfolio', angle: -90, position: 'insideLeft' }} />
                  <Tooltip
                    content={({ active, payload }) => {
                      if (active && payload && payload.length) {
                        const data = payload[0].payload as SDGImpact;
                        return (
                          <Paper sx={{ p: 1 }}>
                            <Typography variant="body2" fontWeight="bold">SDG {data.sdg_number}</Typography>
                            <Typography variant="caption">{data.sdg_name}</Typography>
                            <Typography variant="body2">{data.allocation_pct.toFixed(1)}%</Typography>
                          </Paper>
                        );
                      }
                      return null;
                    }}
                  />
                  <Bar dataKey="allocation_pct">
                    {sdgImpacts.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={SDG_COLORS[entry.sdg_number - 1]} />
                    ))}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            </Paper>
          </Grid>
        </Grid>
      )}

      {/* Violations Tab */}
      {tab === 1 && (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Ticker</TableCell>
                <TableCell>Violation</TableCell>
                <TableCell>Severity</TableCell>
                <TableCell align="right">Market Value</TableCell>
                <TableCell>Recommended Action</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {violations.map((violation) => (
                <TableRow key={violation.violation_id}>
                  <TableCell>{violation.ticker}</TableCell>
                  <TableCell>
                    <Typography variant="body2" fontWeight="medium">
                      {violation.violation_type.replace('_', ' ')}
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      {violation.violation_description}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={violation.severity}
                      size="small"
                      color={getSeverityColor(violation.severity)}
                    />
                  </TableCell>
                  <TableCell align="right">
                    ${violation.market_value.toLocaleString()}
                  </TableCell>
                  <TableCell>
                    <Chip label={violation.recommended_action} size="small" variant="outlined" />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}
    </Box>
  );
};
