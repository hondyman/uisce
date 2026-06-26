import { useState, useEffect } from 'react';
import {
  Box,
  Container,
  Typography,
  Button,
  Chip,
  Tabs,
  Tab,
} from '@mui/material';
import { useLocation, useParams, useNavigate } from 'react-router-dom';
import { ArrowLeft } from 'lucide-react';
import type { Entities, Entity } from '../types/entity-schema';
// import EntityDrawerTreeView from '../components/EntityDrawerTreeView'; // Replaced by EntityDetailsEditor
import EntityDetailsEditor from '../components/EntityDetailsEditor';
import { fetchEntitySchema } from '../api/entitySchema';
import { useTenant } from '../contexts/TenantContext';
import { devLog, devError } from '../utils/devLogger';
import { filterValidationRulesForEntity, type AnnotatedValidationRule } from '../utils/validationRules';
import ValidationsTab from '../components/validation/ValidationsTab';
import { useBusinessEntitySemanticLayer } from '../hooks/useBusinessEntitySemanticLayer';
import SemanticAssetsTab from '../components/entity/SemanticAssetsTab';
import RelationshipsTab from '../components/relationship/RelationshipsTab';
import { ValidationRuleCreator } from '../components/ValidationRules/ValidationRuleCreator';
import { useNotification } from '../hooks/useNotification';

// Transform backend response structure to match frontend expectations
// Backend returns fields in a nested config object, but frontend expects them at top level
function transformEntitySchema(rawSchema: any): Entities {
  const transformed: Entities = {};

  for (const [key, rawEntity] of Object.entries(rawSchema)) {
    const entity = rawEntity as any;
    
    // Flatten config to top level
    const config = entity.config || {};
    const transformed_entity: Entity = {
      id: entity.id, // Ensure ID is passed if available
      key: entity.id || key,
      name: entity.name || '',
      businessName: config.businessName || entity.display_name || entity.name,
      technicalName: config.technical_name || entity.technical_name,
      description: config.description,
      entity_fields: config.entity_fields || [],
      subtypes: {},
      isCore: config.isCore,
      coreFields: [],
      customFields: [],
    };

    // Transform subtypes
    if (entity.subtypes && typeof entity.subtypes === 'object') {
      for (const [subtypeId, rawSubtype] of Object.entries(entity.subtypes)) {
        const subtype = rawSubtype as any;
        const subtypeConfig = subtype.config || {};
        
        // Subtypes inherit entity_fields from their parent, plus have their own subtype_fields
        const inheritedFields = subtypeConfig.entity_fields || config.entity_fields || [];
        
        devLog(`Transforming subtype ${subtypeId}:`, {
          hasSubtypeConfig: !!subtype.config,
          inheritedFieldsCount: inheritedFields.length,
          subtypeFieldsCount: (subtypeConfig.subtype_fields || []).length,
        });

        transformed_entity.subtypes[subtypeId] = {
          key: subtypeId,
          name: subtype.name || '',
          businessName: subtypeConfig.businessName || subtype.display_name || subtype.name,
          technicalName: subtypeConfig.technical_name || subtype.technical_name,
          entity_fields: inheritedFields,
          subtype_fields: subtypeConfig.subtype_fields || [],
          isCore: subtypeConfig.isCore,
          basedOnEntity: config.basedOnEntity,
        };
      }
    }

    transformed[key] = transformed_entity;
  }

  return transformed;
}

export default function EntityDetailsPage() {
  const { entityKey } = useParams<{ entityKey: string }>();
  const location = useLocation();
  const navigate = useNavigate();
  const { tenant: contextTenant, datasource: contextDatasource } = useTenant();

  // Prioritize state from navigation, but fall back to context.
  const tenant = location.state?.tenant || contextTenant;
  const datasource = location.state?.datasource || contextDatasource;

  // Check if tenant scope is selected
  const hasTenantScope = !!(tenant?.id && datasource?.id);

  const [entities, setEntities] = useState(location.state?.entities as Entities | null ?? null);
  const [loading, setLoading] = useState(!location.state?.entities);
  const [validationRules, setValidationRules] = useState([] as AnnotatedValidationRule[]);
  const [activeTab, setActiveTab] = useState('entity');
  const [validationRuleCreatorOpen, setValidationRuleCreatorOpen] = useState(false);
  const [rawSchema, setRawSchema] = useState<any>(null);
  const [availableEntities, setAvailableEntities] = useState<any[]>([]);
  const [editingRule, setEditingRule] = useState<AnnotatedValidationRule | null>(null);
  const notification = useNotification();

  // Initialize semantic layer for business entity
  const semanticLayer = useBusinessEntitySemanticLayer({
    tenantId: tenant?.id || '',
    datasourceId: datasource?.id || datasource?.alpha_tenant_instance_id || '',
    businessEntityId: (entityKey && entities?.[entityKey]?.id) || '',
    businessEntityName: (entityKey && entities?.[entityKey]?.name) || entityKey || '',
    semanticTermIds: [],
    sourceTableNames: [],
  });

  // Fetch validation rules for this entity
  const fetchValidationRules = async () => {
    devLog('🟠 fetchValidationRules called', { 
      tenant: tenant?.id, 
      datasource: datasource?.id, 
      entityKey, 
      hasEntities: !!entities 
    });
    if (!tenant || !datasource || !entityKey || !entities || !entities[entityKey]) {
      devLog('🔴 Early return - missing params');
      return;
    }

    try {
      let allRules: any[] = [];
      let page = 1;
      let hasMore = true;

      // Fetch all pages of validation rules for this SPECIFIC entity
      while (hasMore) {
        devLog(`🟡 Fetching page ${page} for entity: ${entityKey}...`);
        
        // Build query with entity filter - PREFER entity_ids (UUID) over entities (name)
        const params = new URLSearchParams({
          tenant_id: tenant.id,
          tenant_instance_id: datasource.id || datasource.alpha_tenant_instance_id,
          page: String(page),
          limit: '100',
        });

        // Use entity name for filtering
        devLog(`📍 Using entity name filtering: ${entityKey}`);
        params.append('entities', entityKey);
        
        const res = await fetch(
          `/api/validation-rules?${params.toString()}`,
          {
            headers: { 
              'X-Tenant-ID': tenant.id,
              'X-Tenant-Datasource-ID': datasource.id || datasource.alpha_tenant_instance_id,
            },
          }
        );
        if (!res.ok) throw new Error(`Failed to fetch validation rules: ${res.statusText}`);
        const data = await res.json();
        const raw = Array.isArray(data) ? data : (data.rules || []);
        
        allRules = allRules.concat(raw);
        hasMore = data.has_more;
        page++;
      }
      
      const filtered = filterValidationRulesForEntity(entityKey, entities[entityKey], allRules);
      setValidationRules(filtered);
    } catch (err) { devError('EntityDetailsPage: Failed to fetch validation rules:', err); }
  };

  // FIX: All hooks must be called before any conditional returns.
  // Moved the validation rules useEffect and other state hooks to the top.
  useEffect(() => {
    fetchValidationRules();
  }, [tenant, datasource, entityKey, entities]);

  // Refresh validation rules when validations tab becomes active
  useEffect(() => {
    if (activeTab === 'validations') {
      fetchValidationRules();
    }
  }, [activeTab]);

  useEffect(() => {
    // If entities were not passed in state (e.g., page refresh), fetch them.
    const shouldFetch = !location.state?.entities;

    if (shouldFetch && tenant && datasource) {
      devLog('EntityDetailsPage: Fetching entities for tenant:', tenant.id, 'datasource:', datasource.id || datasource.alpha_tenant_instance_id);
      setLoading(true);
      fetchEntitySchema(tenant.id, datasource.id || datasource.alpha_tenant_instance_id)
        .then((schema) => {
          // Transform backend response structure to match frontend expectations
          const transformedSchema = transformEntitySchema(schema);
          setEntities(transformedSchema || {});
        })
        .catch((error) => {
          devError('EntityDetailsPage: Error fetching schema:', error);
        })
        .finally(() => {
          setLoading(false);
        });
    }
  }, [location.state?.entities, tenant, datasource]);

  // Fetch full schema for rule creator
  useEffect(() => {
    const loadSchema = async () => {
      if (!tenant || !datasource) return;
      try {
        const schema = await fetchEntitySchema(tenant.id, datasource.id || datasource.alpha_tenant_instance_id);
        setRawSchema(schema);
        setAvailableEntities(Object.keys(schema).sort());
        
        // If entities state wasn't provided via location, transform and set it
        if (!entities) {
          setEntities(transformEntitySchema(schema));
        }
      } catch (error) {
        devError('Error fetching entity schema:', error);
      }
    };
    loadSchema();
  }, [tenant?.id, datasource?.id, entities]);

  const handleAddRule = () => {
    setEditingRule(null);
    setValidationRuleCreatorOpen(true);
  };

  const handleEditRule = (rule: AnnotatedValidationRule) => {
    setEditingRule(rule);
    setValidationRuleCreatorOpen(true);
  };

  const handleSaveRule = async (rule: any) => {
    // Refresh rules after save
    await fetchValidationRules();
    notification.success(editingRule ? 'Rule updated successfully' : 'Rule created successfully');
    setValidationRuleCreatorOpen(false);
    setEditingRule(null);
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center gap-3 p-12 min-h-screen bg-[#f5f7f8] dark:bg-[#101922]">
        <div className="w-8 h-8 border-4 border-slate-200 dark:border-slate-700 border-t-blue-500 dark:border-t-blue-400 rounded-full animate-spin"></div>
        <span className="text-slate-700 dark:text-slate-300 font-medium">Loading...</span>
      </div>
    );
  }

  if (!entityKey || !entities || !entities[entityKey]) {
    return (
      <div className="p-12 min-h-screen bg-[#f5f7f8] dark:bg-[#101922]">
        <div className="bg-white dark:bg-slate-900 rounded-lg p-8 text-center max-w-md mx-auto shadow-sm border border-gray-200 dark:border-gray-800">
          <div className="text-4xl mb-4">📦</div>
          <h3 className="text-lg font-bold text-slate-900 dark:text-slate-50 mb-2">Entity not found</h3>
          <div className="text-sm text-slate-600 dark:text-slate-400 space-y-1 mb-6">
            <p>Data not available or entity key is missing.</p>
            {!hasTenantScope && <p>Please select a tenant and datasource first.</p>}
            {entityKey && <p className="text-xs font-mono mt-2">Key: {entityKey}</p>}
            {entities && <p className="text-xs mt-2">Available: {Object.keys(entities).slice(0, 3).join(', ')}</p>}
          </div>
          <button
            onClick={() => navigate('/admin/entity-manager')}
            className="inline-flex items-center gap-2 px-4 py-2 bg-slate-900 dark:bg-slate-100 text-white dark:text-slate-900 rounded-lg font-medium hover:bg-slate-800 dark:hover:bg-slate-200 transition-colors"
          >
            <ArrowLeft size={16} />
            Back to Entity Manager
          </button>
        </div>
      </div>
    );
  }

  const entity = entities[entityKey];

  const handleEntityUpdate = (updatedEntity: Entity) => {
    // This update is local to this page. The parent page will need to be refreshed
    // or state needs to be managed globally for changes to persist across navigation.
    setEntities((prev: Entities | null) => (prev ? { ...prev, [entityKey]: updatedEntity } : null));
  };

  const tabs = [
    {
      key: 'entity',
      label: 'Details',
      children: (
        <EntityDetailsEditor
          entityKey={entityKey}
          entity={entity}
          entities={entities}
          datasourceId={datasource?.id || datasource?.alpha_tenant_instance_id}
          onEntityUpdate={handleEntityUpdate}
          validationRules={validationRules}
        />
      ),
    },
    {
      key: 'relationships',
      label: 'Relationships',
      children:
        tenant && datasource ? (
          <RelationshipsTab
            tenantId={tenant.id}
            datasourceId={datasource.id || datasource.alpha_tenant_instance_id}
            entityId={entityKey}
            entityName={entity.businessName || entity.name}
            suggestions={semanticLayer.relationshipSuggestions}
            suggestionsLoading={semanticLayer.suggestionsLoading}
            suggestionsError={semanticLayer.suggestionsError}
            onApplySuggestion={async (suggestion) => {
              await semanticLayer.applyRelationshipSuggestion(suggestion);
            }}
          />
        ) : (
          <div className="p-6 text-center text-slate-500 dark:text-slate-400">
            Please select a tenant and datasource to view relationships
          </div>
        ),
    },
    {
      key: 'validations',
      label: 'Validations',
      children: (
        <ValidationsTab
          entity={entity}
          rules={validationRules}
          onAddRule={handleAddRule}
          onEditRule={handleEditRule}
          onCrossEntitySave={(condition) => {
            devLog('Cross-entity condition saved:', condition);
            // TODO: Persist to backend
          }}
        />
      ),
    },
    {
      key: 'semantic-models',
      label: 'Semantic Models',
      children: (
        <SemanticAssetsTab
          semanticAssets={semanticLayer.semanticAssets}
          isLoading={semanticLayer.assetsLoading || semanticLayer.modelGenerationLoading}
          error={semanticLayer.modelError}
          onGenerateCoreModel={async () => { await semanticLayer.generateCoreModel(); }}
          onCreateCustomModel={async (name) => { await semanticLayer.createCustomModel(name); }}
          onGenerateCoreView={async () => { await semanticLayer.generateCoreView(); }}
          onCreateCustomView={async (name) => { await semanticLayer.createCustomView(name); }}
          businessEntityName={entityKey || 'Entity'}
        />
      ),
    },
  ];

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'background.default', pb: 8 }}>
      {/* Header / Navigation */}
      <Box sx={{ bgcolor: 'background.paper', borderBottom: 1, borderColor: 'divider', px: 3, py: 1.5, position: 'sticky', top: 0, zIndex: 1100 }}>
         <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 3 }}>
               <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
                  <Box sx={{ 
                      width: 32, height: 32, 
                      bgcolor: 'primary.light', 
                      color: 'primary.main',
                      borderRadius: 1, 
                      display: 'flex', alignItems: 'center', justifyContent: 'center' 
                  }}>
                     <span className="material-symbols-outlined">dataset</span>
                  </Box>
                  <Typography variant="subtitle1" fontWeight="bold">Business Object Manager</Typography>
               </Box>
               
               {/* Search - Hidden on mobile */}
               <Box sx={{ display: { xs: 'none', md: 'flex' }, alignItems: 'center', bgcolor: 'action.hover', borderRadius: 1, px: 1.5, py: 0.5, gap: 1, width: 250 }}>
                  <span className="material-symbols-outlined" style={{ fontSize: 20, color: 'text.secondary' }}>search</span>
                  <input 
                      style={{ 
                          border: 'none', outline: 'none', background: 'transparent', 
                          fontSize: '0.875rem', width: '100%', color: 'inherit' 
                      }} 
                      placeholder="Search objects..." 
                  />
               </Box>
            </Box>
            
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 3 }}>
                <Box component="nav" sx={{ display: { xs: 'none', md: 'flex' }, gap: 3 }}>
                   <Button color="inherit" onClick={() => navigate('/dashboard')}>Dashboard</Button>
                   <Button color="primary" onClick={() => navigate('/entity-config')}>Objects</Button>
                   <Button color="inherit">Rules</Button>
                   <Button color="inherit">Settings</Button>
                </Box>
                 
               <Box sx={{ width: 32, height: 32, borderRadius: '50%', bgcolor: 'action.selected' }} />
            </Box>
         </Box>
      </Box>
      
      {/* Main Content */}
      <Container maxWidth="xl" sx={{ py: 3 }}>
        {/* Breadcrumbs */}
        <Box sx={{ mb: 2 }}>
            <div className="flex items-center gap-2 text-sm text-gray-500">
                 <span className="cursor-pointer hover:underline" onClick={() => navigate('/')}>Home</span>
                 <span className="material-symbols-outlined text-[16px]">chevron_right</span>
                 <span className="cursor-pointer hover:underline" onClick={() => navigate('/entity-config')}>Business Objects</span>
                 <span className="material-symbols-outlined text-[16px]">chevron_right</span>
                 <span className="font-medium text-gray-900 dark:text-gray-100">{entity.businessName || entity.name}</span>
            </div>
        </Box>

        {/* Page Title & Actions */}
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 3, flexWrap: 'wrap', gap: 2 }}>
            <Box>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 0.5 }}>
                    <Typography variant="h4" fontWeight="900" component="h1">
                        {entity.businessName || entity.name}
                    </Typography>
                    <Chip 
                        label={entity.isCore ? 'Core' : 'Active'} 
                        color={entity.isCore ? 'primary' : 'success'} 
                        variant="outlined" 
                        size="small" 
                    />
                </Box>
                <Typography variant="body1" color="text.secondary">
                    {entity.description || "Core business object definition."} <Typography component="span" variant="caption" sx={{ fontFamily: 'monospace' }}>({entity.key})</Typography>
                </Typography>
            </Box>
            
            <Box sx={{ display: 'flex', gap: 2 }}>
                <Button variant="outlined" startIcon={<span className="material-symbols-outlined">edit</span>}>
                    Edit Object
                </Button>
                <Button variant="contained" startIcon={<span className="material-symbols-outlined">add</span>}>
                    Create Child
                </Button>
            </Box>
        </Box>

        {/* Tabs */}
        <Box sx={{ mb: 3 }}>
             <div className="border-b border-gray-200 dark:border-gray-700">
                 <nav className="-mb-px flex space-x-8" aria-label="Tabs">
                    {tabs.map((tab) => {
                       const isActive = activeTab === tab.key;
                       let iconName = 'article';
                       if (tab.key === 'entity') iconName = 'account_tree';
                       if (tab.key === 'relationships') iconName = 'link';
                       if (tab.key === 'validations') iconName = 'verified';
                       if (tab.key === 'semantic-models') iconName = 'schema';

                       return (
                        <button
                          key={tab.key}
                          onClick={() => setActiveTab(tab.key)}
                          className={`
                            whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm flex items-center gap-2
                            ${isActive 
                               ? 'border-blue-500 text-blue-600' 
                               : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'}
                          `}
                        >
                          <span className="material-symbols-outlined text-[20px]">{iconName}</span>
                          {tab.label}
                          {tab.key === 'validations' && validationRules.length > 0 && (
                             <span className="bg-gray-100 text-gray-900 py-0.5 px-2.5 rounded-full text-xs ml-2">
                                {validationRules.length}
                             </span>
                          )}
                        </button>
                       );
                    })}
                 </nav>
             </div>
        </Box>

        {/* Tab Content */}
        <Box sx={{ minHeight: 600 }}>
            {tabs.find(t => t.key === activeTab)?.children}
        </Box>

      </Container>

      <ValidationRuleCreator
        isOpen={validationRuleCreatorOpen}
        onClose={() => {
          setValidationRuleCreatorOpen(false);
          setEditingRule(null);
        }}
        onSave={handleSaveRule}
        tenantId={tenant?.id || ''}
        datasourceId={datasource?.id || datasource?.alpha_tenant_instance_id || ''}
        availableEntities={availableEntities}
        entitySchema={rawSchema}
        editingRule={editingRule as any}
        defaultTargetEntity={entityKey}
      />
    </Box>
  );
}