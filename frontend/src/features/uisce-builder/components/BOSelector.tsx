import React, { useState, useEffect } from 'react';
import {
  Box,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Typography,
  Chip,
  CircularProgress,
  Tooltip,
} from '@mui/material';
import BusinessIcon from '@mui/icons-material/Business';
import axios from '@/utils/axiosClient';

interface BusinessObject {
  id: string;
  key: string;
  display_name: string;
  description?: string;
  fields?: BOField[];
}

interface BOField {
  name: string;
  label: string;
  data_type: string;
  required?: boolean;
}

interface BOSelectorProps {
  selectedBOId: string | null;
  onSelectBO: (bo: BusinessObject | null) => void;
}

const BOSelector: React.FC<BOSelectorProps> = ({ selectedBOId, onSelectBO }) => {
  const [businessObjects, setBusinessObjects] = useState<BusinessObject[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchBusinessObjects();
  }, []);

  const fetchBusinessObjects = async () => {
    try {
      setLoading(true);
      const response = await axios.get('/api/business-objects');
      setBusinessObjects(response.data || []);
      setError(null);
    } catch (err: any) {
      setError('Failed to load business objects');
      // Mock data for demo purposes
      setBusinessObjects([
        { id: 'bo-1', key: 'trade', display_name: 'Trade', description: 'Securities trade transaction', fields: [
          { name: 'trade_id', label: 'Trade ID', data_type: 'string', required: true },
          { name: 'amount', label: 'Amount', data_type: 'number', required: true },
          { name: 'counterparty', label: 'Counterparty', data_type: 'string', required: true },
          { name: 'trade_date', label: 'Trade Date', data_type: 'date', required: true },
          { name: 'settlement_date', label: 'Settlement Date', data_type: 'date' },
          { name: 'asset_class', label: 'Asset Class', data_type: 'string' },
        ]},
        { id: 'bo-2', key: 'account', display_name: 'Account', description: 'Client investment account', fields: [
          { name: 'account_id', label: 'Account ID', data_type: 'string', required: true },
          { name: 'client_name', label: 'Client Name', data_type: 'string', required: true },
          { name: 'account_type', label: 'Account Type', data_type: 'string' },
          { name: 'balance', label: 'Balance', data_type: 'number' },
        ]},
        { id: 'bo-3', key: 'client', display_name: 'Client', description: 'Client profile', fields: [
          { name: 'client_id', label: 'Client ID', data_type: 'string', required: true },
          { name: 'name', label: 'Name', data_type: 'string', required: true },
          { name: 'risk_profile', label: 'Risk Profile', data_type: 'string' },
          { name: 'kyc_status', label: 'KYC Status', data_type: 'string' },
        ]},
      ]);
    } finally {
      setLoading(false);
    }
  };

  const selectedBO = businessObjects.find(bo => bo.id === selectedBOId) || null;

  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
      <BusinessIcon sx={{ color: 'primary.main' }} />
      <FormControl size="small" sx={{ minWidth: 200 }}>
        <InputLabel id="bo-selector-label">Target Business Object</InputLabel>
        <Select
          labelId="bo-selector-label"
          value={selectedBOId || ''}
          label="Target Business Object"
          onChange={(e) => {
            const bo = businessObjects.find(b => b.id === e.target.value) || null;
            onSelectBO(bo);
          }}
          disabled={loading}
          startAdornment={loading ? <CircularProgress size={16} sx={{ mr: 1 }} /> : null}
        >
          <MenuItem value="">
            <em>None (Generic)</em>
          </MenuItem>
          {businessObjects.map((bo) => (
            <MenuItem key={bo.id} value={bo.id}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <Typography variant="body2">{bo.display_name}</Typography>
                {bo.fields && (
                  <Chip 
                    size="small" 
                    label={`${bo.fields.length} fields`} 
                    sx={{ fontSize: '0.65rem', height: 18 }} 
                  />
                )}
              </Box>
            </MenuItem>
          ))}
        </Select>
      </FormControl>
      
      {selectedBO && (
        <Tooltip title={selectedBO.description || ''}>
          <Typography variant="caption" color="text.secondary">
            {selectedBO.fields?.length || 0} fields available
          </Typography>
        </Tooltip>
      )}
    </Box>
  );
};

export default BOSelector;
export type { BusinessObject, BOField };
