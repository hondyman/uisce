// React default import removed (not used as a value)
import { Box, Typography, Button } from '@mui/material';
import { AccessControlPolicy } from '../../../types';

interface PolicyEditorProps {
  policy: AccessControlPolicy | null;
  simulating: boolean;
  onSave: () => void;
  onCancel: () => void;
  onSimulate: (policy: AccessControlPolicy) => void;
}

export default function PolicyEditor({ policy, simulating, onSave, onCancel, onSimulate }: PolicyEditorProps) {
  return (
    <Box>
      <Typography variant="h6" gutterBottom>
        {policy ? `Edit Policy: ${policy.policy_id}` : 'Create New Policy'}
      </Typography>
      <Typography color="text.secondary">
        Policy editor form will be here. This would include fields for scope, role, permissions, duration, and renewal conditions.
      </Typography>
      <Box sx={{ mt: 2, display: 'flex', gap: 1 }}>
        <Button variant="outlined" color="secondary" onClick={() => onSimulate(policy!)} disabled={!policy || simulating}>{simulating ? 'Simulating...' : 'Simulate'}</Button>
        <Button variant="contained" onClick={onSave} disabled={simulating}>Save</Button>
        <Button variant="outlined" onClick={onCancel}>Cancel</Button>
      </Box>
    </Box>
  );
}