import React, { useState, useEffect } from 'react';
import { Box, Button, FormControl, InputLabel, Select, MenuItem, CircularProgress, Typography } from '@mui/material';
import { LocalizationProvider, DatePicker } from '@mui/x-date-pickers';
import PushPinIcon from '@mui/icons-material/PushPin';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFns';
import { useDrillDown, FilterState } from '../../../contexts/DrillDownContext';
import { format } from 'date-fns';
import { useQuery } from '@tanstack/react-query';
import { apiFetch } from '../../../../lib/apiClient';

const PolicySelect: React.FC<{ value: string; onChange: (value: string) => void }> = ({ value, onChange }) => {
  const { data, isLoading } = useQuery({
    queryKey: ['policies-for-select'],
    queryFn: async () => {
      const res = await apiFetch('/api/rest/policies?order_by=name:asc');
      if (!res.ok) {
        throw new Error(await res.text());
      }
      return res.json();
    },
  });

  return (
    <FormControl size="small" sx={{ m: 1, minWidth: 200 }}>
      <InputLabel>Policy</InputLabel>
      <Select value={value} label="Policy" onChange={(e) => onChange(e.target.value)}>
        {isLoading && <MenuItem disabled><CircularProgress size={20} /></MenuItem>}
        {data?.map((p: any) => (
          <MenuItem key={p.id} value={p.id}>{p.name}</MenuItem>
        ))}
      </Select>
    </FormControl>
  );
};

const VersionSelect: React.FC<{ label: string; policyId: string; value: number; onChange: (value: number) => void }> = ({ label, policyId, value, onChange }) => {
  const { data, isLoading } = useQuery({
    queryKey: ['policy-versions-for-select', policyId],
    queryFn: async () => {
      const res = await apiFetch(`/api/rest/policy-version-history?policy_id=${encodeURIComponent(policyId)}&order_by=version:desc`);
      if (!res.ok) {
        throw new Error(await res.text());
      }
      return res.json();
    },
    enabled: !!policyId,
  });

  return (
    <FormControl size="small" sx={{ m: 1, minWidth: 120 }}>
      <InputLabel>{label}</InputLabel>
      <Select value={value || ''} label={label} onChange={(e) => onChange(Number(e.target.value))}>
        {isLoading && <MenuItem disabled><CircularProgress size={20} /></MenuItem>}
        {data?.map((v: any) => (
          <MenuItem key={v.version} value={v.version}>v{v.version}</MenuItem>
        ))}
      </Select>
    </FormControl>
  );
};

const QuickCompareControls: React.FC<{ onPin: (filters: FilterState) => void }> = ({ onPin }) => {
  const { context, filters, setFilters } = useDrillDown();
  const [localFilters, setLocalFilters] = useState<FilterState | null>(filters);

  useEffect(() => {
    setLocalFilters(filters);
  }, [filters]);

  if (!context || !localFilters) return null;

  const handleApply = () => {
    if (localFilters) {
      setFilters(localFilters);
    }
  };

  const handleFilterChange = (key: keyof FilterState, value: any) => {
    setLocalFilters(prev => prev ? { ...prev, [key]: value } : null);
  };

  const handleDateChange = (key: 'from' | 'to', value: any) => {
    const date: Date | null = value instanceof Date ? value : (value?.toDate ? value.toDate() : null);
    if (date) {
        setLocalFilters(prev => prev ? {
            ...prev,
            dateRange: {
                ...prev.dateRange!,
                [key]: format(date, 'yyyy-MM-dd')
            }
        } : null);
    }
  };

  return (
    <Box sx={{ display: 'flex', alignItems: 'center', p: 1, borderBottom: 1, borderColor: 'divider', flexWrap: 'wrap', backgroundColor: 'action.hover' }}>
      <Typography variant="subtitle2" sx={{ mr: 1, ml: 1, color: 'text.secondary' }}>Quick Compare:</Typography>
      {context === 'policy_compare' && localFilters.policyId && (
        <>
          <PolicySelect value={localFilters.policyId} onChange={(v) => handleFilterChange('policyId', v)} />
          <VersionSelect label="Version A" policyId={localFilters.policyId} value={localFilters.versionA!} onChange={(v) => handleFilterChange('versionA', v)} />
          <VersionSelect label="Version B" policyId={localFilters.policyId} value={localFilters.versionB!} onChange={(v) => handleFilterChange('versionB', v)} />
        </>
      )}

      {(context === 'historical' || context === 'policy_compare') && localFilters.dateRange && (
        <LocalizationProvider dateAdapter={AdapterDateFns}>
            <DatePicker
                label="From Date"
                value={new Date(localFilters.dateRange.from)}
                onChange={(date) => handleDateChange('from', date)}
                slotProps={{ textField: { size: 'small', sx: { m: 1, width: 180 } } }}
            />
            <DatePicker
                label="To Date"
                value={new Date(localFilters.dateRange.to)}
                onChange={(date) => handleDateChange('to', date)}
                slotProps={{ textField: { size: 'small', sx: { m: 1, width: 180 } } }}
            />
        </LocalizationProvider>
      )}

      <Button variant="contained" onClick={handleApply} sx={{ m: 1 }}>Apply</Button>
      <Button variant="outlined" onClick={() => onPin(localFilters!)} sx={{ m: 1 }} startIcon={<PushPinIcon />}>
        Pin
      </Button>
    </Box>
  );
};

export default QuickCompareControls;
