import { useState, useEffect, useCallback } from 'react';
import useBlockableNavigate from '../../../components/RouteBlocker/useBlockableNavigate';
import { useParams } from 'react-router-dom';
import { Box, Typography, CircularProgress, Alert, Paper, Grid, Button } from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import { listAccessPolicies, simulatePolicyChange } from '../../../api';
import { AccessControlPolicy, PolicySimulationResult } from '../../../types';
import PolicyList from '../components/PolicyList';
import EditPolicy from '../../../components/EditPolicy';
import PolicySimulationResultViewer from '../components/PolicySimulationResultViewer';

export default function PolicyManagementPage() {
  const [policies, setPolicies] = useState<AccessControlPolicy[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedPolicy, setSelectedPolicy] = useState<AccessControlPolicy | null>(null);
  const [isEditorOpen, setIsEditorOpen] = useState(false);
  const [simulationResult, setSimulationResult] = useState<PolicySimulationResult | null>(null);
  const navigate = useBlockableNavigate();
  const { policyId } = useParams<{ policyId?: string }>();

  const fetchPolicies = useCallback(async () => {
    try {
      setLoading(true);
      const data = await listAccessPolicies();
      setPolicies(data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch policies');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchPolicies();
  }, [fetchPolicies]);

  // open editor when route contains policyId or 'new'
  useEffect(() => {
    if (!policyId) {
      setIsEditorOpen(false);
      setSelectedPolicy(null);
      return;
    }
    if (policyId === 'new') {
      setSelectedPolicy(null);
      setIsEditorOpen(true);
      return;
    }
    const p = policies.find((x) => x.id === policyId || x.policy_id === policyId) || null;
    setSelectedPolicy(p);
    setIsEditorOpen(true);
  }, [policyId, policies]);

  const handleAddNew = () => {
    setSimulationResult(null);
    void navigate('/policy-management/new');
  };

  const handleEdit = (policy: AccessControlPolicy) => {
    setSimulationResult(null);
    void navigate(`/policy-management/${policy.id || policy.policy_id}`);
  };

  // handleSave no longer needed; EditPolicy will call onSaved to refresh

  const handleSimulate = async (policyToSimulate: AccessControlPolicy) => {
    setSimulationResult(null);
    try {
      const result = await simulatePolicyChange(policyToSimulate);
      setSimulationResult(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Simulation failed');
    }
  };

  if (loading) return <CircularProgress />;
  if (error) return <Alert severity="error">{error}</Alert>;

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h4">Policy Engine</Typography>
        <Button variant="contained" startIcon={<AddIcon />} onClick={handleAddNew}>
          New Policy
        </Button>
      </Box>
      <Grid container spacing={3}>
        <Grid item xs={12} md={isEditorOpen ? 6 : 12}>
          <Paper sx={{ p: 2 }}>
            <PolicyList policies={policies} onEdit={handleEdit} />
          </Paper>
        </Grid>
        {isEditorOpen && (
          <Grid item xs={12} md={6}>
            <Paper sx={{ p: 2 }}>
              <EditPolicy
                id={selectedPolicy?.id}
                onSaved={(updated: AccessControlPolicy) => {
                  navigate('/policy-management');
                  setIsEditorOpen(false);
                  setSelectedPolicy(updated);
                  setSimulationResult(null);
                  fetchPolicies();
                }}
                onCancel={() => {
                  navigate('/policy-management');
                  setIsEditorOpen(false);
                  setSelectedPolicy(null);
                  setSimulationResult(null);
                }}
                onSimulate={async (policy: AccessControlPolicy) => {
                  await handleSimulate(policy);
                }}
              />
              {simulationResult && <PolicySimulationResultViewer result={simulationResult} />}
            </Paper>
          </Grid>
        )}
      </Grid>
    </Box>
  );
}