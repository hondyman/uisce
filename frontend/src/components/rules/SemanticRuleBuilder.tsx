import { useState, useCallback, useEffect } from 'react';
import {
  Box,
  AppBar,
  Toolbar,
  Typography,
  Button,
  Tabs,
  Tab,
  Container,
  CircularProgress,
  Paper,
  Grid,
  IconButton,
  Chip,
  Stack,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
} from '@mui/material';
import {
  Add as AddIcon,
  Preview as PreviewIcon,
  History as HistoryIcon,
  Download as DownloadIcon,
} from '@mui/icons-material';
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  DragEndEvent,
} from '@dnd-kit/core';
import {
  SortableContext,
  verticalListSortingStrategy,
  arrayMove,
} from '@dnd-kit/sortable';

import { SemanticCatalog } from './SemanticCatalog';
import { PriorityHierarchyEditor } from './PriorityHierarchyEditor';
import { SimulationPanel } from './SimulationPanel';
import { RuleVersionControl } from './RuleVersionControl';
import { TemplateBrowser } from '../TemplateBrowser';
import { useRuleBuilder } from '../../hooks/useRuleBuilder';

interface SemanticRuleBuilderProps {
  businessObject: string;
  initialRuleId?: string;
  onRulePublished?: (rule: any) => void;
  readOnly?: boolean;
}

/**
 * SemanticRuleBuilder Component (Material-UI)
 * Production-ready UI for semantic-driven priority hierarchy rules
 */
export const SemanticRuleBuilder = ({
  businessObject,
  initialRuleId,
  onRulePublished,
  readOnly = false,
}: SemanticRuleBuilderProps) => {
  const { rule, loading, error, addStep, updateStep, deleteStep, reorderSteps } =
    useRuleBuilder(initialRuleId, businessObject);

  const [expandedSteps, setExpandedSteps] = useState<Set<string>>(new Set());
  const [testData, setTestData] = useState<any>(null);
  const [activeTab, setActiveTab] = useState(0);
  const [simulationResults, setSimulationResults] = useState<any>(null);

  const sensors = useSensors(
    useSensor(PointerSensor, { distance: 8 }),
    useSensor(KeyboardSensor)
  );

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (over && active.id !== over.id && rule?.steps) {
      const oldIndex = rule.steps.findIndex((s: any) => s.id === active.id);
      const newIndex = rule.steps.findIndex((s: any) => s.id === over.id);
      const newSteps = arrayMove(rule.steps, oldIndex, newIndex);
      reorderSteps(newSteps);
    }
  };

  const toggleStepExpanded = useCallback((stepId: string) => {
    setExpandedSteps((prev) => {
      const next = new Set(prev);
      next.has(stepId) ? next.delete(stepId) : next.add(stepId);
      return next;
    });
  }, []);

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Box sx={{ p: 4, textAlign: 'center' }}>
        <Typography color="error">{error}</Typography>
      </Box>
    );
  }

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100vh', backgroundColor: '#f5f5f5' }}>
      {/* Header */}
      <AppBar position="static" elevation={1}>
        <Toolbar>
          <Box sx={{ flex: 1 }}>
            <Typography variant="h6" fontWeight="bold">
              Semantic Rule Builder
            </Typography>
            <Typography variant="caption" sx={{ display: 'block', mt: 0.5 }}>
              Building rules for: <strong>{businessObject}</strong>
            </Typography>
          </Box>
          <Button
            startIcon={<HistoryIcon />}
            color="inherit"
            onClick={() => setActiveTab(2)}
          >
            Version History
          </Button>
          {rule?.status === 'draft' && !readOnly && (
            <Button startIcon={<PreviewIcon />} color="inherit" sx={{ ml: 1 }}>
              Preview
            </Button>
          )}
        </Toolbar>

        {/* Tabs */}
        <Tabs value={activeTab} onChange={(e, newValue) => setActiveTab(newValue)}>
          <Tab label="Rule Builder" />
          <Tab label="From Template" />
          <Tab label="Governance" />
          <Tab label="Versions" />
        </Tabs>
      </AppBar>

      {/* Main Content */}
      <Box sx={{ flex: 1, overflow: 'auto', p: 2 }}>
        {activeTab === 0 && (
          <Grid container spacing={2} sx={{ height: '100%' }}>
            {/* Left: Semantic Catalog */}
            <Grid item xs={12} md={3} sx={{ overflow: 'auto' }}>
              <Paper elevation={1} sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
                <SemanticCatalog businessObject={businessObject} />
              </Paper>
            </Grid>

            {/* Center: Priority Hierarchy */}
            <Grid item xs={12} md={6} sx={{ overflow: 'auto' }}>
              <Paper elevation={1} sx={{ height: '100%', display: 'flex', flexDirection: 'column', p: 2 }}>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                  <Typography variant="subtitle1" fontWeight="600">
                    Priority Selection Hierarchy
                  </Typography>
                  {!readOnly && (
                    <Button startIcon={<AddIcon />} variant="contained" size="small" onClick={() => addStep()}>
                      Add Priority
                    </Button>
                  )}
                </Box>

                <DndContext
                  sensors={sensors}
                  collisionDetection={closestCenter}
                  onDragEnd={handleDragEnd}
                >
                  <SortableContext
                    items={rule?.steps?.map((s: any) => s.id) || []}
                    strategy={verticalListSortingStrategy}
                    disabled={readOnly}
                  >
                    <Stack spacing={1.5} sx={{ flex: 1 }}>
                      {rule?.steps && rule.steps.length > 0 ? (
                        rule.steps.map((step: any) => (
                          <PriorityHierarchyEditor
                            key={step.id}
                            step={step}
                            isExpanded={expandedSteps.has(step.id)}
                            onToggleExpand={() => toggleStepExpanded(step.id)}
                            onUpdate={(updates) => updateStep(step.id, updates)}
                            onDelete={() => deleteStep(step.id)}
                            readOnly={readOnly}
                          />
                        ))
                      ) : (
                        <Paper
                          sx={{
                            p: 3,
                            textAlign: 'center',
                            border: '2px dashed',
                            borderColor: 'divider',
                          }}
                        >
                          <Typography variant="body2" fontWeight="500" color="textSecondary">
                            No priority rules defined
                          </Typography>
                          <Typography variant="caption" color="textDisabled" sx={{ mt: 1, display: 'block' }}>
                            Add your first priority rule to determine how data sources are selected
                          </Typography>
                          {!readOnly && (
                            <Button
                              startIcon={<AddIcon />}
                              variant="outlined"
                              size="small"
                              onClick={() => addStep()}
                              sx={{ mt: 2 }}
                            >
                              Add Priority Rule
                            </Button>
                          )}
                        </Paper>
                      )}
                    </Stack>
                  </SortableContext>
                </DndContext>

                {/* Default Action */}
                {rule?.steps && rule.steps.length > 0 && (
                  <Paper sx={{ p: 2, mt: 2, backgroundColor: 'action.hover' }}>
                    <Typography variant="subtitle2" fontWeight="600" sx={{ mb: 1 }}>
                      <Chip label="DEFAULT" size="small" variant="outlined" sx={{ mr: 1 }} />
                      When no other rules match
                    </Typography>
                    <FormControl fullWidth size="small">
                      <Select defaultValue="RAISE_EXCEPTION" disabled={readOnly}>
                        <MenuItem value="RAISE_EXCEPTION">Raise exception</MenuItem>
                        <MenuItem value="USE_DEFAULT_VALUE">Use default value</MenuItem>
                        <MenuItem value="SKIP">Skip processing</MenuItem>
                      </Select>
                    </FormControl>
                  </Paper>
                )}
              </Paper>
            </Grid>

            {/* Right: Simulation */}
            <Grid item xs={12} md={3} sx={{ overflow: 'auto' }}>
              <Paper elevation={1} sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
                <SimulationPanel
                  rule={rule}
                  businessObject={businessObject}
                  testData={testData}
                  onTestDataChange={setTestData}
                  simulationResults={simulationResults}
                />
              </Paper>
            </Grid>
          </Grid>
        )}

        {activeTab === 1 && (
          <Box sx={{ maxWidth: '1200px', mx: 'auto' }}>
            <TemplateBrowser 
              businessObject={businessObject} 
              onRuleCreated={(ruleId) => {
                // Optionally reload rule data and switch to builder tab
                setActiveTab(0);
              }} 
            />
          </Box>
        )}

        {activeTab === 2 && (
          <Box sx={{ maxWidth: '1000px', mx: 'auto' }}>
            <Typography variant="h5" fontWeight="bold" sx={{ mb: 3 }}>
              Governance & Approvals
            </Typography>
            {/* Governance content */}
          </Box>
        )}

        {activeTab === 3 && (
          <Box sx={{ maxWidth: '1200px', mx: 'auto' }}>
            <RuleVersionControl
              ruleId={rule?.id || ''}
              versions={[]}
              currentVersionId={rule?.version || 1}
              onVersionSelect={() => {}}
            />
          </Box>
        )}
      </Box>

      {/* Footer */}
      <Paper elevation={2} sx={{ p: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Typography variant="caption" color="textSecondary">
          {rule?.steps?.length || 0} priority rules • Last modified:{' '}
          {rule?.updatedAt ? new Date(rule.updatedAt).toLocaleDateString() : 'N/A'}
        </Typography>
        <Stack direction="row" spacing={1}>
          <Button variant="outlined">Discard</Button>
          <Button variant="contained">Save Draft</Button>
          {rule?.status === 'draft' && (
            <Button variant="contained" color="success">
              Submit for Review
            </Button>
          )}
        </Stack>
      </Paper>
    </Box>
  );
};

export default SemanticRuleBuilder;
