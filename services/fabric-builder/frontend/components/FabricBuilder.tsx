import React, { useState, useEffect } from 'react';
import {
  Box,
  Card,
  CardContent,
  CardHeader,
  Button,
  Grid,
  CircularProgress,
  Typography,
  Stack,
  Paper
} from '@mui/material';
import { FabricModel, Extension } from '../types/fabric';

interface FabricBuilderProps {
  tenantId: string;
  datasourceId: string;
}

export const FabricBuilder: React.FC<FabricBuilderProps> = ({ tenantId, datasourceId }) => {
  const [models, setModels] = useState<FabricModel[]>([]);
  const [extensions, setExtensions] = useState<Extension[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadFabricData();
  }, [tenantId, datasourceId]);

  const loadFabricData = async () => {
    try {
      // TODO: Replace with actual API calls
      const modelsResponse = await fetch(`/api/fabric/models?datasource_id=${datasourceId}`);
      const extensionsResponse = await fetch(`/api/fabric/extensions?datasource_id=${datasourceId}`);

      const modelsData = await modelsResponse.json();
      const extensionsData = await extensionsResponse.json();

      setModels(modelsData.models || []);
      setExtensions(extensionsData || []);
    } catch (error) {
      console.error('Failed to load fabric data:', error);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', py: 4 }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h5" sx={{ fontWeight: 'bold', mb: 4 }}>
        Fabric Builder
      </Typography>

      <Grid container spacing={3}>
        {/* Models Section */}
        <Grid item xs={12} md={6}>
          <Card sx={{ boxShadow: 2 }}>
            <CardHeader title="Semantic Models" />
            <CardContent>
              <Stack spacing={2} sx={{ mb: 2 }}>
                {models.map((model) => (
                  <Paper key={model.id} sx={{ p: 2, border: '1px solid', borderColor: 'divider' }}>
                    <Typography variant="subtitle2" sx={{ fontWeight: 'medium' }}>
                      {model.name}
                    </Typography>
                    <Typography variant="body2" color="textSecondary">
                      {model.description}
                    </Typography>
                  </Paper>
                ))}
              </Stack>
              <Button
                fullWidth
                variant="contained"
                color="primary"
              >
                Create Model
              </Button>
            </CardContent>
          </Card>
        </Grid>

        {/* Extensions Section */}
        <Grid item xs={12} md={6}>
          <Card sx={{ boxShadow: 2 }}>
            <CardHeader title="Extensions" />
            <CardContent>
              <Stack spacing={2} sx={{ mb: 2 }}>
                {extensions.map((extension) => (
                  <Paper key={extension.id} sx={{ p: 2, border: '1px solid', borderColor: 'divider' }}>
                    <Typography variant="subtitle2" sx={{ fontWeight: 'medium' }}>
                      {extension.name}
                    </Typography>
                    <Typography variant="body2" color="textSecondary">
                      {extension.type}
                    </Typography>
                  </Paper>
                ))}
              </Stack>
              <Button
                fullWidth
                variant="contained"
                color="success"
              >
                Create Extension
              </Button>
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );
};