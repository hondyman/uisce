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
  IconButton
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';

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
  const [relatedObjects, setRelatedObjects] = useState<RelationshipResult[]>([]);
  const [semanticFields, setSemanticFields] = useState<SemanticFieldResult[]>([]);

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
        {businessObject?.driverTableId ? (
           <>
            <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
              <Tabs value={activeTab} onChange={handleTabChange} aria-label="relationship wizard tabs">
                <Tab label={`Related Objects (${relatedObjects.length})`} />
                <Tab label={`Semantic Fields (${semanticFields.length})`} />
              </Tabs>
            </Box>

            {loading ? (
              <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
                <CircularProgress />
              </Box>
            ) : error ? (
              <Box sx={{ p: 2 }}>
                <Alert severity="error">{error}</Alert>
              </Box>
            ) : (
              <>
                <TabPanel value={activeTab} index={0}>
                  {relatedObjects.length > 0 ? (
                    <TableContainer component={Paper} variant="outlined">
                      <Table size="small">
                        <TableHead>
                          <TableRow>
                            <TableCell>Related Object</TableCell>
                            <TableCell>Relationship Type</TableCell>
                            <TableCell>Description/Key</TableCell>
                          </TableRow>
                        </TableHead>
                        <TableBody>
                          {relatedObjects.map((row, index) => (
                            <TableRow key={index}>
                              <TableCell>{row.relatedObjectName}</TableCell>
                              <TableCell>{row.relationshipType}</TableCell>
                              <TableCell>{row.description}</TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    </TableContainer>
                  ) : (
                    <Typography color="text.secondary">No related objects found.</Typography>
                  )}
                </TabPanel>

                <TabPanel value={activeTab} index={1}>
                  {semanticFields.length > 0 ? (
                    <TableContainer component={Paper} variant="outlined">
                      <Table size="small">
                        <TableHead>
                          <TableRow>
                            <TableCell>Field Name</TableCell>
                            <TableCell>Semantic Term</TableCell>
                            <TableCell>Name</TableCell>
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
                    <Typography color="text.secondary">No semantic field mappings found.</Typography>
                  )}
                </TabPanel>
              </>
            )}
           </>
        ) : (
          <Box sx={{ p: 2 }}>
             <Alert severity="warning">
               This Business Object is not mapped to a Driver Table. Relationships cannot be automatically discovered.
             </Alert>
          </Box>
        )}
      </DialogContent>
      
      <DialogActions>
        <Button onClick={onClose}>Close</Button>
      </DialogActions>
    </Dialog>
  );
};
