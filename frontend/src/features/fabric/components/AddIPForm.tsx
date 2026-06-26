import React, { useState } from 'react';
import {
  Stack,
  TextField,
  Button,
  Box,
  Alert
} from '@mui/material';
import { isValidIpOrWildcard } from '../utils/ipValidation';

interface AddIPFormProps {
  _tenantId: string;
  onSubmit: (ipAddress: string, label: string, description: string) => void;
  onCancel: () => void;
}

const AddIPForm: React.FC<AddIPFormProps> = ({ _tenantId, onSubmit, onCancel }) => {
  const [ipAddress, setIpAddress] = useState('');
  const [label, setLabel] = useState('');
  const [description, setDescription] = useState('');
  const [error, setError] = useState('');

  const handleSubmit = () => {
    setError('');

    if (!ipAddress.trim()) {
      setError('IP address is required');
      return;
    }

    if (!isValidIpOrWildcard(ipAddress)) {
      setError('Please enter a valid IP address or wildcard pattern (e.g., 192.168.1.0/24, 10.0.0.*)');
      return;
    }

    onSubmit(ipAddress, label, description);
  };

  return (
    <Stack spacing={2}>
      {error && <Alert severity="error">{error}</Alert>}
      
      <TextField
        fullWidth
        label="IP Address"
        placeholder="e.g., 192.168.1.100 or 10.0.0.*/24"
        value={ipAddress}
        onChange={(e) => setIpAddress(e.target.value)}
        variant="outlined"
        size="small"
        error={Boolean(error) && !ipAddress.trim()}
        helperText="IPv4, IPv6, or CIDR format"
      />

      <TextField
        fullWidth
        label="Label (Optional)"
        placeholder="e.g., Office Network, VPN Gateway"
        value={label}
        onChange={(e) => setLabel(e.target.value)}
        variant="outlined"
        size="small"
      />

      <TextField
        fullWidth
        label="Description (Optional)"
        placeholder="Additional notes about this IP"
        value={description}
        onChange={(e) => setDescription(e.target.value)}
        variant="outlined"
        size="small"
        multiline
        rows={3}
      />

      <Box sx={{ display: 'flex', gap: 1, justifyContent: 'flex-end', pt: 2 }}>
        <Button onClick={onCancel} variant="outlined">
          Cancel
        </Button>
        <Button 
          onClick={handleSubmit}
          variant="contained"
          disabled={!ipAddress.trim()}
        >
          Add IP Address
        </Button>
      </Box>
    </Stack>
  );
};

export default AddIPForm;
