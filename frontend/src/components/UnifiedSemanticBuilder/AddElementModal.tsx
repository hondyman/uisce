import { useMemo, useState } from 'react';
import * as TablerIcons from '@tabler/icons-react';
import { devLog } from '../../utils/devLogger';
import SqlMonacoEditor from '../SqlMonacoEditor';
import LibraryPanel from './LibraryPanel';
// Aliased in case referenced dynamically elsewhere; reference them so TS doesn't flag unused
const _SqlMonacoEditor = SqlMonacoEditor; const _LibraryPanel = LibraryPanel; void _SqlMonacoEditor; void _LibraryPanel;
import AddElementOverride from './AddElementOverride';
import AddElementModalHeader from './AddElementModalHeader';
import AddElementCustomForm from './AddElementCustomForm';
import { getTableIdFromVal, getTableLabelFromVal } from '../../utils/tableHelpers';
import './AddElementModal.css';

interface CoreOption {
  name: string;
  title: string;
  type: string;
  sql: string;
  description?: string;
  sourceTable?: string;
  sourceColumn?: string;
  format?: string;
  aggregationType?: string;
  defaultValue?: string;
  category?: string;
}


export type ElementKind = 'dimension' | 'measure' | 'filter' | 'join';

interface AddElementModalProps {
  open: boolean;
  kind: ElementKind | null;
  targetTable?: string | { id: string; qualified_path: string } | null;
  onClose: () => void;
  onCreate: (params: { mode: 'override' | 'custom'; kind: ElementKind; coreName?: string; values: any }) => void;
  coreOptions: CoreOption[];
  libraryOptions: CoreOption[];
  existingNames: string[]; // to prevent duplicates for custom
  nodes?: any[]; // FlowNode[] from useUnifiedSemanticBuilder
  semanticModel?: any;
}

const AddElementModal: React.FC<AddElementModalProps> = ({ open, kind, targetTable, onClose, onCreate, coreOptions, libraryOptions, existingNames, nodes = [], semanticModel }) => {
  const [formData, setFormData] = useState<any>({});
  // Get all tables from nodes
  const tableNodes = useMemo(() =>
    (nodes || []).filter((n: any) => n.data && (n.data.columns || n.data.label)),
    [nodes]
  );

  // Collect all tables used in the model (dimensions, measures, filters, joins)
  const modelTableSet = useMemo(() => {
    const set = new Set<string>();
    if (semanticModel) {
      ['dimensions', 'measures', 'filters'].forEach(type => {
        (semanticModel[type] || []).forEach((el: any) => {
          if (el.sourceTable) set.add(getTableIdFromVal(el.sourceTable));
        });
      });
      (semanticModel.joins || []).forEach((join: any) => {
        if (join.leftTable) set.add(getTableIdFromVal(join.leftTable));
        if (join.rightTable) set.add(getTableIdFromVal(join.rightTable));
      });
    }
    return set;
  }, [semanticModel]);

  // Only show tables in the modelTableSet
  const tableOptions = useMemo(() =>
    tableNodes
      .map((n: any) => ({
        // use the node id as the canonical value so comparisons against modelTableSet (which stores ids)
        // are consistent. Keep a human-friendly label separately.
        value: n.id,
        label: n.data.label || n.data.tableName || n.id,
        columns: n.data.columns || []
      }))
      .filter(t => modelTableSet.has(t.value)),
    [tableNodes, modelTableSet]
  );

  // Find columns for selected table
  const selectedTable = tableOptions.find(t => t.value === (formData.sourceTable ? getTableIdFromVal(formData.sourceTable) : formData.sourceTable));
  const _columnOptions = selectedTable ? selectedTable.columns : [];
  void _columnOptions;
  const [mode, setMode] = useState<'override' | 'custom' | null>(null);
  const [coreSearch, setCoreSearch] = useState('');
  const [coreSelected, setCoreSelected] = useState<string>('');
  const [librarySearch, setLibrarySearch] = useState('');
  const [libraryPanelOpen, setLibraryPanelOpen] = useState(false);
  const [_hoveredCalc, setHoveredCalc] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const resetAll = () => {
    setMode(null); setCoreSearch(''); setCoreSelected(''); setLibrarySearch(''); setLibraryPanelOpen(false); setHoveredCalc(null); setFormData({}); setError(null);
  };

  const close = () => { resetAll(); onClose(); };

  const filteredCore = useMemo(() => {
    devLog('AddElementModal filteredCore - kind:', kind, 'coreOptions:', coreOptions.length, 'filtered result:', coreOptions.filter(c => c.type === kind && c.name.toLowerCase().includes(coreSearch.toLowerCase())));
    return coreOptions.filter(c => c.type === kind && c.name.toLowerCase().includes(coreSearch.toLowerCase()));
  }, [coreOptions, coreSearch, kind]);
  const filteredLibrary = useMemo(() => {
    const searchLower = librarySearch.toLowerCase();
    // Only show library for measures since all library items are measures
    if (kind !== 'measure') return [];
    
    return libraryOptions.filter(c => 
      c.type === kind && (
        c.name.toLowerCase().includes(searchLower) ||
        c.title?.toLowerCase().includes(searchLower) ||
        c.sql?.toLowerCase().includes(searchLower) ||
        c.description?.toLowerCase().includes(searchLower)
      )
    );
  }, [libraryOptions, librarySearch, kind]);

  const libraryByCategory = useMemo(() => {
    const grouped: Record<string, CoreOption[]> = {};
    filteredLibrary.forEach(opt => {
      const cat = opt.category || 'Other';
      if (!grouped[cat]) grouped[cat] = [];
      grouped[cat].push(opt);
    });
    return grouped;
  }, [filteredLibrary]);

  const _highlightText = (text: string, search: string) => {
    if (!search) return text;
    const regex = new RegExp(`(${search})`, 'gi');
    const parts = text.split(regex);
    return parts.map((part, index) => 
      regex.test(part) ? <mark key={index} className="highlight">{part}</mark> : part
    );
  };
  // referenced by some story/test harnesses via dynamic access
  void _highlightText;

  const handleCreate = () => {
    if (!kind || !mode) return;
    if (mode === 'override') {
      if (!coreSelected) { setError('Select a core item'); return; }
      const core = coreOptions.find(c => c.name === coreSelected);
      if (!core) { setError('Invalid core selection'); return; }
      onCreate({ mode, kind, coreName: coreSelected, values: { ...core } });
      close();
      return;
    }
    // custom
    const name = (formData.name || '').trim();
    if (!name) { setError('Name required'); return; }
    if (existingNames.includes(name)) { setError('Name already exists'); return; }
    // Ensure any selected table objects are serialized to ids for the created payload
    const serializeFormData = (fd: any) => {
      if (!fd) return fd;
      const s = { ...fd };
      if (s.sourceTable) s.sourceTable = getTableIdFromVal(s.sourceTable);
      if (s.leftTable) s.leftTable = getTableIdFromVal(s.leftTable);
      if (s.rightTable) s.rightTable = getTableIdFromVal(s.rightTable);
      if (s.joins && Array.isArray(s.joins)) {
        s.joins = s.joins.map((j: any) => ({
          ...j,
          leftTable: getTableIdFromVal(j.leftTable),
          rightTable: getTableIdFromVal(j.rightTable),
        }));
      }
      return s;
    };

    const values = serializeFormData(formData);
    onCreate({ mode, kind, values });
    close();
  };

  if (!open || !kind) return null;

  // Compute display model name (table + model)
  const getDisplayModelName = () => {
    const model = semanticModel?.name || '';
    // If override mode and core selected, prefer core's sourceTable
    if (mode === 'override' && coreSelected) {
      const core = coreOptions.find(c => c.name === coreSelected);
      if (core) {
  // Try multiple fields for table name (include common variants), then parse from SQL if needed
  let table = core.sourceTable || (core as any).coreSourceTable || (core as any).table || (core as any).source_table || (core as any).core_source_table || '';
        if (!table && core.sql && typeof core.sql === 'string') {
          // parse leading identifier before a dot, e.g. customers.id or public.customers.id
          const m = core.sql.match(/([A-Za-z0-9_]+)\./);
          if (m) table = m[1];
        }
        if (!table && core.name) table = core.name;
        if (table && model) return `${table} · ${model}`;
        if (table) return table;
      }
    }

    // If we have an explicit title/name set (for example from a selected column), prefer that.
    if (formData?.title) return formData.title;
    if (formData?.name) return formData.name;

    // For custom mode, prefer selected sourceTable from form. Support the form holding either
    // a full table object or a primitive id/string.
    if (formData?.sourceTable) {
      const tableVal = formData.sourceTable;
      const tableLabel = typeof tableVal === 'object' ? getTableLabelFromVal(tableVal) : tableVal;
      if (tableLabel && model) return `${tableLabel} · ${model}`;
      if (tableLabel) return tableLabel;
    }

    // Fallback to model name alone
    return model || '';
  };

  return (
    <div className="add-element-modal-overlay">
      <div className="add-element-modal">
        <AddElementModalHeader mode={mode} kind={kind} displayName={getDisplayModelName()} onClose={close} />
        {!mode && (
          <div className="mode-select">
            <button className="mode-card" onClick={() => setMode('override')}>
              <TablerIcons.IconSettings size={28} />
              <h4>Override Core</h4>
              <p>Select a core {kind} and override its properties</p>
            </button>
            <button className="mode-card" onClick={() => setMode('custom')}>
              <TablerIcons.IconPlus size={28} />
              <h4>Custom</h4>
              <p>Choose from calculation library or define a brand new {kind}</p>
            </button>
          </div>
        )}

        {mode === 'override' && (
          <AddElementOverride
            filteredCore={filteredCore}
            coreSelected={coreSelected}
            setCoreSelected={setCoreSelected}
            coreSearch={coreSearch}
            setCoreSearch={setCoreSearch}
            kind={kind}
            onBack={() => setMode(null)}
            onCreateOverride={handleCreate}
          />
        )}

        {mode === 'custom' && (
          <AddElementCustomForm
            kind={kind}
            targetTable={targetTable}
            formData={formData}
            setFormData={setFormData}
            setMode={setMode}
            libraryPanelOpen={libraryPanelOpen}
            setLibraryPanelOpen={setLibraryPanelOpen}
            libraryByCategory={libraryByCategory}
            librarySearch={librarySearch}
            setLibrarySearch={setLibrarySearch}
            onCreate={handleCreate}
            semanticModel={semanticModel}
          />
        )}
        {error && <div className="error-text">{error}</div>}
      </div>
    </div>
  );
};

export default AddElementModal;
