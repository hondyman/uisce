import { useEffect, useMemo, useState, useRef } from 'react';
import type { ReactNode } from 'react';
import {
  Box,
  Button,
  ButtonGroup,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  IconButton,
  InputAdornment,
  InputLabel,
  MenuItem,
  Select,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
// MUI TreeView migrated from @mui/lab to @mui/x-tree-view
// Keep using legacy @mui/lab TreeView until full migration can be done without type errors.
// Revert change to avoid runtime breakage; proper refactor to @mui/x-tree-view pending.
import { TreeView, TreeItem } from '@mui/lab';
import {
  Add,
  BarChart,
  ChevronRight,
  Close,
  ContentCopy,
  DataObject,
  Delete,
  Edit,
  ExpandMore,
  Folder,
  Layers,
  Search,
} from '@mui/icons-material';
import * as yaml from 'js-yaml';
import MonacoCodeEditor from './LazyMonacoEditor';

interface BusinessObjectField {
  key: string;
  name: string;
  displayName?: string;
  technicalName?: string;
  type: string;
  isCore?: boolean;
}

interface BusinessObject {
  id: string;
  name: string;
  display_name: string;
  description?: string;
  config?: {
    fields?: BusinessObjectField[];
  };
}

type SemanticType = 'dimension' | 'measure' | 'time';

interface SemanticField {
  key: string;
  displayName: string;
  fieldType: string;
  semanticType: SemanticType;
  description: string;
  isCustom: boolean;
}

type FilterTab = 'all' | 'custom' | 'inherited';
const FILTER_TABS: FilterTab[] = ['all', 'custom', 'inherited'];

const determineSemanticType = ({ name, type }: BusinessObjectField): SemanticType => {
  const normalizedName = (name || '').toLowerCase();
  const normalizedType = (type || '').toLowerCase();
  const timeKeywords = ['date', 'time', 'timestamp', 'created_at', 'updated_at', 'period', 'year', 'month'];

  if (timeKeywords.some(keyword => normalizedName.includes(keyword)) || ['date', 'datetime', 'timestamp'].includes(normalizedType)) {
    return 'time';
  }

  const measureCandidates = ['number', 'integer', 'decimal', 'float', 'double', 'currency', 'amount', 'percentage'];
  if (measureCandidates.includes(normalizedType)) {
    return 'measure';
  }

  return 'dimension';
};

const mapToSemanticField = (field: BusinessObjectField): SemanticField => ({
  key: field.key,
  displayName: field.displayName || field.name,
  fieldType: field.type,
  semanticType: determineSemanticType(field),
  description: field.technicalName || '',
  isCustom: field.isCore === false,
});

const semanticTypeBadge = (type: SemanticType) => {
  switch (type) {
    case 'dimension':
      return 'bg-blue-100 text-blue-700';
    case 'measure':
      return 'bg-emerald-100 text-emerald-700';
    case 'time':
      return 'bg-purple-100 text-purple-700';
    default:
      return 'bg-gray-100 text-gray-700';
  }
};

const escapeRegExp = (value: string) => value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
const highlightMatches = (text: string, query?: string) => {
  if (!query) {
    return text;
  }

  const regex = new RegExp(`(${escapeRegExp(query)})`, 'gi');
  const highlighted: Array<string | ReactNode> = [];
  let lastIndex = 0;
  let match: RegExpExecArray | null;

  while ((match = regex.exec(text)) !== null) {
    highlighted.push(text.slice(lastIndex, match.index));
    highlighted.push(
      <span key={`${match.index}-${match[0]}`} className="bg-yellow-200 font-semibold text-slate-900">
        {match[0]}
      </span>
    );
    lastIndex = regex.lastIndex;
  }

  highlighted.push(text.slice(lastIndex));
  return highlighted.length === 1 ? highlighted[0] : highlighted;
};

const filterLinesForQuery = (value: string, query: string) => {
  if (!query) {
    return '';
  }

  const lines = value.split('\n');
  const matches = new Set<number>();
  lines.forEach((line, index) => {
    if (line.toLowerCase().includes(query)) {
      matches.add(index);
      if (index > 0) matches.add(index - 1);
      if (index < lines.length - 1) matches.add(index + 1);
    }
  });

  if (!matches.size) {
    return '';
  }

  return lines.filter((_, idx) => matches.has(idx)).join('\n');
};

const buildSemanticModelPayload = (fields: SemanticField[], model: 'core' | 'custom', businessObjectName: string) => {
  const dimensions = fields.filter(field => field.semanticType === 'dimension');
  const measures = fields.filter(field => field.semanticType === 'measure');
  const timeDimensions = fields.filter(field => field.semanticType === 'time');

  const payload: Record<string, any> = {
    name: `${businessObjectName}_${model}`,
    sql_table: `\${${businessObjectName}}`,
    dimensions: dimensions.map(d => ({
      name: d.key,
      title: d.displayName,
      type: 'string',
    })),
    measures: measures.map(m => ({
      name: m.key,
      title: m.displayName,
      type: 'sum',
    })),
  };

  if (timeDimensions.length) {
    payload.time_dimensions = timeDimensions.map(t => ({
      name: t.key,
      title: t.displayName,
      type: 'time',
    }));
  }

  if (model === 'custom') {
    payload.extends = `${businessObjectName}_core`;
  }

  return payload;
};

interface SemanticModelTabProps {
  businessObject: BusinessObject;
  tenantId: string;
  datasourceId: string;
}

export default function SemanticModelTab({
  businessObject,
  tenantId: _tenantId,
  datasourceId: _datasourceId,
}: SemanticModelTabProps) {
  const [coreFields, setCoreFields] = useState<SemanticField[]>([]);
  const [customFields, setCustomFields] = useState<SemanticField[]>([]);
  const [selectedModel, setSelectedModel] = useState<'core' | 'custom'>('core');
  const [filterTab, setFilterTab] = useState<FilterTab>('all');
  const [searchTerm, setSearchTerm] = useState('');
  const [editingField, setEditingField] = useState<SemanticField | null>(null);
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [codeViewMode, setCodeViewMode] = useState<'yaml' | 'json'>('yaml');
  const [expandedIds, setExpandedIds] = useState<string[]>(['core-model', 'custom-model', 'custom-fields']);

  // refs to the rendered DOM nodes for tree labels so we can reveal/scroll-to them
  const nodeRefs = useRef<Record<string, HTMLDivElement | null>>({});
  // monaco editor API handle (from MonacoCodeEditor.onMount)
  const monacoApiRef = useRef<any>(null);

  useEffect(() => {
    const fields = businessObject.config?.fields || [];
    setCoreFields(fields.filter(field => field.isCore !== false).map(mapToSemanticField));
    setCustomFields(fields.filter(field => field.isCore === false).map(mapToSemanticField));
  }, [businessObject]);

  const normalizedSearchTerm = searchTerm.trim();
  const searchTermLower = normalizedSearchTerm.toLowerCase();

  const filteredFields = useMemo(() => {
    const base = selectedModel === 'core' ? coreFields : customFields;
    let result = base;

    if (filterTab === 'custom') {
      result = result.filter(field => field.isCustom);
    } else if (filterTab === 'inherited') {
      result = result.filter(field => !field.isCustom);
    }

    if (searchTermLower) {
      result = result.filter(field =>
        field.displayName.toLowerCase().includes(searchTermLower) ||
        field.key.toLowerCase().includes(searchTermLower)
      );
    }

    return result;
  }, [selectedModel, filterTab, searchTermLower, coreFields, customFields]);

  const coreModelPayload = useMemo(() => buildSemanticModelPayload(coreFields, 'core', businessObject.name), [coreFields, businessObject.name]);
  const customModelPayload = useMemo(() => buildSemanticModelPayload(customFields, 'custom', businessObject.name), [customFields, businessObject.name]);

  const selectedModelPayload = selectedModel === 'core' ? coreModelPayload : customModelPayload;

  const yamlBase = useMemo(() => yaml.dump(selectedModelPayload, { indent: 2, lineWidth: -1 }), [selectedModelPayload]);
  const jsonBase = useMemo(() => JSON.stringify(selectedModelPayload, null, 2), [selectedModelPayload]);

  const yamlForSearch = useMemo(() => {
    if (!searchTermLower) return yamlBase;
    const filtered = filterLinesForQuery(yamlBase, searchTermLower);
    return filtered || yamlBase;
  }, [yamlBase, searchTermLower]);

  const jsonForSearch = useMemo(() => {
    if (!searchTermLower) return jsonBase;
    const filtered = filterLinesForQuery(jsonBase, searchTermLower);
    return filtered || jsonBase;
  }, [jsonBase, searchTermLower]);

  const monacoValue = codeViewMode === 'yaml' ? yamlForSearch : jsonForSearch;

  const renderHighlightedText = (value: string) => (
    <>{highlightMatches(value, normalizedSearchTerm)}</>
  );

  // Auto-expand tree nodes when search term matches children or model labels
  useEffect(() => {
    if (!searchTerm.trim()) {
      setExpandedIds(['core-model', 'custom-model', 'custom-fields']);
      return;
    }

    const term = searchTerm.trim().toLowerCase();
    const newExpanded = new Set<string>();

    // expand roots when the model name itself matches
    if (businessObject.name.toLowerCase().includes(term)) {
      newExpanded.add('core-model');
      newExpanded.add('custom-model');
      newExpanded.add('custom-fields');
    }

    // expand any core field matches
    coreFields.forEach(f => {
      if (f.displayName.toLowerCase().includes(term) || f.key.toLowerCase().includes(term)) {
        newExpanded.add('core-model');
        newExpanded.add(`core-field-${f.key}`);
      }
    });

    // expand custom model and its list when custom field matches
    customFields.forEach(f => {
      if (f.displayName.toLowerCase().includes(term) || f.key.toLowerCase().includes(term)) {
        newExpanded.add('custom-model');
        newExpanded.add('custom-fields');
        newExpanded.add(`custom-field-${f.key}`);
      }
    });

    // always keep model roots visible
    newExpanded.add('core-model');
    newExpanded.add('custom-model');
    setExpandedIds(Array.from(newExpanded));
  }, [searchTerm, coreFields, customFields, businessObject.name]);

  // When search changes and we've expanded matching nodes, reveal the first matching node in the tree
  useEffect(() => {
    const term = searchTerm.trim().toLowerCase();
    if (!term) return;

    let firstId: string | null = null;

    const coreMatch = coreFields.find(f => f.displayName.toLowerCase().includes(term) || f.key.toLowerCase().includes(term));
    if (coreMatch) {
      firstId = `core-field-${coreMatch.key}`;
    } else {
      const customMatch = customFields.find(f => f.displayName.toLowerCase().includes(term) || f.key.toLowerCase().includes(term));
      if (customMatch) {
        firstId = `custom-field-${customMatch.key}`;
      }
    }

    if (!firstId) {
      if (businessObject.name.toLowerCase().includes(term)) {
        firstId = 'core-model';
      }
    }

    if (!firstId) return;

    // Wait for the DOM to update after expansion then scroll the node into view
    const t = window.setTimeout(() => {
      const el = nodeRefs.current[firstId!];
      if (el && typeof el.scrollIntoView === 'function') {
        try {
          el.scrollIntoView({ behavior: 'smooth', block: 'center', inline: 'nearest' });
          // attempt to focus for keyboard users
          if (typeof (el as any).focus === 'function') {
            (el as any).focus();
          }
        } catch (err) {
          // ignore
        }
      }
    }, 60);

    return () => window.clearTimeout(t);
  }, [expandedIds, searchTerm, coreFields, customFields, businessObject.name]);

    // reveal a field (by key) in the Monaco editor if available
    const revealFieldInEditor = (fieldKey: string) => {
      try {
        const api = monacoApiRef.current;
        // prefer Monaco-powered search/reveal if available
        if (api?.findAndReveal && typeof api.findAndReveal === 'function') {
          const found = api.findAndReveal(fieldKey);
          if (found && found > 0) return;
        }

        // fallback to text-based search
        const text = api?.getValue?.() ?? monacoValue;
        if (!text) return;
        // try JSON style first: "name": "fieldKey"
        let idx = -1;
        if (codeViewMode === 'json') {
          idx = text.indexOf(`"name": "${fieldKey}"`);
          if (idx === -1) idx = text.indexOf(`"${fieldKey}"`);
        } else {
          // yaml: look for `name: fieldKey` or `- name: fieldKey`
          idx = text.indexOf(`name: ${fieldKey}`);
          if (idx === -1) idx = text.indexOf(`- name: ${fieldKey}`);
        }
        if (idx === -1) {
          // fallback: search for key anywhere
          idx = text.indexOf(fieldKey);
        }
        if (idx === -1) return;
        const line = text.slice(0, idx).split('\n').length;
        if (api?.revealRange) api.revealRange(line, line);
        else if (api?.revealLine) api.revealLine(line, 'center');
        if (api?.focus) api.focus();
      } catch (e) {
        // ignore
      }
    };

  const handleSelectModel = (model: 'core' | 'custom') => {
    setSelectedModel(model);
    setFilterTab('all');
  };

  const handleOpenEdit = (field: SemanticField) => {
    setEditingField(field);
    setIsDialogOpen(true);
  };

  const handleAddCustomField = () => {
    const newField: SemanticField = {
      key: `custom_field_${Date.now()}`,
      displayName: 'New Custom Field',
      fieldType: 'string',
      semanticType: 'dimension',
      description: '',
      isCustom: true,
    };
    setCustomFields(prev => [...prev, newField]);
    setSelectedModel('custom');
    setFilterTab('all');
  };

  const handleDeleteCustomField = (fieldKey: string) => {
    setCustomFields(prev => prev.filter(field => field.key !== fieldKey));
  };

  const copyCurrentConfig = () => {
    const payload = codeViewMode === 'yaml' ? yamlForSearch : jsonForSearch;
    navigator.clipboard?.writeText(payload);
  };

  const handleClearSearch = () => setSearchTerm('');

  const modelsTree = (
    <TreeView
      defaultCollapseIcon={<ExpandMore className="text-slate-500" />}
      defaultExpandIcon={<ChevronRight className="text-slate-500" />}
      expanded={expandedIds}
      onNodeToggle={(_event: any, nodeIds: string[]) => setExpandedIds(nodeIds)}
      sx={{ flexGrow: 1 }}
    >
      <TreeItem
        nodeId="core-model"
        label={
          <Box
            ref={(el: HTMLDivElement | null) => (nodeRefs.current['core-model'] = el)}
            id={`semantic-tree-core-model`}
            tabIndex={-1}
            onClick={() => handleSelectModel('core')}
            className={`flex items-center justify-between rounded-xl px-3 py-2 cursor-pointer transition-colors ${
              selectedModel === 'core' ? 'bg-blue-50 text-blue-700' : 'hover:bg-slate-50 text-slate-700'
            }`}
          >
            <Box className="flex items-center gap-2">
              <Layers className="text-blue-600" fontSize="small" />
              <Box>
                <Typography variant="body2" className="font-medium">
                  {renderHighlightedText(`${businessObject.name}_core.yml`)}
                </Typography>
                <Typography variant="caption" className="text-slate-500">
                  {renderHighlightedText(`${coreFields.length} fields`)}
                </Typography>
              </Box>
            </Box>
          </Box>
        }
      />
      {coreFields.map(field => (
        <TreeItem
          key={`core-field-${field.key}`}
          nodeId={`core-field-${field.key}`}
          label={
            <Box
              ref={(el: HTMLDivElement | null) => (nodeRefs.current[`core-field-${field.key}`] = el)}
              id={`semantic-tree-core-field-${field.key}`}
              className="flex items-center gap-2 rounded-xl px-3 py-2 text-slate-500"
            >
              <DataObject fontSize="small" className="text-slate-500" />
              <Box>
                <Typography variant="caption" className="font-semibold text-slate-600">
                  {field.displayName}
                </Typography>
                <Typography variant="caption" className="text-slate-400">
                  {field.key}
                </Typography>
              </Box>
            </Box>
          }
        />
      ))}
      <TreeItem
        nodeId="custom-model"
        label={
          <Box
            ref={(el: HTMLDivElement | null) => (nodeRefs.current['custom-model'] = el)}
            id={`semantic-tree-custom-model`}
            tabIndex={-1}
            onClick={() => handleSelectModel('custom')}
            className={`flex items-center justify-between rounded-xl px-3 py-2 cursor-pointer transition-colors ${
              selectedModel === 'custom' ? 'bg-green-50 text-green-700' : 'hover:bg-slate-50 text-slate-700'
            }`}
          >
            <Box className="flex items-center gap-2">
              <Folder className="text-green-600" fontSize="small" />
              <Box>
                <Typography variant="body2" className="font-medium">
                  {renderHighlightedText(`${businessObject.name}_custom.yml`)}
                </Typography>
                <Typography variant="caption" className="text-slate-500">
                  {renderHighlightedText(`extends ${businessObject.name}_core`)}
                </Typography>
              </Box>
            </Box>
          </Box>
        }
      >
        <TreeItem
          nodeId="custom-fields"
          label={
            <Box
              ref={(el: HTMLDivElement | null) => (nodeRefs.current['custom-fields'] = el)}
              id={`semantic-tree-custom-fields`}
              className="flex items-center gap-2"
            >
              <BarChart fontSize="small" className="text-slate-500" />
              <Typography variant="caption" className="font-semibold text-slate-500">
                Custom Fields
              </Typography>
              <Chip label={customFields.length} size="small" className="text-slate-500" />
            </Box>
          }
        >
          {customFields.length === 0 && (
            <TreeItem
              nodeId="no-custom-fields"
              label={
                <Typography
                  ref={(el: HTMLDivElement | null) => (nodeRefs.current['no-custom-fields'] = el)}
                  variant="caption"
                  className="text-slate-500"
                >
                  No custom fields yet
                </Typography>
              }
            />
          )}
          {customFields.map(field => (
            <TreeItem
              key={field.key}
              nodeId={`custom-field-${field.key}`}
              label={
                <Box
                  ref={(el: HTMLDivElement | null) => (nodeRefs.current[`custom-field-${field.key}`] = el)}
                  id={`semantic-tree-custom-field-${field.key}`}
                  className="flex items-center gap-1"
                >
                  <DataObject fontSize="small" className="text-slate-500" />
                  <Box className="flex-1">
                    <Button
                      size="small"
                      variant="text"
                      onClick={(e) => { e.stopPropagation(); revealFieldInEditor(field.key); }}
                      className="normal-case p-0"
                      sx={{ textTransform: 'none', color: 'inherit', px: 0 }}
                    >
                      <Box sx={{ textAlign: 'left' }}>
                        <Typography variant="caption" className="font-semibold text-slate-600">
                          {renderHighlightedText(field.displayName)}
                        </Typography>
                        <Typography variant="caption" className="text-slate-400">
                          {renderHighlightedText(field.key)}
                        </Typography>
                      </Box>
                    </Button>
                  </Box>
                </Box>
              }
            />
          ))}
        </TreeItem>
      </TreeItem>
      {coreFields.map(field => (
        <TreeItem
          key={`core-field-${field.key}`}
          nodeId={`core-field-${field.key}`}
          label={
              <Box
                ref={(el: HTMLDivElement | null) => (nodeRefs.current[`core-field-${field.key}`] = el)}
                id={`semantic-tree-core-field-${field.key}-2`}
                className="flex items-center gap-2 rounded-xl px-3 py-2 text-slate-500"
              >
                <DataObject fontSize="small" className="text-slate-500" />
                <Box className="flex-1 flex items-center justify-between">
                  <div>
                    <Button
                      size="small"
                      variant="text"
                      onClick={(e) => { e.stopPropagation(); revealFieldInEditor(field.key); }}
                      className="normal-case p-0"
                      sx={{ textTransform: 'none', color: 'inherit', px: 0 }}
                    >
                      <Box sx={{ textAlign: 'left' }}>
                        <Typography variant="caption" className="font-semibold text-slate-600">
                          {renderHighlightedText(field.displayName)}
                        </Typography>
                        <Typography variant="caption" className="text-slate-400">
                          {renderHighlightedText(field.key)}
                        </Typography>
                      </Box>
                    </Button>
                  </div>
                </Box>
              </Box>
          }
        />
      ))}
    </TreeView>
  );

  return (
    <div className="flex h-full min-h-[720px] w-full overflow-hidden bg-slate-50">
      <aside className="w-96 border-r border-slate-200 bg-white">
        <div className="px-6 py-4 border-b border-slate-200">
          <Typography variant="subtitle1" className="font-semibold text-slate-700">
            Semantic Models
          </Typography>
        </div>
        <div className="px-6 py-4 space-y-4 overflow-y-auto h-[calc(100%-64px)]">
          <div className="rounded-2xl border border-slate-200 bg-slate-50 p-4">
            <Typography variant="caption" className="font-semibold text-slate-500 uppercase">
              Browse models
            </Typography>
            <Box className="mt-3 rounded-2xl bg-white p-3 shadow-sm">
              {modelsTree}
            </Box>
          </div>
        </div>
      </aside>

      <main className="flex flex-1 flex-col bg-white">
        <header className="flex items-center justify-between border-b border-slate-200 px-6 py-4">
          <div>
            <Typography variant="h6" className="text-slate-900 font-semibold">
              {selectedModel === 'core'
                ? `${businessObject.display_name} (Core)`
                : `${businessObject.display_name} (Custom)`}
            </Typography>
            <Typography variant="body2" className="text-slate-500">
              {selectedModel === 'core'
                ? 'Core semantic fields automatically detected by the builder'
                : 'Create custom fields to extend the core model.'}
            </Typography>
          </div>
          {selectedModel === 'custom' && (
            <Button
              startIcon={<Add />}
              variant="contained"
              color="primary"
              onClick={handleAddCustomField}
              className="normal-case px-4 py-2 text-sm"
            >
              Add Field
            </Button>
          )}
        </header>

        <div className="flex flex-1 overflow-hidden">
          <section className="flex flex-1 flex-col border-r border-slate-200">
            <div className="flex flex-col gap-3 border-b border-slate-200 px-6 py-4">
              <Stack direction="row" spacing={1}>
                {FILTER_TABS.map(tab => (
                  <Button
                    key={tab}
                    variant={filterTab === tab ? 'contained' : 'outlined'}
                    size="small"
                    onClick={() => setFilterTab(tab)}
                    className="capitalize"
                  >
                    {tab}
                  </Button>
                ))}
              </Stack>
              <TextField
                size="small"
                variant="outlined"
                placeholder="Type to filter fields..."
                value={searchTerm}
                onChange={e => setSearchTerm(e.target.value)}
                className="max-w-sm"
                InputProps={{
                  startAdornment: (
                    <InputAdornment position="start">
                      <Search fontSize="small" className="text-slate-500" />
                    </InputAdornment>
                  ),
                  endAdornment: normalizedSearchTerm ? (
                    <InputAdornment position="end">
                      <IconButton size="small" onClick={handleClearSearch}>
                        <Close fontSize="small" />
                      </IconButton>
                    </InputAdornment>
                  ) : undefined,
                }}
              />
            </div>

            <div className="flex-1 overflow-y-auto">
              <table className="w-full text-left text-sm">
                <thead className="bg-slate-50 text-slate-600">
                  <tr>
                    <th className="px-6 py-3 font-semibold">Field</th>
                    <th className="px-6 py-3 font-semibold">Type</th>
                    <th className="px-6 py-3 font-semibold">Semantic</th>
                    <th className="px-6 py-3 font-semibold">Status</th>
                    <th className="px-6 py-3 font-semibold text-right">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredFields.map(field => (
                    <tr key={field.key} className="border-b border-slate-100 hover:bg-slate-50">
                      <td className="px-6 py-4">
                                <div className="flex items-center justify-between">
                                          <Button
                                            size="small"
                                            variant="text"
                                            onClick={() => revealFieldInEditor(field.key)}
                                            className="normal-case p-0"
                                            sx={{ textTransform: 'none', color: 'inherit', px: 0 }}
                                          >
                                            <Box sx={{ textAlign: 'left' }}>
                                              <Typography className="font-medium text-slate-800">
                                                {renderHighlightedText(field.displayName)}
                                              </Typography>
                                              <Typography variant="caption" className="text-slate-500">
                                                {renderHighlightedText(field.key)}
                                              </Typography>
                                            </Box>
                                          </Button>
                                </div>
                      </td>
                      <td className="px-6 py-4 text-slate-600">{field.fieldType}</td>
                      <td className="px-6 py-4">
                        <span
                          className={`inline-flex rounded-full px-3 py-1 text-xs font-semibold ${semanticTypeBadge(field.semanticType)}`}
                        >
                          {field.semanticType}
                        </span>
                      </td>
                      <td className="px-6 py-4">
                        <Chip
                          label={field.isCustom ? 'Custom' : 'Core'}
                          size="small"
                          color={field.isCustom ? 'success' : 'primary'}
                        />
                      </td>
                      <td className="px-6 py-4 text-right">
                        <IconButton title="Edit semantic field" size="small" onClick={() => handleOpenEdit(field)}>
                          <Edit fontSize="small" />
                        </IconButton>
                        {field.isCustom && (
                          <IconButton
                            title="Delete custom field"
                            size="small"
                            onClick={() => handleDeleteCustomField(field.key)}
                          >
                            <Delete fontSize="small" />
                          </IconButton>
                        )}
                      </td>
                    </tr>
                  ))}
                  {filteredFields.length === 0 && (
                    <tr>
                      <td colSpan={5} className="px-6 py-8 text-center text-slate-500">
                        No fields match the current filters. Try adjusting the search or switching models.
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </section>

          <section className="w-96 min-w-[320px] border-l border-slate-200 bg-slate-50 flex flex-col">
            <div className="flex items-center justify-between border-b border-slate-200 px-6 py-4">
              <Typography variant="subtitle2" className="font-semibold text-slate-700">
                Semantic Configuration
              </Typography>
              <div className="flex items-center gap-2">
                <ButtonGroup size="small" variant="outlined">
                  <Button
                    variant={codeViewMode === 'yaml' ? 'contained' : 'outlined'}
                    onClick={() => setCodeViewMode('yaml')}
                  >
                    YAML
                  </Button>
                  <Button
                    variant={codeViewMode === 'json' ? 'contained' : 'outlined'}
                    onClick={() => setCodeViewMode('json')}
                  >
                    JSON
                  </Button>
                </ButtonGroup>
                <IconButton size="small" title="Copy config" onClick={copyCurrentConfig}>
                  <ContentCopy className="text-slate-500" />
                </IconButton>
              </div>
            </div>
            <div className="flex-1 overflow-hidden p-4">
              <div className="h-full min-h-[360px] overflow-hidden rounded-lg border border-slate-200 bg-white shadow-inner">
                <MonacoCodeEditor
                  value={monacoValue}
                  language={codeViewMode === 'yaml' ? 'yaml' : 'json'}
                  readOnly
                  highlight={normalizedSearchTerm}
                  onMount={(api: any) => { monacoApiRef.current = api; }}
                />
              </div>
            </div>
          </section>
        </div>
      </main>

      <Dialog open={isDialogOpen} onClose={() => setIsDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Edit Semantic Field</DialogTitle>
        <DialogContent>
          <Stack spacing={3} className="pt-2">
            <TextField label="Field Key" value={editingField?.key || ''} disabled fullWidth size="small" />
            <TextField
              label="Display Name"
              value={editingField?.displayName || ''}
              onChange={event => setEditingField(prev => (prev ? { ...prev, displayName: event.target.value } : prev))}
              fullWidth
              size="small"
            />
            <FormControl fullWidth size="small">
              <InputLabel>Semantic Type</InputLabel>
              <Select
                label="Semantic Type"
                value={editingField?.semanticType || 'dimension'}
                onChange={event =>
                  setEditingField(prev => (prev ? { ...prev, semanticType: event.target.value as SemanticType } : prev))
                }
              >
                <MenuItem value="dimension">Dimension</MenuItem>
                <MenuItem value="measure">Measure</MenuItem>
                <MenuItem value="time">Time Dimension</MenuItem>
              </Select>
            </FormControl>
            <FormControl fullWidth size="small">
              <InputLabel>Field Type</InputLabel>
              <Select
                label="Field Type"
                value={editingField?.fieldType || 'string'}
                onChange={event => setEditingField(prev => (prev ? { ...prev, fieldType: event.target.value } : prev))}
              >
                <MenuItem value="string">String</MenuItem>
                <MenuItem value="number">Number</MenuItem>
                <MenuItem value="date">Date</MenuItem>
                <MenuItem value="datetime">DateTime</MenuItem>
                <MenuItem value="boolean">Boolean</MenuItem>
              </Select>
            </FormControl>
            <TextField
              label="Description"
              value={editingField?.description || ''}
              onChange={event => setEditingField(prev => (prev ? { ...prev, description: event.target.value } : prev))}
              multiline
              minRows={3}
              fullWidth
              size="small"
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setIsDialogOpen(false)}>Cancel</Button>
          <Button onClick={() => isDialogOpen && setIsDialogOpen(false)} variant="contained" color="primary">
            Save Changes
          </Button>
        </DialogActions>
      </Dialog>
    </div>
  );
}
