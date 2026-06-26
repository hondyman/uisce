// @ts-nocheck
import { useState, useCallback, useEffect } from 'react';
import type { FC } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Tabs,
  Tab,
  Chip,
  Tooltip,
  Card,
  CardContent,
  Button,
  CircularProgress,
  Box,
  Alert,
  TextField,
  FormControlLabel,
  Checkbox,
} from '@mui/material';
import { ArrowForward, Link as LinkIcon, Refresh as RefreshIcon } from '@mui/icons-material';
import './RelationshipDiscoveryModal.module.css';
import RelationshipPathVisualizer from './RelationshipPathVisualizer';

interface EnhancedRelatedEntity {
  entity_id: string;
  entity_name: string;
  table_name: string;
  link_type: 'DIRECT_FK' | 'SEMANTIC' | 'MULTI_HOP';
  cardinality: '1:1' | '1:N' | 'N:1' | 'N:M';
  confidence: number;
  confidence_reason: string;
  foreign_key_path: string;
  semantic_term_name?: string;
}

interface RelationshipPath {
  path_id: string;
  source_entity_id: string;
  target_entity_id: string;
  hierarchy_depth: number;
  hops: Array<{
    order: number;
    entity_id: string;
    entity_name: string;
    link_type: string;
    cardinality: string;
  }>;
  total_confidence: number;
  total_cardinality: string;
}

interface RelationshipDiscoveryModalProps {
  visible: boolean;
  entityAttributeId: string;
  entityName: string;
  tenantId: string;
  datasourceId: string;
  onClose: () => void;
  onApplyRelationship: (relationship: EnhancedRelatedEntity) => Promise<void>;
}

const RelationshipDiscoveryModal: FC<RelationshipDiscoveryModalProps> = ({
  visible,
  entityAttributeId,
  entityName,
  tenantId,
  datasourceId,
  onClose,
  onApplyRelationship,
}) => {
  const [tabValue, setTabValue] = useState(0);
  const [loading, setLoading] = useState(false);
  const [directRelationships, setDirectRelationships] = useState<EnhancedRelatedEntity[]>([]);
  const [multiHopPaths, setMultiHopPaths] = useState<RelationshipPath[]>([]);
  const [selectedRelationship, setSelectedRelationship] = useState<EnhancedRelatedEntity | null>(
    null
  );
  const [maxHopDepth, setMaxHopDepth] = useState(3);
  const [includeMultiHop, setIncludeMultiHop] = useState(true);
  const [applying, setApplying] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Discover relationships
  const discoverRelationships = useCallback(async () => {
    if (!entityAttributeId) {
      setError('Entity ID is required');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const response = await fetch('/api/relationships/discover', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
          'X-Tenant-Datasource-ID': datasourceId,
        },
        body: JSON.stringify({
          entity_attribute_id: entityAttributeId,
          include_multi_hop: includeMultiHop,
          max_hop_depth: maxHopDepth,
        }),
      });

      if (!response.ok) {
        throw new Error(`Failed to discover relationships: ${response.statusText}`);
      }

      const data = await response.json();
      setDirectRelationships(data.direct_relationships || []);
      setMultiHopPaths(data.multi_hop_paths || []);

      if (data.direct_relationships?.length === 0 && data.multi_hop_paths?.length === 0) {
        setError('No relationships discovered. This entity may be standalone.');
      }
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMsg);
    } finally {
      setLoading(false);
    }
  }, [entityAttributeId, tenantId, datasourceId, includeMultiHop, maxHopDepth]);

  // Auto-discover on modal open
  useEffect(() => {
    if (visible) {
      discoverRelationships();
    }
  }, [visible, discoverRelationships]);

  // Apply selected relationship
  const handleApplyRelationship = async (relationship: EnhancedRelatedEntity) => {
    setApplying(true);
    setError(null);

    try {
      const response = await fetch('/api/relationships/apply', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
          'X-Tenant-Datasource-ID': datasourceId,
        },
        body: JSON.stringify({
          sourceEntity: entityAttributeId,
          targetEntity: relationship.entity_id,
          edgeType: relationship.link_type,
          cardinality: relationship.cardinality,
          confidence: relationship.confidence,
          foreignKeyPath: relationship.foreign_key_path,
        }),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.message || `Failed to apply relationship: ${response.status}`);
      }

      await onApplyRelationship(relationship);
      setSelectedRelationship(null);
      setError(null);
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to apply relationship';
      setError(errorMsg);
    } finally {
      setApplying(false);
    }
  };

  // Render confidence chip with color
  const renderConfidenceChip = (confidence: number) => {
    let color: 'success' | 'warning' | 'error' = 'success';
    if (confidence < 0.7) {
      color = 'warning';
    }
    if (confidence < 0.5) {
      color = 'error';
    }

    return (
      <Tooltip title={`Confidence: ${(confidence * 100).toFixed(0)}%`}>
        <Chip
          label={`${(confidence * 100).toFixed(0)}%`}
          color={color}
          variant="outlined"
          size="small"
        />
      </Tooltip>
    );
  };

  // Render link type chip
  const renderLinkTypeChip = (linkType: string) => {
    const colorMap: { [key: string]: string } = {
      DIRECT_FK: '#2196f3',
      SEMANTIC: '#9c27b0',
      MULTI_HOP: '#ff9800',
    };

    return (
      <Chip
        label={linkType}
        size="small"
        sx={{ backgroundColor: colorMap[linkType] || '#9e9e9e', color: 'white' }}
      />
    );
  };

  return (
    <Dialog open={visible} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>Discover Relationships for: {entityName}</DialogTitle>
      <DialogContent dividers>
        {error && (
          <Alert severity="error" className="mb-4">
            {error}
          </Alert>
        )}

        <Tabs value={tabValue} onChange={(_, newValue) => setTabValue(newValue)}>
          <Tab
            label={
              <Box className="flex items-center gap-2">
                Direct Relationships
                <Chip
                  label={directRelationships.length}
                  size="small"
                  color="success"
                  variant="outlined"
                />
              </Box>
            }
          />
          <Tab
            label={
              <Box className="flex items-center gap-2">
                Multi-Hop Paths
                <Chip
                  label={multiHopPaths.length}
                  size="small"
                  color="warning"
                  variant="outlined"
                />
              </Box>
            }
          />
        </Tabs>

        {/* Direct Relationships Tab */}
        {tabValue === 0 && (
          <Box className="mt-4">
            {loading ? (
              <Box className="flex justify-center py-8">
                <CircularProgress />
              </Box>
            ) : directRelationships.length === 0 ? (
              <Box className="py-8 text-center text-gray-500">
                No direct relationships found
              </Box>
            ) : (
              <Box className="space-y-4">
                {directRelationships.map((rel) => (
                  <Card
                    key={`${rel.entity_id}-${rel.link_type}`}
                    className={`cursor-pointer transition-all ${
                      selectedRelationship?.entity_id === rel.entity_id
                        ? 'ring-2 ring-blue-500'
                        : 'hover:shadow-lg'
                    }`}
                    onClick={() => setSelectedRelationship(rel)}
                  >
                    <CardContent>
                      <Box className="mb-3 flex items-center justify-between">
                        <Box className="flex items-center gap-2 text-sm font-medium">
                          <span>{entityName}</span>
                          <ArrowForward fontSize="small" />
                          <span>{rel.entity_name}</span>
                        </Box>
                        <Box className="flex gap-2">
                          {renderLinkTypeChip(rel.link_type)}
                          {renderConfidenceChip(rel.confidence)}
                        </Box>
                      </Box>

                      <Box className="space-y-2 text-sm mb-4">
                        <Box className="flex justify-between">
                          <span className="font-medium text-gray-600">Cardinality:</span>
                          <span>{rel.cardinality}</span>
                        </Box>
                        <Box className="flex justify-between">
                          <span className="font-medium text-gray-600">FK Path:</span>
                          <code className="text-xs bg-gray-100 px-2 py-1 rounded">
                            {rel.foreign_key_path}
                          </code>
                        </Box>
                        {rel.semantic_term_name && (
                          <Box className="flex justify-between">
                            <span className="font-medium text-gray-600">Semantic:</span>
                            <span>{rel.semantic_term_name}</span>
                          </Box>
                        )}
                        <Box className="flex justify-between">
                          <span className="font-medium text-gray-600">Reason:</span>
                          <span>{rel.confidence_reason}</span>
                        </Box>
                      </Box>

                      <Button
                        variant="contained"
                        size="small"
                        disabled={applying && selectedRelationship?.entity_id === rel.entity_id}
                        onClick={(e) => {
                          e.stopPropagation();
                          handleApplyRelationship(rel);
                        }}
                      >
                        {applying && selectedRelationship?.entity_id === rel.entity_id ? (
                          <CircularProgress size={16} className="mr-2" />
                        ) : null}
                        Apply
                      </Button>
                    </CardContent>
                  </Card>
                ))}
              </Box>
            )}
          </Box>
        )}

        {/* Multi-Hop Paths Tab */}
        {tabValue === 1 && (
          <Box className="mt-4">
            <Box className="mb-4 space-y-3">
              <FormControlLabel
                control={
                  <Checkbox
                    checked={includeMultiHop}
                    onChange={(e) => setIncludeMultiHop(e.target.checked)}
                  />
                }
                label="Include multi-hop paths"
              />
              <TextField
                type="number"
                label="Max hop depth"
                size="small"
                inputProps={{ min: 1, max: 5 }}
                value={maxHopDepth}
                onChange={(e) => setMaxHopDepth(parseInt(e.target.value))}
              />
            </Box>

            {loading ? (
              <Box className="flex justify-center py-8">
                <CircularProgress />
              </Box>
            ) : multiHopPaths.length === 0 ? (
              <Box className="py-8 text-center text-gray-500">
                No multi-hop paths found
              </Box>
            ) : (
              <Box className="space-y-4">
                {multiHopPaths.map((path) => (
                  <Card key={path.path_id}>
                    <CardContent>
                      <RelationshipPathVisualizer
                        path={path}
                        onApply={() => {
                          setError('Multi-hop relationship apply not yet implemented');
                        }}
                      />
                    </CardContent>
                  </Card>
                ))}
              </Box>
            )}
          </Box>
        )}

        {/* Selected Relationship Preview */}
        {selectedRelationship && (
          <Card className="mt-6 bg-blue-50">
            <CardContent>
              <Box className="flex items-center gap-2 mb-3">
                <LinkIcon fontSize="small" />
                <h4 className="font-bold">Selected Relationship</h4>
              </Box>
              <Box className="mb-3 text-sm">
                <span className="font-medium">{entityName}</span>
                <ArrowForward className="inline mx-2" fontSize="small" />
                <span className="font-medium">{selectedRelationship.entity_name}</span>
              </Box>
              <Box className="space-y-1 text-sm">
                <Box>
                  <strong>Type:</strong> {selectedRelationship.link_type}
                </Box>
                <Box>
                  <strong>Cardinality:</strong> {selectedRelationship.cardinality}
                </Box>
                <Box>
                  <strong>Confidence:</strong> {(selectedRelationship.confidence * 100).toFixed(0)}%
                </Box>
              </Box>
            </CardContent>
          </Card>
        )}
      </DialogContent>

      <DialogActions>
        <Button onClick={onClose}>Close</Button>
        <Button
          variant="contained"
          startIcon={<RefreshIcon />}
          onClick={discoverRelationships}
          disabled={loading}
        >
          Refresh Discovery
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default RelationshipDiscoveryModal;
