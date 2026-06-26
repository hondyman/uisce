import { useState, useEffect, useCallback } from 'react';
import { devError } from './utils/devLogger';
import { Box, Typography, Paper, TextField, Button, CircularProgress, Alert, Chip, Select, MenuItem, FormControl, InputLabel } from '@mui/material';
import { listSemanticViews, evaluateAccess, simulateAccess, getDecisionTrace } from './api';
import { SemanticViewMeta, EvaluateAccessResponse, SimulatedClaim, AccessDecisionTrace } from './types';

// Sub-components for clarity

const AssetSelector = ({ views, selected, onSelect }: { views: SemanticViewMeta[], selected: string, onSelect: (id: string) => void }) => (
  <FormControl fullWidth>
    <InputLabel>Asset to Inspect</InputLabel>
    <Select value={selected} label="Asset to Inspect" onChange={(e) => onSelect(e.target.value)}>
      {views.map(v => <MenuItem key={v.id} value={v.id}>{v.name}</MenuItem>)}
    </Select>
  </FormControl>
);

const AccessResultPanel = ({ title, response }: { title: string, response: EvaluateAccessResponse | null }) => {
  if (!response) return null;
  return (
    <Paper variant="outlined" sx={{ p: 2, mt: 2, borderColor: response.decision === 'allow' ? 'success.main' : 'error.main' }}>
      <Typography variant="h6">{title}</Typography>
      <Chip label={response.decision.toUpperCase()} color={response.decision === 'allow' ? 'success' : 'error'} sx={{ mr: 1 }} />
      <Typography variant="body2" component="span">({response.reason})</Typography>
    </Paper>
  );
};

const SimulationPanel = ({ assetId, onSimulate, loading }: { assetId: string, onSimulate: (claims: SimulatedClaim[]) => void, loading: boolean }) => {
  const [claims, setClaims] = useState<SimulatedClaim[]>([]);

  const handleAddClaim = () => {
    if (assetId) {
      setClaims([...claims, { model_id: assetId, permission: 'read' }]);
    }
  };

  const handleRun = () => {
    onSimulate(claims);
  };

  return (
    <Paper variant="outlined" sx={{ p: 2, mt: 2 }}>
      <Typography variant="h6">Simulation Sandbox</Typography>
      <Typography color="text.secondary" sx={{ mb: 2 }}>Add temporary claims to see how access would change.</Typography>
      {claims.map((claim, index) => (
        <Box key={index} sx={{ display: 'flex', gap: 1, mb: 1 }}>
          <TextField value={claim.model_id} label="Model ID" disabled size="small" />
          <TextField value={claim.permission} label="Permission" size="small" />
        </Box>
      ))}
      <Button onClick={handleAddClaim} disabled={!assetId}>+ Add Simulated Claim</Button>
      <Button onClick={handleRun} variant="contained" sx={{ ml: 2 }} disabled={loading}>
        {loading ? <CircularProgress size={24} /> : 'Run Simulation'}
      </Button>
    </Paper>
  );
};

const DecisionTraceViewer = ({ decisionId }: { decisionId: string | null }) => {
  const [trace, setTrace] = useState<AccessDecisionTrace | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!decisionId) {
      setTrace(null);
      return;
    }
    const fetchTrace = async () => {
      setLoading(true);
      try {
        const data = await getDecisionTrace(decisionId);
        setTrace(data);
      } catch (e) {
        devError(e);
        setTrace(null);
      } finally {
        setLoading(false);
      }
    };
    fetchTrace();
  }, [decisionId]);

  if (!decisionId) return null;
  if (loading) return <CircularProgress size={24} sx={{ mt: 2 }} />;
  if (!trace) return <Alert severity="warning" sx={{ mt: 2 }}>Could not load decision trace.</Alert>;

  return (
    <Paper variant="outlined" sx={{ p: 2, mt: 2, fontFamily: 'monospace', whiteSpace: 'pre-wrap', backgroundColor: '#f5f5f5', maxHeight: 400, overflow: 'auto' }}>
      <Typography variant="h6" sx={{ fontFamily: 'sans-serif' }}>Decision Trace</Typography>
      <pre>{JSON.stringify(trace, null, 2)}</pre>
    </Paper>
  );
};

export default function AccessDebuggerPage() {
  const [views, setViews] = useState<SemanticViewMeta[]>([]);
  const [selectedAssetId, setSelectedAssetId] = useState<string>('');
  const [loading, setLoading] = useState(false);
  const [currentAccess, setCurrentAccess] = useState<EvaluateAccessResponse | null>(null);
  const [simulatedAccess, setSimulatedAccess] = useState<EvaluateAccessResponse | null>(null);

  const userId = 'patrick';
  const tenantId = 'acme_corp';

  useEffect(() => {
    listSemanticViews('mock-datasource-id').then(setViews).catch(err => devError('Failed to load semantic views for AccessDebuggerPage:', err));
  }, []);

  const handleInspect = useCallback(async () => {
    if (!selectedAssetId) return;
    setLoading(true);
    setCurrentAccess(null);
    setSimulatedAccess(null);
    try {
      const response = await evaluateAccess({ user_id: userId, tenant_id: tenantId, asset_id: selectedAssetId, action: 'read' });
      setCurrentAccess(response);
    } catch (e) { devError(e); } finally { setLoading(false); }
  }, [selectedAssetId]);

  const handleSimulate = async (simulatedClaims: SimulatedClaim[]) => {
    if (!selectedAssetId) return;
    setLoading(true);
    setSimulatedAccess(null);
    try {
      const response = await simulateAccess({ user_id: userId, tenant_id: tenantId, asset_id: selectedAssetId, action: 'read', simulated_claims: simulatedClaims });
      setSimulatedAccess(response);
    } catch (e) { devError(e); } finally { setLoading(false); }
  };

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h4" gutterBottom>Self-Service Access Debugger</Typography>
      <Paper sx={{ p: 2 }}>
        <Box sx={{ display: 'flex', gap: 2, alignItems: 'center' }}>
          <AssetSelector views={views} selected={selectedAssetId} onSelect={setSelectedAssetId} />
          <Button variant="contained" onClick={handleInspect} disabled={!selectedAssetId || loading}>{loading && !simulatedAccess ? <CircularProgress size={24} /> : 'Inspect Access'}</Button>
        </Box>
        {currentAccess && <AccessResultPanel title="Current Access" response={currentAccess} />}
        {currentAccess && <DecisionTraceViewer decisionId={currentAccess.decision_id} />}
        {currentAccess && <SimulationPanel assetId={selectedAssetId} onSimulate={handleSimulate} loading={loading && !!simulatedAccess} />}
        {simulatedAccess && <AccessResultPanel title="Simulated Access" response={simulatedAccess} />}
        {simulatedAccess && <DecisionTraceViewer decisionId={simulatedAccess.decision_id} />}
      </Paper>
    </Box>
  );
}