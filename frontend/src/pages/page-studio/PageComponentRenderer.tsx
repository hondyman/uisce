import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Box, Typography, Dialog, DialogContent, DialogTitle, IconButton, Drawer } from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import { ComponentDefinition, DataSourceDefinition, BusinessObjectDataSourceConfig } from '../../types/pageStudio';
import { fetchBOTerms } from '../../features/query-builder/services/queryBuilderApi';
import type { SemanticTermView } from '../../features/query-builder/types/queryDef';
import ReportWidgetRenderer, { type WidgetStyleOptions } from '../../components/reporting/ReportWidgetRenderer';
import BOFormWidget from '../../components/reporting/BOFormWidget';
import BODirectTable, { TableStyleOptions } from './BODirectTable';
import TextWidget from './TextWidget';
import ButtonWidget from './ButtonWidget';
import FixCommandWidget from './FixCommandWidget';
import HyperlinkWidget from './HyperlinkWidget';
import TileWidget from './TileWidget';
import EmbeddedPageContent from './EmbeddedPageContent';
import { useSelection } from './SelectionContext';
import { usePresentationOverlay, usePresentationRuntime } from './PresentationRuntime';
import { useCrossFilterStore, crossFilterKey } from '../../store/useCrossFilterStore';
import SavedQueryWidget, { SavedQueryParamBinding } from './SavedQueryWidget';
import FormFieldsDesigner, { type FieldLayoutEntry, type FieldOverrideEntry, isFormLikeWidget } from './FormFieldsDesigner';
import { TableDesignPlaceholder, ChartDesignPlaceholder } from './WidgetDesignPlaceholder';

/** Table widget's configurable row-click behavior (component.props.rowClickAction), set via PropertiesPanel. */
export type RowClickAction = 'select' | 'navigate' | 'openModal' | 'openDrawer';

interface PageComponentRendererProps {
  component: ComponentDefinition;
  dataSources: DataSourceDefinition[];
  tenantId: string;
  /** 'design' (Page Studio's canvas) shows structure only, never real record
   * values - a Form renders FormFieldsDesigner instead of BOFormWidget.
   * Every other call site (DraftPreview, PageBrowser, EmbeddedPageContent,
   * the Alpha Query Builder's preview) is 'preview': real, live data,
   * matching what an end user actually sees. */
  mode?: 'design' | 'preview';
  /** Design mode only: which field of a Form is currently selected, and how to change that - PropertiesPanel.tsx reads/drives this via a componentId::field::fieldName selection id. */
  selectedFieldName?: string | null;
  onSelectField?: (fieldName: string | null) => void;
  onUpdateFieldLayout?: (fieldName: string, entry: Partial<FieldLayoutEntry>) => void;
  onReorderFields?: (orderedFieldNames: string[]) => void;
  onUnhideField?: (fieldName: string) => void;
}

const COMPONENT_TO_WIDGET_TYPE: Record<string, string> = {
  Table: 'table',
  LineChart: 'chart',
  KPIGroup: 'gauge',
  Slicer: 'slicer',
};

/**
 * Renders a page-studio component against real data, the same way
 * ReportWidgetRenderer does for Report Builder canvas elements — this was
 * previously a static "Component Preview: {type}" label for every type.
 * Resolves its Business Object binding from the page's dataSources (added
 * via DataBindingsPanel) rather than needing its own separate binding UI;
 * when the component hasn't been pointed at a specific one, it falls back
 * to the page's first bound Business Object, which is the common case for
 * a single-BO page.
 */
// Renders `component.style` (font/color/spacing set via the Style tab in
// PropertiesPanel) as a wrapping Box around whatever the widget itself
// renders - a generic mechanism every widget type gets for free, rather
// than each renderer branch re-implementing style application.
const PageComponentRenderer: React.FC<PageComponentRendererProps> = ({
  component, dataSources, tenantId, mode = 'preview',
  selectedFieldName = null, onSelectField, onUpdateFieldLayout, onReorderFields, onUnhideField,
}) => {
  const overlay = usePresentationOverlay(component.id);
  const { setRecord, setEditingField } = usePresentationRuntime();
  const styled = (children: React.ReactNode) => (
    <Box sx={{ ...(component.style as React.CSSProperties | undefined), ...overlay?.style } as React.CSSProperties}>{children}</Box>
  );

  const { selection, select } = useSelection();
  const navigate = useNavigate();
  const crossFilters = useCrossFilterStore((s) => s.filters);
  const [modalRecordId, setModalRecordId] = useState<string | null>(null);
  const [drawerRecordId, setDrawerRecordId] = useState<string | null>(null);

  const boSources = dataSources.filter((d) => d.type === 'business_object');
  const configured = component.props?.dataSourceId
    ? boSources.find((d) => d.id === component.props!.dataSourceId)
    : undefined;
  const source = configured || boSources[0];
  const cfg = source?.config as unknown as BusinessObjectDataSourceConfig | undefined;

  const [terms, setTerms] = useState<SemanticTermView[]>([]);
  const [termsLoaded, setTermsLoaded] = useState(false);

  useEffect(() => {
    if (!cfg?.boId || !cfg?.bindingId) return;
    let cancelled = false;
    setTermsLoaded(false);
    fetchBOTerms(cfg.boId, cfg.bindingId)
      .then((t) => {
        if (!cancelled) {
          setTerms(t);
          setTermsLoaded(true);
        }
      })
      .catch(() => {
        if (!cancelled) setTermsLoaded(true);
      });
    return () => {
      cancelled = true;
    };
  }, [cfg?.boId, cfg?.bindingId]);

  if (mode !== 'design' && overlay?.hidden) return null;

  if (component.type === 'Text') {
    return styled(<TextWidget component={component} boName={cfg?.boKey} />);
  }

  if (component.type === 'Button') {
    const action = (component.props?.action as string) || 'navigate';
    if (action === 'command' || String(component.props?.command || '').startsWith('fix.')) {
      return styled(<FixCommandWidget component={component} mode={mode} />);
    }
    return styled(<ButtonWidget component={component} />);
  }

  if (component.type === 'FixCommand') {
    return styled(<FixCommandWidget component={component} mode={mode} />);
  }

  if (component.type === 'Hyperlink') {
    return styled(<HyperlinkWidget component={component} />);
  }

  if (component.type === 'Tile') {
    return styled(<TileWidget component={component} />);
  }

  if (isFormLikeWidget(component.type)) {
    if (!cfg?.boId) {
      return styled(
        <Box sx={{ p: 2, textAlign: 'center' }}>
          <Typography variant="caption" color="textSecondary">Add a Business Object data source to bind this form</Typography>
        </Box>
      );
    }
    // Design mode: structure only, never real record values (matching how
    // Salesforce/PeopleSoft page layout editors work) - a resizable,
    // reorderable, click-to-select field grid instead of a live-bound form.
    // DetailPanel is a leftover palette alias of Form (same selected-record
    // editor); keep rendering it so already-placed instances still work.
    if (mode === 'design') {
      return styled(
        <FormFieldsDesigner
          componentId={component.id}
          boId={cfg.boId}
          tenantId={tenantId}
          fieldLayout={component.props?.fieldLayout as Record<string, FieldLayoutEntry> | undefined}
          fieldOverrides={component.props?.fieldOverrides as Record<string, FieldOverrideEntry> | undefined}
          selectedFieldName={selectedFieldName}
          onSelectField={(name) => onSelectField?.(name)}
          onLayoutChange={(name, entry) => onUpdateFieldLayout?.(name, entry)}
          onReorder={(names) => onReorderFields?.(names)}
          onUnhideField={(name) => onUnhideField?.(name)}
        />
      );
    }
    // A Form bound to the same BO as the page's current master-detail
    // selection (and not itself a masterFilter child) shows/edits that one
    // selected record - e.g. an "Order Detail" tab's Form updates the Order
    // row picked in a master Table on another tab, instead of always being
    // a blank create form.
    const formRecordId = !cfg.masterFilter && selection?.boId === cfg.boId ? selection.recordId : undefined;
    return styled(
      <BOFormWidget
        boId={cfg.boId}
        tenantId={tenantId}
        recordId={formRecordId}
        fieldLayout={component.props?.fieldLayout as Record<string, FieldLayoutEntry> | undefined}
        fieldOverrides={component.props?.fieldOverrides as Record<string, FieldOverrideEntry> | undefined}
        readOnly={overlay?.readOnly}
        componentId={component.id}
        onRecordLoaded={setRecord}
        onFieldEdit={(name, value, all) => { setRecord(all); setEditingField(name, value); }}
        onFieldChange={(_name, _value, all) => { setRecord(all); setEditingField(null); }}
      />
    );
  }

  if (component.type === 'Table') {
    if (!cfg?.boId) {
      return styled(
        <Box sx={{ p: 2, textAlign: 'center' }}>
          <Typography variant="caption" color="textSecondary">Add a Business Object data source to bind this Table</Typography>
        </Box>
      );
    }
    if (mode === 'design') {
      const visibleColumns = component.props?.visibleColumns as string[] | undefined;
      const columns = visibleColumns && visibleColumns.length > 0
        ? visibleColumns
        : terms.slice(0, 5).map((t) => t.displayName);
      return styled(<TableDesignPlaceholder title={component.label || cfg.displayName} columns={columns} />);
    }
    // Master-detail: a Table whose data source declares masterFilter is a
    // detail child, scoped to whatever record is currently selected
    // elsewhere on the page (any tab); one with no masterFilter acts as a
    // potential master list - clicking a row either selects that record
    // in-page (the default, for master-detail) or - a List page's row,
    // e.g. an Orders list linking to an Order Detail page - navigates to
    // another page, opens it in a modal, or opens it in a side panel,
    // per rowClickAction/rowClickTargetSlug (set via PropertiesPanel).
    // Harmless to wire on every plain Table even when the page has no
    // detail widgets or link target configured at all.
    const rowClickAction = ((component.props?.rowClickAction as RowClickAction | undefined) || 'select');
    const rowClickTargetSlug = component.props?.rowClickTargetSlug as string | undefined;
    const handleRowClick = (row: Record<string, unknown>) => {
      const recordId = String(row.id ?? '');
      if (rowClickAction !== 'select' && rowClickTargetSlug) {
        if (rowClickAction === 'navigate') { navigate(`/pages/${rowClickTargetSlug}/${recordId}`); return; }
        if (rowClickAction === 'openModal') { setModalRecordId(recordId); return; }
        if (rowClickAction === 'openDrawer') { setDrawerRecordId(recordId); return; }
      }
      select({ boId: cfg.boId, recordId });
      setRecord(row);
    };
    // A master (non-detail) Table also reacts to a Slicer on this page: if
    // any term on this Table's own Business Object has an active
    // cross-filter value (see useCrossFilterStore), scope the table to it -
    // the same "click Status=NEW on the Slicer, the Orders list narrows"
    // behavior Chart/KPI widgets already got for free via ReportWidgetRenderer.
    const activeCrossFilter = !cfg.masterFilter
      ? terms
          .map((t) => ({ term: t, value: crossFilters[crossFilterKey(cfg.boId, t.termNodeId)] }))
          .find((f) => f.value !== undefined)
      : undefined;
    return styled(
      <>
        <BODirectTable
          boId={cfg.boId}
          title={component.label || cfg.displayName}
          tableStyle={component.props?.tableStyle as TableStyleOptions | undefined}
          visibleColumns={component.props?.visibleColumns as string[] | undefined}
          filterField={cfg.masterFilter?.fkField || activeCrossFilter?.term.termKey}
          filterValue={cfg.masterFilter ? selection?.recordId : activeCrossFilter ? String(activeCrossFilter.value) : undefined}
          onRowSelect={!cfg.masterFilter ? handleRowClick : undefined}
          selectedRowId={!cfg.masterFilter && selection?.boId === cfg.boId ? selection.recordId : undefined}
        />
        {rowClickTargetSlug && (
          <Dialog open={!!modalRecordId} onClose={() => setModalRecordId(null)} maxWidth="lg" fullWidth>
            <DialogTitle sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              Details
              <IconButton size="small" onClick={() => setModalRecordId(null)}><CloseIcon fontSize="small" /></IconButton>
            </DialogTitle>
            <DialogContent dividers>
              {modalRecordId && <EmbeddedPageContent slug={rowClickTargetSlug} tenantId={tenantId} recordId={modalRecordId} />}
            </DialogContent>
          </Dialog>
        )}
        {rowClickTargetSlug && (
          <Drawer anchor="right" open={!!drawerRecordId} onClose={() => setDrawerRecordId(null)}>
            <Box sx={{ width: { xs: '100vw', sm: 480 }, p: 3 }}>
              <IconButton size="small" onClick={() => setDrawerRecordId(null)} sx={{ mb: 1 }}><CloseIcon fontSize="small" /></IconButton>
              {drawerRecordId && <EmbeddedPageContent slug={rowClickTargetSlug} tenantId={tenantId} recordId={drawerRecordId} />}
            </Box>
          </Drawer>
        )}
      </>
    );
  }

  const widgetType = COMPONENT_TO_WIDGET_TYPE[component.type];
  if (!widgetType) {
    return styled(
      <Box sx={{ height: 60, bgcolor: 'rgba(0,0,0,0.02)', borderRadius: 1, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <Typography variant="caption" color="textSecondary">Component Preview: {component.type}</Typography>
      </Box>
    );
  }

  const widgetStyle = (
    widgetType === 'chart' ? component.props?.chartStyle
    : widgetType === 'gauge' ? component.props?.kpiStyle
    : widgetType === 'slicer' ? component.props?.slicerStyle
    : undefined
  ) as WidgetStyleOptions | undefined;

  // Saved-query mode (component.props.savedQueryId, set via PropertiesPanel's
  // "Data Source: Use a saved query" toggle) bypasses this widget's own
  // field auto-pick entirely - the saved query already carries its own
  // boId/bindingId/dimensions/measures server-side, so it doesn't even need
  // this component to have its own Business Object data source bound.
  if (component.props?.savedQueryId) {
    if (mode === 'design') {
      return styled(
        <ChartDesignPlaceholder
          kind={widgetType as 'slicer' | 'chart' | 'gauge'}
          title={component.label || 'Saved Query'}
          dimensions={[]}
          measures={[]}
        />
      );
    }
    return styled(
      <Box>
        <Typography variant="subtitle2" fontWeight={700} sx={{ mb: 1 }}>{component.label || 'Saved Query'}</Typography>
        <SavedQueryWidget
          savedQueryId={component.props.savedQueryId as string}
          widgetType={widgetType as 'slicer' | 'chart' | 'gauge'}
          paramBindings={component.props.savedQueryParams as Record<string, SavedQueryParamBinding> | undefined}
          style={widgetStyle}
        />
      </Box>
    );
  }

  if (!cfg?.boId || !cfg?.bindingId) {
    return styled(
      <Box sx={{ p: 2, textAlign: 'center' }}>
        <Typography variant="caption" color="textSecondary">Add a Business Object data source to bind this {component.type}</Typography>
      </Box>
    );
  }

  if (!termsLoaded) {
    return null;
  }

  const dims = terms.filter((t) => t.role === 'DIMENSION');
  const measures = terms.filter((t) => t.role === 'MEASURE' || t.role === 'CALCULATED');

  // Explicit field picks from PropertiesPanel's "Fields" section
  // (component.props.dimensionTermIds/measureTermIds) win when present;
  // falling back to "first dimension/first measure found" otherwise keeps
  // every widget placed before this existed working exactly as before.
  const pickedDimIds = component.props?.dimensionTermIds as string[] | undefined;
  const pickedMeasureIds = component.props?.measureTermIds as string[] | undefined;
  const measureAgg = (component.props?.measureAgg as string | undefined) || 'SUM';
  const chartType = (component.props?.chartType as 'bar' | 'line' | 'pie' | undefined) || 'bar';

  const selectedDims = pickedDimIds && pickedDimIds.length > 0
    ? pickedDimIds.map((id) => dims.find((d) => d.termNodeId === id)).filter((t): t is SemanticTermView => !!t)
    : dims.slice(0, 1);
  const selectedMeasures = pickedMeasureIds && pickedMeasureIds.length > 0
    ? pickedMeasureIds.map((id) => measures.find((m) => m.termNodeId === id)).filter((t): t is SemanticTermView => !!t)
    : measures.slice(0, 1);

  if (mode === 'design') {
    return styled(
      <ChartDesignPlaceholder
        kind={widgetType as 'slicer' | 'chart' | 'gauge'}
        chartType={chartType}
        title={component.label || cfg.displayName}
        dimensions={selectedDims.map((t) => t.displayName)}
        measures={selectedMeasures.map((t) => t.displayName)}
      />
    );
  }

  const binding = widgetType === 'gauge'
    ? {
        boId: cfg.boId, bindingId: cfg.bindingId, tenantId,
        measures: (selectedMeasures.length > 0 ? selectedMeasures : selectedDims.length > 0 ? [selectedDims[0]] : [])
          .map((t) => ({ termNodeId: t.termNodeId, alias: t.displayName, agg: selectedMeasures.length > 0 ? measureAgg : 'COUNT' })),
      }
    : widgetType === 'chart'
      ? {
          boId: cfg.boId,
          bindingId: cfg.bindingId,
          tenantId,
          dimensions: selectedDims.map((t) => ({ termNodeId: t.termNodeId, alias: t.displayName })),
          measures: selectedMeasures.map((t) => ({ termNodeId: t.termNodeId, alias: t.displayName, agg: measureAgg })),
          chartType,
        }
      : {
          // Slicer: filters by exactly the chosen dimension - ReportWidgetRenderer
          // only ever reads dimensions[0], so sending more than one is pointless
          // and used to silently pick whatever term happened to sort first.
          boId: cfg.boId,
          bindingId: cfg.bindingId,
          tenantId,
          dimensions: selectedDims.slice(0, 1).map((t) => ({ termNodeId: t.termNodeId, alias: t.displayName })),
          measures: [],
        };

  const widgetTitle = component.label || cfg.displayName;

  if ((binding.dimensions?.length || 0) === 0 && (binding.measures?.length || 0) === 0) {
    return styled(
      <Box sx={{ p: 2, textAlign: 'center' }}>
        <Typography variant="subtitle2" fontWeight={700}>{widgetTitle}</Typography>
        <Typography variant="caption" color="textSecondary">{cfg.displayName} has no resolved fields to display</Typography>
      </Box>
    );
  }

  return styled(
    <Box>
      <Typography variant="subtitle2" fontWeight={700} sx={{ mb: 1 }}>{widgetTitle}</Typography>
      <ReportWidgetRenderer type={widgetType} binding={binding} style={widgetStyle} />
    </Box>
  );
};

export default PageComponentRenderer;
