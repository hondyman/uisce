import React, { useEffect, useState } from 'react';
import {
  Box, Typography, Paper, Divider, TextField, MenuItem, Stack,
  ToggleButtonGroup, ToggleButton, FormControlLabel, Switch, Checkbox, Button, Tooltip,
  Tabs, Tab, IconButton,
} from '@mui/material';
import DeleteIcon from '@mui/icons-material/Delete';
import { CorePageDefinition, PageLayout, BusinessObjectDataSourceConfig } from '../../types/pageStudio';
import ExpressionEditorField from '../../components/ExpressionBuilder/ExpressionEditorField';
import { fetchBOTerms, fetchBOSchema } from '../../features/query-builder/services/queryBuilderApi';
import type { SemanticTermView, BOSchemaField } from '../../features/query-builder/types/queryDef';
import type { FieldLayoutEntry, FieldOverrideEntry, FieldStyleEntry } from './FormFieldsDesigner';
import { iconForField } from './FormFieldsDesigner';
import { PageStudioApi, PageStudioPage } from '../../api/pageStudio';
import { listSavedQueries } from '../../features/query-builder/services/savedQueryApi';
import type { SavedQuery } from '../../features/query-builder/types/queryDef';
import { apiClient } from '../../utils/apiClient';

/** Mirrors backend/internal/metadata/businessobject_service.go's RelationshipResult - the
 * single centralized source of BO-to-BO structural relationships (real FK column,
 * cardinality, driving tables), also used by DataBindingsPanel.tsx and NewPageWizard.tsx. */
interface BORelationship {
  relatedObjectName: string;
  targetObjectId: string;
  relationshipType: string;
  cardinality: string;
  joinCondition: string;
  sourceDriverTable: string;
  targetDriverTable: string;
}

/** Pulls the child-side column out of a resolved "child.col = parent.col" joinCondition. */
const fkColumnFromJoinCondition = (joinCondition: string): string => {
  const left = joinCondition.split('=')[0]?.trim() || '';
  const dot = left.lastIndexOf('.');
  return dot >= 0 ? left.slice(dot + 1) : left;
};

interface PropertiesPanelProps {
  selectedId: string | null;
  /** Changes the current selection - used after a field-level action (delete, change-field) moves or clears what was selected, so the panel doesn't keep showing stale properties for something that's now hidden or reassigned. */
  onSelectComponent?: (id: string | null) => void;
  draft: CorePageDefinition;
  setDraft: React.Dispatch<React.SetStateAction<CorePageDefinition>>;
  /** The active tab's layout (or draft.layout when the page has no tabs) - see LayoutCanvas.tsx for why this is passed separately rather than always reading draft.layout. */
  layout: PageLayout;
  onLayoutChange: (updater: (layout: PageLayout) => PageLayout) => void;
  tenantId: string;
}

const FONT_FAMILIES = ['inherit', 'Arial, sans-serif', 'Georgia, serif', '"Courier New", monospace', 'Roboto, sans-serif'];
const FONT_WEIGHTS = ['400', '500', '600', '700'];
const TEXT_ALIGNS = ['left', 'center', 'right', 'justify'];

/** One-click Excel "Table Style" swatches - header color + banded-row colors, modeled on Excel's built-in light table style gallery. */
const EXCEL_BAND_PRESETS: { name: string; headerBg: string; headerColor: string; bandColor: string; bandColorAlt: string }[] = [
  { name: 'Blue',   headerBg: '#1a2332', headerColor: '#ffffff', bandColor: '#dbe5f1', bandColorAlt: '#ffffff' },
  { name: 'Orange', headerBg: '#c65911', headerColor: '#ffffff', bandColor: '#fbe5d5', bandColorAlt: '#ffffff' },
  { name: 'Gray',   headerBg: '#595959', headerColor: '#ffffff', bandColor: '#e7e6e6', bandColorAlt: '#ffffff' },
  { name: 'Gold',   headerBg: '#bf8f00', headerColor: '#ffffff', bandColor: '#fff2cc', bandColorAlt: '#ffffff' },
  { name: 'Green',  headerBg: '#375623', headerColor: '#ffffff', bandColor: '#e2efda', bandColorAlt: '#ffffff' },
  { name: 'Purple', headerBg: '#5b2d82', headerColor: '#ffffff', bandColor: '#e6dcf0', bandColorAlt: '#ffffff' },
];

/**
 * Real properties editor for the selected layout node or placed widget -
 * this used to be a stub that only echoed the selected id back (see
 * git history), with a prop signature (`selectedNode`/`onUpdate`) that
 * didn't even match what PageEditor.tsx passes it. Rebuilt against the
 * actual (selectedId, draft, setDraft) contract every other panel here
 * uses, with a Style section (every widget) and a Content section
 * (Text widgets only) that can source its value from a literal or from
 * the same centralized expression engine the Rule Builder uses.
 */
/** Matches LayoutCanvas.tsx's field-selection id convention: `${componentId}::field::${fieldName}`. */
const FIELD_SELECTION_SEP = '::field::';
const parseFieldSelection = (selectedId: string | null): { componentId: string; fieldName: string } | null => {
  if (!selectedId) return null;
  const idx = selectedId.indexOf(FIELD_SELECTION_SEP);
  if (idx < 0) return null;
  return { componentId: selectedId.slice(0, idx), fieldName: selectedId.slice(idx + FIELD_SELECTION_SEP.length) };
};

const PropertiesPanel: React.FC<PropertiesPanelProps> = ({ selectedId, onSelectComponent, draft, setDraft, layout, onLayoutChange, tenantId }) => {
  const fieldSelection = parseFieldSelection(selectedId);
  const componentId = fieldSelection?.componentId ?? selectedId;
  const selectedComponent = componentId ? draft.components?.[componentId] : undefined;

  // The Table's Columns section needs the *actual* BO it's bound to (via
  // props.dataSourceId, same resolution PageComponentRenderer.tsx uses),
  // not just the page's first Business Object data source - a
  // multi-data-source page (several related objects) has one Table per
  // object, each needing its own field list. Declared before the
  // `!selectedId` early return below since hooks can't follow one.
  const boSources = (draft.dataSources || []).filter((d) => d.type === 'business_object');
  // Field-bound widget types: each resolves its Business Object data source
  // the same way PageComponentRenderer.tsx does, so this panel can offer a
  // real field picker for every one of them instead of just Table's columns.
  const FIELD_BOUND_TYPES = ['Table', 'Slicer', 'LineChart', 'KPIGroup', 'Form'];
  const tableSource = selectedComponent && FIELD_BOUND_TYPES.includes(selectedComponent.type)
    ? (selectedComponent.props?.dataSourceId ? boSources.find((d) => d.id === selectedComponent.props!.dataSourceId) : boSources[0])
    : undefined;
  const tableCfg = tableSource?.config as unknown as BusinessObjectDataSourceConfig | undefined;
  const [tableFields, setTableFields] = useState<SemanticTermView[]>([]);
  const dimensionFields = tableFields.filter((t) => t.role === 'DIMENSION');
  const measureFields = tableFields.filter((t) => t.role === 'MEASURE' || t.role === 'CALCULATED');

  // Saved queries for the bound BO - lets a Slicer/Chart/KPI widget source
  // from a reusable, parameterized query (built and saved in the Query
  // Studio / Alpha Query Builder) instead of always auto-picking its own
  // first field. See PageComponentRenderer.tsx's savedQueryId branch for
  // how this actually executes.
  const SAVED_QUERY_TYPES = ['Slicer', 'LineChart', 'KPIGroup'];
  const [savedQueriesForBO, setSavedQueriesForBO] = useState<SavedQuery[]>([]);
  useEffect(() => {
    if (!selectedComponent || !SAVED_QUERY_TYPES.includes(selectedComponent.type) || !tableCfg?.boId) {
      setSavedQueriesForBO([]);
      return;
    }
    let cancelled = false;
    listSavedQueries(tableCfg.boId).then((qs) => { if (!cancelled) setSavedQueriesForBO(qs); }).catch(() => { if (!cancelled) setSavedQueriesForBO([]); });
    return () => { cancelled = true; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedComponent?.type, tableCfg?.boId]);

  // Master-detail candidates for a Table: relationships FROM this table's own
  // bound BO where the FK actually lives on this table's own driving table
  // (i.e. this BO is the "many" side pointing at some other record on the
  // page) - sourced from the same centralized, real-FK-backed endpoint
  // DataBindingsPanel.tsx and NewPageWizard.tsx already use, so a user picks
  // a real relationship instead of typing a column name from memory.
  const [masterDetailOptions, setMasterDetailOptions] = useState<BORelationship[]>([]);
  useEffect(() => {
    if (!selectedComponent || selectedComponent.type !== 'Table' || !tableCfg?.boId) {
      setMasterDetailOptions([]);
      return;
    }
    let cancelled = false;
    apiClient<{ relatedObjects?: BORelationship[] }>(`/business-objects/${tableCfg.boId}/relationships`)
      .then((data) => {
        if (cancelled) return;
        // 'belongs_to', fetched for THIS table's own boId, is specifically
        // "this BO points at a parent" (cardinality 1:N, joinCondition's
        // left side already this BO's own column) - the shape a master-
        // detail child table needs. The endpoint's other rows for this same
        // boId ('foreign_key') describe the mirror image (some other BO
        // pointing at this one), which isn't useful here.
        const rels = (data?.relatedObjects || []).filter(
          (r) => r.relationshipType.toLowerCase() === 'belongs_to' && !!r.joinCondition
        );
        setMasterDetailOptions(rels);
      })
      .catch(() => { if (!cancelled) setMasterDetailOptions([]); });
    return () => { cancelled = true; };
  }, [selectedComponent?.type, tableCfg?.boId]);

  // A Form field's own schema (real displayName/required/type, for defaults
  // when no override is set yet) - only fetched when a field is actually
  // selected (see FIELD_SELECTION_SEP parsing above), against the Form's
  // own bound BO.
  const [fieldSchema, setFieldSchema] = useState<BOSchemaField[]>([]);
  useEffect(() => {
    if (!fieldSelection || !tableCfg?.boId) {
      setFieldSchema([]);
      return;
    }
    let cancelled = false;
    fetchBOSchema(tableCfg.boId, tenantId || 'default')
      .then((s) => { if (!cancelled) setFieldSchema(s.fields); })
      .catch(() => { if (!cancelled) setFieldSchema([]); });
    return () => { cancelled = true; };
  }, [fieldSelection?.componentId, fieldSelection?.fieldName, tableCfg?.boId]);

  // Properties panel used to stack every section (content, master-detail,
  // columns, table style, generic style, layout) in one long scroll - split
  // into tabs so a given widget's editing surface fits without endless
  // scrolling. Reset to the first tab whenever the selection changes so a
  // different widget doesn't inherit e.g. "Style" from the last one.
  const [panelTab, setPanelTab] = useState(0);
  useEffect(() => { setPanelTab(0); }, [selectedId]);

  // Target-page picker for a Table's row-click "Navigate/Modal/Drawer"
  // behaviors - fetched once, lazily, only useful while a Table is selected.
  const [availablePages, setAvailablePages] = useState<PageStudioPage[]>([]);
  useEffect(() => {
    if (selectedComponent?.type !== 'Table') return;
    let cancelled = false;
    PageStudioApi.listPages().then((pages) => { if (!cancelled) setAvailablePages(pages); }).catch(() => { if (!cancelled) setAvailablePages([]); });
    return () => { cancelled = true; };
  }, [selectedComponent?.type]);
  useEffect(() => {
    if (!tableCfg?.boId) { setTableFields([]); return; }
    let cancelled = false;
    fetchBOTerms(tableCfg.boId, tableCfg.bindingId || '')
      .then((terms) => { if (!cancelled) setTableFields(terms); })
      .catch(() => { if (!cancelled) setTableFields([]); });
    return () => { cancelled = true; };
  }, [tableCfg?.boId, tableCfg?.bindingId]);

  if (!selectedId) {
    return (
      <Paper elevation={0} sx={{ p: 2, height: '100%', bgcolor: 'background.paper' }}>
        <Typography variant="body2" color="text.secondary">
          Select a component to edit its properties
        </Typography>
      </Paper>
    );
  }

  const layoutNode = layout?.nodes?.[componentId!];
  const component = selectedComponent;

  if (!layoutNode && !component) {
    return (
      <Paper elevation={0} sx={{ p: 2, height: '100%', bgcolor: 'background.paper' }}>
        <Typography variant="body2" color="text.secondary">Unknown selection</Typography>
      </Paper>
    );
  }

  const style = (layoutNode?.style || component?.style || {}) as Record<string, string>;

  const updateStyle = (patch: Record<string, string>) => {
    if (layoutNode) {
      onLayoutChange((prev) => ({
        ...prev,
        nodes: {
          ...prev.nodes,
          [componentId!]: { ...prev.nodes[componentId!], style: { ...prev.nodes[componentId!].style, ...patch } },
        },
      }));
    } else if (component) {
      setDraft((prev) => ({
        ...prev,
        components: {
          ...prev.components,
          [componentId!]: { ...prev.components[componentId!], style: { ...prev.components[componentId!].style, ...patch } },
        },
      }));
    }
  };

  const updateProps = (patch: Record<string, unknown>) => {
    if (!component) return;
    setDraft((prev) => ({
      ...prev,
      components: {
        ...prev.components,
        [componentId!]: { ...prev.components[componentId!], props: { ...prev.components[componentId!].props, ...patch } },
      },
    }));
  };

  // Master-detail scoping lives on the DATA SOURCE (BusinessObjectDataSourceConfig.masterFilter),
  // not the widget instance, since it describes how that BO's rows relate
  // to the page's selection (see SelectionContext.tsx) rather than
  // anything about this particular Table's presentation.
  const updateTableMasterFilter = (masterFilter: { fkField: string } | undefined) => {
    if (!tableSource) return;
    setDraft((prev) => ({
      ...prev,
      dataSources: (prev.dataSources || []).map((d) =>
        d.id === tableSource.id
          ? { ...d, config: { ...d.config, masterFilter } }
          : d
      ),
    }));
  };

  const updateNodeProps = (patch: Record<string, unknown>) => {
    if (!layoutNode) return;
    onLayoutChange((prev) => ({
      ...prev,
      nodes: {
        ...prev.nodes,
        [componentId!]: { ...prev.nodes[componentId!], props: { ...prev.nodes[componentId!].props, ...patch } },
      },
    }));
  };

  const boSource = (draft.dataSources || []).find((d) => d.type === 'business_object');
  const boKey = (boSource?.config as unknown as BusinessObjectDataSourceConfig | undefined)?.boKey;

  const contentMode = (component?.props?.contentMode as string) || 'literal';

  // A selected Form FIELD (FormFieldsDesigner.tsx tile click) gets its own
  // small, focused editor instead of the whole-widget Properties/Style/
  // Layout tabs below - label, width (as a fraction of the 12-col grid,
  // matching what dragging the tile's resize handle already does), required,
  // and visibility, mirroring how Salesforce/PeopleSoft field-level
  // properties work (separate from the page-layout-level properties).
  if (fieldSelection && component?.type === 'Form') {
    const fieldName = fieldSelection.fieldName;
    const fieldMeta = fieldSchema.find((f) => f.name === fieldName);
    const fieldLayout = (component.props?.fieldLayout as Record<string, FieldLayoutEntry> | undefined)?.[fieldName];
    const fieldOverride = (component.props?.fieldOverrides as Record<string, FieldOverrideEntry> | undefined)?.[fieldName];
    const colSpan = fieldLayout?.colSpan ?? 12;

    const updateFieldLayout = (patch: Partial<FieldLayoutEntry>) => {
      updateProps({
        fieldLayout: {
          ...(component.props?.fieldLayout as Record<string, FieldLayoutEntry> | undefined),
          [fieldName]: { order: fieldLayout?.order ?? 0, colSpan: fieldLayout?.colSpan ?? 12, ...patch },
        },
      });
    };
    const updateFieldOverride = (patch: Partial<FieldOverrideEntry>) => {
      updateProps({
        fieldOverrides: {
          ...(component.props?.fieldOverrides as Record<string, FieldOverrideEntry> | undefined),
          [fieldName]: { ...fieldOverride, ...patch },
        },
      });
    };
    const updateFieldStyle = (patch: Partial<FieldStyleEntry>) => {
      updateFieldOverride({ style: { ...fieldOverride?.style, ...patch } });
    };

    // "Delete" doesn't destroy the field - it's a real column on the
    // Business Object, not something a page layout owns - it hides it from
    // this form and returns focus to the Form itself. See the "Hidden
    // fields" restore list rendered by FormFieldsDesigner.tsx's canvas for
    // how a deleted field comes back.
    const handleDeleteField = () => {
      updateFieldOverride({ hidden: true });
      onSelectComponent?.(componentId);
    };

    // Swaps which schema field occupies THIS slot - same position
    // (order/colSpan) and visual style carry over, but the field's own
    // label/required reset to that new field's own defaults rather than
    // inheriting ones that described the old field. The field this
    // replaces is hidden (not deleted from the schema), same as the
    // delete button above, so it's still recoverable.
    const sameTypeFields = fieldMeta
      ? fieldSchema.filter((f) => f.name !== fieldName && (f.type || 'text') === (fieldMeta.type || 'text'))
      : [];
    const handleChangeField = (newFieldName: string) => {
      const newFieldOverrides = (component.props?.fieldOverrides as Record<string, FieldOverrideEntry> | undefined) || {};
      const newFieldLayouts = (component.props?.fieldLayout as Record<string, FieldLayoutEntry> | undefined) || {};
      updateProps({
        fieldLayout: {
          ...newFieldLayouts,
          [newFieldName]: { order: fieldLayout?.order ?? 0, colSpan: fieldLayout?.colSpan ?? 12 },
        },
        fieldOverrides: {
          ...newFieldOverrides,
          [fieldName]: { ...fieldOverride, hidden: true },
          [newFieldName]: { style: fieldOverride?.style },
        },
      });
      onSelectComponent?.(`${componentId}${FIELD_SELECTION_SEP}${newFieldName}`);
    };

    return (
      <Paper elevation={0} sx={{ height: '100%', bgcolor: 'background.paper', overflowY: 'auto' }}>
        <Box sx={{ p: 2 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1.5 }}>
            <Box>
              <Typography variant="overline" color="text.secondary" fontWeight="bold">Field</Typography>
              <Typography variant="subtitle2">{fieldMeta?.displayName || fieldName}</Typography>
            </Box>
            <Tooltip title="Delete this field">
              <IconButton size="small" onClick={handleDeleteField}>
                <DeleteIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          </Box>

          {sameTypeFields.length > 0 && (
            <TextField
              select
              size="small"
              fullWidth
              label="Change field"
              value=""
              onChange={(e) => handleChangeField(e.target.value)}
              sx={{ mb: 2 }}
              helperText={`Swap in a different ${fieldMeta?.type || 'text'} field, keeping this position and style.`}
            >
              {sameTypeFields.map((f) => {
                const FieldIcon = iconForField(f);
                return (
                  <MenuItem key={f.name} value={f.name} sx={{ display: 'flex', gap: 1 }}>
                    <FieldIcon sx={{ fontSize: 15, color: 'text.secondary' }} />
                    {f.displayName || f.name}
                  </MenuItem>
                );
              })}
            </TextField>
          )}

          <Box sx={{ mb: 2, p: 1.25, borderRadius: 1, bgcolor: 'action.hover' }}>
            <Typography variant="overline" color="text.secondary" sx={{ fontSize: 10, fontWeight: 700 }}>Data Binding</Typography>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', mt: 0.25 }}>
              <Typography variant="caption" color="text.secondary">Column</Typography>
              <Typography variant="caption" fontFamily="monospace">{fieldMeta?.physicalColumn || fieldMeta?.name || fieldName}</Typography>
            </Box>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Typography variant="caption" color="text.secondary">Type</Typography>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                {fieldMeta && (() => { const FieldIcon = iconForField(fieldMeta); return <FieldIcon sx={{ fontSize: 14, color: 'text.secondary' }} />; })()}
                <Typography variant="caption" fontFamily="monospace">{fieldMeta?.type || 'text'}</Typography>
              </Box>
            </Box>
            {fieldMeta?.referenceBoId && (
              <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                <Typography variant="caption" color="text.secondary">References</Typography>
                <Typography variant="caption" fontFamily="monospace">Business Object</Typography>
              </Box>
            )}
            <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5, fontStyle: 'italic' }}>
              Resolved from the Business Object's real schema - use "Change field" above to point this slot at a different field.
            </Typography>
          </Box>

          <TextField
            size="small"
            fullWidth
            label="Label"
            placeholder={fieldMeta?.displayName || fieldName}
            value={fieldOverride?.label ?? ''}
            onChange={(e) => updateFieldOverride({ label: e.target.value || undefined })}
            sx={{ mb: 2 }}
            helperText="Leave blank to use the field's default label."
          />

          <TextField
            select
            size="small"
            fullWidth
            label="Width"
            value={colSpan}
            onChange={(e) => updateFieldLayout({ colSpan: Number(e.target.value) })}
            sx={{ mb: 2 }}
            helperText="Fields with room on the same row sit side by side, same as dragging the tile's right edge."
          >
            <MenuItem value={12}>Full width (12/12)</MenuItem>
            <MenuItem value={6}>Half (6/12)</MenuItem>
            <MenuItem value={4}>One third (4/12)</MenuItem>
            <MenuItem value={3}>One quarter (3/12)</MenuItem>
            <MenuItem value={8}>Two thirds (8/12)</MenuItem>
            <MenuItem value={9}>Three quarters (9/12)</MenuItem>
          </TextField>

          <FormControlLabel
            control={
              <Checkbox
                size="small"
                checked={fieldOverride?.required ?? !!fieldMeta?.required}
                onChange={(e) => updateFieldOverride({ required: e.target.checked })}
              />
            }
            label={<Typography variant="body2">Required</Typography>}
          />
          <Divider sx={{ mb: 2, mt: 2 }} />
          <Typography variant="overline" color="text.secondary" fontWeight="bold">Label Style</Typography>
          <Stack direction="row" spacing={1} sx={{ mt: 1, mb: 1.5 }}>
            <TextField
              select
              size="small"
              label="Size"
              value={fieldOverride?.style?.fontSize || 'medium'}
              onChange={(e) => updateFieldStyle({ fontSize: e.target.value as FieldStyleEntry['fontSize'] })}
              sx={{ flex: 1 }}
            >
              <MenuItem value="small">Small</MenuItem>
              <MenuItem value="medium">Medium</MenuItem>
              <MenuItem value="large">Large</MenuItem>
            </TextField>
            <TextField
              size="small"
              label="Color"
              type="color"
              value={fieldOverride?.style?.color || '#000000'}
              onChange={(e) => updateFieldStyle({ color: e.target.value })}
              sx={{ width: 90 }}
            />
          </Stack>
          <FormControlLabel
            control={
              <Checkbox
                size="small"
                checked={!!fieldOverride?.style?.bold}
                onChange={(e) => updateFieldStyle({ bold: e.target.checked })}
              />
            }
            label={<Typography variant="body2">Bold</Typography>}
          />
        </Box>
      </Paper>
    );
  }

  return (
    <Paper elevation={0} sx={{ height: '100%', bgcolor: 'background.paper', overflowY: 'auto', display: 'flex', flexDirection: 'column' }}>
      <Box sx={{ p: 2, pb: 0 }}>
        <Typography variant="subtitle2" sx={{ mb: 0.5 }}>
          {component ? component.type : layoutNode?.type} <Typography component="span" variant="caption" color="text.secondary">({selectedId})</Typography>
        </Typography>
      </Box>
      <Tabs
        value={panelTab}
        onChange={(_, v) => setPanelTab(v)}
        variant="fullWidth"
        sx={{ minHeight: 36, borderBottom: 1, borderColor: 'divider', mx: 2 }}
      >
        <Tab label="Properties" sx={{ minHeight: 36, py: 0.5, fontSize: 12 }} />
        <Tab label="Style" sx={{ minHeight: 36, py: 0.5, fontSize: 12 }} />
        <Tab label="Layout" sx={{ minHeight: 36, py: 0.5, fontSize: 12 }} />
      </Tabs>
      <Box sx={{ p: 2, flex: 1, overflowY: 'auto' }}>
      {panelTab === 0 && (<>
      {component?.type === 'Text' && (
        <Box sx={{ mb: 3 }}>
          <Typography variant="overline" color="text.secondary" fontWeight="bold">Content</Typography>
          <ToggleButtonGroup
            size="small"
            exclusive
            value={contentMode}
            onChange={(_, v) => v && updateProps({ contentMode: v })}
            sx={{ mt: 1, mb: 1.5, display: 'flex' }}
          >
            <ToggleButton value="literal" sx={{ flex: 1 }}>Static text</ToggleButton>
            <ToggleButton value="expression" sx={{ flex: 1 }}>Dynamic (code)</ToggleButton>
          </ToggleButtonGroup>

          {contentMode === 'literal' ? (
            <TextField
              fullWidth
              multiline
              minRows={2}
              size="small"
              label="Text"
              value={(component.props?.text as string) || ''}
              onChange={(e) => updateProps({ text: e.target.value })}
            />
          ) : (
            <ExpressionEditorField
              label="Expression"
              value={(component.props?.textExpression as string) || ''}
              onChange={(v) => updateProps({ textExpression: v })}
              boName={boKey}
            />
          )}

          <TextField
            select
            fullWidth
            size="small"
            label="Text style"
            value={(component.props?.variant as string) || 'body1'}
            onChange={(e) => updateProps({ variant: e.target.value })}
            sx={{ mt: 1.5 }}
          >
            {['h4', 'h5', 'h6', 'subtitle1', 'body1', 'body2', 'caption'].map((v) => (
              <MenuItem key={v} value={v}>{v}</MenuItem>
            ))}
          </TextField>
        </Box>
      )}

      {component?.type === 'Hyperlink' && (
        <Box sx={{ mb: 3 }}>
          <Typography variant="overline" color="text.secondary" fontWeight="bold">Link</Typography>
          <Stack spacing={1.5} sx={{ mt: 1 }}>
            <TextField
              fullWidth size="small" label="Label"
              value={(component.props?.label as string) || ''}
              onChange={(e) => updateProps({ label: e.target.value })}
            />
            <TextField
              fullWidth size="small" label="Target page slug (internal)"
              placeholder="e.g. order-details"
              value={(component.props?.targetPageSlug as string) || ''}
              onChange={(e) => updateProps({ targetPageSlug: e.target.value })}
            />
            <TextField
              fullWidth size="small" label="External URL (used if no target page)"
              value={(component.props?.href as string) || ''}
              onChange={(e) => updateProps({ href: e.target.value })}
            />
          </Stack>
        </Box>
      )}

      {component?.type === 'Button' && (
        <Box sx={{ mb: 3 }}>
          <Typography variant="overline" color="text.secondary" fontWeight="bold">Button</Typography>
          <Stack spacing={1.5} sx={{ mt: 1 }}>
            <TextField
              fullWidth size="small" label="Label"
              value={(component.props?.label as string) || ''}
              onChange={(e) => updateProps({ label: e.target.value })}
            />
            <TextField
              select fullWidth size="small" label="Action"
              value={(component.props?.action as string) || 'navigate'}
              onChange={(e) => updateProps({ action: e.target.value })}
            >
              <MenuItem value="navigate">Navigate to a page</MenuItem>
              <MenuItem value="openModal">Open a page in a modal</MenuItem>
              <MenuItem value="openUrl">Open an external URL</MenuItem>
            </TextField>
            {((component.props?.action as string) || 'navigate') !== 'openUrl' ? (
              <TextField
                fullWidth size="small" label="Target page slug"
                placeholder="e.g. order-details"
                value={(component.props?.targetPageSlug as string) || ''}
                onChange={(e) => updateProps({ targetPageSlug: e.target.value })}
              />
            ) : (
              <TextField
                fullWidth size="small" label="URL"
                value={(component.props?.url as string) || ''}
                onChange={(e) => updateProps({ url: e.target.value })}
              />
            )}
            <TextField
              select fullWidth size="small" label="Button style"
              value={(component.props?.variant as string) || 'contained'}
              onChange={(e) => updateProps({ variant: e.target.value })}
            >
              <MenuItem value="contained">Contained</MenuItem>
              <MenuItem value="outlined">Outlined</MenuItem>
              <MenuItem value="text">Text</MenuItem>
            </TextField>
          </Stack>
        </Box>
      )}

      {layoutNode?.type === 'Panel' && (
        <Box sx={{ mb: 3 }}>
          <Typography variant="overline" color="text.secondary" fontWeight="bold">Panel</Typography>
          <Stack spacing={1.5} sx={{ mt: 1 }}>
            <ToggleButtonGroup
              size="small"
              exclusive
              value={(layoutNode.props?.side as string) || 'right'}
              onChange={(_, v) => v && updateNodeProps({ side: v })}
              sx={{ display: 'flex' }}
            >
              <ToggleButton value="left" sx={{ flex: 1 }}>Left side</ToggleButton>
              <ToggleButton value="right" sx={{ flex: 1 }}>Right side</ToggleButton>
            </ToggleButtonGroup>
            <TextField
              fullWidth size="small" label="Label (shown on the collapse toggle)"
              value={(layoutNode.props?.label as string) || ''}
              onChange={(e) => updateNodeProps({ label: e.target.value })}
            />
            <TextField
              fullWidth size="small" type="number" label="Width (px)"
              value={(layoutNode.props?.widthPx as number) ?? 320}
              onChange={(e) => updateNodeProps({ widthPx: Number(e.target.value) || 320 })}
            />
            <FormControlLabel
              control={<Switch checked={(layoutNode.props?.collapsible as boolean) ?? true} onChange={(e) => updateNodeProps({ collapsible: e.target.checked })} />}
              label="Collapsible (slides open/closed)"
            />
            <FormControlLabel
              control={<Switch checked={(layoutNode.props?.defaultOpen as boolean) ?? true} onChange={(e) => updateNodeProps({ defaultOpen: e.target.checked })} />}
              label="Open by default"
            />
          </Stack>
        </Box>
      )}

      {component?.type === 'Tile' && (
        <Box sx={{ mb: 3 }}>
          <Typography variant="overline" color="text.secondary" fontWeight="bold">Tile</Typography>
          <Stack spacing={1.5} sx={{ mt: 1 }}>
            <TextField
              fullWidth size="small" label="Title"
              value={(component.props?.title as string) || ''}
              onChange={(e) => updateProps({ title: e.target.value })}
            />
            <TextField
              fullWidth size="small" label="Value (optional)"
              placeholder="e.g. 42, or leave blank"
              value={(component.props?.value as string) || ''}
              onChange={(e) => updateProps({ value: e.target.value })}
            />
            <TextField
              select fullWidth size="small" label="Action"
              value={(component.props?.action as string) || 'navigate'}
              onChange={(e) => updateProps({ action: e.target.value })}
            >
              <MenuItem value="navigate">Navigate to a page</MenuItem>
              <MenuItem value="openModal">Open a page in a modal</MenuItem>
              <MenuItem value="openUrl">Open an external URL</MenuItem>
            </TextField>
            {((component.props?.action as string) || 'navigate') !== 'openUrl' ? (
              <TextField
                fullWidth size="small" label="Target page slug"
                placeholder="e.g. order-details"
                value={(component.props?.targetPageSlug as string) || ''}
                onChange={(e) => updateProps({ targetPageSlug: e.target.value })}
              />
            ) : (
              <TextField
                fullWidth size="small" label="URL"
                value={(component.props?.url as string) || ''}
                onChange={(e) => updateProps({ url: e.target.value })}
              />
            )}
          </Stack>
        </Box>
      )}

      {component && SAVED_QUERY_TYPES.includes(component.type) && (
        <Box sx={{ mb: 3 }}>
          <Typography variant="overline" color="text.secondary" fontWeight="bold">Data Source</Typography>
          <FormControlLabel
            sx={{ display: 'block', mt: 0.5 }}
            control={
              <Switch
                size="small"
                checked={!!component.props?.savedQueryId}
                onChange={(e) => updateProps({ savedQueryId: e.target.checked ? (savedQueriesForBO[0]?.id || '') : undefined, savedQueryParams: undefined })}
                disabled={savedQueriesForBO.length === 0}
              />
            }
            label={<Typography variant="body2">Use a saved query{savedQueriesForBO.length === 0 ? ' (none saved for this Business Object yet - build one in Query Studio)' : ''}</Typography>}
          />
          {!!component.props?.savedQueryId && (
            <Box sx={{ mt: 1.5 }}>
              <TextField
                select size="small" fullWidth label="Saved query"
                value={component.props.savedQueryId as string}
                onChange={(e) => updateProps({ savedQueryId: e.target.value, savedQueryParams: undefined })}
              >
                {savedQueriesForBO.map((sq) => <MenuItem key={sq.id} value={sq.id}>{sq.name}</MenuItem>)}
              </TextField>
              {(() => {
                const sq = savedQueriesForBO.find((q) => q.id === component.props?.savedQueryId);
                const params = sq?.state.parameters || [];
                if (params.length === 0) return null;
                const bindings = (component.props?.savedQueryParams as Record<string, { mode: string; value?: string }> | undefined) || {};
                return (
                  <Stack spacing={1.5} sx={{ mt: 1.5 }}>
                    <Typography variant="caption" color="text.secondary">
                      Parameters — bind each to a fixed value or to whatever record is selected elsewhere on this page.
                    </Typography>
                    {params.map((p) => {
                      const binding = bindings[p.name] || { mode: 'static', value: p.default != null ? String(p.default) : '' };
                      return (
                        <Box key={p.name} sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
                          <Typography variant="caption" sx={{ width: 90, flexShrink: 0 }}>{p.label || p.name}</Typography>
                          <TextField
                            select size="small" sx={{ width: 130 }}
                            value={binding.mode}
                            onChange={(e) => updateProps({ savedQueryParams: { ...bindings, [p.name]: { ...binding, mode: e.target.value } } })}
                          >
                            <MenuItem value="static">Fixed value</MenuItem>
                            <MenuItem value="selection">Page selection</MenuItem>
                          </TextField>
                          {binding.mode === 'static' && (
                            <TextField
                              size="small" sx={{ flex: 1 }} placeholder={p.default != null ? String(p.default) : 'Value'}
                              value={binding.value || ''}
                              onChange={(e) => updateProps({ savedQueryParams: { ...bindings, [p.name]: { ...binding, value: e.target.value } } })}
                            />
                          )}
                        </Box>
                      );
                    })}
                  </Stack>
                );
              })()}
            </Box>
          )}
        </Box>
      )}

      {component?.type === 'Slicer' && !component.props?.savedQueryId && (
        <Box sx={{ mb: 3 }}>
          <Typography variant="overline" color="text.secondary" fontWeight="bold">Filter Field</Typography>
          <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5, mb: 1 }}>
            Filters every other widget on this page bound to the same field and Business Object.
          </Typography>
          {!tableCfg?.boId ? (
            <Typography variant="caption" color="text.secondary">Add a Business Object data source to bind this Slicer first.</Typography>
          ) : dimensionFields.length === 0 ? (
            <Typography variant="caption" color="text.secondary">Loading fields…</Typography>
          ) : (
            <TextField
              select
              size="small"
              fullWidth
              value={(component.props?.dimensionTermIds as string[] | undefined)?.[0] || ''}
              onChange={(e) => updateProps({ dimensionTermIds: e.target.value ? [e.target.value] : [] })}
              helperText="Which field this slicer filters by"
            >
              {dimensionFields.map((f) => <MenuItem key={f.termNodeId} value={f.termNodeId}>{f.displayName}</MenuItem>)}
            </TextField>
          )}
        </Box>
      )}

      {component?.type === 'LineChart' && !component.props?.savedQueryId && (
        <Box sx={{ mb: 3 }}>
          <Typography variant="overline" color="text.secondary" fontWeight="bold">Chart Fields</Typography>
          {!tableCfg?.boId ? (
            <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 1 }}>Add a Business Object data source to bind this chart first.</Typography>
          ) : (dimensionFields.length === 0 && measureFields.length === 0) ? (
            <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 1 }}>Loading fields…</Typography>
          ) : (
            <Stack spacing={1.5} sx={{ mt: 1 }}>
              <TextField
                select size="small" fullWidth label="Category (X axis)"
                value={(component.props?.dimensionTermIds as string[] | undefined)?.[0] || ''}
                onChange={(e) => updateProps({ dimensionTermIds: e.target.value ? [e.target.value] : [] })}
              >
                {dimensionFields.map((f) => <MenuItem key={f.termNodeId} value={f.termNodeId}>{f.displayName}</MenuItem>)}
              </TextField>
              <TextField
                select size="small" fullWidth label="Value (Y axis)"
                value={(component.props?.measureTermIds as string[] | undefined)?.[0] || ''}
                onChange={(e) => updateProps({ measureTermIds: e.target.value ? [e.target.value] : [] })}
              >
                {measureFields.map((f) => <MenuItem key={f.termNodeId} value={f.termNodeId}>{f.displayName}</MenuItem>)}
              </TextField>
              <TextField
                select size="small" fullWidth label="Aggregation"
                value={(component.props?.measureAgg as string) || 'SUM'}
                onChange={(e) => updateProps({ measureAgg: e.target.value })}
              >
                {['SUM', 'AVG', 'MIN', 'MAX', 'COUNT', 'COUNT_DISTINCT'].map((a) => <MenuItem key={a} value={a}>{a}</MenuItem>)}
              </TextField>
              <TextField
                select size="small" fullWidth label="Visual type"
                value={(component.props?.chartType as string) || 'bar'}
                onChange={(e) => updateProps({ chartType: e.target.value })}
              >
                <MenuItem value="bar">Bar</MenuItem>
                <MenuItem value="line">Line</MenuItem>
                <MenuItem value="pie">Pie</MenuItem>
              </TextField>
            </Stack>
          )}
        </Box>
      )}

      {component?.type === 'KPIGroup' && !component.props?.savedQueryId && (
        <Box sx={{ mb: 3 }}>
          <Typography variant="overline" color="text.secondary" fontWeight="bold">KPI Fields</Typography>
          {!tableCfg?.boId ? (
            <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 1 }}>Add a Business Object data source to bind this KPI first.</Typography>
          ) : measureFields.length === 0 ? (
            <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 1 }}>Loading fields…</Typography>
          ) : (
            <Stack spacing={1.5} sx={{ mt: 1 }}>
              <TextField
                select size="small" fullWidth label="Measure"
                value={(component.props?.measureTermIds as string[] | undefined)?.[0] || ''}
                onChange={(e) => updateProps({ measureTermIds: e.target.value ? [e.target.value] : [] })}
              >
                {measureFields.map((f) => <MenuItem key={f.termNodeId} value={f.termNodeId}>{f.displayName}</MenuItem>)}
              </TextField>
              <TextField
                select size="small" fullWidth label="Aggregation"
                value={(component.props?.measureAgg as string) || 'SUM'}
                onChange={(e) => updateProps({ measureAgg: e.target.value })}
              >
                {['SUM', 'AVG', 'MIN', 'MAX', 'COUNT', 'COUNT_DISTINCT'].map((a) => <MenuItem key={a} value={a}>{a}</MenuItem>)}
              </TextField>
            </Stack>
          )}
        </Box>
      )}

      {component?.type === 'Table' && (
        <Box sx={{ mb: 3 }}>
          <Typography variant="overline" color="text.secondary" fontWeight="bold">Master-Detail</Typography>
          <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5, mb: 1 }}>
            By default this table's row clicks make it a "master" list — selecting a row
            makes that record available to other widgets on this page (any tab). Turn
            this on instead to make it a "detail" table that only shows rows related to
            whichever record is currently selected elsewhere on the page.
          </Typography>
          <FormControlLabel
            control={
              <Switch
                size="small"
                checked={!!tableCfg?.masterFilter}
                onChange={(e) => updateTableMasterFilter(e.target.checked ? { fkField: tableCfg?.masterFilter?.fkField || '' } : undefined)}
              />
            }
            label={<Typography variant="body2">Show only rows related to the selected record</Typography>}
          />
          {tableCfg?.masterFilter && (
            masterDetailOptions.length > 0 ? (
              <TextField
                select
                size="small"
                fullWidth
                sx={{ mt: 1 }}
                label="Related to"
                value={masterDetailOptions.find((r) => fkColumnFromJoinCondition(r.joinCondition) === tableCfg.masterFilter?.fkField)?.targetObjectId || ''}
                onChange={(e) => {
                  const rel = masterDetailOptions.find((r) => r.targetObjectId === e.target.value);
                  if (rel) updateTableMasterFilter({ fkField: fkColumnFromJoinCondition(rel.joinCondition) });
                }}
                helperText="Resolved from the real foreign key in the database - the same relationship data Data Binding uses."
              >
                {masterDetailOptions.map((r) => (
                  <MenuItem key={r.targetObjectId} value={r.targetObjectId}>
                    {r.relatedObjectName} ({r.cardinality}, via {fkColumnFromJoinCondition(r.joinCondition)})
                  </MenuItem>
                ))}
              </TextField>
            ) : (
              <TextField
                size="small"
                fullWidth
                sx={{ mt: 1 }}
                label="Foreign key column on this table"
                placeholder="e.g. order_id"
                value={tableCfg.masterFilter.fkField}
                onChange={(e) => updateTableMasterFilter({ fkField: e.target.value })}
                helperText="No real foreign key relationship was found for this Business Object - enter the physical column that references the selected record's id manually."
              />
            )
          )}
        </Box>
      )}

      {component?.type === 'Table' && (
        <Box sx={{ mb: 3 }}>
          <Typography variant="overline" color="text.secondary" fontWeight="bold">Row Click</Typography>
          <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5, mb: 1 }}>
            What happens when someone clicks a row - the classic "list page → detail page" pattern (a List of Orders linking to an Order Detail page), or open the detail inline instead of navigating away.
          </Typography>
          <TextField
            select
            size="small"
            fullWidth
            value={(component.props?.rowClickAction as string) || 'select'}
            onChange={(e) => updateProps({ rowClickAction: e.target.value })}
          >
            <MenuItem value="select">Select record on this page (master-detail)</MenuItem>
            <MenuItem value="navigate">Navigate to another page</MenuItem>
            <MenuItem value="openModal">Open another page in a modal</MenuItem>
            <MenuItem value="openDrawer">Open another page in a side panel</MenuItem>
          </TextField>
          {(component.props?.rowClickAction as string) && (component.props?.rowClickAction as string) !== 'select' && (
            <TextField
              select
              size="small"
              fullWidth
              sx={{ mt: 1.5 }}
              label="Target page"
              value={(component.props?.rowClickTargetSlug as string) || ''}
              onChange={(e) => updateProps({ rowClickTargetSlug: e.target.value })}
              helperText="That page's primary Business Object will be scoped to the row you clicked."
            >
              {availablePages.length === 0 ? (
                <MenuItem value="" disabled>No other pages found</MenuItem>
              ) : (
                availablePages.map((p) => <MenuItem key={p.id} value={p.slug}>{p.name}</MenuItem>)
              )}
            </TextField>
          )}
        </Box>
      )}

      {component?.type === 'Table' && (
        <Box sx={{ mb: 3 }}>
          <Typography variant="overline" color="text.secondary" fontWeight="bold">Columns</Typography>
          <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5, mb: 1 }}>
            Every field's mapping, shown as physical column → semantic label. Uncheck a field to hide it from the table.
          </Typography>
          {!tableCfg?.boId ? (
            <Typography variant="caption" color="text.secondary">Add a Business Object data source to bind this Table first.</Typography>
          ) : tableFields.length === 0 ? (
            <Typography variant="caption" color="text.secondary">Loading fields…</Typography>
          ) : (
            <>
              <Stack direction="row" spacing={1} sx={{ mb: 1 }}>
                <Button size="small" onClick={() => updateProps({ visibleColumns: undefined })}>Show all</Button>
                <Button size="small" onClick={() => updateProps({ visibleColumns: [] })}>Hide all</Button>
              </Stack>
              <Stack sx={{ maxHeight: 240, overflowY: 'auto' }}>
                {tableFields.map((field) => {
                  const configured = component.props?.visibleColumns as string[] | undefined;
                  const checked = !configured || configured.includes(field.termKey);
                  return (
                    <FormControlLabel
                      key={field.termKey}
                      sx={{ ml: 0 }}
                      control={
                        <Checkbox
                          size="small"
                          checked={checked}
                          onChange={(e) => {
                            // No explicit list yet means "show everything" -
                            // the first uncheck has to materialize the full
                            // list (minus this field) rather than starting
                            // from empty, or every other field would vanish too.
                            const base = configured || tableFields.map((f) => f.termKey);
                            const next = e.target.checked
                              ? [...base, field.termKey]
                              : base.filter((k) => k !== field.termKey);
                            updateProps({ visibleColumns: next });
                          }}
                        />
                      }
                      label={
                        <Typography variant="caption">
                          <Typography component="span" variant="caption" color="text.secondary" sx={{ fontFamily: 'monospace' }}>
                            {field.termKey}
                          </Typography>
                          {' → '}{field.displayName}
                        </Typography>
                      }
                    />
                  );
                })}
              </Stack>
            </>
          )}
        </Box>
      )}
      </>)}

      {panelTab === 1 && (<>
      {component?.type === 'Table' && (
        <Box sx={{ mb: 3 }}>
          <Typography variant="overline" color="text.secondary" fontWeight="bold">Table Style</Typography>
          <Stack spacing={1.5} sx={{ mt: 1 }}>
            <FormControlLabel
              control={
                <Switch
                  checked={!!(component.props?.tableStyle as Record<string, unknown> | undefined)?.banded}
                  onChange={(e) => updateProps({ tableStyle: { ...(component.props?.tableStyle as object), banded: e.target.checked } })}
                />
              }
              label="Banded rows (alternating row color, like Excel)"
            />
            {!!(component.props?.tableStyle as Record<string, unknown> | undefined)?.banded && (
              <Box sx={{ pl: 1, borderLeft: '2px solid', borderColor: 'divider' }}>
                <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 0.5 }}>
                  Quick styles
                </Typography>
                <Stack direction="row" spacing={1} sx={{ mb: 1.5, flexWrap: 'wrap', rowGap: 1 }}>
                  {EXCEL_BAND_PRESETS.map((preset) => (
                    <Tooltip key={preset.name} title={preset.name}>
                      <Box
                        onClick={() => updateProps({ tableStyle: { ...(component.props?.tableStyle as object), bandColor: preset.bandColor, bandColorAlt: preset.bandColorAlt, headerBg: preset.headerBg, headerColor: preset.headerColor } })}
                        sx={{
                          width: 28, height: 28, borderRadius: 1, cursor: 'pointer',
                          border: '1px solid', borderColor: 'divider',
                          background: `linear-gradient(135deg, ${preset.headerBg} 50%, ${preset.bandColor} 50%)`,
                          '&:hover': { outline: '2px solid', outlineColor: 'primary.main' },
                        }}
                      />
                    </Tooltip>
                  ))}
                </Stack>
                <TextField
                  fullWidth size="small" type="color" label="Band color (odd rows)"
                  sx={{ mb: 1.5 }}
                  value={(component.props?.tableStyle as Record<string, string> | undefined)?.bandColor || '#f1f5f9'}
                  onChange={(e) => updateProps({ tableStyle: { ...(component.props?.tableStyle as object), bandColor: e.target.value } })}
                />
                <TextField
                  fullWidth size="small" type="color" label="Band color (even rows)"
                  sx={{ mb: 1.5 }}
                  value={(component.props?.tableStyle as Record<string, string> | undefined)?.bandColorAlt || '#ffffff'}
                  onChange={(e) => updateProps({ tableStyle: { ...(component.props?.tableStyle as object), bandColorAlt: e.target.value } })}
                />
              </Box>
            )}
            <FormControlLabel
              control={
                <Switch
                  checked={!!(component.props?.tableStyle as Record<string, unknown> | undefined)?.gridLines}
                  onChange={(e) => updateProps({ tableStyle: { ...(component.props?.tableStyle as object), gridLines: e.target.checked } })}
                />
              }
              label="Grid lines (border around every cell)"
            />
            <FormControlLabel
              control={
                <Switch
                  checked={!!(component.props?.tableStyle as Record<string, unknown> | undefined)?.dense}
                  onChange={(e) => updateProps({ tableStyle: { ...(component.props?.tableStyle as object), dense: e.target.checked } })}
                />
              }
              label="Compact row height"
            />
            <TextField
              fullWidth size="small" type="color" label="Header background"
              value={(component.props?.tableStyle as Record<string, string> | undefined)?.headerBg || '#1a2332'}
              onChange={(e) => updateProps({ tableStyle: { ...(component.props?.tableStyle as object), headerBg: e.target.value } })}
            />
            <TextField
              fullWidth size="small" type="color" label="Header text color"
              value={(component.props?.tableStyle as Record<string, string> | undefined)?.headerColor || '#ffffff'}
              onChange={(e) => updateProps({ tableStyle: { ...(component.props?.tableStyle as object), headerColor: e.target.value } })}
            />
          </Stack>
        </Box>
      )}

      {component?.type === 'LineChart' && (
        <Box sx={{ mb: 3 }}>
          <Typography variant="overline" color="text.secondary" fontWeight="bold">Chart Style</Typography>
          <Stack spacing={1.5} sx={{ mt: 1 }}>
            <TextField
              fullWidth size="small" type="color" label="Series color"
              value={(component.props?.chartStyle as Record<string, string> | undefined)?.color || '#1976d2'}
              onChange={(e) => updateProps({ chartStyle: { ...(component.props?.chartStyle as object), color: e.target.value } })}
            />
            {(component.props?.chartType as string) === 'pie' && (
              <FormControlLabel
                control={
                  <Switch
                    checked={!!(component.props?.chartStyle as Record<string, unknown> | undefined)?.showLegend}
                    onChange={(e) => updateProps({ chartStyle: { ...(component.props?.chartStyle as object), showLegend: e.target.checked } })}
                  />
                }
                label="Show legend"
              />
            )}
          </Stack>
        </Box>
      )}

      {component?.type === 'KPIGroup' && (
        <Box sx={{ mb: 3 }}>
          <Typography variant="overline" color="text.secondary" fontWeight="bold">KPI Style</Typography>
          <Stack spacing={1.5} sx={{ mt: 1 }}>
            <TextField
              fullWidth size="small" type="color" label="Value color"
              value={(component.props?.kpiStyle as Record<string, string> | undefined)?.valueColor || '#000000'}
              onChange={(e) => updateProps({ kpiStyle: { ...(component.props?.kpiStyle as object), valueColor: e.target.value } })}
            />
            <TextField
              fullWidth size="small" type="number" label="Value font size (px)"
              value={(component.props?.kpiStyle as Record<string, number> | undefined)?.valueFontSize || ''}
              onChange={(e) => updateProps({ kpiStyle: { ...(component.props?.kpiStyle as object), valueFontSize: e.target.value ? Number(e.target.value) : undefined } })}
            />
            <TextField
              fullWidth size="small" label="Label (overrides column name)"
              value={(component.props?.kpiStyle as Record<string, string> | undefined)?.label || ''}
              onChange={(e) => updateProps({ kpiStyle: { ...(component.props?.kpiStyle as object), label: e.target.value } })}
            />
          </Stack>
        </Box>
      )}

      {component?.type === 'Slicer' && (
        <Box sx={{ mb: 3 }}>
          <Typography variant="overline" color="text.secondary" fontWeight="bold">Slicer Style</Typography>
          <Stack spacing={1.5} sx={{ mt: 1 }}>
            <TextField
              fullWidth size="small" type="color" label="Active chip color"
              value={(component.props?.slicerStyle as Record<string, string> | undefined)?.activeColor || '#1976d2'}
              onChange={(e) => updateProps({ slicerStyle: { ...(component.props?.slicerStyle as object), activeColor: e.target.value } })}
            />
            <TextField
              select fullWidth size="small" label="Inactive chip style"
              value={(component.props?.slicerStyle as Record<string, string> | undefined)?.variant || 'outlined'}
              onChange={(e) => updateProps({ slicerStyle: { ...(component.props?.slicerStyle as object), variant: e.target.value } })}
            >
              <MenuItem value="outlined">Outlined</MenuItem>
              <MenuItem value="filled">Filled</MenuItem>
            </TextField>
          </Stack>
        </Box>
      )}
      </>)}

      {panelTab === 2 && (<>
      {component && (
        <Box sx={{ mb: 3 }}>
          <Typography variant="overline" color="text.secondary" fontWeight="bold">Layout</Typography>
          <Stack spacing={1.5} sx={{ mt: 1 }}>
            <FormControlLabel
              control={
                <Switch
                  checked={!!style.resize}
                  onChange={(e) => {
                    if (e.target.checked) {
                      updateStyle({ resize: 'both', overflow: 'auto', width: style.width || '100%', height: style.height || '320px' });
                    } else {
                      updateStyle({ resize: '', overflow: '' });
                    }
                  }}
                />
              }
              label="Let viewers drag-resize this widget"
            />
            <TextField
              fullWidth size="small" label="Width (e.g. 400px or 100%)"
              value={style.width || ''}
              onChange={(e) => updateStyle({ width: e.target.value })}
            />
            <TextField
              fullWidth size="small" label="Height (e.g. 320px)"
              value={style.height || ''}
              onChange={(e) => updateStyle({ height: e.target.value })}
            />
          </Stack>
        </Box>
      )}
      </>)}

      {panelTab === 1 && (<>
      <Box sx={{ mb: 1 }}>
        <Typography variant="overline" color="text.secondary" fontWeight="bold">Style</Typography>
        <Stack spacing={1.5} sx={{ mt: 1 }}>
          <TextField
            select
            fullWidth
            size="small"
            label="Font family"
            value={style.fontFamily || 'inherit'}
            onChange={(e) => updateStyle({ fontFamily: e.target.value })}
          >
            {FONT_FAMILIES.map((f) => <MenuItem key={f} value={f}>{f}</MenuItem>)}
          </TextField>

          <TextField
            fullWidth
            size="small"
            label="Font size (px)"
            type="number"
            value={style.fontSize ? parseInt(style.fontSize, 10) : ''}
            onChange={(e) => updateStyle({ fontSize: e.target.value ? `${e.target.value}px` : '' })}
          />

          <TextField
            select
            fullWidth
            size="small"
            label="Font weight"
            value={style.fontWeight || '400'}
            onChange={(e) => updateStyle({ fontWeight: e.target.value })}
          >
            {FONT_WEIGHTS.map((w) => <MenuItem key={w} value={w}>{w}</MenuItem>)}
          </TextField>

          <TextField
            select
            fullWidth
            size="small"
            label="Text align"
            value={style.textAlign || 'left'}
            onChange={(e) => updateStyle({ textAlign: e.target.value })}
          >
            {TEXT_ALIGNS.map((a) => <MenuItem key={a} value={a}>{a}</MenuItem>)}
          </TextField>

          <TextField
            fullWidth
            size="small"
            label="Text color"
            type="color"
            value={style.color || '#000000'}
            onChange={(e) => updateStyle({ color: e.target.value })}
          />

          <FormControlLabel
            control={<Switch checked={!!style.backgroundColor} onChange={(e) => updateStyle({ backgroundColor: e.target.checked ? '#ffffff' : '' })} />}
            label="Background color"
          />
          {!!style.backgroundColor && (
            <TextField
              fullWidth
              size="small"
              type="color"
              value={style.backgroundColor}
              onChange={(e) => updateStyle({ backgroundColor: e.target.value })}
            />
          )}

          <TextField
            fullWidth
            size="small"
            label="Padding (px)"
            type="number"
            value={style.padding ? parseInt(style.padding, 10) : ''}
            onChange={(e) => updateStyle({ padding: e.target.value ? `${e.target.value}px` : '' })}
          />

          <TextField
            fullWidth
            size="small"
            label="Border radius (px)"
            type="number"
            value={style.borderRadius ? parseInt(style.borderRadius, 10) : ''}
            onChange={(e) => updateStyle({ borderRadius: e.target.value ? `${e.target.value}px` : '' })}
          />
        </Stack>
      </Box>
      </>)}
      </Box>
    </Paper>
  );
};

export default PropertiesPanel;
