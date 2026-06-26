import { useState, useEffect } from 'react';
import { Box, Typography, Tabs, Tab, Paper, CircularProgress as _CircularProgress } from '@mui/material';
import { listClaimConflicts, detectClaimDrift, listClaimBundles } from './api';
import type { ClaimConflict, SemanticModelClaim, ClaimBundle } from './types';
import AutomationPanel from './AutomationPanel';

// Mock Panels - in a real app, these would be fleshed-out components
const ConflictsPanel: React.FC = () => {
  const [conflicts, setConflicts] = useState<ClaimConflict[]>([]);
  useEffect(() => { listClaimConflicts('patrick').then(setConflicts); }, []);
  return <Paper sx={{ p: 2, mt: 2 }}>{conflicts.length > 0 ? `${conflicts.length} active conflict(s) detected.` : 'No active conflicts detected.'}</Paper>;
};

const DriftPanel: React.FC = () => {
  const [drifted, setDrifted] = useState<SemanticModelClaim[]>([]);
  useEffect(() => { detectClaimDrift().then(setDrifted); }, []);
  return <Paper sx={{ p: 2, mt: 2 }}>{drifted.length > 0 ? `${drifted.length} claim(s) showing usage drift.` : 'No claim drift detected.'}</Paper>;
};

const BundlesPanel: React.FC = () => {
  const [bundles, setBundles] = useState<ClaimBundle[]>([]);
  useEffect(() => { listClaimBundles().then(setBundles); }, []);
  return (
    <Paper sx={{ p: 2, mt: 2 }}>
      <Typography variant="h6">Available Claim Bundles</Typography>
      <ul>
        {bundles.map(b => <li key={b.id}>{b.name}: {b.description}</li>)}
      </ul>
    </Paper>
  );
};

const ExplainAccessPanel: React.FC = () => {
  // This would have a UI to select a user and tenant
  // and then call `getEffectiveClaims`
  return (
    <Paper sx={{ p: 2, mt: 2 }}>
      <Typography variant="h6">Explain Access</Typography>
      <Typography color="text.secondary">
        Select a user and tenant to see a fully resolved list of their permissions, including source, conflicts, and drift status.
      </Typography>
      {/* UI to select user/tenant and display results would go here */}
    </Paper>
  );
};


export default function AccessIntelligenceDashboard() {
  const [activeTab, setActiveTab] = useState(0);

  const handleTabChange = (_event: React.SyntheticEvent, newValue: number) => {
    setActiveTab(newValue);
  };

  const renderTabContent = () => {
    switch (activeTab) {
      case 0:
        return <ExplainAccessPanel />;
      case 1:
        return <ConflictsPanel />;
      case 2:
        return <DriftPanel />;
      case 3:
        return <BundlesPanel />;
      case 4:
        return <AutomationPanel />;
      default:
        return null;
    }
  };

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h4" gutterBottom>Access Intelligence Dashboard</Typography>
      <Typography color="text.secondary" sx={{ mb: 3 }}>
        A unified dashboard for managing claim conflicts, bundles, drift, and tenant isolation.
      </Typography>

      <Box sx={{ borderBottom: 1, borderColor: 'divider', mb: 3 }}>
        <Tabs value={activeTab} onChange={handleTabChange}>
          <Tab label="Explain Access" />
          <Tab label="Conflicts" />
          <Tab label="Drift" />
          <Tab label="Bundles" />
          <Tab label="Automation" />
        </Tabs>
      </Box>

      {renderTabContent()}
    </Box>
  );
}