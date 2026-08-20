import React, { useState, useMemo } from 'react';
import {
  Layers,
  Sparkles,
  Database,
  ArrowRight,
  CheckCircle2,
  AlertTriangle,
  XCircle,
  HelpCircle,
  Plus,
  Trash2,
  Edit3,
  Star,
  Link2,
  ShieldCheck,
  ChevronDown,
  ChevronRight,
  ExternalLink,
  Code2,
  Cpu,
  Eye,
  Key,
  Hash,
  Compass,
  Check,
  X,
  RefreshCw,
  Search,
  Filter,
  Save,
  Send,
  Zap,
} from 'lucide-react';

// ==========================================
// 1. TYPE DEFINITIONS & DATA CONTRACTS
// ==========================================

export type BOType = 'ENTITY' | 'FACT' | 'DIMENSION' | 'BRIDGE' | 'REFERENCE';
export type FieldRole = 'KEY' | 'DIMENSION' | 'MEASURE' | 'ATTRIBUTE' | 'CALCULATED';
export type BindingRequirement = 'REQUIRED' | 'OPTIONAL' | 'BACKEND_SPECIFIC' | 'CALCULATED' | 'INTERNAL';
export type BindingStatus = 'RESOLVED' | 'PARTIALLY_RESOLVED' | 'UNRESOLVED' | 'NOT_APPLICABLE';
export type EligibilitySource = 'DIRECT' | 'RELATED' | 'CALCULATED' | 'MANUAL';
export type TransformationType = 'NONE' | 'SQL_EXPRESSION' | 'NORMALIZE' | 'FUNCTION_LOOKUP' | 'JSON_PATH';

export interface ColumnMappingCandidate {
  columnNodeId: string;
  columnName: string;
  tableName: string;
  sourceType: 'DIRECT' | 'RELATED';
  isPrimary: boolean;
}

export interface ScopedSemanticTerm {
  termNodeId: string;
  termKey: string;
  termName: string;
  dataType: string;
  defaultRole: FieldRole;
  eligibilitySource: EligibilitySource;
  mappings: ColumnMappingCandidate[];
  requiredInputTerms?: string[];
  description?: string;
  bloombergMnemonic?: string;
}

export interface BOFieldOverride {
  fieldRole: FieldRole;
  aggregationType: 'NONE' | 'SUM' | 'AVG' | 'COUNT' | 'MIN' | 'MAX' | 'CUSTOM';
  bindingRequirement: BindingRequirement;
  nullable: boolean;
  isExposed: boolean;
  transformationType: TransformationType;
  transformationSql?: string;
  overrideReason?: string;
  inheritsDefaults: boolean;
}

export interface BOFieldDraft {
  fieldId: string;
  termNodeId: string;
  termKey: string;
  termName: string;
  dataType: string;
  eligibilitySource: EligibilitySource;
  overrides: BOFieldOverride;
  selectedColumnMapping?: Record<string, string>; // backendId -> columnNodeId
}

export interface BORelationshipDraft {
  relId: string;
  relKey: string;
  toBoKey: string;
  toBoName: string;
  cardinality: '1:1' | '1:N' | 'M:1' | 'M:N';
  joinType: 'INNER' | 'LEFT' | 'FULL' | 'CROSS';
  basisTermKey: string;
  joinConditionSql: string;
}

export interface BOBindingDraft {
  backendId: string;
  backendName: string;
  engineType: 'POSTGRES' | 'STARROCKS' | 'ICEBERG' | 'SNOWFLAKE' | 'REST_API';
  drivingNodeId: string;
  drivingTableName: string;
  isDefault: boolean;
  pkColumnName: string;
  suggestedBkTerm: string;
}

export interface BusinessObjectDraft {
  boKey: string;
  boName: string;
  description: string;
  boType: BOType;
  classificationPath: string;
  businessKeyTerm: string;
  semanticIdTerm: string;
  grainTerm: string;
  isCore: boolean;
  status: 'DRAFT' | 'IN_REVIEW' | 'APPROVED' | 'PUBLISHED';
}

// ==========================================
// 2. MOCK DATA & PRELOADED CATALOGS
// ==========================================

const MOCK_BACKENDS: Array<{ id: string; name: string; engine: BOBindingDraft['engineType']; defaultTable: string }> = [
  { id: 'b-pg-01', name: 'PostgreSQL OLTP (Northwind)', engine: 'POSTGRES', defaultTable: 'public.customers' },
  { id: 'b-sr-01', name: 'StarRocks Hot (DW Lake)', engine: 'STARROCKS', defaultTable: 'analytics.fact_customers_daily' },
  { id: 'b-sf-01', name: 'Snowflake CRM (Contacts)', engine: 'SNOWFLAKE', defaultTable: 'crm.contacts' },
  { id: 'b-ice-01', name: 'Apache Iceberg Cold Store', engine: 'ICEBERG', defaultTable: 'iceberg.t_99e.orders_archive' },
];

const MOCK_DISCOVERED_TERMS: ScopedSemanticTerm[] = [
  {
    termNodeId: 't-101',
    termKey: 'customer_identifier',
    termName: 'Customer Identifier',
    dataType: 'UUID',
    defaultRole: 'KEY',
    eligibilitySource: 'DIRECT',
    mappings: [
      { columnNodeId: 'c-1', columnName: 'customer_id', tableName: 'customers', sourceType: 'DIRECT', isPrimary: true },
      { columnNodeId: 'c-2', columnName: 'client_ref_id', tableName: 'orders', sourceType: 'RELATED', isPrimary: false },
    ],
  },
  {
    termNodeId: 't-102',
    termKey: 'customer_bk',
    termName: 'Customer Natural Key',
    dataType: 'VARCHAR(64)',
    defaultRole: 'KEY',
    eligibilitySource: 'DIRECT',
    mappings: [{ columnNodeId: 'c-3', columnName: 'customer_code', tableName: 'customers', sourceType: 'DIRECT', isPrimary: true }],
  },
  {
    termNodeId: 't-103',
    termKey: 'customer_sid',
    termName: 'Customer Semantic ID',
    dataType: 'UUID',
    defaultRole: 'KEY',
    eligibilitySource: 'DIRECT',
    mappings: [{ columnNodeId: 'c-4', columnName: 'customer_id', tableName: 'customers', sourceType: 'DIRECT', isPrimary: true }],
  },
  {
    termNodeId: 't-104',
    termKey: 'company_name',
    termName: 'Company Name',
    dataType: 'VARCHAR(255)',
    defaultRole: 'DIMENSION',
    eligibilitySource: 'DIRECT',
    bloombergMnemonic: 'ISSUER',
    mappings: [{ columnNodeId: 'c-5', columnName: 'company_name', tableName: 'customers', sourceType: 'DIRECT', isPrimary: true }],
  },
  {
    termNodeId: 't-105',
    termKey: 'country_code',
    termName: 'Country Code (ISO-2)',
    dataType: 'CHAR(2)',
    defaultRole: 'DIMENSION',
    eligibilitySource: 'DIRECT',
    bloombergMnemonic: 'CNTRY_ISSUE_ISO',
    mappings: [{ columnNodeId: 'c-6', columnName: 'country', tableName: 'customers', sourceType: 'DIRECT', isPrimary: true }],
  },
  {
    termNodeId: 't-106',
    termKey: 'order_date',
    termName: 'Order Placement Date',
    dataType: 'TIMESTAMPTZ',
    defaultRole: 'DIMENSION',
    eligibilitySource: 'RELATED',
    mappings: [{ columnNodeId: 'c-7', columnName: 'placed_at', tableName: 'orders', sourceType: 'RELATED', isPrimary: true }],
  },
  {
    termNodeId: 't-107',
    termKey: 'total_revenue',
    termName: 'Total Customer Lifetime Revenue',
    dataType: 'NUMERIC(18,4)',
    defaultRole: 'MEASURE',
    eligibilitySource: 'CALCULATED',
    requiredInputTerms: ['unit_price', 'quantity', 'discount'],
    mappings: [],
  },
  {
    termNodeId: 't-108',
    termKey: 'risk_rating_score',
    termName: 'External Credit Risk Score',
    dataType: 'NUMERIC(5,2)',
    defaultRole: 'ATTRIBUTE',
    eligibilitySource: 'MANUAL',
    mappings: [],
  },
];

// ==========================================
// 3. MAIN COMPONENT
// ==========================================

export const BusinessObjectStudio: React.FC = () => {
  // --- Section 1: Business Object Definition State ---
  const [boDraft, setBoDraft] = useState<BusinessObjectDraft>({
    boKey: 'customer',
    boName: 'Customer Entity',
    description: 'Master operational and analytical customer model binding CRM, Orders, and Lakehouse.',
    boType: 'ENTITY',
    classificationPath: 'Enterprise > CRM > Core Party',
    businessKeyTerm: 'customer_bk',
    semanticIdTerm: 'customer_sid',
    grainTerm: 'customer_bk',
    isCore: true,
    status: 'DRAFT',
  });

  // --- Section 2: Bindings State ---
  const [bindings, setBindings] = useState<BOBindingDraft[]>([
    {
      backendId: 'b-pg-01',
      backendName: 'PostgreSQL OLTP (Northwind)',
      engineType: 'POSTGRES',
      drivingNodeId: 'n-pg-cust',
      drivingTableName: 'public.customers',
      isDefault: true,
      pkColumnName: 'customer_id',
      suggestedBkTerm: 'customer_bk',
    },
    {
      backendId: 'b-sf-01',
      backendName: 'Snowflake CRM (Contacts)',
      engineType: 'SNOWFLAKE',
      drivingNodeId: 'n-sf-cont',
      drivingTableName: 'crm.contacts',
      isDefault: false,
      pkColumnName: 'contact_id',
      suggestedBkTerm: 'customer_bk',
    },
  ]);

  // --- Section 3: Selected Fields State ---
  const [selectedFields, setSelectedFields] = useState<BOFieldDraft[]>([
    {
      fieldId: 'fld-1',
      termNodeId: 't-101',
      termKey: 'customer_identifier',
      termName: 'Customer Identifier',
      dataType: 'UUID',
      eligibilitySource: 'DIRECT',
      overrides: {
        fieldRole: 'KEY',
        aggregationType: 'NONE',
        bindingRequirement: 'REQUIRED',
        nullable: false,
        isExposed: true,
        transformationType: 'NONE',
        inheritsDefaults: true,
      },
      selectedColumnMapping: { 'b-pg-01': 'c-1', 'b-sf-01': 'c-10' },
    },
    {
      fieldId: 'fld-2',
      termNodeId: 't-102',
      termKey: 'customer_bk',
      termName: 'Customer Natural Key',
      dataType: 'VARCHAR(64)',
      eligibilitySource: 'DIRECT',
      overrides: {
        fieldRole: 'KEY',
        aggregationType: 'NONE',
        bindingRequirement: 'REQUIRED',
        nullable: false,
        isExposed: true,
        transformationType: 'NONE',
        inheritsDefaults: true,
      },
      selectedColumnMapping: { 'b-pg-01': 'c-3', 'b-sf-01': 'c-11' },
    },
    {
      fieldId: 'fld-3',
      termNodeId: 't-104',
      termKey: 'company_name',
      termName: 'Company Name',
      dataType: 'VARCHAR(255)',
      eligibilitySource: 'DIRECT',
      overrides: {
        fieldRole: 'DIMENSION',
        aggregationType: 'NONE',
        bindingRequirement: 'REQUIRED',
        nullable: false,
        isExposed: true,
        transformationType: 'NONE',
        inheritsDefaults: true,
      },
      selectedColumnMapping: { 'b-pg-01': 'c-5', 'b-sf-01': 'c-12' },
    },
    {
      fieldId: 'fld-4',
      termNodeId: 't-105',
      termKey: 'country_code',
      termName: 'Country Code (ISO-2)',
      dataType: 'CHAR(2)',
      eligibilitySource: 'DIRECT',
      overrides: {
        fieldRole: 'DIMENSION',
        aggregationType: 'NONE',
        bindingRequirement: 'REQUIRED',
        nullable: false,
        isExposed: true,
        transformationType: 'NORMALIZE',
        transformationSql: 'lookup_iso2(country)',
        overrideReason: 'Standardize ISO2 standard representation',
        inheritsDefaults: false,
      },
      selectedColumnMapping: { 'b-pg-01': 'c-6', 'b-sf-01': 'c-13' },
    },
    {
      fieldId: 'fld-5',
      termNodeId: 't-107',
      termKey: 'total_revenue',
      termName: 'Total Customer Lifetime Revenue',
      dataType: 'NUMERIC(18,4)',
      eligibilitySource: 'CALCULATED',
      overrides: {
        fieldRole: 'MEASURE',
        aggregationType: 'SUM',
        bindingRequirement: 'CALCULATED',
        nullable: true,
        isExposed: true,
        transformationType: 'SQL_EXPRESSION',
        transformationSql: 'SUM(order_total)',
        inheritsDefaults: true,
      },
      selectedColumnMapping: {},
    },
  ]);

  // --- Section 4: Relationships State ---
  const [relationships, setRelationships] = useState<BORelationshipDraft[]>([
    {
      relId: 'rel-1',
      relKey: 'customer_to_orders',
      toBoKey: 'order',
      toBoName: 'Customer Orders',
      cardinality: '1:N',
      joinType: 'LEFT',
      basisTermKey: 'customer_identifier',
      joinConditionSql: 'customers.customer_id = orders.customer_id',
    },
  ]);

  // --- UI Filter & Drawer State ---
  const [termPickerFilter, setTermPickerFilter] = useState<'ALL' | 'DIRECT' | 'RELATED' | 'CALCULATED' | 'MANUAL'>('ALL');
  const [termSearchQuery, setTermSearchQuery] = useState('');
  const [activeEditingField, setActiveEditingField] = useState<BOFieldDraft | null>(null);
  const [resolvingMultiMappingTerm, setResolvingMultiMappingTerm] = useState<ScopedSemanticTerm | null>(null);
  const [showLineagePreview, setShowLineagePreview] = useState(false);
  const [isPublishing, setIsPublishing] = useState(false);

  // ==========================================
  // 4. COMPUTED VALIDATION & PUBLISH GATE
  // ==========================================

  const validationSummary = useMemo(() => {
    const hasIdentity = Boolean(boDraft.businessKeyTerm && boDraft.semanticIdTerm && boDraft.grainTerm);
    const requiredFields = selectedFields.filter((f) => f.overrides.bindingRequirement === 'REQUIRED');
    
    // Check if required fields are mapped across all active bindings
    const unresolvedRequired = requiredFields.filter((f) => {
      if (f.eligibilitySource === 'CALCULATED') return false;
      return bindings.some((b) => !f.selectedColumnMapping || !f.selectedColumnMapping[b.backendId]);
    });

    const isReadyToPublish = hasIdentity && unresolvedRequired.length === 0 && bindings.length > 0;

    return {
      hasIdentity,
      totalFields: selectedFields.length,
      requiredCount: requiredFields.length,
      unresolvedCount: unresolvedRequired.length,
      bindingsConfigured: bindings.length,
      relationshipsCount: relationships.length,
      isReadyToPublish,
    };
  }, [boDraft, selectedFields, bindings, relationships]);

  // ==========================================
  // 5. HANDLERS & MUTATIONS
  // ==========================================

  const handleToggleTerm = (term: ScopedSemanticTerm) => {
    const existingIndex = selectedFields.findIndex((f) => f.termNodeId === term.termNodeId);
    if (existingIndex >= 0) {
      setSelectedFields(selectedFields.filter((_, i) => i !== existingIndex));
      if (activeEditingField?.termNodeId === term.termNodeId) {
        setActiveEditingField(null);
      }
    } else {
      // If multiple column mappings exist, open resolver modal
      if (term.mappings.length > 1) {
        setResolvingMultiMappingTerm(term);
        return;
      }

      // Auto-assign primary column mapping
      const initialMapping: Record<string, string> = {};
      if (term.mappings.length === 1) {
        bindings.forEach((b) => {
          initialMapping[b.backendId] = term.mappings[0].columnNodeId;
        });
      }

      const newField: BOFieldDraft = {
        fieldId: `fld-${Date.now()}`,
        termNodeId: term.termNodeId,
        termKey: term.termKey,
        termName: term.termName,
        dataType: term.dataType,
        eligibilitySource: term.eligibilitySource,
        overrides: {
          fieldRole: term.defaultRole,
          aggregationType: term.defaultRole === 'MEASURE' ? 'SUM' : 'NONE',
          bindingRequirement:
            term.defaultRole === 'KEY'
              ? 'REQUIRED'
              : term.eligibilitySource === 'CALCULATED'
              ? 'CALCULATED'
              : 'REQUIRED',
          nullable: term.defaultRole !== 'KEY',
          isExposed: true,
          transformationType: 'NONE',
          inheritsDefaults: true,
        },
        selectedColumnMapping: initialMapping,
      };
      setSelectedFields([...selectedFields, newField]);
    }
  };

  const handleSetDefaultBinding = (backendId: string) => {
    setBindings(
      bindings.map((b) => ({
        ...b,
        isDefault: b.backendId === backendId,
      }))
    );
  };

  const handlePublish = async () => {
    setIsPublishing(true);
    // Simulate commit to backend API: POST /api/business-objects/save
    setTimeout(() => {
      setIsPublishing(false);
      setBoDraft((prev) => ({ ...prev, status: 'PUBLISHED' }));
    }, 900);
  };

  // Filtered term list for picker
  const filteredDiscoveredTerms = useMemo(() => {
    return MOCK_DISCOVERED_TERMS.filter((t) => {
      const matchesFilter = termPickerFilter === 'ALL' || t.eligibilitySource === termPickerFilter;
      const matchesSearch =
        t.termName.toLowerCase().includes(termSearchQuery.toLowerCase()) ||
        t.termKey.toLowerCase().includes(termSearchQuery.toLowerCase()) ||
        (t.bloombergMnemonic && t.bloombergMnemonic.toLowerCase().includes(termSearchQuery.toLowerCase()));
      return matchesFilter && matchesSearch;
    });
  }, [termPickerFilter, termSearchQuery]);

  return (
    <div className="min-h-screen bg-[#070D19] text-slate-100 p-4 md:p-6 font-sans">
      {/* ────────────────────────────────────────────────────────────────────────── */}
      {/* COMMAND BAR & HEADER */}
      {/* ────────────────────────────────────────────────────────────────────────── */}
      <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-4 pb-6 border-b border-slate-800/80">
        <div>
          <div className="flex items-center gap-3">
            <div className="p-2.5 rounded-xl bg-teal-500/10 border border-teal-500/30 text-teal-400 shadow-[0_0_15px_rgba(20,184,166,0.15)]">
              <Layers className="w-6 h-6" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-xl font-bold text-white tracking-tight">Business Object Studio</h1>
                <span className="text-xs px-2 py-0.5 rounded-full font-mono font-semibold bg-teal-500/20 text-teal-300 border border-teal-500/30">
                  SINGLE-SCREEN
                </span>
                <span
                  className={`text-xs px-2.5 py-0.5 rounded-full font-semibold border ${
                    boDraft.status === 'PUBLISHED'
                      ? 'bg-emerald-500/20 text-emerald-300 border-emerald-500/40'
                      : 'bg-amber-500/20 text-amber-300 border-amber-500/40'
                  }`}
                >
                  {boDraft.status}
                </span>
              </div>
              <p className="text-xs text-slate-400 mt-0.5">
                Declarative semantic entity contract with binding-aware auto-discovery and multi-backend coverage.
              </p>
            </div>
          </div>
        </div>

        <div className="flex items-center gap-2.5 self-end lg:self-center">
          <button
            onClick={() => setShowLineagePreview(!showLineagePreview)}
            className="flex items-center gap-1.5 px-3 py-2 text-xs font-medium text-slate-300 bg-slate-800/80 hover:bg-slate-700 rounded-lg border border-slate-700 transition"
          >
            <Compass className="w-4 h-4 text-teal-400" />
            {showLineagePreview ? 'Hide Graph Preview' : 'Preview Lineage'}
          </button>
          <button className="flex items-center gap-1.5 px-3 py-2 text-xs font-medium text-slate-300 bg-slate-800/80 hover:bg-slate-700 rounded-lg border border-slate-700 transition">
            <Save className="w-4 h-4 text-slate-400" /> Save Draft
          </button>
          <button
            onClick={handlePublish}
            disabled={!validationSummary.isReadyToPublish || isPublishing}
            className="flex items-center gap-2 px-4 py-2 bg-gradient-to-r from-teal-500 to-emerald-600 hover:from-teal-600 hover:to-emerald-700 disabled:opacity-40 disabled:cursor-not-allowed text-white text-xs font-bold rounded-lg shadow-[0_0_20px_rgba(20,184,166,0.3)] transition"
          >
            {isPublishing ? <RefreshCw className="w-4 h-4 animate-spin" /> : <Send className="w-4 h-4" />}
            Publish Business Object
          </button>
        </div>
      </div>

      {/* ────────────────────────────────────────────────────────────────────────── */}
      {/* MAIN STUDIO GRID */}
      {/* ────────────────────────────────────────────────────────────────────────── */}
      <div className="grid grid-cols-1 xl:grid-cols-12 gap-6 mt-6">
        {/* LEFT COLUMN: DEFINITION & SCOPED TERM PICKER (5 Cols) */}
        <div className="xl:col-span-5 space-y-6">
          {/* SECTION 1: BUSINESS OBJECT DEFINITION */}
          <div className="bg-[#0B1528] rounded-xl border border-slate-800 p-5 shadow-lg relative overflow-hidden">
            <div className="absolute top-0 left-0 right-0 h-1 bg-gradient-to-r from-teal-500 via-emerald-500 to-amber-500" />
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-sm font-bold uppercase tracking-wider text-slate-300 flex items-center gap-2">
                <Key className="w-4 h-4 text-teal-400" /> 1. Semantic Shell & Identity Triple
              </h2>
              <span className="text-[10px] font-mono text-slate-400 bg-slate-900 px-2 py-0.5 rounded border border-slate-800">
                Rule 1 & 2 Compliant
              </span>
            </div>

            <div className="space-y-3.5">
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-[11px] font-semibold text-slate-400 uppercase mb-1">BO Name</label>
                  <input
                    type="text"
                    value={boDraft.boName}
                    onChange={(e) => setBoDraft({ ...boDraft, boName: e.target.value })}
                    className="w-full bg-[#070E1B] border border-slate-700/80 rounded-lg px-3 py-1.5 text-xs text-white focus:border-teal-500 focus:outline-none"
                  />
                </div>
                <div>
                  <label className="block text-[11px] font-semibold text-slate-400 uppercase mb-1">BO Key (Machine)</label>
                  <input
                    type="text"
                    value={boDraft.boKey}
                    onChange={(e) => setBoDraft({ ...boDraft, boKey: e.target.value })}
                    className="w-full bg-[#070E1B] border border-slate-700/80 rounded-lg px-3 py-1.5 text-xs font-mono text-teal-300 focus:border-teal-500 focus:outline-none"
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-[11px] font-semibold text-slate-400 uppercase mb-1">Entity Type</label>
                  <select
                    value={boDraft.boType}
                    onChange={(e) => setBoDraft({ ...boDraft, boType: e.target.value as BOType })}
                    className="w-full bg-[#070E1B] border border-slate-700/80 rounded-lg px-3 py-1.5 text-xs text-slate-200 focus:border-teal-500 focus:outline-none"
                  >
                    <option value="ENTITY">ENTITY (Master Domain)</option>
                    <option value="FACT">FACT (Transactional Event)</option>
                    <option value="DIMENSION">DIMENSION (Attribute Group)</option>
                    <option value="BRIDGE">BRIDGE (Associative Map)</option>
                  </select>
                </div>
                <div>
                  <label className="block text-[11px] font-semibold text-slate-400 uppercase mb-1">Level 3 Classification</label>
                  <input
                    type="text"
                    value={boDraft.classificationPath}
                    onChange={(e) => setBoDraft({ ...boDraft, classificationPath: e.target.value })}
                    className="w-full bg-[#070E1B] border border-slate-700/80 rounded-lg px-3 py-1.5 text-xs text-slate-300 focus:border-teal-500 focus:outline-none"
                  />
                </div>
              </div>

              {/* IDENTITY TRIPLE MATRIX */}
              <div className="p-3 bg-[#070E1B] rounded-lg border border-slate-800 space-y-2.5">
                <div className="text-[11px] font-bold text-teal-400 uppercase tracking-wide flex items-center justify-between">
                  <span>Identity Resolution Triple</span>
                  <ShieldCheck className="w-3.5 h-3.5 text-emerald-400" />
                </div>
                <div className="grid grid-cols-3 gap-2 text-xs">
                  <div>
                    <span className="text-[10px] text-slate-500 block">Business Key (BK)</span>
                    <span className="font-mono text-slate-300 text-[11px] font-semibold">
                      {boDraft.businessKeyTerm}
                    </span>
                  </div>
                  <div>
                    <span className="text-[10px] text-slate-500 block">Semantic ID (SID)</span>
                    <span className="font-mono text-slate-300 text-[11px] font-semibold">
                      {boDraft.semanticIdTerm}
                    </span>
                  </div>
                  <div>
                    <span className="text-[10px] text-slate-500 block">Grain Anchor</span>
                    <span className="font-mono text-slate-300 text-[11px] font-semibold">
                      {boDraft.grainTerm}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* SECTION 2: SCOPED SEMANTIC TERM PICKER */}
          <div className="bg-[#0B1528] rounded-xl border border-slate-800 p-5 shadow-lg">
            <div className="flex items-center justify-between mb-3">
              <div>
                <h2 className="text-sm font-bold uppercase tracking-wider text-slate-300 flex items-center gap-2">
                  <Database className="w-4 h-4 text-emerald-400" /> 2. Scoped Term Picker
                </h2>
                <p className="text-[11px] text-slate-400 mt-0.5">
                  Filtered by active driving tables ({bindings.length} bindings attached).
                </p>
              </div>
              <span className="text-[10px] px-2 py-0.5 bg-emerald-500/10 text-emerald-300 border border-emerald-500/30 rounded font-mono">
                {filteredDiscoveredTerms.length} Eligible
              </span>
            </div>

            {/* SEARCH & SCOPE TABS */}
            <div className="space-y-2.5 mb-3.5">
              <div className="relative">
                <Search className="w-3.5 h-3.5 absolute left-3 top-2.5 text-slate-400" />
                <input
                  type="text"
                  placeholder="Search semantic terms or Bloomberg tags..."
                  value={termSearchQuery}
                  onChange={(e) => setTermSearchQuery(e.target.value)}
                  className="w-full pl-8 pr-3 py-1.5 bg-[#070E1B] border border-slate-700/80 rounded-lg text-xs text-white focus:outline-none focus:border-teal-500"
                />
              </div>

              <div className="flex items-center gap-1 bg-[#070E1B] p-1 rounded-lg border border-slate-800 text-[11px]">
                {(['ALL', 'DIRECT', 'RELATED', 'CALCULATED', 'MANUAL'] as const).map((filter) => (
                  <button
                    key={filter}
                    onClick={() => setTermPickerFilter(filter)}
                    className={`flex-1 py-1 rounded font-medium transition ${
                      termPickerFilter === filter
                        ? 'bg-teal-500/20 text-teal-300 border border-teal-500/40 shadow-sm'
                        : 'text-slate-400 hover:text-slate-200'
                    }`}
                  >
                    {filter}
                  </button>
                ))}
              </div>
            </div>

            {/* TERM ROWS LIST */}
            <div className="space-y-2 max-h-[360px] overflow-y-auto pr-1">
              {filteredDiscoveredTerms.map((term) => {
                const isSelected = selectedFields.some((f) => f.termNodeId === term.termNodeId);
                return (
                  <div
                    key={term.termNodeId}
                    onClick={() => handleToggleTerm(term)}
                    className={`p-3 rounded-lg border transition-all cursor-pointer flex items-center justify-between ${
                      isSelected
                        ? 'bg-teal-950/30 border-teal-500/60 shadow-[inset_0_0_12px_rgba(20,184,166,0.1)]'
                        : 'bg-[#070E1B] border-slate-800/90 hover:border-slate-700'
                    }`}
                  >
                    <div className="space-y-1">
                      <div className="flex items-center gap-2">
                        <input
                          type="checkbox"
                          checked={isSelected}
                          onChange={() => {}} // Handled by parent div
                          className="rounded border-slate-700 text-teal-500 focus:ring-0 pointer-events-none"
                        />
                        <span className="font-semibold text-xs text-slate-100">{term.termName}</span>
                        <span className="text-[9px] font-mono px-1.5 py-0.2 bg-slate-800 text-slate-300 rounded">
                          {term.dataType}
                        </span>
                        {term.bloombergMnemonic && (
                          <span className="text-[9px] font-mono font-bold px-1.5 py-0.2 bg-amber-500/20 text-amber-300 border border-amber-500/30 rounded">
                            🟧 {term.bloombergMnemonic}
                          </span>
                        )}
                      </div>
                      <div className="flex items-center gap-2 text-[10px] text-slate-400 font-mono">
                        <span>/{term.termKey}</span>
                        <span>•</span>
                        <span
                          className={`font-semibold ${
                            term.eligibilitySource === 'DIRECT'
                              ? 'text-emerald-400'
                              : term.eligibilitySource === 'RELATED'
                              ? 'text-indigo-400'
                              : term.eligibilitySource === 'CALCULATED'
                              ? 'text-amber-400'
                              : 'text-rose-400'
                          }`}
                        >
                          {term.eligibilitySource}
                        </span>
                        {term.mappings.length > 1 && (
                          <span className="text-amber-400 bg-amber-950/40 px-1 rounded border border-amber-800">
                            {term.mappings.length} column mappings
                          </span>
                        )}
                      </div>
                    </div>

                    <div className="flex items-center gap-2">
                      <span className="text-[10px] px-2 py-0.5 bg-slate-800 rounded text-slate-300 font-mono">
                        {term.defaultRole}
                      </span>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        </div>

        {/* RIGHT COLUMN: MULTI-BACKEND MATRIX & FIELD OVERRIDES (7 Cols) */}
        <div className="xl:col-span-7 space-y-6">
          {/* SECTION 3: EXPANDABLE MULTI-BACKEND BINDINGS */}
          <div className="bg-[#0B1528] rounded-xl border border-slate-800 p-5 shadow-lg">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h2 className="text-sm font-bold uppercase tracking-wider text-slate-300 flex items-center gap-2">
                  <Cpu className="w-4 h-4 text-teal-400" /> 3. Multi-Backend Binding Matrix
                </h2>
                <p className="text-[11px] text-slate-400 mt-0.5">
                  Configure physical driving tables and verify cross-backend field coverage.
                </p>
              </div>
              <button className="flex items-center gap-1 px-2.5 py-1 text-xs bg-slate-800 hover:bg-slate-700 text-teal-300 rounded-lg border border-slate-700 transition">
                <Plus className="w-3.5 h-3.5" /> Add Backend Binding
              </button>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-3 mb-4">
              {bindings.map((b) => (
                <div
                  key={b.backendId}
                  className={`p-3.5 rounded-xl border transition-all ${
                    b.isDefault
                      ? 'bg-gradient-to-b from-[#0e2238] to-[#081524] border-teal-500/50 shadow-[0_0_15px_rgba(20,184,166,0.1)]'
                      : 'bg-[#070E1B] border-slate-800 hover:border-slate-700'
                  }`}
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <span className="font-semibold text-xs text-white">{b.backendName}</span>
                    </div>
                    {b.isDefault ? (
                      <span className="flex items-center gap-1 text-[10px] font-bold px-2 py-0.5 bg-teal-500/20 text-teal-300 border border-teal-500/40 rounded-full">
                        <Star className="w-3 h-3 fill-teal-400" /> DEFAULT
                      </span>
                    ) : (
                      <button
                        onClick={() => handleSetDefaultBinding(b.backendId)}
                        className="text-[10px] text-slate-400 hover:text-teal-300 flex items-center gap-1"
                      >
                        Set Default
                      </button>
                    )}
                  </div>

                  <div className="mt-2.5 space-y-1 text-xs">
                    <div className="flex items-center justify-between text-slate-400">
                      <span className="text-[10px] uppercase">Driving Table:</span>
                      <span className="font-mono text-slate-200 font-semibold">{b.drivingTableName}</span>
                    </div>
                    <div className="flex items-center justify-between text-slate-400">
                      <span className="text-[10px] uppercase">PK Detection:</span>
                      <span className="font-mono text-emerald-400 font-semibold">{b.pkColumnName} ➔ {b.suggestedBkTerm}</span>
                    </div>
                  </div>
                </div>
              ))}
            </div>

            {/* SELECTED FIELDS RESOLUTION MATRIX */}
            <div className="border-t border-slate-800 pt-4">
              <div className="flex items-center justify-between mb-3">
                <span className="text-xs font-bold uppercase tracking-wider text-slate-300">
                  Active Field Bindings ({selectedFields.length})
                </span>
                <span className="text-[11px] text-slate-400">
                  Click field row to configure role overrides & transformations.
                </span>
              </div>

              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs">
                  <thead>
                    <tr className="border-b border-slate-800 text-[10px] uppercase tracking-wider text-slate-500 font-mono">
                      <th className="pb-2">Field / Term</th>
                      <th className="pb-2">Role</th>
                      <th className="pb-2">Requirement</th>
                      {bindings.map((b) => (
                        <th key={b.backendId} className="pb-2">
                          {b.engineType}
                        </th>
                      ))}
                      <th className="pb-2 text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-800/60 font-mono">
                    {selectedFields.map((field) => {
                      const isEditing = activeEditingField?.fieldId === field.fieldId;
                      return (
                        <React.Fragment key={field.fieldId}>
                          <tr
                            onClick={() => setActiveEditingField(isEditing ? null : field)}
                            className={`hover:bg-slate-800/40 transition cursor-pointer ${
                              isEditing ? 'bg-teal-950/30' : ''
                            }`}
                          >
                            <td className="py-2.5 font-sans font-medium text-slate-200">
                              <div className="flex items-center gap-1.5">
                                {isEditing ? (
                                  <ChevronDown className="w-3.5 h-3.5 text-teal-400" />
                                ) : (
                                  <ChevronRight className="w-3.5 h-3.5 text-slate-500" />
                                )}
                                <span>{field.termName}</span>
                              </div>
                              <span className="text-[10px] text-slate-500 block font-mono pl-5">
                                {field.termKey}
                              </span>
                            </td>

                            <td className="py-2.5">
                              <span className="px-1.5 py-0.5 bg-slate-800 text-slate-300 rounded text-[10px]">
                                {field.overrides.fieldRole}
                              </span>
                            </td>

                            <td className="py-2.5">
                              <span
                                className={`px-1.5 py-0.5 rounded text-[10px] font-semibold ${
                                  field.overrides.bindingRequirement === 'REQUIRED'
                                    ? 'bg-rose-500/20 text-rose-300'
                                    : field.overrides.bindingRequirement === 'CALCULATED'
                                    ? 'bg-amber-500/20 text-amber-300'
                                    : 'bg-slate-800 text-slate-400'
                                }`}
                              >
                                {field.overrides.bindingRequirement}
                              </span>
                            </td>

                            {bindings.map((b) => {
                              const mappingNodeId = field.selectedColumnMapping?.[b.backendId];
                              const isBound = Boolean(mappingNodeId || field.eligibilitySource === 'CALCULATED');
                              return (
                                <td key={b.backendId} className="py-2.5">
                                  {isBound ? (
                                    <span className="inline-flex items-center gap-1 text-[11px] text-emerald-400">
                                      <CheckCircle2 className="w-3.5 h-3.5" /> Bound
                                      {field.overrides.transformationType !== 'NONE' && (
                                        <span className="text-[9px] bg-amber-500/20 text-amber-300 px-1 rounded">
                                          🔧
                                        </span>
                                      )}
                                    </span>
                                  ) : (
                                    <span className="inline-flex items-center gap-1 text-[11px] text-amber-400">
                                      <AlertTriangle className="w-3.5 h-3.5" /> Unmapped
                                    </span>
                                  )}
                                </td>
                              );
                            })}

                            <td className="py-2.5 text-right">
                              <button
                                onClick={(e) => {
                                  e.stopPropagation();
                                  setSelectedFields(selectedFields.filter((f) => f.fieldId !== field.fieldId));
                                }}
                                className="p-1 hover:text-rose-400 text-slate-500 transition"
                              >
                                <Trash2 className="w-3.5 h-3.5" />
                              </button>
                            </td>
                          </tr>

                          {/* INLINE FIELD DETAIL / OVERRIDE PANEL */}
                          {isEditing && (
                            <tr>
                              <td colSpan={3 + bindings.length + 1} className="p-4 bg-[#070E1B] border-b border-teal-500/30">
                                <div className="space-y-3 font-sans">
                                  <div className="flex items-center justify-between border-b border-slate-800 pb-2">
                                    <span className="text-xs font-bold text-teal-300 uppercase tracking-wide flex items-center gap-2">
                                      <Edit3 className="w-3.5 h-3.5" /> Field Overrides: {field.termName}
                                    </span>
                                    <label className="flex items-center gap-1.5 text-xs text-slate-400 cursor-pointer">
                                      <input
                                        type="checkbox"
                                        checked={field.overrides.inheritsDefaults}
                                        onChange={(e) => {
                                          const checked = e.target.checked;
                                          setSelectedFields(
                                            selectedFields.map((f) =>
                                              f.fieldId === field.fieldId
                                                ? {
                                                    ...f,
                                                    overrides: { ...f.overrides, inheritsDefaults: checked },
                                                  }
                                                : f
                                            )
                                          );
                                        }}
                                        className="rounded border-slate-700 text-teal-500 focus:ring-0"
                                      />
                                      Inherit Default Semantic Meaning
                                    </label>
                                  </div>

                                  <div className="grid grid-cols-1 md:grid-cols-4 gap-3 text-xs">
                                    <div>
                                      <label className="block text-[10px] text-slate-400 uppercase mb-1">Field Role</label>
                                      <select
                                        value={field.overrides.fieldRole}
                                        onChange={(e) => {
                                          const role = e.target.value as FieldRole;
                                          setSelectedFields(
                                            selectedFields.map((f) =>
                                              f.fieldId === field.fieldId
                                                ? { ...f, overrides: { ...f.overrides, fieldRole: role } }
                                                : f
                                            )
                                          );
                                        }}
                                        className="w-full bg-[#0B1528] border border-slate-700 rounded px-2.5 py-1.5 text-xs text-white"
                                      >
                                        <option value="KEY">KEY</option>
                                        <option value="DIMENSION">DIMENSION</option>
                                        <option value="MEASURE">MEASURE</option>
                                        <option value="ATTRIBUTE">ATTRIBUTE</option>
                                      </select>
                                    </div>

                                    <div>
                                      <label className="block text-[10px] text-slate-400 uppercase mb-1">Requirement</label>
                                      <select
                                        value={field.overrides.bindingRequirement}
                                        onChange={(e) => {
                                          const req = e.target.value as BindingRequirement;
                                          setSelectedFields(
                                            selectedFields.map((f) =>
                                              f.fieldId === field.fieldId
                                                ? { ...f, overrides: { ...f.overrides, bindingRequirement: req } }
                                                : f
                                            )
                                          );
                                        }}
                                        className="w-full bg-[#0B1528] border border-slate-700 rounded px-2.5 py-1.5 text-xs text-white"
                                      >
                                        <option value="REQUIRED">REQUIRED (All Backends)</option>
                                        <option value="OPTIONAL">OPTIONAL (Inject Null)</option>
                                        <option value="BACKEND_SPECIFIC">BACKEND SPECIFIC</option>
                                        <option value="CALCULATED">CALCULATED</option>
                                      </select>
                                    </div>

                                    <div>
                                      <label className="block text-[10px] text-slate-400 uppercase mb-1">Transformation</label>
                                      <select
                                        value={field.overrides.transformationType}
                                        onChange={(e) => {
                                          const trans = e.target.value as TransformationType;
                                          setSelectedFields(
                                            selectedFields.map((f) =>
                                              f.fieldId === field.fieldId
                                                ? { ...f, overrides: { ...f.overrides, transformationType: trans } }
                                                : f
                                            )
                                          );
                                        }}
                                        className="w-full bg-[#0B1528] border border-slate-700 rounded px-2.5 py-1.5 text-xs text-white"
                                      >
                                        <option value="NONE">NONE (Raw Column)</option>
                                        <option value="NORMALIZE">NORMALIZE (ISO / Format)</option>
                                        <option value="SQL_EXPRESSION">SQL EXPRESSION</option>
                                        <option value="JSON_PATH">JSON PATH EXTRACTION</option>
                                      </select>
                                    </div>

                                    <div>
                                      <label className="block text-[10px] text-slate-400 uppercase mb-1">Aggregation</label>
                                      <select
                                        value={field.overrides.aggregationType}
                                        onChange={(e) => {
                                          const agg = e.target.value as any;
                                          setSelectedFields(
                                            selectedFields.map((f) =>
                                              f.fieldId === field.fieldId
                                                ? { ...f, overrides: { ...f.overrides, aggregationType: agg } }
                                                : f
                                            )
                                          );
                                        }}
                                        className="w-full bg-[#0B1528] border border-slate-700 rounded px-2.5 py-1.5 text-xs text-white"
                                      >
                                        <option value="NONE">NONE</option>
                                        <option value="SUM">SUM</option>
                                        <option value="AVG">AVG</option>
                                        <option value="COUNT">COUNT</option>
                                      </select>
                                    </div>
                                  </div>

                                  {field.overrides.transformationType !== 'NONE' && (
                                    <div>
                                      <label className="block text-[10px] text-slate-400 uppercase mb-1">
                                        Transformation Expression / SQL
                                      </label>
                                      <input
                                        type="text"
                                        value={field.overrides.transformationSql || ''}
                                        onChange={(e) => {
                                          const sql = e.target.value;
                                          setSelectedFields(
                                            selectedFields.map((f) =>
                                              f.fieldId === field.fieldId
                                                ? { ...f, overrides: { ...f.overrides, transformationSql: sql } }
                                                : f
                                            )
                                          );
                                        }}
                                        placeholder="e.g. lookup_iso2(country_code) or SUM(line_total)"
                                        className="w-full bg-[#0B1528] border border-slate-700 font-mono text-xs text-amber-300 rounded px-3 py-1.5"
                                      />
                                    </div>
                                  )}
                                </div>
                              </td>
                            </tr>
                          )}
                        </React.Fragment>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            </div>
          </div>

          {/* SECTION 4: DECLARED RELATIONSHIPS */}
          <div className="bg-[#0B1528] rounded-xl border border-slate-800 p-5 shadow-lg">
            <div className="flex items-center justify-between mb-3">
              <h2 className="text-sm font-bold uppercase tracking-wider text-slate-300 flex items-center gap-2">
                <Link2 className="w-4 h-4 text-indigo-400" /> 4. Declared Relationships ({relationships.length})
              </h2>
              <button className="flex items-center gap-1 text-xs text-indigo-400 hover:text-indigo-300">
                <Plus className="w-3.5 h-3.5" /> Add Relation
              </button>
            </div>

            <div className="space-y-2.5">
              {relationships.map((rel) => (
                <div
                  key={rel.relId}
                  className="p-3 bg-[#070E1B] rounded-lg border border-slate-800 flex items-center justify-between text-xs"
                >
                  <div className="space-y-0.5">
                    <div className="flex items-center gap-2">
                      <span className="font-semibold text-slate-200">{boDraft.boName}</span>
                      <ArrowRight className="w-3.5 h-3.5 text-indigo-400" />
                      <span className="font-semibold text-indigo-300">{rel.toBoName}</span>
                      <span className="px-1.5 py-0.2 bg-slate-800 text-[10px] font-mono rounded">
                        {rel.cardinality}
                      </span>
                    </div>
                    <span className="text-[10px] font-mono text-slate-400">Join: {rel.joinConditionSql}</span>
                  </div>
                  <button
                    onClick={() => setRelationships(relationships.filter((r) => r.relId !== rel.relId))}
                    className="text-slate-500 hover:text-rose-400 p-1"
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                </div>
              ))}
            </div>
          </div>

          {/* SECTION 5: VALIDATION SUMMARY & PUBLISH GATE */}
          <div className="bg-[#0B1528] rounded-xl border border-slate-800 p-5 shadow-lg">
            <h2 className="text-sm font-bold uppercase tracking-wider text-slate-300 flex items-center gap-2 mb-3">
              <ShieldCheck className="w-4 h-4 text-emerald-400" /> 5. Validation & Publish Gate
            </h2>

            <div className="grid grid-cols-2 md:grid-cols-4 gap-3 text-center">
              <div className="p-2.5 rounded-lg bg-[#070E1B] border border-slate-800">
                <span className="text-[10px] text-slate-400 block uppercase">Identity Triple</span>
                <span className="text-xs font-bold text-emerald-400 flex items-center justify-center gap-1 mt-1">
                  <CheckCircle2 className="w-3.5 h-3.5" /> VALID
                </span>
              </div>

              <div className="p-2.5 rounded-lg bg-[#070E1B] border border-slate-800">
                <span className="text-[10px] text-slate-400 block uppercase">Required Fields</span>
                <span
                  className={`text-xs font-bold flex items-center justify-center gap-1 mt-1 ${
                    validationSummary.unresolvedCount === 0 ? 'text-emerald-400' : 'text-rose-400'
                  }`}
                >
                  {validationSummary.unresolvedCount === 0 ? (
                    <CheckCircle2 className="w-3.5 h-3.5" />
                  ) : (
                    <XCircle className="w-3.5 h-3.5" />
                  )}
                  {validationSummary.requiredCount - validationSummary.unresolvedCount}/{validationSummary.requiredCount} Bound
                </span>
              </div>

              <div className="p-2.5 rounded-lg bg-[#070E1B] border border-slate-800">
                <span className="text-[10px] text-slate-400 block uppercase">Backends Configured</span>
                <span className="text-xs font-bold text-slate-200 mt-1 block">
                  {validationSummary.bindingsConfigured} Active
                </span>
              </div>

              <div className="p-2.5 rounded-lg bg-[#070E1B] border border-slate-800">
                <span className="text-[10px] text-slate-400 block uppercase">Publish Gate</span>
                <span
                  className={`text-xs font-bold uppercase mt-1 block ${
                    validationSummary.isReadyToPublish ? 'text-emerald-400' : 'text-amber-400'
                  }`}
                >
                  {validationSummary.isReadyToPublish ? 'READY TO PUBLISH' : 'BLOCKED (DRAFT)'}
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* ────────────────────────────────────────────────────────────────────────── */}
      {/* MODAL: MULTI-MAPPING COLUMN RESOLUTION */}
      {/* ────────────────────────────────────────────────────────────────────────── */}
      {resolvingMultiMappingTerm && (
        <div className="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-[#0B1528] rounded-xl max-w-md w-full border border-slate-700 shadow-2xl p-5 space-y-4">
            <div className="flex items-center justify-between border-b border-slate-800 pb-3">
              <div>
                <h3 className="font-bold text-white text-sm">Disambiguate Column Mapping</h3>
                <p className="text-xs text-slate-400 mt-0.5">
                  {resolvingMultiMappingTerm.termName} maps to multiple physical columns.
                </p>
              </div>
              <button onClick={() => setResolvingMultiMappingTerm(null)} className="text-slate-400 hover:text-white">
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="space-y-2">
              <span className="text-[11px] font-semibold text-slate-400 uppercase">
                Select Primary Source Column:
              </span>
              {resolvingMultiMappingTerm.mappings.map((m) => (
                <div
                  key={m.columnNodeId}
                  onClick={() => {
                    // Create field with user-selected primary mapping
                    const mappingRecord: Record<string, string> = {};
                    bindings.forEach((b) => {
                      mappingRecord[b.backendId] = m.columnNodeId;
                    });
                    const newField: BOFieldDraft = {
                      fieldId: `fld-${Date.now()}`,
                      termNodeId: resolvingMultiMappingTerm.termNodeId,
                      termKey: resolvingMultiMappingTerm.termKey,
                      termName: resolvingMultiMappingTerm.termName,
                      dataType: resolvingMultiMappingTerm.dataType,
                      eligibilitySource: resolvingMultiMappingTerm.eligibilitySource,
                      overrides: {
                        fieldRole: resolvingMultiMappingTerm.defaultRole,
                        aggregationType: 'NONE',
                        bindingRequirement: 'REQUIRED',
                        nullable: false,
                        isExposed: true,
                        transformationType: 'NONE',
                        inheritsDefaults: true,
                      },
                      selectedColumnMapping: mappingRecord,
                    };
                    setSelectedFields([...selectedFields, newField]);
                    setResolvingMultiMappingTerm(null);
                  }}
                  className="p-3 bg-[#070E1B] rounded-lg border border-slate-800 hover:border-teal-500 cursor-pointer transition flex items-center justify-between"
                >
                  <div className="space-y-0.5">
                    <span className="font-mono text-xs text-teal-300 font-semibold">
                      {m.tableName}.{m.columnName}
                    </span>
                    <span className="text-[10px] text-slate-500 block uppercase">
                      Source: {m.sourceType} {m.isPrimary ? '(Primary Default)' : ''}
                    </span>
                  </div>
                  <span className="text-xs px-2 py-1 bg-slate-800 text-slate-300 rounded">Select</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* ────────────────────────────────────────────────────────────────────────── */}
      {/* DRAWER: LINEAGE & GRAPH TOPOLOGY PREVIEW */}
      {/* ────────────────────────────────────────────────────────────────────────── */}
      {showLineagePreview && (
        <div className="fixed bottom-0 left-0 right-0 h-64 bg-[#070E1B]/95 backdrop-blur-xl border-t border-teal-500/40 p-5 z-40 shadow-2xl overflow-y-auto">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-xs font-bold uppercase tracking-wider text-teal-300 flex items-center gap-2">
              <Compass className="w-4 h-4" /> Live Semantic-to-Physical Lineage Projection
            </h3>
            <button onClick={() => setShowLineagePreview(false)} className="text-slate-400 hover:text-white">
              <X className="w-4 h-4" />
            </button>
          </div>

          <div className="flex items-center gap-4 text-xs font-mono">
            <div className="p-3 rounded-lg bg-[#0B1528] border border-teal-500/30 text-teal-300">
              [Tier 1: Business Object]<br />
              💼 {boDraft.boName} ({boDraft.boKey})
            </div>
            <ArrowRight className="w-4 h-4 text-slate-500" />
            <div className="p-3 rounded-lg bg-[#0B1528] border border-indigo-500/30 text-indigo-300">
              [Tier 2: Semantic Mesh]<br />
              🧠 {selectedFields.length} Mapped Terms
            </div>
            <ArrowRight className="w-4 h-4 text-slate-500" />
            <div className="p-3 rounded-lg bg-[#0B1528] border border-emerald-500/30 text-emerald-300">
              [Tier 3: Multi-Backend Storage]<br />
              🗄️ {bindings.map((b) => b.engineType).join(' | ')}
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default BusinessObjectStudio;
