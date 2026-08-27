import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  Stack,
  Chip,
  Button,
  TextField,
  MenuItem,
  Divider,
  Grid,
  Tooltip
} from '@mui/material';
import {
  AccountTree as GraphIcon,
  Storage as DataIcon,
  AutoAwesome as AIIcon,
  Link as LinkIcon,
  ArrowForward as ArrowIcon,
  CheckCircle as ValidIcon
} from '@mui/icons-material';

export interface BORelation {
  sourceBO: string;
  targetBO: string;
  edgeType: 'JOINS_TO' | 'FEEDS_INTO' | 'USES_INPUT' | 'MAPS_TO';
  foreignKey: string;
  confidence: number;
}

export interface SmartDataBindingProps {
  pageId?: string;
  selectedComponentId?: string | null;
  onApplyBinding?: (binding: { boKey: string; fieldKeys: string[]; joinPath: string }) => void;
}

export const SmartDataBindingStudio: React.FC<SmartDataBindingProps> = ({
  selectedComponentId,
  onApplyBinding
}) => {
  const [selectedBO, setSelectedBO] = useState('trade_order');
  const [selectedFields, setSelectedFields] = useState<string[]>(['order_id', 'execution_price', 'quantity']);

  const businessObjects = [
    { key: 'trade_order', label: 'Trade Order (oms.trade_order)' },
    { key: 'account', label: 'Account (oms.account)' },
    { key: 'position', label: 'Position (oms.position)' },
    { key: 'customer', label: 'Customer (master.customer)' }
  ];

  const graphRelations: BORelation[] = [
    {
      sourceBO: 'trade_order',
      targetBO: 'account',
      edgeType: 'JOINS_TO',
      foreignKey: 'account_id',
      confidence: 0.99
    },
    {
      sourceBO: 'trade_order',
      targetBO: 'position',
      edgeType: 'FEEDS_INTO',
      foreignKey: 'security_id',
      confidence: 0.98
    },
    {
      sourceBO: 'account',
      targetBO: 'customer',
      edgeType: 'USES_INPUT',
      foreignKey: 'customer_id',
      confidence: 0.95
    }
  ];

  const availableFields: Record<string, Array<{ key: string; label: string; role: 'DIM' | 'MEASURE' | 'KEY' }>> = {
    trade_order: [
      { key: 'order_id', label: 'Order ID', role: 'KEY' },
      { key: 'execution_price', label: 'Execution Price', role: 'MEASURE' },
      { key: 'quantity', label: 'Quantity', role: 'MEASURE' },
      { key: 'status', label: 'Order Status', role: 'DIM' },
      { key: 'order_date', label: 'Order Date', role: 'DIM' }
    ],
    account: [
      { key: 'account_number', label: 'Account Number', role: 'KEY' },
      { key: 'account_name', label: 'Account Name', role: 'DIM' },
      { key: 'total_aum', label: 'Total AUM', role: 'MEASURE' }
    ]
  };

  const toggleField = (fKey: string) => {
    setSelectedFields(prev =>
      prev.includes(fKey) ? prev.filter(k => k !== fKey) : [...prev, fKey]
    );
  };

  const handleApply = () => {
    if (onApplyBinding) {
      onApplyBinding({
        boKey: selectedBO,
        fieldKeys: selectedFields,
        joinPath: `${selectedBO} -> account (via account_id)`
      });
    }
  };

  return (
    <Box sx={{ p: 2, height: '100%', overflowY: 'auto', bgcolor: '#071526', color: '#F8FAFC' }}>
      {/* Header */}
      <Box display="flex" justifyContent="space-between" alignItems="center" pb={1.5} mb={2} borderBottom="1px solid #1E293B">
        <Stack direction="row" spacing={1} alignItems="center">
          <DataIcon sx={{ color: '#00D4FF', fontSize: 20 }} />
          <Typography variant="subtitle2" sx={{ fontWeight: 700, fontSize: 13, textTransform: 'uppercase' }}>
            Semantic BO & Graph Data Binding
          </Typography>
        </Stack>
        <Chip
          icon={<ValidIcon sx={{ fontSize: 14, color: '#10B981 !important' }} />}
          label="Graph Active"
          size="small"
          sx={{ bgcolor: '#064E3B', color: '#34D399', fontSize: 10, fontWeight: 700 }}
        />
      </Box>

      {/* Target Component */}
      <Paper sx={{ p: 1.5, mb: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
        <Typography variant="caption" sx={{ color: '#94A3B8', display: 'block', fontSize: 10 }}>Bound UI Component</Typography>
        <Typography variant="body2" sx={{ fontWeight: 700, color: '#38BDF8', fontSize: 12 }}>
          {selectedComponentId ? `Component [${selectedComponentId}]` : 'Selected Canvas Node'}
        </Typography>
      </Paper>

      {/* Select Driving BO */}
      <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 700, mb: 0.5, display: 'block' }}>
        Driving Business Object
      </Typography>
      <TextField
        select
        fullWidth
        size="small"
        value={selectedBO}
        onChange={e => setSelectedBO(e.target.value)}
        sx={{
          mb: 2,
          bgcolor: '#0B1E36',
          borderRadius: 1,
          '& .MuiInputBase-input': { color: '#F8FAFC', fontSize: 12, fontWeight: 600 }
        }}
      >
        {businessObjects.map(bo => (
          <MenuItem key={bo.key} value={bo.key} sx={{ fontSize: 12 }}>
            {bo.label}
          </MenuItem>
        ))}
      </TextField>

      {/* Graph Relationship Traversal (Topological Joins) */}
      <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 700, mb: 1, display: 'block' }}>
        Traversable Graph Relationships (`JOINS_TO`, `FEEDS_INTO`)
      </Typography>
      <Stack spacing={1} mb={2}>
        {graphRelations
          .filter(r => r.sourceBO === selectedBO || r.targetBO === selectedBO)
          .map((r, idx) => (
            <Paper
              key={idx}
              sx={{
                p: 1,
                bgcolor: '#0E1E38',
                border: '1px solid #1E293B',
                borderRadius: 1,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between'
              }}
            >
              <Stack direction="row" spacing={1} alignItems="center">
                <GraphIcon sx={{ color: '#38BDF8', fontSize: 16 }} />
                <Box>
                  <Typography variant="body2" sx={{ fontWeight: 700, fontSize: 11, color: '#F8FAFC' }}>
                    {r.sourceBO} <ArrowIcon sx={{ fontSize: 10, mx: 0.5 }} /> {r.targetBO}
                  </Typography>
                  <Typography variant="caption" sx={{ color: '#94A3B8', fontSize: 9 }}>
                    via {r.foreignKey} ({r.edgeType})
                  </Typography>
                </Box>
              </Stack>
              <Chip
                label={`${(r.confidence * 100).toFixed(0)}% Match`}
                size="small"
                sx={{ bgcolor: 'rgba(56, 189, 248, 0.1)', color: '#38BDF8', fontSize: 9, height: 18 }}
              />
            </Paper>
          ))}
      </Stack>

      {/* Available Fields Selection */}
      <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 700, mb: 1, display: 'block' }}>
        Select Projected Semantic Fields
      </Typography>
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1, mb: 2 }}>
        {(availableFields[selectedBO] || []).map(f => {
          const isSelected = selectedFields.includes(f.key);
          return (
            <Chip
              key={f.key}
              label={`${f.label} [${f.role}]`}
              clickable
              onClick={() => toggleField(f.key)}
              size="small"
              sx={{
                bgcolor: isSelected ? 'rgba(0, 212, 255, 0.2)' : '#0B1E36',
                color: isSelected ? '#00D4FF' : '#94A3B8',
                border: isSelected ? '1px solid #00D4FF' : '1px solid #1E293B',
                fontWeight: 600,
                fontSize: 10
              }}
            />
          );
        })}
      </Box>

      {/* Action Button */}
      <Button
        fullWidth
        variant="contained"
        startIcon={<LinkIcon />}
        onClick={handleApply}
        sx={{
          bgcolor: '#0284C7',
          color: '#FFFFFF',
          textTransform: 'none',
          fontSize: 12,
          fontWeight: 700,
          '&:hover': { bgcolor: '#0369A1' }
        }}
      >
        Bind {selectedFields.length} Fields to Component
      </Button>
    </Box>
  );
};

export default SmartDataBindingStudio;
