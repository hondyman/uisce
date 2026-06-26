import React, { useState } from 'react';
import { gql, useQuery, useMutation } from '@apollo/client';
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

const GET_SIMULATION_OPTIONS = gql`
  query GetSimulationOptions {
    policy_rules(order_by: { id: asc }) {
      id
      name: id # Using ID as name for now
      description
    }
    tenant_product_datasource(order_by: { source_name: asc }) {
      id
      source_name
    }
  }
`;

const SIMULATE_POLICY_RUN = gql`
  mutation SimulatePolicyRun($policyId: uuid!, $fromDs: String!, $toDs: String!) {
    SimulatePolicyRun(policy_id: $policyId, from_ds: $fromDs, to_ds: $toDs) {
      policy_id
      summary {
        breaking
        medium
        low
      }
      violations {
        rule_id
        severity
        message
        qualified_path
      }
      changelog_md
    }
  }
`;

export const FORECAST_POLICY_RUN = gql`
  mutation ForecastPolicyRun($fromDs: String!, $toDs: String!) {
    forecast_policy_run(from_ds: $fromDs, to_ds: $toDs) {
      policy_id
      policy_name
      block_probability
      confidence
      top_factors
    }
  }
`;

const PolicySimulationPage: React.FC = () => {
  const [policyId, setPolicyId] = useState('');
  const [fromDs, setFromDs] = useState('');
  const [toDs, setToDs] = useState('');

  const { data: optionsData, loading: optionsLoading, error: optionsError } = useQuery(GET_SIMULATION_OPTIONS);

  const [runForecast, { data: forecastData, loading: forecastLoading, error: forecastError }] = useMutation(FORECAST_POLICY_RUN);

  const [runSimulation, { data: simData, loading: simLoading, error: simError }] = useMutation(SIMULATE_POLICY_RUN);

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (policyId && fromDs && toDs) {
      runSimulation({
        variables: {
          policyId,
          fromDs,
          toDs,
        },
      });
      // Also run the forecast in parallel
      runForecast({
        variables: {
          fromDs,
          toDs,
        },
      });
    }
  };

  if (optionsLoading) {
    return <CircularProgress />;
  }

  if (optionsError) {
    return <Alert severity="error">Failed to load simulation options: {optionsError.message}</Alert>;
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
          <Grid item xs={12}>
            <FormControl fullWidth required>
              <InputLabel id="policy-select-label">Policy</InputLabel>
              <Select
                labelId="policy-select-label"
                id="policy-select"
                value={policyId}
                label="Policy"
                onChange={(e) => setPolicyId(e.target.value)}
              >
                {optionsData?.policy_rules.map((p: any) => (
                  <MenuItem key={p.id} value={p.id}>
                    {p.name}
                  </MenuItem>
                ))}
              </Select>
              <FormHelperText>Select the policy version to test against.</FormHelperText>
            </FormControl>
          </Grid>
          <Grid item xs={12} sm={6}>
            <FormControl fullWidth required>
              <InputLabel id="from-ds-select-label">From Datasource</InputLabel>
              <Select
                labelId="from-ds-select-label"
                id="from-ds-select"
                value={fromDs}
                label="From Datasource"
                onChange={(e) => setFromDs(e.target.value)}
              >
                {optionsData?.tenant_product_datasource.map((ds: any) => (
                  <MenuItem key={ds.id} value={ds.id}>
                    {ds.source_name}
                  </MenuItem>
                ))}
              </Select>
              <FormHelperText>The 'before' state for the simulation.</FormHelperText>
            </FormControl>
          </Grid>
          <Grid item xs={12} sm={6}>
            <FormControl fullWidth required>
              <InputLabel id="to-ds-select-label">To Datasource</InputLabel>
              <Select
                labelId="to-ds-select-label"
                id="to-ds-select"
                value={toDs}
                label="To Datasource"
                onChange={(e) => setToDs(e.target.value)}
              >
                {optionsData?.tenant_product_datasource.map((ds: any) => (
                  <MenuItem key={ds.id} value={ds.id}>
                    {ds.source_name}
                  </MenuItem>
                ))}
              </Select>
              <FormHelperText>The 'after' state for the simulation.</FormHelperText>
            </FormControl>
          </Grid>
          <Grid item xs={12}>
            <Button type="submit" variant="contained" disabled={simLoading || !policyId || !fromDs || !toDs}>
              {simLoading ? <CircularProgress size={24} /> : 'Run Simulation'}
            </Button>
          </Grid>
        </Grid>
      </Paper>

      {simError && (
        <Alert severity="error" sx={{ mt: 3 }}>
          Simulation failed: {simError.message}
        </Alert>
      )}

      {simData?.SimulatePolicyRun && <SimulationResultDetail result={simData.SimulatePolicyRun} />}

      <ForecastPanel data={forecastData?.forecast_policy_run} loading={forecastLoading} error={forecastError} />

      <WhatIfEditor />
    </Box>
  );
};

export default PolicySimulationPage;