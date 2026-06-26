/**
 * Semantic Objects Selector Component
 *
 * Handles selection of measures and dimensions for the bundle
 */

import React, { useState, useMemo } from 'react';
import {
  Box,
  Typography,
  Paper,
  Chip,
  Alert,
  CircularProgress,
  Grid,
  IconButton,
} from '@mui/material';
import RefreshIcon from '@mui/icons-material/Refresh';
import { SemanticObjectReference } from '../../types/bundles';

interface SemanticObjectsSelectorProps {
  includedMeasures: SemanticObjectReference[];
  includedDimensions: SemanticObjectReference[];
  allObjects: SemanticObjectReference[];
  loadingObjects: boolean;
  objectsError: string | null;
  onMeasuresChange: (measures: SemanticObjectReference[]) => void;
  onDimensionsChange: (dimensions: SemanticObjectReference[]) => void;
  onRefreshObjects: () => void;
}

export const SemanticObjectsSelector: React.FC<SemanticObjectsSelectorProps> = ({
  includedMeasures,
  includedDimensions,
  allObjects,
  loadingObjects,
  objectsError,
  onMeasuresChange,
  onDimensionsChange,
  onRefreshObjects,
}) => {
  const [searchTerm, setSearchTerm] = useState('');

  const filteredObjects = useMemo(() => {
    if (!searchTerm) return allObjects;
    return allObjects.filter(obj =>
      (obj.name && obj.name.toLowerCase().includes(searchTerm.toLowerCase())) ||
      (obj.title && obj.title.toLowerCase().includes(searchTerm.toLowerCase()))
    );
  }, [allObjects, searchTerm]);

  const handleAddMeasure = (measure: SemanticObjectReference) => {
    if (!includedMeasures.find(m => m.id === measure.id)) {
      onMeasuresChange([...includedMeasures, measure]);
    }
  };

  const handleRemoveMeasure = (measureId: string) => {
    onMeasuresChange(includedMeasures.filter(m => m.id !== measureId));
  };

  const handleAddDimension = (dimension: SemanticObjectReference) => {
    if (!includedDimensions.find(d => d.id === dimension.id)) {
      onDimensionsChange([...includedDimensions, dimension]);
    }
  };

  const handleRemoveDimension = (dimensionId: string) => {
    onDimensionsChange(includedDimensions.filter(d => d.id !== dimensionId));
  };

  return (
    <Box sx={{ mb: 4 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
        <Typography variant="h6">Semantic Objects</Typography>
        <IconButton onClick={onRefreshObjects} disabled={loadingObjects}>
          <RefreshIcon />
        </IconButton>
      </Box>

      {objectsError && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {objectsError}
        </Alert>
      )}

      <Grid container spacing={3}>
        {/* Measures Section */}
        <Grid item xs={12} md={6}>
          <Paper sx={{ p: 2 }}>
            <Typography variant="subtitle1" gutterBottom>
              Measures ({includedMeasures.length})
            </Typography>

            {includedMeasures.length > 0 && (
              <Box sx={{ mb: 2 }}>
                {includedMeasures.map((measure) => (
                  <Chip
                    key={measure.id}
                    label={measure.name || measure.id}
                    onDelete={() => handleRemoveMeasure(measure.id)}
                    sx={{ mr: 1, mb: 1 }}
                  />
                ))}
              </Box>
            )}

            {loadingObjects ? (
              <CircularProgress size={24} />
            ) : (
              <Box sx={{ maxHeight: 200, overflow: 'auto' }}>
                {filteredObjects
                  .filter(obj => obj.type === 'measure')
                  .filter(obj => !includedMeasures.find(m => m.id === obj.id))
                  .map((obj) => (
                    <Chip
                      key={obj.id}
                      label={obj.name || obj.id}
                      onClick={() => handleAddMeasure(obj)}
                      variant="outlined"
                      sx={{ mr: 1, mb: 1, cursor: 'pointer' }}
                    />
                  ))}
              </Box>
            )}
          </Paper>
        </Grid>

        {/* Dimensions Section */}
        <Grid item xs={12} md={6}>
          <Paper sx={{ p: 2 }}>
            <Typography variant="subtitle1" gutterBottom>
              Dimensions ({includedDimensions.length})
            </Typography>

            {includedDimensions.length > 0 && (
              <Box sx={{ mb: 2 }}>
                {includedDimensions.map((dimension) => (
                  <Chip
                    key={dimension.id}
                    label={dimension.name || dimension.id}
                    onDelete={() => handleRemoveDimension(dimension.id)}
                    sx={{ mr: 1, mb: 1 }}
                  />
                ))}
              </Box>
            )}

            {loadingObjects ? (
              <CircularProgress size={24} />
            ) : (
              <Box sx={{ maxHeight: 200, overflow: 'auto' }}>
                {filteredObjects
                  .filter(obj => obj.type === 'dimension')
                  .filter(obj => !includedDimensions.find(d => d.id === obj.id))
                  .map((obj) => (
                    <Chip
                      key={obj.id}
                      label={obj.name || obj.id}
                      onClick={() => handleAddDimension(obj)}
                      variant="outlined"
                      sx={{ mr: 1, mb: 1, cursor: 'pointer' }}
                    />
                  ))}
              </Box>
            )}
          </Paper>
        </Grid>
      </Grid>
    </Box>
  );
};