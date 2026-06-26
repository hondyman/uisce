import React, { useState, useEffect } from 'react';
import {
  Autocomplete,
  TextField,
  Box,
  Typography,
  Chip,
  CircularProgress,
  Paper,
  Stack,
} from '@mui/material';
import { Business as BusinessIcon } from '@mui/icons-material';

export interface BusinessObject {
  id: string;
  displayName: string;
  description?: string;
  category?: string;
  icon?: string;
}

interface BusinessObjectSelectorProps {
  value: string | null;
  onChange: (businessObjectId: string | null) => void;
  label?: string;
  placeholder?: string;
  required?: boolean;
  disabled?: boolean;
  helperText?: string;
  error?: boolean;
}

export const BusinessObjectSelector: React.FC<BusinessObjectSelectorProps> = ({
  value,
  onChange,
  label = 'Data Type',
  placeholder = 'Search for a data type...',
  required = false,
  disabled = false,
  helperText,
  error = false,
}) => {
  const [businessObjects, setBusinessObjects] = useState<BusinessObject[]>([]);
  const [loading, setLoading] = useState(false);
  const [inputValue, setInputValue] = useState('');

  // Fetch business objects from API
  useEffect(() => {
    const fetchBusinessObjects = async () => {
      setLoading(true);
      try {
        // TODO: Replace with actual API call
        const response = await fetch('/api/business-objects');
        if (response.ok) {
          const data = await response.json();
          setBusinessObjects(data);
        } else {
          // Fallback to mock data
          setBusinessObjects([
            { id: 'bo:portfolio', displayName: 'Portfolio', description: 'Investment portfolios', category: 'Investments' },
            { id: 'bo:client', displayName: 'Client', description: 'Client information', category: 'CRM' },
            { id: 'bo:account', displayName: 'Account', description: 'Financial accounts', category: 'Finance' },
            { id: 'bo:transaction', displayName: 'Transaction', description: 'Financial transactions', category: 'Finance' },
            { id: 'bo:holding', displayName: 'Holding', description: 'Investment holdings', category: 'Investments' },
            { id: 'bo:security', displayName: 'Security', description: 'Financial securities', category: 'Investments' },
            { id: 'bo:advisor', displayName: 'Advisor', description: 'Financial advisors', category: 'CRM' },
          ]);
        }
      } catch (error) {
        console.error('Failed to fetch business objects:', error);
        // Use mock data on error
        setBusinessObjects([
          { id: 'bo:portfolio', displayName: 'Portfolio', description: 'Investment portfolios', category: 'Investments' },
          { id: 'bo:client', displayName: 'Client', description: 'Client information', category: 'CRM' },
        ]);
      } finally {
        setLoading(false);
      }
    };

    void fetchBusinessObjects();
  }, []);

  const selectedBO = businessObjects.find((bo) => bo.id === value) || null;

  return (
    <Autocomplete
      value={selectedBO}
      onChange={(_, newValue) => onChange(newValue?.id || null)}
      inputValue={inputValue}
      onInputChange={(_, newInputValue) => setInputValue(newInputValue)}
      options={businessObjects}
      groupBy={(option) => option.category || 'Other'}
      getOptionLabel={(option) => option.displayName}
      loading={loading}
      disabled={disabled}
      renderInput={(params) => (
        <TextField
          {...params}
          label={label}
          placeholder={placeholder}
          required={required}
          error={error}
          helperText={helperText}
          InputProps={{
            ...params.InputProps,
            startAdornment: (
              <>
                <BusinessIcon sx={{ mr: 1, color: 'action.active' }} />
                {params.InputProps.startAdornment}
              </>
            ),
            endAdornment: (
              <>
                {loading ? <CircularProgress color="inherit" size={20} /> : null}
                {params.InputProps.endAdornment}
              </>
            ),
          }}
        />
      )}
      renderOption={(props, option) => (
        <li {...props}>
          <Stack sx={{ width: '100%' }}>
            <Stack direction="row" alignItems="center" spacing={1}>
              <Typography variant="body2" sx={{ fontWeight: 500 }}>
                {option.displayName}
              </Typography>
              {option.category && (
                <Chip label={option.category} size="small" variant="outlined" />
              )}
            </Stack>
            {option.description && (
              <Typography variant="caption" color="text.secondary">
                {option.description}
              </Typography>
            )}
          </Stack>
        </li>
      )}
      renderTags={(value, getTagProps) =>
        value.map((option, index) => (
          <Chip
            {...getTagProps({ index })}
            key={option.id}
            icon={<BusinessIcon />}
            label={option.displayName}
            size="small"
          />
        ))
      }
      PaperComponent={({ children }) => (
        <Paper elevation={8}>{children}</Paper>
      )}
    />
  );
};

export default BusinessObjectSelector;
