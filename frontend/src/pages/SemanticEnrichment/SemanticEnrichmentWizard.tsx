import React, { useState, useEffect } from 'react';
import {
  Box,
  Button,
  Card,
  CardContent,
  CircularProgress,
  Grid,
  Typography,
  Alert,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Checkbox,
  Paper,
  Chip,
  Stack,
  Divider,
} from '@mui/material';
import { AutoAwesome, Check, Save } from '@mui/icons-material';
import BusinessEntitySemanticService, { MappingResult } from '../../services/businessEntitySemanticService';
import { useAccess } from '../../contexts/AccessContext';

const SemanticEnrichmentWizard: React.FC = () => {
  const { currentTenant: tenant, currentDatasource: datasource } = useAccess();
  const [mappings, setMappings] = useState<MappingResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [stats, setStats] = useState<{ total: number; selected: number } | null>(null);

  // Get service instance
  const getService = () => {
    if (!tenant?.id || !datasource?.id) {
      throw new Error('Please select a tenant and datasource first.');
    }
    return new BusinessEntitySemanticService(tenant.id, datasource.id);
  };

  useEffect(() => {
    // Update stats whenever mappings change
    const selectedCount = mappings.filter(m => m.selected).length;
    setStats({ total: mappings.length, selected: selectedCount });
  }, [mappings]);

  const handleGenerate = async () => {
    setLoading(true);
    setError(null);
    setSuccess(null);
    setMappings([]);

    try {
      const service = getService();
      const results = await service.generateSemanticMappings();
      // Ensure results are typed correctly or mapped if needed
      if (Array.isArray(results)) {
         setMappings(results.map((m: any) => ({ ...m, selected: m.confidence > 0.7 }))); // Default select high confidence
      } else {
         // Handle case where API might return wrapped object
         setMappings([]);
         setError("Unexpected API response format");
      }
    } catch (err: any) {
      console.error(err);
      setError(err.message || 'Failed to generate mappings.');
    } finally {
      setLoading(false);
    }
  };

  const handleApply = async () => {
    const selectedMappings = mappings.filter(m => m.selected);
    if (selectedMappings.length === 0) return;

    setLoading(true);
    setError(null);
    setSuccess(null);

    try {
      const service = getService();
      const result = await service.applySemanticMappings(selectedMappings);
      setSuccess(`Successfully applied ${result.applied_count} mappings!`);
      // Update local state to reflect changes (maybe remove applied ones or mark them)
      // For now, clear the list or re-fetch could be options. Let's keep them but show success.
    } catch (err: any) {
      console.error(err);
      setError(err.message || 'Failed to apply mappings.');
    } finally {
      setLoading(false);
    }
  };

  const handleToggleSelect = (index: number) => {
    setMappings(prev => {
      const newMappings = [...prev];
      newMappings[index] = { ...newMappings[index], selected: !newMappings[index].selected };
      return newMappings;
    });
  };

  const handleSelectAll = (select: boolean) => {
      setMappings(prev => prev.map(m => ({ ...m, selected: select })));
  };

  return (
    <Box sx={{ p: 3 }}>
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 3 }}>
        <Box>
          <Typography variant="h4" component="h1" gutterBottom>
            Semantic Enrichment Wizard
          </Typography>
          <Typography variant="body1" color="text.secondary">
            Automatically discover and apply semantic terms to your database columns using AI and pattern matching.
          </Typography>
          {tenant && datasource ? (
             <Typography variant="caption" color="text.secondary" sx={{ mt: 1, display: 'block' }}>
                Context: <strong>{tenant.name}</strong> / <strong>{datasource.source_name}</strong>
             </Typography>
          ) : (
             <Alert severity="warning" sx={{ mt: 2 }}>
                Please select a Tenant and Datasource to proceed.
             </Alert>
          )}
        </Box>
        <Button
          variant="contained"
          size="large"
          startIcon={loading ? <CircularProgress size={20} color="inherit" /> : <AutoAwesome />}
          onClick={handleGenerate}
          disabled={loading || !tenant || !datasource}
        >
          {loading ? 'Analyzing...' : 'Run Analysis'}
        </Button>
      </Stack>

      {error && <Alert severity="error" sx={{ mb: 3 }}>{error}</Alert>}
      {success && <Alert severity="success" sx={{ mb: 3 }}>{success}</Alert>}

      {mappings.length > 0 && (
        <Card>
          <CardContent>
            <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 2 }}>
              <Typography variant="h6">
                Recommendations ({stats?.selected} selected / {stats?.total} total)
              </Typography>
              <Box>
                  <Button onClick={() => handleSelectAll(true)} sx={{ mr: 1 }}>Select All</Button>
                  <Button onClick={() => handleSelectAll(false)} sx={{ mr: 2 }}>Deselect All</Button>
                  <Button
                    variant="contained"
                    color="primary"
                    startIcon={<Save />}
                    onClick={handleApply}
                    disabled={loading || (stats?.selected === 0)}
                  >
                    Apply Selected
                  </Button>
              </Box>
            </Stack>

            <TableContainer component={Paper} variant="outlined" sx={{ maxHeight: 600 }}>
              <Table stickyHeader>
                <TableHead>
                  <TableRow>
                    <TableCell padding="checkbox">
                      <Checkbox
                        indeterminate={mappings.some(m => m.selected) && !mappings.every(m => m.selected)}
                        checked={mappings.length > 0 && mappings.every(m => m.selected)}
                        onChange={(e) => handleSelectAll(e.target.checked)}
                      />
                    </TableCell>
                    <TableCell>Table / Column</TableCell>
                    <TableCell>Suggested Semantic Term</TableCell>
                    <TableCell>Confidence</TableCell>
                    <TableCell>Reason</TableCell>
                    <TableCell>Status</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {mappings.map((mapping, index) => (
                    <TableRow key={index} hover selected={mapping.selected}>
                      <TableCell padding="checkbox">
                        <Checkbox
                          checked={mapping.selected}
                          onChange={() => handleToggleSelect(index)}
                        />
                      </TableCell>
                      <TableCell>
                        <Stack>
                            <Typography variant="body2" fontWeight="bold">{mapping.database_column?.table}</Typography>
                            <Typography variant="caption" color="text.secondary">{mapping.database_column?.column} ({mapping.database_column?.data_type})</Typography>
                        </Stack>
                      </TableCell>
                      <TableCell>
                        <Chip
                            label={mapping.semantic_term}
                            color="info"
                            variant={mapping.is_new_term ? "outlined" : "filled"}
                            size="small"
                        />
                         {mapping.is_new_term && <Typography variant="caption" display="block" color="text.secondary">New Term</Typography>}
                      </TableCell>
                      <TableCell>
                        <Box sx={{ display: 'flex', alignItems: 'center' }}>
                           <CircularProgress
                                variant="determinate"
                                value={mapping.confidence * 100}
                                color={mapping.confidence > 0.8 ? "success" : mapping.confidence > 0.5 ? "warning" : "error"}
                                size={24}
                                sx={{ mr: 1 }}
                            />
                            <Typography variant="body2">{Math.round(mapping.confidence * 100)}%</Typography>
                        </Box>
                      </TableCell>
                      <TableCell>
                        <Typography variant="body2" sx={{ maxWidth: 300 }}>
                            {mapping.match_reason || mapping.match_reason || "Pattern Match"}
                        </Typography>
                      </TableCell>
                      <TableCell>
                         {mapping.edge_exists ? (
                             <Chip icon={<Check />} label="Linked" color="success" size="small" variant="outlined" />
                         ) : (
                             <Chip label="Unlinked" color="default" size="small" variant="outlined" />
                         )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          </CardContent>
        </Card>
      )}
      
      {mappings.length === 0 && !loading && !error && (
        <Box sx={{ mt: 5, textAlign: 'center', color: 'text.secondary' }}>
             <Typography variant="h6">No mappings generated yet.</Typography>
             <Typography>Click "Run Analysis" to start discovery.</Typography>
        </Box>
      )}
    </Box>
  );
};

export default SemanticEnrichmentWizard;
