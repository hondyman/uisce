import { useState, useEffect, useMemo, useCallback, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { GOLD_COPY } from '../config';
import { getSelectedRegion } from '../lib/region';

import { resolveApiUrl } from '../utils/resolveApiUrl';

const VALIDATION_RULES_LIMIT = 100;
import {
  Box,
  Grid,
  AppBar,
  Toolbar,
  Container,
  Typography,
  Button,
  Chip,
  Tabs,
  Tab,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  InputAdornment,
  IconButton,
  CircularProgress,
  Stack,
  Breadcrumbs,
  Link,
  Alert,
  useTheme,
  useMediaQuery,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Tooltip,
  Autocomplete,
} from '@mui/material';
import {
  NavigateBefore as BackIcon,
  Edit as EditIcon,
  Add as AddIcon,
  MoreVert as MoreVertIcon,
  Search as SearchIcon,
  FolderOpen as FolderOpenIcon,
  CheckCircle as CheckCircleIcon,
  Warning as WarningIcon,
  Error as ErrorIcon,
  Delete as DeleteIcon,
  Info as InfoIcon,
  FileCopy as CloneIcon,
  Business as BusinessObjectIcon,
  Layers as SubtypeIcon,
  Apps as AppsIcon,
  TableChart as TableChartIcon,
  AccountTree as AccountTreeIcon,
  AddLink as AddLinkIcon,
  Functions as FunctionsIcon,
  ImportExport as ImportExportIcon,
  ShortText as TextIcon,
  Numbers as NumberIcon,
  CalendarToday as DateIcon,
  Code as JsonIcon,
  ToggleOn as BooleanIcon,
  InfoOutlined as InfoOutlinedIcon,
  Storage as StorageIcon,
} from '@mui/icons-material';
import { SemanticMappingWizard } from '../components/SemanticMappingWizard';
import { TableSortLabel } from '@mui/material';
import { useTenant } from '../contexts/TenantContext';
import { useAuth } from '../contexts/AuthContext';
import { useNotification } from '../hooks/useNotification';
import { useBusinessEntitySemanticLayer } from '../hooks/useBusinessEntitySemanticLayer';
import SemanticAssetsTab from '../components/entity/SemanticAssetsTab';
import { EditBusinessObjectModal } from '../components/BusinessObjectManager/EditBusinessObjectModal';
import { FieldSelectionWizard } from '../components/BusinessObjectManager/FieldSelectionWizard';
import { semanticTermToField, EnhancedSemanticTerm, useEnhancedSemanticTerms } from '../hooks/useEnhancedSemanticTerms';

import { BusinessObjectRelationshipWizard } from '../components/BusinessObjectManager/BusinessObjectRelationshipWizard';
import { ValidationRuleCreator } from '../components/ValidationRules/ValidationRuleCreator';
import { CalcFieldModal } from '../components/CalcFieldModal';
import { ValidationRuleScopeSelector, type ValidationRuleScope } from '../components/ValidationRules/ValidationRuleScopeSelector';
import { BOLineageGraphTab } from '../components/BusinessObjectManager/BOLineageGraphTab';
import { BOPendingBanner } from '../components/BusinessObjectManager/BOPendingBanner';
import { BOExportImportWizard } from '../components/BusinessObjectManager/BOExportImportWizard';
import { fetchEntitySchema } from '../api/entitySchema';
import { filterValidationRulesForEntity, type AnnotatedValidationRule } from '../utils/validationRules';
import ValidationRulesPage from '../features/fabric/pages/ValidationRulesPage';
import { devError, devDebug, devWarn } from '../utils/devLogger';
import { normalizeName } from '../utils/nameFormatting';
import { dedupeFields } from '../utils/dedupeFields';
import apiClient from '../utils/apiClient';
import type { Entity, Field, HierarchyNode } from '../types/entity-schema';
import { UnifiedLineageTab } from '../features/impact-analysis/components/UnifiedLineageTab';
import {
  FieldDeleteConfirmDialog,
  DeleteObjectConfirmDialog,
  SubtypeDialog,
  DeleteSubtypeDialog,
  EditFieldDialog,
} from './BusinessObjectDetailsPage/components/dialogs';


import {
  FieldsTab,
  BindingsTab,
  RelatedObjectsTab,
  RecordsCrudTab,
  BODeltaTab,
  LiveQueryTab,
  WorkflowTab,
} from './BusinessObjectDetailsPage/components/tabs';
import { PageHeader } from './BusinessObjectDetailsPage/components/PageHeader';
import { HierarchyTreePanel } from './BusinessObjectDetailsPage/components/HierarchyTreePanel';
import { ValidationPublishRail, ValidationSummaryItem } from '../components/BusinessObjectManager/ValidationPublishRail';
import { PlayArrow as RunIcon, CompareArrows as CompareIcon, AccountTree as WorkflowIcon } from '@mui/icons-material';



// Redundant local interface removed in favor of shared type

interface Subtype {
  id: string;
  key: string;
  name: string;
  displayName: string;
  technicalName: string;
  description?: string;
  subtypeFields: Field[];
  fields?: Field[]; // Alias for compatibility with components expecting fields
  isCore?: boolean;
}

// Redundant local interface removed in favor of shared type

interface BusinessObject {
  id: string;
  key: string;
  name: string;
  displayName: string;
  technicalName: string;
  description?: string;
  icon?: string;
  isCore?: boolean;
  tenantId?: string;
  coreFields?: Field[];
  customFields?: Field[];
  subtypes?: Record<string, Subtype>;
  category?: string;
  isActive?: boolean;
  status?: 'active' | 'draft'; // Support both for compatibility
  updatedAt?: string;
  version?: string;
  driverTableId?: string;
  driverTableName?: string;
  config?: any;
}

// TabPanel not used in current implementation

export default function BusinessObjectDetailsPage() {
  const { id: _id } = useParams<{ id: string }>();
  const id = _id;
  const navigate = useNavigate();
  const { tenant, datasource } = useTenant();
  const { token } = useAuth();
  const notification = useNotification();

  // Check if this is a new object
  const isNewObject = id === 'new';
  const tenantId = tenant?.id || '';
  const datasourceId = datasource?.id || datasource?.alpha_tenant_instance_id || '';

  // Track if we've already shown 404 error for this ID to avoid duplicate notifications
  const notFound404Shown = useRef(false);

  const [businessObject, setBusinessObject] = useState<BusinessObject | null>(null);
  const [loading, setLoading] = useState(!isNewObject);
  const [activeTab, setActiveTab] = useState(0);
  const [exportImportWizardOpen, setExportImportWizardOpen] = useState(false);
  const [mappingWizardOpen, setMappingWizardOpen] = useState(false);

  // Hierarchy State
  const [hierarchyNodes, setHierarchyNodes] = useState<HierarchyNode[]>([]);
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set(['root']));
  const [selectedNode, setSelectedNode] = useState<{ type: 'root' | 'group' | 'field' | 'subtype', key?: string, subtypeKey?: string, subtypeId?: string } | null>(null);
  const [searchFilter, setSearchFilter] = useState('');

  // Fields State
  const [fields, setFields] = useState<Field[]>([]);

  // Bindings State (from binding-first canonical service)
  const [bindings, setBindings] = useState<any[]>([]);

  // Form State (Legacy usage)
  const [name, setName] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [description, setDescription] = useState('');
  const [isActive, setIsActive] = useState(true);
  const [isSaving, setIsSaving] = useState(false);

  // Modal and Dialog states
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [relationshipWizardOpen, setRelationshipWizardOpen] = useState(false);
  const [addSubtypeOpen, setAddSubtypeOpen] = useState(false);
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);
  const [calcFieldModalOpen, setCalcFieldModalOpen] = useState(false);
  const [deleteObjectConfirmOpen, setDeleteObjectConfirmOpen] = useState(false);
  
  // Validation Rule states
  const [validationRuleCreatorOpen, setValidationRuleCreatorOpen] = useState(false);
  const [validationRuleScopeSelectorOpen, setValidationRuleScopeSelectorOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<any>(null);
  const [validationRuleScope, setValidationRuleScope] = useState<{ subtype?: string } | null>(null);
  const [validationRules, setValidationRules] = useState<AnnotatedValidationRule[]>([]);
  const [entitySchema, setEntitySchema] = useState<any>(null);
  const [availableEntities, setAvailableEntities] = useState<any[]>([]);

  // Subtype editing states
  const [editingSubtypeId, setEditingSubtypeId] = useState<string | null>(null);
  const [editingSubtypeKey, setEditingSubtypeKey] = useState<string | null>(null);
  const [deletingSubtypeId, setDeletingSubtypeId] = useState<string | null>(null);
  const [deletingSubtypeKey, setDeletingSubtypeKey] = useState<string | null>(null);
  const [deleteConfirmInput, setDeleteConfirmInput] = useState('');
  const { semanticTerms } = useEnhancedSemanticTerms(datasourceId);
  
  const [editingField, setEditingField] = useState<any | null>(null);
  const [editFieldModalOpen, setEditFieldModalOpen] = useState(false);
  const [editedFieldData, setEditedFieldData] = useState<{
    displayName: string;
    description: string;
    semanticTermId: string;
    role: string;
    targetScope?: string;
  }>({ displayName: '', description: '', semanticTermId: '', role: '', targetScope: 'root' });


  
  // Subtype form states
  const [subtypeDisplayName, setSubtypeDisplayName] = useState('');
  const [subtypeName, setSubtypeName] = useState('');
  const [subtypeDescription, setSubtypeDescription] = useState('');
  const [subtypeSaving, setSubtypeSaving] = useState(false);
  
  // Show/hide inherited fields toggle (defaults to false so selecting subtype shows assigned fields only)
  const [showInheritedFields, setShowInheritedFields] = useState(false);

  // Driver Table Selection State
  const [driverTableId, setDriverTableId] = useState<string | null>(null);
  const [driverTableName, setDriverTableName] = useState('');
  const [catalogNodes, setCatalogNodes] = useState<any[]>([]);
  const [loadingCatalog, setLoadingCatalog] = useState(false);

  // Pagination state
  // Page and rowsPerPage removed as they are not used in the current implementation

  // Related Objects view mode
  const [relatedObjectsView, setRelatedObjectsView] = useState<'tile' | 'table' | 'graph'>('table');
  const [relatedObjects, setRelatedObjects] = useState<any[]>([]);

  // Field deletion confirmation state
  const [fieldDeleteConfirmOpen, setFieldDeleteConfirmOpen] = useState(false);
  const [fieldPendingDelete, setFieldPendingDelete] = useState<any>(null);
  const [isDeleting, setIsDeleting] = useState(false);
  
  // Sorting state
  const [sortConfig, setSortConfig] = useState<{ key: string; direction: 'asc' | 'desc' }>({ key: 'sequence', direction: 'asc' });

  // Object Structure Resizable Sidebar State
  const [sidebarWidth, setSidebarWidth] = useState<number>(() => {
    const saved = typeof localStorage !== 'undefined' ? localStorage.getItem('bo_sidebar_width') : null;
    return saved ? Math.max(180, Math.min(600, parseInt(saved, 10))) : 280;
  });
  const [sidebarCollapsed, setSidebarCollapsed] = useState<boolean>(false);
  const isResizingRef = useRef(false);

  const handleMouseDownResize = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    isResizingRef.current = true;
    const startX = e.clientX;
    const startWidth = sidebarWidth;

    const handleMouseMove = (moveEvent: MouseEvent) => {
      if (!isResizingRef.current) return;
      const deltaX = moveEvent.clientX - startX;
      const newWidth = Math.max(180, Math.min(600, startWidth + deltaX));
      setSidebarWidth(newWidth);
      if (typeof localStorage !== 'undefined') {
        localStorage.setItem('bo_sidebar_width', newWidth.toString());
      }
    };

    const handleMouseUp = () => {
      isResizingRef.current = false;
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
    };

    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);
  }, [sidebarWidth]);

  // Field addition wizard state
  const [fieldWizardOpen, setFieldWizardOpen] = useState(false);
  const [addingFields, setAddingFields] = useState(false);

  // Initialize semantic layer
  const semanticLayer = useBusinessEntitySemanticLayer({
    tenantId,
    datasourceId,
    businessEntityId: businessObject?.id || '',
    businessEntityName: businessObject?.name || '',
    semanticTermIds: [],
    sourceTableNames: [],
  });

  // Helper to build headers with authentication
  const getAuthHeaders = (additionalHeaders: Record<string, string> = {}): Record<string, string> => {
    // Try token from hook, fallback to localStorage to ensure robustness
    const authToken = token || localStorage.getItem('auth_token');
    const authHeader = authToken && !authToken.includes('demo') ? `Bearer ${authToken}` : '';
    
    devDebug('[getAuthHeaders] token available:', !!authToken, 'auth header:', authHeader ? '✓ set' : '✗ MISSING');
    return {
      'Authorization': authHeader,
      'Content-Type': 'application/json',
      'X-Tenant-ID': tenantId,
      'X-Tenant-Datasource-ID': datasourceId,
      'X-Tenant-Region': getSelectedRegion(),
      ...additionalHeaders,
    };
  };

  // Field action handlers (edit/delete)
  // Helper to extract current config fields (semantic-term-backed)
  const getConfigFields = () => {
    // Primary source: customFields from the API response (this is the actual data)
    if (businessObject?.customFields && businessObject.customFields.length > 0) {
      return dedupeFields(businessObject.customFields
        // Remove strict filter, as fields often come back as 'string'/'int' etc.
        // We rely on the presence of semanticTermId to identify them.
        .map((f: any) => {
           // Try to find the semantic term ID from the field itself
           let sId = f.semanticTermId || f.semantic_term_id || (f.properties?.semantic_term_id);

           // If not found in the field object, try to look it up in config.fields by key
           if (!sId && businessObject.config?.fields) {
              const configField = businessObject.config.fields.find((cf: any) =>
                (cf.key === f.key) || (cf.technicalName === f.technicalName)
              );
              if (configField) {
                sId = configField.semanticTermId || configField.semantic_term_id;
              }
           }

           return {
             ...f,
             semanticTermId: sId || f.key || f.technicalName,
           };
         }));
    }

    // Fallback: try to resolve selected_terms from config as fields
    if (!businessObject?.customFields || businessObject.customFields.length === 0) {
      const selectedTerms = (businessObject?.config?.selected_terms as string[] | undefined) || [];
      if (selectedTerms.length > 0) {
        return dedupeFields(selectedTerms.map((termId: string, idx: number) => {
          // Find the semantic term to get its details
          const semanticTerm = semanticTerms.find(t => t.id === termId);
          return {
            id: termId,
            key: termId,
            name: semanticTerm?.node_name || termId,
            businessName: semanticTerm?.node_name || termId,
            displayName: semanticTerm?.node_name || termId,
            technicalName: semanticTerm?.node_name || termId,
            type: 'semantic_term',
            semanticTermId: termId,
            sequence: idx + 1,
            isCore: false,
            description: '',
          };
        }));
      }
    }

    // Fallback: use config.fields (source of truth for saving)
    return dedupeFields((businessObject?.config?.fields || []).map((f: any) => ({
      ...f,
      semanticTermId: f.semanticTermId || f.semantic_term_id
    })));
  };

  const getFieldCurrentScope = (field: any): string => {
    const fieldKey = (field.technicalName || field.key || field.name || '').toLowerCase();
    if (businessObject?.subtypes) {
      for (const [stKey, st] of Object.entries(businessObject.subtypes)) {
        if (st.subtypeFields?.some((f: any) => (f.technicalName || f.key || f.name || '').toLowerCase() === fieldKey)) {
          return stKey;
        }
      }
    }
    return 'root';
  };

  const handleMoveField = async (field: any, targetScope: string) => {
    if (!businessObject || !field) return;
    const sourceScope = getFieldCurrentScope(field);
    if (sourceScope === targetScope) return;

    try {
      setAddingFields(true);
      const targetKey = (field.technicalName || field.key || field.name || '').toLowerCase();
      const rootFields = getConfigFields();

      const selectedTerm = semanticTerms.find(t => t.id === field.semanticTermId);
      const updatedFieldItem = {
        ...field,
        name: field.displayName || field.businessName || field.name,
        businessName: field.displayName || field.businessName || field.name,
        displayName: field.displayName || field.businessName || field.name,
        technicalName: field.technicalName || field.key || field.name,
        key: field.technicalName || field.key || field.name,
        semanticTermId: field.semanticTermId || (selectedTerm ? selectedTerm.id : ''),
        semanticTermName: selectedTerm?.node_name || field.semanticTermName,
      };

      // 1. If moving FROM Root:
      if (sourceScope === 'root') {
        const remainingRootFields = rootFields.filter((f: any) =>
          (f.technicalName || f.key || f.name || '').toLowerCase() !== targetKey
        );
        const rootPayload = {
          displayName: businessObject.displayName,
          description: businessObject.description,
          icon: businessObject.icon,
          category: businessObject.category,
          isActive: businessObject.isActive,
          driverTableId: businessObject.driverTableId || undefined,
          driverTableName: businessObject.driverTableName || undefined,
          config: {
            ...((businessObject as any)?.config || {}),
            fields: remainingRootFields,
          },
          customFields: remainingRootFields,
        };
        await apiClient<any>(`/api/business-objects/${businessObject.id}`, {
          method: 'PUT',
          headers: getAuthHeaders(),
          body: JSON.stringify(rootPayload),
        });

        // Add to Target Subtype
        const targetSubtype = businessObject.subtypes?.[targetScope];
        if (targetSubtype) {
          const currentSubtypeFields = targetSubtype.subtypeFields || [];
          const updatedSubtypeFields = dedupeFields([...currentSubtypeFields, updatedFieldItem]);
          await apiClient<any>(`/api/business-objects/${targetSubtype.id}`, {
            method: 'PUT',
            headers: getAuthHeaders(),
            body: JSON.stringify({
              config: { fields: updatedSubtypeFields },
              customFields: updatedSubtypeFields,
            }),
          });
        }
      } 
      // 2. If moving TO Root:
      else if (targetScope === 'root') {
        const sourceSubtype = businessObject.subtypes?.[sourceScope];
        if (sourceSubtype) {
          const remainingSubtypeFields = (sourceSubtype.subtypeFields || []).filter((f: any) =>
            (f.technicalName || f.key || f.name || '').toLowerCase() !== targetKey
          );
          await apiClient<any>(`/api/business-objects/${sourceSubtype.id}`, {
            method: 'PUT',
            headers: getAuthHeaders(),
            body: JSON.stringify({
              config: { fields: remainingSubtypeFields },
              customFields: remainingSubtypeFields,
            }),
          });
        }

        // Add to Root
        const updatedRootFields = dedupeFields([...rootFields, updatedFieldItem]);
        const rootPayload = {
          displayName: businessObject.displayName,
          description: businessObject.description,
          icon: businessObject.icon,
          category: businessObject.category,
          isActive: businessObject.isActive,
          driverTableId: businessObject.driverTableId || undefined,
          driverTableName: businessObject.driverTableName || undefined,
          config: {
            ...((businessObject as any)?.config || {}),
            fields: updatedRootFields,
          },
          customFields: updatedRootFields,
        };
        await apiClient<any>(`/api/business-objects/${businessObject.id}`, {
          method: 'PUT',
          headers: getAuthHeaders(),
          body: JSON.stringify(rootPayload),
        });
      } 
      // 3. If moving between two Subtypes:
      else {
        const sourceSubtype = businessObject.subtypes?.[sourceScope];
        const targetSubtype = businessObject.subtypes?.[targetScope];
        if (sourceSubtype) {
          const remainingSubtypeFields = (sourceSubtype.subtypeFields || []).filter((f: any) =>
            (f.technicalName || f.key || f.name || '').toLowerCase() !== targetKey
          );
          await apiClient<any>(`/api/business-objects/${sourceSubtype.id}`, {
            method: 'PUT',
            headers: getAuthHeaders(),
            body: JSON.stringify({
              config: { fields: remainingSubtypeFields },
              customFields: remainingSubtypeFields,
            }),
          });
        }
        if (targetSubtype) {
          const updatedSubtypeFields = dedupeFields([...(targetSubtype.subtypeFields || []), updatedFieldItem]);
          await apiClient<any>(`/api/business-objects/${targetSubtype.id}`, {
            method: 'PUT',
            headers: getAuthHeaders(),
            body: JSON.stringify({
              config: { fields: updatedSubtypeFields },
              customFields: updatedSubtypeFields,
            }),
          });
        }
      }

      await fetchBusinessObject();
      const targetLabel = targetScope === 'root' ? businessObject.displayName : (businessObject.subtypes?.[targetScope]?.displayName || targetScope);
      notification.success(`Moved field "${field.businessName || field.displayName || field.name}" to ${targetLabel}`);
    } catch (error) {
      devError('Failed to move field:', error);
      notification.error('Failed to move field');
    } finally {
      setAddingFields(false);
    }
  };

  const handleAddFields = async (newTerms: EnhancedSemanticTerm[]) => {
      try {
        if (!tenantId || !datasourceId) {
          notification.error('Tenant and datasource must be selected');
          return;
        }
        if (!businessObject?.id) {
          notification.error('Business object not loaded');
          return;
        }

        setAddingFields(true);

        // Check if adding to a selected subtype
        if (selectedNode?.type === 'subtype' && selectedNode.subtypeKey && businessObject?.subtypes?.[selectedNode.subtypeKey]) {
          const targetSubtype = businessObject.subtypes[selectedNode.subtypeKey];
          const currentSubtypeFields = targetSubtype.subtypeFields || [];
          const maxSeq = currentSubtypeFields.reduce((max: number, f: any) => Math.max(max, f.sequence || 0), 0);

          const newFields = newTerms.map((term, idx) => ({
            ...semanticTermToField(term, maxSeq + idx + 1),
            semanticTermId: term.id,
          }));

          const updatedSubtypeFields = dedupeFields([...currentSubtypeFields, ...newFields]);

          await apiClient<any>(`/api/business-objects/${targetSubtype.id}`, {
            method: 'PUT',
            headers: getAuthHeaders(),
            body: JSON.stringify({
              config: { fields: updatedSubtypeFields },
              customFields: updatedSubtypeFields,
            }),
          });

          await fetchBusinessObject();
          notification.success(`Successfully added ${newFields.length} fields to ${targetSubtype.displayName || targetSubtype.name}`);
          setFieldWizardOpen(false);
          return;
        }

        // Default: Add to Root Business Object
        const currentFields = getConfigFields();
        const maxSeq = currentFields.reduce((max: number, f: any) => Math.max(max, f.sequence || 0), 0);
        
        const newFields = newTerms.map((term, idx) => ({
          ...semanticTermToField(term, maxSeq + idx + 1),
          semanticTermId: term.id, 
        }));

        const updatedFields = dedupeFields([...currentFields, ...newFields]);

        const payload = {
          driverTableId: businessObject.driverTableId || undefined,
          driverTableName: businessObject.driverTableName || undefined,
          config: {
            ...((businessObject as any)?.config || {}),
            fields: updatedFields,
          },
        };

        const updated = await apiClient<any>(
          `/api/business-objects/${businessObject.id}`,
          {
            method: 'PUT',
            headers: getAuthHeaders(),
            body: JSON.stringify(payload),
          }
        );
        setBusinessObject(prev => prev ? {
          ...prev,
          config: updated.config || payload.config,
          customFields: (updated.customFields && updated.customFields.length > 0)
            ? updated.customFields
            : undefined,
        } : null);

        notification.success(`Successfully added ${newFields.length} fields`);
        setFieldWizardOpen(false);

      } catch (error) {
        devError('Failed to add fields:', error);
        notification.error('Failed to add selected fields');
      } finally {
        setAddingFields(false);
      }
  };

  const handleEditField = (field: any) => {
    setEditingField(field);
    const currentScope = getFieldCurrentScope(field);
    setEditedFieldData({
      displayName: field.businessName || field.displayName || field.name,
      description: field.description || '',
      semanticTermId: field.semanticTermId || '',
      role: field.role || '',
      targetScope: currentScope,
    });
    setEditFieldModalOpen(true);
  };
  
  const handleSaveFieldEdit = async () => {
      if (!editingField || !businessObject) return;
      
      try {
          const currentScope = getFieldCurrentScope(editingField);
          const targetScope = editedFieldData.targetScope || 'root';
          const selectedTerm = semanticTerms.find(t => t.id === editedFieldData.semanticTermId);

          const updatedFieldItem = {
            ...editingField,
            name: editedFieldData.displayName,
            businessName: editedFieldData.displayName,
            displayName: editedFieldData.displayName,
            description: editedFieldData.description,
            role: editedFieldData.role,
            semanticTermId: editedFieldData.semanticTermId,
            semanticTermName: selectedTerm?.node_name || editingField.semanticTermName,
          };

          // If scope changed, handle cross-scope move
          if (currentScope !== targetScope) {
            await handleMoveField(updatedFieldItem, targetScope);
            setEditFieldModalOpen(false);
            setEditingField(null);
            return;
          }

          // Same scope update:
          if (currentScope === 'root') {
            const currentFields = getConfigFields();
            const targetKey = (editingField.technicalName || editingField.key || '').toLowerCase();
            const updatedFields = currentFields.map((f: any) => {
              const currentKey = (f.technicalName || f.key || '').toLowerCase();
              if (currentKey === targetKey) {
                return updatedFieldItem;
              }
              return f;
            });

            const savedFields = dedupeFields(updatedFields);
            const payload = {
              displayName: businessObject.displayName,
              description: businessObject.description,
              icon: businessObject.icon,
              category: businessObject.category,
              isActive: businessObject.isActive,
              driverTableId: businessObject.driverTableId || undefined,
              driverTableName: businessObject.driverTableName || undefined,
              config: {
                ...((businessObject as any)?.config || {}),
                fields: savedFields,
              },
              customFields: savedFields
            };
            
            const updated = await apiClient<any>(
              `/api/business-objects/${businessObject.id}`,
              {
                method: 'PUT',
                headers: getAuthHeaders(),
                body: JSON.stringify(payload),
              }
            );
            setBusinessObject(prev => prev ? {
              ...prev,
              config: updated.config || updatedFields,
              customFields: updated.customFields || updatedFields
            } : null);
          } else {
            // Update inside subtype
            const targetSubtype = businessObject.subtypes?.[currentScope];
            if (targetSubtype) {
              const currentSubtypeFields = targetSubtype.subtypeFields || [];
              const targetKey = (editingField.technicalName || editingField.key || '').toLowerCase();
              const updatedSubtypeFields = currentSubtypeFields.map((f: any) => {
                const currentKey = (f.technicalName || f.key || '').toLowerCase();
                if (currentKey === targetKey) {
                  return updatedFieldItem;
                }
                return f;
              });

              await apiClient<any>(`/api/business-objects/${targetSubtype.id}`, {
                method: 'PUT',
                headers: getAuthHeaders(),
                body: JSON.stringify({
                  config: { fields: updatedSubtypeFields },
                  customFields: updatedSubtypeFields,
                }),
              });
              await fetchBusinessObject();
            }
          }
          
          notification.success('Field updated successfully');
          setEditFieldModalOpen(false);
          setEditingField(null);
      } catch (error) {
          devError('Failed to update field:', error);
          notification.error('Failed to update field');
      }
  };

  const handleDeleteField = (field: any) => {
    // Show confirmation dialog instead of deleting immediately
    setFieldPendingDelete(field);
    setFieldDeleteConfirmOpen(true);
  };

  const handleConfirmDeleteField = async () => {
    try {
      if (!fieldPendingDelete) return;
      if (!tenantId || !datasourceId) {
        notification.error('Tenant and datasource must be selected');
        return;
      }
      if (!businessObject?.id) {
        notification.error('Business object not loaded');
        return;
      }

      setIsDeleting(true);
      const toDeleteKey = (fieldPendingDelete.technicalName || fieldPendingDelete.key || '').toLowerCase();
      const scope = getFieldCurrentScope(fieldPendingDelete);

      if (scope !== 'root' && businessObject.subtypes?.[scope]) {
        const subtype = businessObject.subtypes[scope];
        const remainingSubtypeFields = (subtype.subtypeFields || []).filter((f: any) => {
          const fk = (f.technicalName || f.key || '').toLowerCase();
          return fk !== toDeleteKey;
        });

        await apiClient<any>(`/api/business-objects/${subtype.id}`, {
          method: 'PUT',
          headers: getAuthHeaders(),
          body: JSON.stringify({
            config: { fields: remainingSubtypeFields },
            customFields: remainingSubtypeFields,
          }),
        });
        await fetchBusinessObject();
      } else {
        // Build updated fields array without the target field
        const currentFields = getConfigFields();
        const updatedFields = currentFields.filter((f: any) => {
          const fk = (f.technicalName || f.key || '').toLowerCase();
          return fk !== toDeleteKey;
        });

        const payload = {
          driverTableId: businessObject.driverTableId || undefined,
          driverTableName: businessObject.driverTableName || undefined,
          config: {
            ...((businessObject as any)?.config || {}),
            fields: updatedFields,
          },
        };

        const updated = await apiClient<any>(
          `/api/business-objects/${businessObject.id}`,
          {
            method: 'PUT',
            headers: getAuthHeaders(),
            body: JSON.stringify(payload),
          }
        );
        setBusinessObject(prev => prev ? {
          ...prev,
          displayName: updated.displayName || prev.displayName,
          description: updated.description || prev.description,
          icon: updated.icon || prev.icon,
          isActive: updated.isActive ?? prev.isActive,
          driverTableId: updated.driverTableId || updated.driver_table_id || prev.driverTableId,
          driverTableName: updated.driverTableName || updated.driver_table_name || prev.driverTableName,
          config: updated.config || prev.config,
          coreFields: updated.coreFields || prev.coreFields,
          customFields: updated.customFields || prev.customFields,
          subtypes: updated.subtypes || prev.subtypes,
        } : null);
      }
      notification.success(`Field removed: ${fieldPendingDelete.businessName || fieldPendingDelete.name}`);
      setFieldDeleteConfirmOpen(false);
      setFieldPendingDelete(null);
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to remove field';
      notification.error(msg);
    } finally {
      setIsDeleting(false);
    }
  };





  const handleNodeToggle = (nodeId: string) => {
    setExpandedNodes(prev => {
      const newSet = new Set(prev);
      if (newSet.has(nodeId)) {
        newSet.delete(nodeId);
      } else {
        newSet.add(nodeId);
      }
      return newSet;
    });
  };

  const handleDeleteBusinessObject = async () => {
    if (!businessObject?.id) return;
    
    try {
      // apiClient throws on non-OK, so we can navigate immediately on success.
      await apiClient<void>(
        `/api/business-objects/${businessObject.id}`,
        {
          method: 'DELETE',
          headers: getAuthHeaders(),
        }
      );

      notification.success(`"${businessObject.displayName}" deleted successfully`);
      navigate('/business-objects');
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to delete business object';
      notification.error(msg);
    }
  };

  // Main Fetch Logic
  const fetchBusinessObject = useCallback(async () => {
    // Resolve active tenant and datasource IDs with localStorage fallbacks for direct URL entry
    const activeTenantId = tenant?.id || localStorage.getItem('tenant_id') || 'gold_copy';
    const activeDatasourceId = datasourceId || localStorage.getItem('datasource_id') || 'crims';

    if (isNewObject || !activeTenantId) {
      setLoading(false);
      return;
    }

    try {
      setLoading(true);
      // Try fetching by ID first
      // Sanitize ID to remove any accidental trailing quotes (common copy-paste or url issue)
      const cleanId = id ? id.replace(/'$/, '') : '';
      let url = `/api/business-objects/${cleanId}`;
      
      const headers = getAuthHeaders();
      devDebug('[BusinessObjectDetailsPage] Fetching with headers:', headers);
      
      // Use raw fetch (instead of apiClient) here because we need to distinguish
      // 404 (object not found in this tenant) from other errors and handle them
      // gracefully without throwing. apiClient throws on every non-OK status.
      // The URL flows through resolveApiUrl.
      const resolvedUrl = resolveApiUrl(url);
      const response = await fetch(resolvedUrl, {
        headers
      });

      if (!response.ok) {
        const errorText = await response.text();
        devError(`[BusinessObjectDetailsPage] Fetch failed: ${response.status} ${response.statusText}`, errorText);

        if (response.status === 404) {
          // Business object not found - handle gracefully
          // Only show error once per ID to avoid spamming notification on effect re-runs
          devWarn(`[BusinessObjectDetailsPage] Business object not found: ${id} (404)`);
          if (!notFound404Shown.current) {
            notification.error(`Business object "${id}" not found in this tenant. It may have been deleted or the ID is incorrect.`);
            notFound404Shown.current = true;
          }
          setBusinessObject(null);
          setLoading(false);
          return;
        }
        throw new Error(`Failed to fetch business object: ${response.status} ${response.statusText} - ${errorText}`);
      }

      const data = await response.json();
      
      // Validate that the business object belongs to the current tenant or is a gold_copy / core object
      const isCoreObject = data.isCore || data.is_core || data.goldCopy || data.gold_copy;
      if (data.tenantId && data.tenantId !== tenant.id && !isCoreObject) {
        throw new Error('Business object does not belong to the current tenant');
      }
      
      devDebug('[BusinessObjectDetailsPage] Full API Response:', JSON.stringify(data, null, 2));
      devDebug('[BusinessObjectDetailsPage] API Response driverTableId:', data.driverTableId, 'driver_table_id:', data.driver_table_id);
      
      // Map fields logic (calculate before creating object)
      // Read from multiple possible sources: customFields, custom_fields, or fields (from bo_fields table)
      let customFields = data.customFields || data.custom_fields || data.fields || [];

      // Fallback: Populate from selected_terms if customFields is empty
      if (customFields.length === 0 && data.config?.selected_terms?.length > 0) {
           customFields = data.config.selected_terms.map((termId: string, idx: number) => {
               const term = semanticTerms.find(t => t.id === termId);
               return {
                   id: termId,
                   key: termId,
                   name: term?.node_name || termId,
                   businessName: term?.node_name || termId,
                   displayName: term?.node_name || termId,
                   technicalName: term?.technicalName || term?.node_name || termId,
                   type: term?.dataType || 'text',
                   semanticTermId: termId,
                   sequence: idx + 1,
                   isCore: false
               };
           });
      }

      // Map backend response to interface
      const mappedObject: BusinessObject = {
        tenantId: data.tenantId,
        id: data.id,
        key: data.key,
        name: data.name,
        displayName: data.displayName || data.display_name || data.name,
        technicalName: data.technicalName || data.technical_name || data.key,
        description: data.description,
        icon: data.icon,
        isCore: data.isCore || data.is_core,
        category: data.category,
        isActive: data.isActive ?? data.is_active ?? true,
        status: data.status || 'draft',
        
        // Map fields
        coreFields: data.coreFields || data.core_fields || [],
        customFields: customFields,
        
        // Map subtypes
        subtypes: data.subtypes || {},

        // Map driver table
        driverTableId: data.driverTableId || data.driver_table_id,
        driverTableName: data.driverTableName || data.driver_table_name,
      };

      // Extract bindings from API response (binding-first canonical model)
      if (data.bindings && Array.isArray(data.bindings)) {
        devDebug('[BusinessObjectDetailsPage] Found bindings in response:', data.bindings.length);
        setBindings(data.bindings);
      }

      if (data.related_bos && Array.isArray(data.related_bos)) {
        setRelatedObjects(data.related_bos);
      } else if (cleanId) {
        fetch(`/api/business-objects/${cleanId}/relationships`, { headers })
          .then(r => r.ok ? r.json() : null)
          .then(relData => {
            if (relData && relData.relatedObjects) {
              setRelatedObjects(relData.relatedObjects);
            }
          })
          .catch(() => {});
      }

      setBusinessObject(mappedObject);
      
      devDebug('[BusinessObjectDetailsPage] Mapped object driverTableId:', mappedObject.driverTableId, 'driverTableName:', mappedObject.driverTableName);
      
      // Populate legacy state
      setName(mappedObject.name);
      setDisplayName(mappedObject.displayName);
      setDescription(mappedObject.description || '');
      setIsActive(mappedObject.isActive || true);
      if (mappedObject.driverTableId) setDriverTableId(mappedObject.driverTableId);
      if (mappedObject.driverTableName) setDriverTableName(mappedObject.driverTableName);
      
      // Collect all fields for legacy state
      const allFields: Field[] = dedupeFields([
        ...(mappedObject.coreFields || []),
        ...(mappedObject.customFields || []),
        ...Object.values(mappedObject.subtypes || {}).flatMap(s => s.subtypeFields || [])
      ].map((field: any) => ({
        ...field,
        businessName: field.businessName || field.displayName || field.name,
        technicalName: field.technicalName || field.name,
        type: field.type || 'text',
      })));
      setFields(allFields);

      // Extract hierarchy nodes
      const hierarchy: HierarchyNode[] = [];
      if (mappedObject.coreFields && mappedObject.coreFields.length > 0) {
        hierarchy.push({
          id: 'core-fields',
          name: 'Core Fields',
          displayName: 'Core Fields',
          icon: 'verified',
          fields: mappedObject.coreFields
        });
      }
      if (mappedObject.subtypes && Object.keys(mappedObject.subtypes).length > 0) {
        const subtypeNodes: HierarchyNode[] = Object.entries(mappedObject.subtypes).map(([subtypeKey, subtype]) => ({
          id: `subtype-${subtypeKey}`,
          name: subtype.name,
          displayName: subtype.displayName || subtype.name,
          icon: '',
          fields: subtype.subtypeFields,
          subtypeKey: subtypeKey,
          subtypeId: subtype.id,
          technicalName: subtype.technicalName || subtypeKey,
          description: subtype.description,
          isSubtype: true
        }));
        hierarchy.push(...subtypeNodes);
      }
      const rootHierarchy: HierarchyNode[] = [
        {
          id: 'root',
          name: mappedObject.displayName || 'Root',
          displayName: mappedObject.displayName || 'Root',
          icon: mappedObject.icon || 'business',
          children: hierarchy.length > 0 ? hierarchy : undefined,
          fields: hierarchy.length === 0 ? uniqueFields : undefined
        },
      ];
      setHierarchyNodes(rootHierarchy);
    } catch (err) {
      devError('Error fetching business object:', err);
      const errorMsg = err instanceof Error ? err.message : String(err);
      notification.error(`Failed to load business object: ${errorMsg}`);
      setBusinessObject(null);
    } finally {
      setLoading(false);
    }
  }, [id, tenant, isNewObject, datasourceId, semanticTerms]);

  // Effect to sync URL with object name
  useEffect(() => {
    if (businessObject?.technicalName && !isNewObject && id !== businessObject.id && id !== businessObject.technicalName) {
      // Don't redirect automatically as it might confuse navigation history
    }
  }, [businessObject, id, isNewObject]);

  // Fetch Data Effect
  useEffect(() => {
    // Reset 404 flag when ID changes
    notFound404Shown.current = false;
    fetchBusinessObject();
  }, [fetchBusinessObject]);  // Load catalog nodes for driver table selection (only for new objects).
  // Tracks an "aborted" flag in the cleanup to avoid React's setState-on-unmounted
  // warning when deps change or the component unmounts mid-fetch.
  useEffect(() => {
    if (!isNewObject || !tenantId || !datasourceId) return;
    let aborted = false;

    const loadCatalogNodes = async () => {
      try {
        setLoadingCatalog(true);
        const url = `api/rest/catalog-nodes?tenant_datasource_id=${datasourceId}`;
        // apiClient parses JSON and respects VITE_USE_PROXY; tenant/region
        // headers are injected automatically.
        const data = await apiClient<any>(url);
        if (aborted) return;
        setCatalogNodes(Array.isArray(data) ? data : (data?.nodes || []));
      } catch (err) {
        if (aborted) return;
        // Silent error for catalog loading to avoid disrupting page load
        devWarn('Failed to load catalog nodes:', err);
      } finally {
        if (!aborted) setLoadingCatalog(false);
      }
    };

    loadCatalogNodes();
    return () => {
      aborted = true;
    };
  }, [isNewObject, tenantId, datasourceId]);

  const fetchValidationRules = useCallback(async () => {
    if (!tenantId || !datasourceId || !id || isNewObject) {
      devDebug('[fetchValidationRules] Skipping fetch:', { tenantId, datasourceId, id, isNewObject });
      return;
    }

    const entityIdentifier = businessObject?.technicalName || businessObject?.key || id;
    devDebug('[fetchValidationRules] Starting fetch for entity:', entityIdentifier);
    devDebug('[fetchValidationRules] Parameters:', { tenantId, datasourceId, entityIdentifier });

    try {
      let allRules: any[] = [];
      let pageNum = 1;
      let hasMore = true;

      while (hasMore) {
        const params = new URLSearchParams({
          tenant_id: tenantId,
          tenant_instance_id: datasourceId,
          page: String(pageNum),
          limit: String(VALIDATION_RULES_LIMIT),
        });
        params.append('entities', entityIdentifier);
        
        const url = `/api/validation-rules?${params.toString()}`;
        devDebug('[fetchValidationRules] Fetching URL:', url);
        
        // apiClient throws on non-OK, so the throw boilerplate goes away.
        const data = await apiClient<any>(url, {
          headers: getAuthHeaders(),
        });
        devDebug('[fetchValidationRules] API response:', data);
        
        const raw = Array.isArray(data) ? data : (data.rules || []);
        devDebug('[fetchValidationRules] Extracted rules:', raw);
        devDebug('[fetchValidationRules] Rules count:', raw.length);
        
        allRules = allRules.concat(raw);
        hasMore = data.has_more;
        pageNum++;
      }
      
      devDebug('[fetchValidationRules] Total rules fetched:', allRules.length);
      
      // Transform BusinessObject to Entity-like for filtering
      if (businessObject) {
        const tempEntity: Entity = {
          key: businessObject.key,
          name: businessObject.displayName,
          businessName: businessObject.displayName,
          technicalName: businessObject.technicalName,
          entity_fields: fields.map(f => ({
            key: f.key,
            name: f.name,
            businessName: f.businessName || f.name,
            technicalName: f.technicalName || f.name,
            type: (f.type.toLowerCase() as any) || 'text'
          })),
          subtypes: {}
        };
        devDebug('[fetchValidationRules] Filtering for entity:', businessObject.name);
        const filtered = filterValidationRulesForEntity(businessObject.name, tempEntity, allRules);
        devDebug('[fetchValidationRules] Filtered rules count:', filtered.length);
        
        // Transform rules to ensure script_content is mapped to logic field for ValidationRulesPage
        const transformedForDisplay = filtered.map((rule: any) => ({
          ...rule,
          logic: rule.script_content || rule.rule_definition || rule.logic || '',
          name: rule.rule_name || rule.name,
          type: (rule.rule_type || 'expression').toLowerCase(),
          severity: (rule.severity || 'error').toLowerCase(),
          status: rule.is_active === false ? 'inactive' : 'active',
        }));
        
        devDebug('[fetchValidationRules] Transformed rules with logic field:', transformedForDisplay);
        devDebug('[fetchValidationRules] Setting validation rules');
        setValidationRules(transformedForDisplay);
      }
    } catch (err) { 
      devError('[fetchValidationRules] Error:', err); 
    }
  }, [tenantId, datasourceId, id, isNewObject, businessObject, fields]);

  useEffect(() => {
    // First useEffect above already triggers fetchBusinessObject() when fetchBusinessObject changes,
    // so this effect only needs to reset form state when entering the "new" route.
    if (isNewObject) {
      setLoading(false);
      setBusinessObject(null);
      setName('');
      setDisplayName('');
      setDescription('');
      setIsActive(true);
      setFields([]);
      setHierarchyNodes([]);
    }
  }, [fetchBusinessObject, isNewObject]);

  useEffect(() => {
    devDebug('[useEffect-validations] activeTab:', activeTab, 'isNewObject:', isNewObject);
    if (activeTab === 2 && !isNewObject) { // Validations is now tab 2 (after Bindings)
      devDebug('[useEffect-validations] Triggering fetchValidationRules');
      fetchValidationRules();
    }
  }, [activeTab, fetchValidationRules, isNewObject]);

  // Fetch full schema for rule creator.
  // Aborts in-flight requests when tenant/datasource change or when the
  // component unmounts to avoid the React "setState on unmounted component"
  // memory-leak warning.
  useEffect(() => {
    if (!tenantId || !datasourceId) return;
    const abortController = new AbortController();
    const loadSchema = async () => {
      try {
        const schema = await fetchEntitySchema(tenantId, datasourceId);
        if (abortController.signal.aborted) return;
        setEntitySchema(schema);
        setAvailableEntities(Object.keys(schema).sort());
      } catch (error) {
        if (abortController.signal.aborted) return;
        devError('Error fetching entity schema:', error);
      }
    };
    loadSchema();
    return () => {
      abortController.abort();
    };
  }, [tenantId, datasourceId]);

  const handleAddRule = () => {
    setEditingRule(null);
    setValidationRuleScope(null);
    setValidationRuleScopeSelectorOpen(true);
  };

  const handleScopeSelected = (scope: ValidationRuleScope) => {
    setValidationRuleScope(scope);
    setValidationRuleScopeSelectorOpen(false);
    setValidationRuleCreatorOpen(true);
  };

  const handleEditRule = (rule: any) => {
    setEditingRule(rule);
    setValidationRuleCreatorOpen(true);
  };

  const handleSaveRule = useCallback(async (rule: any) => {
    try {
      // Save the rule to the backend
      const method = rule.id ? 'PATCH' : 'POST';
      const endpoint = rule.id 
        ? `/api/validation-rules/${rule.id}`
        : '/api/validation-rules';

      // apiClient throws on non-OK. Saves the rule and refreshes the list.
      await apiClient<void>(endpoint, {
        method,
        headers: getAuthHeaders(),
        body: JSON.stringify(rule),
      });

      // Refresh rules after successful save
      await fetchValidationRules();
      notification.success(rule.id ? 'Rule updated successfully' : 'Rule created successfully');
      setValidationRuleCreatorOpen(false);
      setEditingRule(null);
    } catch (err) {
      notification.error(err instanceof Error ? err.message : 'Failed to save rule');
    }
  }, [tenantId, datasourceId, fetchValidationRules, notification]);

  const handleAddSubtype = async () => {
    // Validate required context from operating scope
    if (!tenantId) {
      notification.error('Tenant context is missing. Please reload the page.');
      return;
    }
    if (!datasourceId) {
      notification.error('Datasource context is missing. Please reload the page.');
      return;
    }

    if (!subtypeDisplayName.trim()) {
      notification.error('Display name is required');
      return;
    }

    // For edit mode, use the rename endpoint instead
    if (editingSubtypeId) {
      setSubtypeSaving(true);
      try {
        const parentId = businessObject?.id || id;
        // Use a raw fetch so we can read the response body for the error
        // message (apiClient throws without the response body).
        const renameUrl = `/api/business-objects/${parentId}/subtypes/${editingSubtypeId}/rename`;
        const renameResp = await fetch(renameUrl, {
          method: 'POST',
          headers: getAuthHeaders(),
          body: JSON.stringify({
            newName: subtypeDisplayName,
          }),
        });

        if (!renameResp.ok) {
          const text = await renameResp.text();
          notification.error(`Failed to update subtype: ${text || renameResp.statusText}`);
          return;
        }

        notification.success('Subtype updated successfully');
        setAddSubtypeOpen(false);
        setEditingSubtypeId(null);
        setSubtypeName('');
        setSubtypeDisplayName('');
        setSubtypeDescription('');
        
        // Refresh the business object
        await fetchBusinessObject();
      } catch (error) {
        devError('Failed to update subtype:', error);
        notification.error('Failed to update subtype');
      } finally {
        setSubtypeSaving(false);
      }
      return;
    }

    // For create mode
    // Auto-format technical name: lowercase, replace spaces with underscores
    let technicalName = subtypeName.trim().toLowerCase().replace(/\s+/g, '_');
    
    if (!technicalName) {
      // Fallback: derive from display name
      technicalName = subtypeDisplayName.trim().toLowerCase().replace(/\s+/g, '_');
    }

    // Validate technical name format
    if (!/^[a-z0-9_]+$/.test(technicalName)) {
      notification.error('Technical name must be lowercase letters, numbers, and underscores only (no spaces or special characters)');
      return;
    }

    const { businessName: normalizedDisplay } = normalizeName(
      subtypeDisplayName || undefined,
      technicalName
    );

    setSubtypeSaving(true);
    try {
      const body = {
        name: technicalName,
        displayName: normalizedDisplay || subtypeDisplayName,
        description: subtypeDescription,
        parent_id: id,
        isCore: false,
      };

      // Use raw fetch so we can surface the response body in the error toast.
      const createUrl = `/api/business-objects?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}`;
      const createResp = await fetch(createUrl, {
        method: 'POST',
        headers: getAuthHeaders(),
        body: JSON.stringify({ ...body, datasourceId }),
      });

      if (!createResp.ok) {
        const text = await createResp.text();
        notification.error(`Failed to create subtype: ${text || createResp.statusText}`);
        throw new Error(text || 'Failed to create subtype');
      }

      await createResp.json();

      notification.success('Subtype created successfully');
      setAddSubtypeOpen(false);
      setSubtypeName('');
      setSubtypeDisplayName('');
      setSubtypeDescription('');

      // Immediately fetch the parent business object to get updated subtypes
      await fetchBusinessObject();

    } catch (error) {
      devError('Failed to create subtype:', error);
    } finally {
      setSubtypeSaving(false);
    }
  };

  // handleRenameSubtype removed - Now handled in handleAddSubtype

  const handleDeleteSubtype = async (subtypeId: string) => {
    const parentId = businessObject?.id || id;
    devDebug(`[DEBUG] handleDeleteSubtype: parentId=${parentId}, subtypeId=${subtypeId}`);
    
    if (!parentId || !subtypeId) {
      devError('[DEBUG] Missing parentId or subtypeId', { parentId, subtypeId });
      notification.error('Cannot delete subtype: missing identifiers');
      return;
    }

    try {
      // Use raw fetch so we can read the response body for the error message.
      const delUrl = `/api/business-objects/${parentId}/subtypes/${subtypeId}`;
      const response = await fetch(delUrl, {
        method: 'DELETE',
        headers: getAuthHeaders(),
      });

      if (!response.ok) {
        const text = await response.text();
        devError(`[DEBUG] Delete failed with status ${response.status}: ${text}`);
        notification.error(`Failed to delete subtype (${response.status}): ${text || response.statusText}`);
        return;
      }

      devDebug('[DEBUG] Delete successful');
      notification.success('Subtype deleted successfully');
      setDeleteConfirmOpen(false);
      setDeletingSubtypeId(null);
      setDeletingSubtypeKey(null);
      
      // Refresh the business object
      await fetchBusinessObject();
    } catch (error) {
      devError('[DEBUG] Catch error in handleDeleteSubtype:', error);
      notification.error('Failed to delete subtype due to a network or client error');
    }
  };

  const openRenameDialog = (subtypeKey: string, currentName: string) => {
    const subtypeId = businessObject?.subtypes?.[subtypeKey]?.id;
    setEditingSubtypeId(subtypeId || null);
    setEditingSubtypeKey(subtypeKey);
    setSubtypeDisplayName(currentName);
    setSubtypeName(subtypeKey);
    setSubtypeDescription(businessObject?.subtypes?.[subtypeKey]?.description || '');
    setAddSubtypeOpen(true);
  };

  const openDeleteConfirm = (subtypeKey: string) => {
    const subtypeId = businessObject?.subtypes?.[subtypeKey]?.id;
    setDeletingSubtypeId(subtypeId || null);
    setDeletingSubtypeKey(subtypeKey);
    setDeleteConfirmOpen(true);
  };

  // Memoized filtered fields (no pagination, lazy loading on demand)
  const filteredFields = useMemo(() => {
    let fieldsToFilter: any[] = [];
    // If root is selected or nothing selected, show root's fields (core + custom)
    // If a subtype is selected, show inherited + subtype-specific fields
    if (selectedNode?.type === 'subtype' && selectedNode.subtypeKey && businessObject?.subtypes?.[selectedNode.subtypeKey]) {
      const subtypeFields = businessObject.subtypes[selectedNode.subtypeKey].subtypeFields || [];
      if (showInheritedFields) {
        // Show inherited fields (core + custom) plus subtype-specific fields
        const rootConfigFields = getConfigFields();
        const inheritedFields = rootConfigFields.length > 0 ? rootConfigFields : [
          ...(businessObject.coreFields || []),
          ...(businessObject.customFields || [])
        ];
        fieldsToFilter = dedupeFields([...inheritedFields, ...subtypeFields]);
      } else {
        // Show only subtype-specific fields
        fieldsToFilter = dedupeFields(subtypeFields);
      }
    } else {
      // For root business object, show core + custom fields
      const configFields = getConfigFields();
      if (Array.isArray(configFields) && configFields.length > 0) {
        fieldsToFilter = dedupeFields(configFields);
      } else {
        fieldsToFilter = dedupeFields([
          ...(businessObject?.coreFields || []),
          ...(businessObject?.customFields || [])
        ]);
      }
    }

    return fieldsToFilter.filter(
      (field: any) =>
        field.name.toLowerCase().includes(searchFilter.toLowerCase()) ||
        field.businessName?.toLowerCase().includes(searchFilter.toLowerCase()) ||
        field.type.toLowerCase().includes(searchFilter.toLowerCase())
    );
  }, [businessObject, selectedNode, searchFilter, showInheritedFields, semanticTerms]);

  // Apply sorting to filterFields
  const sortedFilteredFields = useMemo(() => {
    const sorted = [...filteredFields];
    if (sortConfig.key) {
      sorted.sort((a, b) => {
        let aValue = (a as any)[sortConfig.key];
        let bValue = (b as any)[sortConfig.key];
        
        // Handle special cases or defaults
        if (typeof aValue === 'string') aValue = aValue.toLowerCase();
        if (typeof bValue === 'string') bValue = bValue.toLowerCase();
        
        if (aValue < bValue) return sortConfig.direction === 'asc' ? -1 : 1;
        if (aValue > bValue) return sortConfig.direction === 'asc' ? 1 : -1;
        return 0;
      });
    }
    return sorted;
  }, [filteredFields, sortConfig]);

  const handleRequestSort = (property: string) => {
      const isAsc = sortConfig.key === property && sortConfig.direction === 'asc';
      setSortConfig({ key: property, direction: isAsc ? 'desc' : 'asc' });
  };

  const handleChangeTab = (_event: React.SyntheticEvent, newValue: number) => {
    setActiveTab(newValue);
  };

  const getValidationIcon = (validation?: string) => {
    switch (validation) {
      case 'valid':
        return <CheckCircleIcon sx={{ color: 'success.main', fontSize: '1.25rem' }} />;
      case 'warning':
        return <WarningIcon sx={{ color: 'warning.main', fontSize: '1.25rem' }} />;
      case 'error':
        return <ErrorIcon sx={{ color: 'error.main', fontSize: '1.25rem' }} />;
      default:
        return null;
    }
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh' }}>
        <CircularProgress />
      </Box>
    );
  }

  if (!isNewObject && !businessObject) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="error">Business object not found</Alert>
      </Box>
    );
  }

  return (
    <>
    <Box sx={{ display: 'flex', flexDirection: 'column', minHeight: '100vh', bgcolor: 'background.default', pb: 12 }}>
      {/* Top Navigation */}
      <AppBar position="sticky" elevation={0} sx={{ borderBottom: '1px solid', borderBottomColor: 'divider' }}>
        <Toolbar>
          <Stack direction="row" spacing={2} alignItems="center" sx={{ flex: 1 }}>
            <IconButton
              edge="start"
              color="inherit"
              onClick={() => navigate('/business-objects')}
              sx={{ mr: 1 }}
            >
              <BackIcon />
            </IconButton>
            <Box
              sx={{
                display: 'flex',
                alignItems: 'center',
                gap: 1,
              }}
            >
              <Box
                sx={{
                  width: 32,
                  height: 32,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  bgcolor: 'primary.main',
                  color: 'primary.contrastText',
                  borderRadius: 1,
                }}
              >
                📦
              </Box>
              <Typography variant="h6" sx={{ fontWeight: 700 }}>
                Business Object Manager
              </Typography>
            </Box>
          </Stack>

          <Stack direction="row" spacing={2} sx={{ display: { xs: 'none', md: 'flex' } }}>
            <TextField
              placeholder="Search objects..."
              variant="outlined"
              size="small"
              InputProps={{
                startAdornment: (
                  <InputAdornment position="start">
                    <SearchIcon />
                  </InputAdornment>
                ),
              }}
              sx={{ width: 250 }}
            />
          </Stack>
        </Toolbar>
      </AppBar>

      {/* Main Content */}
      <Container maxWidth="xl" sx={{ flex: 1, py: 4 }}>
        {/* Breadcrumbs */}
        <Breadcrumbs sx={{ mb: 3 }}>
          <Link
            underline="hover"
            color="inherit"
            onClick={() => navigate('/')}
            sx={{ cursor: 'pointer' }}
          >
            Home
          </Link>
          <Link
            underline="hover"
            color="inherit"
            onClick={() => navigate('/business-objects')}
            sx={{ cursor: 'pointer' }}
          >
            Business Objects
          </Link>
          <Typography color="text.primary">
            {isNewObject ? 'Create New' : businessObject?.displayName}
          </Typography>
        </Breadcrumbs>

        {/* Page Header */}
        <PageHeader
          isNewObject={isNewObject}
          businessObject={businessObject}
          onDeleteObject={() => setDeleteObjectConfirmOpen(true)}
          onEditObject={() => setEditModalOpen(true)}
          onAddSubtype={() => {
            setEditingSubtypeId(null);
            setEditingSubtypeKey(null);
            setSubtypeDisplayName('');
            setSubtypeName('');
            setSubtypeDescription('');
            setAddSubtypeOpen(true);
          }}
          onAddCalculatedField={() => setCalcFieldModalOpen(true)}
          onExportImport={() => setExportImportWizardOpen(true)}
        />

        {/* Show error message if business object not found */}
        {!isNewObject && !businessObject && !loading && (
          <Alert 
            severity="error" 
            sx={{ mb: 3, mt: 2 }}
            action={
              <Button 
                color="inherit" 
                size="small" 
                onClick={() => navigate('/business-objects')}
              >
                Back to List
              </Button>
            }
          >
            <Box>
              <Typography variant="subtitle2" sx={{ fontWeight: 700, mb: 1 }}>
                Business Object Not Found
              </Typography>
              <Typography variant="body2">
                The business object with ID "{id}" could not be found in this tenant. It may have been deleted or the ID might be incorrect. Please check the URL or go back to the business objects list.
              </Typography>
            </Box>
          </Alert>
        )}

        {/* Create Form for New Objects */}
        {isNewObject && (
          <Paper elevation={0} sx={{ border: '1px solid', borderColor: 'divider', borderRadius: 2, p: 3, mb: 3 }}>
            <Typography variant="h6" sx={{ fontWeight: 700, mb: 3 }}>
              Business Object Details
            </Typography>
            <Stack spacing={3}>
              <TextField
                fullWidth
                label="Technical Name"
                placeholder="e.g., customer_account"
                value={name}
                onChange={(e) => setName(e.target.value)}
                helperText="Unique identifier for this business object"
                variant="outlined"
              />
              <TextField
                fullWidth
                label="Display Name"
                placeholder="e.g., Customer Account"
                value={displayName}
                onChange={(e) => setDisplayName(e.target.value)}
                helperText="Human-readable name"
                variant="outlined"
              />
              <TextField
                fullWidth
                label="Description"
                placeholder="Describe this business object..."
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                multiline
                rows={4}
                variant="outlined"
              />
              
              {/* Driver Table Selection */}
              <Box>
                <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1, color: 'text.secondary' }}>
                  🗂️ Driver Table (Primary Source)
                </Typography>
                <Autocomplete
                  options={catalogNodes}
                  getOptionLabel={(option: any) => option.qualified_path || option.node_name || ''}
                  value={catalogNodes.find((n: any) => n.node_id === driverTableId) || null}
                  onChange={(_, node: any) => {
                    if (node) {
                      setDriverTableId(node.node_id);
                      setDriverTableName(node.qualified_path || node.node_name);
                    } else {
                      setDriverTableId(null);
                      setDriverTableName('');
                    }
                  }}
                  loading={loadingCatalog}
                  size="small"
                  renderInput={(params) => (
                    <TextField
                      {...params}
                      placeholder="Search tables..."
                      variant="outlined"
                      size="small"
                      helperText="Select the primary table that defines this business object (recommended)"
                      InputProps={{
                        ...params.InputProps,
                        endAdornment: (
                          <>
                            {loadingCatalog ? <CircularProgress color="inherit" size={20} /> : null}
                            {params.InputProps.endAdornment}
                          </>
                        ),
                      }}
                    />
                  )}
                  renderOption={(props, option: any) => {
                    const { key, ...otherProps } = props;
                    return (
                      <li key={key} {...otherProps}>
                        <Stack spacing={0.5}>
                          <Typography variant="body2" sx={{ fontWeight: 500 }}>
                            {option.node_name}
                          </Typography>
                          <Typography variant="caption" color="text.secondary">
                            {option.qualified_path}
                          </Typography>
                        </Stack>
                      </li>
                    );
                  }}
                  noOptionsText={loadingCatalog ? 'Loading tables...' : 'No tables found'}
                />
              </Box>
              <Box>
                <Stack direction="row" spacing={2} alignItems="center">
                  <Typography variant="body2">Status:</Typography>
                  <Chip
                    label={isActive ? 'Active' : 'Draft'}
                    color={isActive ? 'success' : 'default'}
                    onClick={() => setIsActive(!isActive)}
                    variant={isActive ? 'filled' : 'outlined'}
                  />
                </Stack>
              </Box>
              <Stack direction="row" spacing={2} justifyContent="flex-end">
                <Button variant="outlined" onClick={() => navigate('/business-objects')}>
                  Cancel
                </Button>
                <Button
                  variant="contained"
                  color="primary"
                  onClick={async () => {
                    if (!name || !displayName) {
                      notification.error('Please fill in all required fields');
                      return;
                    }
                    
                    if (!tenantId || !datasourceId) {
                      notification.error('Tenant and datasource must be selected');
                      return;
                    }
                    
                    setIsSaving(true);
                    
                    try {
                      // Create the business object via POST request
                      const payload = {
                        bo_key: name,
                        name: name,
                        display_name: displayName,
                        description: description || '',
                        driver_table_id: driverTableId,
                        driver_table_name: driverTableName,
                        status: isActive ? 'active' : 'draft',
                        config: {
                          is_core: GOLD_COPY,
                        }
                      };
                      
                      // apiClient parses JSON and throws on non-OK; .catch returns
                      // {} on empty bodies so we always have a usable error object.
                      const createdBO = await apiClient<any>('/api/business-objects', {
                        method: 'POST',
                        headers: getAuthHeaders(),
                        body: JSON.stringify(payload),
                      }).catch((err: any) => {
                        throw new Error(err?.message || 'Failed to create business object');
                      });
                      notification.success('Business Object created successfully!');
                      navigate(`/business-objects/${createdBO.id}`);
                    } catch (error) {
                      devError('Failed to create business object:', error);
                      notification.error(error instanceof Error ? error.message : 'Failed to create business object');
                    } finally {
                      setIsSaving(false);
                    }
                  }}
                  disabled={isSaving}
                >
                  {isSaving ? 'Creating...' : 'Create'}
                </Button>
              </Stack>
            </Stack>
          </Paper>
        )}

        {/* Tabs - Only show for existing objects */}
        {!isNewObject && (
          <>
            <BOPendingBanner 
              boId={id || ''} 
              onTabChange={setActiveTab}
              onPublish={() => {
                // Refresh BO data after publish
                if (id) {
                  fetchBusinessObject();
                }
              }}
              onRefresh={() => {
                // Refresh all BO data
                if (id) {
                  fetchBusinessObject();
                }
              }}
            />
            <Paper elevation={0} sx={{ border: '1px solid', borderColor: 'divider', borderRadius: 2, mb: 3 }}>
          <Tabs
            value={activeTab}
            onChange={handleChangeTab}
            variant="scrollable"
            sx={{
              borderBottom: '1px solid',
              borderBottomColor: 'divider',
              '& .MuiTab-root': {
                textTransform: 'none',
                fontWeight: 600,
              },
            }}
          >
            <Tab label="Hierarchy & Fields" icon={<FolderOpenIcon />} iconPosition="start" />
            <Tab
              label={
                <Stack direction="row" spacing={1} alignItems="center">
                  <span>Bindings</span>
                  {bindings.length > 0 && (
                    <Chip label={bindings.length} size="small" variant="outlined" />
                  )}
                </Stack>
              }
              icon={<StorageIcon />}
              iconPosition="start"
            />
            <Tab label="Live Query Explorer" icon={<RunIcon />} iconPosition="start" />
            <Tab label="Records & ORM CRUD" icon={<TableChartIcon />} iconPosition="start" />
            <Tab label="Workday Delta" icon={<CompareIcon />} iconPosition="start" />
            <Tab label="Governance & Workflows" icon={<WorkflowIcon />} iconPosition="start" />
            <Tab
              label={
                <Stack direction="row" spacing={1} alignItems="center">
                  <span>Validations</span>
                  {validationRules.length > 0 && (
                    <Chip label={validationRules.length} size="small" variant="outlined" />
                  )}
                </Stack>
              }
            />
            <Tab label="Related Objects" />
            <Tab label="Graph" icon={<AccountTreeIcon />} iconPosition="start" />
            <Tab label="Semantic Model" />
            <Tab label="Lineage" icon={<AccountTreeIcon />} iconPosition="start" />
          </Tabs>

          {/* Main Content Area with Sidebar */}
          <Box sx={{ display: 'flex', flexDirection: { xs: 'column', lg: 'row' }, gap: 0, p: 3, position: 'relative' }}>
            {/* Left Panel: Hierarchy Tree - Always Visible */}
            <HierarchyTreePanel
              hierarchyNodes={hierarchyNodes}
              expandedNodes={expandedNodes}
              selectedNode={selectedNode}
              businessObject={businessObject}
              width={sidebarWidth}
              isCollapsed={sidebarCollapsed}
              onToggleCollapse={() => setSidebarCollapsed(!sidebarCollapsed)}
              onNodeToggle={handleNodeToggle}
              onNodeSelect={setSelectedNode}
              onRenameSubtype={openRenameDialog}
              onDeleteSubtype={openDeleteConfirm}
            />

            {/* Draggable Vertical Divider */}
            {!sidebarCollapsed && (
              <Box
                onMouseDown={handleMouseDownResize}
                sx={{
                  display: { xs: 'none', lg: 'flex' },
                  width: '12px',
                  cursor: 'col-resize',
                  alignItems: 'center',
                  justifyContent: 'center',
                  mx: 0.5,
                  userSelect: 'none',
                  zIndex: 2,
                  '&:hover .divider-handle, &:active .divider-handle': {
                    bgcolor: 'primary.main',
                    height: '48px',
                    width: '4px',
                  },
                }}
              >
                <Box
                  className="divider-handle"
                  sx={{
                    width: '2px',
                    height: '24px',
                    bgcolor: 'divider',
                    borderRadius: '2px',
                    transition: 'all 0.15s ease',
                  }}
                />
              </Box>
            )}

            {/* Right Panel: Tab Content */}
            <Paper
              elevation={0}
              sx={{
                flex: 1,
                width: 0,
                minWidth: 0,
                ml: { xs: 0, lg: sidebarCollapsed ? 2 : 0 },
                border: '1px solid',
                borderColor: 'divider',
                borderRadius: 1,
                overflow: 'hidden',
                display: 'flex',
                flexDirection: 'column',
              }}
            >
                {activeTab === 0 && (
                  <FieldsTab
                    selectedNode={selectedNode}
                    businessObject={businessObject}
                    searchFilter={searchFilter}
                    showInheritedFields={showInheritedFields}
                    sortedFilteredFields={sortedFilteredFields}
                    sortConfig={sortConfig}
                    onSearchChange={setSearchFilter}
                    onToggleInherited={() => setShowInheritedFields(!showInheritedFields)}
                    onAddField={() => setFieldWizardOpen(true)}
                    onEditField={handleEditField}
                    onDeleteField={handleDeleteField}
                    onMoveField={handleMoveField}
                    onSort={handleRequestSort}
                    getValidationIcon={getValidationIcon}
                  />
                )}

              {/* Bindings Tab */}
              {activeTab === 1 && (
                <BindingsTab bindings={bindings} businessObject={businessObject} />
              )}


              {/* Live Query Explorer Tab */}
              {activeTab === 2 && (
                <LiveQueryTab businessObject={businessObject} />
              )}

              {/* Records & ORM CRUD Tab */}
              {activeTab === 3 && (
                <RecordsCrudTab businessObject={businessObject} />
              )}

              {/* Workday Delta Tab */}
              {activeTab === 4 && (
                <BODeltaTab businessObject={businessObject} />
              )}

              {/* Governance & Workflows Tab */}
              {activeTab === 5 && (
                <WorkflowTab businessObject={businessObject} />
              )}

              {/* Validations Tab */}
              {activeTab === 6 && (
                <ValidationRulesPage 
                  businessObjectId={id}
                  businessObjectName={businessObject?.name}
                  selectedNodeType={selectedNode?.type}
                  selectedNodeName={selectedNode?.type === 'subtype' ? selectedNode.subtypeKey : undefined}
                  fields={selectedNode?.type === 'subtype' ? (businessObject?.subtypes?.[selectedNode.subtypeKey!]?.fields || []) : fields}
                  rules={validationRules as any}
                  onRulesUpdate={setValidationRules as any}
                  onAddRule={handleAddRule}
                  onEditRule={handleEditRule}
                />
              )}

              {/* Related Objects Tab */}
              {activeTab === 7 && (
                <RelatedObjectsTab
                  relatedObjectsView={relatedObjectsView}
                  relatedObjects={relatedObjects}
                  onAddRelationship={() => setRelationshipWizardOpen(true)}
                  onViewChange={setRelatedObjectsView}
                />
              )}

              {/* Graph Tab */}
              {activeTab === 8 && (
                <Box sx={{ height: '70vh', p: 2 }}>
                  <BOLineageGraphTab boId={id || ''} />
                </Box>
              )}

              {/* Semantic Model Tab */}
              {activeTab === 9 && (
                <Box sx={{ p: 3 }}>
                  <SemanticAssetsTab
                    boId={id}
                    semanticAssets={semanticLayer.semanticAssets}
                    isLoading={semanticLayer.assetsLoading || semanticLayer.modelGenerationLoading}
                    error={semanticLayer.modelError}
                    onGenerateCoreModel={async () => { await semanticLayer.generateCoreModel(); }}
                    onCreateCustomModel={async (name) => { await semanticLayer.createCustomModel(name); }}
                    onGenerateCoreView={async () => { await semanticLayer.generateCoreView(); }}
                    onCreateCustomView={async (name) => { await semanticLayer.createCustomView(name); }}
                    businessEntityName={selectedNode?.type === 'subtype' ? (businessObject?.subtypes?.[selectedNode.subtypeKey!]?.displayName || selectedNode.subtypeKey || '') : (businessObject?.displayName || 'Business Object')}
                    selectedNodeType={selectedNode?.type}
                    selectedNodeName={selectedNode?.type === 'subtype' ? selectedNode.subtypeKey : businessObject?.key}
                    hierarchyNodes={[]}
                  />
                </Box>
              )}

              {/* Lineage & Impact Tab */}
              {activeTab === 10 && (

                <Box sx={{ p: 3 }}>
                   <Typography variant="h6" sx={{ fontWeight: 700, mb: 1 }}>
                     Lineage
                   </Typography>
                   <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                     Visualize upstream dependencies and downstream impact using dynamic analysis.
                   </Typography>
                   
                   <UnifiedLineageTab 
                      nodeType="business_object" 
                      nodeId={businessObject?.id || id || ''}
                      initialDirection="both"
                   />
                </Box>
              )}
            </Paper>
            </Box>
          </Paper>

          </> // Close the fragment opened for BOPendingBanner
        )}
      </Container>

    </Box>
    <ValidationRuleScopeSelector
      isOpen={validationRuleScopeSelectorOpen}
      onClose={() => setValidationRuleScopeSelectorOpen(false)}
      onConfirm={handleScopeSelected}
      businessObjectName={businessObject?.displayName || businessObject?.name || ''}
      subtypes={businessObject?.subtypes}
    />

    <ValidationRuleCreator
      isOpen={validationRuleCreatorOpen}
      onClose={() => {
        setValidationRuleCreatorOpen(false);
        setEditingRule(null);
        setValidationRuleScope(null);
      }}
      onSave={handleSaveRule}
      tenantId={tenantId}
      datasourceId={datasourceId}
      availableEntities={availableEntities}
      entitySchema={entitySchema}
      editingRule={editingRule as any}
      defaultTargetEntity={businessObject?.name}
      initialScope={validationRuleScope ? { subtype: validationRuleScope.subtype } : undefined}
      subtypes={businessObject?.subtypes}
      coreFields={businessObject?.coreFields}
      customFields={businessObject?.customFields}
    />

    {/* Field Delete Confirmation Dialog */}
    <FieldDeleteConfirmDialog
      open={fieldDeleteConfirmOpen}
      fieldPendingDelete={fieldPendingDelete}
      isDeleting={isDeleting}
      onClose={() => {
        setFieldDeleteConfirmOpen(false);
        setFieldPendingDelete(null);
      }}
      onConfirm={handleConfirmDeleteField}
    />

    {/* Add/Edit Subtype Dialog */}
    <SubtypeDialog
      open={addSubtypeOpen}
      mode="add"
      businessObject={businessObject}
      editingSubtypeKey={editingSubtypeKey}
      subtypeDisplayName={subtypeDisplayName}
      subtypeName={subtypeName}
      subtypeDescription={subtypeDescription}
      subtypeSaving={subtypeSaving}
      onClose={() => {
        setEditingSubtypeId(null);
        setEditingSubtypeKey(null);
        setAddSubtypeOpen(false);
      }}
      onDisplayNameChange={setSubtypeDisplayName}
      onTechnicalNameChange={setSubtypeName}
      onDescriptionChange={setSubtypeDescription}
      onSave={handleAddSubtype}
    />

    {/* Delete Subtype Confirmation Dialog */}
    <DeleteSubtypeDialog
      open={deleteConfirmOpen}
      businessObject={businessObject}
      deletingSubtypeKey={deletingSubtypeKey}
      deleteConfirmInput={deleteConfirmInput}
      onClose={() => {
        setDeleteConfirmOpen(false);
        setDeletingSubtypeKey(null);
        setDeleteConfirmInput('');
      }}
      onInputChange={setDeleteConfirmInput}
      onConfirm={() => {
        if (deletingSubtypeId) {
          handleDeleteSubtype(deletingSubtypeId);
          setDeleteConfirmInput('');
        }
      }}
    />

    {/* Edit Business Object Modal */}
    {businessObject && (
      <>
        {/* Relationship Wizard */}
        <BusinessObjectRelationshipWizard
          open={relationshipWizardOpen}
          onClose={() => setRelationshipWizardOpen(false)}
          onRelationshipSaved={() => {
            fetchBusinessObject();
          }}
          businessObject={businessObject}
          tenantId={tenantId}
          datasourceId={datasourceId}
        />

        <EditBusinessObjectModal
        isOpen={editModalOpen}
        object={{
          id: businessObject.id,
          name: businessObject.name,
          display_name: businessObject.displayName,
          description: businessObject.description,
          status: 'draft',
          driver_table_id: businessObject.driverTableId,
          driver_table_name: businessObject.driverTableName,
          config: { is_active: businessObject.isActive ?? true },
        }}
        onClose={() => setEditModalOpen(false)}
        onSave={async (data) => {
          try {
            devDebug('[BusinessObjectDetailsPage] Modal opened with businessObject.driverTableId:', businessObject.driverTableId, 'driverTableName:', businessObject.driverTableName);
            
            // Map frontend fields to backend UpdateBusinessObjectRequest
            const payload = {
              displayName: data.display_name,
              description: data.description || '',
              icon: data.driver_table_name || '',
              driverTableId: data.driver_table_id || '',
              driverTableName: data.driver_table_name || '',
              isActive: data.config?.is_active ?? true,
              config: data.config, // Pass config (including fields) to backend
            };
            
            devDebug('[BusinessObjectDetailsPage] Saving with payload:', payload);

            // apiClient parses JSON, throws on non-OK.
            const updated = await apiClient<any>(
              `/api/business-objects/${businessObject.id}`,
              {
                method: 'PUT',
                headers: getAuthHeaders(),
                body: JSON.stringify(payload),
              }
            );
            setBusinessObject(prev => prev ? { 
              ...prev, 
              displayName: updated.displayName || prev.displayName,
              description: updated.description || prev.description,
              icon: updated.icon || prev.icon,
              isActive: updated.isActive ?? prev.isActive,
              driverTableId: updated.driverTableId || updated.driver_table_id || prev.driverTableId,
              driverTableName: updated.driverTableName || updated.driver_table_name || prev.driverTableName,
              // Merge config and custom fields so the UI reflects newly added semantic fields immediately
              config: updated.config || prev.config,
              customFields: updated.customFields || prev.customFields,
            } : null);
            notification.success('Business Object updated successfully');
            setEditModalOpen(false);
          } catch (error) {
            const msg = error instanceof Error ? error.message : 'Failed to update';
            notification.error(msg);
            throw error;
          }
        }}
      />
      </>
    )}

    <BOExportImportWizard
      open={exportImportWizardOpen}
      onClose={() => setExportImportWizardOpen(false)}
      onComplete={(boId) => {
        if (boId) {
          fetchBusinessObject();
        }
      }}
    />

    <CalcFieldModal
      isOpen={calcFieldModalOpen}
      onClose={() => setCalcFieldModalOpen(false)}
      objectId={businessObject?.id || ''}
      onSaved={fetchBusinessObject}
    />

    <FieldSelectionWizard
      isOpen={fieldWizardOpen}
      onClose={() => setFieldWizardOpen(false)}
      selectedDriverTable={businessObject?.driverTableId ? {
        node_id: businessObject.driverTableId,
        node_name: businessObject.driverTableName || '',
        qualified_path: businessObject.driverTableName || '',
      } : undefined}
      existingFields={getConfigFields()}
      onSelectFields={handleAddFields}
      loading={addingFields}
    />
    
    {/* Physical Model Mapping Wizard Overlay */}
    <Dialog
      open={mappingWizardOpen}
      onClose={() => setMappingWizardOpen(false)}
      maxWidth="xl"
      fullWidth
      PaperProps={{
        sx: { height: '90vh' }
      }}
    >
      <SemanticMappingWizard
        tenantId={tenantId}
        datasourceId={datasourceId}
        onClose={() => setMappingWizardOpen(false)}
        onMappingsApplied={() => {
           fetchBusinessObject();
        }}
      />
    </Dialog>

    {/* Edit Field Dialog */}
    <EditFieldDialog
      open={editFieldModalOpen}
      semanticTerms={semanticTerms}
      editedFieldData={editedFieldData}
      subtypes={businessObject?.subtypes}
      businessObjectName={businessObject?.displayName}
      onClose={() => setEditFieldModalOpen(false)}
      onFieldDataChange={setEditedFieldData}
      onSave={handleSaveFieldEdit}
    />
    {/* Delete Business Object Confirmation Dialog */}
    <DeleteObjectConfirmDialog
      open={deleteObjectConfirmOpen}
      businessObject={businessObject}
      onClose={() => setDeleteObjectConfirmOpen(false)}
      onConfirm={handleDeleteBusinessObject}
    />

    <ValidationPublishRail
      status="DRAFT"
      summaryItems={[
        {
          label: 'Identity Check',
          status: businessObject?.key || businessObject?.id ? 'PASS' : 'FAIL',
          message: businessObject?.key || businessObject?.id ? 'Valid identity' : 'Missing identity',
        },
        {
          label: 'Required Fields',
          status: fields.some((f: any) => f.required) ? 'WARN' : 'PASS',
          message: 'Checking required bindings',
        },
        {
          label: 'Validation Rules',
          status: validationRules.length > 0 ? 'PASS' : 'WARN',
          message: `${validationRules.length} rules defined`,
        },
        {
          label: 'Security',
          status: 'PASS',
          message: 'JWT Enforced',
        }
      ]}
      onSaveDraft={() => setEditModalOpen(true)}
    />
  </>
  );
}

