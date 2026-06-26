import React, { useState, useMemo } from 'react';
import {
  Box,
  Button,
  Card,
  CardContent,
  CardHeader as _CardHeader,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  Grid,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  TextField,
  Typography,
  Alert,
  InputAdornment,
} from '@mui/material';
import _ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import SearchIcon from '@mui/icons-material/Search';
import StorageIcon from '@mui/icons-material/Storage';
import LinkIcon from '@mui/icons-material/Link';
import CallReceivedIcon from '@mui/icons-material/CallReceived';
import FolderIcon from '@mui/icons-material/Folder';

/**
 * Field metadata for validation
 */
export interface FieldMetadata {
  name: string;
  dataType: 'string' | 'number' | 'date' | 'boolean' | 'object' | 'array';
  nullable: boolean;
  format?: string; // e.g., 'email', 'phone', 'uuid', 'iso-date'
  maxLength?: number;
  precision?: number; // for numbers
  relatedEntity?: string; // for foreign keys
  description?: string;
}

/**
 * Entity definition for relationship browsing
 */
export interface EntityDefinition {
  name: string;
  displayName: string;
  fields: FieldMetadata[];
  relationships: RelationshipDefinition[];
  description?: string;
}

/**
 * Relationship between entities
 */
export interface RelationshipDefinition {
  name: string;
  targetEntity: string;
  cardinality: 'one-to-one' | 'one-to-many' | 'many-to-many';
  foreignKeyField: string;
}

interface AdvancedFieldSelectorProps {
  onFieldSelected: (fieldPath: string, metadata: FieldMetadata) => void;
  entities: EntityDefinition[];
  currentEntity?: string; // Pre-selected entity
}

/**
 * Advanced Field Selector Component
 * 
 * Provides:
 * - Visual entity relationship browser
 * - Dot notation support (employee.department.name)
 * - Related entity traversal
 * - Field metadata display
 * - Search across all fields
 */
const AdvancedFieldSelector: React.FC<AdvancedFieldSelectorProps> = ({
  onFieldSelected,
  entities,
  currentEntity,
}) => {
  const [open, setOpen] = useState(false);
  const [selectedEntity, setSelectedEntity] = useState<string>(currentEntity || '');
  const [searchQuery, setSearchQuery] = useState('');
  const [fieldPath, setFieldPath] = useState<string[]>([]);
  const [selectedField, setSelectedField] = useState<FieldMetadata | null>(null);

  // Find entity definition by name
  const getEntity = (name: string) => entities.find(e => e.name === name);

  // Get current entity definition
  const currentEntityDef = selectedEntity ? getEntity(selectedEntity) : null;

  // Get fields for current level (could be root entity or related entity)
  const getCurrentFields = () => {
    let entity = currentEntityDef;
    
    // Traverse through relationship path
    for (const pathSegment of fieldPath) {
      if (!entity) break;
      const relationship = entity.relationships.find(r => r.name === pathSegment);
      if (relationship) {
        entity = getEntity(relationship.targetEntity);
      }
    }
    
    return entity?.fields || [];
  };

  // Get current relationships
  const getCurrentRelationships = () => {
    let entity = currentEntityDef;
    
    // Traverse through relationship path
    for (const pathSegment of fieldPath) {
      if (!entity) break;
      const relationship = entity.relationships.find(r => r.name === pathSegment);
      if (relationship) {
        entity = getEntity(relationship.targetEntity);
      }
    }
    
    return entity?.relationships || [];
  };

  const currentFields = getCurrentFields();
  const currentRelationships = getCurrentRelationships();

  // Filter fields based on search
  const filteredFields = useMemo(() => {
    if (!searchQuery) return currentFields;
    return currentFields.filter(f =>
      f.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      f.description?.toLowerCase().includes(searchQuery.toLowerCase())
    );
  }, [currentFields, searchQuery]);

  // Handle field selection
  const handleSelectField = (field: FieldMetadata) => {
    const fullPath = [...fieldPath, field.name].join('.');
    setSelectedField(field);
    onFieldSelected(fullPath, field);
    handleClose();
  };

  // Handle navigating to related entity
  const handleSelectRelationship = (relationship: RelationshipDefinition) => {
    setFieldPath([...fieldPath, relationship.name]);
  };

  // Handle going back in path
  const handleBack = () => {
    setFieldPath(fieldPath.slice(0, -1));
    setSearchQuery('');
  };

  // Handle close
  const handleClose = () => {
    setOpen(false);
    setFieldPath([]);
    setSearchQuery('');
  };

  // Render data type badge
  const renderDataTypeBadge = (dataType: string) => {
    const colors: Record<string, 'default' | 'primary' | 'secondary'> = {
      string: 'default',
      number: 'primary',
      date: 'secondary',
      boolean: 'default',
      object: 'primary',
      array: 'secondary',
    };
    return <Chip label={dataType} size="small" color={colors[dataType] || 'default'} />;
  };

  return (
    <Box>
      {/* Trigger Button */}
      <Button
        variant="outlined"
        endIcon={<FolderIcon />}
        onClick={() => setOpen(true)}
        fullWidth
        sx={{ justifyContent: 'flex-start' }}
      >
        {selectedField ? `${selectedEntity}.${fieldPath.join('.')}.${selectedField.name}` : 'Select Field...'}
      </Button>

      {/* Field Selector Dialog */}
      <Dialog open={open} onClose={handleClose} maxWidth="md" fullWidth>
        <DialogTitle>
          Advanced Field Selector
          {fieldPath.length > 0 && (
            <Typography variant="caption" display="block" sx={{ mt: 1 }}>
              Path: {selectedEntity}.{fieldPath.join('.')}
            </Typography>
          )}
        </DialogTitle>

        <DialogContent sx={{ pt: 2 }}>
          {/* Entity Selection (if no entity selected yet) */}
          {!selectedEntity && (
            <Box sx={{ mb: 3 }}>
              <Typography variant="subtitle2" sx={{ mb: 2 }}>
                Select an Entity
              </Typography>
              <Grid container spacing={2}>
                {entities.map(entity => (
                  <Grid item xs={12} sm={6} md={4} key={entity.name}>
                    <Card
                      sx={{
                        cursor: 'pointer',
                        '&:hover': { boxShadow: 3 },
                      }}
                      onClick={() => setSelectedEntity(entity.name)}
                    >
                      <CardContent sx={{ pb: 1 }}>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                          <StorageIcon />
                          <Typography variant="h6" sx={{ fontSize: '1rem' }}>
                            {entity.displayName}
                          </Typography>
                        </Box>
                        <Typography variant="caption" color="textSecondary">
                          {entity.fields.length} fields
                        </Typography>
                      </CardContent>
                    </Card>
                  </Grid>
                ))}
              </Grid>
            </Box>
          )}

          {/* Fields & Relationships Display */}
          {selectedEntity && (
            <Box>
              {/* Breadcrumb Navigation */}
              {fieldPath.length > 0 && (
                <Box sx={{ mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
                  <Button size="small" onClick={() => setSelectedEntity('')}>
                    {selectedEntity}
                  </Button>
                  {fieldPath.map((segment, idx) => (
                    <Box key={idx} sx={{ display: 'flex', alignItems: 'center' }}>
                      <Typography sx={{ mx: 1 }}>→</Typography>
                      <Button
                        size="small"
                        onClick={() => setFieldPath(fieldPath.slice(0, idx))}
                      >
                        {segment}
                      </Button>
                    </Box>
                  ))}
                </Box>
              )}

              {/* Search */}
              <TextField
                fullWidth
                placeholder="Search fields..."
                value={searchQuery}
                onChange={e => setSearchQuery(e.target.value)}
                InputProps={{
                  startAdornment: (
                    <InputAdornment position="start">
                      <SearchIcon />
                    </InputAdornment>
                  ),
                }}
                sx={{ mb: 2 }}
              />

              {/* Fields List */}
              <Typography variant="subtitle2" sx={{ mb: 1 }}>
                Direct Fields ({filteredFields.length})
              </Typography>
              <List>
                {filteredFields.map(field => (
                  <ListItem
                    key={field.name}
                    secondaryAction={renderDataTypeBadge(field.dataType)}
                    sx={{ mb: 1, border: 1, borderColor: 'divider', borderRadius: 1 }}
                    disablePadding
                  >
                    <ListItemButton onClick={() => handleSelectField(field)}>
                      <ListItemIcon>
                        <CallReceivedIcon />
                      </ListItemIcon>
                      <ListItemText
                        primary={field.name}
                        secondary={
                          <>
                            <Typography component="span" variant="body2" color="textSecondary">
                              {field.dataType}
                              {field.format && ` (${field.format})`}
                            </Typography>
                            {field.description && (
                              <Typography component="span" variant="caption" display="block">
                                {field.description}
                              </Typography>
                            )}
                          </>
                        }
                      />
                    </ListItemButton>
                  </ListItem>
                ))}
              </List>

              {filteredFields.length === 0 && searchQuery && (
                <Alert severity="info">No fields match your search</Alert>
              )}

              <Divider sx={{ my: 2 }} />

              {/* Related Entities */}
              {currentRelationships.length > 0 && (
                <Box>
                  <Typography variant="subtitle2" sx={{ mb: 1 }}>
                    Related Entities ({currentRelationships.length})
                  </Typography>
                  <List>
                    {currentRelationships.map(rel => (
                      <ListItem
                        key={rel.name}
                        secondaryAction={
                          <Chip
                            label={rel.cardinality}
                            size="small"
                            variant="outlined"
                          />
                        }
                        sx={{ mb: 1, border: 1, borderColor: 'divider', borderRadius: 1 }}
                        disablePadding
                      >
                        <ListItemButton onClick={() => handleSelectRelationship(rel)}>
                          <ListItemIcon>
                            <LinkIcon />
                          </ListItemIcon>
                          <ListItemText
                            primary={`${rel.name} → ${rel.targetEntity}`}
                            secondary={`via ${rel.foreignKeyField}`}
                          />
                        </ListItemButton>
                      </ListItem>
                    ))}
                  </List>
                </Box>
              )}

              {currentRelationships.length === 0 && fieldPath.length === 0 && (
                <Typography variant="body2" color="textSecondary">
                  No related entities
                </Typography>
              )}
            </Box>
          )}
        </DialogContent>

        <DialogActions>
          {fieldPath.length > 0 && (
            <Button onClick={handleBack}>Back</Button>
          )}
          <Box sx={{ flex: 1 }} />
          <Button onClick={handleClose}>Close</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default AdvancedFieldSelector;
