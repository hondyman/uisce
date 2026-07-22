import React, { useState, Suspense } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import {
  Box,
  Typography,
  CircularProgress,
  Alert,
  Paper,
  FormControlLabel,
  Switch,
  Button,
  Tabs,
  Tab,
  Grid,
  List,
  ListItemButton,
  ListItemText,
  Chip,
  ButtonGroup,
} from '@mui/material';
import LazySyntaxHighlighter from '../../../components/LazySyntaxHighlighter';
import yaml from 'js-yaml';
import { format } from 'date-fns';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiFetch } from '../../../../lib/apiClient';

const PolicyOverviewTab: React.FC<{ policy: any }> = ({ policy }) => {
  const latestVersion = policy.versions?.[0];
  const usageStats = {
    evaluations: 12345,
    blockRate: 0.05,
    topCodes: ['rule-001', 'rule-007', 'rule-003'],
  };

  return (
    <Grid container spacing={3}>
      <Grid size={{ 'xs': 12, 'md': 8 }}>
        <Typography variant="h6" gutterBottom>Metadata</Typography>
        <Paper variant="outlined" sx={{ p: 2 }}>
          <Typography variant="body2" color="text.secondary" gutterBottom>Description</Typography>
          <Typography paragraph>{policy.description || 'No description provided.'}</Typography>
          <Typography variant="body2" color="text.secondary">Owner</Typography>
          <Typography paragraph>{latestVersion?.author || 'system'}</Typography>
          <Typography variant="body2" color="text.secondary">Last Updated</Typography>
          <Typography>{latestVersion ? format(new Date(latestVersion.created_at), 'yyyy-MM-dd HH:mm') : 'N/A'}</Typography>
        </Paper>

        <Typography variant="h6" gutterBottom sx={{ mt: 3 }}>Usage Stats (Last 30 Days)</Typography>
        <Paper variant="outlined" sx={{ p: 2 }}>
          <Grid container spacing={2}>
            <Grid size={4}>
              <Typography variant="h5">{usageStats.evaluations.toLocaleString()}</Typography>
              <Typography color="text.secondary">Evaluations</Typography>
            </Grid>
            <Grid size={4}>
              <Typography variant="h5">{(usageStats.blockRate * 100).toFixed(1)}%</Typography>
              <Typography color="text.secondary">Block Rate</Typography>
            </Grid>
            <Grid size={4}>
              <Typography color="text.secondary">Top Violations</Typography>
              <Box>
                {usageStats.topCodes.map(code => <Chip key={code} label={code} size="small" sx={{ mr: 0.5 }} />)}
              </Box>
            </Grid>
          </Grid>
        </Paper>
      </Grid>
      <Grid size={{ 'xs': 12, 'md': 4 }}>
        <Typography variant="h6" gutterBottom>Quick Actions</Typography>
        <ButtonGroup orientation="vertical" fullWidth>
          <Button>Simulate</Button>
          <Button>Replay</Button>
          <Button>Forecast</Button>
          <Button component={RouterLink} to={`/fabric/policies/${policy.id}/history`}>Compare Versions</Button>
        </ButtonGroup>
      </Grid>
    </Grid>
  );
};

const PolicyVersionHistoryTab: React.FC<{ policyId: string }> = ({ policyId }) => {
  const { data, isLoading, error } = useQuery({
    queryKey: ['policy-versions', policyId],
    queryFn: async () => {
      const res = await apiFetch(`/api/rest/policy-version-history?policy_id=${encodeURIComponent(policyId)}`);
      if (!res.ok) {
        throw new Error(await res.text());
      }
      return res.json();
    },
    enabled: !!policyId,
  });

  if (isLoading) return <CircularProgress />;
  if (error) return <Alert severity="error">Failed to load version history: {(error as Error).message}</Alert>;

  return (
    <Paper variant="outlined">
      <List dense>
        {data?.map((v: any) => (
          <ListItemButton key={v.id} component={RouterLink} to={`/fabric/policies/${policyId}/history?versionB=${v.id}`}>
            <ListItemText
              primary={`v${v.version} - ${v.change_summary || 'Initial Version'}`}
              secondary={`${format(new Date(v.created_at), 'yyyy-MM-dd')} by ${v.author || 'system'}`}
            />
          </ListItemButton>
        ))}
      </List>
    </Paper>
  );
};

const PolicySpecTab: React.FC<{ spec: any }> = ({ spec }) => {
  const specAsYaml = yaml.dump(spec);
  return (
    <Paper variant="outlined" sx={{ maxHeight: 'calc(100vh - 350px)', overflow: 'auto' }}>
      <Suspense fallback={<div>Loading code...</div>}>
        <LazySyntaxHighlighter language="yaml" showLineNumbers>
          {specAsYaml}
        </LazySyntaxHighlighter>
      </Suspense>
    </Paper>
  );
};

interface PolicyDetailProps {
  policyId: string;
}

const PolicyDetail: React.FC<PolicyDetailProps> = ({ policyId }) => {
  const [activeTab, setActiveTab] = useState(0);
  const queryClient = useQueryClient();

  const { data, isLoading, error } = useQuery({
    queryKey: ['policy-detail', policyId],
    queryFn: async () => {
      const res = await apiFetch(`/api/rest/policies/${encodeURIComponent(policyId)}`);
      if (!res.ok) {
        throw new Error(await res.text());
      }
      const result = await res.json();
      return Array.isArray(result) ? result[0] : result;
    },
    enabled: !!policyId,
  });

  const setPolicyActiveMutation = useMutation({
    mutationFn: async ({ id, isActive }: { id: string; isActive: boolean }) => {
      const res = await apiFetch(`/api/rest/policies/${encodeURIComponent(id)}`, {
        method: 'PUT',
        body: JSON.stringify({ is_active: isActive }),
      });
      if (!res.ok) {
        throw new Error(await res.text());
      }
      return res.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['policy-detail', policyId] });
    },
  });

  if (isLoading) return <CircularProgress />;
  if (error) return <Alert severity="error">Failed to load policy details: {(error as Error).message}</Alert>;

  const policy = data;

  if (!policy) {
    return <Alert severity="warning">Policy not found.</Alert>;
  }

  const handleToggleActive = (event: React.ChangeEvent<HTMLInputElement>) => {
    setPolicyActiveMutation.mutate({ id: policy.id, isActive: event.target.checked });
  };

  const handleTabChange = (_event: React.SyntheticEvent, newValue: number) => {
    setActiveTab(newValue);
  };

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
        <Box>
          <Typography variant="h5">{policy.name} <Chip label={`v${policy.versions?.[0]?.version || 'N/A'}`} size="small" /></Typography>
          <Typography color="text.secondary" variant="body2" sx={{ fontFamily: 'monospace' }}>ID: {policy.id}</Typography>
        </Box>
        <FormControlLabel
          control={<Switch checked={policy.is_active} onChange={handleToggleActive} disabled={setPolicyActiveMutation.isPending} />}
          label="Active"
        />
      </Box>

      <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
        <Tabs value={activeTab} onChange={handleTabChange} aria-label="policy detail tabs">
          <Tab label="Overview" />
          <Tab label="Version History" />
          <Tab label="Spec" />
          <Tab label="Violations" disabled />
          <Tab label="Linked Standards" disabled />
        </Tabs>
      </Box>

      <Box sx={{ pt: 3 }}>
        {activeTab === 0 && <PolicyOverviewTab policy={policy} />}
        {activeTab === 1 && <PolicyVersionHistoryTab policyId={policy.id} />}
        {activeTab === 2 && <PolicySpecTab spec={policy.spec} />}
      </Box>
    </Box>
  );
};

export default PolicyDetail;
