import React, { useEffect, useState } from 'react';
import { Box, Typography } from '@mui/material';
import { ComponentDefinition, DataSourceDefinition, BusinessObjectDataSourceConfig } from '../../types/pageStudio';
import { fetchBOTerms } from '../../features/query-builder/services/queryBuilderApi';
import type { SemanticTermView } from '../../features/query-builder/types/queryDef';
import ReportWidgetRenderer from '../../components/reporting/ReportWidgetRenderer';
import BOFormWidget from '../../components/reporting/BOFormWidget';

interface PageComponentRendererProps {
  component: ComponentDefinition;
  dataSources: DataSourceDefinition[];
  tenantId: string;
}

const COMPONENT_TO_WIDGET_TYPE: Record<string, string> = {
  Table: 'table',
  LineChart: 'chart',
  KPIGroup: 'gauge',
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
const PageComponentRenderer: React.FC<PageComponentRendererProps> = ({ component, dataSources, tenantId }) => {
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

  if (component.type === 'Form') {
    if (!cfg?.boId) {
      return (
        <Box sx={{ p: 2, textAlign: 'center' }}>
          <Typography variant="caption" color="textSecondary">Add a Business Object data source to bind this form</Typography>
        </Box>
      );
    }
    return <BOFormWidget boId={cfg.boId} tenantId={tenantId} />;
  }

  const widgetType = COMPONENT_TO_WIDGET_TYPE[component.type];
  if (!widgetType) {
    return (
      <Box sx={{ height: 60, bgcolor: 'rgba(0,0,0,0.02)', borderRadius: 1, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <Typography variant="caption" color="textSecondary">Component Preview: {component.type}</Typography>
      </Box>
    );
  }

  if (!cfg?.boId || !cfg?.bindingId) {
    return (
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

  const binding = widgetType === 'gauge'
    ? { boId: cfg.boId, bindingId: cfg.bindingId, tenantId, measures: (measures[0] ? [measures[0]] : dims[0] ? [dims[0]] : []).map((t) => ({ termNodeId: t.termNodeId, alias: t.displayName, agg: measures[0] ? 'SUM' : 'COUNT' })) }
    : widgetType === 'chart'
      ? {
          boId: cfg.boId,
          bindingId: cfg.bindingId,
          tenantId,
          dimensions: dims[0] ? [{ termNodeId: dims[0].termNodeId, alias: dims[0].displayName }] : [],
          measures: measures[0] ? [{ termNodeId: measures[0].termNodeId, alias: measures[0].displayName, agg: 'SUM' }] : [],
          chartType: 'bar' as const,
        }
      : {
          boId: cfg.boId,
          bindingId: cfg.bindingId,
          tenantId,
          dimensions: dims.slice(0, 4).map((t) => ({ termNodeId: t.termNodeId, alias: t.displayName })),
          measures: measures.slice(0, 2).map((t) => ({ termNodeId: t.termNodeId, alias: t.displayName, agg: 'SUM' })),
        };

  if ((binding.dimensions?.length || 0) === 0 && (binding.measures?.length || 0) === 0) {
    return (
      <Box sx={{ p: 2, textAlign: 'center' }}>
        <Typography variant="caption" color="textSecondary">{cfg.displayName} has no resolved fields to display</Typography>
      </Box>
    );
  }

  return <ReportWidgetRenderer type={widgetType} binding={binding} />;
};

export default PageComponentRenderer;
