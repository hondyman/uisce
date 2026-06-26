/**
 * Report Designer Component
 * 
 * A drag-drop SSRS-style report designer for creating and editing
 * report definitions using the semantic reporting platform.
 */

import React, { useState, useCallback, useMemo } from 'react';
import {
  Box,
  Paper,
  Typography,
  Drawer,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  ListItemButton,
  Divider,
  TextField,
  Button,
  IconButton,
  Toolbar,
  Tabs,
  Tab,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Chip,
  Alert,
  Snackbar,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Tooltip,
  Card,
  CardContent,
} from '@mui/material';
import {
  DndContext,
  DragOverlay,
  useDraggable,
  useDroppable,
  DragEndEvent,
  DragStartEvent,
} from '@dnd-kit/core';
import {
  Save,
  Undo,
  Redo,
  Preview,
  Settings,
  ExpandMore,
  Add,
  Delete,
  DragIndicator,
  TextFields,
  Image,
  TableChart,
  BarChart,
  PieChart,
  ShowChart,
  Dashboard,
  DataObject,
  Numbers,
  CalendarMonth,
  ToggleOn,
  FormatListBulleted,
  Close,
} from '@mui/icons-material';
import {
  useReportDefinition,
  useCreateReportDefinition,
  useUpdateReportDefinition,
} from '../../hooks/useSemanticReporting';
import {
  ReportLayout,
  ReportSection,
  Parameter,
  DataBinding,
  TableColumn,
  LayoutElement,
  CreateDefinitionRequest,
} from '../../api/semanticReporting';
import { useTenant } from '../../contexts/TenantContext';
import { devLog } from '../../utils/devLogger';

// ============================================================================
// TYPES
// ============================================================================

interface DesignerElement {
  id: string;
  type: string;
  label: string;
  icon: React.ReactNode;
}

interface DesignerSection {
  id: string;
  type: 'summary' | 'table' | 'chart' | 'text';
  title: string;
  data_binding?: string;
  columns?: TableColumn[];
  elements?: LayoutElement[];
}

interface ReportDesignerProps {
  reportId?: string; // If provided, edit existing report
  onSave?: (reportId: string) => void;
  onCancel?: () => void;
}

// ============================================================================
// TOOLBOX ELEMENTS
// ============================================================================

const SECTION_ELEMENTS: DesignerElement[] = [
  { id: 'summary', type: 'summary', label: 'Summary Section', icon: <Dashboard /> },
  { id: 'table', type: 'table', label: 'Data Table', icon: <TableChart /> },
  { id: 'chart-bar', type: 'chart', label: 'Bar Chart', icon: <BarChart /> },
  { id: 'chart-line', type: 'chart', label: 'Line Chart', icon: <ShowChart /> },
  { id: 'chart-pie', type: 'chart', label: 'Pie Chart', icon: <PieChart /> },
  { id: 'text', type: 'text', label: 'Text Block', icon: <TextFields /> },
];

const FIELD_ELEMENTS: DesignerElement[] = [
  { id: 'text-field', type: 'text', label: 'Text', icon: <TextFields /> },
  { id: 'image-field', type: 'image', label: 'Image', icon: <Image /> },
  { id: 'kpi-card', type: 'kpiCard', label: 'KPI Card', icon: <Numbers /> },
];

const PARAMETER_TYPES: DesignerElement[] = [
  { id: 'param-string', type: 'string', label: 'Text', icon: <TextFields /> },
  { id: 'param-number', type: 'number', label: 'Number', icon: <Numbers /> },
  { id: 'param-date', type: 'date', label: 'Date', icon: <CalendarMonth /> },
  { id: 'param-boolean', type: 'boolean', label: 'Yes/No', icon: <ToggleOn /> },
  { id: 'param-select', type: 'select', label: 'Dropdown', icon: <FormatListBulleted /> },
];

// ============================================================================
// DRAGGABLE TOOLBOX ITEM
// ============================================================================

const ToolboxItem: React.FC<{ element: DesignerElement }> = ({ element }) => {
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
    id: `toolbox-${element.id}`,
    data: { element },
  });

  return (
    <ListItem disablePadding>
      <ListItemButton
        ref={setNodeRef}
        {...listeners}
        {...attributes}
        sx={{
          opacity: isDragging ? 0.5 : 1,
          cursor: 'grab',
          '&:active': { cursor: 'grabbing' },
        }}
      >
        <ListItemIcon sx={{ minWidth: 36 }}>{element.icon}</ListItemIcon>
        <ListItemText primary={element.label} primaryTypographyProps={{ variant: 'body2' }} />
      </ListItemButton>
    </ListItem>
  );
};

// ============================================================================
// DROPPABLE CANVAS AREA
// ============================================================================

interface CanvasDropZoneProps {
  children: React.ReactNode;
  onDrop: (element: DesignerElement, position: number) => void;
}

const CanvasDropZone: React.FC<CanvasDropZoneProps> = ({ children }) => {
  const { setNodeRef, isOver } = useDroppable({ id: 'canvas-drop-zone' });

  return (
    <Box
      ref={setNodeRef}
      sx={{
        minHeight: 400,
        p: 2,
        bgcolor: isOver ? 'action.hover' : 'grey.50',
        border: '2px dashed',
        borderColor: isOver ? 'primary.main' : 'grey.300',
        borderRadius: 1,
        transition: 'all 0.2s',
      }}
    >
      {children}
    </Box>
  );
};

// ============================================================================
// SECTION CARD (Rendered in Canvas)
// ============================================================================

interface SectionCardProps {
  section: DesignerSection;
  isSelected: boolean;
  onSelect: () => void;
  onDelete: () => void;
  onUpdate: (updates: Partial<DesignerSection>) => void;
}

const SectionCard: React.FC<SectionCardProps> = ({
  section,
  isSelected,
  onSelect,
  onDelete,
  onUpdate,
}) => {
  const { attributes, listeners, setNodeRef: setDragRef } = useDraggable({
    id: `section-${section.id}`,
    data: { section },
  });

  return (
    <Card
      ref={setDragRef}
      onClick={onSelect}
      sx={{
        mb: 2,
        cursor: 'pointer',
        border: 2,
        borderColor: isSelected ? 'primary.main' : 'transparent',
        '&:hover': { borderColor: isSelected ? 'primary.main' : 'grey.300' },
      }}
    >
      <CardContent>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <IconButton size="small" {...listeners} {...attributes} sx={{ cursor: 'grab' }}>
            <DragIndicator />
          </IconButton>
          <Typography variant="subtitle1" sx={{ flexGrow: 1 }}>
            {section.title || `${section.type} section`}
          </Typography>
          <Chip label={section.type} size="small" />
          <IconButton size="small" onClick={(e) => { e.stopPropagation(); onDelete(); }}>
            <Delete fontSize="small" />
          </IconButton>
        </Box>

        {section.type === 'table' && section.columns && (
          <Box sx={{ mt: 1, display: 'flex', gap: 0.5, flexWrap: 'wrap' }}>
            {section.columns.map((col, idx) => (
              <Chip key={idx} label={col.label} size="small" variant="outlined" />
            ))}
            {section.columns.length === 0 && (
              <Typography variant="caption" color="text.secondary">
                No columns configured
              </Typography>
            )}
          </Box>
        )}

        {section.data_binding && (
          <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 1 }}>
            Data: {section.data_binding}
          </Typography>
        )}
      </CardContent>
    </Card>
  );
};

// ============================================================================
// PROPERTIES PANEL
// ============================================================================

interface PropertiesPanelProps {
  section: DesignerSection | null;
  onUpdate: (updates: Partial<DesignerSection>) => void;
  dataBindings: string[];
}

const PropertiesPanel: React.FC<PropertiesPanelProps> = ({ section, onUpdate, dataBindings }) => {
  if (!section) {
    return (
      <Box sx={{ p: 2, textAlign: 'center' }}>
        <Typography color="text.secondary">
          Select a section to edit its properties
        </Typography>
      </Box>
    );
  }

  return (
    <Box sx={{ p: 2 }}>
      <Typography variant="subtitle2" gutterBottom>
        Section Properties
      </Typography>

      <TextField
        label="Title"
        value={section.title || ''}
        onChange={(e) => onUpdate({ title: e.target.value })}
        fullWidth
        size="small"
        sx={{ mb: 2 }}
      />

      <FormControl fullWidth size="small" sx={{ mb: 2 }}>
        <InputLabel>Data Binding</InputLabel>
        <Select
          value={section.data_binding || ''}
          label="Data Binding"
          onChange={(e) => onUpdate({ data_binding: e.target.value })}
        >
          <MenuItem value="">None</MenuItem>
          {dataBindings.map((binding) => (
            <MenuItem key={binding} value={binding}>
              {binding}
            </MenuItem>
          ))}
        </Select>
      </FormControl>

      {section.type === 'table' && (
        <>
          <Typography variant="subtitle2" sx={{ mt: 2, mb: 1 }}>
            Columns
          </Typography>
          {section.columns?.map((col, idx) => (
            <Box key={idx} sx={{ display: 'flex', gap: 1, mb: 1 }}>
              <TextField
                label="Label"
                value={col.label}
                size="small"
                sx={{ flex: 1 }}
                onChange={(e) => {
                  const newColumns = [...(section.columns || [])];
                  newColumns[idx] = { ...col, label: e.target.value };
                  onUpdate({ columns: newColumns });
                }}
              />
              <IconButton
                size="small"
                onClick={() => {
                  const newColumns = section.columns?.filter((_, i) => i !== idx);
                  onUpdate({ columns: newColumns });
                }}
              >
                <Delete fontSize="small" />
              </IconButton>
            </Box>
          ))}
          <Button
            size="small"
            startIcon={<Add />}
            onClick={() => {
              const newColumns = [...(section.columns || []), { label: 'New Column', dimension: '' }];
              onUpdate({ columns: newColumns });
            }}
          >
            Add Column
          </Button>
        </>
      )}
    </Box>
  );
};

// ============================================================================
// MAIN REPORT DESIGNER COMPONENT
// ============================================================================

const ReportDesigner: React.FC<ReportDesignerProps> = ({ reportId, onSave, onCancel }) => {
  const { isSelected } = useTenant();

  // Report metadata state
  const [reportKey, setReportKey] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [description, setDescription] = useState('');
  const [category, setCategory] = useState('');

  // Layout state
  const [sections, setSections] = useState<DesignerSection[]>([]);
  const [parameters, setParameters] = useState<Parameter[]>([]);
  const [dataBindings, setDataBindings] = useState<Record<string, DataBinding>>({});
  const [selectedSectionId, setSelectedSectionId] = useState<string | null>(null);

  // UI state
  const [activeTab, setActiveTab] = useState(0);
  const [activeDragItem, setActiveDragItem] = useState<DesignerElement | null>(null);
  const [snackbar, setSnackbar] = useState<{ open: boolean; message: string; severity: 'success' | 'error' }>({
    open: false,
    message: '',
    severity: 'success',
  });
  const [dataBindingDialogOpen, setDataBindingDialogOpen] = useState(false);
  const [parameterDialogOpen, setParameterDialogOpen] = useState(false);

  // Load existing report if editing
  const { data: existingReport, isLoading: loadingReport } = useReportDefinition(reportId);
  const createMutation = useCreateReportDefinition();
  const updateMutation = useUpdateReportDefinition();

  // Initialize from existing report
  React.useEffect(() => {
    if (existingReport) {
      setReportKey(existingReport.report_key);
      setDisplayName(existingReport.display_name);
      setDescription(existingReport.description || '');
      setCategory(existingReport.category || '');

      if (existingReport.definition) {
        // Convert layout sections to designer sections
        const designerSections: DesignerSection[] = existingReport.definition.layout.body.sections.map((s, idx) => ({
          id: `section-${idx}`,
          type: s.type as DesignerSection['type'],
          title: s.title || '',
          data_binding: s.data_binding,
          columns: s.columns,
          elements: s.elements,
        }));
        setSections(designerSections);
        setParameters(existingReport.definition.parameters || []);
        setDataBindings(existingReport.definition.data_bindings || {});
      }
    }
  }, [existingReport]);

  // Selected section
  const selectedSection = useMemo(
    () => sections.find((s) => s.id === selectedSectionId) || null,
    [sections, selectedSectionId]
  );

  // Drag handlers
  const handleDragStart = (event: DragStartEvent) => {
    const { active } = event;
    if (active.data.current?.element) {
      setActiveDragItem(active.data.current.element);
    }
  };

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    setActiveDragItem(null);

    if (over?.id === 'canvas-drop-zone' && active.data.current?.element) {
      const element = active.data.current.element as DesignerElement;
      
      // Create new section
      const newSection: DesignerSection = {
        id: `section-${Date.now()}`,
        type: element.type as DesignerSection['type'],
        title: '',
        columns: element.type === 'table' ? [] : undefined,
        elements: element.type === 'text' ? [] : undefined,
      };

      setSections((prev) => [...prev, newSection]);
      setSelectedSectionId(newSection.id);
    }
  };

  // Section handlers
  const handleUpdateSection = useCallback((updates: Partial<DesignerSection>) => {
    if (!selectedSectionId) return;
    setSections((prev) =>
      prev.map((s) => (s.id === selectedSectionId ? { ...s, ...updates } : s))
    );
  }, [selectedSectionId]);

  const handleDeleteSection = useCallback((sectionId: string) => {
    setSections((prev) => prev.filter((s) => s.id !== sectionId));
    if (selectedSectionId === sectionId) {
      setSelectedSectionId(null);
    }
  }, [selectedSectionId]);

  // Build report layout from designer state
  const buildReportLayout = (): ReportLayout => {
    return {
      metadata: {
        display_name: displayName,
        description,
        category,
        page_size: 'Letter',
        orientation: 'portrait',
      },
      data_bindings: dataBindings,
      parameters,
      layout: {
        body: {
          sections: sections.map((s) => ({
            id: s.id,
            type: s.type,
            title: s.title,
            data_binding: s.data_binding,
            columns: s.columns,
            elements: s.elements,
          })),
        },
      },
    };
  };

  // Save handler
  const handleSave = async () => {
    if (!reportKey || !displayName) {
      setSnackbar({ open: true, message: 'Report key and display name are required', severity: 'error' });
      return;
    }

    const layout = buildReportLayout();

    try {
      if (reportId && existingReport) {
        // Update existing
        await updateMutation.mutateAsync({
          id: reportId,
          updates: {
            display_name: displayName,
            description,
            category,
            definition: layout,
          },
        });
        setSnackbar({ open: true, message: 'Report updated successfully', severity: 'success' });
        onSave?.(reportId);
      } else {
        // Create new
        const request: CreateDefinitionRequest = {
          report_key: reportKey,
          display_name: displayName,
          description,
          category,
          definition: layout,
        };
        const created = await createMutation.mutateAsync(request);
        setSnackbar({ open: true, message: 'Report created successfully', severity: 'success' });
        onSave?.(created.id);
      }
    } catch (err) {
      devLog('Failed to save report:', err);
      setSnackbar({ open: true, message: 'Failed to save report', severity: 'error' });
    }
  };

  if (!isSelected) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="warning">
          Please select a tenant and datasource to create reports.
        </Alert>
      </Box>
    );
  }

  return (
    <DndContext onDragStart={handleDragStart} onDragEnd={handleDragEnd}>
      <Box sx={{ display: 'flex', height: '100vh' }}>
        {/* Left Toolbox */}
        <Drawer
          variant="permanent"
          sx={{
            width: 240,
            flexShrink: 0,
            '& .MuiDrawer-paper': { width: 240, position: 'relative' },
          }}
        >
          <Toolbar variant="dense">
            <Typography variant="subtitle1">Toolbox</Typography>
          </Toolbar>
          <Divider />

          <Accordion defaultExpanded>
            <AccordionSummary expandIcon={<ExpandMore />}>
              <Typography variant="body2">Sections</Typography>
            </AccordionSummary>
            <AccordionDetails sx={{ p: 0 }}>
              <List dense disablePadding>
                {SECTION_ELEMENTS.map((el) => (
                  <ToolboxItem key={el.id} element={el} />
                ))}
              </List>
            </AccordionDetails>
          </Accordion>

          <Accordion>
            <AccordionSummary expandIcon={<ExpandMore />}>
              <Typography variant="body2">Fields</Typography>
            </AccordionSummary>
            <AccordionDetails sx={{ p: 0 }}>
              <List dense disablePadding>
                {FIELD_ELEMENTS.map((el) => (
                  <ToolboxItem key={el.id} element={el} />
                ))}
              </List>
            </AccordionDetails>
          </Accordion>

          <Divider sx={{ my: 1 }} />

          <Box sx={{ p: 1 }}>
            <Button
              fullWidth
              size="small"
              startIcon={<DataObject />}
              onClick={() => setDataBindingDialogOpen(true)}
            >
              Data Bindings ({Object.keys(dataBindings).length})
            </Button>
            <Button
              fullWidth
              size="small"
              startIcon={<Settings />}
              onClick={() => setParameterDialogOpen(true)}
              sx={{ mt: 1 }}
            >
              Parameters ({parameters.length})
            </Button>
          </Box>
        </Drawer>

        {/* Main Content */}
        <Box sx={{ flexGrow: 1, display: 'flex', flexDirection: 'column' }}>
          {/* Toolbar */}
          <Toolbar variant="dense" sx={{ borderBottom: 1, borderColor: 'divider' }}>
            <TextField
              label="Report Key"
              value={reportKey}
              onChange={(e) => setReportKey(e.target.value)}
              size="small"
              sx={{ width: 150, mr: 1 }}
              disabled={!!reportId}
            />
            <TextField
              label="Display Name"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              size="small"
              sx={{ width: 200, mr: 1 }}
            />
            <TextField
              label="Category"
              value={category}
              onChange={(e) => setCategory(e.target.value)}
              size="small"
              sx={{ width: 120, mr: 2 }}
            />

            <Box sx={{ flexGrow: 1 }} />

            <Tooltip title="Undo">
              <IconButton size="small">
                <Undo />
              </IconButton>
            </Tooltip>
            <Tooltip title="Redo">
              <IconButton size="small">
                <Redo />
              </IconButton>
            </Tooltip>
            <Divider orientation="vertical" flexItem sx={{ mx: 1 }} />
            <Tooltip title="Preview">
              <IconButton size="small">
                <Preview />
              </IconButton>
            </Tooltip>
            <Button
              variant="contained"
              size="small"
              startIcon={<Save />}
              onClick={handleSave}
              disabled={createMutation.isPending || updateMutation.isPending}
              sx={{ ml: 1 }}
            >
              Save
            </Button>
            {onCancel && (
              <IconButton size="small" onClick={onCancel} sx={{ ml: 1 }}>
                <Close />
              </IconButton>
            )}
          </Toolbar>

          {/* Canvas */}
          <Box sx={{ flexGrow: 1, overflow: 'auto', p: 2 }}>
            <Paper sx={{ maxWidth: 850, mx: 'auto', p: 2 }}>
              <Typography variant="h6" gutterBottom>
                {displayName || 'Untitled Report'}
              </Typography>
              {description && (
                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                  {description}
                </Typography>
              )}

              <CanvasDropZone onDrop={() => {}}>
                {sections.length === 0 ? (
                  <Box sx={{ textAlign: 'center', py: 4 }}>
                    <Typography color="text.secondary">
                      Drag sections from the toolbox to build your report
                    </Typography>
                  </Box>
                ) : (
                  sections.map((section) => (
                    <SectionCard
                      key={section.id}
                      section={section}
                      isSelected={section.id === selectedSectionId}
                      onSelect={() => setSelectedSectionId(section.id)}
                      onDelete={() => handleDeleteSection(section.id)}
                      onUpdate={handleUpdateSection}
                    />
                  ))
                )}
              </CanvasDropZone>
            </Paper>
          </Box>
        </Box>

        {/* Right Properties Panel */}
        <Drawer
          variant="permanent"
          anchor="right"
          sx={{
            width: 280,
            flexShrink: 0,
            '& .MuiDrawer-paper': { width: 280, position: 'relative' },
          }}
        >
          <Toolbar variant="dense">
            <Typography variant="subtitle1">Properties</Typography>
          </Toolbar>
          <Divider />
          <PropertiesPanel
            section={selectedSection}
            onUpdate={handleUpdateSection}
            dataBindings={Object.keys(dataBindings)}
          />
        </Drawer>

        {/* Drag Overlay */}
        <DragOverlay>
          {activeDragItem && (
            <Paper sx={{ p: 1, display: 'flex', alignItems: 'center', gap: 1 }}>
              {activeDragItem.icon}
              <Typography variant="body2">{activeDragItem.label}</Typography>
            </Paper>
          )}
        </DragOverlay>

        {/* Snackbar */}
        <Snackbar
          open={snackbar.open}
          autoHideDuration={4000}
          onClose={() => setSnackbar((s) => ({ ...s, open: false }))}
        >
          <Alert severity={snackbar.severity}>{snackbar.message}</Alert>
        </Snackbar>

        {/* Data Binding Dialog */}
        <Dialog open={dataBindingDialogOpen} onClose={() => setDataBindingDialogOpen(false)} maxWidth="md" fullWidth>
          <DialogTitle>Data Bindings</DialogTitle>
          <DialogContent>
            <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
              Configure data sources from your Cube.dev semantic layer.
            </Typography>
            {Object.entries(dataBindings).map(([name, binding]) => (
              <Card key={name} sx={{ mb: 2 }}>
                <CardContent>
                  <Typography variant="subtitle2">{name}</Typography>
                  <Typography variant="caption" color="text.secondary">
                    Cube: {binding.cube} | Measures: {binding.measures.join(', ')}
                  </Typography>
                </CardContent>
              </Card>
            ))}
            <Button startIcon={<Add />} onClick={() => {
              const name = `binding_${Object.keys(dataBindings).length + 1}`;
              setDataBindings((prev) => ({
                ...prev,
                [name]: { cube: '', measures: [], dimensions: [] },
              }));
            }}>
              Add Data Binding
            </Button>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setDataBindingDialogOpen(false)}>Close</Button>
          </DialogActions>
        </Dialog>

        {/* Parameter Dialog */}
        <Dialog open={parameterDialogOpen} onClose={() => setParameterDialogOpen(false)} maxWidth="sm" fullWidth>
          <DialogTitle>Report Parameters</DialogTitle>
          <DialogContent>
            <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
              Define parameters that users can set when running this report.
            </Typography>
            {parameters.map((param, idx) => (
              <Card key={idx} sx={{ mb: 2 }}>
                <CardContent>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                    <Typography variant="subtitle2">{param.label}</Typography>
                    <Chip label={param.type} size="small" />
                  </Box>
                  <Typography variant="caption" color="text.secondary">
                    Name: {param.name} {param.required && '(Required)'}
                  </Typography>
                </CardContent>
              </Card>
            ))}
            <Button startIcon={<Add />} onClick={() => {
              setParameters((prev) => [
                ...prev,
                { name: `param_${prev.length + 1}`, type: 'string', label: 'New Parameter' },
              ]);
            }}>
              Add Parameter
            </Button>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setParameterDialogOpen(false)}>Close</Button>
          </DialogActions>
        </Dialog>
      </Box>
    </DndContext>
  );
};

export default ReportDesigner;
