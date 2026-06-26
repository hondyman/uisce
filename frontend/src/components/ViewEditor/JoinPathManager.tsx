import React, { useState, useEffect } from 'react';
import { Box, Typography, Stack, Card, CardContent, Chip, Autocomplete, TextField, IconButton } from '@mui/material';
import { Delete } from '@mui/icons-material';
import renderCoreCustomChips from '../common/semanticChips';

interface JoinPathManagerProps {
  viewData: any;
  setViewData: (data: any) => void;
  availableCubes: any[];
  hasTenantScope: boolean;
  onRefreshSources: () => void;
}

const JoinPathManager: React.FC<JoinPathManagerProps> = ({ 
  viewData, 
  setViewData, 
  availableCubes, 
  hasTenantScope,
  onRefreshSources: _onRefreshSources
}) => {
  const [joinPathOptions, setJoinPathOptions] = useState<any[]>([]);

  const joinPaths = viewData.join_paths || [];

  // Generate join path options based on available cubes
  useEffect(() => {
    // Mark options that are already selected so we can show them disabled with an "Added" hint
    const existingJoinIds = new Set<string>((joinPaths || []).map((jp: any) => String(jp?.id || jp?.path || '').toLowerCase()));
    const options = availableCubes.map(cube => ({
      id: cube.id,
      label: cube.display_name || cube.model_key,
      value: cube.model_key || cube.id,
      description: cube.description,
      is_core: Boolean(cube.is_core),
      is_custom: Boolean(cube.is_custom),
      alreadyAdded: existingJoinIds.has(String(cube.id).toLowerCase()),
    }));
    setJoinPathOptions(options);
  }, [availableCubes, joinPaths]);

  const handleAddJoinPath = (selectedOption: any) => {
    if (!selectedOption) return;
    
    const newJoinPaths = [...joinPaths, { 
      id: selectedOption.id,
      path: selectedOption.value,
      label: selectedOption.label,
      is_core: Boolean(selectedOption.is_core),
      is_custom: Boolean(selectedOption.is_custom),
    }];
    
    setViewData({
      ...viewData,
      join_paths: newJoinPaths
    });

    // Don't call onRefreshSources directly here. The parent ViewEditor
    // effect depends on selectedRefs (which includes join_paths) and will
    // refresh available sources when viewData changes. Calling it here
    // caused race conditions where the fetch ran with stale viewData.
  };

  const handleRemoveJoinPath = (index: number) => {
    const newJoinPaths = [...joinPaths];
    newJoinPaths.splice(index, 1);
    setViewData({
      ...viewData,
      join_paths: newJoinPaths
    });

    // Let the parent effect handle refreshing available sources instead of
    // invoking onRefreshSources directly to avoid stale-closure races.
  };

  return (
    <Box>
      <Typography variant="subtitle2" gutterBottom>
        Join Paths
      </Typography>
      
      {/* Current Join Paths */}
      <Stack spacing={1} sx={{ mb: 2 }}>
        {joinPaths.map((joinPath: any, index: number) => (
          <Card key={index} variant="outlined">
            <CardContent sx={{ p: 1.5, '&:last-child': { pb: 1.5 } }}>
              <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <Box>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <Typography variant="body2" fontWeight={500}>
                      {joinPath.label || joinPath.path}
                    </Typography>
                    {renderCoreCustomChips(joinPath)}
                  </Box>
                  <Typography variant="caption" color="text.secondary">{joinPath.value || joinPath.path || (joinPath.id ? String(joinPath.id) : '')}</Typography>
                </Box>
                <IconButton 
                  size="small" 
                  color="error" 
                  onClick={() => handleRemoveJoinPath(index)}
                >
                  <Delete fontSize="small" />
                </IconButton>
              </Box>
            </CardContent>
          </Card>
        ))}
      </Stack>

      {/* Add New Join Path */}
      <Autocomplete
        options={joinPathOptions}
        getOptionLabel={(option) => option.label}
        value={null}
        onChange={(_, selectedOption) => handleAddJoinPath(selectedOption)}
        disabled={!hasTenantScope}
        getOptionDisabled={(option) => Boolean(option?.alreadyAdded)}
        renderOption={(props, option) => (
          <li {...props} key={option.id}>
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', width: '100%' }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <Typography variant="body2" fontWeight={600}>{option.label}</Typography>
                {renderCoreCustomChips(option)}
              </Box>
              <Box>
                {option.alreadyAdded ? (
                  <Chip label="Added" size="small" color="default" variant="outlined" />
                ) : null}
              </Box>
            </Box>
          </li>
        )}
        renderInput={(params) => (
          <TextField
            {...params}
            size="small"
            label="Add Join Path"
            placeholder={hasTenantScope ? "Search and add join paths" : "Select tenant & datasource"}
            helperText="Add multiple join paths for complex view relationships"
            disabled={!hasTenantScope}
          />
        )}
      />
    </Box>
  );
};

export default JoinPathManager;
