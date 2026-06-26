import type { FC, ReactElement } from 'react';
import { Box, Typography, Paper, Chip, List, ListItem, ListItemText, Alert } from '@mui/material';
import PeopleIcon from '@mui/icons-material/People';
import ArticleIcon from '@mui/icons-material/Article';
import GppMaybeIcon from '@mui/icons-material/GppMaybe';
import { PolicySimulationResult } from '../../../types';

interface PolicySimulationResultViewerProps {
  result: PolicySimulationResult;
}

const ImpactCard: FC<{ title: string; count: number; icon: ReactElement }> = ({ title, count, icon }) => (
  <Paper variant="outlined" sx={{ p: 2, display: 'flex', alignItems: 'center', gap: 2 }}>
    {icon}
    <Box>
      <Typography variant="h6">{count}</Typography>
      <Typography variant="body2" color="text.secondary">{title}</Typography>
    </Box>
  </Paper>
);

export default function PolicySimulationResultViewer({ result }: PolicySimulationResultViewerProps) {
  const { affected_claims, affected_users, affected_assets, risk_flags } = result;

  return (
    <Box sx={{ mt: 3 }}>
      <Typography variant="h5" gutterBottom>Simulation Impact</Typography>
      <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: 2, mb: 3 }}>
        <ImpactCard title="Affected Users" count={affected_users.length} icon={<PeopleIcon color="primary" />} />
        <ImpactCard title="Affected Assets" count={affected_assets.length} icon={<ArticleIcon color="primary" />} />
      </Box>

      {risk_flags.length > 0 && (
        <Box sx={{ mb: 3 }}>
          <Typography variant="h6" gutterBottom>Risk Summary</Typography>
          <Alert severity="warning" icon={<GppMaybeIcon />}>
            <List dense>
              {risk_flags.map((flag, index) => (
                <ListItem key={index} disableGutters>
                  <ListItemText primary={flag} />
                </ListItem>
              ))}
            </List>
          </Alert>
        </Box>
      )}

      <Box>
        <Typography variant="h6" gutterBottom>Claim Changes</Typography>
        <Paper variant="outlined" sx={{ p: 2, display: 'flex', justifyContent: 'space-around' }}>
          <Chip label={`Added: ${affected_claims.added}`} color="success" />
          <Chip label={`Modified: ${affected_claims.modified}`} color="warning" />
          <Chip label={`Removed: ${affected_claims.removed}`} color="error" />
        </Paper>
      </Box>
    </Box>
  );
}