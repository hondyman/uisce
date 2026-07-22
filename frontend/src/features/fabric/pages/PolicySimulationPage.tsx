import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Box,
  Typography,
  Alert,
  CircularProgress,
  Paper,
  Grid,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Button,
  FormHelperText,
} from '@mui/material';
import SimulationResultDetail from '../components/SimulationResultDetail';
import ForecastPanel from '../components/ForecastPanel';
import WhatIfEditor from '../components/WhatIfEditor';
import { apiFetch } from '../../../lib/apiClient';

const PolicySimulationPage: React.FC = () => {
  const [policyId, setPolicyId] = useState('');
  const [fromDs, setFromDs] = useState('');
  const [toDs, setToDs] = useState('');

  const queryClient = useQueryClient();

  const { data: optionsData, isLoading: optionsLoading, error: optionsError } = useQuery({
    queryKey: ['simulation-options'],
    queryFn: async () => {
      const [policiesRes, datasourcesRes] = await Promise.all([
        apiFetch('/api/rest/policies?order_by=id asc'),
        apiFetch('/api/rest/datasources?order_by=source_name asc'),
      ]);
      if (!policiesRes.ok) throw new Error(await policiesRes.text());
      if (!datasourcesRes.ok) throw new Error(await datasourcesRes.text());
      const policies = await policiesRes.json();
      const datasources = await datasourcesRes.json();
      return { policies, datasources };
    },
  });

  const runForecast = useMutation({
    mutationFn: async ({ fromDs, toDs }: { fromDs: string; toDs: string }) => {
      const res = await apiFetch('/api/rest/forecast-policy-run', {
        method: 'POST',
        body: JSON.stringify({ from_ds: fromDs, to_ds: toDs }),
      });
      if (!res.ok) throw new Error(await res.text());
      return res.json();
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['forecast-policy-run'] }),
  });

  const runSimulation = useMutation({
    mutationFn: async ({ policyId, fromDs, toDs }: { policyId: string; fromDs: string; toDs: string }) => {
      const res = await apiFetch('/api/rest/simulate-policy-run', {
        method: 'POST',
        body: JSON.stringify({ policy_id: policyId, from_ds: fromDs, to_ds: toDs }),
      });
      if (!res.ok) throw new Error(await res.text());
      return res.json();
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['simulate-policy-run'] }),
  });

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (policyId && fromDs && toDs) {
      runSimulation.mutate({ policyId, fromDs, toDs });
      runForecast.mutate({ fromDs, toDs });
    }
  };

  if (optionsLoading) {
    return <CircularProgress />;
  }

  if (optionsError) {
    return <Alert severity="error">Failed to load simulation options: {optionsError?.message}</Alert>;
  }

  return (
    <Box sx={{ flexGrow: 1, p: 3 }}>
      <Typography variant="h4" gutterBottom>
        Policy Simulation Lab
      </Typography>
      <Typography paragraph color="text.secondary">
        Select a policy and a set of changes to simulate the outcome. This allows you to test policy changes in a safe environment before they are enforced.
      </Typography>

      <Paper component="form" onSubmit={handleSubmit} sx={{ p: 3 }}>
        <Grid container spacing={3}>
          <Grid size={12}>
            <FormControl fullWidth required>
              <InputLabel id="policy-select-label">Policy</InputLabel>
              <Select
                labelId="policy-select-label"
                id="policy-select"
                value={policyId}
                label="Policy"
                onChange={(e) => setPolicyId(e.target.value)}
              >
                {(optionsData?.policies || []).map((p: any) => (
                  <MenuItem key={p.id} value={p.id}>
                    {p.name || p.id}
                  </MenuItem>
                ))}
              </Select>
              <FormHelperText>Select the policy version to test against.</FormHelperText>
            </FormControl>
          </Grid>
          <Grid size={{ 'xs': 12, 'sm': 6 }}>
            <FormControl fullWidth required>
              <InputLabel id="from-ds-select-label">From Datasource</InputLabel>
              <Select
                labelId="from-ds-select-label"
                id="from-ds-select"
                value={fromDs}
                label="From Datasource"
                onChange={(e) => setFromDs(e.target.value)}
              >
                {(optionsData?.datasources || []).map((ds: any) => (
                  <MenuItem key={ds.id} value={ds.id}>
                    {ds.source_name}
                  </MenuItem>
                ))}
              </Select>
              <FormHelperText>The 'before' state for the simulation.</FormHelperText>
            </FormControl>
          </Grid>
          <Grid size={{ 'xs': 12, 'sm': 6 }}>
            <FormControl fullWidth required>
              <InputLabel id="to-ds-select-label">To Datasource</InputLabel>
              <Select
                labelId="to-ds-select-label"
                id="to-ds-select"
                value={toDs}
                label="To Datasource"
                onChange={(e) => setToDs(e.target.value)}
              >
                {(optionsData?.datasources || []).map((ds: any) => (
                  <MenuItem key={ds.id} value={ds.id}>
                    {ds.source_name}
                  </MenuItem>
                ))}
              </Select>
              <FormHelperText>The 'after' state for the simulation.</FormHelperText>
            </FormControl>
          </Grid>
          <Grid size={12}>
            <Button type="submit" variant="contained" disabled={runSimulation.isPending || !policyId || !fromDs || !toDs}>
              {runSimulation.isPending ? <CircularProgress size={24} /> : 'Run Simulation'}
            </Button>
          </Grid>
        </Grid>
      </Paper>

      {runSimulation.error && (
        <Alert severity="error" sx={{ mt: 3 }}>
          Simulation failed: {runSimulation.error.message}
        </Alert>
      )}

      {runSimulation.data && <SimulationResultDetail result={runSimulation.data} />}

      <ForecastPanel data={runForecast.data} loading={runForecast.isPending} error={runForecast.error} />

      <WhatIfEditor />
    </Box>
  );
};

export default PolicySimulationPage;