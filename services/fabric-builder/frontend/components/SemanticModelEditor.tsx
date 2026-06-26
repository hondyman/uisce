import React, { useState } from 'react';
import {
  Card,
  CardContent,
  CardHeader,
  TextField,
  Button,
  Grid,
  Box,
  Typography,
  Stack,
  Paper
} from '@mui/material';
import { FabricModel } from '../types/fabric';

interface SemanticModelEditorProps {
  model?: FabricModel;
  onSave: (model: FabricModel) => void;
  onCancel: () => void;
}

export const SemanticModelEditor: React.FC<SemanticModelEditorProps> = ({
  model,
  onSave,
  onCancel
}) => {
  const [formData, setFormData] = useState<Partial<FabricModel>>({
    name: model?.name || '',
    description: model?.description || '',
    schema: model?.schema || {},
    ...model
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const updatedModel: FabricModel = {
      id: model?.id || `model-${Date.now()}`,
      name: formData.name || '',
      description: formData.description || '',
      tenant_id: formData.tenant_id || '',
      datasource_id: formData.datasource_id || '',
      schema: formData.schema || {},
      created_at: model?.created_at || new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };
    onSave(updatedModel);
  };

  const updateSchema = (key: string, value: any) => {
    setFormData(prev => ({
      ...prev,
      schema: {
        ...prev.schema,
        [key]: value
      }
    }));
  };

  return (
    <Card sx={{ boxShadow: 2 }}>
      <CardHeader
        title={model ? 'Edit Semantic Model' : 'Create Semantic Model'}
        titleTypographyProps={{ variant: 'h6', sx: { fontWeight: 'bold' } }}
      />
      <CardContent>
        <Box component="form" onSubmit={handleSubmit} sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
          {/* Basic Information */}
          <Grid container spacing={2}>
            <Grid item xs={12} sm={6}>
              <TextField
                fullWidth
                label="Model Name"
                placeholder="Enter model name"
                value={formData.name}
                onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
                variant="outlined"
                required
              />
            </Grid>

            <Grid item xs={12} sm={6}>
              <TextField
                fullWidth
                label="Description"
                placeholder="Short description (optional)"
                value={formData.description}
                onChange={(e) => setFormData(prev => ({ ...prev, description: e.target.value }))}
                variant="outlined"
              />
            </Grid>
          </Grid>

          {/* Schema Configuration */}
          <Box>
            <Typography variant="subtitle2" sx={{ fontWeight: 'bold', mb: 2 }}>
              Schema Configuration
            </Typography>
            <Paper sx={{ p: 2, border: '1px solid', borderColor: 'divider' }}>
              <Stack spacing={2}>
                <Grid container spacing={2}>
                  <Grid item xs={12} sm={6}>
                    <TextField
                      fullWidth
                      label="Primary Key"
                      placeholder="e.g., id"
                      value={formData.schema?.primary_key || ''}
                      onChange={(e) => updateSchema('primary_key', e.target.value)}
                      variant="outlined"
                      size="small"
                    />
                  </Grid>
                  <Grid item xs={12} sm={6}>
                    <TextField
                      fullWidth
                      label="Display Field"
                      placeholder="e.g., name"
                      value={formData.schema?.display_field || ''}
                      onChange={(e) => updateSchema('display_field', e.target.value)}
                      variant="outlined"
                      size="small"
                    />
                  </Grid>
                </Grid>
                <TextField
                  fullWidth
                  multiline
                  rows={4}
                  label="Field Types (JSON)"
                  placeholder='{"name": "string", "email": "string", "age": "number"}'
                  value={JSON.stringify(formData.schema?.fields || {}, null, 2)}
                  onChange={(e) => {
                    try {
                      const fields = JSON.parse(e.target.value);
                      updateSchema('fields', fields);
                    } catch (err) {
                      // Invalid JSON, keep current value
                    }
                  }}
                  variant="outlined"
                  sx={{ fontFamily: 'monospace' }}
                />
              </Stack>
            </Paper>
          </Box>

          {/* Action Buttons */}
          <Stack direction="row" spacing={1} sx={{ justifyContent: 'flex-end' }}>
            <Button
              variant="outlined"
              onClick={onCancel}
            >
              Cancel
            </Button>
            <Button
              variant="contained"
              color="primary"
              type="submit"
            >
              {model ? 'Update Model' : 'Create Model'}
            </Button>
          </Stack>
        </Box>
      </CardContent>
    </Card>
  );
};
