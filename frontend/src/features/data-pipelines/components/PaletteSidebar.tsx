import React, { useState } from 'react';
import {
  Box,
  Typography,
  TextField,
  InputAdornment,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Chip,
  useTheme,
} from '@mui/material';
import {
  Search,
  ChevronDown,
  Database,
  ArrowRightLeft,
  ShieldCheck,
  Share2,
  FileText,
  Filter,
  Network,
  Award,
  HardDrive,
  Globe,
  GitBranch,
  Workflow,
  Code2,
  Edit3,
  Layers,
  Terminal,
  Zap,
  LineChart,
} from 'lucide-react';
import { TileCategory } from '../types/pipeline';

interface PaletteItem {
  id: string;
  label: string;
  category: TileCategory;
  subType: string;
  iconName: string;
  description: string;
  badge: string;
  defaultConfig: Record<string, any>;
}

const PALETTE_CATEGORIES: { name: string; category: TileCategory; items: PaletteItem[] }[] = [
  {
    name: 'Uuisce Platform Readers',
    category: 'source',
    items: [
      {
        id: 'bo_reader',
        label: 'Tenant Business Object Reader',
        category: 'source',
        subType: 'bo_reader',
        iconName: 'Database',
        description: 'Extracts streaming records from OMS, AltInvest, CashFlow, Master Directory STI tables',
        badge: 'Mode 1 (STI)',
        defaultConfig: { sourceType: 'bo_reader', table: 'oms.account', subtype_code: 'institutional', limit: 5000 },
      },
      {
        id: 'catalog_reader',
        label: 'Catalog Graph Asset Reader',
        category: 'source',
        subType: 'catalog_reader',
        iconName: 'Share2',
        description: 'Extracts catalog_node and catalog_edge metadata filtered by type & Gold Copy',
        badge: 'Mode 2 (Graph)',
        defaultConfig: { sourceType: 'catalog_reader', catalog_type: 'TABLE', limit: 2000 },
      },
    ],
  },
  {
    name: 'External Connectors',
    category: 'source',
    items: [
      {
        id: 'raw_json',
        label: 'Raw JSON / File Feed',
        category: 'source',
        subType: 'raw_json',
        iconName: 'FileText',
        description: 'Ingests CSV, JSON, or tabular batch payloads directly into pipeline',
        badge: 'Feed',
        defaultConfig: { sourceType: 'raw_json', raw_data: [] },
      },
      {
        id: 'rest_source',
        label: 'REST API / Webhook Pull',
        category: 'source',
        subType: 'raw_json',
        iconName: 'Globe',
        description: 'Extracts JSON payloads from external HTTP endpoints',
        badge: 'HTTP',
        defaultConfig: { sourceType: 'raw_json', url: 'https://api.example.com/trades' },
      },
    ],
  },
  {
    name: 'API Builder & Workflow Orchestration',
    category: 'transform',
    items: [
      {
        id: 'api_builder_caller',
        label: 'API Builder Endpoint Invoker',
        category: 'transform',
        subType: 'api_caller',
        iconName: 'Code2',
        description: 'Calls REST APIs designed & published in API Studio / API Inventory',
        badge: 'API Studio',
        defaultConfig: {
          endpoint_url: '/api/v1/customers/verify-kyc',
          method: 'POST',
          merge_output: true,
          target_field: '_api_response',
        },
      },
      {
        id: 'flow_builder_invoker',
        label: 'Flow Builder Workflow Trigger',
        category: 'transform',
        subType: 'workflow_caller',
        iconName: 'Workflow',
        description: 'Triggers Flow Builder / Temporal business workflows & approval pipelines',
        badge: 'Flow Builder',
        defaultConfig: {
          workflow_name: 'Trade Reconciliation & Settlement Approval',
          workflow_id: 'wf-1',
          mode: 'sync',
        },
      },
    ],
  },
  {
    name: 'Business Objects CRUD Operations',
    category: 'transform',
    items: [
      {
        id: 'bo_crud_operator',
        label: 'Business Object CRUD Operator',
        category: 'transform',
        subType: 'bo_crud',
        iconName: 'Edit3',
        description: 'Performs Create, Read, Update, or Soft-Delete against STI Business Objects',
        badge: 'BO CRUD',
        defaultConfig: {
          table: 'oms.account',
          operation: 'UPDATE',
        },
      },
    ],
  },
  {
    name: 'Transformations & Normalizers',
    category: 'transform',
    items: [
      {
        id: 'column_mapper',
        label: 'Column Mapper & Cast',
        category: 'transform',
        subType: 'column_mapper',
        iconName: 'ArrowRightLeft',
        description: 'Renames incoming column keys and casts data types (float, int, uuid, date)',
        badge: 'Mapper',
        defaultConfig: { mappings: {}, types: {} },
      },
      {
        id: 'filter_gate',
        label: 'Filter & Condition Gate',
        category: 'transform',
        subType: 'filter',
        iconName: 'Filter',
        description: 'Filters rows by equality, numerical thresholds, or not-null constraints',
        badge: 'Filter',
        defaultConfig: { field: '', operator: 'eq', value: '' },
      },
      {
        id: 'subtype_allowlist',
        label: 'Subtype Allowlist Enforcer',
        category: 'validator',
        subType: 'subtype_allowlist',
        iconName: 'ShieldCheck',
        description: 'Enforces STI subtype allowlist rules defined in oms.subtype_registry',
        badge: 'Rule 1 Invariant',
        defaultConfig: { root_object: 'account' },
      },
      {
        id: 'graph_synthesizer',
        label: 'Graph Node & Edge Synthesizer',
        category: 'graph_synthesizer',
        subType: 'graph_synthesizer',
        iconName: 'Network',
        description: 'Converts tabular data into Catalog TABLE, ATTRIBUTE nodes & relationship edges',
        badge: 'Mode 2 (Graph)',
        defaultConfig: { parent_field: 'table_name', child_field: 'column_name', data_type_field: 'data_type', edge_predicate: 'COLUMN_OF' },
      },
      {
        id: 'bloomberg_fields_mapper',
        label: 'Bloomberg Fields Dictionary Mapper',
        category: 'transform',
        subType: 'bloomberg_field_mapper',
        iconName: 'LineChart',
        description: 'Maps bb_fields.csv records into BLOOMBERG_FIELD catalog nodes with JSON properties & market sector eligibility',
        badge: 'Data License',
        defaultConfig: { category_prefix: 'bloomberg.fields' },
      },
    ],
  },
  {
    name: 'Uuisce Platform Ingestors (Loaders)',
    category: 'loader',
    items: [
      {
        id: 'bo_loader',
        label: 'Uuisce BO Bulk Loader',
        category: 'loader',
        subType: 'bo_loader',
        iconName: 'Database',
        description: 'Parallel bulk upsert with bitemporal valid_from / valid_to support',
        badge: 'Mode 1 (STI)',
        defaultConfig: { loaderType: 'bo_loader', table: 'oms.trade_order' },
      },
      {
        id: 'catalog_loader',
        label: 'Uuisce Catalog Graph Ingestor',
        category: 'loader',
        subType: 'catalog_loader',
        iconName: 'Layers',
        description: 'Parallel upsert into catalog_node and catalog_edge respecting Gold Copy delta rules',
        badge: 'Mode 2 (Graph)',
        defaultConfig: { loaderType: 'catalog_loader' },
      },
    ],
  },
];

const renderIcon = (iconName: string, color: string) => {
  const props = { size: 16, color, style: { marginRight: 8, flexShrink: 0 } };
  switch (iconName) {
    case 'Database': return <Database {...props} />;
    case 'Share2': return <Share2 {...props} />;
    case 'FileText': return <FileText {...props} />;
    case 'Globe': return <Globe {...props} />;
    case 'Code2': return <Code2 {...props} />;
    case 'Workflow': return <Workflow {...props} />;
    case 'Edit3': return <Edit3 {...props} />;
    case 'ArrowRightLeft': return <ArrowRightLeft {...props} />;
    case 'Filter': return <Filter {...props} />;
    case 'ShieldCheck': return <ShieldCheck {...props} />;
    case 'Network': return <Network {...props} />;
    case 'Layers': return <Layers {...props} />;
    case 'LineChart': return <LineChart {...props} />;
    default: return <Zap {...props} />;
  }
};

export const PaletteSidebar: React.FC = () => {
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';
  const [searchTerm, setSearchTerm] = useState('');

  const onDragStart = (event: React.DragEvent, item: PaletteItem) => {
    const nodeData = {
      label: item.label,
      category: item.category,
      subType: item.subType,
      icon: item.iconName,
      description: item.description,
      badge: item.badge,
      config: { ...item.defaultConfig },
    };
    event.dataTransfer.setData('application/reactflow', JSON.stringify(nodeData));
    event.dataTransfer.effectAllowed = 'move';
  };

  const filteredCategories = PALETTE_CATEGORIES.map((cat) => ({
    ...cat,
    items: cat.items.filter(
      (item) =>
        item.label.toLowerCase().includes(searchTerm.toLowerCase()) ||
        item.description.toLowerCase().includes(searchTerm.toLowerCase()) ||
        item.badge.toLowerCase().includes(searchTerm.toLowerCase())
    ),
  })).filter((cat) => cat.items.length > 0);

  return (
    <Box
      sx={{
        width: 320,
        height: '100%',
        borderRight: `1px solid ${theme.palette.divider}`,
        backgroundColor: isDark ? theme.palette.background.default : '#f8fafc',
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
      }}
    >
      {/* Header */}
      <Box
        sx={{
          p: 2,
          borderBottom: `1px solid ${theme.palette.divider}`,
          backgroundColor: isDark ? theme.palette.background.paper : '#ffffff',
        }}
      >
        <Typography variant="h6" sx={{ fontSize: '0.95rem', fontWeight: 800, color: theme.palette.text.primary }}>
          Pipeline Component Palette
        </Typography>
        <Typography variant="caption" sx={{ color: theme.palette.text.secondary }}>
          Drag tiles onto canvas to construct enterprise parallel DAGs
        </Typography>

        <TextField
          size="small"
          placeholder="Search operators, APIs, workflows..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          fullWidth
          sx={{
            mt: 1.5,
            '& .MuiOutlinedInput-root': {
              backgroundColor: isDark ? 'rgba(255, 255, 255, 0.05)' : '#ffffff',
            },
          }}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <Search size={16} color={isDark ? '#94a3b8' : '#64748b'} />
              </InputAdornment>
            ),
          }}
        />
      </Box>

      {/* Accordions */}
      <Box sx={{ flex: 1, overflowY: 'auto', p: 1.5 }}>
        {filteredCategories.map((cat, idx) => (
          <Accordion
            key={cat.name}
            defaultExpanded={true}
            disableGutters
            elevation={0}
            sx={{
              backgroundColor: 'transparent',
              '&:before': { display: 'none' },
              mb: 1.5,
            }}
          >
            <AccordionSummary
              expandIcon={<ChevronDown size={16} color={isDark ? '#94a3b8' : '#475569'} />}
              sx={{
                minHeight: 36,
                px: 1,
                py: 0.5,
                borderRadius: '6px',
                backgroundColor: isDark ? theme.palette.background.paper : '#ffffff',
                border: `1px solid ${theme.palette.divider}`,
                '& .MuiAccordionSummary-content': { my: 0.5 },
              }}
            >
              <Typography variant="subtitle2" sx={{ fontWeight: 700, fontSize: '0.8rem', color: theme.palette.text.primary }}>
                {cat.name} ({cat.items.length})
              </Typography>
            </AccordionSummary>

            <AccordionDetails sx={{ px: 0, pt: 1, pb: 0 }}>
              {cat.items.map((item) => (
                <Box
                  key={item.id}
                  draggable
                  onDragStart={(e) => onDragStart(e, item)}
                  sx={{
                    p: 1.2,
                    mb: 1,
                    backgroundColor: isDark ? theme.palette.background.paper : '#ffffff',
                    borderRadius: '8px',
                    border: `1px solid ${theme.palette.divider}`,
                    cursor: 'grab',
                    transition: 'all 0.15s ease',
                    boxShadow: isDark ? '0 1px 4px rgba(0,0,0,0.3)' : '0 1px 3px rgba(0,0,0,0.02)',
                    '&:hover': {
                      borderColor: theme.palette.primary.main,
                      boxShadow: isDark
                        ? '0 4px 12px rgba(59, 130, 246, 0.25)'
                        : '0 4px 10px rgba(59, 130, 246, 0.12)',
                      transform: 'translateY(-1px)',
                    },
                    '&:active': {
                      cursor: 'grabbing',
                    },
                  }}
                >
                  <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 0.5 }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', overflow: 'hidden' }}>
                      {renderIcon(item.iconName, isDark ? '#60a5fa' : '#2563eb')}
                      <Typography variant="subtitle2" sx={{ fontSize: '0.8rem', fontWeight: 700, color: theme.palette.text.primary }} noWrap>
                        {item.label}
                      </Typography>
                    </Box>
                    <Chip
                      label={item.badge}
                      size="small"
                      sx={{
                        height: 18,
                        fontSize: '0.6rem',
                        fontWeight: 700,
                        backgroundColor: isDark ? 'rgba(255, 255, 255, 0.08)' : '#f1f5f9',
                        color: theme.palette.text.secondary,
                        flexShrink: 0,
                        ml: 0.5,
                      }}
                    />
                  </Box>
                  <Typography variant="caption" sx={{ color: theme.palette.text.secondary, fontSize: '0.7rem', display: 'block', lineHeight: 1.3 }}>
                    {item.description}
                  </Typography>
                </Box>
              ))}
            </AccordionDetails>
          </Accordion>
        ))}
      </Box>
    </Box>
  );
};
