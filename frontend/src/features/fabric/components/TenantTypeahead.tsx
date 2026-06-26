import React, { useState, useEffect, useMemo } from 'react';
import {
  Autocomplete,
  TextField,
  Chip,
  Box,
  Typography,
  CircularProgress,
  ListItemText,
  Checkbox
} from '@mui/material';
import {
  Business as BusinessIcon,
  Public as PublicIcon
} from '@mui/icons-material';
import { Tenant, ALL_TENANTS_ID, ALL_TENANTS_DISPLAY_NAME } from '../types/ipWhitelist';

interface TenantTypeaheadProps {
  label?: string;
  value: string[];
  onChange: (tenantIds: string[]) => void;
  tenants: Tenant[];
  loading?: boolean;
  multiple?: boolean;
  placeholder?: string;
  helperText?: string;
  error?: boolean;
  disabled?: boolean;
  allowAllTenants?: boolean;
  size?: 'small' | 'medium';
  fullWidth?: boolean;
}

const TenantTypeahead: React.FC<TenantTypeaheadProps> = ({
  label = 'Tenants',
  value,
  onChange,
  tenants,
  loading = false,
  multiple = true,
  placeholder = 'Search and select tenants...',
  helperText,
  error = false,
  disabled = false,
  allowAllTenants = true,
  size = 'medium',
  fullWidth = true
}) => {
  const [inputValue, setInputValue] = useState('');
  const [filteredTenants, setFilteredTenants] = useState<Tenant[]>([]);

  // Combine options with a virtual "All Tenants" option when allowed
  const allOptions = useMemo(() => {
    if (!allowAllTenants) return tenants;
    const allTenantsOption: Tenant = {
      id: ALL_TENANTS_ID,
      displayName: ALL_TENANTS_DISPLAY_NAME,
      description: 'Apply to all tenants in the system'
    };
    return [allTenantsOption, ...tenants];
  }, [allowAllTenants, tenants]);

  useEffect(() => {
    if (!inputValue.trim()) {
      setFilteredTenants(allOptions);
      return;
    }

    const filtered = allOptions.filter(tenant =>
      tenant.displayName.toLowerCase().includes(inputValue.toLowerCase()) ||
      tenant.description?.toLowerCase().includes(inputValue.toLowerCase()) ||
      tenant.id.toLowerCase().includes(inputValue.toLowerCase())
    );

    setFilteredTenants(filtered);
  }, [inputValue, allOptions]);

  const handleChange = (_: any, selectedTenants: Tenant[] | Tenant | null) => {
    const tenantArray = Array.isArray(selectedTenants) ? selectedTenants : selectedTenants ? [selectedTenants] : [];
    const selectedIds = tenantArray.map(t => t.id);
    
    // If "All Tenants" is selected, clear other selections
    if (selectedIds.includes(ALL_TENANTS_ID)) {
      if (value.includes(ALL_TENANTS_ID)) {
        // If All Tenants was already selected and user selected something else, remove All Tenants
        const filteredIds = selectedIds.filter(id => id !== ALL_TENANTS_ID);
        onChange(filteredIds);
      } else {
        // User selected All Tenants, so only keep that
        onChange([ALL_TENANTS_ID]);
      }
    } else {
      onChange(selectedIds);
    }
  };

  const getSelectedTenants = (): Tenant[] => {
    return allOptions.filter(tenant => value.includes(tenant.id));
  };

  const renderOption = (props: any, tenant: Tenant, { selected }: any) => (
    // Autocomplete expects an <li> with provided props; do not wrap with MUI ListItem to avoid nested <li> and key warnings
    <li {...props}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flex: 1, px: 1, py: 0.5 }}>
        <Checkbox checked={selected} size="small" sx={{ mr: 0.5 }} />
        {tenant.id === ALL_TENANTS_ID ? (
          <PublicIcon fontSize="small" color="primary" />
        ) : (
          <BusinessIcon fontSize="small" />
        )}
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <ListItemText
            primary={tenant.displayName}
            secondary={tenant.description}
            primaryTypographyProps={{
              variant: 'body2',
              fontWeight: tenant.id === ALL_TENANTS_ID ? 'bold' : 'normal',
              noWrap: true
            }}
            secondaryTypographyProps={{
              variant: 'caption',
              color: 'text.secondary',
              noWrap: true
            }}
          />
        </Box>
      </Box>
    </li>
  );

  const renderTags = (tagValue: Tenant[], getTagProps: any) => {
    return tagValue.map((tenant, index) => {
      const { key, ...tagProps } = getTagProps({ index }) as { key?: React.Key } & Record<string, any>;
      return (
        <Chip
          key={key ?? tenant.id}
          label={tenant.displayName}
          size={size}
          color={tenant.id === ALL_TENANTS_ID ? 'primary' : 'default'}
          icon={tenant.id === ALL_TENANTS_ID ? <PublicIcon /> : <BusinessIcon />}
          {...tagProps}
          sx={{
            fontWeight: tenant.id === ALL_TENANTS_ID ? 'bold' : 'normal'
          }}
        />
      );
    });
  };

  return (
    <Autocomplete
      multiple={multiple}
      value={getSelectedTenants()}
      onChange={handleChange}
      inputValue={inputValue}
      onInputChange={(_, newInputValue) => setInputValue(newInputValue)}
      options={filteredTenants}
      getOptionLabel={(option) => option.displayName}
      isOptionEqualToValue={(option, value) => option.id === value.id}
      loading={loading}
      disabled={disabled}
      size={size}
      fullWidth={fullWidth}
      filterOptions={(x) => x} // Disable built-in filtering since we handle it ourselves
      renderOption={renderOption}
      renderTags={renderTags}
      renderInput={(params) => (
        <TextField
          {...params}
          label={label}
          placeholder={value.length === 0 ? placeholder : ''}
          helperText={helperText}
          error={error}
          InputProps={{
            ...params.InputProps,
            endAdornment: (
              <>
                {loading ? <CircularProgress color="inherit" size={20} /> : null}
                {params.InputProps.endAdornment}
              </>
            )
          }}
        />
      )}
      noOptionsText={
        <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
          {inputValue ? `No tenants found matching "${inputValue}"` : 'No tenants available'}
        </Typography>
      }
      loadingText={
        <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
          Loading tenants...
        </Typography>
      }
    />
  );
};

export default TenantTypeahead;
