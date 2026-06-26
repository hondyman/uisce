import React, { useState, useEffect } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Button,
  Grid,
  Chip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  IconButton,
  Tooltip,
  Alert,
  Stack,
} from '@mui/material';
import {
  Add as AddIcon,
  Sync as SyncIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  ContentCopy as CopyIcon,
  CheckCircle as CheckIcon,
  Warning as WarningIcon,
} from '@mui/icons-material';
import { FieldTypeBadge } from '../../../components/FieldTypeBadges';

// ============================================================================
// SEMANTIC MODEL MANAGER
// Manages Core → Custom semantic model inheritance
// ============================================================================

interface SemanticModel {
  id: string;
  name: string;
  label: string;
  description?: string;
  model_type: 'core' | 'custom' | 'override';
  source_cube_id?: string;
  business_object_id?: string;
  is_system: boolean;
  status: string;
}

interface SemanticDimension {
  id: string;
  name: string;
  label: string;
  sql: string;
  type: string;
  is_inherited: boolean;
  is_overridden: boolean;
}

interface SemanticMeasure {
  id: string;
  name: string;
  label: string;
  sql: string;
  type: string;
  is_inherited: boolean;
  is_overridden: boolean;
}

export const SemanticModelManager: React.FC = () => {
  const [coreModels, setCoreModels] = useState<SemanticModel[]>([]);
  const [tenantModels, setTenantModels] = useState<SemanticModel[]>([]);
  const [selectedModel, setSelectedModel] = useState<SemanticModel | null>(null);
  const [dimensions, setDimensions] = useState<SemanticDimension[]>([]);
  const [measures, setMeasures] = useState<SemanticMeasure[]>([]);
  const [provisionDialogOpen, setProvisionDialogOpen] = useState(false);
  const [selectedCoreModel, setSelectedCoreModel] = useState<string>('');
  const [syncMessage, setSyncMessage] = useState<string>('');

  useEffect(() => {
    loadCoreModels();
    loadTenantModels();
  }, []);

  const loadCoreModels = async () => {
    // TODO: Call API to get core models
    // const response = await fetch('/api/semantic-models/core');
    // setCoreModels(await response.json());
  };

  const loadTenantModels = async () => {
    // TODO: Call API to get tenant models
    // const response = await fetch('/api/semantic-models/tenant');
    // setTenantModels(await response.json());
  };

  const loadModelDetails = async (modelId: string) => {
    // TODO: Call API to get model with dimensions/measures
    // const response = await fetch(`/api/semantic-models/${modelId}`);
    // const data = await response.json();
    // setSelectedModel(data.model);
    // setDimensions(data.dimensions);
    // setMeasures(data.measures);
  };

  const handleProvisionModel = async () => {
    if (!selectedCoreModel) return;

    try {
      // TODO: Call API to provision tenant model from core
      // const response = await fetch('/api/semantic-models/provision', {
      //   method: 'POST',
      //   body: JSON.stringify({ core_cube_id: selectedCoreModel }),
      // });
      // const newModel = await response.json();
      
      setProvisionDialogOpen(false);
      loadTenantModels();
      setSyncMessage('Model provisioned successfully!');
    } catch (error) {
      console.error('Failed to provision model:', error);
    }
  };

  const handleSyncWithBO = async (modelId: string) => {
    try {
      // TODO: Call API to sync model with BO
      // const response = await fetch(`/api/semantic-models/${modelId}/sync`, {
      //   method: 'POST',
      // });
      // const result = await response.json();
      
      setSyncMessage(`Synced ${0} new fields from business object`);
      loadModelDetails(modelId);
    } catch (error) {
      console.error('Failed to sync model:', error);
    }
  };

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 3 }}>
        <Typography variant="h4">Semantic Models</Typography>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => setProvisionDialogOpen(true)}
        >
          Provision from Core
        </Button>
      </Box>

      {syncMessage && (
        <Alert severity="success" onClose={() => setSyncMessage('')} sx={{ mb: 2 }}>
          {syncMessage}
        </Alert>
      )}

      <Grid container spacing={3}>
        {/* Core Models (Templates) */}
        <Grid item xs={12} md={4}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                Core Models (Templates)
              </Typography>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                Platform-provided templates. Never used directly.
              </Typography>
              <Stack spacing={1}>
                {coreModels.map((model) => (
                  <Card key={model.id} variant="outlined">
                    <CardContent sx={{ p: 2, '&:last-child': { pb: 2 } }}>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <Typography variant="subtitle2">{model.label}</Typography>
                        <Chip label="Core" size="small" color="secondary" />
                      </Box>
                      <Typography variant="caption" color="text.secondary">
                        {model.description}
                      </Typography>
                    </CardContent>
                  </Card>
                ))}
              </Stack>
            </CardContent>
          </Card>
        </Grid>

        {/* Tenant Models */}
        <Grid item xs={12} md={8}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                Your Custom Models
              </Typography>
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>Model</TableCell>
                    <TableCell>Type</TableCell>
                    <TableCell>Source</TableCell>
                    <TableCell>Status</TableCell>
                    <TableCell align="right">Actions</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {tenantModels.map((model) => (
                    <TableRow key={model.id} hover>
                      <TableCell>
                        <Typography variant="body2" fontWeight="medium">
                          {model.label}
                        </Typography>
                        <Typography variant="caption" color="text.secondary">
                          {model.name}
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <Chip
                          label={model.model_type}
                          size="small"
                          color={model.model_type === 'custom' ? 'primary' : 'warning'}
                        />
                      </TableCell>
                      <TableCell>
                        {model.source_cube_id && (
                          <Tooltip title="Extends core model">
                            <Chip
                              icon={<CopyIcon />}
                              label="Core"
                              size="small"
                              variant="outlined"
                            />
                          </Tooltip>
                        )}
                      </TableCell>
                      <TableCell>
                        <Chip
                          label={model.status}
                          size="small"
                          color={model.status === 'active' ? 'success' : 'default'}
                        />
                      </TableCell>
                      <TableCell align="right">
                        <Tooltip title="Sync with Business Object">
                          <IconButton
                            size="small"
                            onClick={() => handleSyncWithBO(model.id)}
                          >
                            <SyncIcon />
                          </IconButton>
                        </Tooltip>
                        <Tooltip title="Edit Model">
                          <IconButton
                            size="small"
                            onClick={() => loadModelDetails(model.id)}
                          >
                            <EditIcon />
                          </IconButton>
                        </Tooltip>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>

          {/* Dimensions & Measures */}
          {selectedModel && (
            <Card sx={{ mt: 2 }}>
              <CardContent>
                <Typography variant="h6" gutterBottom>
                  {selectedModel.label} - Dimensions & Measures
                </Typography>

                <Typography variant="subtitle2" sx={{ mt: 2, mb: 1 }}>
                  Dimensions
                </Typography>
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>Name</TableCell>
                      <TableCell>Type</TableCell>
                      <TableCell>SQL</TableCell>
                      <TableCell>Source</TableCell>
                      <TableCell align="right">Actions</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {dimensions.map((dim) => (
                      <TableRow key={dim.id}>
                        <TableCell>{dim.label}</TableCell>
                        <TableCell>
                          <Chip label={dim.type} size="small" />
                        </TableCell>
                        <TableCell>
                          <Typography variant="caption" sx={{ fontFamily: 'monospace' }}>
                            {dim.sql}
                          </Typography>
                        </TableCell>
                        <TableCell>
                          {dim.is_inherited && (
                            <FieldTypeBadge
                              isInherited
                              inheritedFrom="Core"
                              size="small"
                            />
                          )}
                          {dim.is_overridden && (
                            <Chip
                              label="Overridden"
                              size="small"
                              color="warning"
                              icon={<WarningIcon />}
                            />
                          )}
                        </TableCell>
                        <TableCell align="right">
                          <IconButton size="small">
                            <EditIcon fontSize="small" />
                          </IconButton>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>

                <Typography variant="subtitle2" sx={{ mt: 3, mb: 1 }}>
                  Measures
                </Typography>
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>Name</TableCell>
                      <TableCell>Type</TableCell>
                      <TableCell>SQL</TableCell>
                      <TableCell>Source</TableCell>
                      <TableCell align="right">Actions</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {measures.map((measure) => (
                      <TableRow key={measure.id}>
                        <TableCell>{measure.label}</TableCell>
                        <TableCell>
                          <Chip label={measure.type} size="small" />
                        </TableCell>
                        <TableCell>
                          <Typography variant="caption" sx={{ fontFamily: 'monospace' }}>
                            {measure.sql}
                          </Typography>
                        </TableCell>
                        <TableCell>
                          {measure.is_inherited && (
                            <FieldTypeBadge
                              isInherited
                              inheritedFrom="Core"
                              size="small"
                            />
                          )}
                          {measure.is_overridden && (
                            <Chip
                              label="Overridden"
                              size="small"
                              color="warning"
                              icon={<WarningIcon />}
                            />
                          )}
                        </TableCell>
                        <TableCell align="right">
                          <IconButton size="small">
                            <EditIcon fontSize="small" />
                          </IconButton>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          )}
        </Grid>
      </Grid>

      {/* Provision Dialog */}
      <Dialog open={provisionDialogOpen} onClose={() => setProvisionDialogOpen(false)}>
        <DialogTitle>Provision Custom Model from Core</DialogTitle>
        <DialogContent>
          <FormControl fullWidth sx={{ mt: 2 }}>
            <InputLabel>Select Core Model</InputLabel>
            <Select
              value={selectedCoreModel}
              onChange={(e) => setSelectedCoreModel(e.target.value)}
              label="Select Core Model"
            >
              {coreModels.map((model) => (
                <MenuItem key={model.id} value={model.id}>
                  {model.label}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
          <Alert severity="info" sx={{ mt: 2 }}>
            This will create a custom copy of the core model that you can extend and modify.
          </Alert>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setProvisionDialogOpen(false)}>Cancel</Button>
          <Button
            onClick={handleProvisionModel}
            variant="contained"
            disabled={!selectedCoreModel}
          >
            Provision
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default SemanticModelManager;
