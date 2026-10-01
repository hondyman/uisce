import React, { useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Grid,
  Card,
  CardActionArea,
  CardContent,
  Typography,
  Box,
  Chip,
  Stack,
  TextField,
  InputAdornment,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import DashboardIcon from '@mui/icons-material/Dashboard';
import AssessmentIcon from '@mui/icons-material/Assessment';
import AccountBalanceIcon from '@mui/icons-material/AccountBalance';
import ShowChartIcon from '@mui/icons-material/ShowChart';
import type { CorePageDefinition, ComponentDefinition } from '../../types/pageStudio';

export interface ReportTemplate {
  id: string;
  name: string;
  category: string;
  description: string;
  icon: React.ReactNode;
  tags: string[];
  parameters: string[];
  tilesCount: number;
  buildDraft: (name: string) => Partial<CorePageDefinition>;
}

export const BUILT_IN_REPORT_TEMPLATES: ReportTemplate[] = [
  {
    id: 'exec-kpi-dashboard',
    name: 'Executive KPI Dashboard',
    category: 'Executive',
    description: 'High-level executive overview with revenue, AUM, key KPIs, parameter bar, and monthly trend visualization.',
    icon: <DashboardIcon color="primary" fontSize="large" />,
    tags: ['Executive', 'KPIs', 'Trends', 'Grid'],
    parameters: ['Date Range', 'Region', 'Status'],
    tilesCount: 6,
    buildDraft: (customName: string) => {
      const pageName = customName || 'Executive KPI Dashboard';
      const slug = `exec-kpi-${Math.random().toString(36).slice(2, 6)}`;
      const components: ComponentDefinition[] = [
        {
          id: 'param-bar',
          type: 'report_parameter_bar',
          config: {
            title: 'Report Parameters',
            variables: ['v_date_range', 'v_region', 'v_status'],
            allowReset: true,
          },
        },
        {
          id: 'kpi-rev',
          type: 'kpi_tile',
          config: {
            title: 'Total Revenue',
            aggregation: 'sum',
            format: 'currency',
            targetValue: 10000000,
            gridSpan: { colSpan: 4, rowSpan: 1 },
          },
        },
        {
          id: 'kpi-aum',
          type: 'kpi_tile',
          config: {
            title: 'Assets Under Management',
            aggregation: 'sum',
            format: 'currency',
            targetValue: 500000000,
            gridSpan: { colSpan: 4, rowSpan: 1 },
          },
        },
        {
          id: 'kpi-acc',
          type: 'kpi_tile',
          config: {
            title: 'Active Accounts',
            aggregation: 'count',
            format: 'number',
            gridSpan: { colSpan: 4, rowSpan: 1 },
          },
        },
        {
          id: 'chart-trend',
          type: 'saved_query_widget',
          config: {
            title: 'Revenue & Inflow Trend',
            chartType: 'combo',
            gridSpan: { colSpan: 8, rowSpan: 2 },
          },
        },
        {
          id: 'slicer-reg',
          type: 'slicer',
          config: {
            dimensionTermId: 'term-region',
            slicerMode: 'checkbox',
            gridSpan: { colSpan: 4, rowSpan: 2 },
          },
        },
      ];

      return {
        name: pageName,
        slug,
        description: 'Executive overview dashboard with KPI tiles and filter parameters',
        layout: {
          root: 'root',
          nodes: {
            root: {
              id: 'root',
              type: 'Grid',
              children: components.map((c) => c.id),
              props: {
                layoutKind: 'grid',
                columns: 12,
                gap: 16,
              },
            },
          },
        },
        components,
        dataSources: [],
        app: {
          variables: [
            { name: 'v_date_range', type: 'string', defaultValue: 'YTD' },
            { name: 'v_region', type: 'string', defaultValue: 'ALL' },
            { name: 'v_status', type: 'string', defaultValue: 'ACTIVE' },
          ],
          queries: [],
        },
      };
    },
  },
  {
    id: 'trading-ledger',
    name: 'Financial & Trading Ledger',
    category: 'Trading',
    description: 'Detailed financial ledger report with trade orders, notional totals, desk parameter controls, and breakdown charts.',
    icon: <AccountBalanceIcon color="secondary" fontSize="large" />,
    tags: ['Trading', 'Ledger', 'Settlement', 'Multi-Desk'],
    parameters: ['Trading Desk', 'Currency', 'Asset Class'],
    tilesCount: 4,
    buildDraft: (customName: string) => {
      const pageName = customName || 'Trading Ledger Report';
      const slug = `trading-ledger-${Math.random().toString(36).slice(2, 6)}`;
      const components: ComponentDefinition[] = [
        {
          id: 'param-bar',
          type: 'report_parameter_bar',
          config: {
            title: 'Trading Filters',
            variables: ['v_desk', 'v_ccy', 'v_asset_class'],
            allowReset: true,
          },
        },
        {
          id: 'kpi-notional',
          type: 'kpi_tile',
          config: {
            title: 'Gross Notional Traded',
            aggregation: 'sum',
            format: 'currency',
            gridSpan: { colSpan: 6, rowSpan: 1 },
          },
        },
        {
          id: 'kpi-trades',
          type: 'kpi_tile',
          config: {
            title: 'Executed Orders',
            aggregation: 'count',
            format: 'number',
            gridSpan: { colSpan: 6, rowSpan: 1 },
          },
        },
        {
          id: 'table-ledger',
          type: 'saved_query_widget',
          config: {
            title: 'Trade Allocations & Executions',
            chartType: 'table',
            gridSpan: { colSpan: 12, rowSpan: 3 },
          },
        },
      ];

      return {
        name: pageName,
        slug,
        description: 'Multi-desk trading activity and order settlement ledger',
        layout: {
          root: 'root',
          nodes: {
            root: {
              id: 'root',
              type: 'Grid',
              children: components.map((c) => c.id),
              props: {
                layoutKind: 'grid',
                columns: 12,
                gap: 16,
              },
            },
          },
        },
        components,
        dataSources: [],
        app: {
          variables: [
            { name: 'v_desk', type: 'string', defaultValue: 'ALL' },
            { name: 'v_ccy', type: 'string', defaultValue: 'USD' },
            { name: 'v_asset_class', type: 'string', defaultValue: 'EQUITY' },
          ],
          queries: [],
        },
      };
    },
  },
  {
    id: 'portfolio-risk',
    name: 'Portfolio Risk & Performance',
    category: 'Risk',
    description: 'Risk analytics dashboard with portfolio beta, tracking error, benchmark comparison, and interactive risk scatter plot.',
    icon: <ShowChartIcon color="warning" fontSize="large" />,
    tags: ['Risk', 'Portfolio', 'Benchmark', 'Scatter'],
    parameters: ['Portfolio', 'Benchmark', 'As-Of Date'],
    tilesCount: 4,
    buildDraft: (customName: string) => {
      const pageName = customName || 'Portfolio Risk Monitor';
      const slug = `port-risk-${Math.random().toString(36).slice(2, 6)}`;
      const components: ComponentDefinition[] = [
        {
          id: 'param-bar',
          type: 'report_parameter_bar',
          config: {
            title: 'Risk Parameters',
            variables: ['v_portfolio', 'v_benchmark', 'v_as_of_date'],
            allowReset: true,
          },
        },
        {
          id: 'kpi-vol',
          type: 'kpi_tile',
          config: {
            title: 'Portfolio Volatility',
            aggregation: 'avg',
            format: 'percentage',
            gridSpan: { colSpan: 6, rowSpan: 1 },
          },
        },
        {
          id: 'kpi-sharpe',
          type: 'kpi_tile',
          config: {
            title: 'Sharpe Ratio',
            aggregation: 'avg',
            format: 'number',
            gridSpan: { colSpan: 6, rowSpan: 1 },
          },
        },
        {
          id: 'chart-risk-scatter',
          type: 'saved_query_widget',
          config: {
            title: 'Risk vs Return Scatter',
            chartType: 'scatter',
            gridSpan: { colSpan: 12, rowSpan: 2 },
          },
        },
      ];

      return {
        name: pageName,
        slug,
        description: 'Multi-factor risk analysis and performance metrics',
        layout: {
          root: 'root',
          nodes: {
            root: {
              id: 'root',
              type: 'Grid',
              children: components.map((c) => c.id),
              props: {
                layoutKind: 'grid',
                columns: 12,
                gap: 16,
              },
            },
          },
        },
        components,
        dataSources: [],
        app: {
          variables: [
            { name: 'v_portfolio', type: 'string', defaultValue: 'GLOBAL_GROWTH' },
            { name: 'v_benchmark', type: 'string', defaultValue: 'SP500' },
            { name: 'v_as_of_date', type: 'string', defaultValue: 'TODAY' },
          ],
          queries: [],
        },
      };
    },
  },
];

export const scaffoldBlankGridReport = (name: string): Partial<CorePageDefinition> => {
  const pageName = name || 'New Grid Report';
  const slug = `report-${Math.random().toString(36).slice(2, 6)}`;
  const components: ComponentDefinition[] = [
    {
      id: 'param-bar-default',
      type: 'report_parameter_bar',
      config: {
        title: 'Report Parameters',
        variables: ['v_date_range'],
        allowReset: true,
      },
    },
    {
      id: 'starter-kpi',
      type: 'kpi_tile',
      config: {
        title: 'Primary Metric',
        gridSpan: { colSpan: 4, rowSpan: 1 },
      },
    },
    {
      id: 'starter-chart',
      type: 'saved_query_widget',
      config: {
        title: 'Overview Chart',
        chartType: 'bar',
        gridSpan: { colSpan: 8, rowSpan: 2 },
      },
    },
  ];

  return {
    name: pageName,
    slug,
    description: 'Grid layout report with parameter bar and KPI tiles',
    layout: {
      root: 'root',
      nodes: {
        root: {
          id: 'root',
          type: 'Grid',
          children: components.map((c) => c.id),
          props: {
            layoutKind: 'grid',
            columns: 12,
            gap: 16,
          },
        },
      },
    },
    components,
    dataSources: [],
    app: {
      variables: [{ name: 'v_date_range', type: 'string', defaultValue: 'YTD' }],
      queries: [],
    },
  };
};

interface ReportTemplateGalleryProps {
  open: boolean;
  onClose: () => void;
  onSelectTemplate: (draft: Partial<CorePageDefinition>) => void;
}

export const ReportTemplateGallery: React.FC<ReportTemplateGalleryProps> = ({
  open,
  onClose,
  onSelectTemplate,
}) => {
  const [search, setSearch] = useState('');
  const [selectedCategory, setSelectedCategory] = useState<string>('ALL');

  const filteredTemplates = BUILT_IN_REPORT_TEMPLATES.filter((tmpl) => {
    const matchCat = selectedCategory === 'ALL' || tmpl.category === selectedCategory;
    const matchSearch =
      !search ||
      tmpl.name.toLowerCase().includes(search.toLowerCase()) ||
      tmpl.description.toLowerCase().includes(search.toLowerCase()) ||
      tmpl.tags.some((t) => t.toLowerCase().includes(search.toLowerCase()));
    return matchCat && matchSearch;
  });

  const handlePick = (tmpl: ReportTemplate) => {
    const draft = tmpl.buildDraft(tmpl.name);
    onSelectTemplate(draft);
    onClose();
  };

  const handleBlankReport = () => {
    const draft = scaffoldBlankGridReport('New Report');
    onSelectTemplate(draft);
    onClose();
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="lg" fullWidth>
      <DialogTitle>
        <Stack direction="row" alignItems="center" justifyContent="space-between">
          <Typography variant="h6" fontWeight={700}>
            Report Template Gallery
          </Typography>
          <Button variant="contained" color="primary" onClick={handleBlankReport}>
            Start Blank Grid Report
          </Button>
        </Stack>
      </DialogTitle>
      <DialogContent dividers>
        <Stack spacing={2} sx={{ mb: 3 }}>
          <TextField
            placeholder="Search templates by name, description, tags, or parameters..."
            size="small"
            fullWidth
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            InputProps={{
              startAdornment: (
                <InputAdornment position="start">
                  <SearchIcon color="action" />
                </InputAdornment>
              ),
            }}
          />
          <Stack direction="row" spacing={1}>
            {['ALL', 'Executive', 'Trading', 'Risk'].map((cat) => (
              <Chip
                key={cat}
                label={cat}
                clickable
                color={selectedCategory === cat ? 'primary' : 'default'}
                onClick={() => setSelectedCategory(cat)}
              />
            ))}
          </Stack>
        </Stack>

        <Grid container spacing={2}>
          {filteredTemplates.map((template) => (
            <Grid key={template.id} size={{ xs: 12, md: 4 }}>
              <Card
                variant="outlined"
                sx={{
                  height: '100%',
                  display: 'flex',
                  flexDirection: 'column',
                  borderRadius: 2,
                  transition: 'box-shadow 0.2s',
                  '&:hover': { boxShadow: 4 },
                }}
              >
                <CardActionArea
                  sx={{ flexGrow: 1, p: 2, display: 'flex', flexDirection: 'column', alignItems: 'flex-start' }}
                  onClick={() => handlePick(template)}
                >
                  <Stack direction="row" spacing={1.5} alignItems="center" sx={{ mb: 1.5, width: '100%' }}>
                    {template.icon}
                    <Box sx={{ flexGrow: 1 }}>
                      <Typography variant="subtitle1" fontWeight={700}>
                        {template.name}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        {template.category} • {template.tilesCount} tiles
                      </Typography>
                    </Box>
                  </Stack>

                  <Typography variant="body2" color="text.secondary" sx={{ mb: 2, flexGrow: 1 }}>
                    {template.description}
                  </Typography>

                  <Box sx={{ width: '100%' }}>
                    <Typography variant="caption" fontWeight={600} color="text.secondary" display="block" sx={{ mb: 0.5 }}>
                      Parameters:
                    </Typography>
                    <Stack direction="row" spacing={0.5} flexWrap="wrap" sx={{ gap: 0.5, mb: 1.5 }}>
                      {template.parameters.map((p) => (
                        <Chip key={p} label={p} size="small" variant="outlined" />
                      ))}
                    </Stack>

                    <Stack direction="row" spacing={0.5} flexWrap="wrap" sx={{ gap: 0.5 }}>
                      {template.tags.map((tag) => (
                        <Chip key={tag} label={`#${tag}`} size="small" sx={{ fontSize: '0.7rem' }} />
                      ))}
                    </Stack>
                  </Box>
                </CardActionArea>
              </Card>
            </Grid>
          ))}
        </Grid>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
      </DialogActions>
    </Dialog>
  );
};
