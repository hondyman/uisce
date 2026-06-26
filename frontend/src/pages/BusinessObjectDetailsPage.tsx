import { useState, useEffect, useMemo, useCallback, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { GOLD_COPY } from '../config';
import { getSelectedRegion } from '../lib/region';
import {
  Box,
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
} from '@mui/icons-material';
import { SemanticMappingWizard } from '../components/SemanticMappingWizard';
import { TableSortLabel } from '@mui/material';
// import { useAccess } from '../contexts/AccessContext';
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
import type { Entity, Field, HierarchyNode } from '../types/entity-schema';
import { UnifiedLineageTab } from '../features/impact-analysis/components/UnifiedLineageTab';

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
  // const { currentTenant: tenant, currentDatasource: datasource } = useAccess();
  const { token } = useAuth();
  const notification = useNotification();
  // const theme = useTheme();
  // const _isMobile = useMediaQuery(theme.breakpoints.down('md'));

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
  }>({ displayName: '', description: '', semanticTermId: '', role: '' });


  
  // Subtype form states
  const [subtypeDisplayName, setSubtypeDisplayName] = useState('');
  const [subtypeName, setSubtypeName] = useState('');
  const [subtypeDescription, setSubtypeDescription] = useState('');
  const [subtypeSaving, setSubtypeSaving] = useState(false);
  
  // Show/hide inherited fields toggle
  const [showInheritedFields, setShowInheritedFields] = useState(true);

  // Driver Table Selection State
  const [driverTableId, setDriverTableId] = useState<string | null>(null);
  const [driverTableName, setDriverTableName] = useState('');
  const [catalogNodes, setCatalogNodes] = useState<any[]>([]);
  const [loadingCatalog, setLoadingCatalog] = useState(false);

  // Pagination state
  // Page and rowsPerPage removed as they are not used in the current implementation

  // Related Objects view mode
  const [relatedObjectsView, setRelatedObjectsView] = useState<'tile' | 'table' | 'graph'>('table');

  // Field deletion confirmation state
  const [fieldDeleteConfirmOpen, setFieldDeleteConfirmOpen] = useState(false);
  const [fieldPendingDelete, setFieldPendingDelete] = useState<any>(null);
  const [isDeleting, setIsDeleting] = useState(false);
  
  // Sorting state
  const [sortConfig, setSortConfig] = useState<{ key: string; direction: 'asc' | 'desc' }>({ key: 'sequence', direction: 'asc' });

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
  // Helper to extract current config fields (semantic-term-backed)
  const getConfigFields = () => {
    // Primary source: customFields from the API response (this is the actual data)
    if (businessObject?.customFields && businessObject.customFields.length > 0) {
      return businessObject.customFields
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
        });
    }

    // Fallback: try to resolve selected_terms from config as fields
    if (!businessObject?.customFields || businessObject.customFields.length === 0) {
      const selectedTerms = (businessObject?.config?.selected_terms as string[] | undefined) || [];
      if (selectedTerms.length > 0) {
        return selectedTerms.map((termId: string, idx: number) => {
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
        });
      }
    }

    // Fallback: use config.fields (source of truth for saving)
    return (businessObject?.config?.fields || []).map((f: any) => ({
      ...f,
      semanticTermId: f.semanticTermId || f.semantic_term_id
    }));
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

        setAddingFields(true);

        // Use getConfigFields() as the base to ensure we include fields from bo_fields (source of truth)
        // instead of relying on potentially stale/empty config.fields
        const currentFields = getConfigFields();
        
        // Convert new terms to fields
        // We calculate max sequence to append at end
        const maxSeq = currentFields.reduce((max: number, f: any) => Math.max(max, f.sequence || 0), 0);
        
        const newFields = newTerms.map((term, idx) => ({
          ...semanticTermToField(term, maxSeq + idx + 1),
          // Ensure we store the semanticTermId which is crucial for mapping
          semanticTermId: term.id, 
        }));

        const updatedFields = [...currentFields, ...newFields];

        const payload = {
          // Preserve driver table context
          driverTableId: businessObject.driverTableId || undefined,
          driverTableName: businessObject.driverTableName || undefined,
          config: {
            ...((businessObject as any)?.config || {}),
            fields: updatedFields,
          },
        };

        const resp = await fetch(`/api/business-objects/${businessObject.id}`,
          {
            method: 'PUT',
            headers: getAuthHeaders(),
            body: JSON.stringify(payload),
          }
        );

        if (!resp.ok) {
          const text = await resp.text();
          throw new Error(text || 'Failed to update business object');
        }

        const updated = await resp.json();
        setBusinessObject(prev => prev ? {
          ...prev,
          // Use payload.config if backend response is missing it (optimistic/robust update)
          config: updated.config || payload.config,
          // If updated fields returned, use them. Otherwise unset customFields so getConfigFields uses config.fields fallback.
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
    setEditedFieldData({
      displayName: field.businessName || field.name,
      description: field.description || '',
      semanticTermId: field.semanticTermId || '',
      role: field.role || '',
    });
    setEditFieldModalOpen(true);
  };
  
  const handleSaveFieldEdit = async () => {
      if (!editingField || !businessObject) return;
      
      try {
          // Update the specific field in the list
          const currentFields = getConfigFields();
          const updatedFields = currentFields.map((f: any) => {
              const currentKey = (f.technicalName || f.key || '').toLowerCase();
              const targetKey = (editingField.technicalName || editingField.key || '').toLowerCase();
              
              if (currentKey === targetKey) {
                  // Find selected semantic term to enrich data
                  const selectedTerm = semanticTerms.find(t => t.id === editedFieldData.semanticTermId);
                  
                  return {
                      ...f,
                      name: editedFieldData.displayName,
                      businessName: editedFieldData.displayName,
                      description: editedFieldData.description,
                      role: editedFieldData.role,
                      semanticTermId: editedFieldData.semanticTermId,
                      semanticTermName: selectedTerm?.node_name,
                  };
              }
              return f;
          });

          // Send update to backend
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
                fields: updatedFields,
            },
            customFields: updatedFields
          };
          
          const response = await fetch(`/api/business-objects/${businessObject.id}`, {
              method: 'PUT',
              headers: getAuthHeaders(),
              body: JSON.stringify(payload),
          });
          
          if (!response.ok) throw new Error('Failed to update field');
          
          const updated = await response.json();
          setBusinessObject(prev => prev ? { 
               ...prev, 
               config: updated.config || updatedFields, 
               customFields: updated.customFields || updatedFields
          } : null);
          
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

      // Build updated fields array without the target field
      const currentFields = getConfigFields();
      // Match on key (technical name) - this is the unique identifier for the field
      const toDeleteKey = (fieldPendingDelete.technicalName || fieldPendingDelete.key || '').toLowerCase();
      
      const updatedFields = currentFields.filter((f: any) => {
        const fk = (f.technicalName || f.key || '').toLowerCase();
        // Only keep fields that don't match the key we're deleting
        return fk !== toDeleteKey;
      });

      const payload = {
        // Preserve driver table context to keep semantic edges consistent for remaining fields
        driverTableId: businessObject.driverTableId || undefined,
        driverTableName: businessObject.driverTableName || undefined,
        config: {
          ...((businessObject as any)?.config || {}),
          fields: updatedFields,
        },
      };

      const resp = await fetch(`/api/business-objects/${businessObject.id}`,
        {
          method: 'PUT',
          headers: getAuthHeaders(),
          body: JSON.stringify(payload),
        }
      );

      if (!resp.ok) {
        const text = await resp.text();
        throw new Error(text || 'Failed to update business object');
      }

      const updated = await resp.json();
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
      const response = await fetch(`/api/business-objects/${businessObject.id}`, {
        method: 'DELETE',
        headers: getAuthHeaders(),
      });

      if (!response.ok) {
        throw new Error('Failed to delete business object');
      }

      notification.success(`"${businessObject.displayName}" deleted successfully`);
      navigate('/business-objects');
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to delete business object';
      notification.error(msg);
    }
  };

  // Main Fetch Logic
  const fetchBusinessObject = useCallback(async () => {
    if (isNewObject || !tenant?.id || !datasourceId) {
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
      
      const response = await fetch(url, {
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
      
      // Validate that the business object belongs to the current tenant
      if (data.tenantId && data.tenantId !== tenant.id) {
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
      const allFields: Field[] = [
        ...(mappedObject.coreFields || []),
        ...(mappedObject.customFields || []),
        ...Object.values(mappedObject.subtypes || {}).flatMap(s => s.subtypeFields || [])
      ].map((field: any) => ({
        ...field,
        businessName: field.businessName || field.displayName || field.name,
        technicalName: field.technicalName || field.name,
        type: field.type || 'text',
      }));

      const uniqueFields = Array.from(new Map(allFields.map(f => [f.key, f])).values());
      setFields(uniqueFields);

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
  }, [fetchBusinessObject]);

  // Load catalog nodes for driver table selection (only for new objects)
  useEffect(() => {
    if (!isNewObject || !tenantId || !datasourceId) return;

    const loadCatalogNodes = async () => {
      try {
        setLoadingCatalog(true);
        const url = `/api/catalog/nodes?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}&type=table`;
        
        const response = await fetch(url, {
            headers: {
              'X-Tenant-ID': tenantId,
              'X-Tenant-Datasource-ID': datasourceId,
              'X-Tenant-Region': getSelectedRegion(),
            },
          }
        );

        if (!response.ok) {
          throw new Error('Failed to load catalog nodes');
        }

        const data = await response.json();
        setCatalogNodes(Array.isArray(data) ? data : (data?.nodes || []));
      } catch (err) {
        // Silent error for catalog loading to avoid disrupting page load
        devWarn('Failed to load catalog nodes:', err);
      } finally {
        setLoadingCatalog(false);
      }
    };

    loadCatalogNodes();
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
          limit: '100',
        });
        params.append('entities', entityIdentifier);
        
        const url = `/api/validation-rules?${params.toString()}`;
        devDebug('[fetchValidationRules] Fetching URL:', url);
        
        const res = await fetch(url, {
          headers: getAuthHeaders(),
        });
        
        devDebug('[fetchValidationRules] Response status:', res.status);
        
        if (!res.ok) {
          const errorText = await res.text();
          devError('[fetchValidationRules] API error:', errorText);
          throw new Error(`Failed to fetch validation rules: ${res.status} ${errorText}`);
        }
        const data = await res.json();
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
    if (isNewObject) {
      setLoading(false);
      setBusinessObject(null);
      setName('');
      setDisplayName('');
      setDescription('');
      setIsActive(true);
      setFields([]);
      setHierarchyNodes([]);
    } else {
      fetchBusinessObject();
    }
  }, [fetchBusinessObject, isNewObject]);

  useEffect(() => {
    devDebug('[useEffect-validations] activeTab:', activeTab, 'isNewObject:', isNewObject);
    if (activeTab === 1 && !isNewObject) { // Updated index from 2 to 1 for Validations
      devDebug('[useEffect-validations] Triggering fetchValidationRules');
      fetchValidationRules();
    }
  }, [activeTab, fetchValidationRules, isNewObject]);

  // Fetch full schema for rule creator
  useEffect(() => {
    const loadSchema = async () => {
      if (!tenantId || !datasourceId) return;
      try {
        const schema = await fetchEntitySchema(tenantId, datasourceId);
        setEntitySchema(schema);
        setAvailableEntities(Object.keys(schema).sort());
      } catch (error) {
        devError('Error fetching entity schema:', error);
      }
    };
    loadSchema();
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

      const response = await fetch(endpoint, {
        method,
        headers: getAuthHeaders(),
        body: JSON.stringify(rule),
      });

      if (!response.ok) {
        throw new Error(`Failed to save rule: ${response.statusText}`);
      }

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
        const response = await fetch(`/api/business-objects/${parentId}/subtypes/${editingSubtypeId}/rename`, {
          method: 'POST',
          headers: getAuthHeaders(),
          body: JSON.stringify({
            newName: subtypeDisplayName,
          }),
        });

        if (!response.ok) {
          const text = await response.text();
          notification.error(`Failed to update subtype: ${text || response.statusText}`);
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

      const response = await fetch(`/api/business-objects?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}`, {
        method: 'POST',
        headers: getAuthHeaders(),
        body: JSON.stringify({ ...body, datasourceId }),
      });

      if (!response.ok) {
        const text = await response.text();
        notification.error(`Failed to create subtype: ${text || response.statusText}`);
        throw new Error(text || 'Failed to create subtype');
      }

      // const _created = await response.json();
      await response.json();

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
      const response = await fetch(`/api/business-objects/${parentId}/subtypes/${subtypeId}`, {
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

  // Transform current BO state into Entity type for shared components
  // currentEntity is created but not used - keeping for potential future use
  // const _currentEntity: Entity | null = useMemo(() => {
  //   if (!businessObject) return null;
  //   return {
  //     key: id,
  //     name: businessObject.displayName,
  //     businessName: businessObject.displayName,
  //     technicalName: businessObject.technicalName,
  //     description: businessObject.description,
  //     entity_fields: fields.map(f => ({
  //       key: f.key,
  //       name: f.name,
  //       businessName: f.businessName || f.name,
  //       technicalName: f.technicalName || f.name,
  //       type: (f.type.toLowerCase() as any) || 'text',
  //       isCore: businessObject.isCore
  //     })),
  //     subtypes: {},
  //     isCore: businessObject.isCore
  //   };
  // }, [businessObject, id, fields]);

  // Memoized filtered fields (no pagination, lazy loading on demand)
  const filteredFields = useMemo(() => {
    // If root is selected or nothing selected, show root's fields (core + custom)
    // If a subtype is selected, show inherited + subtype-specific fields
    let fieldsToFilter = []
    
    // Use getConfigFields() which resolves selected_terms if needed
    const configFields = getConfigFields();
    
    if (Array.isArray(configFields) && configFields.length > 0) {
      // Use config fields which include semantic term data and resolved selected_terms
      fieldsToFilter = configFields;
    } else if (selectedNode?.type === 'subtype' && selectedNode.subtypeKey && businessObject?.subtypes?.[selectedNode.subtypeKey]) {
      const subtypeFields = businessObject.subtypes[selectedNode.subtypeKey].subtypeFields || [];
      if (showInheritedFields) {
        // Show inherited fields (core + custom) plus subtype-specific fields
        const inheritedFields = [
          ...(businessObject.coreFields || []),
          ...(businessObject.customFields || [])
        ];
        fieldsToFilter = [...inheritedFields, ...subtypeFields];
      } else {
        // Show only subtype-specific fields
        fieldsToFilter = subtypeFields;
      }
    } else {
      // For root business object, show core + custom fields
      fieldsToFilter = [
        ...(businessObject?.coreFields || []),
        ...(businessObject?.customFields || [])
      ];
    }

    return fieldsToFilter.filter(
      (field) =>
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
    <Box sx={{ display: 'flex', flexDirection: 'column', minHeight: '100vh', bgcolor: 'background.default' }}>
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
        <Stack direction={{ xs: 'column', sm: 'row' }} justifyContent="space-between" alignItems={{ xs: 'flex-start', sm: 'center' }} spacing={2} sx={{ mb: 4 }}>
          <Stack direction="column" spacing={1}>
            <Stack direction="row" spacing={2} alignItems="center">
              <Typography variant="h4" sx={{ fontWeight: 900 }}>
                {isNewObject ? 'Create New Business Object' : businessObject?.displayName}
              </Typography>
              {!isNewObject && (
                <Chip
                  label={businessObject?.status === 'active' ? 'Active' : 'Draft'}
                  color={businessObject?.status === 'active' ? 'success' : 'warning'}
                  variant="filled"
                  size="small"
                />
              )}
            </Stack>
            <Typography variant="body2" color="text.secondary">
              {isNewObject 
                ? 'Define a new business object and configure its fields and hierarchy.' 
                : businessObject?.description || 'Core data model for business operations.'}
            </Typography>
          </Stack>

            {isNewObject ? (
              <Chip label="New" color="success" size="small" />
            ) : (
              <Stack direction="row" alignItems="center" spacing={1}>
                {businessObject?.isActive ? (
                  <Chip label="Active" color="success" size="small" />
                ) : (
                  <Chip label="Draft" color="default" size="small" />
                )}
                <Tooltip title="Delete Business Object">
                  <IconButton 
                    size="small" 
                    onClick={() => setDeleteObjectConfirmOpen(true)}
                    sx={{ color: 'text.secondary', '&:hover': { color: 'error.main' } }}
                  >
                    <DeleteIcon />
                  </IconButton>
                </Tooltip>

                <Tooltip title="Edit Object">
                  <IconButton
                    size="medium"
                    onClick={() => setEditModalOpen(true)}
                    sx={{ color: 'primary.main', ml: 1 }}
                    disabled={!businessObject}
                  >
                    <EditIcon sx={{ fontSize: 32 }} />
                  </IconButton>
                </Tooltip>

                <Tooltip title="Add Subtype">
                  <IconButton
                    size="medium"
                    onClick={() => {
                        setEditingSubtypeId(null);
                        setEditingSubtypeKey(null);
                        setSubtypeDisplayName('');
                        setSubtypeName('');
                        setSubtypeDescription('');
                        setAddSubtypeOpen(true);
                    }}
                    sx={{ color: 'primary.main' }}
                    disabled={!businessObject}
                  >
                    <AddIcon sx={{ fontSize: 32 }} />
                  </IconButton>
                </Tooltip>

                <Tooltip title="Add Calculated Field">
                   <IconButton
                      size="medium"
                      onClick={() => setCalcFieldModalOpen(true)}
                      sx={{ color: 'secondary.main' }}
                      disabled={!businessObject}
                   >
                      <FunctionsIcon sx={{ fontSize: 32 }} />
                   </IconButton>
                </Tooltip>
                
                <Tooltip title="Export/Import">
                   <IconButton
                      size="medium"
                      onClick={() => setExportImportWizardOpen(true)}
                      sx={{ color: 'info.main' }}
                      disabled={!businessObject}
                   >
                     <ImportExportIcon sx={{ fontSize: 32 }} />
                   </IconButton>
                </Tooltip>
              </Stack>
            )}

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


        </Stack>

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
                      
                      const response = await fetch('/api/business-objects', {
                        method: 'POST',
                        headers: getAuthHeaders(),
                        body: JSON.stringify(payload),
                      });

                      if (!response.ok) {
                        const errorData = await response.json().catch(() => ({}));
                        throw new Error(errorData.message || 'Failed to create business object');
                      }

                      const createdBO = await response.json();
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
                fontWeight: 500,
              },
            }}
          >
            <Tab label="Hierarchy & Fields" icon={<FolderOpenIcon />} iconPosition="start" />
            {/* <Tab label="Terms" icon={<CategoryOutlinedIcon />} iconPosition="start" /> REMOVED - Merged into Hierarchy & Fields */}
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
          <Box sx={{ display: 'flex', flexDirection: { xs: 'column', lg: 'row' }, gap: 3, p: 3 }}>
            {/* Left Panel: Hierarchy Tree - Always Visible */}
            <Paper
              elevation={0}
              sx={{
                width: { xs: '100%', lg: '30%' },
                border: '1px solid',
                borderColor: 'divider',
                borderRadius: 1,
                overflow: 'hidden',
                display: 'flex',
                flexDirection: 'column',
              }}
            >
              <Box sx={{ p: 2, borderBottom: '1px solid', borderBottomColor: 'divider' }}>
                <Typography variant="subtitle2" sx={{ fontWeight: 700, mb: 2, textTransform: 'uppercase' }}>
                  Object Structure
                </Typography>
                  <TextField
                    fullWidth
                    placeholder="Filter hierarchy..."
                    variant="outlined"
                    size="small"
                    InputProps={{
                      startAdornment: (
                        <InputAdornment position="start">
                          <SearchIcon />
                        </InputAdornment>
                      ),
                    }}
                  />
                </Box>

                <Box sx={{ flex: 1, overflow: 'auto', p: 1 }}>
                  <HierarchyTree
                          nodes={hierarchyNodes}
                          expandedNodes={expandedNodes}
                          onNodeToggle={handleNodeToggle}
                          _businessObject={businessObject}
                    selectedNode={selectedNode}
                    onNodeSelect={setSelectedNode}
                    onRenameSubtype={openRenameDialog}
                    onDeleteSubtype={openDeleteConfirm}
                  />
                </Box>

                <Box
                  sx={{
                    p: 2,
                    borderTop: '1px solid',
                    borderTopColor: 'divider',
                    bgcolor: 'action.hover',
                  }}
                >
                  <Stack direction="row" justifyContent="space-between" alignItems="center">
                    <Typography variant="caption" color="text.secondary">
                      Last modified: 2 hours ago
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      {businessObject?.version}
                    </Typography>
                  </Stack>
                </Box>
              </Paper>

              {/* Right Panel: Tab Content */}
              <Paper
                elevation={0}
                sx={{
                  width: { xs: '100%', lg: '70%' },
                  border: '1px solid',
                  borderColor: 'divider',
                  borderRadius: 1,
                  overflow: 'hidden',
                  display: 'flex',
                  flexDirection: 'column',
                }}
              >
                {activeTab === 0 && (
                  <Box sx={{ display: 'flex', flexDirection: 'column', flex: 1 }}>
                    <Box sx={{ p: 3, borderBottom: '1px solid', borderBottomColor: 'divider' }}>
                      <Stack direction={{ xs: 'column', sm: 'row' }} justifyContent="space-between" alignItems={{ xs: 'flex-start', sm: 'center' }} spacing={2}>
                        <Box>
                          {selectedNode?.type === 'subtype' ? (
                            <>
                              <Typography variant="h6" sx={{ fontWeight: 700, mb: 0.5 }}>
                                Fields for '{businessObject?.subtypes?.[selectedNode.subtypeKey!]?.displayName || selectedNode.subtypeKey}'
                              </Typography>
                              <Typography variant="body2" color="text.secondary">
                                Showing {showInheritedFields ? 'inherited + subtype-specific' : 'subtype-specific only'} fields.
                              </Typography>
                            </>
                          ) : (
                            <>
                              <Typography variant="h6" sx={{ fontWeight: 700, mb: 0.5 }}>
                                Fields for '{businessObject?.displayName}'
                              </Typography>
                              <Typography variant="body2" color="text.secondary">
                                Define data types, constraints, and display logic for this node.
                              </Typography>
                            </>
                          )}
                        </Box>
                        <Stack direction="row" spacing={2} alignItems="center">
                          {selectedNode?.type === 'subtype' && (
                            <Button
                              variant={showInheritedFields ? 'contained' : 'outlined'}
                              color="primary"
                              size="small"
                              onClick={() => setShowInheritedFields(!showInheritedFields)}
                            >
                              {showInheritedFields ? 'Hide Inherited' : 'Show Inherited'}
                            </Button>
                          )}
                          <Button
                            variant="contained"
                            color="primary"
                            startIcon={<AddIcon />}
                            size="small"
                            onClick={() => setFieldWizardOpen(true)}
                          >
                            Add Field
                          </Button>
                        </Stack>
                      </Stack>
                    </Box>

                    <TextField
                      fullWidth
                      placeholder="Search fields..."
                      variant="standard"
                      value={searchFilter}
                      onChange={(e) => {
                        setSearchFilter(e.target.value);
                      }}
                      InputProps={{
                        startAdornment: (
                          <InputAdornment position="start">
                            <SearchIcon fontSize="small" />
                          </InputAdornment>
                        ),
                      }}
                      sx={{
                        px: 3,
                        py: 1,
                        mb: 2,
                        '& .MuiInput-underline:before': {
                          borderBottomColor: 'divider',
                        },
                      }}
                    />

                    <TableContainer sx={{ flex: 1 }}>
                      <Table stickyHeader>
                        <TableHead>
                          <TableRow sx={{ bgcolor: 'action.hover' }}>
                            <TableCell sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: '0.75rem' }}>
                              <TableSortLabel
                                active={sortConfig.key === 'technicalName'}
                                direction={sortConfig.key === 'technicalName' ? sortConfig.direction : 'asc'}
                                onClick={() => handleRequestSort('technicalName')}
                              >
                                Technical Name
                              </TableSortLabel>
                            </TableCell>
                            <TableCell sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: '0.75rem' }}>
                              <TableSortLabel
                                active={sortConfig.key === 'businessName'}
                                direction={sortConfig.key === 'businessName' ? sortConfig.direction : 'asc'}
                                onClick={() => handleRequestSort('businessName')}
                              >
                                Display Label
                              </TableSortLabel>
                            </TableCell>
                            {selectedNode?.type === 'subtype' && (
                              <TableCell sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: '0.75rem' }}>
                                Type
                              </TableCell>
                            )}
                            <TableCell sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: '0.75rem' }}>
                              <TableSortLabel
                                active={sortConfig.key === 'type'}
                                direction={sortConfig.key === 'type' ? sortConfig.direction : 'asc'}
                                onClick={() => handleRequestSort('type')}
                              >
                                Data Type
                              </TableSortLabel>
                            </TableCell>
                            <TableCell sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: '0.75rem' }}>
                              Validation
                            </TableCell>
                            <TableCell align="right" sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: '0.75rem' }}>
                              Actions
                            </TableCell>
                          </TableRow>
                        </TableHead>
                        <TableBody>
                          {sortedFilteredFields.map((field) => {
                            // Determine if field is inherited (from parent) or assigned to subtype
                            const isInherited = selectedNode?.type === 'subtype' && 
                              showInheritedFields && 
                              (businessObject?.coreFields?.some(f => f.key === field.key) ||
                               businessObject?.customFields?.some(f => f.key === field.key));
                            
                            // Determine visual style based on data type
                            const getDataTypeConfig = (type: string) => {
                              const t = type.toLowerCase();
                              if (t.includes('int') || t.includes('number') || t.includes('decimal') || t.includes('float') || t.includes('double')) {
                                return { icon: <NumberIcon sx={{ fontSize: 16 }} />, color: 'success', label: type };
                              } else if (t.includes('date') || t.includes('time')) {
                                return { icon: <DateIcon sx={{ fontSize: 16 }} />, color: 'secondary', label: type };
                              } else if (t.includes('bool')) {
                                return { icon: <BooleanIcon sx={{ fontSize: 16 }} />, color: 'warning', label: type };
                              } else if (t.includes('json') || t.includes('obj') || t.includes('arr')) {
                                return { icon: <JsonIcon sx={{ fontSize: 16 }} />, color: 'info', label: type };
                              } else {
                                return { icon: <TextIcon sx={{ fontSize: 16 }} />, color: 'primary', label: type };
                              }
                            };
                            
                            const typeConfig = getDataTypeConfig(field.type);

                            return (
                            <TableRow
                              key={field.key}
                              hover
                              sx={{
                                '&:hover': {
                                  bgcolor: 'action.hover',
                                },
                                opacity: isInherited ? 0.8 : 1,
                              }}
                            >
                            <TableCell sx={{ fontWeight: 600, fontFamily: 'monospace', fontSize: '0.85rem' }}>
                              {field.technicalName || field.name}
                            </TableCell>
                              <TableCell>
                                <Stack direction="row" spacing={1} alignItems="center">
                                  <Typography variant="body2" sx={{ fontWeight: 500 }}>
                                    {field.businessName || field.name}
                                  </Typography>
                                  {field.description && (
                                    <Tooltip title={field.description} arrow placement="right">
                                      <InfoOutlinedIcon sx={{ fontSize: 16, color: 'text.secondary', cursor: 'help' }} />
                                    </Tooltip>
                                  )}
                                </Stack>
                              </TableCell>
                              {selectedNode?.type === 'subtype' && (
                                <TableCell>
                                  <Chip
                                    label={isInherited ? 'Inherited' : 'Assigned'}
                                    size="small"
                                    variant="filled"
                                    color={isInherited ? 'default' : 'primary'}
                                    sx={{
                                      fontWeight: 600,
                                      fontSize: '0.7rem',
                                    }}
                                  />
                                </TableCell>
                              )}

                              <TableCell>
                                <Chip
                                  icon={typeConfig.icon}
                                  label={typeConfig.label}
                                  size="small"
                                  color={typeConfig.color as any}
                                  variant="outlined"
                                  sx={{ fontWeight: 500, border: '1px solid' }}
                                />
                              </TableCell>
                              <TableCell>
                                <Stack direction="row" spacing={1} alignItems="center">
                                  {getValidationIcon(field.validation)}
                                  <Typography variant="body2" color="text.secondary">
                                    {field.validationMessage || '-'}
                                  </Typography>
                                </Stack>
                              </TableCell>
                              <TableCell align="right">
                                <Stack direction="row" spacing={0.5} justifyContent="flex-end">
                                  <Tooltip title="Edit field">
                                    <IconButton size="small" onClick={() => handleEditField(field)} sx={{ '&:hover': { color: 'primary.main' } }}>
                                      <EditIcon fontSize="small" />
                                    </IconButton>
                                  </Tooltip>
                                  <Tooltip title="Delete field">
                                    <IconButton size="small" onClick={() => handleDeleteField(field)} sx={{ '&:hover': { color: 'error.main' } }}>
                                      <DeleteIcon fontSize="small" />
                                    </IconButton>
                                  </Tooltip>
                                  <IconButton size="small" sx={{ '&:hover': { color: 'primary.main' } }}>
                                    <MoreVertIcon fontSize="small" />
                                  </IconButton>
                                </Stack>
                              </TableCell>
                            </TableRow>
                            );
                          })}
                        </TableBody>
                      </Table>
                    </TableContainer>
                  </Box>
                )}

              {/* Terms Tab - REMOVED */}
              {activeTab === 1 && (
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

              {/* Validations Tab - Moved to index 1, effectively replaced above. Keeping index 2 for Related Objects replacement.*/}
              {/* Related Objects Tab - Moved to index 2 */}
              {activeTab === 2 && (
                <Box sx={{ p: 3 }}>
                  <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 3 }}>
                    <Typography variant="h6">Related Objects</Typography>
                    <Stack direction="row" spacing={2} alignItems="center">
                      <Tooltip title="Add Relationship">
                        <IconButton
                          size="large"
                          onClick={() => setRelationshipWizardOpen(true)}
                          sx={{ color: 'primary.main' }}
                        >
                          <AddLinkIcon sx={{ fontSize: 32 }} />
                        </IconButton>
                      </Tooltip>
                      <Tooltip title="Tile View">
                        <IconButton
                          size="medium"
                          onClick={() => setRelatedObjectsView('tile')}
                          component="button"
                          color={relatedObjectsView === 'tile' ? 'primary' : 'default'}
                          sx={{
                            border: relatedObjectsView === 'tile' ? '2px solid' : '1px solid',
                            borderColor: relatedObjectsView === 'tile' ? 'primary.main' : 'divider',
                          }}
                        >
                          <AppsIcon sx={{ fontSize: 28 }} />
                        </IconButton>
                      </Tooltip>
                      <Tooltip title="Table View">
                        <IconButton
                          size="medium"
                          onClick={() => setRelatedObjectsView('table')}
                          component="button"
                          color={relatedObjectsView === 'table' ? 'primary' : 'default'}
                          sx={{
                            border: relatedObjectsView === 'table' ? '2px solid' : '1px solid',
                            borderColor: relatedObjectsView === 'table' ? 'primary.main' : 'divider',
                          }}
                        >
                          <TableChartIcon sx={{ fontSize: 28 }} />
                        </IconButton>
                      </Tooltip>
                      <Tooltip title="Graph View">
                        <IconButton
                          size="medium"
                          onClick={() => setRelatedObjectsView('graph')}
                          component="button"
                          color={relatedObjectsView === 'graph' ? 'primary' : 'default'}
                          sx={{
                            border: relatedObjectsView === 'graph' ? '2px solid' : '1px solid',
                            borderColor: relatedObjectsView === 'graph' ? 'primary.main' : 'divider',
                          }}
                        >
                          <AccountTreeIcon sx={{ fontSize: 28 }} />
                        </IconButton>
                      </Tooltip>
                    </Stack>
                  </Stack>

                  {/* Tile View */}
                  {relatedObjectsView === 'tile' && (
                    <Box>
                      <Typography variant="body2" color="text.secondary" sx={{ textAlign: 'center', py: 5 }}>
                        No related objects found. Related objects will appear here once they are linked to this business object.
                      </Typography>
                    </Box>
                  )}

                  {/* Table View */}
                  {relatedObjectsView === 'table' && (
                    <Box>
                      <Typography variant="body2" color="text.secondary" sx={{ textAlign: 'center', py: 5 }}>
                        No related objects found. Related objects will appear here once they are linked to this business object.
                      </Typography>
                    </Box>
                  )}

                  {/* Graph View */}
                  {relatedObjectsView === 'graph' && (
                    <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', py: 5, minHeight: 300 }}>
                      <Typography variant="body2" color="text.secondary">
                        No related objects to display. Graph visualization will appear here once relationships are established.
                      </Typography>
                    </Box>
                  )}
                </Box>
              )}

              {/* Graph Tab - Moved to index 3 */}
              {activeTab === 3 && (
                <Box sx={{ height: '70vh', p: 2 }}>
                  <BOLineageGraphTab boId={id || ''} />
                </Box>
              )}

              {/* Semantic Model Tab - Moved to index 4 */}
              {activeTab === 4 && (
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

              {/* Lineage & Impact Tab - Moved to index 5 */}
              {/* Lineage & Impact Tab - Moved to index 5 */}
              {activeTab === 5 && (
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
    <Dialog 
      open={fieldDeleteConfirmOpen} 
      onClose={() => {
        setFieldDeleteConfirmOpen(false);
        setFieldPendingDelete(null);
      }} 
      maxWidth="sm" 
      fullWidth
    >
      <DialogTitle sx={{ fontWeight: 700, color: 'error.main' }}>🗑️ Remove Field?</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ mt: 2 }}>
          <Alert severity="error">
            This will permanently remove this field from the business object. This action cannot be undone.
          </Alert>
          {fieldPendingDelete && (
            <Box sx={{ bgcolor: 'action.hover', p: 2, borderRadius: 1 }}>
              <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 0.5 }}>
                Field to remove:
              </Typography>
              <Typography variant="subtitle2" sx={{ fontWeight: 700 }}>
                {fieldPendingDelete.businessName || fieldPendingDelete.name}
              </Typography>
              <Typography variant="caption" color="text.secondary" sx={{ fontFamily: 'monospace' }}>
                {fieldPendingDelete.technicalName || fieldPendingDelete.key}
              </Typography>
              {(fieldPendingDelete.semanticTermName || fieldPendingDelete.semantic_term_name) && (
                <>
                  <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 1 }}>
                    Semantic Term:
                  </Typography>
                  <Chip 
                    label={fieldPendingDelete.semanticTermName || fieldPendingDelete.semantic_term_name}
                    size="small"
                    variant="outlined"
                    sx={{ mt: 0.5 }}
                  />
                </>
              )}
            </Box>
          )}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button 
          onClick={() => {
            setFieldDeleteConfirmOpen(false);
            setFieldPendingDelete(null);
          }}
          disabled={isDeleting}
        >
          Cancel
        </Button>
        <Button 
          variant="contained" 
          color="error"
          onClick={handleConfirmDeleteField}
          disabled={isDeleting}
        >
          {isDeleting ? 'Removing...' : 'Remove Field'}
        </Button>
      </DialogActions>
    </Dialog>

    {/* Add/Edit Subtype Dialog */}
    <Dialog 
      open={addSubtypeOpen} 
      onClose={() => {
        setEditingSubtypeId(null);
        setEditingSubtypeKey(null);
        setAddSubtypeOpen(false);
      }} 
      maxWidth="sm" 
      fullWidth
    >
      <DialogTitle sx={{ fontWeight: 700, fontSize: '1.25rem' }}>
        {editingSubtypeKey ? '✏️ Edit Subtype' : '➕ Add New Subtype'}
      </DialogTitle>
      <DialogContent>
        <Stack spacing={3} sx={{ mt: 2 }}>
          <TextField
            fullWidth
            label="Display Name"
            placeholder="e.g., Commercial Customer"
            value={subtypeDisplayName}
            onChange={(e) => setSubtypeDisplayName(e.target.value)}
            helperText="Human-readable name for this subtype"
            variant="outlined"
            autoFocus
          />
          <TextField
            fullWidth
            label="Technical Name"
            placeholder="e.g., commercial_customer"
            value={subtypeName}
            onChange={(e) => setSubtypeName(e.target.value)}
            helperText="Lowercase letters, numbers, and underscores only. Leave empty to auto-generate from display name."
            variant="outlined"
          />
          {!subtypeName.trim() && subtypeDisplayName.trim() && (
            <Typography variant="body2" color="primary" sx={{ p: 1.5, bgcolor: 'action.hover', borderRadius: 1 }}>
              <strong>Suggested technical name:</strong> <code>{subtypeDisplayName.trim().toLowerCase().replace(/\s+/g, '_')}</code>
            </Typography>
          )}
          <TextField
            fullWidth
            label="Description"
            placeholder="Describe what this subtype represents..."
            value={subtypeDescription}
            onChange={(e) => setSubtypeDescription(e.target.value)}
            helperText="Optional. Helps other team members understand this variation"
            multiline
            rows={3}
            variant="outlined"
          />
          <Alert severity="info" icon={<InfoIcon />}>
            Subtypes inherit all core fields from {businessObject?.displayName} and can have their own additional fields.
          </Alert>
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={() => {
          setEditingSubtypeId(null);
          setEditingSubtypeKey(null);
          setAddSubtypeOpen(false);
        }}>Cancel</Button>
        <Button 
          variant="contained" 
          onClick={handleAddSubtype} 
          disabled={subtypeSaving || !subtypeDisplayName.trim()}
        >
          {subtypeSaving ? 'Saving...' : editingSubtypeKey ? 'Update Subtype' : 'Create Subtype'}
        </Button>
      </DialogActions>
    </Dialog>

    {/* Rename Subtype Dialog */}
    {/* REMOVED - Edit now uses the Add dialog */}

    {/* Delete Subtype Confirmation Dialog */}
    <Dialog 
      open={deleteConfirmOpen} 
      onClose={() => {
        setDeleteConfirmOpen(false);
        setDeletingSubtypeKey(null);
        setDeleteConfirmInput('');
      }} 
      maxWidth="sm" 
      fullWidth
    >
      <DialogTitle sx={{ fontWeight: 700, color: 'error.main' }}>🗑️ Delete Subtype?</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ mt: 2 }}>
          <Alert severity="error">
            This will permanently delete this subtype and cannot be undone.
          </Alert>
          {deletingSubtypeKey && businessObject?.subtypes && businessObject.subtypes[deletingSubtypeKey] && (
            <>
              <Box sx={{ bgcolor: 'action.hover', p: 2, borderRadius: 1 }}>
                <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 0.5 }}>
                  Deleting:
                </Typography>
                <Typography variant="subtitle2" sx={{ fontWeight: 700 }}>
                  {businessObject.subtypes[deletingSubtypeKey].displayName || businessObject.subtypes[deletingSubtypeKey].name}
                </Typography>
                <Typography variant="caption" color="text.secondary" sx={{ fontFamily: 'monospace' }}>
                  {businessObject.subtypes[deletingSubtypeKey].technicalName || deletingSubtypeKey}
                </Typography>
              </Box>
              
              <Box>
                <Typography variant="body2" sx={{ fontWeight: 600, mb: 1 }}>
                  To confirm, type the technical name:
                </Typography>
                <Typography 
                  variant="body2" 
                  sx={{ 
                    fontFamily: 'monospace', 
                    bgcolor: 'background.paper',
                    p: 1,
                    borderRadius: 1,
                    border: '1px solid',
                    borderColor: 'divider',
                    mb: 2
                  }}
                >
                  {businessObject.subtypes[deletingSubtypeKey].technicalName || deletingSubtypeKey}
                </Typography>
                <TextField
                  fullWidth
                  size="small"
                  placeholder="Enter technical name to confirm"
                  value={deleteConfirmInput}
                  onChange={(e) => setDeleteConfirmInput(e.target.value)}
                  sx={{ 
                    '& .MuiOutlinedInput-root': {
                      '&.Mui-focused fieldset': {
                        borderColor: 'error.main',
                      }
                    }
                  }}
                />
              </Box>
            </>
          )}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={() => {
          setDeleteConfirmOpen(false);
          setDeletingSubtypeId(null);
          setDeletingSubtypeKey(null);
          setDeleteConfirmInput('');
        }}>Cancel</Button>
        <Button 
          variant="contained" 
          color="error"
          disabled={deleteConfirmInput !== (deletingSubtypeKey && businessObject?.subtypes?.[deletingSubtypeKey]?.technicalName || deletingSubtypeKey)}
          onClick={() => {
            if (deletingSubtypeId) {
              handleDeleteSubtype(deletingSubtypeId);
              setDeleteConfirmInput('');
            }
          }}
        >
          Delete Permanently
        </Button>
      </DialogActions>
    </Dialog>

    {/* Edit Business Object Modal */}
    {businessObject && (
      <>
        {/* Relationship Wizard */}
        <BusinessObjectRelationshipWizard
          open={relationshipWizardOpen}
          onClose={() => setRelationshipWizardOpen(false)}
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

            const response = await fetch(`/api/business-objects/${businessObject.id}`, {
              method: 'PUT',
              headers: getAuthHeaders(),
              body: JSON.stringify(payload),
            });

            if (!response.ok) {
              throw new Error('Failed to update business object');
            }

            const updated = await response.json();
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
    <Dialog 
      open={editFieldModalOpen} 
      onClose={() => setEditFieldModalOpen(false)}
      maxWidth="sm"
      fullWidth
    >
      <DialogTitle>Edit Field</DialogTitle>
      <DialogContent sx={{ pt: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>
        <TextField
          label="Display Name"
          fullWidth
          value={editedFieldData.displayName}
          onChange={(e) => setEditedFieldData(prev => ({ ...prev, displayName: e.target.value }))}
          sx={{ mt: 1 }}
        />
        <TextField
          label="Description"
          fullWidth
          multiline
          rows={3}
          value={editedFieldData.description}
          onChange={(e) => setEditedFieldData(prev => ({ ...prev, description: e.target.value }))}
        />
        
        <Autocomplete
          options={semanticTerms}
          getOptionLabel={(option) => option.node_name}
          value={semanticTerms.find(t => t.id === editedFieldData.semanticTermId) || null}
          onChange={(_, newValue) => {
              setEditedFieldData(prev => ({ 
                  ...prev, 
                  semanticTermId: newValue?.id || '' 
              }));
          }}
          renderInput={(params) => <TextField {...params} label="Semantic Term" />}
        />
        
        <TextField
          id="role-select"
          select
          label="Role"
          fullWidth
          InputLabelProps={{ id: 'role-select-label' }}
          value={editedFieldData.role}
          onChange={(e) => setEditedFieldData(prev => ({ ...prev, role: e.target.value }))}
          SelectProps={{ native: true, inputProps: { 'aria-label': 'Select field role', title: 'Select the role for this field', id: 'role-select', 'aria-labelledby': 'role-select-label' } }}
        >
            <option value="">None</option>
            <option value="DIMENSION">Dimension</option>
            <option value="MEASURE">Measure</option>
            <option value="ATTRIBUTE">Attribute</option>
        </TextField>

      </DialogContent>
      <DialogActions>
        <Button onClick={() => setEditFieldModalOpen(false)}>Cancel</Button>
        <Button variant="contained" onClick={handleSaveFieldEdit}>Save</Button>
      </DialogActions>
    </Dialog>
    {/* Delete Business Object Confirmation Dialog */}
    <Dialog
      open={deleteObjectConfirmOpen}
      onClose={() => setDeleteObjectConfirmOpen(false)}
      maxWidth="sm"
      fullWidth
    >
      <DialogTitle sx={{ fontWeight: 700, color: 'error.main' }}>🗑️ Delete Business Object?</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ mt: 2 }}>
          <Alert severity="error">
            Are you sure you want to delete this Business Object? This action cannot be undone and will delete all associated data and configuration.
          </Alert>
          <Box sx={{ bgcolor: 'action.hover', p: 2, borderRadius: 1 }}>
            <Typography variant="subtitle2" sx={{ fontWeight: 700 }}>
              {businessObject?.displayName}
            </Typography>
            <Typography variant="caption" color="text.secondary" sx={{ fontFamily: 'monospace' }}>
              {businessObject?.technicalName}
            </Typography>
          </Box>
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={() => setDeleteObjectConfirmOpen(false)}>Cancel</Button>
        <Button 
          variant="contained" 
          color="error" 
          onClick={handleDeleteBusinessObject}
        >
          Delete Permanently
        </Button>
      </DialogActions>
    </Dialog>
  </>
  );
}

// Hierarchy Tree Component
interface HierarchyTreeProps {
  nodes: HierarchyNode[];
  expandedNodes: Set<string>;
  onNodeToggle: (nodeId: string) => void;
  _businessObject?: BusinessObject | null;
  selectedNode: any;
  onNodeSelect: (node: any) => void;
  onRenameSubtype: (key: string, name: string) => void;
  onDeleteSubtype: (key: string) => void;
}

function HierarchyTree({
  nodes,
  expandedNodes,
  onNodeToggle,
  // _businessObject,
  selectedNode,
  onNodeSelect,
  onRenameSubtype,
  onDeleteSubtype,
}: HierarchyTreeProps) {
  return (
    <Box component="ul" sx={{ listStyle: 'none', p: 0, m: 0 }}>
      {nodes.map((node) => (
        <HierarchyTreeNode
          key={node.id}
          node={node}
          expandedNodes={expandedNodes}
          onNodeToggle={onNodeToggle}
          selectedNode={selectedNode}
          onNodeSelect={onNodeSelect}
          onRenameSubtype={onRenameSubtype}
          onDeleteSubtype={onDeleteSubtype}
        />
      ))}
    </Box>
  );
}

function HierarchyTreeNode({
  node,
  expandedNodes,
  onNodeToggle,
  selectedNode,
  onNodeSelect,
  onRenameSubtype,
  onDeleteSubtype,
}: {
  node: HierarchyNode;
  expandedNodes: Set<string>;
  onNodeToggle: (nodeId: string) => void;
  selectedNode: any;
  onNodeSelect: (node: any) => void;
  onRenameSubtype: (key: string, name: string) => void;
  onDeleteSubtype: (key: string) => void;
}) {
  const hasChildren = node.children && node.children.length > 0;
  const isSubtypeNode = (node as any).isSubtype;
  const subtypeKey = (node as any).subtypeKey;
  const technicalName = (node as any).technicalName;
  const isRootNode = node.id === 'root';
  const isSelected = (isRootNode && !selectedNode) || (selectedNode?.subtypeKey === subtypeKey && isSubtypeNode);

  const handleNodeClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (isRootNode) {
      onNodeSelect(null);
    } else if (isSubtypeNode) {
      onNodeSelect({ type: 'subtype', subtypeKey, key: node.id });
    }
  };

  return (
    <Box component="li" sx={{ listStyle: 'none', mb: 0.5 }}>
      <Stack
        direction="row"
        spacing={1}
        alignItems="center"
        onClick={handleNodeClick}
        sx={{
          p: 1,
          borderRadius: 1,
          cursor: 'pointer',
          bgcolor: 
            isSelected 
              ? 'primary.light' 
              : node.id === 'root' ? 'primary.light' : 'transparent',
          color: 
            isSelected || node.id === 'root' 
              ? 'primary.main' 
              : 'text.primary',
          fontWeight: node.id === 'root' || isSelected ? 700 : 400,
          transition: 'all 0.2s ease',
          '&:hover': {
            bgcolor: node.id === 'root' || isSelected ? 'primary.light' : 'action.hover',
          },
        }}
      >
        <Box sx={{ width: 24, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          {isRootNode ? (
            <BusinessObjectIcon sx={{ fontSize: '1.25rem', color: 'primary.main' }} />
          ) : isSubtypeNode ? (
            <SubtypeIcon sx={{ fontSize: '1.25rem', color: 'info.main' }} />
          ) : null}
        </Box>
        <Stack direction="column" spacing={0} flex={1}>
          <Typography variant="body2">{node.displayName}</Typography>
          {technicalName && isSubtypeNode && (
            <Typography 
              variant="caption" 
              color="text.secondary" 
              sx={{ fontFamily: 'monospace', fontSize: '0.7rem' }}
              title={`Technical name: ${technicalName}`}
            >
              {technicalName}
            </Typography>
          )}
        </Stack>

        {isSubtypeNode && (
          <Stack direction="row" spacing={0.5} sx={{ ml: 'auto' }}>
            <IconButton
              size="small"
              color="primary"
              onClick={(e) => {
                e.stopPropagation();
                onRenameSubtype(subtypeKey || '', node.displayName || '');
              }}
              title="Edit subtype"
              sx={{
                '&:hover': { bgcolor: 'primary.light', color: 'primary.dark' },
              }}
            >
              <EditIcon fontSize="small" />
            </IconButton>
            <IconButton
              size="small"
              color="error"
              onClick={(e) => {
                e.stopPropagation();
                onDeleteSubtype(subtypeKey);
              }}
              title="Delete subtype"
              sx={{
                '&:hover': { bgcolor: 'error.light', color: 'error.dark' },
              }}
            >
              <DeleteIcon fontSize="small" />
            </IconButton>
            <IconButton
              size="small"
              color="info"
              onClick={(e) => {
                e.stopPropagation();
                // Clone functionality will be added later
              }}
              title="Clone subtype"
              sx={{
                '&:hover': { bgcolor: 'info.light', color: 'info.dark' },
              }}
            >
              <CloneIcon fontSize="small" />
            </IconButton>
          </Stack>
        )}
      </Stack>

      {hasChildren && (
        <Box component="ul" sx={{ listStyle: 'none', p: 0, m: 0, pl: 2, borderLeft: '2px solid', borderLeftColor: 'divider', ml: 2 }}>
          {node.children?.map((child) => (
            <HierarchyTreeNode
              key={child.id}
              node={child}
              expandedNodes={expandedNodes}
              onNodeToggle={onNodeToggle}
              selectedNode={selectedNode}
              onNodeSelect={onNodeSelect}
              onRenameSubtype={onRenameSubtype}
              onDeleteSubtype={onDeleteSubtype}
            />
          ))}
        </Box>
      )}


    </Box>
  );
}
