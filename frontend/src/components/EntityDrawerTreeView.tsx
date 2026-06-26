// @ts-nocheck
import React, { useState, useMemo, useRef, useEffect } from 'react';
import {
  Button,
  Chip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Box,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Tooltip,
  InputAdornment,
} from '@mui/material';
import { SimpleTreeView as TreeView, TreeItem } from '@mui/x-tree-view';
import {
  ChevronUp,
  ChevronDown,
  Plus,
  Trash2,
  Search,
  CheckCircle,
  PlusCircle,
} from 'lucide-react';

import type { Entities, Entity, Field } from '../types/entity-schema';
import { useEnhancedSemanticTerms, semanticTermToField, searchSemanticTerms } from '../hooks/useEnhancedSemanticTerms';
import { normalizeName } from '../utils/nameFormatting';
import { devLog, devWarn, devError } from '../utils/devLogger';
import { useConfirm } from './ConfirmProvider';
import { useNotification } from '../hooks/useNotification';
import styles from './EntityDrawerTreeView.module.css';
import { createEvent } from '../api/events';
import { saveEntitySchema } from '../api/entitySchema';
import { useTenant } from '../contexts/TenantContext';
import { useSnackbar } from 'notistack';

interface EntityDrawerTreeViewProps {
  entityKey: string;
  entity: Entity;
  entities: Entities;
  datasourceId?: string;
  onEntityUpdate: (updatedEntity: Entity) => void;
  validationRules?: any[];
}

interface SelectedNode {
  type: 'entity' | 'subtype';
  subtypeKey?: string;
}

interface HierarchyNode {
  title: string | React.ReactNode;
  key: string;
  children?: HierarchyNode[];
}

export default function EntityDrawerTreeView({
  entityKey,
  entity,
  entities,
  datasourceId,
  onEntityUpdate,
  validationRules,
  // allow parent to request focus when this editor is mounted via route navigation
  focusOnMount,
}: EntityDrawerTreeViewProps & { focusOnMount?: boolean }) {
  const [selectedNode, setSelectedNode] = useState<SelectedNode>({ type: 'entity' });
  const [editingEntity, setEditingEntity] = useState<Entity>(entity);
  const [semanticSearchTerm, setSemanticSearchTerm] = useState('');
  const [showFieldModal, setShowFieldModal] = useState(false);
  const [selectedFieldTarget, setSelectedFieldTarget] = useState<{ subtypeKey?: string } | null>(null);
  const [selectedTermIds, setSelectedTermIds] = useState<string[]>([]);
  const [addSubtypeModalVisible, setAddSubtypeModalVisible] = useState(false);
  const [addSubtypeFormData, setAddSubtypeFormData] = useState({ name: '', description: '' });
  const [addSubtypeFormErrors, setAddSubtypeFormErrors] = useState({ name: '', description: '' });
  const [inheritedExpanded, setInheritedExpanded] = useState(false);
  const [showValidationRulesModal, setShowValidationRulesModal] = useState(false);
  const [selectedFieldForRules, setSelectedFieldForRules] = useState<any>(null);
  const saveButtonRef = useRef<HTMLButtonElement | null>(null);
  const containerRef = useRef<HTMLDivElement | null>(null);

  const { semanticTerms } = useEnhancedSemanticTerms(datasourceId);
  const { tenant, datasource } = useTenant();
  const { enqueueSnackbar } = useSnackbar();
  const confirm = useConfirm();
  const notification = useNotification();

  // Build hierarchy tree
  const hierarchyTree: HierarchyNode[] = useMemo(() => [
    {
      title: (
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', width: '100%' }}>
          <strong>{editingEntity.businessName || editingEntity.name}</strong>
          <Chip label="core" color="primary" size="small" style={{ marginLeft: '8px' }} />
        </div>
      ),
      key: 'entity',
      children: editingEntity.subtypes
        ? Object.entries(editingEntity.subtypes).map(([subtypeKey, subtype]) => ({
            title: (
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', width: '100%' }}>
                <strong>{subtype.businessName || subtype.name}</strong>
                <div style={{ display: 'flex', alignItems: 'center' }}>
                  <Chip
                    label={subtype.isCore ? 'core' : 'custom'} 
                    color={subtype.isCore ? 'primary' : 'success'} 
                    size="small" 
                    style={{ marginLeft: '8px' }}
                  />
                  {!subtype.isCore && (
                  <Button
                    type="text"
                    size="small"
                    color="error"
                    icon={<Trash2 />}
                    onClick={async (e) => {
                      e.stopPropagation();
                      if (!(await confirm({ title: 'Delete subtype', description: 'Delete this subtype? This will permanently remove the subtype and all its fields.' }))) return;
                      handleDeleteSubtype(subtypeKey);
                      notification.success('Subtype deleted');
                    }}
                    style={{ marginLeft: '8px' }}
                  />
                  )}
                </div>
              </div>
            ),
            key: subtypeKey,
          }))
        : [],
    },
  ], [editingEntity]);

  // focus the first meaningful control inside the editor when mounted via route navigation
  useEffect(() => {
    if (!focusOnMount) return;

    // allow layout to render
    const t = setTimeout(() => {
      try {
        const root = containerRef.current as HTMLElement | null;
        if (root) {
          // query for input/select/textarea or any button inside the editor
          const el = root.querySelector<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement | HTMLButtonElement>('input:not([type=hidden]), textarea, select, button');
          if (el) {
            (el as HTMLElement).focus();
            return;
          }
        }

        // fallback
        saveButtonRef.current?.focus();
      } catch (e) {
        // ignore
        saveButtonRef.current?.focus();
      }
    }, 150);

    return () => clearTimeout(t);
  }, [focusOnMount]);

  // Sync editingEntity when entity prop changes
  useEffect(() => {
    setEditingEntity(entity);
    // when the entity changes, always reset the view to the top-level entity
    setSelectedNode({ type: 'entity' });
    devLog('EntityDrawerTreeView: entity prop changed, updating editingEntity:', entity);
  }, [entity, entityKey]);

  // Debug initial entity state
  // Debug initial entity state
  useEffect(() => {
    devLog('EntityDrawerTreeView mounted with entity:', entity);
    devLog('Entity has subtypes:', entity.subtypes ? Object.keys(entity.subtypes) : 'none');
  }, []); // Empty dependency array for mount only

  // Get fields for selected node
  const getSelectedFields = () => {
    if (selectedNode.type === 'entity') {
      return {
        inherited: [],
        assigned: editingEntity.entity_fields || [],
      };
    }

    if (selectedNode.type === 'subtype' && selectedNode.subtypeKey) {
      const subtype = editingEntity.subtypes?.[selectedNode.subtypeKey];
      if (!subtype) {
        devWarn('Subtype not found:', selectedNode.subtypeKey, 'Available subtypes:', Object.keys(editingEntity.subtypes || {}));
        return { inherited: [], assigned: [] };
      }

      devLog('Selected subtype:', selectedNode.subtypeKey);
      devLog('Subtype data:', subtype);
      devLog('Subtype entity_fields:', subtype.entity_fields);
      devLog('Subtype subtype_fields:', subtype.subtype_fields);

      return {
        inherited: editingEntity.entity_fields || [],
        assigned: subtype.subtype_fields || [],
      };
    }

    return { inherited: [], assigned: [] };
  };

  const { inherited, assigned } = getSelectedFields();

  // Helper to get validation rules for a specific field
  const getValidationRulesForField = (field: Field) => {
    if (!validationRules || validationRules.length === 0) return [];
    
    // Filter rules that target this field
    // Rules can have field-level conditions stored in condition_json
    return validationRules.filter((rule: any) => {
      if (!rule.condition_json) return false;
      
      try {
        const condition = typeof rule.condition_json === 'string' 
          ? JSON.parse(rule.condition_json) 
          : rule.condition_json;
        
        // Check if rule applies to this field
        return condition?.field === field.key || 
               condition?.field_name === field.technicalName ||
               condition?.fields?.includes(field.key);
      } catch (e) {
        return false;
      }
    });
  };

  const renderTreeItems = (nodes: HierarchyNode[]): React.ReactNode => {
    return nodes.map((node) => (
      <TreeItem key={node.key} itemId={node.key} label={node.title}>
        {node.children && renderTreeItems(node.children)}
      </TreeItem>
    ));
  };

  const handleTreeSelect = (event: React.SyntheticEvent, itemIds: string[]) => {
    if (itemIds.length === 0) return;

    const key = itemIds[0];
    if (key === 'entity') {
      setSelectedNode({ type: 'entity' });
    } else {
      setSelectedNode({ type: 'subtype', subtypeKey: key });
    }
  };

  const handleAddField = (subtypeKey?: string) => {
    devWarn('handleAddField called with subtypeKey:', subtypeKey);
    devWarn('semanticTerms length:', semanticTerms.length);
    setSelectedFieldTarget({ subtypeKey });
    setSelectedTermIds([]); // Reset selected terms
    setSemanticSearchTerm(''); // Reset search
    setShowFieldModal(true);
  };

  const _handleSelectSemanticTerm = (termId: string) => {
    devWarn('handleSelectSemanticTerm called with termId:', termId);
    const term = semanticTerms.find((t) => t.id === termId);
    devWarn('Found term:', term);
    if (!term) return;

    // Check if this semantic term is already mapped in inherited fields (when adding to subtype)
    if (selectedFieldTarget?.subtypeKey) {
      const isAlreadyInherited = inherited.some(field => field.semanticTermId === termId);
      if (isAlreadyInherited) {
        enqueueSnackbar(`Semantic term "${term.businessName || term.node_name}" is already mapped in the parent entity and inherited by this subtype.`, { variant: 'warning' });
        return;
      }
    }

    const newField = semanticTermToField(term, 0);
    // Ensure the field has a unique key
    if (!newField.key) {
      newField.key = `field_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    devWarn('Created new field:', newField);
    const updated = JSON.parse(JSON.stringify(editingEntity));

    if (!selectedFieldTarget?.subtypeKey) {
      // Adding to entity fields
      if (!updated.entity_fields) updated.entity_fields = [];
      updated.entity_fields.push(newField);
    } else {
      // Adding to subtype fields
      const subtypeKey = selectedFieldTarget.subtypeKey;
      if (!updated.subtypes[subtypeKey].subtype_fields) {
        updated.subtypes[subtypeKey].subtype_fields = [];
      }
      updated.subtypes[subtypeKey].subtype_fields.push(newField);
    }

    setEditingEntity(updated);
    setShowFieldModal(false);
    enqueueSnackbar('Field added', { variant: 'success' });
  };

  const handleAddMultipleFields = () => {
    if (selectedTermIds.length === 0) return;

    // Check if any selected semantic terms are already mapped in inherited fields (when adding to subtype)
    if (selectedFieldTarget?.subtypeKey) {
      const alreadyInheritedTerms: string[] = [];
      selectedTermIds.forEach(termId => {
        const isAlreadyInherited = inherited.some(field => field.semanticTermId === termId);
        if (isAlreadyInherited) {
          const term = semanticTerms.find((t) => t.id === termId);
          if (term) {
            alreadyInheritedTerms.push(term.businessName || term.node_name);
          }
        }
      });

      if (alreadyInheritedTerms.length > 0) {
        enqueueSnackbar(`The following semantic terms are already mapped in the parent entity and inherited by this subtype: ${alreadyInheritedTerms.join(', ')}`, { variant: 'warning' });
        return;
      }
    }

    const updated = JSON.parse(JSON.stringify(editingEntity));
    let addedCount = 0;

    selectedTermIds.forEach(termId => {
      const term = semanticTerms.find((t) => t.id === termId);
      if (!term) return;

      const newField = semanticTermToField(term, 0);
      // Ensure the field has a unique key
      if (!newField.key) {
        newField.key = `field_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
      }

      if (!selectedFieldTarget?.subtypeKey) {
        // Adding to entity fields
        if (!updated.entity_fields) updated.entity_fields = [];
        updated.entity_fields.push(newField);
      } else {
        // Adding to subtype fields
        const subtypeKey = selectedFieldTarget.subtypeKey;
        if (!updated.subtypes) updated.subtypes = {};
        if (!updated.subtypes[subtypeKey]) {
          // Subtype doesn't exist, skip adding field
          devWarn(`Cannot add field to non-existent subtype: ${subtypeKey}`);
          return;
        }
        if (!updated.subtypes[subtypeKey].subtype_fields) {
          updated.subtypes[subtypeKey].subtype_fields = [];
        }
        updated.subtypes[subtypeKey].subtype_fields.push(newField);
      }
      addedCount++;
    });

    setEditingEntity(updated);
    setShowFieldModal(false);
    setSelectedTermIds([]);
    enqueueSnackbar(`${addedCount} field${addedCount > 1 ? 's' : ''} added`, { variant: 'success' });
  };

  const handleDeleteField = (fieldKey: string) => {
    const updated = JSON.parse(JSON.stringify(editingEntity));

    if (selectedNode.type === 'entity') {
      updated.entity_fields = (updated.entity_fields || []).filter(
        (f: Field) => f.key !== fieldKey
      );
    } else if (selectedNode.type === 'subtype' && selectedNode.subtypeKey) {
      const subtype = updated.subtypes?.[selectedNode.subtypeKey];
      if (subtype) {
        subtype.subtype_fields = (subtype.subtype_fields || []).filter(
          (f: Field) => f.key !== fieldKey
        );
      }
    }

    setEditingEntity(updated);
    enqueueSnackbar('Field deleted', { variant: 'success' });
  };

  const handleMoveField = (index: number, direction: 'up' | 'down') => {
    const updated = JSON.parse(JSON.stringify(editingEntity));
    const newIndex = direction === 'up' ? index - 1 : index + 1;

    if (selectedNode.type === 'entity') {
      const fields = updated.entity_fields || [];
      [fields[index], fields[newIndex]] = [fields[newIndex], fields[index]];
    } else if (selectedNode.type === 'subtype' && selectedNode.subtypeKey) {
      const subtype = updated.subtypes?.[selectedNode.subtypeKey];
      if (subtype) {
        const fields = subtype.subtype_fields || [];
        [fields[index], fields[newIndex]] = [fields[newIndex], fields[index]];
      }
    }

    setEditingEntity(updated);
  };

  const handleDeleteSubtype = (subtypeKey: string) => {
    const updated = JSON.parse(JSON.stringify(editingEntity));
    if (updated.subtypes && updated.subtypes[subtypeKey]) {
      delete updated.subtypes[subtypeKey];
      setEditingEntity(updated);
      // If the deleted subtype was selected, select the entity instead
      if (selectedNode.type === 'subtype' && selectedNode.subtypeKey === subtypeKey) {
        setSelectedNode({ type: 'entity' });
      }
      enqueueSnackbar('Subtype deleted', { variant: 'success' });
    }
  };

  const handleSave = async () => {
    // detect changes between original entity prop and editingEntity
    try {
      const original = entity;
      const changedFields: Array<{ field: string; oldValue: any; newValue: any }> = [];

      const origFields = (original.entity_fields || []).reduce((acc: any, f: any) => {
        acc[f.key] = f;
        return acc;
      }, {});
      const editFields = (editingEntity.entity_fields || []).reduce((acc: any, f: any) => {
        acc[f.key] = f;
        return acc;
      }, {});

      // compare entity-level fields
      for (const key of Object.keys({ ...origFields, ...editFields })) {
        const o = origFields[key];
        const n = editFields[key];
        const oVal = o ? (o.semanticTermName ?? o.name ?? null) : null;
        const nVal = n ? (n.semanticTermName ?? n.name ?? null) : null;
        if (JSON.stringify(oVal) !== JSON.stringify(nVal)) {
          changedFields.push({ field: key, oldValue: oVal, newValue: nVal });
        }
      }

      // compare subtype fields
      for (const subtypeKey of Object.keys(editingEntity.subtypes || {})) {
        const origSubtype = original.subtypes?.[subtypeKey];
        const editSubtype = editingEntity.subtypes?.[subtypeKey];
        const origSF = (origSubtype?.subtype_fields || []).reduce((acc: any, f: any) => { acc[f.key]=f; return acc; }, {});
        const editSF = (editSubtype?.subtype_fields || []).reduce((acc: any, f: any) => { acc[f.key]=f; return acc; }, {});
        for (const k of Object.keys({ ...origSF, ...editSF })) {
          const o = origSF[k];
          const n = editSF[k];
          const oVal = o ? (o.semanticTermName ?? o.name ?? null) : null;
          const nVal = n ? (n.semanticTermName ?? n.name ?? null) : null;
          if (JSON.stringify(oVal) !== JSON.stringify(nVal)) {
            changedFields.push({ field: `${subtypeKey}.${k}`, oldValue: oVal, newValue: nVal });
          }
        }
      }

      // Create events for each changed field (fire-and-forget)
      for (const ch of changedFields) {
        createEvent({
          bo_type: entityKey,
          bo_id: entity.technicalName || entityKey, // best-effort id
          field_name: ch.field,
          old_value: ch.oldValue,
          new_value: ch.newValue,
          changed_by: 'system',
        }).catch((err) => {
          // log but don't block save
          devWarn('createEvent failed', err);
        });
      }

      onEntityUpdate(editingEntity);
      
      // Save to backend
      devLog && devLog('Saving entity to backend:', { entityKey, editingEntity, hasSubtypes: !!editingEntity.subtypes });
      const updatedEntities = { ...entities, [entityKey]: editingEntity };
      devLog && devLog('Full entities payload:', updatedEntities);
      try {
        await saveEntitySchema(updatedEntities, tenant?.id, datasource?.id);
        devLog && devLog('Entity saved successfully to backend');
      } catch (saveError) {
        devError('Failed to save entity to backend:', saveError);
        throw saveError; // Re-throw so it goes to the catch block below
      }
      
      enqueueSnackbar('Entity updated and saved to backend', { variant: 'success' });
    } catch (err) {
      devError('handleSave error', err);
      
      // Provide user-friendly error message
      let errorMessage = 'Failed to save entity changes';
      if (err instanceof Error) {
        if (err.message.includes('entity_attribute')) {
          errorMessage = 'Backend schema migration needed - please contact support';
        } else if (err.message.includes('404')) {
          errorMessage = 'Backend endpoint not available - please contact support';
        } else {
          errorMessage = `Error: ${err.message}`;
        }
      }
      
      enqueueSnackbar(errorMessage, { variant: 'error' });
    }
  };

  const _fieldColumns = [
    {
      title: 'Business Name',
      dataIndex: 'businessName',
      key: 'businessName',
      width: 120,
    },
    {
      title: 'Technical Name',
      dataIndex: 'technicalName',
      key: 'technicalName',
      width: 120,
    },
    {
      title: 'Type',
      dataIndex: 'type',
      key: 'type',
      width: 80,
    },
    {
      title: 'Semantic Term',
      dataIndex: 'semanticTermName',
      key: 'semanticTermName',
      width: 100,
    },
  ];

  const _assignedFieldColumns = [
    ..._fieldColumns,
    {
      title: 'Actions',
      key: 'actions',
      width: 100,
      render: (_: any, record: Field, index: number) => (
        <Box sx={{ display: 'flex', gap: 1 }}>
          <Tooltip title="Move up">
            <Button
              type="text"
              size="small"
              icon={<ChevronUp />}
              disabled={index === 0}
              onClick={() => handleMoveField(index, 'up')}
            />
          </Tooltip>
          <Tooltip title="Move down">
            <Button
              type="text"
              size="small"
              icon={<ChevronDown />}
              disabled={index === assigned.length - 1}
              onClick={() => handleMoveField(index, 'down')}
            />
          </Tooltip>
          <Button
            type="text"
            color="error"
            size="small"
            icon={<Trash2 />}
            onClick={async () => {
              if (!(await confirm({ title: 'Delete field', description: 'Delete field?' }))) return;
              handleDeleteField(record.key);
              notification.success('Field deleted');
            }}
          />
        </Box>
      ),
    },
  ];

  return (
    <div className={styles.container} ref={containerRef}>
      {/* Main Header with Save Button */}
      <div className={styles.mainHeader}>
        <div className={styles.mainHeaderContent}>
          <h2>{editingEntity.businessName || editingEntity.name}</h2>
        </div>
        <Button type="primary" onClick={handleSave} ref={saveButtonRef} className={styles.saveButton}>
          💾 Save Entity Changes
        </Button>
      </div>

      <Box className={styles.layout} sx={{ display: 'flex', height: '100%' }}>
        {/* Left Sider: Tree Navigation */}
        <Box className={styles.sider}>
          <div className={styles.treeContainer}>
              <div className={styles.treeHeader}>
                <h4>Entity Structure</h4>
                <Button size="small" onClick={() => setAddSubtypeModalVisible(true)}>
                  <Plus /> Subtype
                </Button>
              </div>
              <TreeView
                selectedItems={selectedNode.type === 'entity' ? ['entity'] : [selectedNode.subtypeKey || 'entity']}
                onItemSelectionToggle={(event, itemId, isSelected) => {
                  // only process the selection, not deselection
                  if (isSelected) {
                    // The TreeView component passes an array of IDs to onSelectedItemsChange, but only a single ID to onItemSelectionToggle.
                    // We wrap it in an array to reuse the existing handler.
                    handleTreeSelect(event, [itemId]);
                  }
                }}
                defaultExpandedItems={editingEntity.subtypes && Object.keys(editingEntity.subtypes).length > 0 ? Object.keys(editingEntity.subtypes) : []}
              >
                {renderTreeItems(hierarchyTree)}
              </TreeView>
            </div>
        </Box>

        {/* Right Content: Fields */}
        <Box className={styles.content} sx={{ flex: 1, overflow: 'auto' }}>
          {selectedNode ? (
            <>
              <div className={styles.header}>
                <h3>
                  {selectedNode.type === 'entity'
                    ? `${entity.businessName || entity.name} Fields`
                    : `${entity.subtypes?.[selectedNode.subtypeKey!]?.businessName || entity.subtypes?.[selectedNode.subtypeKey!]?.name} Fields`}
                </h3>
              </div>

              {inherited.length > 0 && (
                <div className={styles.inheritedSection}>
                  <div className={styles.inheritedHeader} onClick={() => setInheritedExpanded(!inheritedExpanded)}>
                    <span className={styles.inheritedToggle}>{inheritedExpanded ? '▼' : '▶'}</span>
                    <h4>🔒 Inherited Fields ({inherited.length})</h4>
                  </div>
                  {inheritedExpanded && (
                    <TableContainer component={Paper} className={styles.inheritedTable}>
                      <Table size="small">
                        <TableHead sx={{ backgroundColor: '#e0e0e0' }}>
                          <TableRow>
                            <TableCell>Display Name</TableCell>
                            <TableCell>Technical ID</TableCell>
                            <TableCell>Type</TableCell>
                            <TableCell>Semantic Link</TableCell>
                          </TableRow>
                        </TableHead>
                        <TableBody>
                          {inherited.map((field) => {
                            const fieldRules = getValidationRulesForField(field);
                            return (
                            <TableRow key={field.key}>
                              <TableCell>
                                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                                  {field.businessName}
                                  {fieldRules.length > 0 && (
                                    <Tooltip title={`${fieldRules.length} validation rule${fieldRules.length > 1 ? 's' : ''} assigned`}>
                                      <CheckCircle 
                                        size={16} 
                                        color="#059669"
                                        sx={{ cursor: 'pointer' }}
                                        onClick={() => {
                                          setSelectedFieldForRules({ field, rules: fieldRules });
                                          setShowValidationRulesModal(true);
                                        }}
                                      />
                                    </Tooltip>
                                  )}
                                </Box>
                              </TableCell>
                              <TableCell>{field.technicalName}</TableCell>
                              <TableCell>{field.type}</TableCell>
                              <TableCell>{field.semanticTermName}</TableCell>
                            </TableRow>
                            );
                          })}
                        </TableBody>
                      </Table>
                    </TableContainer>
                  )}
                </div>
              )}

              <div className={styles.assignedSection}>
                <div className={styles.assignedHeader}>
                  <h4>✏️ Assigned Fields ({assigned.length})</h4>
                  <Button
                    variant="contained"
                    size="small"
                    startIcon={<Plus />}
                    onClick={() => handleAddField(selectedNode.subtypeKey)}
                  >
                    Add
                  </Button>
                </div>
                <TableContainer component={Paper} className={styles.assignedTable}>
                  <Table size="small">
                    <TableHead sx={{ backgroundColor: '#e0e0e0' }}>
                      <TableRow>
                        <TableCell>Display Name</TableCell>
                        <TableCell>Technical ID</TableCell>
                        <TableCell>Type</TableCell>
                        <TableCell>Semantic Link</TableCell>
                        <TableCell align="right">Actions</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {assigned.map((field, index) => {
                        const fieldRules = getValidationRulesForField(field);
                        return (
                        <TableRow key={field.key}>
                          <TableCell>
                            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                              {field.businessName}
                              {fieldRules.length > 0 && (
                                <Tooltip title={`${fieldRules.length} validation rule${fieldRules.length > 1 ? 's' : ''} assigned`}>
                                  <CheckCircle 
                                    size={16} 
                                    color="#059669"
                                    sx={{ cursor: 'pointer' }}
                                    onClick={() => {
                                      setSelectedFieldForRules({ field, rules: fieldRules });
                                      setShowValidationRulesModal(true);
                                    }}
                                  />
                                </Tooltip>
                              )}
                            </Box>
                          </TableCell>
                          <TableCell>{field.technicalName}</TableCell>
                          <TableCell>{field.type}</TableCell>
                          <TableCell>{field.semanticTermName}</TableCell>
                          <TableCell align="right">
                            <Box sx={{ display: 'flex', gap: 1 }}>
                              <Tooltip title="Move up">
                                <span>
                                  <Button
                                    size="small"
                                    disabled={index === 0}
                                    onClick={() => handleMoveField(index, 'up')}
                                  >
                                    <ChevronUp />
                                  </Button>
                                </span>
                              </Tooltip>
                              <Tooltip title="Move down">
                                <span>
                                  <Button
                                    size="small"
                                    disabled={index === assigned.length - 1}
                                    onClick={() => handleMoveField(index, 'down')}
                                  >
                                    <ChevronDown />
                                  </Button>
                                </span>
                              </Tooltip>
                              <Tooltip title="Delete field">
                                <Button
                                  size="small"
                                  color="error"
                                  onClick={() => handleDeleteField(field.key)}
                                >
                                  <Trash2 />
                                </Button>
                              </Tooltip>
                            </Box>
                          </TableCell>
                        </TableRow>
                        );
                      })}
                    </TableBody>
                  </Table>
                </TableContainer>
              </div>
            </>
          ) : (
            <div className={styles.emptyState}>
              Select an entity or subtype from the tree
            </div>
          )}
        </Box>
      </Box>

      {/* Add Fields Modal */}
      <Dialog
        open={showFieldModal}
        onClose={() => {
          setShowFieldModal(false);
          setSelectedTermIds([]);
          setSemanticSearchTerm('');
        }}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle>
          Add Fields to {selectedNode.type === 'entity' ? editingEntity.businessName || editingEntity.name : editingEntity.subtypes?.[selectedNode.subtypeKey!]?.businessName || editingEntity.subtypes?.[selectedNode.subtypeKey!]?.name}
        </DialogTitle>
        <DialogContent>
          <div className={styles.modalSearchContainer}>
            <TextField
              placeholder="Search semantic terms..."
              value={semanticSearchTerm}
              onChange={(e) => setSemanticSearchTerm(e.target.value)}
              InputProps={{
                startAdornment: (
                  <InputAdornment position="start">
                    <Search size={16} />
                  </InputAdornment>
                ),
              }}
              fullWidth
            />
          </div>
          <div className={styles.modalSelectedCount}>
            {selectedTermIds.length > 0 && (
              <div>Selected: {selectedTermIds.length} term{selectedTermIds.length > 1 ? 's' : ''}</div>
            )}
          </div>
          <div className={styles.modalScrollableList}>
            {searchSemanticTerms(semanticTerms, semanticSearchTerm).map((term) => {
              const isSelected = selectedTermIds.includes(term.id);
              return (
                <div
                  key={term.id}
                  className={`${styles.semanticTermItemModal} ${isSelected ? styles.selected : ''}`}
                  onClick={() => {
                    setSelectedTermIds(prev =>
                      isSelected
                        ? prev.filter(id => id !== term.id)
                        : [...prev, term.id]
                    );
                  }}
                >
                  <div>
                    <div className={styles.semanticTermNameModal}>
                      {term.node_name}
                    </div>
                    <div className={styles.semanticTermDetailsModal}>
                      {term.technicalName} | {term.dataType}
                    </div>
                  </div>
                  <div>
                    {isSelected ? (
                      <CheckCircle className={styles.iconCheckCircle} />
                    ) : (
                      <PlusCircle className={styles.iconPlusCircle} />
                    )}
                  </div>
                </div>
              );
            })}
            {searchSemanticTerms(semanticTerms, semanticSearchTerm).length === 0 && (
              <div className={styles.emptyStateModal}>
                No semantic terms found
              </div>
            )}
          </div>
        </DialogContent>
        <DialogActions>
          <Button
            onClick={() => {
              setShowFieldModal(false);
              setSelectedTermIds([]);
              setSemanticSearchTerm('');
            }}
          >
            Cancel
          </Button>
          <Button
            onClick={handleAddMultipleFields}
            disabled={selectedTermIds.length === 0}
            variant="contained"
          >
            Add Selected ({selectedTermIds.length})
          </Button>
        </DialogActions>
      </Dialog>
      <Dialog
        open={addSubtypeModalVisible}
        onClose={() => {
          setAddSubtypeModalVisible(false);
          setAddSubtypeFormData({ name: '', description: '' });
          setAddSubtypeFormErrors({ name: '', description: '' });
        }}
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle>Add New Subtype</DialogTitle>
        <DialogContent>
          <Box sx={{ pt: 2 }}>
            <TextField
              fullWidth
              label="Subtype Name"
              value={addSubtypeFormData.name}
              onChange={(e) => {
                setAddSubtypeFormData(prev => ({ ...prev, name: e.target.value }));
                if (addSubtypeFormErrors.name) {
                  setAddSubtypeFormErrors(prev => ({ ...prev, name: '' }));
                }
              }}
              error={!!addSubtypeFormErrors.name}
              helperText={addSubtypeFormErrors.name}
              placeholder="Enter subtype name (e.g., Individual Client, Corporate Client)"
              sx={{ mb: 2 }}
            />
            <TextField
              fullWidth
              label="Description (Optional)"
              value={addSubtypeFormData.description}
              onChange={(e) => {
                setAddSubtypeFormData(prev => ({ ...prev, description: e.target.value }));
              }}
              multiline
              rows={3}
              placeholder="Describe what this subtype represents"
            />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button 
            onClick={() => {
              setAddSubtypeModalVisible(false);
              setAddSubtypeFormData({ name: '', description: '' });
              setAddSubtypeFormErrors({ name: '', description: '' });
            }}
          >
            Cancel
          </Button>
          <Button 
            onClick={() => {
              // Validate form
              const errors = { name: '', description: '' };
              if (!addSubtypeFormData.name.trim()) {
                errors.name = 'Please enter a subtype name';
              } else if (addSubtypeFormData.name.trim().length < 2) {
                errors.name = 'Subtype name must be at least 2 characters';
              }
              
              if (errors.name) {
                setAddSubtypeFormErrors(errors);
                return;
              }

              const { businessName, technicalName } = normalizeName(addSubtypeFormData.name, undefined);
              const subtypeKey = technicalName;
              const newSubtype = {
                name: businessName,
                businessName,
                technicalName,
                subtype_fields: [],
                isCore: false,
              } as any;

              const updated = JSON.parse(JSON.stringify(editingEntity));
              updated.subtypes = { ...(updated.subtypes || {}), [subtypeKey]: newSubtype };
              setEditingEntity(updated);
              // select the newly created subtype in the editor
              setSelectedNode({ type: 'subtype', subtypeKey });
              setAddSubtypeModalVisible(false);
              setAddSubtypeFormData({ name: '', description: '' });
              setAddSubtypeFormErrors({ name: '', description: '' });
              enqueueSnackbar(`Subtype "${businessName}" added`, { variant: 'success' });
            }}
            variant="contained"
          >
            Add Subtype
          </Button>
        </DialogActions>
      </Dialog>

      {/* Validation Rules Modal */}
      <Dialog 
        open={showValidationRulesModal} 
        onClose={() => setShowValidationRulesModal(false)}
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle>
          Validation Rules for "{selectedFieldForRules?.field?.businessName}"
        </DialogTitle>
        <DialogContent>
          {selectedFieldForRules?.rules && selectedFieldForRules.rules.length > 0 ? (
            <Box sx={{ mt: 2 }}>
              <TableContainer component={Paper} variant="outlined">
                <Table size="small">
                  <TableHead>
                    <TableRow sx={{ backgroundColor: '#f5f5f5' }}>
                      <TableCell><strong>Rule Name</strong></TableCell>
                      <TableCell><strong>Type</strong></TableCell>
                      <TableCell><strong>Severity</strong></TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {selectedFieldForRules.rules.map((rule: any) => (
                      <TableRow key={rule.id}>
                        <TableCell>{rule.rule_name}</TableCell>
                        <TableCell>{rule.rule_type || 'Standard'}</TableCell>
                        <TableCell>
                          <Chip 
                            label={rule.severity || 'Info'} 
                            size="small"
                            color={
                              rule.severity === 'Error' ? 'error' :
                              rule.severity === 'Warning' ? 'warning' :
                              'default'
                            }
                          />
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
              {selectedFieldForRules.rules[0]?.description && (
                <Box sx={{ mt: 2 }}>
                  <strong>Description:</strong>
                  <Box sx={{ mt: 1, p: 1, backgroundColor: '#f9f9f9', borderRadius: 1, fontSize: '0.875rem' }}>
                    {selectedFieldForRules.rules[0].description}
                  </Box>
                </Box>
              )}
            </Box>
          ) : (
            <Box sx={{ mt: 2, textAlign: 'center', color: '#999' }}>
              No validation rules found for this field.
            </Box>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setShowValidationRulesModal(false)}>Close</Button>
        </DialogActions>
      </Dialog>
    </div>
  );
}
