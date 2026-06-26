// React default import removed (not used as a value)
import { Box, Typography, Paper } from '@mui/material';
import ClaimAwareLineageViewer from '../components/ClaimAwareLineageViewer';

export default function ClaimAwareLineagePage() {
  // In a real app, these would come from URL params or state management
  const assetId = "metric:avg_order_value";
  const userId = "patrick";

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h4" gutterBottom>
        Claim-Aware Lineage Explorer
      </Typography>
      <Typography color="text.secondary" sx={{ mb: 2 }}>
        Visualizing lineage for asset <strong>{assetId}</strong> as user <strong>{userId}</strong>.
      </Typography>
      <Paper sx={{ p: 2, height: '70vh' }}>
        <ClaimAwareLineageViewer assetId={assetId} userId={userId} />
      </Paper>
    </Box>
  );
}