import React, { useEffect, useState } from 'react';
import { Box, Button, Container, Grid, Typography, CircularProgress, Alert } from '@mui/material';
import { Add as AddIcon, Refresh as RefreshIcon } from '@mui/icons-material';
import { sloApi, SLODefinition } from '../api/sloApi';
import SLOCard from '../components/observability/SLOCard';

const SLODashboard: React.FC = () => {
  const [slos, setSlos] = useState<SLODefinition[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchSLOs = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await sloApi.listSLOs();
      setSlos(data);
    } catch (err: any) {
      setError(err.message || 'Failed to load SLOs');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchSLOs();
  }, []);

  return (
    <Container maxWidth="xl" sx={{ mt: 4, mb: 4 }}>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={4}>
        <Typography variant="h4" component="h1">
          SLO Dashboard
        </Typography>
        <Box display="flex" gap={2}>
            <Button 
                variant="outlined" 
                startIcon={<RefreshIcon />} 
                onClick={fetchSLOs}
            >
                Refresh
            </Button>
            <Button 
                variant="contained" 
                startIcon={<AddIcon />} 
                onClick={() => alert('Create SLO modal coming soon')}
            >
                Create SLO
            </Button>
        </Box>
      </Box>

      {error && (
        <Alert severity="error" sx={{ mb: 3 }}>
          {error}
        </Alert>
      )}

      {loading ? (
        <Box display="flex" justifyContent="center" p={4}>
          <CircularProgress />
        </Box>
      ) : (
        <Grid container spacing={3}>
          {slos.map((slo) => (
            <Grid item xs={12} sm={6} md={4} lg={3} key={slo.id}>
              <SLOCard slo={slo} status="unknown" />
            </Grid>
          ))}
          {slos.length === 0 && !error && (
            <Box width="100%" p={4} textAlign="center">
              <Typography color="text.secondary">
                No active SLOs found. Create one to get started.
              </Typography>
            </Box>
          )}
        </Grid>
      )}
    </Container>
  );
};

export default SLODashboard;
