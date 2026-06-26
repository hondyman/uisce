import React, { useEffect, useState } from 'react';
import { Box, Button, Card, CardContent, Container, Grid, TextField, Typography, CircularProgress, Alert } from '@mui/material';
import { FactorExposureChart } from '../../../components/analytics/FactorExposureChart';
import { AttributionTable } from '../../../components/analytics/AttributionTable';
import SearchIcon from '@mui/icons-material/Search';
import { useTenant } from '../../../../contexts/TenantContext';
import { getSelectedRegion } from '../../../../lib/region';

interface ExposureData {
  portfolio_id: string;
  betas: Record<string, number>;
  r_squared: number;
}

interface AttributionData {
  TotalReturn: number;
  AlphaContribution: number;
  FactorContributions: Record<string, number>;
  Residual: number;
}

export const FactorAnalyticsPage: React.FC = () => {
  const { tenant, datasource } = useTenant();
  const [portfolioID, setPortfolioID] = useState<string>('demo-portfolio-001');
  const [exposure, setExposure] = useState<ExposureData | null>(null);
  const [attribution, setAttribution] = useState<AttributionData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchData = async () => {
    if (!portfolioID.trim()) return;
    setLoading(true);
    setError(null);
    try {
      // Parallel fetch
      const [expRes, attrRes] = await Promise.all([
        fetch(`/api/analytics/factors/exposure/${portfolioID}`, {
          headers: {
            'X-Tenant-ID': tenant?.id || '',
            'X-Tenant-Datasource-ID': datasource?.id || '',
            'X-Tenant-Region': getSelectedRegion(),
          },
        }),
        fetch(`/api/analytics/factors/attribution/${portfolioID}`, {
          headers: {
            'X-Tenant-ID': tenant?.id || '',
            'X-Tenant-Datasource-ID': datasource?.id || '',
            'X-Tenant-Region': getSelectedRegion(),
          },
        })
      ]);

      if (!expRes.ok || !attrRes.ok) {
        throw new Error('Failed to fetch analytics data');
      }

      const expData = await expRes.json();
      const attrData = await attrRes.json();

      setExposure(expData);
      setAttribution(attrData);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setExposure(null);
      setAttribution(null);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  return (
    <Container maxWidth="xl" sx={{ mt: 4, mb: 4 }}>
      <Typography variant="h4" gutterBottom component="div" sx={{ fontWeight: 'bold', color: 'primary.main' }}>
        Factor Analytics
      </Typography>

      <Box sx={{ mb: 4, display: 'flex', gap: 2 }}>
        <TextField
          label="Portfolio ID"
          variant="outlined"
          size="small"
          value={portfolioID}
          onChange={(e) => setPortfolioID(e.target.value)}
          sx={{ width: 300 }}
        />
        <Button variant="contained" startIcon={<SearchIcon />} onClick={fetchData} disabled={loading}>
          Analyze
        </Button>
      </Box>

      {error && <Alert severity="error" sx={{ mb: 3 }}>{error}</Alert>}
      
      {loading ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', p: 8 }}>
          <CircularProgress />
        </Box>
      ) : (
        <Grid container spacing={3}>
          {exposure && (
            <Grid item xs={12} md={6}>
              <Card elevation={2}>
                <CardContent>
                  <FactorExposureChart betas={exposure.betas} />
                  <Typography variant="body2" color="text.secondary" align="center" sx={{ mt: 2 }}>
                    R-Squared: {(exposure.r_squared * 100).toFixed(1)}%
                  </Typography>
                </CardContent>
              </Card>
            </Grid>
          )}

          {attribution && (
            <Grid item xs={12} md={6}>
              <Card elevation={2}>
                <CardContent>
                  <AttributionTable data={attribution} />
                </CardContent>
              </Card>
            </Grid>
          )}

          {!exposure && !attribution && !error && (
            <Grid item xs={12}>
              <Typography variant="body1" color="text.secondary">
                Enter a portfolio ID to view analytics.
              </Typography>
            </Grid>
          )}
        </Grid>
      )}
    </Container>
  );
};
