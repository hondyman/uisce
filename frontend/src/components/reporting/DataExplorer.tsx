import React, { useState, useCallback, useMemo } from 'react';
import {
  Box,
  Typography,
  Button,
  TextField,
  IconButton,
  Chip,
  Paper,
  Collapse,
  Avatar,
  Divider,
  ToggleButtonGroup,
  ToggleButton,
  InputAdornment,
  Switch,
} from '@mui/material';
import {
  PlayArrow as PlayIcon,
  Save as SaveIcon,
  Download as DownloadIcon,
  Code as CodeIcon,
  Search as SearchIcon,
  Add as AddIcon,
  Close as CloseIcon,
  ExpandMore as ExpandMoreIcon,
  FilterAlt as FilterIcon,
  ShowChart as LineChartIcon,
  BarChart as BarChartIcon,
  PieChart as PieChartIcon,
  ChevronLeft as ChevronLeftIcon,
  ChevronRight as ChevronRightIcon,
  Dataset as DatasetIcon,
  Edit as EditIcon,
} from '@mui/icons-material';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip as RechartsTooltip, ResponsiveContainer } from 'recharts';

// ============================================================================
// CUBE.DEV STYLE QUERY BUILDER
// Yellow accent theme, collapsible schema, filter pills, inline results
// ============================================================================

interface CubeMember {
  name: string;
  display_name: string;
  type: string;
  icon: 'measure' | 'dimension' | 'time';
}

interface QueryFilter {
  id: string;
  member: string;
  operator: string;
  values: string[];
}

const SAMPLE_SCHEMA = {
  Orders: [
    { name: 'total_revenue', display_name: 'Total Revenue', type: 'sum', icon: 'measure' as const },
    { name: 'order_count', display_name: 'Order Count', type: 'count', icon: 'measure' as const },
    { name: 'status', display_name: 'Status', type: 'string', icon: 'dimension' as const },
    { name: 'created_at', display_name: 'Created At', type: 'time', icon: 'time' as const },
  ],
  Users: [
    { name: 'user_count', display_name: 'User Count', type: 'count', icon: 'measure' as const },
    { name: 'city', display_name: 'City', type: 'string', icon: 'dimension' as const },
    { name: 'country', display_name: 'Country', type: 'string', icon: 'dimension' as const },
  ],
};

const SAMPLE_DATA = [
  { date: '2023-10-24', city: 'San Francisco', status: 'Completed', revenue: 1240 },
  { date: '2023-10-23', city: 'New York', status: 'Completed', revenue: 3450.5 },
  { date: '2023-10-23', city: 'Austin', status: 'Processing', revenue: 890 },
  { date: '2023-10-22', city: 'London', status: 'Completed', revenue: 2100 },
  { date: '2023-10-22', city: 'Berlin', status: 'Cancelled', revenue: 0 },
];

const CHART_DATA = [
  { x: 0, y: 45 },
  { x: 20, y: 35 },
  { x: 40, y: 25 },
  { x: 60, y: 15 },
  { x: 80, y: 20 },
  { x: 100, y: 5 },
];

export const DataExplorer: React.FC = () => {
  const [queryName, setQueryName] = useState('Untitled Query');
  const [isEditingName, setIsEditingName] = useState(false);
  const [schemaSearch, setSchemaSearch] = useState('');
  const [expandedSections, setExpandedSections] = useState<Record<string, boolean>>({ Orders: true, Users: true });
  const [filters, setFilters] = useState<QueryFilter[]>([]);
  const [viewMode, setViewMode] = useState<'chart' | 'table' | 'json'>('table');
  const [chartType, setChartType] = useState<'line' | 'bar' | 'pie'>('line');
  const [activeTab, setActiveTab] = useState<'results' | 'sql' | 'rest' | 'graphql'>('results');

  const toggleSection = (section: string) => {
    setExpandedSections(prev => ({ ...prev, [section]: !prev[section] }));
  };

  const removeFilter = (id: string) => {
    setFilters(prev => prev.filter(f => f.id !== id));
  };

  const getFieldIcon = (type: 'measure' | 'dimension' | 'time') => {
    const iconProps = { sx: { fontSize: 18 } };
    switch (type) {
      case 'measure':
        return <Typography sx={{ fontSize: 18, fontWeight: 'bold', color: '#f97316' }}>123</Typography>;
      case 'time':
        return <Typography sx={{ fontSize: 18, color: '#a855f7' }}>📅</Typography>;
      default:
        return <Typography sx={{ fontSize: 18, fontWeight: 'bold', color: '#3b82f6' }}>abc</Typography>;
    }
  };

  const getStatusChip = (status: string) => {
    const colors: Record<string, { bg: string; text: string }> = {
      Completed: { bg: '#d1fae5', text: '#065f46' },
      Processing: { bg: '#fef3c7', text: '#92400e' },
      Cancelled: { bg: '#f3f4f6', text: '#374151' },
    };
    const color = colors[status] || colors.Cancelled;
    return (
      <Chip
        label={status}
        size="small"
        sx={{
          bgcolor: color.bg,
          color: color.text,
          fontWeight: 700,
          fontSize: 11,
          height: 22,
        }}
      />
    );
  };

  // --- Model Selection ---
  const [selectedModel, setSelectedModel] = useState<string | null>(null);
  const [isModelDialogOpen, setIsModelDialogOpen] = useState(true); // Open by default if no model

  const MOCK_MODELS = [
    { id: 'm1', name: 'Orders & Users', description: 'E-commerce transactions and user demographics', category: 'Sales' },
    { id: 'm2', name: 'Product Inventory', description: 'Stock levels, suppliers, and product details', category: 'Inventory' },
    { id: 'm3', name: 'Web Analytics', description: 'Page views, sessions, and conversion events', category: 'Marketing' },
  ];

  const handleModelSelect = (modelId: string) => {
    setSelectedModel(modelId);
    setIsModelDialogOpen(false);
    // In a real app, we would fetch the schema for this model here
  };

  const [isFilterModalOpen, setIsFilterModalOpen] = useState(false);
  const [newFilter, setNewFilter] = useState<Partial<QueryFilter>>({
    operator: 'equals',
    values: [''],
  });

  const handleOpenFilterModal = (member?: string) => {
    setNewFilter({
      id: Math.random().toString(36).substr(2, 9),
      member: member || '',
      operator: 'equals',
      values: [''],
    });
    setIsFilterModalOpen(true);
  };

  const handleAddFilter = () => {
    if (newFilter.member && newFilter.operator && newFilter.values?.[0]) {
      setFilters(prev => [...prev, newFilter as QueryFilter]);
      setIsFilterModalOpen(false);
    }
  };

  // Export Dialog State
  const [isExportDialogOpen, setIsExportDialogOpen] = useState(false);
  const [exportTab, setExportTab] = useState('csv');
  const [exportSettings, setExportSettings] = useState({
    csv: { delimiter: ',', encoding: 'UTF-8', headerRow: true, quoteAll: false },
    json: { format: 'pretty', flatten: false, excludeNulls: false },
    xml: { rootName: 'QueryResult', rowName: 'Record', useAttributes: false },
    image: { width: 1280, height: 720, transparent: true, quality: 100 },
    pdf: { pageSize: 'A4', orientation: 'portrait', includeTable: false },
  });

  // --- Render Model Selection Overlay if no model selected ---
  if (!selectedModel) {
    return (
      <Box sx={{ p: 4, height: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', bgcolor: '#fcfcf9' }}>
        <Paper 
          elevation={0} 
          sx={{ 
            width: '100%', 
            maxWidth: 600, 
            p: 4, 
            borderRadius: 4, 
            border: '1px solid #e6e6db',
            textAlign: 'center'
          }}
        >
          <Avatar sx={{ width: 64, height: 64, bgcolor: '#f4f4ec', color: '#8c8b5f', mx: 'auto', mb: 2 }}>
            <DatasetIcon sx={{ fontSize: 32 }} />
          </Avatar>
          <Typography variant="h5" fontWeight={700} gutterBottom>
            Select a Semantic Model
          </Typography>
          <Typography color="text.secondary" sx={{ mb: 4 }}>
            Choose a data model to start building your query.
          </Typography>

          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            {MOCK_MODELS.map((model) => (
              <Button
                key={model.id}
                onClick={() => handleModelSelect(model.id)}
                sx={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 2,
                  p: 2,
                  textAlign: 'left',
                  borderRadius: 3,
                  border: '1px solid #e6e6db',
                  bgcolor: '#fff',
                  color: 'inherit',
                  transition: 'all 0.2s',
                  '&:hover': {
                    bgcolor: '#f9f9f0',
                    borderColor: '#8c8b5f',
                    transform: 'translateY(-2px)',
                    boxShadow: '0 4px 12px rgba(0,0,0,0.05)'
                  }
                }}
              >
                <Avatar variant="rounded" sx={{ bgcolor: '#f4f4ec', color: '#8c8b5f' }}>
                  <DatasetIcon />
                </Avatar>
                <Box sx={{ flex: 1 }}>
                  <Typography variant="subtitle1" fontWeight={600}>{model.name}</Typography>
                  <Typography variant="body2" color="text.secondary">{model.description}</Typography>
                </Box>
                <ChevronRightIcon sx={{ color: '#dcdcdc' }} />
              </Button>
            ))}
          </Box>
        </Paper>
      </Box>
    );
  }

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100vh', bgcolor: '#f8f8f5' }}>
      {/* Export Configuration Dialog */}
      {isExportDialogOpen && (
        <Box sx={{ position: 'fixed', inset: 0, zIndex: 1300, display: 'flex', alignItems: 'center', justifyContent: 'center', p: 2 }}>
          <Box sx={{ position: 'absolute', inset: 0, bgcolor: 'rgba(0,0,0,0.6)', backdropFilter: 'blur(4px)' }} onClick={() => setIsExportDialogOpen(false)} />
          <Paper
            elevation={24}
            sx={{
              position: 'relative',
              width: '100%',
              maxWidth: 800,
              maxHeight: '90vh',
              borderRadius: 2,
              bgcolor: '#1a1a2e', // Background Dark
              color: '#E0E7FF', // Text Main
              overflow: 'hidden',
              display: 'flex',
              flexDirection: 'column',
              border: '1px solid #4A4A63', // Border Dark
            }}
          >
            {/* Dialog Header */}
            <Box sx={{ px: 3, py: 2, borderBottom: '1px solid #4A4A63', bgcolor: '#2c2c44', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <Box>
                <Typography variant="h6" fontWeight={700}>Configure Export Job</Typography>
                <Typography variant="caption" color="#9CA3AF">Define detailed settings for your data export.</Typography>
              </Box>
              <IconButton onClick={() => setIsExportDialogOpen(false)} sx={{ color: '#9CA3AF', '&:hover': { color: '#E0E7FF', bgcolor: '#363650' } }}>
                <CloseIcon />
              </IconButton>
            </Box>

            {/* Dialog Content */}
            <Box sx={{ display: 'flex', flex: 1, overflow: 'hidden', flexDirection: 'column' }}>
              {/* Tabs */}
              <Box sx={{ display: 'flex', borderBottom: '1px solid #4A4A63', bgcolor: '#2c2c44', px: 2, pt: 1 }}>
                {[
                  { id: 'csv', label: 'CSV', icon: 'csv' },
                  { id: 'json', label: 'JSON', icon: 'data_object' },
                  { id: 'xml', label: 'XML', icon: 'code' },
                  { id: 'svg', label: 'SVG', icon: 'image' },
                  { id: 'pdf', label: 'PDF', icon: 'picture_as_pdf' },
                  { id: 'jpeg', label: 'JPEG', icon: 'photo' },
                ].map((tab) => (
                  <Button
                    key={tab.id}
                    onClick={() => setExportTab(tab.id)}
                    startIcon={<span className="material-symbols-outlined" style={{ fontSize: 18 }}>{tab.icon}</span>}
                    sx={{
                      color: exportTab === tab.id ? '#3B82F6' : '#9CA3AF',
                      borderBottom: exportTab === tab.id ? '2px solid #3B82F6' : '2px solid transparent',
                      borderRadius: '8px 8px 0 0',
                      px: 2,
                      py: 1,
                      textTransform: 'none',
                      bgcolor: exportTab === tab.id ? '#1a1a2e' : 'transparent',
                      '&:hover': { color: '#E0E7FF', bgcolor: exportTab === tab.id ? '#1a1a2e' : '#363650' },
                    }}
                  >
                    {tab.label}
                  </Button>
                ))}
              </Box>

              {/* Tab Panel */}
              <Box sx={{ p: 4, overflowY: 'auto', flex: 1, bgcolor: '#1a1a2e' }}>
                <Typography variant="h6" fontWeight={600} gutterBottom>
                  {exportTab.toUpperCase()} Export Settings
                </Typography>
                <Typography variant="body2" color="#9CA3AF" sx={{ mb: 4 }}>
                  Configure options specific to {exportTab.toUpperCase()} file generation.
                </Typography>

                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
                  {exportTab === 'csv' && (
                    <>
                      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <Typography variant="body2" fontWeight={500}>Field Delimiter:</Typography>
                        <TextField
                          select
                          value={exportSettings.csv.delimiter}
                          onChange={(e) => setExportSettings({ ...exportSettings, csv: { ...exportSettings.csv, delimiter: e.target.value } })}
                          SelectProps={{ native: true }}
                          size="small"
                          sx={{ width: 250, bgcolor: '#2c2c44', borderRadius: 1, '& .MuiOutlinedInput-notchedOutline': { borderColor: '#4A4A63' }, '& .MuiSelect-select': { color: '#E0E7FF' } }}
                        >
                          <option value=",">Comma (,)</option>
                          <option value=";">Semicolon (;)</option>
                          <option value="\t">Tab (\t)</option>
                          <option value="|">Pipe (|)</option>
                        </TextField>
                      </Box>
                      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <Typography variant="body2" fontWeight={500}>Character Encoding:</Typography>
                        <TextField
                          select
                          value={exportSettings.csv.encoding}
                          onChange={(e) => setExportSettings({ ...exportSettings, csv: { ...exportSettings.csv, encoding: e.target.value } })}
                          SelectProps={{ native: true }}
                          size="small"
                          sx={{ width: 250, bgcolor: '#2c2c44', borderRadius: 1, '& .MuiOutlinedInput-notchedOutline': { borderColor: '#4A4A63' }, '& .MuiSelect-select': { color: '#E0E7FF' } }}
                        >
                          <option value="UTF-8">UTF-8 (Recommended)</option>
                          <option value="UTF-16">UTF-16</option>
                          <option value="ISO-8859-1">ISO-8859-1 (Latin-1)</option>
                        </TextField>
                      </Box>
                      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <Typography variant="body2" fontWeight={500}>Include Header Row:</Typography>
                        <Switch
                          checked={exportSettings.csv.headerRow}
                          onChange={(e) => setExportSettings({ ...exportSettings, csv: { ...exportSettings.csv, headerRow: e.target.checked } })}
                          sx={{ '& .MuiSwitch-switchBase.Mui-checked': { color: '#3B82F6' }, '& .MuiSwitch-switchBase.Mui-checked + .MuiSwitch-track': { backgroundColor: '#3B82F6' } }}
                        />
                      </Box>
                      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <Typography variant="body2" fontWeight={500}>Always Quote Fields:</Typography>
                        <Switch
                          checked={exportSettings.csv.quoteAll}
                          onChange={(e) => setExportSettings({ ...exportSettings, csv: { ...exportSettings.csv, quoteAll: e.target.checked } })}
                          sx={{ '& .MuiSwitch-switchBase.Mui-checked': { color: '#3B82F6' }, '& .MuiSwitch-switchBase.Mui-checked + .MuiSwitch-track': { backgroundColor: '#3B82F6' } }}
                        />
                      </Box>
                    </>
                  )}
                  {exportTab === 'json' && (
                    <>
                      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <Typography variant="body2" fontWeight={500}>Output Format:</Typography>
                        <TextField
                          select
                          value={exportSettings.json.format}
                          onChange={(e) => setExportSettings({ ...exportSettings, json: { ...exportSettings.json, format: e.target.value } })}
                          SelectProps={{ native: true }}
                          size="small"
                          sx={{ width: 250, bgcolor: '#2c2c44', borderRadius: 1, '& .MuiOutlinedInput-notchedOutline': { borderColor: '#4A4A63' }, '& .MuiSelect-select': { color: '#E0E7FF' } }}
                        >
                          <option value="pretty">Pretty Print (Readable)</option>
                          <option value="minify">Minified (Compact)</option>
                        </TextField>
                      </Box>
                      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <Typography variant="body2" fontWeight={500}>Flatten Nested Data:</Typography>
                        <Switch
                          checked={exportSettings.json.flatten}
                          onChange={(e) => setExportSettings({ ...exportSettings, json: { ...exportSettings.json, flatten: e.target.checked } })}
                          sx={{ '& .MuiSwitch-switchBase.Mui-checked': { color: '#3B82F6' }, '& .MuiSwitch-switchBase.Mui-checked + .MuiSwitch-track': { backgroundColor: '#3B82F6' } }}
                        />
                      </Box>
                      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <Typography variant="body2" fontWeight={500}>Exclude Null Values:</Typography>
                        <Switch
                          checked={exportSettings.json.excludeNulls}
                          onChange={(e) => setExportSettings({ ...exportSettings, json: { ...exportSettings.json, excludeNulls: e.target.checked } })}
                          sx={{ '& .MuiSwitch-switchBase.Mui-checked': { color: '#3B82F6' }, '& .MuiSwitch-switchBase.Mui-checked + .MuiSwitch-track': { backgroundColor: '#3B82F6' } }}
                        />
                      </Box>
                    </>
                  )}
                  {exportTab === 'xml' && (
                    <>
                      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <Typography variant="body2" fontWeight={500}>Root Element Name:</Typography>
                        <TextField
                          value={exportSettings.xml.rootName}
                          onChange={(e) => setExportSettings({ ...exportSettings, xml: { ...exportSettings.xml, rootName: e.target.value } })}
                          size="small"
                          sx={{ width: 250, bgcolor: '#2c2c44', borderRadius: 1, '& .MuiOutlinedInput-notchedOutline': { borderColor: '#4A4A63' }, '& .MuiInputBase-input': { color: '#E0E7FF' } }}
                        />
                      </Box>
                      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <Typography variant="body2" fontWeight={500}>Row Element Name:</Typography>
                        <TextField
                          value={exportSettings.xml.rowName}
                          onChange={(e) => setExportSettings({ ...exportSettings, xml: { ...exportSettings.xml, rowName: e.target.value } })}
                          size="small"
                          sx={{ width: 250, bgcolor: '#2c2c44', borderRadius: 1, '& .MuiOutlinedInput-notchedOutline': { borderColor: '#4A4A63' }, '& .MuiInputBase-input': { color: '#E0E7FF' } }}
                        />
                      </Box>
                      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <Typography variant="body2" fontWeight={500}>Use Attributes for Fields:</Typography>
                        <Switch
                          checked={exportSettings.xml.useAttributes}
                          onChange={(e) => setExportSettings({ ...exportSettings, xml: { ...exportSettings.xml, useAttributes: e.target.checked } })}
                          sx={{ '& .MuiSwitch-switchBase.Mui-checked': { color: '#3B82F6' }, '& .MuiSwitch-switchBase.Mui-checked + .MuiSwitch-track': { backgroundColor: '#3B82F6' } }}
                        />
                      </Box>
                    </>
                  )}
                  {exportTab === 'svg' && (
                    <>
                      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <Typography variant="body2" fontWeight={500}>Custom Dimensions:</Typography>
                        <Box sx={{ display: 'flex', gap: 1, width: 250 }}>
                          <TextField
                            placeholder="Width"
                            type="number"
                            value={exportSettings.image.width}
                            onChange={(e) => setExportSettings({ ...exportSettings, image: { ...exportSettings.image, width: Number(e.target.value) } })}
                            size="small"
                            sx={{ flex: 1, bgcolor: '#2c2c44', borderRadius: 1, '& .MuiOutlinedInput-notchedOutline': { borderColor: '#4A4A63' }, '& .MuiInputBase-input': { color: '#E0E7FF' } }}
                          />
                          <TextField
                            placeholder="Height"
                            type="number"
                            value={exportSettings.image.height}
                            onChange={(e) => setExportSettings({ ...exportSettings, image: { ...exportSettings.image, height: Number(e.target.value) } })}
                            size="small"
                            sx={{ flex: 1, bgcolor: '#2c2c44', borderRadius: 1, '& .MuiOutlinedInput-notchedOutline': { borderColor: '#4A4A63' }, '& .MuiInputBase-input': { color: '#E0E7FF' } }}
                          />
                        </Box>
                      </Box>
                      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <Typography variant="body2" fontWeight={500}>Transparent Background:</Typography>
                        <Switch
                          checked={exportSettings.image.transparent}
                          onChange={(e) => setExportSettings({ ...exportSettings, image: { ...exportSettings.image, transparent: e.target.checked } })}
                          sx={{ '& .MuiSwitch-switchBase.Mui-checked': { color: '#3B82F6' }, '& .MuiSwitch-switchBase.Mui-checked + .MuiSwitch-track': { backgroundColor: '#3B82F6' } }}
                        />
                      </Box>
                    </>
                  )}
                  {exportTab === 'pdf' && (
                    <>
                      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <Typography variant="body2" fontWeight={500}>Page Size:</Typography>
                        <TextField
                          select
                          value={exportSettings.pdf.pageSize}
                          onChange={(e) => setExportSettings({ ...exportSettings, pdf: { ...exportSettings.pdf, pageSize: e.target.value } })}
                          SelectProps={{ native: true }}
                          size="small"
                          sx={{ width: 250, bgcolor: '#2c2c44', borderRadius: 1, '& .MuiOutlinedInput-notchedOutline': { borderColor: '#4A4A63' }, '& .MuiSelect-select': { color: '#E0E7FF' } }}
                        >
                          <option value="A4">A4</option>
                          <option value="Letter">Letter</option>
                          <option value="Legal">Legal</option>
                        </TextField>
                      </Box>
                      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <Typography variant="body2" fontWeight={500}>Page Orientation:</Typography>
                        <TextField
                          select
                          value={exportSettings.pdf.orientation}
                          onChange={(e) => setExportSettings({ ...exportSettings, pdf: { ...exportSettings.pdf, orientation: e.target.value } })}
                          SelectProps={{ native: true }}
                          size="small"
                          sx={{ width: 250, bgcolor: '#2c2c44', borderRadius: 1, '& .MuiOutlinedInput-notchedOutline': { borderColor: '#4A4A63' }, '& .MuiSelect-select': { color: '#E0E7FF' } }}
                        >
                          <option value="portrait">Portrait</option>
                          <option value="landscape">Landscape</option>
                        </TextField>
                      </Box>
                      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <Typography variant="body2" fontWeight={500}>Include Raw Data Table:</Typography>
                        <Switch
                          checked={exportSettings.pdf.includeTable}
                          onChange={(e) => setExportSettings({ ...exportSettings, pdf: { ...exportSettings.pdf, includeTable: e.target.checked } })}
                          sx={{ '& .MuiSwitch-switchBase.Mui-checked': { color: '#3B82F6' }, '& .MuiSwitch-switchBase.Mui-checked + .MuiSwitch-track': { backgroundColor: '#3B82F6' } }}
                        />
                      </Box>
                    </>
                  )}
                  {exportTab === 'jpeg' && (
                    <>
                      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <Typography variant="body2" fontWeight={500}>Image Quality:</Typography>
                        <TextField
                          select
                          value={exportSettings.image.quality}
                          onChange={(e) => setExportSettings({ ...exportSettings, image: { ...exportSettings.image, quality: Number(e.target.value) } })}
                          SelectProps={{ native: true }}
                          size="small"
                          sx={{ width: 250, bgcolor: '#2c2c44', borderRadius: 1, '& .MuiOutlinedInput-notchedOutline': { borderColor: '#4A4A63' }, '& .MuiSelect-select': { color: '#E0E7FF' } }}
                        >
                          <option value="100">100% (High)</option>
                          <option value="90">90% (Medium)</option>
                          <option value="80">80% (Standard)</option>
                          <option value="70">70% (Low)</option>
                        </TextField>
                      </Box>
                      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <Typography variant="body2" fontWeight={500}>Resolution (DPI):</Typography>
                        <TextField
                          type="number"
                          value="300"
                          size="small"
                          sx={{ width: 250, bgcolor: '#2c2c44', borderRadius: 1, '& .MuiOutlinedInput-notchedOutline': { borderColor: '#4A4A63' }, '& .MuiInputBase-input': { color: '#E0E7FF' } }}
                        />
                      </Box>
                      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <Typography variant="body2" fontWeight={500}>Include Visualization Legend:</Typography>
                        <Switch
                          defaultChecked
                          sx={{ '& .MuiSwitch-switchBase.Mui-checked': { color: '#3B82F6' }, '& .MuiSwitch-switchBase.Mui-checked + .MuiSwitch-track': { backgroundColor: '#3B82F6' } }}
                        />
                      </Box>
                    </>
                  )}
                </Box>
              </Box>
            </Box>

            {/* Dialog Footer */}
            <Box sx={{ px: 3, py: 2, bgcolor: '#2c2c44', borderTop: '1px solid #4A4A63', display: 'flex', justifyContent: 'flex-end', gap: 2 }}>
              <Button onClick={() => setIsExportDialogOpen(false)} sx={{ color: '#9CA3AF', '&:hover': { color: '#E0E7FF', bgcolor: '#363650' } }}>Cancel</Button>
              <Button variant="contained" startIcon={<DownloadIcon />} sx={{ bgcolor: '#3B82F6', '&:hover': { bgcolor: '#2563EB' } }}>Initiate Export</Button>
            </Box>
          </Paper>
        </Box>
      )}

      {/* Add Filter Modal */}
      {isFilterModalOpen && (
        <Box sx={{ position: 'fixed', inset: 0, zIndex: 1300, display: 'flex', alignItems: 'center', justifyContent: 'center', p: 2 }}>
          <Box sx={{ position: 'absolute', inset: 0, bgcolor: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(2px)' }} onClick={() => setIsFilterModalOpen(false)} />
          <Paper
            elevation={24}
            sx={{
              position: 'relative',
              width: '100%',
              maxWidth: 500,
              borderRadius: 4,
              bgcolor: 'white',
              overflow: 'hidden',
              border: '1px solid #e6e6db',
            }}
          >
            {/* Modal Header */}
            <Box sx={{ px: 3, py: 2, borderBottom: '1px solid #e6e6db', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
                <Box sx={{ width: 40, height: 40, borderRadius: '50%', bgcolor: '#f5f5f0', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                  <FilterIcon sx={{ color: '#8c8b5f' }} />
                </Box>
                <Box>
                  <Typography variant="h6" fontWeight={700} sx={{ lineHeight: 1.2 }}>Add Filter</Typography>
                  <Typography variant="caption" color="#8c8b5f">Refine your query results</Typography>
                </Box>
              </Box>
              <IconButton onClick={() => setIsFilterModalOpen(false)}>
                <CloseIcon />
              </IconButton>
            </Box>

            {/* Modal Body */}
            <Box sx={{ p: 3, display: 'flex', flexDirection: 'column', gap: 2.5 }}>
              <Box>
                <Typography variant="caption" fontWeight={700} color="#8c8b5f" sx={{ textTransform: 'uppercase', letterSpacing: 1, mb: 1, display: 'block' }}>
                  Dimension
                </Typography>
                <TextField
                  select
                  fullWidth
                  value={newFilter.member}
                  onChange={(e) => setNewFilter({ ...newFilter, member: e.target.value })}
                  SelectProps={{ native: true }}
                  sx={{
                    '& .MuiOutlinedInput-root': {
                      borderRadius: 3,
                      bgcolor: '#f8f8f5',
                      '& fieldset': { borderColor: '#e6e6db' },
                    },
                  }}
                >
                  <option value="" disabled>Select a field...</option>
                  {Object.entries(SAMPLE_SCHEMA).map(([section, fields]) => (
                    <optgroup key={section} label={section}>
                      {fields.map(f => (
                        <option key={`${section}.${f.name}`} value={`${section}.${f.display_name}`}>
                          {section}.{f.display_name}
                        </option>
                      ))}
                    </optgroup>
                  ))}
                </TextField>
              </Box>

              <Box sx={{ display: 'flex', gap: 2 }}>
                <Box sx={{ flex: 1 }}>
                  <Typography variant="caption" fontWeight={700} color="#8c8b5f" sx={{ textTransform: 'uppercase', letterSpacing: 1, mb: 1, display: 'block' }}>
                    Operator
                  </Typography>
                  <TextField
                    select
                    fullWidth
                    value={newFilter.operator}
                    onChange={(e) => setNewFilter({ ...newFilter, operator: e.target.value })}
                    SelectProps={{ native: true }}
                    sx={{
                      '& .MuiOutlinedInput-root': {
                        borderRadius: 3,
                        bgcolor: '#f8f8f5',
                        '& fieldset': { borderColor: '#e6e6db' },
                      },
                    }}
                  >
                    <option value="equals">is</option>
                    <option value="not_equals">is not</option>
                    <option value="contains">contains</option>
                    <option value="set">is set</option>
                    <option value="not_set">is not set</option>
                    <option value="is in">is in</option>
                  </TextField>
                </Box>
                <Box sx={{ flex: 1 }}>
                  <Typography variant="caption" fontWeight={700} color="#8c8b5f" sx={{ textTransform: 'uppercase', letterSpacing: 1, mb: 1, display: 'block' }}>
                    Value
                  </Typography>
                  <TextField
                    fullWidth
                    placeholder="Enter value..."
                    value={newFilter.values?.[0] || ''}
                    onChange={(e) => setNewFilter({ ...newFilter, values: [e.target.value] })}
                    sx={{
                      '& .MuiOutlinedInput-root': {
                        borderRadius: 3,
                        bgcolor: '#f8f8f5',
                        '& fieldset': { borderColor: '#e6e6db' },
                      },
                    }}
                  />
                </Box>
              </Box>

              <Box sx={{ p: 1.5, borderRadius: 2, bgcolor: 'rgba(249, 245, 6, 0.1)', border: '1px solid rgba(249, 245, 6, 0.2)', display: 'flex', gap: 1 }}>
                <Typography sx={{ fontSize: 16 }}>💡</Typography>
                <Typography variant="caption" color="#181811">
                  Filtering by <strong>{newFilter.member || 'a field'}</strong> will update the total revenue chart instantly.
                </Typography>
              </Box>
            </Box>

            {/* Modal Footer */}
            <Box sx={{ p: 2, bgcolor: '#f8f8f5', borderTop: '1px solid #e6e6db', display: 'flex', justifyContent: 'flex-end', gap: 1.5 }}>
              <Button
                onClick={() => setIsFilterModalOpen(false)}
                sx={{
                  color: '#8c8b5f',
                  fontWeight: 700,
                  textTransform: 'none',
                  borderRadius: '9999px',
                  px: 3,
                  '&:hover': { bgcolor: 'white' },
                }}
              >
                Cancel
              </Button>
              <Button
                onClick={handleAddFilter}
                variant="contained"
                disabled={!newFilter.member || !newFilter.values?.[0]}
                sx={{
                  bgcolor: '#f9f506',
                  color: '#181811',
                  borderRadius: '9999px',
                  fontWeight: 700,
                  textTransform: 'none',
                  px: 3,
                  boxShadow: 'none',
                  '&:hover': { bgcolor: '#e6e205', boxShadow: 'none' },
                  '&:disabled': { bgcolor: '#e6e6db', color: '#8c8b5f' },
                }}
              >
                Apply Filter
              </Button>
            </Box>
          </Paper>
        </Box>
      )}

      {/* Header */}
      <Paper
        elevation={0}
        sx={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          borderBottom: '1px solid #e6e6db',
          px: 3,
          py: 1.5,
          bgcolor: 'white',
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 3 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Box
              sx={{
                width: 32,
                height: 32,
                bgcolor: '#f9f506',
                borderRadius: 2,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <DatasetIcon sx={{ fontSize: 20, color: 'black' }} />
            </Box>
            <Typography variant="body1" fontWeight="bold">
              CubeQuery
            </Typography>
          </Box>

          <Divider orientation="vertical" flexItem sx={{ height: 24, alignSelf: 'center' }} />

          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Typography variant="body2" color="#8c8b5f" fontWeight={500}>
              Query:
            </Typography>
            {isEditingName ? (
              <TextField
                value={queryName}
                onChange={(e) => setQueryName(e.target.value)}
                onBlur={() => setIsEditingName(false)}
                autoFocus
                variant="standard"
                sx={{ width: 300 }}
              />
            ) : (
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, cursor: 'pointer' }} onClick={() => setIsEditingName(true)}>
                <Typography variant="body1" fontWeight={600}>
                  {queryName}
                </Typography>
                <EditIcon sx={{ fontSize: 16, color: '#8c8b5f' }} />
              </Box>
            )}
          </Box>
        </Box>

        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          <Box sx={{ display: 'flex', gap: 0.5 }}>
            <IconButton size="small" sx={{ color: '#181811' }}>
              <SaveIcon fontSize="small" />
            </IconButton>
            <IconButton size="small" sx={{ color: '#181811' }} onClick={() => setIsExportDialogOpen(true)}>
              <DownloadIcon fontSize="small" />
            </IconButton>
            <IconButton size="small" sx={{ color: '#181811' }}>
              <CodeIcon fontSize="small" />
            </IconButton>
          </Box>

          <Button
            variant="contained"
            startIcon={<PlayIcon />}
            sx={{
              bgcolor: '#f9f506',
              color: '#181811',
              borderRadius: '9999px',
              px: 3,
              py: 1,
              fontWeight: 700,
              fontSize: 14,
              textTransform: 'none',
              boxShadow: 1,
              '&:hover': { bgcolor: '#e6e205' },
            }}
          >
            Run Query
          </Button>

          <Avatar sx={{ width: 40, height: 40, border: '1px solid #e6e6db' }} />
        </Box>
      </Paper>

      {/* Main Content */}
      <Box sx={{ display: 'flex', flex: 1, overflow: 'hidden' }}>
        {/* Schema Sidebar */}
        <Paper
          elevation={0}
          sx={{
            width: 320,
            borderRight: '1px solid #e6e6db',
            bgcolor: '#f8f8f5',
            display: 'flex',
            flexDirection: 'column',
            overflow: 'hidden',
          }}
        >
          <Box sx={{ p: 2, borderBottom: '1px solid #e6e6db' }}>
            <TextField
              placeholder="Search schema..."
              value={schemaSearch}
              onChange={(e) => setSchemaSearch(e.target.value)}
              size="small"
              fullWidth
              sx={{
                '& .MuiOutlinedInput-root': {
                  borderRadius: 3,
                  bgcolor: 'white',
                  '& fieldset': { borderColor: '#e6e6db' },
                },
              }}
              InputProps={{
                startAdornment: (
                  <InputAdornment position="start">
                    <SearchIcon sx={{ fontSize: 20, color: '#8c8b5f' }} />
                  </InputAdornment>
                ),
              }}
            />
          </Box>

          <Box sx={{ flex: 1, overflow: 'auto', p: 2 }}>
            {Object.entries(SAMPLE_SCHEMA).map(([sectionName, fields]) => (
              <Box key={sectionName} sx={{ mb: 3 }}>
                <Box
                  sx={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    px: 1,
                    mb: 1,
                    cursor: 'pointer',
                  }}
                  onClick={() => toggleSection(sectionName)}
                >
                  <Typography variant="caption" fontWeight={700} sx={{ textTransform: 'uppercase', letterSpacing: 1, color: '#8c8b5f' }}>
                    {sectionName}
                  </Typography>
                  <ExpandMoreIcon
                    sx={{
                      fontSize: 16,
                      color: '#8c8b5f',
                      transform: expandedSections[sectionName] ? 'rotate(180deg)' : 'rotate(0deg)',
                      transition: 'transform 0.2s',
                    }}
                  />
                </Box>

                <Collapse in={expandedSections[sectionName]}>
                  <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
                    {fields.map((field) => (
                      <Box
                        key={field.name}
                        sx={{
                          display: 'flex',
                          alignItems: 'center',
                          gap: 1.5,
                          p: 1,
                          borderRadius: 2,
                          cursor: 'pointer',
                          transition: 'background 0.2s',
                          '&:hover': {
                            bgcolor: 'rgba(249, 245, 6, 0.2)',
                            '& .field-actions': { opacity: 1 },
                          },
                        }}
                      >
                        {getFieldIcon(field.icon)}
                        <Typography variant="body2" fontWeight={500} sx={{ flex: 1 }}>
                          {field.display_name}
                        </Typography>
                        <Box className="field-actions" sx={{ display: 'flex', gap: 0.5, opacity: 0, transition: 'opacity 0.2s' }}>
                          <IconButton size="small" sx={{ p: 0.5 }} onClick={(e) => { e.stopPropagation(); handleOpenFilterModal(`${sectionName}.${field.display_name}`); }}>
                            <FilterIcon sx={{ fontSize: 16 }} />
                          </IconButton>
                          <IconButton size="small" sx={{ p: 0.5 }}>
                            <AddIcon sx={{ fontSize: 16 }} />
                          </IconButton>
                        </Box>
                      </Box>
                    ))}
                  </Box>
                </Collapse>
              </Box>
            ))}
          </Box>
        </Paper>

        {/* Results Area */}


      {/* Main Content - Results & Visualizations */}
      <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
        {/* Toolbar */}
        <Box sx={{ 
          height: 64, 
          borderBottom: '1px solid #e6e6db', 
          display: 'flex', 
          alignItems: 'center', 
          justifyContent: 'space-between',
          px: 3,
          bgcolor: '#fff'
        }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
            <Button
              variant="contained"
              startIcon={<PlayIcon />}
              sx={{ 
                bgcolor: '#f9f506', // Yellow Accent
                color: '#181811',
                fontWeight: 700,
                boxShadow: 'none',
                borderRadius: '8px',
                '&:hover': { bgcolor: '#e6e205', boxShadow: 'none' }
              }}
            >
              Run Query
            </Button>
            <Divider orientation="vertical" flexItem sx={{ height: 24, my: 'auto' }} />
             <Button
              variant="outlined"
              startIcon={<SaveIcon />}
              sx={{ borderRadius: '8px', borderColor: '#e6e6db', color: '#181811', fontWeight: 600 }}
            >
              Save
            </Button>
            <Button
              variant="outlined"
              startIcon={<DownloadIcon />}
              sx={{ borderRadius: '8px', borderColor: '#e6e6db', color: '#181811', fontWeight: 600 }}
            >
              Export
            </Button>
          </Box>

          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
             {/* Limit Selector */}
             <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, bgcolor: '#f4f4ec', p: 0.5, borderRadius: '8px' }}>
                <Typography variant="caption" fontWeight={700} sx={{ ml: 1, color: '#8c8b5f' }}>LIMIT</Typography>
                <TextField 
                  variant="standard" 
                  value="1000" 
                  InputProps={{ disableUnderline: true, sx: { fontSize: 14, fontWeight: 600, width: 40, textAlign: 'center' } }} 
                />
             </Box>
          </Box>
        </Box>

        {/* Filters Bar (if active) */}
        {filters.length > 0 && (
          <Box sx={{ px: 3, py: 1.5, borderBottom: '1px solid #e6e6db', display: 'flex', flexWrap: 'wrap', gap: 1, bgcolor: '#fcfcf9' }}>
            {filters.map(filter => (
               <Chip 
                 key={filter.id}
                 label={
                   <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                     <Typography variant="caption" fontWeight={700} color="#575752">{filter.member}</Typography>
                     <Typography variant="caption" color="#8c8b5f">{filter.operator}</Typography>
                     <Typography variant="caption" fontWeight={600} color="#181811">{filter.values[0]}</Typography>
                   </Box>
                 }
                 onDelete={() => removeFilter(filter.id)}
                 sx={{ 
                   bgcolor: '#fff', 
                   border: '1px solid #e6e6db', 
                   borderRadius: '8px',
                   '& .MuiChip-deleteIcon': { fontSize: 16, color: '#a0a096' } 
                 }}
               />
            ))}
            <Button 
              size="small" 
              startIcon={<AddIcon />} 
              sx={{ color: '#8c8b5f', fontWeight: 600, textTransform: 'none' }}
              onClick={() => setIsFilterModalOpen(true)}
            >
              Add Filter
            </Button>
          </Box>
        )}

        {/* Results Area */}
        <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', p: 3, gap: 3, bgcolor: '#fafafa', overflowY: 'auto' }}>
           
           {/* Chart Section */}
           <Paper elevation={0} sx={{ p: 3, border: '1px solid #e6e6db', borderRadius: 4 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 3 }}>
                <Typography variant="h6" fontWeight={700}>Visualizations</Typography>
                <ToggleButtonGroup
                  value={chartType}
                  exclusive
                  onChange={(e, v) => v && setChartType(v)}
                  size="small"
                  sx={{ 
                    bgcolor: '#f4f4ec', 
                    borderRadius: 2,
                    '& .MuiToggleButton-root': { border: 'none', borderRadius: 2, m: 0.5, py: 0.5, px: 1 }
                  }}
                >
                  <ToggleButton value="line"><LineChartIcon fontSize="small" /></ToggleButton>
                  <ToggleButton value="bar"><BarChartIcon fontSize="small" /></ToggleButton>
                  <ToggleButton value="pie"><PieChartIcon fontSize="small" /></ToggleButton>
                </ToggleButtonGroup>
              </Box>
              
              <Box sx={{ height: 300, width: '100%' }}>
                 <ResponsiveContainer width="100%" height="100%">
                    <LineChart data={CHART_DATA}>
                       <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                       <XAxis dataKey="x" stroke="#a0a096" tick={{ fontSize: 12 }} />
                       <YAxis stroke="#a0a096" tick={{ fontSize: 12 }} />
                       <RechartsTooltip 
                         contentStyle={{ borderRadius: 12, border: 'none', boxShadow: '0 4px 20px rgba(0,0,0,0.08)' }}
                       />
                       <Line 
                         type="monotone" 
                         dataKey="y" 
                         stroke="#f9f506" 
                         strokeWidth={3} 
                         dot={{ fill: '#181811', strokeWidth: 0, r: 4 }} 
                         activeDot={{ r: 6 }} 
                       />
                    </LineChart>
                 </ResponsiveContainer>
              </Box>
           </Paper>

           {/* Table Section */}
           <Paper elevation={0} sx={{ border: '1px solid #e6e6db', borderRadius: 4, overflow: 'hidden', flex: 1 }}>
              <Box sx={{ px: 3, py: 2, borderBottom: '1px solid #e6e6db', display: 'flex', gap: 2 }}>
                 <Button 
                   size="small" 
                   sx={{ 
                     color: activeTab === 'results' ? '#181811' : '#8c8b5f', 
                     fontWeight: 700,
                     borderBottom: activeTab === 'results' ? '2px solid #f9f506' : 'none',
                     borderRadius: 0,
                     pb: 1
                   }}
                   onClick={() => setActiveTab('results')}
                 >
                   Results (150)
                 </Button>
                 <Button 
                   size="small" 
                   sx={{ 
                     color: activeTab === 'sql' ? '#181811' : '#8c8b5f', 
                     fontWeight: 700,
                     borderBottom: activeTab === 'sql' ? '2px solid #f9f506' : 'none',
                     borderRadius: 0,
                     pb: 1
                   }}
                   onClick={() => setActiveTab('sql')}
                 >
                   Generated SQL
                 </Button>
              </Box>
              
              <Box sx={{ p: 0, overflowX: 'auto' }}>
                 {activeTab === 'results' && (
                    <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 14 }}>
                       <thead>
                          <tr style={{ background: '#fcfcf9', borderBottom: '1px solid #e6e6db' }}>
                             <th style={{ padding: '12px 24px', textAlign: 'left', color: '#575752', fontWeight: 600 }}>Date</th>
                             <th style={{ padding: '12px 24px', textAlign: 'left', color: '#575752', fontWeight: 600 }}>City</th>
                             <th style={{ padding: '12px 24px', textAlign: 'left', color: '#575752', fontWeight: 600 }}>Status</th>
                             <th style={{ padding: '12px 24px', textAlign: 'right', color: '#575752', fontWeight: 600 }}>Revenue</th>
                          </tr>
                       </thead>
                       <tbody>
                          {SAMPLE_DATA.map((row, i) => (
                             <tr key={i} style={{ borderBottom: '1px solid #f5f5f0' }}>
                                <td style={{ padding: '12px 24px', color: '#181811' }}>{row.date}</td>
                                <td style={{ padding: '12px 24px', color: '#181811' }}>{row.city}</td>
                                <td style={{ padding: '12px 24px' }}>{getStatusChip(row.status)}</td>
                                <td style={{ padding: '12px 24px', textAlign: 'right', fontWeight: 600, color: '#181811' }}>
                                  ${row.revenue.toLocaleString()}
                                </td>
                             </tr>
                          ))}
                       </tbody>
                    </table>
                 )}
                 {activeTab === 'sql' && (
                   <Paper sx={{ p: 3, bgcolor: '#1e1e1e', borderRadius: 0, color: '#d4d4d4', fontFamily: 'monospace', fontSize: 13, minHeight: 200 }}>
                     <Typography component="pre" sx={{ m: 0, whiteSpace: 'pre-wrap' }}>
                       {`SELECT\n  orders.created_at,\n  users.city,\n  orders.status,\n  SUM(orders.total_revenue) as total_revenue\nFROM orders\nJOIN users ON orders.user_id = users.id\nWHERE orders.status = 'Completed'\n  AND orders.created_at >= NOW() - INTERVAL '30 days'\nGROUP BY orders.created_at, users.city, orders.status\nORDER BY orders.created_at DESC;`}
                     </Typography>
                   </Paper>
                 )}
              </Box>
           </Paper>
        </Box>
      </Box>
    </Box>
    </Box>
  );
};

export default DataExplorer;
