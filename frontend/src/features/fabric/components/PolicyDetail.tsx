import React, { useState, Suspense } from 'react';
import { gql, useQuery, useMutation } from '@apollo/client';
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

const GET_POLICY_DETAIL = gql`
  query GetPolicyDetail($id: uuid!) {
    policy_rules_by_pk(id: $id) {
      id
      name
      description
      is_active
      spec
      versions(order_by: { version: desc }, limit: 1) {
        version
        author
        created_at
      }
    }
  }
`;

const SET_POLICY_ACTIVE = gql`
  mutation SetPolicyActive($id: uuid!, $isActive: Boolean!) {
    update_policy_rules_by_pk(pk_columns: { id: $id }, _set: { is_active: $isActive }) {
      id
      is_active
    }
  }
`;

const GET_POLICY_VERSIONS = gql`
  query GetPolicyVersions($policyId: uuid!) {
    policy_version_history(where: { policy_id: { _eq: $policyId } }, order_by: { version: desc }) {
      id
      version
      author
      created_at
      change_summary
    }
  }
`;

const PolicyOverviewTab: React.FC<{ policy: any }> = ({ policy }) => {
  const latestVersion = policy.versions?.[0];
  // Mock data for stats as per the design
  const usageStats = {
    evaluations: 12345,
    blockRate: 0.05,
    topCodes: ['rule-001', 'rule-007', 'rule-003'],
  };

  return (
    <Grid container spacing={3}>
      <Grid item xs={12} md={8}>
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
            <Grid item xs={4}>
              <Typography variant="h5">{usageStats.evaluations.toLocaleString()}</Typography>
              <Typography color="text.secondary">Evaluations</Typography>
            </Grid>
            <Grid item xs={4}>
              <Typography variant="h5">{(usageStats.blockRate * 100).toFixed(1)}%</Typography>
              <Typography color="text.secondary">Block Rate</Typography>
            </Grid>
            <Grid item xs={4}>
              <Typography color="text.secondary">Top Violations</Typography>
              <Box>
                {usageStats.topCodes.map(code => <Chip key={code} label={code} size="small" sx={{ mr: 0.5 }} />)}
              </Box>
            </Grid>
          </Grid>
        </Paper>
      </Grid>
      <Grid item xs={12} md={4}>
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
  const { data, loading, error } = useQuery(GET_POLICY_VERSIONS, { variables: { policyId } });

  if (loading) return <CircularProgress />;
  if (error) return <Alert severity="error">Failed to load version history: {error.message}</Alert>;

  return (
    <Paper variant="outlined">
      <List dense>
        {data?.policy_version_history.map((v: any) => (
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
  const { data, loading, error } = useQuery(GET_POLICY_DETAIL, {
    variables: { id: policyId },
  });
  const [setPolicyActive, { loading: mutationLoading }] = useMutation(SET_POLICY_ACTIVE);

  if (loading) return <CircularProgress />;
  if (error) return <Alert severity="error">Failed to load policy details: {error.message}</Alert>;

  const policy = data?.policy_rules_by_pk;

  if (!policy) {
    return <Alert severity="warning">Policy not found.</Alert>;
  }

  const handleToggleActive = (event: React.ChangeEvent<HTMLInputElement>) => {
    setPolicyActive({
      variables: { id: policy.id, isActive: event.target.checked },
      optimisticResponse: {
        update_policy_rules_by_pk: {
          __typename: 'policy_rules',
          id: policy.id,
          is_active: event.target.checked,
        },
      },
    });
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
          control={<Switch checked={policy.is_active} onChange={handleToggleActive} disabled={mutationLoading} />}
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