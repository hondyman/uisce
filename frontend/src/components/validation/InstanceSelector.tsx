import React, { useState, useEffect } from 'react';
import {
  Box,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Autocomplete,
  TextField,
  CircularProgress,
  Typography,
  Chip,
} from '@mui/material';

interface BusinessObject {
  id: string;
  name: string;
  displayName?: string;
}

interface BusinessObjectInstance {
  id: string;
  coreFieldValues: Record<string, any>;
  customFieldValues: Record<string, any>;
}

interface InstanceSelectorProps {
  tenantId: string;
  datasourceId: string;
  onInstanceSelect: (instanceId: string, instance: BusinessObjectInstance | null) => void;
  selectedInstanceId?: string;
}

export const InstanceSelector: React.FC<InstanceSelectorProps> = ({
  tenantId,
  datasourceId,
  onInstanceSelect,
  selectedInstanceId,
}) => {
  const [businessObjects, setBusinessObjects] = useState<BusinessObject[]>([]);
  const [selectedBO, setSelectedBO] = useState<string>('');
  const [instances, setInstances] = useState<BusinessObjectInstance[]>([]);
  const [selectedInstance, setSelectedInstance] = useState<BusinessObjectInstance | null>(null);
  const [loading, setLoading] = useState(false);
  const [loadingInstances, setLoadingInstances] = useState(false);

  // Load business objects on mount
  useEffect(() => {
    const loadBusinessObjects = async () => {
      setLoading(true);
      try {
        const response = await fetch(
          `/api/business-objects?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}`
        );
        if (response.ok) {
          const data = await response.json();
          setBusinessObjects(data);
        }
      } catch (error) {
        console.error('Failed to load business objects:', error);
      } finally {
        setLoading(false);
      }
    };

    if (tenantId && datasourceId) {
      loadBusinessObjects();
    }
  }, [tenantId, datasourceId]);

  // Load instances when BO is selected
  useEffect(() => {
    const loadInstances = async () => {
      if (!selectedBO) {
        setInstances([]);
        return;
      }

      setLoadingInstances(true);
      try {
        const response = await fetch(
          `/api/business-objects/${selectedBO}/instances?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}&limit=100`
        );
        if (response.ok) {
          const data = await response.json();
          setInstances(data.instances || []);
        }
      } catch (error) {
        console.error('Failed to load instances:', error);
      } finally {
        setLoadingInstances(false);
      }
    };

    loadInstances();
  }, [selectedBO, tenantId, datasourceId]);

  const handleBOChange = (event: any) => {
    setSelectedBO(event.target.value);
    setSelectedInstance(null);
    onInstanceSelect('', null);
  };

  const handleInstanceChange = (_event: any, value: BusinessObjectInstance | null) => {
    setSelectedInstance(value);
    onInstanceSelect(value?.id || '', value);
  };

  const getInstanceLabel = (instance: BusinessObjectInstance) => {
    // Try to find identifying fields
    const core = instance.coreFieldValues || {};
    const custom = instance.customFieldValues || {};
    
    const name = core.name || custom.name || core.displayName || custom.displayName;
    const id = core.id || custom.id || instance.id.substring(0, 8);
    
    return name ? `${name} (${id})` : `Instance ${id}`;
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <FormControl fullWidth>
        <InputLabel>Business Object Type</InputLabel>
        <Select
          value={selectedBO}
          onChange={handleBOChange}
          label="Business Object Type"
          disabled={loading}
        >
          {loading && (
            <MenuItem value="">
              <CircularProgress size={20} sx={{ mr: 1 }} />
              Loading...
            </MenuItem>
          )}
          {businessObjects.map((bo) => (
            <MenuItem key={bo.id} value={bo.id}>
              {bo.displayName || bo.name}
            </MenuItem>
          ))}
        </Select>
      </FormControl>

      {selectedBO && (
        <Autocomplete
          options={instances}
          value={selectedInstance}
          onChange={handleInstanceChange}
          getOptionLabel={getInstanceLabel}
          loading={loadingInstances}
          renderInput={(params) => (
            <TextField
              {...params}
              label="Select Instance"
              placeholder="Search for an instance..."
              InputProps={{
                ...params.InputProps,
                endAdornment: (
                  <>
                    {loadingInstances && <CircularProgress size={20} />}
                    {params.InputProps.endAdornment}
                  </>
                ),
              }}
            />
          )}
          renderOption={(props, option) => (
            <li {...props}>
              <Box sx={{ display: 'flex', flexDirection: 'column', width: '100%' }}>
                <Typography variant="body2">{getInstanceLabel(option)}</Typography>
                <Typography variant="caption" color="text.secondary">
                  ID: {option.id.substring(0, 13)}...
                </Typography>
              </Box>
            </li>
          )}
        />
      )}

      {selectedInstance && (
        <Box sx={{ mt: 1 }}>
          <Typography variant="caption" color="text.secondary" gutterBottom>
            Instance Preview:
          </Typography>
          <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, mt: 0.5 }}>
            {Object.entries({ ...selectedInstance.coreFieldValues, ...selectedInstance.customFieldValues })
              .slice(0, 5)
              .map(([key, value]) => (
                <Chip
                  key={key}
                  label={`${key}: ${String(value).substring(0, 20)}`}
                  size="small"
                  variant="outlined"
                />
              ))}
          </Box>
        </Box>
      )}
    </Box>
  );
};

export default InstanceSelector;
