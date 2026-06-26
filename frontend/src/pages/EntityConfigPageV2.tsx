// @ts-nocheck
import { useState, useMemo, useEffect, useRef } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import './EntityConfigPageV2.css';
import {
  Card,
  Button,
  TextField,
  Grid,
  Stack,
  Box,
  Tooltip,
  Chip,
  Typography,
  Badge,
  Divider,
} from '@mui/material';
import {
  Plus as PlusOutlined,
  Edit as EditOutlined,
  Search as SearchOutlined,
  Save as SaveOutlined,
  Link as LinkOutlined,
} from 'lucide-react';
import { saveEntitySchema, fetchEntitySchema } from '../api/entitySchema';
import type { Entities, Entity, Subtype as _Subtype, Field as _Field } from '../types/entity-schema';
import { devLog } from '../utils/devLogger';
import { hasTenantScope } from '../utils/tenantScope';
import ProfessionalSearchInput from '../components/common/ProfessionalSearchInput';
import { businessToTechnicalName as _businessToTechnicalName, technicalToBusinessName as _technicalToBusinessName, normalizeName } from '../utils/nameFormatting';
import { useTenant } from '../contexts/TenantContext';
import { useConfirm } from '../components/ConfirmProvider';
import { useNotification } from '../hooks/useNotification';
// Entity details are now opened on their own page; modal component removed from this page

/* eslint-disable */

// Core Business Objects (Workday-style seed data)
// Empty entities object - will be populated entirely from API/database
const INITIAL_ENTITIES: Entities = {};

interface SemanticTerm {
  id: string;
  node_name: string;
  description: string;
}

interface EditingEntityState {
  entityKey: string;
  selectedSubtypeKey?: string; // Currently viewing which subtype
}

interface FieldEditState {
  entityKey: string;
  subtypeKey?: string;
  fieldKey?: string;
  level: 'entity' | 'subtype';
}

// Types for validation rules

export default function EntityConfigPageV2() {
  const [entities, setEntities] = useState<Entities>(INITIAL_ENTITIES);
  const [initialEntities, setInitialEntities] = useState<Entities>(INITIAL_ENTITIES);
  const [searchTerm, setSearchTerm] = useState('');
  const [isSaving, setIsSaving] = useState(false);
  const [editingState, setEditingState] = useState<EditingEntityState | null>(null);
  const [semanticTerms, setSemanticTerms] = useState<SemanticTerm[]>([]);
  const [loadingSemanticTerms, setLoadingSemanticTerms] = useState(false);
  const [formValues, setFormValues] = useState({ name: '', description: '' });
  const { tenant, datasource } = useTenant();
  const navigate = useNavigate();
  const params = useParams();
  const editorRef = useRef<HTMLDivElement | null>(null);

  // If URL includes an entityKey param, set editing state so the UI can focus that entity
  useEffect(() => {
    // const key = params.entityKey;
    // if (key && (key === 'new' || entities[key])) {
    //   // open the schema tab and set editing state
    //   setMainViewTab('schema');
    //   setEditingState({ entityKey: key });
    //   // scroll the inline editor into view after render
    //   setTimeout(() => {
    //     if (editorRef.current) editorRef.current.scrollIntoView({ behavior: 'smooth', block: 'start' });
    //   }, 150);
    // }
  }, [params.entityKey]);

  // Load saved schema from backend on mount
  useEffect(() => {
    const loadSchema = async () => {
      if (!hasTenantScope()) {
        devLog('[EntityConfigPageV2] No tenant scope, using core BOs');
        return;
      }

      try {
        devLog('[EntityConfigPageV2] Loading schema from backend');
        const savedSchema = await fetchEntitySchema(tenant?.id, datasource?.id);

        if (Object.keys(savedSchema).length > 0) {
          devLog('[EntityConfigPageV2] Schema loaded:', { savedSchema });
          // Use schema directly from API/database (no merging with hardcoded data)
          setInitialEntities(savedSchema);
          setEntities(savedSchema);
        }
      } catch (error) {
        devLog('[EntityConfigPageV2] Error loading schema:', { error });
      }
    };

    loadSchema();
  }, [tenant?.id, datasource?.id]);

  // Load semantic terms for field selection
  useEffect(() => {
    const loadSemanticTerms = async () => {
      setLoadingSemanticTerms(true);
      try {
        // TODO: Implement GraphQL call to fetch semantic terms
        // For now, using mock data
        setSemanticTerms([
          { id: '1', node_name: 'Customer Profile', description: 'Customer identity and demographics' },
          { id: '2', node_name: 'Account Balance', description: 'Current account balance' },
          { id: '3', node_name: 'Transaction History', description: 'Historical transactions' },
        ]);
      } catch (error) {
        devLog('[EntityConfigPageV2] Error loading semantic terms:', { error });
      } finally {
        setLoadingSemanticTerms(false);
      }
    };

    if (hasTenantScope()) {
      loadSemanticTerms();
    }
  }, []);

  // Compute changes (delta)
  const computeChanges = useMemo(() => {
    const changed: string[] = [];
    const deleted: string[] = [];

    for (const key of Object.keys(entities)) {
      if (!(key in initialEntities)) {
        changed.push(key);
      } else if (JSON.stringify(entities[key]) !== JSON.stringify(initialEntities[key])) {
        changed.push(key);
      }
    }

    for (const key of Object.keys(initialEntities)) {
      if (!(key in entities)) {
        deleted.push(key);
      }
    }

    return { changed, deleted };
  }, [entities, initialEntities]);

  // Filter entities by search term
  const filteredEntities = useMemo(() => {
    const term = searchTerm.toLowerCase();
    return Object.fromEntries(
      Object.entries(entities).filter(([key, entity]) => {
        return (
          key.toLowerCase().includes(term) ||
          entity.name.toLowerCase().includes(term) ||
          entity.businessName?.toLowerCase().includes(term) ||
          entity.technicalName?.toLowerCase().includes(term) ||
          entity.description?.toLowerCase().includes(term) ||
          Object.values(entity.subtypes).some(
            (s) => s.name.toLowerCase().includes(term) ||
                   s.businessName?.toLowerCase().includes(term) ||
                   s.technicalName?.toLowerCase().includes(term)
          )
        );
      })
    );
  }, [entities, searchTerm]);

  const saveAndApply = async () => {
    devLog('[saveAndApply] Saving changes...');
    setIsSaving(true);

    try {
      if (!hasTenantScope()) {
        console.error('Please select a tenant first');
        return;
      }

      const { changed, deleted } = computeChanges;
      const changedEntities = Object.fromEntries(
        changed.map((key) => [key, entities[key]])
      );

      const payload = { changed: changedEntities, deleted };
      await saveEntitySchema(payload, tenant?.id, datasource?.id);

      setInitialEntities(entities);
      devDebug(`✅ Saved! ${changed.length} changed, ${deleted.length} deleted`);
    } catch (error) {
      devLog('[saveAndApply] Error:', { error });
      console.error('Failed to save schema');
    } finally {
      setIsSaving(false);
    }
  };

  const handleAddEntity = () => {
    // Navigate to the 'new' route which will render the create form inline
    navigate('/entity-config/new');
    setEditingState({ entityKey: 'new' });
  };

  const createEntityFromValues = (values: any) => {
    const { businessName, technicalName } = normalizeName(values.name, undefined);
    const key = technicalName;

    const newEntity: Entity = {
      name: businessName,
      businessName,
      technicalName,
      description: values.description,
      entity_fields: [],
      subtypes: {},
      isCore: false,
      coreFields: [],
      customFields: [],
    };
    setEntities({ ...entities, [key]: newEntity });
    devDebug('Entity created!');
    // navigate to the newly created entity editor
    navigate(`/entity-config/${encodeURIComponent(key)}`);
    setEditingState({ entityKey: key });
    setFormValues({ name: '', description: '' });
  };

  const handleEditEntity = (entityKey: string) => {
    // Navigate to the dedicated entity details page
    navigate(`/entity-config/${encodeURIComponent(entityKey)}`, {
      state: { entities, tenant, datasource },
    });
  };

  const handleRenameEntity = (entityKey: string, newBusinessName: string) => {
    const entity = entities[entityKey];
    const { businessName, technicalName } = normalizeName(newBusinessName, entity.technicalName);

    const updatedEntity: Entity = {
      ...entity,
      name: businessName,
      businessName,
      technicalName,
    };

    setEntities({ ...entities, [entityKey]: updatedEntity });
    devDebug(`Entity renamed to "${businessName}"`);
  };

  const handleCloneEntity = (fromKey: string) => {
    const sourceEntity = entities[fromKey];

    // Find next available clone number
    let cloneNum = 1;
    let newKey = `${fromKey}_custom_${cloneNum}`;
    while (newKey in entities) {
      cloneNum++;
      newKey = `${fromKey}_custom_${cloneNum}`;
    }

    const { businessName: clonedBusinessName, technicalName: clonedTechnicalName } = normalizeName(
      `${sourceEntity.businessName || sourceEntity.name} (Custom)`,
      newKey
    );

    const newEntity: Entity = {
      ...sourceEntity,
      name: clonedBusinessName,
      businessName: clonedBusinessName,
      technicalName: clonedTechnicalName,
      isCore: false,
      clonesFrom: fromKey,
      clonesFromKey: fromKey,
      cloneParentName: sourceEntity.businessName || sourceEntity.name,
      coreFields: sourceEntity.entity_fields.filter((f) => f.isCore),
      customFields: [],
    };

    setEntities({ ...entities, [newKey]: newEntity });
    devDebug(`✅ Cloned "${sourceEntity.businessName || sourceEntity.name}" as new custom entity!`);
  };

  const handleDeleteEntity = (entityKey: string) => {
    const newEntities = { ...entities };
    delete newEntities[entityKey];
    setEntities(newEntities);
    devDebug('Entity deleted');
  };

  // modal-based create handlers removed: subtype/field creation is handled inline in the editor component

  const handleDeleteField = (entityKey: string, fieldKey: string, level: 'entity' | 'subtype', subtypeKey?: string) => {
    const entity = entities[entityKey];
    let updatedEntity = entity;

    if (level === 'entity') {
      updatedEntity = {
        ...updatedEntity,
        entity_fields: updatedEntity.entity_fields.filter((f) => f.key !== fieldKey),
        customFields: updatedEntity.customFields?.filter((f) => f.key !== fieldKey),
      };
    } else if (level === 'subtype' && subtypeKey) {
      const newSubtypes = {
        ...updatedEntity.subtypes,
        [subtypeKey]: {
          ...updatedEntity.subtypes[subtypeKey],
          subtype_fields: updatedEntity.subtypes[subtypeKey].subtype_fields.filter((f) => f.key !== fieldKey),
        },
      };
      updatedEntity = { ...updatedEntity, subtypes: newSubtypes };
    }

    setEntities({ ...entities, [entityKey]: updatedEntity });
    devDebug('Field deleted');
  };

  const handleDeleteSubtype = (entityKey: string, subtypeKey: string) => {
    const entity = entities[entityKey];
    const newSubtypes = { ...entity.subtypes };
    delete newSubtypes[subtypeKey];

    setEntities({
      ...entities,
      [entityKey]: { ...entity, subtypes: newSubtypes },
    });
    setEditingState({ entityKey }); // Reset to entity level
    devDebug('Subtype deleted');
  };

  const selectedEntity = editingState ? entities[editingState.entityKey] : null;

  const handleFormSubmit = () => {
    if (formValues.name.trim()) {
      createEntityFromValues(formValues);
    }
  };

  const handleDeleteConfirm = (entityKey: string) => {
    handleDeleteEntity(entityKey);
  };

  return (
    <Box className="entity-config-root" sx={{ p: 2 }}>
      {/* HEADER CARD */}
      <Card sx={{ mb: 3 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Badge color="primary" variant="dot" />
            <Typography variant="h6">Business Object Manager</Typography>
          </Box>
          <Box sx={{ display: 'flex', gap: 1 }}>
            <Button
              startIcon={<SearchOutlined />}
              onClick={() => setSearchTerm('')}
              size="small"
            >
              Clear
            </Button>
            <Button
              variant="contained"
              startIcon={<SaveOutlined />}
              onClick={saveAndApply}
              disabled={computeChanges.changed.length === 0 && computeChanges.deleted.length === 0}
              size="small"
            >
              SAVE & APPLY ({computeChanges.changed.length + computeChanges.deleted.length})
            </Button>
          </Box>
        </Box>

        {/* Schema Configuration Content */}
          <Box>
            <Box sx={{ mb: 2 }}>
              <ProfessionalSearchInput
                value={searchTerm}
                onChange={setSearchTerm}
                placeholder="Search entities by business/technical name, description, subtypes..."
                onClear={() => setSearchTerm('')}
              />
            </Box>

            <Divider sx={{ my: 2 }} />

            {/* ENTITY CARDS GRID */}
            <Grid container spacing={2}>
              {/* Add New Entity Button */}
              <Grid item xs={12} sm={6} md={4} lg={3}>
                <Card
                  onClick={handleAddEntity}
                  sx={{
                    height: '100%',
                    cursor: 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    minHeight: '200px',
                    '&:hover': { bgcolor: 'action.hover' },
                  }}
                >
                  <Stack direction="column" alignItems="center" spacing={1}>
                    <PlusOutlined style={{ fontSize: '32px' }} />
                    <Typography>Add New Entity</Typography>
                  </Stack>
                </Card>
              </Grid>

              {/* Entity Cards */}
              {Object.entries(filteredEntities).map(([entityKey, entity]) => (
                <Grid item xs={12} sm={6} md={4} lg={3} key={entityKey}>
                  <Card
                    onDoubleClick={() => navigate(`/entity-config/${encodeURIComponent(entityKey)}`)}
                    sx={{
                      height: '100%',
                      cursor: 'pointer',
                      '&:hover': { boxShadow: 3 },
                    }}
                  >
                    {/* Badge + Title */}
                    <Box sx={{ mb: 1 }}>
                      <Chip
                        label={entity.isCore ? '🔒 CORE BO' : '✏️ CUSTOM'}
                        color={entity.isCore ? 'primary' : 'success'}
                        size="small"
                      />
                    </Box>

                    {/* Clone Parent Tracking */}
                    {entity.cloneParentName && (
                      <Box sx={{ mb: 1, fontSize: '0.875rem', color: 'text.secondary' }}>
                        <LinkOutlined /> Cloned from: <strong>{entity.cloneParentName}</strong>
                      </Box>
                    )}

                    {/* Entity Name */}
                    <Typography variant="h6" sx={{ mb: 1 }}>
                      {entity.businessName || entity.name}
                    </Typography>
                    {entity.technicalName && (
                      <Box sx={{ mb: 1, fontSize: '0.875rem' }}>
                        Technical: <code>{entity.technicalName}</code>
                      </Box>
                    )}

                    {/* Description */}
                    {entity.description && (
                      <Typography variant="body2" sx={{ mb: 2, color: 'text.secondary' }}>
                        {entity.description}
                      </Typography>
                    )}

                    {/* Subtypes as chips */}
                    {Object.entries(entity.subtypes).length > 0 && (
                      <Box sx={{ mb: 2 }}>
                        <Typography variant="caption" display="block" sx={{ mb: 1 }}>
                          Subtypes:
                        </Typography>
                        <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap' }}>
                          {Object.entries(entity.subtypes).map(([_subtypeKey, subtype]) => (
                            <Chip
                              key={_subtypeKey}
                              label={subtype.businessName || subtype.name}
                              size="small"
                              color={subtype.isCore ? 'primary' : 'default'}
                              variant="outlined"
                            />
                          ))}
                        </Box>
                      </Box>
                    )}

                    {/* Actions */}
                    <Box sx={{ display: 'flex', gap: 1, mt: 2 }}>
                      <Tooltip title="Edit">
                        <Button
                          size="small"
                          icon={<EditOutlined />}
                          onClick={() => handleEditEntity(entityKey)}
                          variant="text"
                        >
                          Edit
                        </Button>
                      </Tooltip>

                      {entity.isCore && (
                        <Tooltip title="Clone">
                          <Button
                            size="small"
                            onClick={() => handleCloneEntity(entityKey)}
                            variant="text"
                          >
                            Clone
                          </Button>
                        </Tooltip>
                      )}

                      {!entity.isCore && (
                        <Tooltip title="Delete">
                          <Button
                            size="small"
                            onClick={async () => {
                                const confirm = useConfirm();
                                const notification = useNotification();
                                if (await confirm({ title: 'Delete entity', description: 'Delete this entity?' })) {
                                  await handleDeleteConfirm(entityKey);
                                  notification.success('Entity deleted');
                                }
                              }}
                            variant="text"
                            sx={{ color: 'error.main' }}
                          >
                            Delete
                          </Button>
                        </Tooltip>
                      )}
                    </Box>
                  </Card>
                </Grid>
              ))}
            </Grid>

            {Object.keys(filteredEntities).length === 0 && (
              <Box sx={{ textAlign: 'center', py: 5 }}>
                <Typography color="textSecondary">No entities found</Typography>
              </Box>
            )}
          </Box>
      </Card>

      {/* Create New Entity - rendered inline when navigating to /entity-config/new */}
      {editingState && editingState.entityKey === 'new' && (
        <Card sx={{ mt: 3 }}>
          <Typography variant="h6" sx={{ mb: 2 }}>Create New Entity</Typography>
          <Stack spacing={2}>
            <TextField
              label="Business Name"
              placeholder="e.g., Client Investor"
              fullWidth
              value={formValues.name}
              onChange={(e) => setFormValues({ ...formValues, name: e.target.value })}
              autoFocus
            />
            <TextField
              label="Description"
              placeholder="Describe this entity"
              fullWidth
              multiline
              rows={3}
              value={formValues.description}
              onChange={(e) => setFormValues({ ...formValues, description: e.target.value })}
            />
            <Box sx={{ display: 'flex', gap: 1 }}>
              <Button variant="contained" onClick={handleFormSubmit}>
                Create
              </Button>
              <Button
                onClick={() => {
                  setEditingState(null);
                  navigate('/entity-config');
                }}
              >
                Cancel
              </Button>
            </Box>
          </Stack>
        </Card>
      )}

      {/* Floating Save Button */}
      {(computeChanges.changed.length > 0 || computeChanges.deleted.length > 0) && (
        <Box
          sx={{
            position: 'fixed',
            bottom: 24,
            right: 24,
            zIndex: 1000,
          }}
        >
          <Button
            variant="contained"
            size="large"
            startIcon={<SaveOutlined />}
            onClick={saveAndApply}
            sx={{ fontSize: '16px', px: 3 }}
          >
            SAVE & APPLY
          </Button>
        </Box>
      )}
    </Box>
  );
}