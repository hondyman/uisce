import React, { useEffect } from 'react';
import { Box, Grid, Typography, Container, Breadcrumbs, Paper } from '@mui/material';
import { PageLayoutSpec, PageComponent } from '../../types/pageDesigner';
import { usePageContextStore } from '../../store/usePageContextStore';
import { EChartRenderer } from './EChartRenderer';
import { DynamicBOGrid } from './DynamicBOGrid';
import { DynamicBOForm } from './DynamicBOForm';
import { LookbackToolbar } from './LookbackToolbar';
import { ScenarioToolbar } from './ScenarioToolbar';
import { AIRecommendationBar } from './AIRecommendationBar';

interface PageRendererProps {
  spec?: PageLayoutSpec;
}

const defaultDemoSpec: PageLayoutSpec = {
  id: 'page_account_master',
  tenant_id: 'core',
  key: 'account_master',
  title: 'Account Master Operations & Analytics',
  domain: 'MDM',
  target_bo_id: 'bo_customer',
  layout: [
    {
      id: 'region_top',
      name: 'Analytics & Listing Region',
      rows: [
        {
          id: 'row_1',
          columns: [
            {
              id: 'col_chart',
              span: 6,
              components: [
                {
                  id: 'comp_chart_region',
                  type: 'BO_ANALYTICS_CHART',
                  title: 'Asset Allocation by Region',
                  bo_id: 'customers',
                  bindings: { dimensions: ['region'], measures: ['balance'] },
                  interactions: {
                    emits_context: [{ source_field: 'region', target_context_key: 'selected_region' }],
                  },
                  config: { chartType: 'bar' },
                },
              ],
            },
            {
              id: 'col_grid',
              span: 6,
              components: [
                {
                  id: 'comp_grid_accounts',
                  type: 'BO_GRID',
                  title: 'Customer Accounts Listing',
                  bo_id: 'customers',
                  bindings: { fields: ['id', 'name', 'region', 'status', 'balance'] },
                  interactions: {
                    emits_context: [
                      { source_field: 'id', target_context_key: 'selected_account_id' },
                      { source_field: 'status', target_context_key: 'selected_account_status' },
                    ],
                    subscribes_to_context: [
                      { context_key: 'selected_region', filter_field: 'region', operator: 'EQ' },
                    ],
                  },
                  config: {},
                },
              ],
            },
          ],
        },
      ],
    },
    {
      id: 'region_bottom',
      name: 'Detail Form Region',
      rows: [
        {
          id: 'row_2',
          columns: [
            {
              id: 'col_form',
              span: 12,
              components: [
                {
                  id: 'comp_form_detail',
                  type: 'BO_FORM',
                  title: 'Account Record Master Detail',
                  bo_id: 'customers',
                  bindings: { fields: ['id', 'name', 'region', 'status', 'notes'] },
                  config: { is_mutable: true },
                },
              ],
            },
          ],
        },
      ],
    },
  ],
  rules: [
    {
      id: 'rule_disable_pending',
      name: 'Disable form if status is PENDING',
      condition: { field: 'selected_account_status', operator: 'EQUALS', value: 'PENDING' },
      actions: [{ target_component_id: 'comp_form_detail', effect: 'DISABLE' }],
    },
  ],
};

export const PageRenderer: React.FC<PageRendererProps> = ({ spec = defaultDemoSpec }) => {
  const contextMap = usePageContextStore((state) => state.contextMap);
  const applyRuleOverrides = usePageContextStore((state) => state.applyRuleOverrides);

  // Evaluate Page Governance Rules on every context store change
  useEffect(() => {
    const overrides: Record<string, { hidden?: boolean; disabled?: boolean; readOnly?: boolean }> = {};

    spec.rules.forEach((rule) => {
      const currentVal = contextMap[rule.condition.field];
      let matches = false;

      if (rule.condition.operator === 'EQUALS') {
        matches = currentVal === rule.condition.value;
      }

      if (matches) {
        rule.actions.forEach((action) => {
          if (action.target_component_id) {
            overrides[action.target_component_id] = {
              ...overrides[action.target_component_id],
              disabled: action.effect === 'DISABLE',
              hidden: action.effect === 'HIDE',
              readOnly: action.effect === 'READ_ONLY',
            };
          }
        });
      }
    });

    applyRuleOverrides(overrides);
  }, [contextMap, spec.rules, applyRuleOverrides]);

  const renderComponent = (comp: PageComponent) => {
    switch (comp.type) {
      case 'BO_ANALYTICS_CHART':
        return <EChartRenderer key={comp.id} component={comp} />;
      case 'BO_GRID':
        return <DynamicBOGrid key={comp.id} component={comp} />;
      case 'BO_FORM':
        return <DynamicBOForm key={comp.id} component={comp} />;
      case 'BO_LOOKBACK_MANAGER':
        return (
          <Paper key={comp.id} sx={{ p: 3, bgcolor: '#1e293b', border: '1px solid #facc15' }}>
            <Typography variant="h6" fontWeight="600" color="#facc15" mb={1}>
              {comp.title} (AI Lookback Manager Active)
            </Typography>
            <Typography variant="body2" color="#94a3b8" mb={2}>
              Point-in-time diff matrix automatically generated for Business Object '{comp.bo_id}'.
            </Typography>
          </Paper>
        );
      case 'AI_RECOMMENDATION_BAR':
        return <AIRecommendationBar key={comp.id} boKeys={[comp.bo_id || 'Customer']} />;
      default:
        return (
          <Box key={comp.id} p={2} sx={{ bgcolor: '#1e293b', color: '#fff', border: '1px solid #334155' }}>
            <Typography variant="subtitle1">{comp.title}</Typography>
          </Box>
        );
    }
  };

  return (
    <Box sx={{ p: 4, bgcolor: '#0f172a', minHeight: '100vh', color: '#f8fafc' }}>
      <Breadcrumbs sx={{ mb: 2, color: '#94a3b8' }}>
        <Typography color="#94a3b8">Page Runtime Engine</Typography>
        <Typography color="#38bdf8">{spec.domain}</Typography>
      </Breadcrumbs>
      <Typography variant="h4" fontWeight="700" mb={3}>
        {spec.title}
      </Typography>

      <ScenarioToolbar />
      <LookbackToolbar />

      {spec.layout.map((region) => (
        <Box key={region.id} mb={4}>
          {region.rows.map((row) => (
            <Grid container spacing={3} key={row.id} mb={3}>
              {row.columns.map((col) => (
                <Grid key={col.id} size={{ xs: 12, md: col.span }}>
                  {col.components.map((comp) => renderComponent(comp))}
                </Grid>
              ))}
            </Grid>
          ))}
        </Box>
      ))}
    </Box>
  );
};
