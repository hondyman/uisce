import React, { useState, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Tabs,
  Tab,
  Box,
  Typography,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  CircularProgress,
  Alert,
  IconButton,
  TextField,
  MenuItem,
  FormControl,
  InputLabel,
  Select,
  Stack,
  Autocomplete
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import AddIcon from '@mui/icons-material/Add';
import { getSelectedRegion } from '../../lib/region';

interface RelationshipResult {
  relatedObjectName: string;
  relationshipType: string;
  description: string;
}

interface SemanticFieldResult {
  fieldName: string;
  semanticTermName: string;
  edge_type_name: string;
}

interface BusinessObjectRelationshipWizardProps {
  open: boolean;
  onClose: () => void;
  businessObject: any;
  tenantId: string;
  datasourceId: string;
}

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;

  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`relationship-tabpanel-${index}`}
      aria-labelledby={`relationship-tab-${index}`}
      {...other}
    >
      {value === index && (
        <Box sx={{ p: 3 }}>
          {children}
        </Box>
      )}
    </div>
  );
}

export const BusinessObjectRelationshipWizard: React.FC<BusinessObjectRelationshipWizardProps> = ({
  open,
  onClose,
  businessObject,
  tenantId,
  datasourceId
}) => {
  const [activeTab, setActiveTab] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);
  const [relatedObjects, setRelatedObjects] = useState<RelationshipResult[]>([]);
  const [semanticFields, setSemanticFields] = useState<SemanticFieldResult[]>([]);

  // Add Relationship Form State
  const [availableBOs, setAvailableBOs] = useState<any[]>([]);
  const [selectedTargetBO, setSelectedTargetBO] = useState<any | null>(null);
  const [relationshipType, setRelationshipType] = useState('association');
  const [cardinality, setCardinality] = useState('1:N');
  const [description, setDescription] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Helper to build headers with authentication
  const getAuthHeaders = (additionalHeaders: Record<string, string> = {}): Record<string, string> => {
    const token = typeof localStorage !== 'undefined' ? localStorage.getItem('auth_token') : null;
    const authHeader = token && !token.includes('demo') ? `Bearer ${token}` : '';
    
    return {
      'Authorization': authHeader,
      'Content-Type': 'application/json',
      'X-Tenant-ID': tenantId,
      'X-Tenant-Datasource-ID': datasourceId,
      'X-Tenant-Region': getSelectedRegion(),
      ...additionalHeaders,
    };
  };

  useEffect(() => {
    if (open && businessObject?.id) {
      fetchRelationships();
      fetchAvailableBOs();
    }
  }, [open, businessObject]);

  const fetchRelationships = async () => {
    if (!businessObject?.id) return;
    
    setLoading(true);
    setError(null);
    
    try {
      const response = await fetch(`/api/business-objects/${businessObject.id}/relationships`, {
        headers: getAuthHeaders(),
      });

      if (!response.ok) {
        throw new Error('Failed to fetch relationships');
      }

      const data = await response.json();
      setRelatedObjects(data.relatedObjects || []);
      setSemanticFields(data.semanticFields || []);
    } catch (err) {
      console.error('Error fetching relationships:', err);
      setError('Failed to load relationships. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const fetchAvailableBOs = async () => {
    try {
      const response = await fetch('/api/business-objects', {
        headers: getAuthHeaders(),
      });
      if (response.ok) {
        const data = await response.json();
        const list = Array.isArray(data)
          ? data
          : Object.entries(data || {}).map(([id, obj]: [string, any]) => ({ ...obj, id }));
        // Filter out current BO
        setAvailableBOs(list.filter((b: any) => b.id !== businessObject?.id));
      }
    } catch (err) {
      console.error('Failed to fetch business objects:', err);
    }
  };

  const handleSaveRelationship = async () => {
    if (!selectedTargetBO) {
      setError('Please select a target Business Object');
      return;
    }

    setIsSubmitting(true);
    setError(null);
    setSuccessMsg(null);

    try {
      const response = await fetch(`/api/business-objects/${businessObject.id}/relationships`, {
        method: 'POST',
        headers: getAuthHeaders(),
        body: JSON.stringify({
          targetObjectId: selectedTargetBO.id,
          relationshipType,
          cardinality,
          description,
        }),
      });

      if (!response.ok) {
        const errText = await response.text();
        throw new Error(errText || 'Failed to save relationship');
      }

      setSuccessMsg(`Successfully related to ${selectedTargetBO.displayName || selectedTargetBO.name}`);
      setSelectedTargetBO(null);
      setDescription('');
      fetchRelationships();
      setActiveTab(0);
    } catch (err: any) {
      setError(err.message || 'An error occurred while saving the relationship.');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleTabChange = (_event: React.SyntheticEvent, newValue: number) => {
    setActiveTab(newValue);
  };

  return (
    <Dialog 
      open={open} 
      onClose={onClose}
      maxWidth="md"
      fullWidth
    >
      <DialogTitle sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Typography variant="h6">Relationship Wizard: {businessObject?.displayName || businessObject?.name}</Typography>
        <IconButton onClick={onClose} size="small">
          <CloseIcon />
        </IconButton>
      </DialogTitle>
      
      <DialogContent>
        <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
          <Tabs value={activeTab} onChange={handleTabChange} aria-label="relationship wizard tabs">
            <Tab label={`Related Objects (${relatedObjects.length})`} />
            <Tab label={`Semantic Fields (${semanticFields.length})`} />
            <Tab label="Add Relationship" icon={<AddIcon fontSize="small" />} iconPosition="start" />
          </Tabs>
        </Box>

        {successMsg && (
          <Box sx={{ p: 2 }}>
            <Alert severity="success">{successMsg}</Alert>
          </Box>
        )}

        {loading ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
            <CircularProgress />
          </Box>
        ) : (
          <>
            {/* Related Objects Tab */}
            <TabPanel value={activeTab} index={0}>
              {relatedObjects.length > 0 ? (
                <TableContainer component={Paper} variant="outlined">
                  <Table size="small">
                    <TableHead>
                      <TableRow>
                        <TableCell>Related Object</TableCell>
                        <TableCell>Relationship Type</TableCell>
                        <TableCell>Description / Cardinality</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {relatedObjects.map((row, index) => (
                        <TableRow key={index}>
                          <TableCell sx={{ fontWeight: 'bold' }}>{row.relatedObjectName}</TableCell>
                          <TableCell>{row.relationshipType}</TableCell>
                          <TableCell>{row.description}</TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
              ) : (
                <Typography color="text.secondary">No related objects found. Click "Add Relationship" to create one.</Typography>
              )}
            </TabPanel>

            {/* Semantic Fields Tab */}
            <TabPanel value={activeTab} index={1}>
              {semanticFields.length > 0 ? (
                <TableContainer component={Paper} variant="outlined">
                  <Table size="small">
                    <TableHead>
                      <TableRow>
                        <TableCell>Field Name</TableCell>
                        <TableCell>Semantic Term</TableCell>
                        <TableCell>Edge Type</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {semanticFields.map((row, index) => (
                        <TableRow key={index}>
                          <TableCell>{row.fieldName}</TableCell>
                          <TableCell>{row.semanticTermName}</TableCell>
                          <TableCell>{row.edge_type_name}</TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
              ) : (
                <Typography color="text.secondary">No semantic field mappings found for driver table.</Typography>
              )}
            </TabPanel>

            {/* Add Relationship Tab */}
            <TabPanel value={activeTab} index={2}>
              {error && (
                <Alert severity="error" sx={{ mb: 3 }}>{error}</Alert>
              )}

              <Stack spacing={3}>
                <Typography variant="body2" color="text.secondary">
                  Create a governed relationship link between <strong>{businessObject?.displayName || businessObject?.name}</strong> and another Business Object.
                </Typography>

                <Autocomplete
                  options={availableBOs}
                  getOptionLabel={(option) => `${option.displayName || option.name} (${option.key})`}
                  value={selectedTargetBO}
                  onChange={(_e, newValue) => setSelectedTargetBO(newValue)}
                  renderInput={(params) => (
                    <TextField {...params} label="Target Business Object" required placeholder="Select target object..." />
                  )}
                />

                <Stack direction="row" spacing={2}>
                  <FormControl fullWidth>
                    <InputLabel>Relationship Type</InputLabel>
                    <Select
                      value={relationshipType}
                      label="Relationship Type"
                      onChange={(e) => setRelationshipType(e.target.value)}
                    >
                      <MenuItem value="association">Association</MenuItem>
                      <MenuItem value="foreign_key">Foreign Key</MenuItem>
                      <MenuItem value="parent_child">Parent - Child</MenuItem>
                      <MenuItem value="one_to_many">One to Many</MenuItem>
                      <MenuItem value="many_to_one">Many to One</MenuItem>
                      <MenuItem value="reference">Reference</MenuItem>
                    </Select>
                  </FormControl>

                  <FormControl fullWidth>
                    <InputLabel>Cardinality</InputLabel>
                    <Select
                      value={cardinality}
                      label="Cardinality"
                      onChange={(e) => setCardinality(e.target.value)}
                    >
                      <MenuItem value="1:1">1 : 1 (One-to-One)</MenuItem>
                      <MenuItem value="1:N">1 : N (One-to-Many)</MenuItem>
                      <MenuItem value="N:1">N : 1 (Many-to-One)</MenuItem>
                      <MenuItem value="M:N">M : N (Many-to-Many)</MenuItem>
                    </Select>
                  </FormControl>
                </Stack>

                <TextField
                  label="Description / Join Condition SQL"
                  placeholder="e.g. t0.category_id = t1.id"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  multiline
                  rows={2}
                  fullWidth
                />

                <Box sx={{ display: 'flex', justifyContent: 'flex-end', mt: 2 }}>
                  <Button
                    variant="contained"
                    color="primary"
                    onClick={handleSaveRelationship}
                    disabled={isSubmitting || !selectedTargetBO}
                    startIcon={isSubmitting ? <CircularProgress size={18} color="inherit" /> : <AddIcon />}
                  >
                    Save Relationship
                  </Button>
                </Box>
              </Stack>
            </TabPanel>
          </>
        )}
      </DialogContent>
      
      <DialogActions>
        <Button onClick={onClose}>Close</Button>
      </DialogActions>
    </Dialog>
  );
};
